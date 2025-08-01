package gonkaopenai

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	cryptotypes "github.com/cometbft/cometbft/proto/tendermint/crypto"
	"github.com/cosmos/gogoproto/proto"
	ics23 "github.com/cosmos/ics23/go"
	"net/http"
)

var ErrInvalidEpoch = errors.New("invalid epoch")

// GetParticipantsWithProof fetches participants from the specified base URL and epoch,
// verifies the proof, and returns a list of Endpoints.
// This function is independent of the GonkaOpenAI client.
// Specify "current" as the epoch to fetch the current participants.
func GetParticipantsWithProof(ctx context.Context, baseURL string, epoch string) ([]Endpoint, error) {
	if epoch == "" {
		return nil, ErrInvalidEpoch
	}

	// Ensure baseURL doesn't end with a slash
	if baseURL != "" && baseURL[len(baseURL)-1] == '/' {
		baseURL = baseURL[:len(baseURL)-1]
	}

	url := fmt.Sprintf("%s/v1/epochs/%v/participants", baseURL, epoch)

	// Create a new HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch participants with proof: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch participants with proof: status code %d", resp.StatusCode)
	}

	// Decode the response
	var participantResp ActiveParticipantWithProof
	if err := json.NewDecoder(resp.Body).Decode(&participantResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	val, err := hex.DecodeString(participantResp.ActiveParticipantsBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode participants bytes: %w", err)
	}

	err = VerifyIAVLProofAgainstAppHash(participantResp.Block.AppHash, participantResp.ProofOps.Ops, val)
	if err != nil {
		return nil, fmt.Errorf("failed to verify participants proof: %w", err)
	}

	// Convert ActiveParticipants to Endpoints
	endpoints := make([]Endpoint, 0, len(participantResp.ActiveParticipants.Participants))
	for _, participant := range participantResp.ActiveParticipants.Participants {
		endpoints = append(endpoints, Endpoint{
			URL:     participant.InferenceUrl,
			Address: participant.Index,
		})
	}
	return endpoints, nil
}

// VerifyIAVLProofAgainstAppHash verifies the correctness of an ABCIQuery response for ActiveParticipants.
//
// In our case, ActiveParticipants always return proofOps consisting of exactly two items:
//   1) ics23:iavl   — proves that the key (active_participants_by_epoch) → value (active participants entities) is indeed present in the IAVL tree of the "inference" store;
//   2) ics23:simple — proves that the root of this store (storeRoot) is included in the block’s AppHash.
//
// Thanks to this fixed proof structure, we can implement a simpler verification function
// without dealing with a fully generic chain of nested Merkle proofs.
// The function:
//   - first verifies key → value in the IAVL store (and obtains storeRoot);
//   - then verifies that the storeRoot is included in the block’s AppHash;
//   - if both checks succeed, the value is guaranteed to be part of the application state
//     signed by validators at the given block height.

func VerifyIAVLProofAgainstAppHash(appHash []byte, proofOps []cryptotypes.ProofOp, value []byte) error {
	if len(proofOps) != 2 {
		return fmt.Errorf("expected 2 proof ops, got %d", len(proofOps))
	}

	// Step 1: key → value в store (IAVL)
	iavlOp := proofOps[0]
	if iavlOp.Type != "ics23:iavl" {
		return fmt.Errorf("unexpected first proof op type: %s", iavlOp.Type)
	}
	var iavlProof ics23.CommitmentProof
	if err := proto.Unmarshal(iavlOp.Data, &iavlProof); err != nil {
		return fmt.Errorf("failed to unmarshal IAVL proof: %w", err)
	}
	storeRoot, err := iavlProof.Calculate()
	if err != nil {
		return fmt.Errorf("failed to calculate IAVL proof: %w", err)
	}

	if ok := ics23.VerifyMembership(ics23.IavlSpec, storeRoot, &iavlProof, iavlOp.Key, value); !ok {
		return fmt.Errorf("IAVL proof failed")
	}

	// Step 2: storeRoot → AppHash (Tendermint multistore)
	simpleOp := proofOps[1]
	if simpleOp.Type != "ics23:simple" {
		return fmt.Errorf("unexpected second proof op type: %s", simpleOp.Type)
	}
	var simpleProof ics23.CommitmentProof
	if err := proto.Unmarshal(simpleOp.Data, &simpleProof); err != nil {
		return fmt.Errorf("failed to unmarshal simple proof: %w", err)
	}
	if ok := ics23.VerifyMembership(ics23.TendermintSpec, appHash, &simpleProof, simpleOp.Key, storeRoot); !ok {
		return fmt.Errorf("simple proof failed")
	}
	return nil
}

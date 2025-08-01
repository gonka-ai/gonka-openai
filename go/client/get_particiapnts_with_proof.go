package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	ed "github.com/cometbft/cometbft/crypto/ed25519"
	cryptotypes "github.com/cometbft/cometbft/proto/tendermint/crypto"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/gogoproto/proto"
	ics23 "github.com/cosmos/ics23/go"
)

var ErrInvalidEpoch = errors.New("invalid epoch")

func (g *GonkaOpenAI) GetParticipantsUrls(ctx context.Context, epoch string) ([]string, error) {
	resp, err := g.getParticipants(ctx, epoch)
	if err != nil {
		return nil, err
	}

	val, err := hex.DecodeString(resp.ActiveParticipantsBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch participants with proof: %w", err)
	}

	err = VerifyIAVLProofAgainstAppHash(resp.Block.AppHash, resp.ProofOps.Ops, val)
	if err != nil {
		return nil, fmt.Errorf("failed to verify participants proof: %w", err)
	}

	urls := make([]string, 0)
	for _, participant := range resp.ActiveParticipants.Participants {
		urls = append(urls, participant.InferenceUrl)
	}
	return urls, nil
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

func (g *GonkaOpenAI) VerifyParticipants(ctx context.Context, expectedHashHex []byte) error {
	resp, err := g.getParticipants(ctx, "current")
	if err != nil {
		return err
	}

	var olderEpochValidators map[string]string

	for epochId := resp.ActiveParticipants.EpochId; epochId == 0; {
		if bytes.Equal(resp.Block.AppHash, expectedHashHex) {
			return nil
		}

		if len(olderEpochValidators) != 0 {
			for range {
				// TODO все валидаторы, подписавшие эпоху N+1, должны были быть active participants в эпоху N
			}
		}

		block := resp.Block

		vote := tmproto.Vote{
			Type:   tmproto.PrecommitType,
			Height: block.LastCommit.Height,
			Round:  block.LastCommit.Round,
			BlockID: tmproto.BlockID{
				Hash: block.LastCommit.BlockID.Hash,
				PartSetHeader: tmproto.PartSetHeader{
					Total: block.LastCommit.BlockID.PartSetHeader.Total,
					Hash:  block.LastCommit.BlockID.PartSetHeader.Hash,
				},
			},
		}

		validatorsData := make(map[string]string)
		for _, validator := range resp.Validators {
			validatorsData[validator.Address] = validator.PubKey
		}

		err := verifySignatures(vote, block.ChainID, validatorsData, block.LastCommit.Signatures)
		if err != nil {
			return err
		}

		// TODO verify participants

		olderEpochValidators = validatorsData
		epochId--
	}
	return fmt.Errorf("particiants unverified: expected hash %s, but got %s", hex.EncodeToString(expectedHashHex), resp.Block.AppHash)
}

func (g *GonkaOpenAI) getParticipants(ctx context.Context, epoch string) (*ActiveParticipantWithProof, error) {
	if epoch == "" {
		return nil, ErrInvalidEpoch
	}

	url := fmt.Sprintf("v1/epochs/%v/participants", epoch)
	var resp ActiveParticipantWithProof
	err := g.Get(ctx, url, nil, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch participants with proof: %w", err)
	}
	return &resp, err
}

func verifySignatures(vote tmproto.Vote, chainId string, validators map[string]string, signatures []tmtypes.CommitSig) error {
	for _, signature := range signatures {
		vote.Timestamp = signature.Timestamp
		signBytes := tmtypes.VoteSignBytes(chainId, &vote)

		pubKeyBase64 := validators[signature.ValidatorAddress.String()]
		pubKeyBytes, err := base64.StdEncoding.DecodeString(pubKeyBase64)
		if err != nil {
			return fmt.Errorf("decode pubkey: %w", err)
		}

		pubKey := ed.PubKey(pubKeyBytes)
		if ok := pubKey.VerifySignature(signBytes, signature.Signature); !ok {
			return fmt.Errorf("failed to verify signature for addr %v \n", pubKey.Address().String())
		}
	}
	return nil
}

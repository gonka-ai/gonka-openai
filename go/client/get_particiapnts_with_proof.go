package client

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/gonka-ai/gonka-utils/go/contracts"
	"github.com/gonka-ai/gonka-utils/go/utils"
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

	// TODO after genesis block hash is hardcoded, use VerifyParticipants here
	appHash, err := hex.DecodeString(resp.BlockProof.AppHashHex)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch participants with proof: %w", err)
	}

	err = utils.VerifyIAVLProofAgainstAppHash(appHash, resp.ProofOps.Ops, val)
	if err != nil {
		return nil, fmt.Errorf("failed to verify participants proof: %w", err)
	}

	urls := make([]string, 0)
	for _, participant := range resp.ActiveParticipants.Participants {
		urls = append(urls, participant.InferenceUrl)
	}
	return urls, nil
}

func (g *GonkaOpenAI) VerifyParticipants(ctx context.Context, expectedHashHex string) error {
	return utils.VerifyParticipants(ctx, expectedHashHex, g.getParticipants)
}

func (g *GonkaOpenAI) GetBlock(ctx context.Context, height int64) (*coretypes.ResultBlock, error) {
	url := fmt.Sprintf("v1/block/%v", height)
	var resp coretypes.ResultBlock
	err := g.Get(ctx, url, nil, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch participants with proof: %w", err)
	}
	return &resp, err
}

func (g *GonkaOpenAI) getParticipants(ctx context.Context, epoch string) (*contracts.ActiveParticipantWithProof, error) {
	if epoch == "" {
		return nil, ErrInvalidEpoch
	}

	url := fmt.Sprintf("v1/epochs/%v/participants", epoch)
	var resp contracts.ActiveParticipantWithProof
	err := g.Get(ctx, url, nil, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch participants with proof: %w", err)
	}
	return &resp, err
}

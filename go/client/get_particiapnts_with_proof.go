package client

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	ce "github.com/gonka-ai/gonka-utils/go/common_entity"
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
	err = utils.VerifyIAVLProofAgainstAppHash(resp.Block.AppHash, resp.ProofOps.Ops, val)
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

func (g *GonkaOpenAI) getParticipants(ctx context.Context, epoch string) (*ce.ActiveParticipantWithProof, error) {
	if epoch == "" {
		return nil, ErrInvalidEpoch
	}

	url := fmt.Sprintf("v1/epochs/%v/participants", epoch)
	var resp ce.ActiveParticipantWithProof
	err := g.Get(ctx, url, nil, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch participants with proof: %w", err)
	}
	return &resp, err
}

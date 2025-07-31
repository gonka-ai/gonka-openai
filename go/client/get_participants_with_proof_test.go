package client

import (
	"context"
	"fmt"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_GetParticipants(t *testing.T) {
	cl, err := NewGonkaOpenAI(Options{
		GonkaPrivateKey: "10af8dc1f63fb90cfa39943a5afbf262cd84f24919e7d05653e3b03313e685ce",
		GonkaAddress:    "cosmos1waj8q9g2ekgardafc6plg77rgu2l3vfrclrm4v",
		Endpoints:       []string{"http://localhost:9000"},
		OrgID:           "gonka-client-test-id",
	})
	assert.NoError(t, err)

	urls, err := cl.getParticipants(context.Background(), "1")
	fmt.Println(urls)
	assert.NoError(t, err)
	assert.Len(t, urls, 3)
}

func Test_SignatureVerification(t *testing.T) {
	cl, err := NewGonkaOpenAI(Options{
		GonkaPrivateKey: "10af8dc1f63fb90cfa39943a5afbf262cd84f24919e7d05653e3b03313e685ce",
		GonkaAddress:    "cosmos1waj8q9g2ekgardafc6plg77rgu2l3vfrclrm4v",
		Endpoints:       []string{"http://localhost:9000"},
		OrgID:           "gonka-client-test-id",
	})
	assert.NoError(t, err)

	resp, err := cl.getParticipants(context.Background(), "2")
	assert.NoError(t, err)

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
		Timestamp: block.Time,
	}

	validatorsData := make(map[string]string)
	for _, validator := range resp.Validators {
		validatorsData[validator.Address] = validator.PubKey
	}

	err = verifySignatures(vote, block.ChainID, validatorsData, block.LastCommit.Signatures)
	assert.NoError(t, err)
}

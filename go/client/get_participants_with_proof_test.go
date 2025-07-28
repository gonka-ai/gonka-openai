package client

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

	resp, err := cl.GetParticipantsWithProof(context.Background(), "1")
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	data, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(data))

	val, err := hex.DecodeString(resp.ActiveParticipantsBytes)
	assert.NoError(t, err)

	err = VerifyIAVLProofAgainstAppHash(resp.Block.AppHash, resp.ProofOps.Ops, val)
	assert.NoError(t, err)
}

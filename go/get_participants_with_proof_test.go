package gonkaopenai

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"strings"
	"testing"
)

func Test_GetParticipantsWithProof(t *testing.T) {
	// Use the standalone function with a base URL
	baseURL := "http://localhost:9000"
	endpoints, err := GetParticipantsWithProof(context.Background(), baseURL, "current")

	fmt.Println("Endpoints:", endpoints)
	assert.NoError(t, err)
	assert.Len(t, endpoints, 3)

	// Verify that each endpoint has a URL and Address
	for _, endpoint := range endpoints {
		assert.NotEmpty(t, endpoint.URL, "Endpoint URL should not be empty")
		assert.NotEmpty(t, endpoint.Address, "Endpoint Address should not be empty")
	}
}

// Keep the original test for backward compatibility
func Test_GetParticipants(t *testing.T) {
	// Skip the test immediately to avoid unused variable warnings
	t.Skip("GetParticipantsUrls method has been replaced by the standalone GetParticipantsWithProof function")

	// The following code is kept for reference but is not executed
	_, err := NewGonkaOpenAI(Options{
		GonkaPrivateKey: "10af8dc1f63fb90cfa39943a5afbf262cd84f24919e7d05653e3b03313e685ce",
		GonkaAddress:    "cosmos1waj8q9g2ekgardafc6plg77rgu2l3vfrclrm4v",
		SourceUrl:       "http://localhost:9000",
		OrgID:           "gonka-client-test-id",
	})
	assert.NoError(t, err)
}

func Test_SignatureVerification(t *testing.T) {
	const (
		genesisBlockAppHash = "5BA86012EEDB62551BAB011A76488F05908AE78648C95B645E8FCBCB223CF9C6"
		invalidAppHash      = "SOMEHASH"
		baseURL             = "http://localhost:9000"
	)

	err := os.Setenv(appHashEnv, genesisBlockAppHash)
	assert.NoError(t, err)

	err = os.Setenv(verifyEnabledEnv, "1")
	assert.NoError(t, err)

	endpoints, err := GetParticipantsWithProof(context.Background(), baseURL, "current")
	assert.NoError(t, err)
	assert.Len(t, endpoints, 2)

	err = os.Unsetenv(appHashEnv)
	assert.NoError(t, err)

	err = os.Setenv(appHashEnv, invalidAppHash)
	assert.NoError(t, err)

	endpoints, err = GetParticipantsWithProof(context.Background(), baseURL, "current")
	assert.Error(t, err)
	assert.Len(t, endpoints, 0)
	assert.True(t, strings.Contains(err.Error(), "participants unverified"))
}

/*func Test_BlockProof_And_ValidatorsProofData(t *testing.T) {
	cl, err := NewGonkaOpenAI(Options{
		GonkaPrivateKey: "10af8dc1f63fb90cfa39943a5afbf262cd84f24919e7d05653e3b03313e685ce",
		GonkaAddress:    "cosmos1waj8q9g2ekgardafc6plg77rgu2l3vfrclrm4v",
		Endpoints:       []string{"http://localhost:9010"},
		OrgID:           "gonka-client-test-id",
	})
	assert.NoError(t, err)

	participants, err := cl.getParticipants(context.Background(), "1")
	assert.NoError(t, err)

	var (
		rpcURL = flag.String("rpc", "http://localhost:8101", "CometBFT RPC URL")
	)
	flag.Parse()

	client, err := rpchttp.New(*rpcURL, "/websocket")
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	blockHeight := participants.ActiveParticipants.CreatedAtBlockHeight + 1
	blockRes, err := client.Block(ctx, &blockHeight)
	assert.NoError(t, err)
	assert.Equal(t, strings.ToUpper(participants.BlockProof.AppHashHex), strings.ToUpper(blockRes.Block.AppHash.String()))

	assert.Equal(t, len(blockRes.Block.LastCommit.Signatures), len(participants.ValidatorsProof.Signatures))
	sigData := make(map[string]types.CommitSig)
	for _, origSign := range blockRes.Block.LastCommit.Signatures {
		sigData[strings.ToUpper(origSign.ValidatorAddress.String())] = origSign
	}

	for _, participantsSign := range participants.ValidatorsProof.Signatures {
		origSign, ok := sigData[strings.ToUpper(participantsSign.ValidatorAddressHex)]
		assert.True(t, ok)

		origSign64 := base64.StdEncoding.EncodeToString(origSign.Signature)

		assert.Equal(t, strings.ToUpper(participantsSign.SignatureBase64), strings.ToUpper(origSign64))
		assert.True(t, origSign.Timestamp.Equal(participantsSign.Timestamp))
	}
}*/

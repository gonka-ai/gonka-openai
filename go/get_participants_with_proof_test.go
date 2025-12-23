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
		genesisBlockAppHash = "8c677bc8ee51c988577f37aa4c44a8727f81af0b5b68f702550cff0dce9f7f98" // pass your expected hash
		invalidAppHash      = "SOMEHASH"
		baseURL             = "http://localhost:9020"
		epoch               = "1"
	)

	err := os.Setenv(appHashEnv, genesisBlockAppHash)
	assert.NoError(t, err)

	err = os.Setenv(verifyEnabledEnv, "1")
	assert.NoError(t, err)

	_, err = GetParticipantsWithProof(context.Background(), baseURL, epoch)
	assert.NoError(t, err)

	err = os.Unsetenv(appHashEnv)
	assert.NoError(t, err)

	err = os.Setenv(appHashEnv, invalidAppHash)
	assert.NoError(t, err)

	endpoints, err := GetParticipantsWithProof(context.Background(), baseURL, epoch)
	assert.Error(t, err)
	assert.Len(t, endpoints, 0)
	assert.True(t, strings.Contains(err.Error(), "participants unverified"))
}

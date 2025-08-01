package client

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"strings"
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

	const (
		validAppHash   = "5A8D509CFEE2E1E55897D814A1C0EF7BC9E1291DB5BC5CFC1F1E13F1C93D677F"
		invalidAppHash = "SOMEHASH"
	)

	err = cl.VerifyParticipants(context.Background(), validAppHash)
	assert.NoError(t, err)

	err = cl.VerifyParticipants(context.Background(), invalidAppHash)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "participants unverified"))
}

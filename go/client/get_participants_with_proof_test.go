package client

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	"github.com/cometbft/cometbft/types"
	"github.com/gonka-ai/gonka-utils/go/contracts"
	"github.com/gonka-ai/gonka-utils/go/utils"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
	"time"
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
		genesisBlockAppHash = "5BA86012EEDB62551BAB011A76488F05908AE78648C95B645E8FCBCB223CF9C6"
		firstEpochAppHash   = "11677D45B9F1C8239E8F7A627921DC9AB85F04FAFD739BE464D4325040A89976"
		invalidAppHash      = "SOMEHASH"
	)

	err = cl.VerifyParticipants(context.Background(), genesisBlockAppHash)
	assert.NoError(t, err)

	err = cl.VerifyParticipants(context.Background(), invalidAppHash)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "participants unverified"))
}

func Test_BlockProof_And_ValidatorsProofData(t *testing.T) {
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
}

func TestName(t *testing.T) {
	var (
		rpcURL = flag.String("rpc", "http://localhost:26657", "CometBFT RPC URL")
	)
	flag.Parse()

	client, err := rpchttp.New(*rpcURL, "/websocket")
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	blockHeight := int64(1)
	blockRes, err := client.Block(ctx, &blockHeight)
	assert.NoError(t, err)

	fmt.Println(blockRes.Block.AppHash.String())
}

func TestVerifyGenesisSign(t *testing.T) {
	const (
		expectedBlockIdHash    = "D2E08EEE82A97393C75227FCB6015EAE2917C0B92F28FC4897E855F9B5508ECB"
		expectedPartHeaderHash = "AAC63406F0A9B7DD5EECDF24817316CAFE898A9BB84648DC5E6DCB6619C70375"
	)
	/*	blockIdHash, err := hex.DecodeString(expectedBlockIdHash)
		assert.NoError(t, err)

		partsHeaderHash, err := hex.DecodeString("7A97D9AA41B5F75EC7423F1DFB8B06DD4F08CEE79FE93BD6BDDE60B613F07427")
		assert.NoError(t, err)

		vote := tmproto.Vote{
			Type:   tmproto.PrecommitType,
			Height: 1,
			Round:  0,
			BlockID: tmproto.BlockID{
				Hash: blockIdHash,
				PartSetHeader: tmproto.PartSetHeader{
					Total: 1,
					Hash:  partsHeaderHash,
				},
			},
		}

		signatureData := make([]*contracts.SignatureInfo, 1)
		timeStamp, err := time.Parse(time.RFC3339Nano, "2025-08-17T20:11:48.747521375Z")
		assert.NoError(t, err)

		signatureData[0] = &contracts.SignatureInfo{
			SignatureBase64:     "gS3JAnnOtEk+RgNzueFgcIVM8E82OLZNlNc/P81N/eL0OxS4l6kW3+pF14sf6zh0MJYBTUZO1b9Dh3Zl5J5jCg==",
			ValidatorAddressHex: "44C710E913DFD0792536EC47DB5391BEDDB278B5",
			Timestamp:           timeStamp,
		}

		validatorsData := make(map[string]string)
		validatorsData["44C710E913DFD0792536EC47DB5391BEDDB278B5"] = "anKom9MY7S8GQv5mEyGya36eyFAk9pSGyU1gRrda8v4="
		err = utils.VerifySignatures(vote, "gonka-testnet-7", validatorsData, signatureData)
		assert.NoError(t, err)
	*/
	cl, err := NewGonkaOpenAI(Options{
		GonkaPrivateKey: "10af8dc1f63fb90cfa39943a5afbf262cd84f24919e7d05653e3b03313e685ce",
		GonkaAddress:    "cosmos1waj8q9g2ekgardafc6plg77rgu2l3vfrclrm4v",
		Endpoints:       []string{"http://localhost:9010"},
		OrgID:           "gonka-client-test-id",
	})
	assert.NoError(t, err)

	participants, err := cl.getParticipants(context.Background(), "0")
	assert.NoError(t, err)

	assert.Equal(t, expectedBlockIdHash, participants.ValidatorsProof.BlockId.Hash)
	blockIdHash, err := hex.DecodeString(participants.ValidatorsProof.BlockId.Hash)
	assert.NoError(t, err)

	assert.Equal(t, expectedPartHeaderHash, participants.ValidatorsProof.BlockId.PartSetHeaderHash)
	partsHeaderHash, err := hex.DecodeString(participants.ValidatorsProof.BlockId.PartSetHeaderHash)
	assert.NoError(t, err)

	vote := tmproto.Vote{
		Type:   tmproto.PrecommitType,
		Height: 1,
		Round:  int32(participants.ValidatorsProof.Round),
		BlockID: tmproto.BlockID{
			Hash: blockIdHash,
			PartSetHeader: tmproto.PartSetHeader{
				Total: uint32(participants.ValidatorsProof.BlockId.PartSetHeaderTotal),
				Hash:  partsHeaderHash,
			},
		},
	}

	signatureData := make([]*contracts.SignatureInfo, 1)
	timeStamp, err := time.Parse(time.RFC3339Nano, "2025-08-17T22:23:27.436384392Z")
	assert.NoError(t, err)

	signatureData[0] = &contracts.SignatureInfo{
		SignatureBase64:     "LZ7UbfQ9r3aGJyfcIP0AjqVu72t3cs68IA/U8bgU8Wa5BZ+gHIReLSj8cNh/E6MEFMRsYg7opYieFPF9bsf0CA==",
		ValidatorAddressHex: "F19FBF8E33D4F9C1CC7CDD5B3593AA1224ACF036",
		Timestamp:           timeStamp,
	}

	validatorsData := make(map[string]string)
	validatorsData["F19FBF8E33D4F9C1CC7CDD5B3593AA1224ACF036"] = "r8nuNlSsk4vazS4If1UBvRM1mH1iPxv3sG4KUWmEjTE="
	err = utils.VerifySignatures(vote, "gonka-testnet-7", validatorsData, signatureData)
	assert.NoError(t, err)

}

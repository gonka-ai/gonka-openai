package gonkaopenai

import (
	"bytes"
	"crypto/ecdsa"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcutil/bech32"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/ripemd160" //nolint:SA1019 // RIPEMD-160 is required for Cosmos address generation, standard despite deprecation.
)

// GonkaBaseURL returns a random endpoint URL from the provided list or environment.
func GonkaBaseURL(endpoints []Endpoint) string {
	eps := make([]Endpoint, 0)
	if len(endpoints) > 0 {
		eps = endpoints
	} else {
		eps = GetEndpointsFromEnv()
	}
	if len(eps) == 0 {
		return ""
	}
	if len(eps) == 1 {
		return eps[0].URL
	}
	n, err := crand.Int(crand.Reader, big.NewInt(int64(len(eps))))
	if err != nil {
		return eps[0].URL
	}
	return eps[int(n.Int64())].URL
}

func GetEndpointsFromEnv() []Endpoint {
	env := os.Getenv(EnvEndpoints)
	if env == "" {
		return DefaultEndpoints
	}
	eps := make([]Endpoint, 0)
	// Parse environment endpoints in format "URL;ADDRESS,URL;ADDRESS"
	for _, e := range strings.Split(env, ",") {
		parts := strings.Split(strings.TrimSpace(e), ";")
		if len(parts) == 2 {
			// Format is "URL;ADDRESS"
			url := strings.TrimSpace(parts[0])
			address := strings.TrimSpace(parts[1])
			eps = append(eps, Endpoint{URL: url, Address: address})
		}
		// No backward compatibility: if no address is provided, the endpoint is ignored
	}
	return eps
}

// CustomEndpointSelection allows providing custom strategy.
func CustomEndpointSelection(f func([]Endpoint) string, endpoints []Endpoint) string {
	eps := endpoints
	if len(eps) == 0 {
		eps = GetEndpointsFromEnv()
	}
	return f(eps)
}

// GonkaSignature signs request body with ECDSA secp256k1 and returns base64.
func GonkaSignature(body []byte, privateKeyHex string) (string, error) {
	if strings.HasPrefix(privateKeyHex, "0x") {
		privateKeyHex = privateKeyHex[2:]
	}
	keyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return "", err
	}
	priv, err := crypto.ToECDSA(keyBytes)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(body)
	r, s, err := ecdsa.Sign(crand.Reader, priv, hash[:])
	if err != nil {
		return "", err
	}
	// Low-S normalization
	curveOrder := priv.Params().N
	if s.Cmp(new(big.Int).Rsh(curveOrder, 1)) == 1 {
		s = new(big.Int).Sub(curveOrder, s)
	}
	sigBytes := append(r.Bytes(), s.Bytes()...)
	return base64.StdEncoding.EncodeToString(sigBytes), nil
}

// GonkaAddress derives a Cosmos bech32 address from private key.
func GonkaAddress(privateKeyHex string) (string, error) {
	if strings.HasPrefix(privateKeyHex, "0x") {
		privateKeyHex = privateKeyHex[2:]
	}
	keyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return "", err
	}
	priv, err := crypto.ToECDSA(keyBytes)
	if err != nil {
		return "", err
	}
	pub := crypto.CompressPubkey(&priv.PublicKey)
	sha := sha256.Sum256(pub)
	hasher := ripemd160.New()
	hasher.Write(sha[:])
	ripe := hasher.Sum(nil)
	five, err := bech32.ConvertBits(ripe[:], 8, 5, true)
	if err != nil {
		return "", err
	}
	prefix := strings.Split(GonkaChainID, "-")[0]
	return bech32.Encode(prefix, five)
}

// SignatureComponents contains the components needed for signature generation
type SignatureComponents struct {
	Payload         string
	Timestamp       int64
	TransferAddress string
}

// getSignatureBytes creates the message payload for signing according to the new method
func getSignatureBytes(components SignatureComponents) []byte {
	// Create message payload by concatenating components
	messagePayload := []byte(components.Payload)
	if components.Timestamp > 0 {
		messagePayload = append(messagePayload, []byte(strconv.FormatInt(components.Timestamp, 10))...)
	}
	messagePayload = append(messagePayload, []byte(components.TransferAddress)...)
	return messagePayload
}

// SignComponentsWithKey combines getSignatureBytes and GonkaSignature to create a signature
// from SignatureComponents using the provided private key.
func SignComponentsWithKey(components SignatureComponents, privateKeyHex string) (string, error) {
	// Get the bytes to sign from the components
	dataToSign := getSignatureBytes(components)

	// Sign the data with the private key
	return GonkaSignature(dataToSign, privateKeyHex)
}

type signingRoundTripper struct {
	rt         http.RoundTripper
	privateKey string
	address    string
	endpoints  []Endpoint
}

func (s signingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Generate timestamp in nanoseconds
	timestamp := time.Now().UnixNano()

	// Determine the appropriate transfer address for this request
	var transferAddress string

	// Extract the base URL from the request URL
	if req.URL != nil {
		baseURL := req.URL.Scheme + "://" + req.URL.Host

		// Find the matching endpoint
		for _, endpoint := range s.endpoints {
			if strings.HasPrefix(endpoint.URL, baseURL) {
				transferAddress = endpoint.Address
				break
			}
		}

		// If no matching endpoint found, we can't proceed
		if transferAddress == "" {
			return nil, fmt.Errorf("no transfer address found for endpoint: %s", baseURL)
		}
	} else {
		return nil, fmt.Errorf("request URL is nil")
	}

	var payload string

	if req.Body != nil {
		data, err := io.ReadAll(req.Body)
		if err == nil {
			payload = string(data)
			components := SignatureComponents{
				Payload:         payload,
				Timestamp:       timestamp,
				TransferAddress: transferAddress,
			}

			sig, err := SignComponentsWithKey(components, s.privateKey)
			if err == nil {
				req.Header.Set("Authorization", sig)
			}
		}
		req.Body = io.NopCloser(bytes.NewReader(data))
	} else {
		components := SignatureComponents{
			Payload:         "",
			Timestamp:       timestamp,
			TransferAddress: transferAddress,
		}

		sig, err := SignComponentsWithKey(components, s.privateKey)
		if err == nil {
			req.Header.Set("Authorization", sig)
		}
	}

	// Set headers
	req.Header.Set("X-Requester-Address", s.address)
	req.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))

	return s.rt.RoundTrip(req)
}

type HTTPClientOptions struct {
	PrivateKey string
	Address    string
	Endpoints  []Endpoint
	Client     *http.Client
}

// GonkaHTTPClient creates an HTTP client that signs requests with the private key.
func GonkaHTTPClient(opts HTTPClientOptions) *http.Client {
	if opts.Client == nil {
		opts.Client = &http.Client{}
	}
	if opts.Address == "" {
		addr, err := GonkaAddress(opts.PrivateKey)
		if err == nil {
			opts.Address = addr
		}
	}
	rt := opts.Client.Transport
	if rt == nil {
		rt = http.DefaultTransport
	}
	opts.Client.Transport = signingRoundTripper{
		rt:         rt,
		privateKey: opts.PrivateKey,
		address:    opts.Address,
		endpoints:  opts.Endpoints,
	}
	return opts.Client
}

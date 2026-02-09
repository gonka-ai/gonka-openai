package gonkaopenai

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcutil/bech32"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/ripemd160" //nolint:SA1019 // RIPEMD-160 is required for Cosmos address generation, standard despite deprecation.
)

// FetchAllowedTransferAddresses fetches the allowed transfer addresses via the node's /chain-api/ proxy.
func FetchAllowedTransferAddresses(ctx context.Context, nodeUrl string) ([]string, error) {
	base := strings.TrimRight(nodeUrl, "/")
	if strings.HasSuffix(base, "/v1") {
		base = base[:len(base)-3]
	}
	url := base + "/chain-api/productscience/inference/inference/params"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chain params request failed with status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result struct {
		Params struct {
			TransferAgentAccessParams struct {
				AllowedTransferAddresses []string `json:"allowed_transfer_addresses"`
			} `json:"transfer_agent_access_params"`
		} `json:"params"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result.Params.TransferAgentAccessParams.AllowedTransferAddresses, nil
}

// NodeIdentity represents the identity data returned by a node's /v1/identity endpoint.
type NodeIdentity struct {
	Address        string            `json:"address"`
	WarmKeyAddress string            `json:"warm_key_address"`
	DelegateTA     map[string]string `json:"delegate_ta"`
}

// FetchNodeIdentity fetches the node identity including delegate_ta.
func FetchNodeIdentity(ctx context.Context, nodeUrl string) (*NodeIdentity, error) {
	base := strings.TrimRight(nodeUrl, "/")
	if strings.HasSuffix(base, "/v1") {
		base = base[:len(base)-3]
	}
	url := base + "/v1/identity"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("identity request failed with status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data NodeIdentity `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func ensureV1(url string) string {
	if url == "" {
		return url
	}
	u := strings.TrimRight(url, "/")
	if strings.HasSuffix(u, "/v1") {
		return u
	}
	return u + "/v1"
}

// CustomEndpointSelection allows providing custom strategy.
func CustomEndpointSelection(f func([]Endpoint) string, endpoints []Endpoint) string {
	eps := endpoints
	return f(eps)
}

// GonkaBaseURL randomly selects an endpoint URL from the provided list.
func GonkaBaseURL(endpoints []Endpoint) string {
	if len(endpoints) == 0 {
		return ""
	}

	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	// Select a random endpoint
	randomIndex := rand.Intn(len(endpoints))
	return endpoints[randomIndex].URL
}

// GetEndpointsFromEnv parses endpoints from GONKA_ENDPOINTS env var in the format "url;address, url;address".
func GetEndpointsFromEnv() []Endpoint {
	env := os.Getenv(EnvEndpoints)
	if env == "" {
		return nil
	}
	var out []Endpoint
	for _, part := range strings.Split(env, ",") {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		segs := strings.SplitN(p, ";", 2)
		if len(segs) != 2 {
			continue
		}
		url := strings.TrimSpace(segs[0])
		addr := strings.TrimSpace(segs[1])
		if url != "" && addr != "" {
			out = append(out, Endpoint{URL: url, Address: addr})
		}
	}
	return out
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
	sum := sha256.Sum256([]byte(components.Payload))
	messagePayload := []byte(hex.EncodeToString(sum[:]))

	messagePayload = append(messagePayload, []byte(strconv.FormatInt(components.Timestamp, 10))...)
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
	SourceUrl  string // URL to fetch endpoints from using GetParticipantsWithProof
}

// GonkaHTTPClient creates an HTTP client that signs requests with the private key.
func GonkaHTTPClient(opts HTTPClientOptions) (*http.Client, error) {
	if opts.Client == nil {
		opts.Client = &http.Client{}
	}
	if opts.Address == "" {
		addr, err := GonkaAddress(opts.PrivateKey)
		if err == nil {
			opts.Address = addr
		}
	}

	// Get endpoints from SourceUrl if provided
	endpoints := opts.Endpoints
	if opts.SourceUrl != "" {
		// SourceUrl takes precedence over Endpoints
		var err error
		endpoints, err = GetParticipantsWithProof(context.Background(), opts.SourceUrl, "current")
		if err != nil {
			return nil, fmt.Errorf("failed to get participants with proof: %w", err)
		}

		// Ensure we got at least one endpoint
		if len(endpoints) == 0 {
			return nil, fmt.Errorf("no endpoints found from SourceUrl: %s", opts.SourceUrl)
		}
	}

	// Ensure we have at least one endpoint
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("at least one endpoint must be provided via Endpoints or SourceUrl")
	}

	// Validate that each endpoint has a non-empty address
	for _, endpoint := range endpoints {
		if endpoint.Address == "" {
			return nil, fmt.Errorf("endpoint %s has an empty address, all endpoints must have an address", endpoint.URL)
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
		endpoints:  endpoints,
	}
	return opts.Client, nil
}

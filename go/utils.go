package gonkaopenai

import (
	"bytes"
	"crypto/ecdsa"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"

	"github.com/btcsuite/btcd/btcutil/bech32"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/ripemd160" //nolint:SA1019 // RIPEMD-160 is required for Cosmos address generation, standard despite deprecation.
)

// GonkaBaseURL returns a random endpoint from the provided list or environment.
func GonkaBaseURL(endpoints []string) string {
	eps := make([]string, 0)
	if len(endpoints) > 0 {
		eps = endpoints
	} else if env := os.Getenv(EnvEndpoints); env != "" {
		for _, e := range strings.Split(env, ",") {
			eps = append(eps, strings.TrimSpace(e))
		}
	} else {
		eps = DefaultEndpoints
	}
	if len(eps) == 0 {
		return ""
	}
	if len(eps) == 1 {
		return eps[0]
	}
	n, err := crand.Int(crand.Reader, big.NewInt(int64(len(eps))))
	if err != nil {
		return eps[0]
	}
	return eps[int(n.Int64())]
}

// CustomEndpointSelection allows providing custom strategy.
func CustomEndpointSelection(f func([]string) string, endpoints []string) string {
	eps := endpoints
	if len(eps) == 0 {
		if env := os.Getenv(EnvEndpoints); env != "" {
			for _, e := range strings.Split(env, ",") {
				eps = append(eps, strings.TrimSpace(e))
			}
		} else {
			eps = DefaultEndpoints
		}
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

type signingRoundTripper struct {
	rt         http.RoundTripper
	privateKey string
	address    string
}

func (s signingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		data, err := io.ReadAll(req.Body)
		if err == nil {
			sig, err := GonkaSignature(data, s.privateKey)
			if err == nil {
				req.Header.Set("Authorization", sig)
			}
		}
		req.Body = io.NopCloser(bytes.NewReader(data))
	} else {
		sig, err := GonkaSignature([]byte{}, s.privateKey)
		if err == nil {
			req.Header.Set("Authorization", sig)
		}
	}
	req.Header.Set("X-Requester-Address", s.address)
	return s.rt.RoundTrip(req)
}

type HTTPClientOptions struct {
	PrivateKey string
	Address    string
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
	opts.Client.Transport = signingRoundTripper{rt: rt, privateKey: opts.PrivateKey, address: opts.Address}
	return opts.Client
}

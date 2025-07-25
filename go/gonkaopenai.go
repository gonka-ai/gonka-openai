package gonkaopenai

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// Endpoint represents a Gonka API endpoint with its associated transfer address
type Endpoint struct {
	URL     string
	Address string
}

// Options for creating a GonkaOpenAI client.
type Options struct {
	APIKey                    string
	GonkaPrivateKey           string
	GonkaAddress              string
	Endpoints                 []Endpoint // List of endpoints with their transfer addresses
	EndpointSelectionStrategy func([]Endpoint) string
	HTTPClient                *http.Client
	OrgID                     string
}

// GonkaOpenAI wraps the official openai.Client.
type GonkaOpenAI struct {
	*openai.Client
	privateKey string
	gonkaAddr  string
}

// NewGonkaOpenAI creates a new client configured for the Gonka network.
func NewGonkaOpenAI(opts Options) (*GonkaOpenAI, error) {
	privateKey := opts.GonkaPrivateKey
	if privateKey == "" {
		privateKey = os.Getenv(EnvPrivateKey)
	}
	if privateKey == "" {
		return nil, fmt.Errorf("private key must be provided via opts or %s", EnvPrivateKey)
	}

	// Validate that we have endpoints with addresses
	if len(opts.Endpoints) == 0 {
		return nil, fmt.Errorf("at least one endpoint with address must be provided")
	}

	// Validate that each endpoint has a non-empty address
	for _, endpoint := range opts.Endpoints {
		if endpoint.Address == "" {
			return nil, fmt.Errorf("endpoint %s has an empty address, all endpoints must have an address", endpoint.URL)
		}
	}

	baseURL := ""
	if opts.EndpointSelectionStrategy != nil {
		baseURL = CustomEndpointSelection(opts.EndpointSelectionStrategy, opts.Endpoints)
	} else {
		baseURL = GonkaBaseURL(opts.Endpoints)
	}

	address := opts.GonkaAddress
	if address == "" {
		address = os.Getenv(EnvAddress)
	}
	if address == "" {
		addr, err := GonkaAddress(privateKey)
		if err == nil {
			address = addr
		} else {
			prefix := strings.Split(GonkaChainID, "-")[0]
			if len(privateKey) > 40 {
				address = fmt.Sprintf("%s1%s", prefix, privateKey[:40])
			} else {
				address = fmt.Sprintf("%s1%s", prefix, privateKey)
			}
		}
	}

	httpClient := GonkaHTTPClient(HTTPClientOptions{
		PrivateKey: privateKey,
		Address:    address,
		Endpoints:  opts.Endpoints,
		Client:     opts.HTTPClient,
	})

	var clientOptions []option.RequestOption
	clientOptions = append(clientOptions, option.WithBaseURL(baseURL))
	clientOptions = append(clientOptions, option.WithHTTPClient(httpClient))

	apiKey := opts.APIKey
	if apiKey == "" {
		apiKey = "mock-api-key" // Default mock API key if not provided
	}
	clientOptions = append(clientOptions, option.WithAPIKey(apiKey))

	if opts.OrgID != "" {
		clientOptions = append(clientOptions, option.WithOrganization(opts.OrgID))
	}

	rawClient := openai.NewClient(clientOptions...)
	return &GonkaOpenAI{Client: &rawClient, privateKey: privateKey, gonkaAddr: address}, nil
}

// GonkaAddress returns the configured Gonka address.
func (g *GonkaOpenAI) GonkaAddress() string { return g.gonkaAddr }

// PrivateKey returns the private key used for signing.
func (g *GonkaOpenAI) PrivateKey() string { return g.privateKey }

// ExampleChatCompletion demonstrates a simple call using the Gonka client.
func ExampleChatCompletion(ctx context.Context, g *GonkaOpenAI) (*openai.ChatCompletion, error) {
	return g.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: "Qwen/QwQ-32B",
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("Hello!"), // Use UserMessage helper
		},
	})
}

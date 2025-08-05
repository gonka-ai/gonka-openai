package gonkaopenai_test

import (
	"context"
	gonkaopenai "github.com/gonka-ai/gonka-openai/go"
	"github.com/joho/godotenv"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"os"
	"testing"
)

var defaultModel = "Qwen/QwQ-32B" // Default model for testing
func TestExampleUsage(t *testing.T) {
	err := godotenv.Load()
	if err != nil {
		t.Log("Note: .env file not found or could not be loaded. Proceeding with existing environment variables.")
	}

	if os.Getenv(gonkaopenai.EnvPrivateKey) == "" {
		t.Log("Missing required environment variable: GONKA_PRIVATE_KEY")
		t.Skip("Skipping test: Missing GONKA_PRIVATE_KEY") // Use t.Skip for conditional test skipping
		return
	}

	t.Log("\n------ Test Environment ------") // Use t.Log for test output

	// Define a source URL for fetching endpoints
	sourceUrl := "http://localhost:9000"
	t.Log("Using Source URL:", sourceUrl)

	// The APIKey is often a mock or test-specific key in test environments
	client, err := gonkaopenai.NewGonkaOpenAI(gonkaopenai.Options{
		GonkaPrivateKey: os.Getenv(gonkaopenai.EnvPrivateKey),
		SourceUrl:       sourceUrl,
	})
	if err != nil {
		t.Fatalf("Error creating client: %v", err) // Use t.Fatalf to fail the test on critical errors
		return
	}

	t.Log("\nSending request...")
	resp, err := client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
		Model: defaultModel, // Model as a string, consistent with gonkaopenai.go
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("Hello! Tell me a short joke for a test."), // Using UserMessage helper
		},
	})
	if err != nil {
		t.Fatalf("Error during API call: %v", err)
		return
	}

	if len(resp.Choices) == 0 {
		t.Fatal("Expected at least one choice in the response, got none.")
		return
	}

	t.Log("\nResponse:")
	t.Log(resp.Choices[0].Message.Content)
}

// TestDirectOpenAIUsage tests the OpenAI API directly without the Gonka wrapper.
func TestDirectOpenAIUsage(t *testing.T) {
	// Attempt to load .env file
	err := godotenv.Load()
	if err != nil {
		t.Log("Note: .env file not found or could not be loaded. Proceeding with existing environment variables.")
	}

	// 1. Get Gonka Private Key
	gonkaPrivateKey := os.Getenv(gonkaopenai.EnvPrivateKey)
	if gonkaPrivateKey == "" {
		t.Log("Missing required environment variable: GONKA_PRIVATE_KEY for Gonka HTTP client setup.")
		t.Skip("Skipping test: Missing GONKA_PRIVATE_KEY")
		return
	}

	t.Log("\n------ Test Manually Configured Gonka Client (using openai.Client) ------")

	// 2. Determine Gonka Address (mirroring logic from NewGonkaOpenAI)
	gonkaAddress := os.Getenv(gonkaopenai.EnvAddress)
	if gonkaAddress == "" {
		addr, errAddr := gonkaopenai.GonkaAddress(gonkaPrivateKey)
		if errAddr == nil {
			gonkaAddress = addr
		} else {
			t.Logf("Could not derive GonkaAddress automatically (err: %v), fallback might be incomplete without GonkaChainID", errAddr)
		}
	}
	if gonkaAddress == "" {
		t.Log("Warning: GonkaAddress could not be determined. GonkaHTTPClient might fail or use defaults.")
	}

	// 3. Define a source URL for fetching endpoints
	sourceUrl := "http://localhost:9000"
	t.Log("Using Source URL:", sourceUrl)

	t.Log("Using Gonka Private Key (for HTTP client):", gonkaPrivateKey[:5]+"...") // Log a snippet for verification
	t.Log("Using Gonka Address (for HTTP client):", gonkaAddress)

	// 4. Create Gonka HTTP Client with SourceUrl
	customHTTPClient, err := gonkaopenai.GonkaHTTPClient(gonkaopenai.HTTPClientOptions{
		PrivateKey: gonkaPrivateKey,
		Address:    gonkaAddress,
		SourceUrl:  sourceUrl, // Use SourceUrl directly instead of fetching endpoints separately
		Client:     nil,       // No base client override for this test
	})
	if err != nil {
		t.Fatalf("Error creating HTTP client: %v", err)
		return
	}
	t.Log("Custom Gonka HTTP Client configured with SourceUrl.")

	// Get endpoints for baseURL (could also be done with GetParticipantsWithProof)
	endpoints, err := gonkaopenai.GetParticipantsWithProof(context.Background(), sourceUrl, "current")
	if err != nil {
		t.Fatalf("Error fetching endpoints: %v", err)
		return
	}

	// 5. Initialize OpenAI Client with Gonka settings
	// Using the "mock-api-key" as per your previous change.
	// If you want to use a real key for this Gonka setup, change it here or load from env.
	clientAPIKey := "mock-api-key"
	var clientOptions []option.RequestOption
	baseURL := gonkaopenai.GonkaBaseURL(endpoints) // Get a random endpoint URL
	clientOptions = append(clientOptions, option.WithBaseURL(baseURL))
	clientOptions = append(clientOptions, option.WithHTTPClient(customHTTPClient))
	clientOptions = append(clientOptions, option.WithAPIKey(clientAPIKey))

	client := openai.NewClient(clientOptions...)
	t.Log("Manually configured Gonka-like client created with API Key:", clientAPIKey)

	t.Log("Sending request with manually configured Gonka-like client...")
	resp, err := client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
		Model: defaultModel, // Using the same model as TestExampleUsage
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("Hello Gonka-configured client! Tell me a very short story."),
		},
	})
	if err != nil {
		t.Fatalf("Error during API call with Gonka-configured client: %v", err)
		return
	}

	if len(resp.Choices) == 0 {
		t.Fatal("Expected at least one choice in the response, got none.")
		return
	}

	t.Log("\nResponse from Gonka-configured client:")
	t.Log(resp.Choices[0].Message.Content)
}

# Gonka OpenAI for Go

A Go library for using OpenAI's API through the Gonka network.

## Installation

```bash
go get github.com/libermans/gonka-openai/go
```

## Usage

There are two ways to use this library:

### Option 1: Using the GonkaOpenAI wrapper (recommended)

```go
package main

import (
    "context"
    gonkaopenai "github.com/libermans/gonka-openai/go"
)

func main() {
    // Private key can be provided directly or through environment variable GONKA_PRIVATE_KEY
    client, err := gonkaopenai.NewGonkaOpenAI(gonkaopenai.Options{
        GonkaPrivateKey: "0x1234...", // ECDSA private key for signing requests
        Endpoints: []gonkaopenai.Endpoint{
            {URL: "https://gonka1.example.com/v1", Address: "provider_address_1"},
            {URL: "https://gonka2.example.com/v1", Address: "provider_address_2"},
        }, // List of endpoints with their provider addresses
        // Optional parameters:
        // GonkaAddress: "cosmos1...", // Override derived Cosmos address
    })
    if err != nil {
        panic(err)
    }

    resp, err := client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
        Model: "Qwen/QwQ-32B",
        Messages: []openai.ChatCompletionMessageParamUnion{
            openai.UserMessage("Hello!"),
        },
    })
    if err != nil {
        panic(err)
    }

    println(chatCompletion.Choices[0].Message.Content)
}
```

### Option 2: Using the original OpenAI client with a custom HTTP client

```go
package main

import (
    "context"
    "github.com/openai/openai-go"
    "github.com/openai/openai-go/option"
    gonkaopenai "github.com/libermans/gonka-openai/go"
)

func main() {
    // Define endpoints with their provider addresses
    endpoints := []gonkaopenai.Endpoint{
        {URL: "https://gonka1.example.com/v1", Address: "provider_address_1"},
        {URL: "https://gonka2.example.com/v1", Address: "provider_address_2"},
    }

    httpClient := gonkaopenai.GonkaHTTPClient(gonkaopenai.HTTPClientOptions{
        PrivateKey: "0x1234...",
        Endpoints:  endpoints, // List of endpoints with their provider addresses
    })

    client := openai.NewClient(
        option.WithAPIKey("mock-api-key"), // OpenAI requires any key
        option.WithBaseURL(gonkaopenai.GonkaBaseURL(endpoints)), // Randomly selects an endpoint URL
        option.WithHTTPClient(httpClient),
    )

    chatCompletion, err := client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
        Model: "Qwen/QwQ-32B",
        Messages: []openai.ChatCompletionMessageParamUnion{
            openai.UserMessage("Hello!"),
        },
    })
    if err != nil {
        panic(err)
    }

    println(chatCompletion.Choices[0].Message.Content)
}
```

This approach provides the same dynamic request signing as Option 1, but gives you more direct control over the OpenAI client configuration.

## Environment Variables

Instead of passing configuration directly, you can use environment variables:

- `GONKA_PRIVATE_KEY`: Your ECDSA private key for signing requests
- `GONKA_ADDRESS`: (Optional) Override the derived Cosmos address
- `GONKA_ENDPOINTS`: (Optional) Comma-separated list of Gonka network endpoints with their provider addresses in the format "URL;ADDRESS,URL;ADDRESS" (e.g., "https://myendpoint.com;gonka1fgjkdsafjalf,https://anotherendpoint.com;gonka2abcdefghijk")

## Advanced Configuration

### Custom Endpoint Selection

You can provide a custom endpoint selection strategy:

```go
client, err := gonkaopenai.NewGonkaOpenAI(gonkaopenai.Options{
    GonkaPrivateKey: "0x1234...",
    Endpoints: []gonkaopenai.Endpoint{
        {URL: "https://gonka1.example.com/v1", Address: "provider_address_1"},
        {URL: "https://gonka2.example.com/v1", Address: "provider_address_2"},
    },
    EndpointSelectionStrategy: func(endpoints []gonkaopenai.Endpoint) string {
        return endpoints[0].URL // Always select the first endpoint's URL
    },
})
```

### Endpoint Configuration

Each endpoint must have an associated provider address for signature generation. The `Endpoint` type pairs a URL with its provider address:

```go
// Define endpoints with their provider addresses
endpoints := []gonkaopenai.Endpoint{
    {URL: "https://api.gonka.testnet.example.com", Address: "provider_address_1"},
    {URL: "https://api2.gonka.testnet.example.com", Address: "provider_address_2"},
    {URL: "https://api3.gonka.testnet.example.com", Address: "provider_address_3"},
}

// Use with NewGonkaOpenAI
client, err := gonkaopenai.NewGonkaOpenAI(gonkaopenai.Options{
    GonkaPrivateKey: "0x1234...",
    Endpoints:       endpoints,
})
```

Or when using the GonkaHTTPClient directly:

```go
httpClient := gonkaopenai.GonkaHTTPClient(gonkaopenai.HTTPClientOptions{
    PrivateKey: "0x1234...",
    Endpoints:  endpoints,
})
```

This approach ensures that each request is signed with the appropriate provider address for the endpoint it's targeting.

## Building from Source

```bash
git clone https://github.com/yourusername/gonka-openai.git
cd gonka-openai/go
go build ./...
```

## Testing

A simple example program is provided in `test.go`:

```bash
cd go
go run ./example_test.go
```

## License

MIT

# Gonka OpenAI for Go

A Go library for using OpenAI's API through the Gonka network.

## Installation

```bash
go get github.com/gonka-ai/gonka-openai/go
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
    // Private key and SourceUrl can be provided directly or through environment variables
    // GONKA_PRIVATE_KEY and GONKA_SOURCE_URL respectively
    client, err := gonkaopenai.NewGonkaOpenAI(gonkaopenai.Options{
        GonkaPrivateKey: "0x1234...", // ECDSA private key for signing requests
        SourceUrl: "https://api.gonka.testnet.example.com", // Resolve endpoints from this SourceUrl
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
    gonkaopenai "github.com/gonka-ai/gonka-openai/go"
)

func main() {
    // You can use SourceUrl directly in GonkaHTTPClient
    sourceUrl := "https://api.gonka.testnet.example.com"
    
    // GonkaHTTPClient will fetch endpoints from SourceUrl
    httpClient, err := gonkaopenai.GonkaHTTPClient(gonkaopenai.HTTPClientOptions{
        PrivateKey: "0x1234...",
        SourceUrl:  sourceUrl,
    })
    if err != nil {
        panic(err)
    }
    
    // Get endpoints for baseURL
    endpoints, err := gonkaopenai.GetParticipantsWithProof(context.Background(), sourceUrl, "current")
    if err != nil {
        panic(err)
    }

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
- `GONKA_SOURCE_URL`: (Optional) URL to fetch endpoints from
- `GONKA_VERIFY_PROOF`: (Optional) Set to `1` to enable ICS23 proof verification during endpoint discovery. If unset, verification is skipped by default.
- `GONKA_ADDRESS`: (Optional) Override the derived Cosmos address

## Advanced Configuration

### Custom Endpoint Selection

You can provide a custom endpoint selection strategy for the endpoints fetched from `SourceUrl`:

```go
client, err := gonkaopenai.NewGonkaOpenAI(gonkaopenai.Options{
    GonkaPrivateKey: "0x1234...",
    SourceUrl: "https://api.gonka.testnet.example.com",
    EndpointSelectionStrategy: func(endpoints []gonkaopenai.Endpoint) string {
        return endpoints[0].URL // Always select the first endpoint's URL
    },
})
```

Note: The `EndpointSelectionStrategy` field is deprecated but still functional. It will be applied to the endpoints fetched from `SourceUrl`.

### Endpoint Configuration

Endpoints are now exclusively fetched from the `SourceUrl` parameter using the `GetParticipantsWithProof` function. This ensures that all endpoints are properly verified and authenticated.

You can either:

1. Fetch endpoints yourself and pass them to `GonkaHTTPClient`:

```go
// Get endpoints from SourceUrl
sourceUrl := "https://api.gonka.testnet.example.com"
endpoints, err := gonkaopenai.GetParticipantsWithProof(context.Background(), sourceUrl, "current")
if err != nil {
    panic(err)
}

// Use with GonkaHTTPClient directly
httpClient, err := gonkaopenai.GonkaHTTPClient(gonkaopenai.HTTPClientOptions{
    PrivateKey: "0x1234...",
    Endpoints:  endpoints,
})
if err != nil {
    panic(err)
}
```

2. Or let `GonkaHTTPClient` fetch the endpoints for you using `SourceUrl`:

```go
// Let GonkaHTTPClient fetch endpoints for you
httpClient, err := gonkaopenai.GonkaHTTPClient(gonkaopenai.HTTPClientOptions{
    PrivateKey: "0x1234...",
    SourceUrl:  "https://api.gonka.testnet.example.com", // URL to fetch endpoints from
})
if err != nil {
    panic(err)
}
```

Both approaches ensure that each request is signed with the appropriate provider address for the endpoint it's targeting, and that all endpoints are properly verified.

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

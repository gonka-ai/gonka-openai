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
        Endpoints: []string{"https://gonka1.example.com/v1"}, // Gonka endpoints
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
    httpClient := gonkaopenai.GonkaHTTPClient(gonkaopenai.HTTPClientOptions{
        PrivateKey: "0x1234...",
    })

    client := openai.NewClient(
        option.WithAPIKey("mock-api-key"), // OpenAI requires any key
        option.WithBaseURL(gonkaopenai.GonkaBaseURL([]string{"https://gonka1.example.com/v1"})),
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
- `GONKA_ENDPOINTS`: (Optional) Comma-separated list of Gonka network endpoints

## Advanced Configuration

### Custom Endpoint Selection

You can provide a custom endpoint selection strategy:

```go
client, err := gonkaopenai.NewGonkaOpenAI(gonkaopenai.Options{
    GonkaPrivateKey: "0x1234...",
    EndpointSelectionStrategy: func(endpoints []string) string {
        return endpoints[0] // Always select the first
    },
})
```

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
go run ./test.go
```

## License

MIT

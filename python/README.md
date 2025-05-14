# Gonka OpenAI for Python

A Python library for using OpenAI's API through the Gonka network.

## Installation

```bash
pip install gonka-openai
# or
poetry add gonka-openai
```

## Usage

There are two ways to use this library:

### Option 1: Using the GonkaOpenAI wrapper (recommended)

```python
from gonka_openai import GonkaOpenAI

# Private key can be provided directly or through environment variable GONKA_PRIVATE_KEY
client = GonkaOpenAI(
    api_key="mock-api-key",
    gonka_private_key="0x1234...",  # ECDSA private key for signing requests
    # Optional parameters:
    # gonka_address="cosmos1...",  # Override derived Cosmos address
    # endpoints=["https://gonka1.example.com", "https://gonka2.example.com"],  # Custom endpoints
)

# Use exactly like the original OpenAI client
response = client.chat.completions.create(
    model="Qwen/QwQ-32B",
    messages=[{"role": "user", "content": "Hello!"}],
)
```

### Option 2: Using the original OpenAI client with a custom HTTP client

```python
from openai import OpenAI
from gonka_openai import gonka_base_url, gonka_http_client

# Create a custom HTTP client for Gonka with your private key
http_client = gonka_http_client(
    private_key="0x1234...",  # Your private key
    address="cosmos1..."  # Optional address, will derive from private key if not provided
)

# Create an OpenAI client with the custom HTTP client
client = OpenAI(
    api_key="mock-api-key",
    base_url=gonka_base_url(),  # Use Gonka network endpoints
    http_client=http_client  # Use the custom HTTP client that signs requests
)

# Use normally - all requests will be dynamically signed and routed through Gonka
response = client.chat.completions.create(
    model="Qwen/QwQ-32B",
    messages=[{"role": "user", "content": "Hello!"}],
)
```

This approach provides the same dynamic request signing as Option 1, but gives you more direct control over the OpenAI client configuration.

## Environment Variables

Instead of passing configuration directly, you can use environment variables:

- `GONKA_PRIVATE_KEY`: Your ECDSA private key for signing requests
- `GONKA_ADDRESS`: (Optional) Override the derived Cosmos address
- `GONKA_ENDPOINTS`: (Optional) Comma-separated list of Gonka network endpoints

Example with environment variables:

```python
# Set in your environment:
# GONKA_PRIVATE_KEY=0x1234...
# GONKA_ENDPOINTS=https://gonka1.example.com,https://gonka2.example.com

from gonka_openai import GonkaOpenAI

client = GonkaOpenAI(
    api_key="mock-api-key",
    # No need to provide private_key, it will be read from environment
)

# Use normally
response = client.chat.completions.create(
    model="Qwen/QwQ-32B",
    messages=[{"role": "user", "content": "Hello!"}],
)
```

## Advanced Configuration

### Custom Endpoint Selection

You can provide a custom endpoint selection strategy:

```python
from gonka_openai import GonkaOpenAI

def first_endpoint_strategy(endpoints):
    """Always select the first endpoint."""
    return endpoints[0]

client = GonkaOpenAI(
    api_key="mock-api-key",
    gonka_private_key="0x1234...",
    endpoint_selection_strategy=first_endpoint_strategy
)
```

## How It Works

1. **Custom HTTP Client**: The library intercepts all outgoing API requests by wrapping the HTTP client's request method
2. **Request Body Signing**: For each request, the library:
   - Extracts the request body
   - Signs it with your private key using ECDSA
   - Adds the signature to the `Authorization` header
3. **Address Generation**: Your Cosmos address (derived from your private key) is added to the `X-Requester-Address` header
4. **Endpoint Selection**: Requests are routed to the Gonka network using a randomly selected endpoint

## Cryptographic Implementation

The library implements:

1. **ECDSA Signatures**: Using Secp256k1 curve to sign request bodies with the private key
2. **Cosmos Address Generation**: Deriving Cosmos-compatible addresses from private keys using standard bech32 encoding
3. **Dynamic Request Signing**: Using a custom HTTP client implementation to intercept and sign each request before it's sent

## Limitations

The current implementation has a few limitations:

1. **Installation Requirements**: The secp256k1 package requires C dependencies to be installed on your system
2. **Body Extraction**: Some complex body types may use simplified representations for signing
3. **Error Handling**: Error handling is basic and could be improved in future versions
4. **Testing**: Comprehensive testing is recommended before using in production

## Dependencies

- `openai`: Official OpenAI Python client
- `secp256k1`: For ECDSA signature generation
- `bech32`: For Cosmos address encoding
- `python-dotenv`: For environment variable loading

## Building from Source

```bash
git clone https://github.com/yourusername/gonka-openai.git
cd gonka-openai/python
pip install -e .
```

## Testing

To run a simple test that demonstrates the client:

```bash
cd python
python test.py
```

## License

MIT 
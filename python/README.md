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
    api_key="mock-api-key",  # OpenAI requires any key, defaults to "mock-api-key" if not provided
    gonka_private_key="0x1234...",  # ECDSA private key for signing requests
    source_url="https://api.gonka.testnet.example.com",  # Resolve endpoints from this SourceUrl
    # Optional parameters:
    # http_client=custom_client,  # Optional custom HTTP client
)

# Use exactly like the original OpenAI client
response = client.chat.completions.create(
    model="Qwen/QwQ-32B",
    messages=[{"role": "user", "content": "Hello! Tell me a short joke."}],
)
```

### Option 2: Using the original OpenAI client with a custom HTTP client

```python
from openai import OpenAI
from gonka_openai import gonka_http_client, resolve_and_select_endpoint

source_url = "https://api.gonka.testnet.example.com"
endpoints, selected = resolve_and_select_endpoint(source_url=source_url)

# Create a custom HTTP client for Gonka with your private key
http_client = gonka_http_client(
    private_key="0x1234...",  # Your private key
    transfer_address=selected.address  # Provider address from the endpoint
    # Optional parameters:
    # http_client=custom_client,  # Optional custom HTTP client
)

# Create an OpenAI client with the custom HTTP client
client = OpenAI(
    api_key="mock-api-key",  # OpenAI requires any key
    base_url=selected.url,  # Use the URL from the endpoint
    http_client=http_client  # Use the custom HTTP client that signs requests
)

# Use normally - all requests will be dynamically signed and routed through Gonka
response = client.chat.completions.create(
    model="Qwen/QwQ-32B",
    messages=[{"role": "user", "content": "What is the capital of France?"}],
)
```

This approach provides the same dynamic request signing as Option 1, but gives you more direct control over the OpenAI client configuration.

## Environment Variables

Instead of passing configuration directly, you can use environment variables:

- `GONKA_PRIVATE_KEY`: Your ECDSA private key for signing requests
- `GONKA_SOURCE_URL`: URL to fetch participants with proof and resolve endpoints (e.g., "https://gonka1.example.com")
- `GONKA_VERIFY_PROOF`: (Optional) Set to `1` to enable ICS23 proof verification during endpoint discovery. If unset, verification is skipped by default.
- `GONKA_ADDRESS`: (Optional) Override the derived gonka address
- `GONKA_ENDPOINTS`: (Optional) Comma-separated list of Gonka network endpoints in the format "url;address" where address is the provider's gonka address (e.g., "https://gonka1.example.com;gonka1address"). Each endpoint MUST include a provider address.

Example with environment variables:

```python
# Set in your environment:
# GONKA_PRIVATE_KEY=0x1234...
# GONKA_SOURCE_URL=https://gonka1.example.com

from gonka_openai import GonkaOpenAI

client = GonkaOpenAI(
    api_key="mock-api-key",
    # No need to provide endpoints, SourceUrl will be read from environment
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
from gonka_openai.utils import Endpoint

# Create endpoints directly
endpoints = [
    Endpoint(url="https://api.gonka.testnet.example.com", address="gonka1transferaddress"),
    Endpoint(url="https://api2.gonka.testnet.example.com", address="gonka2transferaddress")
]

def first_endpoint_strategy(endpoints):
    """Always select the first endpoint."""
    return endpoints[0]

client = GonkaOpenAI(
    api_key="mock-api-key",
    gonka_private_key="0x1234...",
    endpoints=endpoints,
    endpoint_selection_strategy=first_endpoint_strategy
)

# Use normally
response = client.chat.completions.create(
    model="Qwen/QwQ-32B",
    messages=[{"role": "user", "content": "Hello! Tell me a short joke."}],
)
```

## How It Works

1. **Custom HTTP Client**: The library intercepts all outgoing API requests by wrapping the HTTP client's request method
2. **Request Body Signing**: For each request, the library:
   - Extracts the request body
   - Generates a hybrid timestamp in nanoseconds (required for all requests)
     - Uses a combination of wall clock time and performance counter
     - Ensures timestamps are unique and monotonically increasing
     - Maintains accuracy to standard time servers within at least 30 seconds
   - Concatenates the request body, timestamp, and provider address
   - Signs the concatenated data with your private key using ECDSA
   - Adds the signature to the `Authorization` header
3. **Headers**: The library adds the following headers to each request:
   - `X-Requester-Address`: Your gonka address (derived from your private key)
   - `X-Timestamp`: The timestamp in nanoseconds used for signing (required for all requests)
4. **Endpoint Selection**: Requests are routed to the Gonka network using a randomly selected endpoint
5. **Provider Address**: Each endpoint MUST include a provider address (the gonka address of the provider at that endpoint). This is a required parameter for all requests and will cause requests to fail if not provided.

## Cryptographic Implementation

The library implements:

1. **ECDSA Signatures**: Using Secp256k1 curve to sign request bodies with the private key
2. **Gonka Address Generation**: Deriving gonka-compatible addresses from private keys using standard bech32 encoding

## Participants with proof (standalone)

```python
from gonka_openai import get_participants_with_proof

# Fetch participants from a base URL for an epoch (e.g., "current")
endpoints = get_participants_with_proof(
    base_url="http://localhost:9000",
    epoch="current",
)

for e in endpoints:
    print(e.url, e.address)
```
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
- `bech32`: For gonka address encoding
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
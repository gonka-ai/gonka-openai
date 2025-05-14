# Gonka OpenAI TypeScript Client

A TypeScript client for integrating with the Gonka network and OpenAI API.

## Installation

```bash
npm install
```

## Building

```bash
npm run build
```

## Testing

### Option 1: Running the MJS Test (Recommended)

This test utilizes the pre-compiled JavaScript and properly handles environment variables:

```bash
# Test with mock responses (no real API calls)
npm run test:ts

# Test with real API calls
npm run test:ts:real
```

## Usage

There are two ways to use this library:

### Option 1: Using the GonkaOpenAI wrapper (recommended)

```typescript
import { GonkaOpenAI } from 'gonka-openai';

// Private key can be provided directly or through environment variable GONKA_PRIVATE_KEY
const client = new GonkaOpenAI({
  apiKey: 'your-openai-api-key',
  gonkaPrivateKey: '0x1234...', // ECDSA private key for signing requests
  // Optional parameters:
  // gonkaAddress: 'cosmos1...', // Override derived Cosmos address
  // endpoints: ['https://gonka1.example.com', 'https://gonka2.example.com'], // Custom endpoints
});

// Use exactly like the original OpenAI client
const response = await client.chat.completions.create({
  model: 'gpt-4o',
  messages: [{ role: 'user', content: 'Hello!' }],
});
```

### Option 2: Using the original OpenAI client with a custom fetch function

```typescript
import OpenAI from 'openai';
import { gonkaBaseURL, gonkaFetch } from 'gonka-openai';

// Create a custom fetch function for Gonka with your private key
const fetch = gonkaFetch({
  gonkaPrivateKey: '0x1234...' // Your private key
});

// Create an OpenAI client with the custom fetch function
const client = new OpenAI({
  apiKey: 'your-openai-api-key',
  baseURL: gonkaBaseURL(), // Use Gonka network endpoints 
  fetch: fetch // Use the custom fetch function that signs requests
});

// Use normally - all requests will be dynamically signed and routed through Gonka
const response = await client.chat.completions.create({
  model: 'gpt-4o',
  messages: [{ role: 'user', content: 'Hello!' }],
});
```

This approach provides the same dynamic request signing as Option 1, but gives you more direct control over the OpenAI client configuration.

## Environment Variables

Instead of passing configuration directly, you can use environment variables:

- `GONKA_PRIVATE_KEY`: Your ECDSA private key for signing requests
- `GONKA_ENDPOINTS`: (Optional) Comma-separated list of Gonka network endpoints
- `GONKA_ADDRESS`: (Optional) Override the derived Cosmos address

Example with environment variables:

```typescript
// Set in your environment:
// GONKA_PRIVATE_KEY=0x1234...
// GONKA_ENDPOINTS=https://gonka1.example.com,https://gonka2.example.com

import { GonkaOpenAI } from 'gonka-openai';

const client = new GonkaOpenAI({
  apiKey: 'your-openai-api-key',
  // No need to provide privateKey, it will be read from environment
});

// Use normally
const response = await client.chat.completions.create({
  model: 'gpt-4o',
  messages: [{ role: 'user', content: 'Hello!' }],
});
```

## Advanced Configuration

### Custom Endpoint Selection

You can provide a custom endpoint selection strategy:

```typescript
import { GonkaOpenAI } from 'gonka-openai';

const client = new GonkaOpenAI({
  apiKey: 'your-openai-api-key',
  gonkaPrivateKey: '0x1234...',
  endpointSelectionStrategy: (endpoints) => {
    // Custom selection logic
    return endpoints[0]; // Always use the first endpoint
  }
});
```

## How It Works

1. **Custom Fetch Implementation**: The library intercepts all outgoing API requests by providing a custom `fetch` implementation to the OpenAI client
2. **Request Body Signing**: For each request, the library:
   - Extracts the request body
   - Signs it with your private key using ECDSA
   - Adds the signature to the `Authorization` header
3. **Address Generation**: Your Cosmos address (derived from your private key) is added to the `X-Requester-Address` header
4. **Endpoint Selection**: Requests are routed to the Gonka network using a randomly selected endpoint

## Cryptographic Implementation

The library implements:

1. **ECDSA Signatures**: Using Secp256k1 curve to sign request bodies with the private key
2. **Gonka Address Generation**: Deriving Cosmos-compatible addresses from private keys
3. **Dynamic Request Signing**: Using a custom fetch implementation to intercept and sign each request before it's sent

## Building from Source

```bash
git clone https://github.com/yourusername/gonka-openai.git
cd gonka-openai/typescript
npm install
npm run build
```

## Testing

To run a simple test that demonstrates the client:

```bash
node test.mjs
```

## License

MIT 
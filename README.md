# Gonka OpenAI

This project provides modified OpenAI clients for various programming languages that transparently route API requests through the Gonka network instead of directly to OpenAI.

## Features

- Drop-in replacements for official OpenAI SDK clients
- Automatic request signing with ECDSA
- Gonka address generation from private keys
- Dynamic endpoint selection
- Compatible with the original OpenAI client interfaces

## Available Implementations

- [TypeScript](./typescript/README.md)
- [Python](./python/README.md)
- [Go](./go/README.md)
- [Rust](./rust/README.md) (coming soon)
- [Java](./java/README.md) (coming soon)

## Overview

Each language implementation provides:

1. A complete replacement for the official OpenAI client that handles Gonka network integration internally
2. Helper functions to modify the official OpenAI client for Gonka network compatibility
3. Detailed documentation and examples

## License

MIT 
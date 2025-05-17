module github.com/libermans/gonka-openai/go

go 1.21

toolchain go1.22.6

require (
	github.com/btcsuite/btcd/btcutil v1.1.5
	github.com/ethereum/go-ethereum v1.13.14 // for crypto utilities
	github.com/joho/godotenv v1.5.1 // Added for .env file loading
	github.com/openai/openai-go v0.1.0-beta.10
)

require golang.org/x/crypto v0.32.0

require (
	github.com/btcsuite/btcd/btcec/v2 v2.2.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/holiman/uint256 v1.2.4 // indirect
	github.com/tidwall/gjson v1.14.4 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	golang.org/x/sys v0.29.0 // indirect
)

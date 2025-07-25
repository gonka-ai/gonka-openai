package gonkaopenai

// Environment variable names
const (
	EnvPrivateKey = "GONKA_PRIVATE_KEY"
	EnvAddress    = "GONKA_ADDRESS"
	EnvEndpoints  = "GONKA_ENDPOINTS"
)

// Gonka chain ID used for address derivation
const GonkaChainID = "gonka-testnet-3"

// Default endpoints if none are provided
var DefaultEndpoints = []Endpoint{
	{URL: "https://api.gonka.testnet.example.com", Address: "transfer_address_1"},
	{URL: "https://api2.gonka.testnet.example.com", Address: "transfer_address_2"},
	{URL: "https://api3.gonka.testnet.example.com", Address: "transfer_address_3"},
}

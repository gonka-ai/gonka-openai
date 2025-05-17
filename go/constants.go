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
var DefaultEndpoints = []string{
	"https://api.gonka.testnet.example.com",
	"https://api2.gonka.testnet.example.com",
	"https://api3.gonka.testnet.example.com",
}

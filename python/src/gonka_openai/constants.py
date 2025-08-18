"""
Constants for the GonkaOpenAI library.
"""

import os

# Environment variable names
class ENV:
    PRIVATE_KEY = "GONKA_PRIVATE_KEY"
    ADDRESS = "GONKA_ADDRESS"
    ENDPOINTS = "GONKA_ENDPOINTS"
    SOURCE_URL = "GONKA_SOURCE_URL"

# Chain ID for Gonka network
GONKA_CHAIN_ID = "gonka-testnet-1"

# Default endpoints to use if none are provided
# Format: "url;address" - the part after the semicolon is the transfer address
DEFAULT_ENDPOINTS = [
    "https://api.gonka.testnet.example.com;gonka1default",
    "https://api2.gonka.testnet.example.com;gonka1default",
    "https://api3.gonka.testnet.example.com;gonka1default",
] 
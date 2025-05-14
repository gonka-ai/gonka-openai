"""
Constants for the GonkaOpenAI library.
"""

import os

# Environment variable names
class ENV:
    PRIVATE_KEY = "GONKA_PRIVATE_KEY"
    ADDRESS = "GONKA_ADDRESS"
    ENDPOINTS = "GONKA_ENDPOINTS"

# Chain ID for Gonka network
GONKA_CHAIN_ID = "gonka-testnet-1"

# Default endpoints to use if none are provided
DEFAULT_ENDPOINTS = [
    "https://api.gonka.testnet.example.com",
    "https://api2.gonka.testnet.example.com",
    "https://api3.gonka.testnet.example.com",
] 
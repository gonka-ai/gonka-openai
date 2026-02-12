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
GONKA_CHAIN_ID = "gonka-mainnet"

# Default endpoints to use if none are provided
# Format: "url;address" - the part after the semicolon is the transfer address
DEFAULT_ENDPOINTS = []
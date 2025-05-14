"""
Utility functions for the GonkaOpenAI library.
"""

import os
import json
import random
import hashlib
import base64
import logging
from typing import Any, Callable, Dict, List, Optional, Union, Tuple

# Import necessary libraries for OpenAI client
from openai import DefaultHttpxClient
import httpx
from httpx import Response

# Import cryptographic libraries
import secp256k1
import bech32
# Add ecdsa library import for signature compatibility
from ecdsa import SigningKey, SECP256k1, util

from .constants import ENV, DEFAULT_ENDPOINTS, GONKA_CHAIN_ID

# Configure logger
logger = logging.getLogger("gonka")
logging.basicConfig(level=logging.INFO)

def gonka_base_url(endpoints: Optional[List[str]] = None) -> str:
    """
    Get a random endpoint from the list of available endpoints.
    
    Args:
        endpoints: Optional list of endpoints to choose from
        
    Returns:
        A randomly selected endpoint
    """
    # Try to get endpoints from arguments, environment, or default to hardcoded values
    endpoint_list = endpoints or []
    
    if not endpoint_list:
        env_endpoints = os.environ.get(ENV.ENDPOINTS)
        if env_endpoints:
            endpoint_list = [e.strip() for e in env_endpoints.split(',')]
        else:
            endpoint_list = DEFAULT_ENDPOINTS
    
    # Select a random endpoint
    return random.choice(endpoint_list)


def custom_endpoint_selection(
    endpoint_selection_strategy: Callable[[List[str]], str],
    endpoints: Optional[List[str]] = None
) -> str:
    """
    Custom endpoint selection strategy.
    
    Args:
        endpoint_selection_strategy: A function that selects an endpoint from the list
        endpoints: Optional list of endpoints to choose from
        
    Returns:
        The selected endpoint
    """
    # Get the list of endpoints
    endpoint_list = endpoints or []
    
    if not endpoint_list:
        env_endpoints = os.environ.get(ENV.ENDPOINTS)
        if env_endpoints:
            endpoint_list = [e.strip() for e in env_endpoints.split(',')]
        else:
            endpoint_list = DEFAULT_ENDPOINTS
    
    # Use the provided strategy to select an endpoint
    return endpoint_selection_strategy(endpoint_list)


def gonka_signature(body: Any, private_key_hex: str) -> str:
    """
    Sign a request body with a private key using ECDSA (secp256k1).
    
    Args:
        body: The request body to sign
        private_key_hex: The private key in hex format (with or without 0x prefix)
        
    Returns:
        The signature as a base64 string
    """
    # Remove 0x prefix if present
    private_key_clean = private_key_hex[2:] if private_key_hex.startswith('0x') else private_key_hex
    
    # Create a signing key using ecdsa
    signing_key = SigningKey.from_string(bytes.fromhex(private_key_clean), curve=SECP256k1)
    
    # Use a custom encoder that handles low-S normalization automatically
    def encode_with_low_s(sig_r, sig_s, order):
        # Apply low-s value normalization for signature malleability
        if sig_s > order // 2:
            sig_s = order - sig_s
        # Pack r and s into a byte string
        r_bytes = sig_r.to_bytes(32, byteorder="big")
        s_bytes = sig_s.to_bytes(32, byteorder="big")
        return r_bytes + s_bytes
    
    # Convert body to bytes if it's not already
    if isinstance(body, dict):
        message_bytes = json.dumps(body).encode('utf-8')
    elif isinstance(body, str):
        message_bytes = body.encode('utf-8')
    elif isinstance(body, bytes):
        message_bytes = body
    else:
        raise TypeError(f"Unsupported body type: {type(body)}. Must be dict, str, or bytes.")
    
    # Sign the message with deterministic ECDSA using our custom encoder
    signature = signing_key.sign_deterministic(
        message_bytes,
        hashfunc=hashlib.sha256,
        sigencode=lambda r, s, order: encode_with_low_s(r, s, order)
    )
    
    # Return signature as base64
    return base64.b64encode(signature).decode('utf-8')


def gonka_address(private_key_hex: str) -> str:
    """
    Get the Cosmos address from a private key.
    
    Args:
        private_key_hex: The private key in hex format (with or without 0x prefix)
        
    Returns:
        The Cosmos address
    """
    # Remove 0x prefix if present
    private_key_clean = private_key_hex[2:] if private_key_hex.startswith('0x') else private_key_hex
    
    # Convert hex string to bytes
    private_key_bytes = bytes.fromhex(private_key_clean)
    
    # Create private key object
    privkey = secp256k1.PrivateKey(private_key_bytes)
    
    # Get the public key (33 bytes compressed format)
    pubkey = privkey.pubkey.serialize(compressed=True)
    
    # Create SHA256 hash of the public key
    sha = hashlib.sha256(pubkey).digest()
    
    # Take RIPEMD160 hash of the SHA256 hash
    ripemd = hashlib.new('ripemd160')
    ripemd.update(sha)
    address_bytes = ripemd.digest()
    
    # Convert to 5-bit words for bech32 encoding
    five_bit_words = bech32.convertbits(address_bytes, 8, 5)
    if five_bit_words is None:
        raise ValueError("Error converting address bytes to 5-bit words")
    
    # Get the prefix from the chain id (e.g., 'gonka' from 'gonka-testnet-1')
    prefix = GONKA_CHAIN_ID.split('-')[0]
    
    # Encode with bech32
    address = bech32.bech32_encode(prefix, five_bit_words)
    
    return address


def gonka_http_client(
    private_key: str,
    address: Optional[str] = None,
    http_client: Optional[httpx.Client] = None,
) -> httpx.Client:
    """
    Create a custom HTTP client for OpenAI that signs requests with your private key.
    
    Args:
        private_key: ECDSA private key for signing requests
        address: Optional Cosmos address to use instead of deriving from private key
        http_client: Optional base HTTP client
        
    Returns:
        A custom HTTP client compatible with the OpenAI client
    """
    
    # Use provided private key or fail
    if not private_key:
        raise ValueError("Private key is required")
    
    # Use the provided client or create a new DefaultHttpxClient
    client = http_client or DefaultHttpxClient()
    
    # Derive address if not provided
    resolved_address = address or gonka_address(private_key)
    
    # Wrap the send method to add headers
    original_send = client.send
    
    def wrapped_send(request, *args, **kwargs):
        request_id = random.randint(1000, 9999)
        logger.debug(f"[REQ-{request_id}] {request.method} {request.url}")
        
        # Initialize headers if not provided
        if request.headers is None:
            request.headers = {}
        
        # Add X-Requester-Address header
        request.headers['X-Requester-Address'] = resolved_address
        
        signature = gonka_signature(request.content, private_key)
        request.headers['Authorization'] = signature

        response = original_send(request, *args, **kwargs)
        return response
    
    # Replace the client's send method with the wrapped version
    client.send = wrapped_send
    
    return client 
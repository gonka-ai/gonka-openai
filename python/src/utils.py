"""
Utility functions for the GonkaOpenAI library.
"""

import os
import json
import random
import hashlib
import base64
import logging
import time
from typing import Any, Callable, Dict, List, Optional, Union, Tuple, NamedTuple
from dataclasses import dataclass

# Initialize base values for hybrid timestamp generation
_wall_base = time.time_ns()
_perf_base = time.perf_counter_ns()

def hybrid_timestamp_ns():
    """
    Generate a hybrid timestamp in nanoseconds that combines wall clock time with a performance counter.
    
    This approach ensures:
    1. Timestamps are unique and monotonically increasing (never go backwards)
    2. Timestamps remain accurate to standard time servers (assuming system wall time is accurate)
    
    Returns:
        A unique timestamp in nanoseconds
    """
    return _wall_base + (time.perf_counter_ns() - _perf_base)

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

@dataclass
class Endpoint:
    """
    Represents a Gonka endpoint with URL and transfer address.
    """
    url: str
    address: str
    
    @classmethod
    def parse(cls, endpoint_str: str) -> 'Endpoint':
        """
        Parse an endpoint string in the format 'url;address'.
        
        Args:
            endpoint_str: String in the format 'url;address'
            
        Returns:
            Endpoint object
            
        Raises:
            ValueError: If no transfer address is provided
        """
        parts = endpoint_str.split(';', 1)
        if len(parts) == 2:
            url = parts[0].strip()
            address = parts[1].strip()
            if not address:
                raise ValueError("Transfer address is required and cannot be empty")
            return cls(url=url, address=address)
        raise ValueError("Endpoint must be in the format 'url;address'")

# Configure logger
logger = logging.getLogger("gonka")
logging.basicConfig(level=logging.INFO)

def get_endpoints_from_env_or_default() -> List[Endpoint]:
    """
    Get a list of parsed endpoints from the environment variable or default values.

    Returns:
        List of Endpoint objects

    Raises:
        ValueError: If no valid endpoints with transfer addresses are available
    """
    env_endpoints = os.environ.get(ENV.ENDPOINTS)
    if env_endpoints:
        # Parse endpoints from environment variable
        return [Endpoint.parse(e.strip()) for e in env_endpoints.split(',')]
    else:
        # Use default endpoints which must include transfer addresses
        return [Endpoint.parse(endpoint) for endpoint in DEFAULT_ENDPOINTS]


def gonka_base_url(endpoints: Optional[List[Endpoint]] = None) -> Endpoint:
    """
    Get a random endpoint from the list of available endpoints.
    
    Args:
        endpoints: Optional list of endpoints to choose from
        
    Returns:
        A randomly selected Endpoint object
        
    Raises:
        ValueError: If no valid endpoints with transfer addresses are available
    """
    # Try to get endpoints from arguments, environment, or default to hardcoded values
    endpoint_list = endpoints or get_endpoints_from_env_or_default()
    # Select a random endpoint
    return random.choice(endpoint_list)


def resolve_endpoints(source_url: Optional[str] = None, endpoints: Optional[List[Endpoint]] = None) -> List[Endpoint]:
    """
    Resolve endpoints using SourceUrl first, then provided list, then env/defaults.

    Args:
        source_url: Optional SourceUrl for participants discovery
        endpoints: Optional explicit endpoints list

    Returns:
        List[Endpoint]
    """
    if source_url:
        try:
            # Local import to avoid circular dependency at module import time
            from .get_participants_with_proof import get_participants_with_proof  # type: ignore
            eps = get_participants_with_proof(source_url, "current")
            if eps:
                return eps
        except Exception:
            pass
    if endpoints:
        return endpoints
    return get_endpoints_from_env_or_default()


def resolve_and_select_endpoint(source_url: Optional[str] = None,
                                endpoints: Optional[List[Endpoint]] = None,
                                endpoint_selection_strategy: Optional[Callable[[List[Endpoint]], Endpoint]] = None) -> Tuple[List[Endpoint], Endpoint]:
    """
    Resolve endpoints then select one using optional strategy.
    """
    eps = resolve_endpoints(source_url=source_url, endpoints=endpoints)
    if endpoint_selection_strategy:
        return eps, endpoint_selection_strategy(eps)
    return eps, gonka_base_url(eps)


def custom_endpoint_selection(
    endpoint_selection_strategy: Callable[[List[Endpoint]], Endpoint],
    endpoints: Optional[List[Endpoint]] = None
) -> Endpoint:
    """
    Custom endpoint selection strategy.
    
    Args:
        endpoint_selection_strategy: A function that selects an endpoint from the list
        endpoints: Optional list of endpoints to choose from
        
    Returns:
        The selected Endpoint object
        
    Raises:
        ValueError: If no valid endpoints with transfer addresses are available
    """
    # Get the list of endpoints
    endpoint_list = endpoints or get_endpoints_from_env_or_default()

    return endpoint_selection_strategy(endpoint_list)


def gonka_signature(body: Any, private_key_hex: str, timestamp: int, transfer_address: str) -> str:
    """
    Sign a request body with a private key using ECDSA (secp256k1).
    
    Args:
        body: The request body to sign
        private_key_hex: The private key in hex format (with or without 0x prefix)
        timestamp: Timestamp in nanoseconds
        transfer_address: The transfer address to include in the signature
        
    Returns:
        The signature as a base64 string
        
    Raises:
        ValueError: If timestamp is not provided or transfer_address is empty
    """
    # Validate required parameters
    if not transfer_address:
        raise ValueError("Transfer address is required and cannot be empty")
    
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
        payload_bytes = json.dumps(body).encode('utf-8')
    elif isinstance(body, str):
        payload_bytes = body.encode('utf-8')
    elif isinstance(body, bytes):
        payload_bytes = body
    else:
        raise TypeError(f"Unsupported body type: {type(body)}. Must be dict, str, or bytes.")
    
    # Create message payload by concatenating components as described in the issue
    message_bytes = payload_bytes
    message_bytes += str(timestamp).encode('utf-8')
    message_bytes += transfer_address.encode('utf-8')

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
    transfer_address: Optional[str] = None,
) -> httpx.Client:
    """
    Create a custom HTTP client for OpenAI that signs requests with your private key.
    
    Args:
        private_key: ECDSA private key for signing requests
        address: Optional Cosmos address to use instead of deriving from private key
        http_client: Optional base HTTP client
        transfer_address: Optional transfer address for signing requests
        
    Returns:
        A custom HTTP client compatible with the OpenAI client
        
    Raises:
        ValueError: If private key is not provided or if no valid transfer address is available
    """
    
    # Use provided private key or fail
    if not private_key:
        raise ValueError("Private key is required")
    
    # Use the provided client or create a new DefaultHttpxClient
    client = http_client or DefaultHttpxClient()
    
    # Derive address if not provided
    resolved_address = address or gonka_address(private_key)
    
    # Get the endpoint with transfer address
    endpoint = gonka_base_url()
    
    # Use provided transfer address or get it from the endpoint
    resolved_transfer_address = transfer_address or endpoint.address
    
    # Validate that we have a transfer address
    if not resolved_transfer_address:
        raise ValueError("Transfer address is required and must be provided either directly or through an endpoint")
    
    # Wrap the send method to add headers
    original_send = client.send
    
    def wrapped_send(request, *args, **kwargs):
        request_id = random.randint(1000, 9999)
        logger.debug(f"[REQ-{request_id}] {request.method} {request.url}")
        
        # Initialize headers if not provided
        if request.headers is None:
            request.headers = {}
        
        # Generate unique and accurate timestamp in nanoseconds
        timestamp = hybrid_timestamp_ns()
        
        # Add X-Requester-Address header
        request.headers['X-Requester-Address'] = resolved_address
        
        # Add X-Timestamp header
        request.headers['X-Timestamp'] = str(timestamp)
        
        # Sign the request with timestamp and transfer address
        signature = gonka_signature(
            request.content, 
            private_key, 
            timestamp, 
            resolved_transfer_address
        )
        request.headers['Authorization'] = signature

        response = original_send(request, *args, **kwargs)
        return response
    
    # Replace the client's send method with the wrapped version
    client.send = wrapped_send
    
    return client
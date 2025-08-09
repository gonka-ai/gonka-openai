"""
GonkaOpenAI client that extends the official OpenAI client to work with Gonka network.
"""

import os
from typing import Dict, List, Optional, Any, Union

from openai import OpenAI

from .utils import gonka_base_url, gonka_address as utils_gonka_address, custom_endpoint_selection, gonka_http_client, \
    Endpoint
from .constants import ENV, GONKA_CHAIN_ID
from .utils import get_endpoints_from_env_or_default
from .get_participants_with_proof import get_participants_with_proof

class GonkaOpenAI(OpenAI):
    """
    GonkaOpenAI client that extends the official OpenAI client to work with Gonka network.
    """
    
    def __init__(
        self,
        *,
        gonka_private_key: Optional[str] = None,
        gonka_address: Optional[str] = None,
        endpoints: Optional[List[Endpoint]] = None,
        endpoint_selection_strategy: Optional[callable] = None,
        source_url: Optional[str] = None,
        **kwargs
    ):
        """
        Initialize the GonkaOpenAI client.
        
        Args:
            gonka_private_key: ECDSA private key for signing requests
            gonka_address: Optional Cosmos address to use instead of deriving from private key
            endpoints: Optional list of Gonka network endpoints to use
            endpoint_selection_strategy: Optional strategy for selecting from available endpoints
            **kwargs: Additional arguments to pass to the base OpenAI client
        """
        # Get private key from arguments or environment
        private_key = gonka_private_key or os.environ.get(ENV.PRIVATE_KEY)
        if not private_key:
            raise ValueError(
                f"Private key must be provided either as argument or through {ENV.PRIVATE_KEY} environment variable"
            )
        
        # Determine endpoints per priority:
        # 1) source_url arg or env -> try fetch participants with proof
        # 2) if none, use provided endpoints arg
        # 3) if none, parse endpoints from env (GONKA_ENDPOINTS)
        src_url = source_url or os.environ.get(ENV.SOURCE_URL)
        resolved_endpoints: List[Endpoint] = []
        if src_url:
            try:
                eps = get_participants_with_proof(src_url, "current")
                if eps:
                    resolved_endpoints = eps
            except Exception:
                pass
        if not resolved_endpoints and endpoints:
            resolved_endpoints = endpoints
        if not resolved_endpoints:
            env_eps = os.environ.get(ENV.ENDPOINTS)
            if env_eps:
                try:
                    resolved_endpoints = [Endpoint.parse(e.strip()) for e in env_eps.split(',') if e.strip()]
                except Exception:
                    resolved_endpoints = []

        # Determine the base URL
        if endpoint_selection_strategy:
            # Use custom endpoint selection strategy if provided
            base_endpoint = custom_endpoint_selection(endpoint_selection_strategy, resolved_endpoints)
        else:
            # Default to random endpoint selection
            base_endpoint = gonka_base_url(resolved_endpoints)
        
        # Save the private key for later use
        self._private_key = private_key
        # Reserved for future behavior parity with Go Options.SourceUrl
        self._source_url = source_url
        
        # Get or derive the Gonka address
        address_param = gonka_address
        self._gonka_address = address_param or os.environ.get(ENV.ADDRESS)
        
        # If no address is provided, derive it from the private key
        if not self._gonka_address:
            try:
                # Try to derive the address properly using the utility function
                self._gonka_address = utils_gonka_address(private_key)
            except Exception as e:
                # Fall back to a simplified address if derivation fails
                print(f"Warning: Error deriving address: {e}")
                self._gonka_address = f"{GONKA_CHAIN_ID.split('-')[0]}1{private_key[2:42].lower()}"
        
        # Create a custom HTTP client for request interception and signing
        http_client = gonka_http_client(
            private_key=private_key,
            address=self._gonka_address,
            http_client=kwargs.pop('http_client', None),
            transfer_address=base_endpoint.address
        )
        
        print(f"base_url: {base_endpoint.url}")

        # Set default mock-api-key if no api_key is provided
        if 'api_key' not in kwargs:
            kwargs['api_key'] = "mock-api-key"

        # Initialize the base OpenAI client with our custom HTTP client and base URL
        super().__init__(
            base_url=base_endpoint.url,
            http_client=http_client,
            **kwargs
        )
        
        print("Request signing is enabled through a custom HTTP client implementation.")
        print(f"Using Gonka address: {self._gonka_address}")
    
    @property
    def gonka_address(self) -> str:
        """Get the Gonka address."""
        return self._gonka_address
    
    @property
    def private_key(self) -> str:
        """Get the private key."""
        return self._private_key 
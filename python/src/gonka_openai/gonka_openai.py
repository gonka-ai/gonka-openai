"""
GonkaOpenAI client that extends the official OpenAI client to work with Gonka network.
"""

import os
from typing import Dict, List, Optional, Any, Union

from openai import OpenAI

from .utils import gonka_base_url, gonka_address as utils_gonka_address, custom_endpoint_selection, gonka_http_client, \
    Endpoint, resolve_endpoints, fetch_node_identity
from .constants import ENV, GONKA_CHAIN_ID

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
        ta_endpoint: Optional[Endpoint] = None,
        **kwargs
    ):
        """
        Initialize the GonkaOpenAI client.

        Args:
            gonka_private_key: ECDSA private key for signing requests
            gonka_address: Optional Cosmos address to use instead of deriving from private key
            endpoints: Optional list of Gonka network endpoints to use
            endpoint_selection_strategy: Optional strategy for selecting from available endpoints
            source_url: Optional source URL for participants discovery
            ta_endpoint: Optional manually specified Transfer Agent endpoint (skips dynamic TA selection)
            **kwargs: Additional arguments to pass to the base OpenAI client
        """
        # Get private key from arguments or environment
        private_key = gonka_private_key or os.environ.get(ENV.PRIVATE_KEY)
        if not private_key:
            raise ValueError(
                f"Private key must be provided either as argument or through {ENV.PRIVATE_KEY} environment variable"
            )

        # Use manually specified TA endpoint if provided, otherwise resolve dynamically
        if ta_endpoint is not None:
            base_endpoint = ta_endpoint
        else:
            # Resolve endpoints with filtering by allowed_transfer_addresses and delegate_ta preference
            src_url = source_url or os.environ.get(ENV.SOURCE_URL)
            resolved_endpoints = resolve_endpoints(
                source_url=src_url,
                endpoints=endpoints,
            )

            # Determine the base URL
            if endpoint_selection_strategy:
                base_endpoint = custom_endpoint_selection(endpoint_selection_strategy, resolved_endpoints)
            else:
                base_endpoint = gonka_base_url(resolved_endpoints)
            delegate_ta = fetch_node_identity(base_endpoint.url)
            if delegate_ta:
                original_address = base_endpoint.address
                if endpoint_selection_strategy:
                    base_endpoint = custom_endpoint_selection(endpoint_selection_strategy, delegate_ta)
                else:
                    base_endpoint = gonka_base_url(delegate_ta)
                base_endpoint.address = original_address
        # Save the private key for later use
        self._private_key = private_key
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

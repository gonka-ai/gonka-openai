#!/usr/bin/env python
"""
Test script for the GonkaOpenAI library.
"""

import os
import json
import unittest.mock
from dotenv import load_dotenv

from src.utils import get_endpoints_from_env_or_default

# Load environment variables from .env file
load_dotenv()

# Import httpx for creating a real client
import httpx

# Import the OpenAI client
from openai import OpenAI, DefaultHttpxClient

# Import the GonkaOpenAI client and utilities
from src import GonkaOpenAI, gonka_base_url, gonka_http_client

# Check for required environment variables
required_env_vars = ['GONKA_PRIVATE_KEY']
missing_env_vars = [var for var in required_env_vars if not os.environ.get(var)]

if missing_env_vars:
    print(f"Missing required environment variables: {', '.join(missing_env_vars)}")
    exit(1)

# If GONKA_ENDPOINTS is set, we'll use real endpoints and real requests
USE_REAL_REQUESTS = bool(os.environ.get('GONKA_ENDPOINTS'))

default_model = "Qwen/QwQ-32B"

def run_tests():
    """Run the tests for the GonkaOpenAI library."""
    try:
        # Determine URL based on whether GONKA_ENDPOINTS is set
        selected_url = None
        mock_base_url = None
        
        print("\n------ Test Environment ------")
        if USE_REAL_REQUESTS:
            # Use real endpoints from environment
            selected_url = gonka_base_url()
            print(f"Using real Gonka Base URL: {selected_url}")
            print("Using REAL HTTP requests - this will make actual API calls!")
            test_client = None  # Will use the default client
        else:
            # Use mock URL and mock requests
            mock_base_url = "https://mock-gonka-api.example.com"
            selected_url = mock_base_url  # For consistency
            original_base_url = unittest.mock.patch('src.utils.gonka_base_url', return_value=mock_base_url).start()
            print(f"Using mock Gonka Base URL: {mock_base_url}")
            print("Using MOCK HTTP requests - responses will be simulated")
            # Create a real httpx.Client instance
            test_client = httpx.Client()
            test_client.send = unittest.mock.MagicMock(side_effect=mock_send)
            
        
        # Example 1: Using the GonkaOpenAI wrapper (recommended)
        print("\n------ Example 1: Using GonkaOpenAI wrapper ------")
            
        gonka_client = GonkaOpenAI(
            api_key="mock-api-key",
            gonka_private_key=os.environ.get('GONKA_PRIVATE_KEY'),
            http_client=test_client,
            endpoints=get_endpoints_from_env_or_default()
        )
        
        print(f"Gonka Address: {gonka_client.gonka_address}")
        
        # Make a chat completion request
        print("\nSending first request...")
        chat_response = gonka_client.chat.completions.create(
            model=default_model,
            messages=[{"role": "user", "content": "Hello! Tell me a short joke."}],
        )
        
        print("\nResponse from first request:")
        print(chat_response.choices[0].message.content)
        
        # Example 2: Using the original OpenAI client with a custom HTTP client
        print("\n\n------ Example 2: Using original OpenAI client with custom HTTP client ------")
        
        # Create a custom HTTP client for the OpenAI client
        http_client = gonka_http_client(
            private_key=os.environ.get('GONKA_PRIVATE_KEY'),
            http_client=test_client,
            transfer_address=selected_url.address
        )
        
        # Create a standard OpenAI client with the custom HTTP client
        openai_client = OpenAI(
            api_key="mock-api-key",
            base_url=selected_url.url,
            http_client=http_client
        )
        
        # Make a request with the standard client
        print("\nSending request with standard client + custom HTTP client...")
        standard_response = openai_client.chat.completions.create(
            model=default_model,
            messages=[{"role": "user", "content": "What is the capital of France?"}],
        )
        
        print("\nResponse from standard client:")
        print(standard_response.choices[0].message.content)
        
        if USE_REAL_REQUESTS:
            print("\nNote: These were REAL API responses through the Gonka network")
        else:
            print("\nNote: These were MOCK responses (no actual API calls were made)")
    
    finally:
        # Remove any patches if we're using mocks
        if not USE_REAL_REQUESTS:
            unittest.mock.patch.stopall()

# Mock HTTP response for testing
mock_response_data = {
    "id": "mock-completion-id",
    "object": "chat.completion",
    "created": 1683720588,
    "model": default_model,
    "choices": [
        {
            "message": {
                "role": "assistant",
                "content": "This is a mock response from the API."
            },
            "index": 0,
            "finish_reason": "stop"
        }
    ]
}

# Mock send function for testing (replacing mock_request)
def mock_send(request, **kwargs):
    """Mock the send method to simulate responses and display request details."""
    # Extract request details
    method = request.method
    url = str(request.url)
    headers = dict(request.headers)
    
    # Get the actual data content from the request
    content = request.content
    
    # Log request details
    print("\n--- Request Details ---")
    print(f"Method: {method}")
    print(f"URL: {url}")
    print(f"Content-Type: {headers.get('Content-Type', 'Not specified')}")
    
    print("Headers:")
    for key, value in headers.items():
        if key.lower() == 'authorization':
            print(f"  {key}: {value[:20]}...")  # Truncate authorization header for security
        else:
            print(f"  {key}: {value}")
    
    # Log request body if present
    if content:
        print("Body format: content in request object")
        try:
            decoded = content.decode('utf-8')
            try:
                # Try to parse as JSON for prettier printing
                json_content = json.loads(decoded)
                if 'messages' in json_content:
                    # Truncate messages for brevity
                    print(f"Body: {json.dumps({**json_content, 'messages': '[truncated]'})}")
                else:
                    print(f"Body: {json.dumps(json_content)}")
            except json.JSONDecodeError:
                # Not JSON, print as plain text
                if len(decoded) > 1000:
                    print(f"Content (truncated): {decoded[:1000]}...")
                else:
                    print(f"Content: {decoded}")
        except:
            print(f"Content (binary, length): {len(content)} bytes")
            
    # Create response content as bytes
    response_content = json.dumps(mock_response_data).encode()
    
    # Create and return an httpx.Response
    response = httpx.Response(
        status_code=200,
        headers={
            'content-type': 'application/json',
            'x-request-id': 'mock-request-id',
            'openai-organization': 'mock-org',
            'openai-processing-ms': '42',
            'openai-version': '2023-05-15'
        },
        content=response_content,
        request=request,  # Include the original request
    )
    
    return response

if __name__ == "__main__":
    run_tests() 
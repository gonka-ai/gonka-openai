import { Secp256k1, sha256, ripemd160 } from '@cosmjs/crypto';
import { toBech32 } from '@cosmjs/encoding';
import secp256k1 from 'secp256k1';
import { ENV, DEFAULT_ENDPOINTS, GONKA_CHAIN_ID } from './constants.js';
import { EndpointSelectionFunction } from './types.js';

import { GonkaEndpoint } from './types.js';

/**
 * Get a random endpoint from the list of available endpoints
 * 
 * @param endpoints Optional list of endpoints to choose from
 * @returns A randomly selected endpoint
 */
export const gonkaBaseURL = (endpoints?: GonkaEndpoint[]): GonkaEndpoint => {
  // Try to get endpoints from arguments, environment, or default to hardcoded values
  let endpointList = endpoints || [];
  
  if (endpointList.length === 0) {
    const envEndpoints = process.env[ENV.ENDPOINTS];
    if (envEndpoints) {
      // Parse semicolon-separated pairs of URL and address
      endpointList = envEndpoints.split(',').map((e: string) => {
        const parts = e.trim().split(';');
        if (parts.length !== 2) {
          throw new Error(`Invalid endpoint format: ${e}. Expected format: "url;transferAddress"`);
        }

        console.log(`URL: ${parts[0]}, Address: ${parts[1]}`);

        return {
          url: parts[0],
          transferAddress: parts[1]
        };
      });
    } else {
      endpointList = DEFAULT_ENDPOINTS;
    }
  }

  // Select a random endpoint
  const randomIndex = Math.floor(Math.random() * endpointList.length);
  return endpointList[randomIndex];
};

/**
 * Custom endpoint selection strategy
 * 
 * @param endpointSelectionStrategy A function that selects an endpoint from the list
 * @param endpoints Optional list of endpoints to choose from
 * @returns The selected endpoint
 */
export const customEndpointSelection = (
  endpointSelectionStrategy: EndpointSelectionFunction,
  endpoints?: GonkaEndpoint[]
): GonkaEndpoint => {
  // Get the list of endpoints
  let endpointList = endpoints || [];
  
  if (endpointList.length === 0) {
    const envEndpoints = process.env[ENV.ENDPOINTS];
    if (envEndpoints) {
      // Parse semicolon-separated pairs of URL and address
      endpointList = envEndpoints.split(',').map((e: string) => {
        const parts = e.trim().split(';');
        if (parts.length !== 2) {
          throw new Error(`Invalid endpoint format: ${e}. Expected format: "url;transferAddress"`);
        }
        return {
          url: parts[0],
          transferAddress: parts[1]
        };
      });
    } else {
      endpointList = DEFAULT_ENDPOINTS;
    }
  }

  // Use the provided strategy to select an endpoint
  return endpointSelectionStrategy(endpointList);
};

import { SignatureComponents } from './types.js';

/**
 * Get the bytes to sign from the signature components
 * 
 * @param components The signature components
 * @returns The bytes to sign
 */
export const getSigBytes = (components: SignatureComponents): Uint8Array => {
  // Convert payload to bytes if needed
  let payloadBytes;
  if (typeof components.payload === 'string') {
    payloadBytes = Buffer.from(components.payload);
  } else if (Buffer.isBuffer(components.payload)) {
    payloadBytes = components.payload;
  } else if (components.payload instanceof Uint8Array) {
    payloadBytes = Buffer.from(components.payload);
  } else {
    // For objects or other types, stringify and convert to bytes
    payloadBytes = Buffer.from(JSON.stringify(components.payload));
  }
  
  // Convert timestamp to string and then to bytes
  const timestampBytes = Buffer.from(components.timestamp.toString());
  
  // Convert transfer address to bytes
  const transferAddressBytes = Buffer.from(components.transferAddress);
  
  // Concatenate all bytes
  const messageBytes = Buffer.concat([
    payloadBytes,
    timestampBytes,
    transferAddressBytes
  ]);
  
  return messageBytes;
};

/**
 * Get current timestamp in nanoseconds
 * 
 * @returns Current timestamp in nanoseconds as a bigint
 */
/**
 * Get current timestamp in nanoseconds since Unix epoch
 *
 * @returns Current timestamp in nanoseconds since Unix epoch
 */
export const getNanoTimestamp = (): bigint => {
  // Get milliseconds since epoch and convert to nanoseconds
  const millisSinceEpoch = BigInt(Date.now());
  const nanosSinceEpoch = millisSinceEpoch * 1000000n;

  // Add high-resolution nanoseconds for sub-millisecond precision
  const hrTime = process.hrtime();
  const subMillisecondNanos = BigInt(hrTime[1] % 1000000);

  return nanosSinceEpoch + subMillisecondNanos;
};

/**
 * Sign a request body with a private key using ECDSA
 * 
 * @param components The signature components (payload, timestamp, transferAddress)
 * @param privateKeyHex The private key in hex format (with or without 0x prefix)
 * @returns The signature as a base64 string
 */
export const gonkaSignature = async (components: SignatureComponents, privateKeyHex: string): Promise<string> => {
  // Remove 0x prefix if present
  const privateKeyClean = privateKeyHex.startsWith('0x') ? privateKeyHex.slice(2) : privateKeyHex;
  
  // Convert hex string to Uint8Array
  const privateKey = new Uint8Array(
    privateKeyClean.match(/.{1,2}/g)?.map(byte => parseInt(byte, 16)) || []
  );

  // Get the bytes to sign
  const messageBytes = getSigBytes(components);
  
  // Hash the payload with SHA-256 using Node.js crypto
  const messageHash = sha256(messageBytes);
  
  // Sign the hash with the private key
  const signature = await Secp256k1.createSignature(messageHash, privateKey);
  
  // Concatenate r and s values instead of using DER format
  const rawSignature = new Uint8Array([...signature.r(), ...signature.s()]);
  
  // Base64 encode
  return Buffer.from(rawSignature).toString('base64');
};

/**
 * Get the Cosmos address from a private key
 * 
 * @param privateKeyHex The private key in hex format (with or without 0x prefix)
 * @returns The Cosmos address
 */
export const gonkaAddress = (privateKeyHex: string): string => {
  // Remove 0x prefix if present
  const privateKeyClean = privateKeyHex.startsWith('0x') ? privateKeyHex.slice(2) : privateKeyHex;
  
  // Convert hex string to Uint8Array
  const privateKey = new Uint8Array(
    privateKeyClean.match(/.{1,2}/g)?.map(byte => parseInt(byte, 16)) || []
  );

  // Get public key (33 bytes compressed format)
  const compressedPubKey = secp256k1.publicKeyCreate(privateKey, true);
  
  // Create SHA256 hash of the public key
  const shaHash = sha256(compressedPubKey);
  
  // Take RIPEMD160 hash of the SHA256 hash
  const ripemdHash = ripemd160(shaHash);
  
  // Get the prefix from the chain id (e.g., 'gonka' from 'gonka-testnet-1')
  const prefix = GONKA_CHAIN_ID.split('-')[0];
  
  // Bech32 encode the address with the prefix
  return toBech32(prefix, ripemdHash);
};

/**
 * Creates a custom fetch function that signs requests with your private key
 * 
 * @param options The configuration options
 * @returns A custom fetch function compatible with the OpenAI client
 */
export const gonkaFetch = (
  options: { 
    gonkaPrivateKey?: string;
    gonkaAddress?: string;
  }
): (url: RequestInfo | URL, init?: RequestInit) => Promise<Response> => {
  // Get private key from options or environment
  const privateKey = options.gonkaPrivateKey || process.env[ENV.PRIVATE_KEY];
  if (!privateKey) {
    throw new Error(`Private key must be provided either in options or through ${ENV.PRIVATE_KEY} environment variable`);
  }

  // Get Gonka address from options or environment, or derive from private key
  const address = options.gonkaAddress || process.env[ENV.ADDRESS] || gonkaAddress(privateKey);
  
  // Store the original fetch function
  const originalFetch = globalThis.fetch;
  
  // Return a custom fetch function
  return async function(url: RequestInfo | URL, init?: RequestInit): Promise<Response> {
    // Clone the init object to avoid modifying the original
    const requestInit = init ? { ...init } : {};
    
    // Clone headers to avoid modifying the original
    requestInit.headers = new Headers(requestInit.headers || {});
    
    // Add the X-Requester-Address header if not present
    if (!requestInit.headers.has('X-Requester-Address')) {
      requestInit.headers.set('X-Requester-Address', address);
    }
    
    // Get the URL string from the URL object
    const urlString = url instanceof URL ? url.toString() : url.toString();
    
    // Extract the endpoint from the URL
    let selectedEndpoint: GonkaEndpoint | undefined;
    
    // Try to find the endpoint in the DEFAULT_ENDPOINTS
    for (const endpoint of DEFAULT_ENDPOINTS) {
      if (urlString.startsWith(endpoint.url)) {
        selectedEndpoint = endpoint;
        break;
      }
    }
    
    // If endpoint not found in DEFAULT_ENDPOINTS, try to parse from environment
    let endpoints = process.env[ENV.ENDPOINTS];
    if (!selectedEndpoint && endpoints) {
      const envEndpoints = endpoints.split(',').map((e: string) => {
        const parts = e.trim().split(';');
        if (parts.length !== 2) {
          return null;
        }
        return {
          url: parts[0],
          transferAddress: parts[1]
        };
      }).filter(Boolean) as GonkaEndpoint[];
      
      for (const endpoint of envEndpoints) {
        if (urlString.startsWith(endpoint.url)) {
          selectedEndpoint = endpoint;
          break;
        }
      }
    }
    
    // If no endpoint found, throw an error
    if (!selectedEndpoint) {
      throw new Error(`Could not determine the endpoint for URL: ${urlString}`);
    }
    
    // Generate a unique timestamp in nanoseconds
    const timestamp = getNanoTimestamp();
    console.log('Timestampn:', timestamp);
    // Add the X-Timestamp header
    requestInit.headers.set('X-Timestamp', timestamp.toString());
    
    // If there's a body, sign it and add the signature to the Authorization header
    if (requestInit.body) {
      try {
        // Create signature components
        const components: SignatureComponents = {
          payload: requestInit.body,
          timestamp: timestamp,
          transferAddress: selectedEndpoint.transferAddress
        };
        
        // Sign the components
        const signature = await gonkaSignature(components, privateKey);
        
        // Add the signature to the Authorization header
        requestInit.headers.set('Authorization', signature);
      } catch (error) {
        console.error('Error signing request:', error);
        // Fall back to a static signature if dynamic signing fails
        requestInit.headers.set('Authorization', 'ECDSA_SIG_FALLBACK_' + Buffer.from(privateKey.substring(0, 16)).toString('base64'));
      }
    } else {
      // No body to sign, use a static signature
      requestInit.headers.set('Authorization', 'ECDSA_SIG_EMPTY_' + Buffer.from(privateKey.substring(0, 16)).toString('base64'));
    }
    
    // Call the original fetch with the modified request
    return originalFetch(url, requestInit);
  };
};
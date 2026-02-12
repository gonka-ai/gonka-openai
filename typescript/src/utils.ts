import { Secp256k1, sha256, ripemd160 } from '@cosmjs/crypto';
import { toBech32 } from '@cosmjs/encoding';
import secp256k1 from 'secp256k1';
import { ENV, DEFAULT_ENDPOINTS, GONKA_CHAIN_ID } from './constants.js';
import fetch from 'node-fetch';
import { verifyIcs23 } from './ics23.js';
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

const ensureV1 = (u: string): string => {
  if (!u) return u;
  const noTrail = u.endsWith('/') ? u.slice(0, -1) : u;
  return noTrail.endsWith('/v1') ? noTrail : `${noTrail}/v1`;
};

/**
 * Fetch allowed transfer addresses via node's /chain-api/ proxy
 */
export const fetchAllowedTransferAddresses = async (nodeUrl: string): Promise<Set<string>> => {
  let base = nodeUrl.replace(/\/+$/, '');
  if (base.endsWith('/v1')) base = base.slice(0, -3);
  const url = `${base}/chain-api/productscience/inference/inference/params`;
  try {
    const res = await fetch(url, { method: 'GET' });
    if (!res.ok) throw new Error(`status ${res.status}`);
    const data = await res.json() as any;
    return new Set(data?.params?.transfer_agent_access_params?.allowed_transfer_addresses ?? []);
  } catch {
    return new Set();
  }
};

/**
 * Fetch node identity including delegate_ta, returns endpoints
 */
export const fetchNodeIdentity = async (nodeUrl: string): Promise<GonkaEndpoint[]> => {
  let base = nodeUrl.replace(/\/+$/, '');
  if (base.endsWith('/v1')) base = base.slice(0, -3);
  const url = `${base}/v1/identity`;
  try {
    const res = await fetch(url, { method: 'GET' });
    if (!res.ok) throw new Error(`status ${res.status}`);
    const data = await res.json() as any;
    const delegateTa: Record<string, string> = data?.data?.delegate_ta ?? {};
    return Object.entries(delegateTa).map(([url, addr]) => ({
      url: ensureV1(url),
      transferAddress: addr,
      address: addr,
    }));
  } catch {
    return [];
  }
};

/**
 * Fetch participants with proof and return endpoints (no ICS23 verification here)
 */
export const getParticipantsWithProof = async (sourceUrl: string, epoch: string): Promise<GonkaEndpoint[]> => {
  const base = sourceUrl.endsWith('/') ? sourceUrl.slice(0, -1) : sourceUrl;
  const res = await fetch(`${base}/v1/epochs/${epoch}/participants`, {
    method: 'GET',
    headers: { 'Content-Type': 'application/json' }
  });
  if (!res.ok) throw new Error(`failed to fetch participants: ${res.status}`);
  const payload = await res.json();
  return getParticipantsWithProofFromPayload(payload, process.env['GONKA_VERIFY_PROOF'] === '1');
};

/**
 * Resolve endpoints via SourceUrl, provided list, or environment/defaults.
 * When endpoints are explicitly provided or from env, use them directly without filtering.
 * Only filters by allowed_transfer_addresses and fetches delegate_ta when using sourceUrl.
 */
export const resolveEndpoints = async (opts: {
  sourceUrl?: string;
  endpoints?: GonkaEndpoint[];
}): Promise<GonkaEndpoint[]> => {
  // If endpoints provided directly, use them without filtering
  if (opts.endpoints && opts.endpoints.length) {
    return opts.endpoints;
  }

  // If env endpoints exist, use them without filtering
  const envEndpoints = process.env[ENV.ENDPOINTS];
  if (envEndpoints) {
    return envEndpoints.split(',').map((e: string) => {
      const parts = e.trim().split(';');
      if (parts.length !== 2) {
        throw new Error(`Invalid endpoint format: ${e}. Expected format: "url;transferAddress"`);
      }
      return { url: parts[0], transferAddress: parts[1] } as GonkaEndpoint;
    });
  }

  // Only fetch from sourceUrl and filter if no explicit endpoints provided
  if (opts.sourceUrl) {
    try {
      let eps = await getParticipantsWithProof(opts.sourceUrl, 'current');
      const allowedTransferAddresses = await fetchAllowedTransferAddresses(opts.sourceUrl);
      eps = eps.filter(e => allowedTransferAddresses.has(e.transferAddress) || allowedTransferAddresses.has(e.address || ''));

      // Check for delegate_ta from a resolved endpoint
      if (eps.length > 0) {
        const probe = gonkaBaseURL(eps);
        const delegateTa = await fetchNodeIdentity(probe.url);
        if (delegateTa.length > 0) {
          const originalAddress = probe.address || probe.transferAddress;
          const selected = gonkaBaseURL(delegateTa);
          selected.address = originalAddress;
          selected.transferAddress = originalAddress;
          return [selected];
        }
      }
      return eps;
    } catch (e) {
      console.error(`Error fetching endpoints from sourceUrl ${opts.sourceUrl}:`, e);
    }
  }

  // Fallback to defaults
  return DEFAULT_ENDPOINTS;
};

/**
 * Resolve endpoints then select one using an optional strategy.
 */
export const resolveAndSelectEndpoint = async (opts: {
  sourceUrl?: string;
  endpoints?: GonkaEndpoint[];
  endpointSelectionStrategy?: EndpointSelectionFunction;
  verifyProof?: boolean;
}): Promise<{ endpoints: GonkaEndpoint[]; selected: GonkaEndpoint }> => {
  const endpoints = await resolveEndpoints(opts);
  const selected = opts.endpointSelectionStrategy
    ? customEndpointSelection(opts.endpointSelectionStrategy, endpoints)
    : gonkaBaseURL(endpoints);
  return { endpoints, selected };
};

const fromHex = (hex: string): Uint8Array => {
  const clean = hex.trim().toLowerCase();
  if (!/^[0-9a-f]+$/.test(clean) || clean.length % 2 !== 0) throw new Error('invalid hex');
  const out = new Uint8Array(clean.length / 2);
  for (let i = 0; i < out.length; i++) out[i] = parseInt(clean.substr(i * 2, 2), 16);
  return out;
};

const fromB64 = (b64: string): Uint8Array => new Uint8Array(Buffer.from(b64, 'base64'));

/**
 * Process a participants-with-proof payload and (optionally) verify ICS23 against app hash.
 */
export const getParticipantsWithProofFromPayload = (payload: any, verify: boolean): GonkaEndpoint[] => {
  if (verify) {
    const activeBytesHex = payload?.active_participants_bytes;
    if (!activeBytesHex) throw new Error('missing active_participants_bytes');
    const value = fromHex(activeBytesHex);

    // collect proof ops
    const opsArr = payload?.proof_ops?.ops ?? [];
    if (!Array.isArray(opsArr) || opsArr.length !== 2) throw new Error('expected 2 proof ops');
    const proofOps = opsArr.map((op: any) => ({
      type: op?.type,
      key: fromB64(op?.key ?? ''),
      data: fromB64(op?.data ?? ''),
    }));

    // parse app hash (hex)
    const appHashHex = payload?.block?.app_hash || payload?.block?.header?.app_hash;
    if (!appHashHex) throw new Error('missing app_hash');
    const appHash = fromHex(appHashHex);

    verifyIcs23(appHash, proofOps, value);
  }

  const excludedParticipants = new Set(
    (payload?.excluded_participants ?? []).map((p: any) => p.address)
  );
  const participants = payload?.active_participants?.participants ?? [];
  const endpoints: GonkaEndpoint[] = [];
  for (const p of participants) {
    if (p?.inference_url && p?.index && !excludedParticipants.has(p.index)) {
      const url = ensureV1(p.inference_url);
      const addr = p.index;
      endpoints.push({ url, transferAddress: addr, address: addr });
    }
  }
  return endpoints;
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
  let payloadBytes: Buffer;
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

  // Phase 3: Sign hash of payload instead of raw payload
  const payloadHash = sha256(payloadBytes);
  const payloadHashHex = Buffer.from(payloadHash).toString('hex');

  // Build signature input: hash + timestamp + transfer_address
  const signatureInput = payloadHashHex + components.timestamp.toString() + components.transferAddress;

  return Buffer.from(signatureInput);
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
// TypeScript
export const gonkaSignature = async (components: SignatureComponents, privateKeyHex: string): Promise<string> => {
  // Remove 0x prefix if present
  const privateKeyClean = privateKeyHex.startsWith('0x') ? privateKeyHex.slice(2) : privateKeyHex;

  // Convert hex string to Uint8Array
  const privateKey = new Uint8Array(
    privateKeyClean.match(/.{1,2}/g)?.map((byte) => parseInt(byte, 16)) || []
  );

  // Get the bytes to sign
  const messageBytes = getSigBytes(components);

  // Hash with SHA-256
  const messageHash = sha256(messageBytes);

  // Create signature (r,s) using cosmjs
  const signature = await Secp256k1.createSignature(messageHash, privateKey);

  // If we DON'T pad the signature here, it will be shorter whenever the first byte of r or s is 0 (about once ever 128
    // tries, and will fail to validate
  // Helper: left pad to 32 bytes
  const pad32 = (u: Uint8Array): Uint8Array => {
    if (u.length === 32) return u;
    const out = new Uint8Array(32);
    out.set(u, 32 - u.length);
    return out;
  };

  // Extract r,s and convert to BigInt for low-S normalization
  let r = signature.r();
  let s = signature.s();

  // secp256k1 curve order (n)
  const n = 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141n;
  const halfN = n >> 1n;

  const bi = (u: Uint8Array): bigint => {
    let v = 0n;
    for (const b of u) v = (v << 8n) + BigInt(b);
    return v;
  };

  const toBytes32 = (x: bigint): Uint8Array => {
    const out = new Uint8Array(32);
    for (let i = 31; i >= 0; i--) {
      out[i] = Number(x & 0xffn);
      x >>= 8n;
    }
    return out;
  };

  // Low-S normalization to match other impls
  const sBig = bi(s);
  const sNorm = sBig > halfN ? n - sBig : sBig;

  // Ensure fixed 32-byte r and s
  const r32 = pad32(r);
  const s32 = toBytes32(sNorm);

  const rawSignature = new Uint8Array(64);
  rawSignature.set(r32, 0);
  rawSignature.set(s32, 32);

  // Base64 encode (standard, with padding)
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
    selectedEndpoint?: GonkaEndpoint; // provider address for signing
  }
): (url: RequestInfo | URL, init?: RequestInit) => Promise<Response> => {
  // Get private key from options or environment
  const privateKey = options.gonkaPrivateKey || process.env[ENV.PRIVATE_KEY];
  if (!privateKey) {
    throw new Error(`Private key must be provided either in options or through ${ENV.PRIVATE_KEY} environment variable`);
  }

  // Get Gonka address from options or environment, or derive from private key
  const address = options.gonkaAddress || process.env[ENV.ADDRESS] || gonkaAddress(privateKey);
  
  // Store the original fetch function and fixed endpoint (if provided)
  const originalFetch = globalThis.fetch;
  const fixedSelectedEndpoint: GonkaEndpoint | undefined = options.selectedEndpoint;
  
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
    
    // Determine the endpoint for signing
    let selectedEndpoint: GonkaEndpoint | undefined = fixedSelectedEndpoint;
    if (!selectedEndpoint) {
      // Optional fallback: try to match from env-provided endpoints
      const endpointsEnv = process.env[ENV.ENDPOINTS];
      if (endpointsEnv) {
        const envEndpoints = endpointsEnv.split(',').map((e: string) => {
          const parts = e.trim().split(';');
          if (parts.length !== 2) return null;
          return { url: parts[0], transferAddress: parts[1] } as GonkaEndpoint;
        }).filter(Boolean) as GonkaEndpoint[];
        for (const endpoint of envEndpoints) {
          if (urlString.startsWith(endpoint.url)) {
            selectedEndpoint = endpoint;
            break;
          }
        }
      }
      if (!selectedEndpoint) {
        throw new Error(`Could not determine provider address for URL: ${urlString}. Pass selectedEndpoint to gonkaFetch or set GONKA_ENDPOINTS.`);
      }
    }
    
    // Generate a unique timestamp in nanoseconds
    const timestamp = getNanoTimestamp();
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
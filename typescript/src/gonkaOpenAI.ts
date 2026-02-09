import OpenAI from 'openai';
import { GonkaOpenAIOptions, SignatureComponents, GonkaEndpoint } from './types.js';
import { customEndpointSelection, gonkaBaseURL, gonkaFetch, getNanoTimestamp, resolveEndpoints } from './utils.js';
import { gonkaSignature as signatureFunction } from './utils.js';
import { ENV } from './constants.js';

/**
 * GonkaOpenAI client that extends the official OpenAI client to work with Gonka network
 */
export class GonkaOpenAI extends OpenAI {
  private readonly _privateKey: string;
  
  // Store the selected endpoint
  private readonly _selectedEndpoint: GonkaEndpoint;
  private _sourceUrl?: string;
  private _endpoints?: GonkaEndpoint[];
  
  /**
   * Create a new GonkaOpenAI client
   * 
   * @param options Options for the client
   */
  constructor(options: GonkaOpenAIOptions) {
    // Get private key from options or environment
    const privateKey = options.gonkaPrivateKey || process.env[ENV.PRIVATE_KEY];
    if (!privateKey) {
      throw new Error(`Private key must be provided either in options or through ${ENV.PRIVATE_KEY} environment variable`);
    }

    // Prepare configuration (avoid using `this` before super())
    const sourceUrlLocal = options.sourceUrl || process.env['GONKA_SOURCE_URL'];
    const endpointsLocal: GonkaEndpoint[] | undefined =
      options.endpoints && options.endpoints.length ? options.endpoints : undefined;

    // Determine candidate endpoints for selection (only use provided; env/defaults handled by gonkaBaseURL)
    const resolvedEndpoints: GonkaEndpoint[] = endpointsLocal || [];

    // Determine the endpoint
    let selectedEndpoint: GonkaEndpoint;
    if (options.endpointSelectionStrategy) {
      // Use custom endpoint selection strategy if provided
      selectedEndpoint = customEndpointSelection(options.endpointSelectionStrategy, resolvedEndpoints);
    } else {
      // Default to random endpoint selection
      selectedEndpoint = gonkaBaseURL(resolvedEndpoints);
    }

    // Create the signing fetch function
    const signingFetch = gonkaFetch({
      gonkaPrivateKey: privateKey,
      gonkaAddress: options.gonkaAddress || process.env[ENV.ADDRESS],
      selectedEndpoint: selectedEndpoint,
    });

    // Normalize endpoint alias for consistency
    if (!selectedEndpoint.address) selectedEndpoint.address = selectedEndpoint.transferAddress;

    // Create the OpenAI configuration object
    const openAIConfig = {
      ...options,
      baseURL: selectedEndpoint.url,
    };
    
    // Set default mock-api-key if no apiKey is provided
    if (!openAIConfig.apiKey) {
      openAIConfig.apiKey = "mock-api-key";
    }
    
    // Add the signing fetch function to the configuration
    (openAIConfig as any).fetch = signingFetch;

    // Call OpenAI constructor with the configuration
    super(openAIConfig);
    
    // Save the private key and selected endpoint for signing requests
    this._privateKey = privateKey;
    this._selectedEndpoint = selectedEndpoint;
    this._sourceUrl = sourceUrlLocal;
    this._endpoints = endpointsLocal;
  }
  
  /**
   * Get the private key
   */
  get privateKey(): string {
    return this._privateKey;
  }

  /**
   * Sign a request body with the client's private key
   * 
   * @param body The request body to sign
   * @param transferAddress The Cosmos address of the endpoint provider (optional, uses the selected endpoint's address if not provided)
   * @returns The signature as a base64 string
   */
  async signRequest(body: any, transferAddress?: string): Promise<string> {
    // Generate a unique timestamp in nanoseconds
    const timestamp = getNanoTimestamp();
    
    // Use the provided transfer address or the selected endpoint's address
    const address = transferAddress || this._selectedEndpoint.address || this._selectedEndpoint.transferAddress;
    
    // Create signature components
    const components: SignatureComponents = {
      payload: body,
      timestamp: timestamp,
      transferAddress: address
    };
    
    return await signatureFunction(components, this._privateKey);
  }
  
  /**
   * Get the selected endpoint
   */
  get selectedEndpoint() {
    return this._selectedEndpoint;
  }

  /**
   * Async factory method that resolves endpoints with filtering by
   * allowed_transfer_addresses and delegate_ta preference before constructing the client.
   */
  static async create(options: GonkaOpenAIOptions): Promise<GonkaOpenAI> {
    const resolved = await resolveEndpoints({
      sourceUrl: options.sourceUrl,
      endpoints: options.endpoints,
    });
    return new GonkaOpenAI({
      ...options,
      endpoints: resolved,
      // Clear sourceUrl so the constructor doesn't try to re-resolve
      sourceUrl: undefined,
    });
  }
}

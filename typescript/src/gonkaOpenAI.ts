import OpenAI from 'openai';
import { GonkaOpenAIOptions, SignatureComponents } from './types.js';
import { customEndpointSelection, gonkaBaseURL, gonkaFetch, getNanoTimestamp } from './utils.js';
import { gonkaSignature as signatureFunction } from './utils.js';
import { ENV } from './constants.js';

/**
 * GonkaOpenAI client that extends the official OpenAI client to work with Gonka network
 */
export class GonkaOpenAI extends OpenAI {
  private readonly _privateKey: string;
  
  // Store the selected endpoint
  private readonly _selectedEndpoint;
  
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

    // Determine the endpoint
    let selectedEndpoint;
    if (options.endpointSelectionStrategy) {
      // Use custom endpoint selection strategy if provided
      selectedEndpoint = customEndpointSelection(options.endpointSelectionStrategy, options.endpoints);
    } else {
      // Default to random endpoint selection
      selectedEndpoint = gonkaBaseURL(options.endpoints);
    }

    // Create the signing fetch function
    const signingFetch = gonkaFetch({
      gonkaPrivateKey: privateKey,
      gonkaAddress: options.gonkaAddress || process.env[ENV.ADDRESS]
    });

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
    const address = transferAddress || this._selectedEndpoint.transferAddress;
    
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
} 
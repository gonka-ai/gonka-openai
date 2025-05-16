import OpenAI from 'openai';
import { GonkaOpenAIOptions } from './types.js';
import { customEndpointSelection, gonkaBaseURL, gonkaFetch } from './utils.js';
import { gonkaSignature as signatureFunction } from './utils.js';
import { ENV } from './constants.js';

/**
 * GonkaOpenAI client that extends the official OpenAI client to work with Gonka network
 */
export class GonkaOpenAI extends OpenAI {
  private readonly _privateKey: string;
  
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

    // Determine the base URL
    let baseURL: string;
    if (options.endpointSelectionStrategy) {
      // Use custom endpoint selection strategy if provided
      baseURL = customEndpointSelection(options.endpointSelectionStrategy, options.endpoints);
    } else {
      // Default to random endpoint selection
      baseURL = gonkaBaseURL(options.endpoints);
    }

    // Create the signing fetch function directly (now that it's synchronous)
    const signingFetch = gonkaFetch({
      gonkaPrivateKey: privateKey,
      gonkaAddress: options.gonkaAddress || process.env[ENV.ADDRESS]
    });

    // Create the OpenAI configuration object
    const openAIConfig = {
      ...options,
      baseURL,
    };
    
    // Set default mock-api-key if no apiKey is provided
    if (!openAIConfig.apiKey) {
      openAIConfig.apiKey = "mock-api-key";
    }
    
    // Add the signing fetch function to the configuration
    (openAIConfig as any).fetch = signingFetch;

    // Call OpenAI constructor with the configuration
    super(openAIConfig);
    
    // Save the private key for signing requests
    this._privateKey = privateKey;
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
   * @returns The signature as a base64 string
   */
  async signRequest(body: any): Promise<string> {
    return await signatureFunction(body, this._privateKey);
  }
} 
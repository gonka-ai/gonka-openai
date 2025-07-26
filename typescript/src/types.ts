import { ClientOptions } from 'openai';

/**
 * Endpoint object with URL and TransferAddress
 */
export interface GonkaEndpoint {
  /**
   * URL of the endpoint
   */
  url: string;
  
  /**
   * Cosmos address of the endpoint provider
   */
  transferAddress: string;
}

/**
 * Function type for custom endpoint selection
 */
export type EndpointSelectionFunction = (endpoints: GonkaEndpoint[]) => GonkaEndpoint;

/**
 * Options for the GonkaOpenAI client
 */
export interface GonkaOpenAIOptions extends Omit<ClientOptions, 'baseURL' | 'defaultHeaders'> {
  /**
   * ECDSA private key for signing requests
   * If not provided, will be read from GONKA_PRIVATE_KEY environment variable
   */
  gonkaPrivateKey?: string;

  /**
   * Cosmos address to use as requester
   * If not provided, will be derived from privateKey with chain_id "gonka-testnet-1"
   * Or read from GONKA_ADDRESS environment variable
   */
  gonkaAddress?: string;

  /**
   * List of Gonka network endpoints to use
   * One will be selected randomly for each client instance
   * If not provided, will use DEFAULT_ENDPOINTS or GONKA_ENDPOINTS environment variable
   * Each endpoint must include both a URL and a TransferAddress (Cosmos address of the provider)
   */
  endpoints?: GonkaEndpoint[];

  /**
   * Custom function for signing request bodies
   */
  signFunction?: (body: any, privateKey: string) => string | Promise<string>;

  /**
   * Strategy for selecting from available endpoints
   */
  endpointSelectionStrategy?: EndpointSelectionFunction;
}

/**
 * Custom fetch signature that OpenAI client will accept
 */
export type OpenAIFetch = (url: string | URL | Request, init?: RequestInit) => Promise<Response>;

/**
 * Components required for signature generation
 */
export interface SignatureComponents {
  /**
   * The payload to sign
   */
  payload: any;
  
  /**
   * Timestamp in nanoseconds
   */
  timestamp: bigint;
  
  /**
   * Cosmos address of the endpoint provider
   */
  transferAddress: string;
}

/**
 * Function to sign request body with private key
 */
export type SignatureFunction = (components: SignatureComponents, privateKey: string) => string | Promise<string>; 
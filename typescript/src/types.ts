import { ClientOptions } from 'openai';

/**
 * Function type for custom endpoint selection
 */
export type EndpointSelectionFunction = (endpoints: string[]) => string;

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
   */
  endpoints?: string[];

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
 * Function to sign request body with private key
 */
export type SignatureFunction = (body: any, privateKey: string) => string | Promise<string>; 
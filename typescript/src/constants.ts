import { GonkaEndpoint } from './types.js';

/**
 * Default Gonka network endpoints
 * These will be used if no endpoints are provided in the options or environment
 * Each endpoint includes both a URL and a TransferAddress (Cosmos address of the provider)
 */
export const DEFAULT_ENDPOINTS: GonkaEndpoint[] = [
  {
    url: 'https://api.gonka.testnet.example.com',
    transferAddress: 'gonka1example1address1111111111111111111111'
  },
  {
    url: 'https://api2.gonka.testnet.example.com',
    transferAddress: 'gonka1example2address2222222222222222222222'
  },
  {
    url: 'https://api3.gonka.testnet.example.com',
    transferAddress: 'gonka1example3address3333333333333333333333'
  },
];

/**
 * Chain ID for Gonka testnet
 * Used for deriving the Cosmos address from the private key
 */
export const GONKA_CHAIN_ID = 'gonka-testnet-1';

/**
 * Environment variable names
 */
export const ENV = {
  PRIVATE_KEY: 'GONKA_PRIVATE_KEY',
  ADDRESS: 'GONKA_ADDRESS',
  ENDPOINTS: 'GONKA_ENDPOINTS',
}; 
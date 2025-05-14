/**
 * Default Gonka network endpoints
 * These will be used if no endpoints are provided in the options or environment
 */
export const DEFAULT_ENDPOINTS = [
  'https://api.gonka.testnet.example.com',
  'https://api2.gonka.testnet.example.com',
  'https://api3.gonka.testnet.example.com',
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
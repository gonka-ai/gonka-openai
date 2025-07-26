import OpenAI from 'openai';
import { GonkaOpenAI, gonkaBaseURL, gonkaFetch } from './dist/index.js';
import * as dotenv from 'dotenv';

dotenv.config();
let defaultModel = 'unsloth/llama-3-8b-Instruct';

// Check for required environment variables
const requiredEnvVars = ['GONKA_PRIVATE_KEY'];
const missingEnvVars = requiredEnvVars.filter(envVar => !process.env[envVar]);

if (missingEnvVars.length > 0) {
  console.error(`Missing required environment variables: ${missingEnvVars.join(', ')}`);
  process.exit(1);
}

// Determine if we should use real requests or mock them
const USE_REAL_REQUESTS = Boolean(process.env.GONKA_ENDPOINTS);

// Store original fetch for later restoration if we're mocking
const originalFetch = global.fetch;

// Setup mock fetch only if we're not using real requests
if (!USE_REAL_REQUESTS) {
  // Mock fetch function for testing
  global.fetch = async (url, init) => {
    console.log('\n--- Request Details ---');
    console.log(`URL: ${url}`);
    console.log('Headers:');
    
    // Convert headers object to a normal object for logging
    const headers = {};
    init.headers.forEach((value, key) => {
      headers[key] = key === 'Authorization' 
        ? `${value.substring(0, 20)}...` // Truncate authorization header for security
        : value;
    });
    
    console.log(headers);
    
    // Log request body if present
    if (init.body) {
      try {
        // Try to parse body if it's JSON
        const body = JSON.parse(init.body);
        console.log('Body:', body.messages ? { ...body, messages: '[truncated]' } : body);
      } catch (e) {
        // If not JSON, just log the type
        console.log('Body: [Non-JSON body]', typeof init.body);
      }
    }
    
    // Create response headers
    const responseHeaders = new Map([
      ['content-type', 'application/json'],
      ['x-request-id', 'mock-request-id'],
      ['openai-organization', 'mock-org'],
      ['openai-processing-ms', '42'],
      ['openai-version', '2023-05-15']
    ]);
    
    // Mock successful API response
    return {
      ok: true,
      status: 200,
      headers: responseHeaders,
      json: async () => ({
        id: 'mock-completion-id',
        object: 'chat.completion',
        created: Date.now(),
        model: defaultModel,
        choices: [
          {
            message: {
              role: 'assistant',
              content: 'This is a mock response from the API.'
            },
            index: 0,
            finish_reason: 'stop'
          }
        ]
      })
    };
  };
}

const run = async () => {
  try {
    console.log('\n------ Test Environment ------');
    const selectedEndpoint = gonkaBaseURL();
    console.log('Using Gonka Endpoint:', {
      url: selectedEndpoint.url,
      transferAddress: selectedEndpoint.transferAddress
    });
    
    if (USE_REAL_REQUESTS) {
      console.log("Using REAL HTTP requests - this will make actual API calls!");
    } else {
      console.log("Using MOCK HTTP requests - responses will be simulated");
    }
    
    // Example 1: Using the GonkaOpenAI wrapper (recommended)
    console.log('\n------ Example 1: Using GonkaOpenAI wrapper ------');
    const gonkaClient = new GonkaOpenAI({
      gonkaPrivateKey: process.env.GONKA_PRIVATE_KEY,
      apiKey: 'mock-api-key', // Required by OpenAI client
    });
    
    // Make a chat completion request
    console.log('\nSending first request...');
    const chatResponse = await gonkaClient.chat.completions.create({
      model: defaultModel,
      messages: [{ role: 'user', content: 'Hello! Tell me a short joke.' }],
    });
    
    console.log('\nResponse from first request:');
    console.log(chatResponse.choices[0]?.message?.content);
    
    // Example 2: Using the original OpenAI client with gonkaFetch
    console.log('\n\n------ Example 2: Using original OpenAI client with gonkaFetch ------');
    
    // Get a custom fetch function configured with our private key
    const customFetch = gonkaFetch({
      gonkaPrivateKey: process.env.GONKA_PRIVATE_KEY
    });
    
    // Create a standard OpenAI client with our custom fetch
    const openaiClient = new OpenAI({
      apiKey: 'mock-api-key', // This can be any string when using Gonka
      baseURL: selectedEndpoint.url, // Use the URL property from the endpoint
      fetch: customFetch
    });
    
    // Make a request with the standard client (will be signed by our custom fetch)
    console.log('\nSending request with standard client + custom fetch...');
    const standardResponse = await openaiClient.chat.completions.create({
      model: defaultModel,
      messages: [{ role: 'user', content: 'What is the capital of France?' }],
    });
    
    console.log('\nResponse from standard client:');
    console.log(standardResponse.choices[0]?.message?.content);
    
    if (USE_REAL_REQUESTS) {
      console.log('\nNote: These were REAL API responses through the Gonka network');
    } else {
      console.log('\nNote: These were MOCK responses (no actual API calls were made)');
    }

  } catch (error) {
    console.error('Error occurred:', error);
  } finally {
    // Restore original fetch if we mocked it
    if (!USE_REAL_REQUESTS) {
      global.fetch = originalFetch;
    }
  }
};

run(); 
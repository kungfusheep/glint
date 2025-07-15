/**
 * Basic example of using the Glint TypeScript decoder
 */

import { decode, GlintDecoder } from '../src/index';
import * as fs from 'fs';
import * as path from 'path';

// Mock data to demonstrate the API
// In a real scenario, this would be actual Glint binary data
function createMockGlintData(): Uint8Array {
  // This is a simplified mock - in reality you'd have actual Glint encoded data
  // For demo purposes, we'll create a minimal structure
  const data = new Uint8Array([
    0x00,                           // flags
    0x12, 0x34, 0x56, 0x78,        // CRC32 (little endian)
    0x10,                           // schema size (16 bytes)
    // Schema (simplified)
    0x0E, 0x04, 0x6E, 0x61, 0x6D, 0x65,  // String field "name"
    0x05, 0x03, 0x61, 0x67, 0x65,        // Int32 field "age"
    // Data
    0x05, 0x41, 0x6C, 0x69, 0x63, 0x65,  // String: "Alice"
    0x3C,                                  // Int32: 30 (zigzag)
  ]);
  
  return data;
}

async function main() {
  console.log('Glint TypeScript Decoder Example\n');

  // Create a decoder with custom limits
  const decoder = new GlintDecoder({
    maxStringLength: 1024,
    maxArrayLength: 100,
  });

  try {
    // Create mock data
    const data = createMockGlintData();
    console.log('Mock Glint data created:', data.length, 'bytes');
    
    // This would fail with our mock data since it's not properly formatted
    // but demonstrates the API
    console.log('Attempting to decode...');
    
    // For demo, catch the error and show what a successful decode would look like
    try {
      const result = decoder.decode(data);
      console.log('Decoded result:', result);
    } catch (error) {
      console.log('Expected error with mock data:', (error as Error).message);
      console.log('\nA successful decode would return something like:');
      console.log({
        name: "Alice",
        age: 30,
        tags: ["engineer", "typescript"],
        metadata: {
          created: new Date(),
          active: true
        }
      });
    }

    // Show convenience function
    console.log('\nUsing convenience function:');
    try {
      const result = decode(data);
      console.log('Result:', result);
    } catch (error) {
      console.log('Same error with convenience function:', (error as Error).message);
    }

  } catch (error) {
    console.error('Error:', error);
  }
}

if (require.main === module) {
  main().catch(console.error);
}
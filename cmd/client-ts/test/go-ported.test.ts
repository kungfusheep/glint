/**
 * Tests ported from Go test cases
 */

import * as assert from 'assert';
import { BinaryReader } from '../src/reader';
import { CodegenGlintDecoder as GlintDecoder } from '../src/decoder';

function runGoPortedTests() {
  console.log('Running Go-ported tests...\n');

  testZigzagVarintEncoding();
  testBinaryReaderRobustness();
  testTypeSpecificDecoding();

  console.log('\n✅ All Go-ported tests passed!');
}

function testZigzagVarintEncoding() {
  console.log('Testing zigzag varint encoding (ported from Go)...');
  
  // Test cases from Go TestZigzagVarintEncoding
  const testCases = [
    { input: 1232, expected: 1232 },
    { input: -1232, expected: -1232 },
    { input: 0, expected: 0 },
    { input: 1, expected: 1 },
    { input: -1, expected: -1 },
    { input: 127, expected: 127 },
    { input: -128, expected: -128 },
    { input: 128, expected: 128 },
    { input: -129, expected: -129 },
  ];
  
  testCases.forEach(({ input, expected }, index) => {
    // Manually encode zigzag
    const encoded = encodeZigzag(input);
    const reader = new BinaryReader(encoded);
    const decoded = reader.readInt();
    
    assert.strictEqual(decoded, expected, 
      `Test case ${index}: zigzag decode failed for ${input}`);
  });
  
  console.log('  ✓ Zigzag varint encoding works correctly');
}

function testBinaryReaderRobustness() {
  console.log('Testing binary reader robustness...');
  
  // Test varint overflow protection
  const overflowData = new Uint8Array(10).fill(0xFF); // All continuation bits set
  const reader = new BinaryReader(overflowData);
  
  try {
    reader.readVarint();
    assert.fail('Should have thrown varint overflow error');
  } catch (error) {
    assert.ok((error as Error).message.includes('overflow'), 
      'Should throw varint overflow error');
  }
  
  // Test string length validation
  const invalidStringData = new Uint8Array([
    0xFF, 0xFF, 0xFF, 0xFF, 0x0F, // Very large length
    0x48, 0x65, 0x6C, 0x6C, 0x6F  // "Hello"
  ]);
  const reader2 = new BinaryReader(invalidStringData);
  
  try {
    reader2.readString(100); // Limit to 100 bytes
    assert.fail('Should have thrown string length error');
  } catch (error) {
    assert.ok((error as Error).message.includes('exceeds maximum'), 
      'Should throw string length error');
  }
  
  console.log('  ✓ Binary reader robustness checks work');
}

function testTypeSpecificDecoding() {
  console.log('Testing type-specific decoding...');
  
  // Test boolean edge cases
  const boolData = new Uint8Array([0x00, 0x01, 0xFF]);
  const reader = new BinaryReader(boolData);
  
  assert.strictEqual(reader.readBool(), false, 'Should read false for 0x00');
  assert.strictEqual(reader.readBool(), true, 'Should read true for 0x01');
  assert.strictEqual(reader.readBool(), true, 'Should read true for 0xFF');
  
  // Test int8 sign extension
  const int8Data = new Uint8Array([0x80, 0x7F]); // -128, 127
  const reader2 = new BinaryReader(int8Data);
  
  assert.strictEqual(reader2.readInt8(), -128, 'Should read -128');
  assert.strictEqual(reader2.readInt8(), 127, 'Should read 127');
  
  // Test float32 bit pattern
  const float32Bits = 0x40490FDB; // PI in float32
  const float32Data = encodeVarint(float32Bits);
  const reader3 = new BinaryReader(float32Data);
  const pi = reader3.readFloat32();
  
  assert.ok(Math.abs(pi - 3.14159) < 0.001, 
    `Should read PI, got ${pi}`);
  
  console.log('  ✓ Type-specific decoding works correctly');
}

// Helper functions

function encodeZigzag(value: number): Uint8Array {
  const encoded = (value << 1) ^ (value >> 31);
  return encodeVarint(encoded >>> 0);
}

function encodeVarint(value: number): Uint8Array {
  const result: number[] = [];
  
  while (value >= 0x80) {
    result.push((value & 0x7F) | 0x80);
    value >>>= 7;
  }
  result.push(value & 0x7F);
  
  return new Uint8Array(result);
}

// Run tests if this file is executed directly
if (require.main === module) {
  runGoPortedTests();
}

export { runGoPortedTests };
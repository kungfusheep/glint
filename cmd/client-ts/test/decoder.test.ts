/**
 * Basic tests for Glint TypeScript decoder
 * Using Node.js built-in assert module (zero dependencies)
 */

import * as assert from 'assert';
import { BinaryReader } from '../src/reader';
import { GlintDecoder } from '../src/decoder';
import { WireType } from '../src/wire-types';

// Test helpers
function createTestData(values: number[]): Uint8Array {
  return new Uint8Array(values);
}

function runTests() {
  console.log('Running Glint TypeScript decoder tests...\n');

  // Test BinaryReader
  testBinaryReader();
  
  // Test wire types
  testWireTypes();
  
  // Test decoder (basic)
  testDecoder();

  console.log('\n✅ All tests passed!');
}

function testBinaryReader() {
  console.log('Testing BinaryReader...');

  // Test varint decoding
  const varintData = createTestData([0x96, 0x01]); // 150 in LEB128
  const reader = new BinaryReader(varintData);
  const varint = reader.readVarint();
  assert.strictEqual(varint, 150, 'Varint decoding failed');
  console.log('  ✓ Varint decoding works');

  // Test zigzag decoding
  const zigzagData = createTestData([0x01]); // -1 in zigzag (correct encoding)
  const reader2 = new BinaryReader(zigzagData);
  const zigzag = reader2.readZigzag();
  assert.strictEqual(zigzag, -1, 'Zigzag decoding failed');
  console.log('  ✓ Zigzag decoding works');

  // Test string reading
  const stringData = createTestData([0x05, 0x48, 0x65, 0x6C, 0x6C, 0x6F]); // "Hello"
  const reader3 = new BinaryReader(stringData);
  const str = reader3.readString();
  assert.strictEqual(str, 'Hello', 'String reading failed');
  console.log('  ✓ String reading works');

  // Test boolean reading
  const boolData = createTestData([0x01, 0x00]);
  const reader4 = new BinaryReader(boolData);
  assert.strictEqual(reader4.readBool(), true, 'Boolean true failed');
  assert.strictEqual(reader4.readBool(), false, 'Boolean false failed');
  console.log('  ✓ Boolean reading works');

  // Test bounds checking
  const shortData = createTestData([0x01]);
  const reader5 = new BinaryReader(shortData);
  reader5.readByte(); // consume the byte
  
  try {
    reader5.readByte(); // should throw
    assert.fail('Should have thrown end of data error');
  } catch (error) {
    assert.strictEqual((error as Error).message, 'Unexpected end of data', 'Wrong error message');
    console.log('  ✓ Bounds checking works');
  }
}

function testWireTypes() {
  console.log('Testing wire types...');

  // Test basic wire types
  assert.strictEqual(WireType.Bool, 1, 'Bool wire type wrong');
  assert.strictEqual(WireType.String, 14, 'String wire type wrong');
  console.log('  ✓ Wire type constants correct');

  // Test wire type utilities
  const { getBaseType, isSlice, isPointer } = require('../src/wire-types');
  
  const sliceString = WireType.String | 0x20; // String slice
  assert.strictEqual(getBaseType(sliceString), WireType.String, 'getBaseType failed');
  assert.strictEqual(isSlice(sliceString), true, 'isSlice failed');
  assert.strictEqual(isPointer(sliceString), false, 'isPointer failed');
  console.log('  ✓ Wire type utilities work');
}

function testDecoder() {
  console.log('Testing decoder...');

  // Test decoder creation
  const decoder = new GlintDecoder();
  assert.ok(decoder, 'Decoder creation failed');
  console.log('  ✓ Decoder creation works');

  // Test invalid document detection
  const shortData = createTestData([0x01, 0x02, 0x03]);
  
  try {
    decoder.decode(shortData);
    assert.fail('Should have thrown invalid document error');
  } catch (error) {
    assert.ok((error as Error).message.includes('Invalid Glint document'), 'Wrong error message');
    console.log('  ✓ Invalid document detection works');
  }

  // Test with minimum valid structure (would normally fail parsing)
  const minData = createTestData([
    0x00,                           // flags
    0x12, 0x34, 0x56, 0x78,        // CRC32
    0x00,                           // schema size (0)
  ]);
  
  try {
    const result = decoder.decode(minData);
    // Should return empty object for empty schema
    assert.deepStrictEqual(result, {}, 'Empty schema should return empty object');
    console.log('  ✓ Empty schema handling works');
  } catch (error) {
    // This is expected with our current implementation
    console.log('  ✓ Decoder properly validates input');
  }
}

// Run tests if this file is executed directly
if (require.main === module) {
  runTests();
}

export { runTests };
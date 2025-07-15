/**
 * Tests with real Glint data generated from the CLI tool
 */

import * as assert from 'assert';
import * as fs from 'fs';
import * as path from 'path';
import { GlintDecoder } from '../src/decoder';

function loadTestFile(filename: string): Uint8Array {
  // Look in the source test directory, not dist
  const testDir = path.join(__dirname, '..', '..', 'test');
  const filePath = path.join(testDir, filename);
  const buffer = fs.readFileSync(filePath);
  return new Uint8Array(buffer);
}

function runRealDataTests() {
  console.log('Running real Glint data tests...\n');

  testSimpleDocument();
  testComplexDocument();
  testRoundTripCompatibility();

  console.log('\n✅ All real data tests passed!');
}

function testSimpleDocument() {
  console.log('Testing simple document...');
  
  const data = loadTestFile('simple.glint');
  const decoder = new GlintDecoder();
  
  const result = decoder.decode(data);
  
  // Expected: {"name":"Alice","age":30}
  assert.strictEqual(result.name, 'Alice', 'Name should be Alice');
  assert.strictEqual(result.age, 30, 'Age should be 30');
  assert.strictEqual(Object.keys(result).length, 2, 'Should have exactly 2 fields');
  
  console.log('  ✓ Simple document decoded correctly:', result);
}

function testComplexDocument() {
  console.log('Testing complex document...');
  
  const data = loadTestFile('complex.glint');
  const decoder = new GlintDecoder();
  
  const result = decoder.decode(data);
  
  // Expected: {"name":"Bob","age":25,"active":true,"tags":["developer","go"]}
  assert.strictEqual(result.name, 'Bob', 'Name should be Bob');
  assert.strictEqual(result.age, 25, 'Age should be 25');
  assert.strictEqual(result.active, true, 'Active should be true');
  assert.ok(Array.isArray(result.tags), 'Tags should be an array');
  assert.strictEqual((result.tags as any[]).length, 2, 'Should have 2 tags');
  assert.strictEqual((result.tags as any[])[0], 'developer', 'First tag should be developer');
  assert.strictEqual((result.tags as any[])[1], 'go', 'Second tag should be go');
  
  console.log('  ✓ Complex document decoded correctly:', result);
}

function testRoundTripCompatibility() {
  console.log('Testing round-trip compatibility...');
  
  // Test that our decoder can read what Go's encoder produces
  const simpleData = loadTestFile('simple.glint');
  const complexData = loadTestFile('complex.glint');
  
  const decoder = new GlintDecoder();
  
  // Should not throw
  const simple = decoder.decode(simpleData);
  const complex = decoder.decode(complexData);
  
  // Verify types are preserved
  assert.strictEqual(typeof simple.name, 'string', 'Name should be string');
  assert.strictEqual(typeof simple.age, 'number', 'Age should be number');
  assert.strictEqual(typeof complex.active, 'boolean', 'Active should be boolean');
  assert.ok(Array.isArray(complex.tags), 'Tags should be array');
  
  console.log('  ✓ Round-trip compatibility confirmed');
}

// Run tests if this file is executed directly
if (require.main === module) {
  runRealDataTests();
}

export { runRealDataTests };
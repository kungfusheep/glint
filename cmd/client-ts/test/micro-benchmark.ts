/**
 * Micro-benchmarks to identify performance bottlenecks
 */

import * as fs from 'fs';
import * as path from 'path';
import { GlintDecoder } from '../src/decoder';
import { BinaryReader } from '../src/reader';

// High-resolution timer
function benchmark(name: string, fn: () => void, iterations: number): void {
  // Warmup
  for (let i = 0; i < 100; i++) fn();
  
  // Actual benchmark
  const start = process.hrtime.bigint();
  for (let i = 0; i < iterations; i++) {
    fn();
  }
  const end = process.hrtime.bigint();
  
  const totalNs = Number(end - start);
  const avgNs = totalNs / iterations;
  const avgUs = avgNs / 1000;
  
  console.log(`${name.padEnd(40)} ${iterations.toString().padStart(10)} iterations  ${avgUs.toFixed(3).padStart(10)} µs/op`);
}

function runMicroBenchmarks(): void {
  console.log('🚀 Micro-benchmarks\n');
  
  // Test data
  const testDir = path.join(__dirname, '..', '..', 'test');
  const simpleData = new Uint8Array(fs.readFileSync(path.join(testDir, 'simple.glint')));
  const complexData = new Uint8Array(fs.readFileSync(path.join(testDir, 'complex.glint')));
  const simpleJson = fs.readFileSync(path.join(testDir, 'simple.json'), 'utf8');
  const complexJson = fs.readFileSync(path.join(testDir, 'complex.json'), 'utf8');
  
  console.log('Basic operations:');
  console.log('='.repeat(70));
  
  // Varint decoding
  const varintData = new Uint8Array([0x96, 0x01]); // 150
  benchmark('Varint decode (small)', () => {
    const reader = new BinaryReader(varintData);
    reader.readVarint();
  }, 1000000);
  
  const largeVarint = new Uint8Array([0xff, 0xff, 0xff, 0xff, 0x0f]); // max uint32
  benchmark('Varint decode (large)', () => {
    const reader = new BinaryReader(largeVarint);
    reader.readVarint();
  }, 1000000);
  
  // String decoding
  const smallString = new Uint8Array([0x05, 0x48, 0x65, 0x6c, 0x6c, 0x6f]); // "Hello"
  benchmark('String decode (5 chars)', () => {
    const reader = new BinaryReader(smallString);
    reader.readString();
  }, 1000000);
  
  const mediumString = new Uint8Array([20, ...Array.from('Hello World! Testing').map(c => c.charCodeAt(0))]);
  benchmark('String decode (20 chars)', () => {
    const reader = new BinaryReader(mediumString);
    reader.readString();
  }, 1000000);
  
  // Byte operations
  benchmark('Read single byte', () => {
    const reader = new BinaryReader(new Uint8Array([42]));
    reader.readByte();
  }, 1000000);
  
  benchmark('Read 4 bytes', () => {
    const reader = new BinaryReader(new Uint8Array([1, 2, 3, 4, 5, 6]));
    reader.readBytes(4);
  }, 1000000);
  
  // Number decoding
  const int32Data = new Uint8Array([0x78, 0x56, 0x34, 0x12]); // little-endian
  benchmark('Int32 decode', () => {
    const reader = new BinaryReader(int32Data);
    reader.readInt32();
  }, 1000000);
  
  // TextDecoder performance
  const helloBytes = new Uint8Array([72, 101, 108, 108, 111]);
  const textDecoder = new TextDecoder();
  benchmark('TextDecoder.decode', () => {
    textDecoder.decode(helloBytes);
  }, 1000000);
  
  // Buffer.toString comparison
  benchmark('Buffer.toString', () => {
    Buffer.from(helloBytes).toString('utf-8');
  }, 1000000);
  
  console.log('\nFull document parsing:');
  console.log('='.repeat(70));
  
  // Decoder creation
  benchmark('GlintDecoder creation', () => {
    new GlintDecoder();
  }, 100000);
  
  // Full decoding
  const glintDecoder = new GlintDecoder();
  benchmark('Glint decode (simple)', () => {
    glintDecoder.decode(simpleData);
  }, 10000);
  
  benchmark('Glint decode (complex)', () => {
    glintDecoder.decode(complexData);
  }, 10000);
  
  // JSON comparison
  benchmark('JSON parse (simple)', () => {
    JSON.parse(simpleJson);
  }, 10000);
  
  benchmark('JSON parse (complex)', () => {
    JSON.parse(complexJson);
  }, 10000);
  
  console.log('\nMemory allocations:');
  console.log('='.repeat(70));
  
  // Measure object creation overhead
  benchmark('Empty object creation', () => {
    const obj = {};
  }, 1000000);
  
  benchmark('Object with 5 properties', () => {
    const obj = { a: 1, b: 2, c: 3, d: 4, e: 5 };
  }, 1000000);
  
  benchmark('Array creation (empty)', () => {
    const arr = [];
  }, 1000000);
  
  benchmark('Array creation (5 elements)', () => {
    const arr = [1, 2, 3, 4, 5];
  }, 1000000);
  
  benchmark('Uint8Array slice', () => {
    const data = new Uint8Array([1, 2, 3, 4, 5, 6, 7, 8]);
    data.slice(2, 6);
  }, 1000000);
  
  benchmark('Uint8Array subarray', () => {
    const data = new Uint8Array([1, 2, 3, 4, 5, 6, 7, 8]);
    data.subarray(2, 6);
  }, 1000000);
  
  // Analysis
  console.log('\n📊 Performance Analysis:');
  console.log('='.repeat(70));
  
  // Calculate ratios
  const glintSimpleTime = 727.6; // from previous benchmark
  const jsonSimpleTime = 140.7;
  const glintComplexTime = 1279.7;
  const jsonComplexTime = 259.3;
  
  console.log(`Simple: Glint is ${(glintSimpleTime / jsonSimpleTime).toFixed(1)}x slower than JSON`);
  console.log(`Complex: Glint is ${(glintComplexTime / jsonComplexTime).toFixed(1)}x slower than JSON`);
  
  // Size comparison
  console.log(`\nSize efficiency:`);
  console.log(`Simple: Glint ${simpleData.length}B vs JSON ${simpleJson.length}B (${((1 - simpleData.length/simpleJson.length) * 100).toFixed(1)}% smaller)`);
  console.log(`Complex: Glint ${complexData.length}B vs JSON ${complexJson.length}B (${((1 - complexData.length/complexJson.length) * 100).toFixed(1)}% smaller)`);
}

// Run benchmarks
runMicroBenchmarks();
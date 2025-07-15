/**
 * Compare original vs optimized BinaryReader performance
 */

import { BinaryReader } from '../src/reader';
import { BinaryReaderOptimized } from '../src/reader-optimized';

function benchmark(name: string, fn: () => void, iterations: number): number {
  // Warmup
  for (let i = 0; i < 100; i++) fn();
  
  const start = process.hrtime.bigint();
  for (let i = 0; i < iterations; i++) {
    fn();
  }
  const end = process.hrtime.bigint();
  
  const totalNs = Number(end - start);
  const avgNs = totalNs / iterations;
  const avgUs = avgNs / 1000;
  
  console.log(`${name.padEnd(40)} ${iterations.toString().padStart(10)} iterations  ${avgUs.toFixed(3).padStart(10)} µs/op`);
  return avgUs;
}

function compareReaders(): void {
  console.log('🚀 BinaryReader Performance Comparison\n');
  
  // Test data
  const singleByteVarint = new Uint8Array([42]); // Single byte
  const twoByteVarint = new Uint8Array([0x96, 0x01]); // 150
  const threeByteVarint = new Uint8Array([0xE5, 0x8E, 0x26]); // 624485
  const largeVarint = new Uint8Array([0xff, 0xff, 0xff, 0xff, 0x0f]); // max uint32
  
  const smallString = new Uint8Array([0x05, 0x48, 0x65, 0x6c, 0x6c, 0x6f]); // "Hello"
  const mediumString = new Uint8Array([20, ...Array.from('Hello World! Testing').map(c => c.charCodeAt(0))]);
  const largeString = new Uint8Array([100, ...Array.from('Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore.').map(c => c.charCodeAt(0))]);
  
  console.log('Varint decoding:');
  console.log('='.repeat(70));
  
  // Single byte varint
  const orig1 = benchmark('Original - single byte varint', () => {
    const reader = new BinaryReader(singleByteVarint);
    reader.readVarint();
  }, 1000000);
  
  const opt1 = benchmark('Optimized - single byte varint', () => {
    const reader = new BinaryReaderOptimized(singleByteVarint);
    reader.readVarint();
  }, 1000000);
  
  console.log(`  → Speedup: ${(orig1 / opt1).toFixed(2)}x\n`);
  
  // Two byte varint
  const orig2 = benchmark('Original - two byte varint', () => {
    const reader = new BinaryReader(twoByteVarint);
    reader.readVarint();
  }, 1000000);
  
  const opt2 = benchmark('Optimized - two byte varint', () => {
    const reader = new BinaryReaderOptimized(twoByteVarint);
    reader.readVarint();
  }, 1000000);
  
  console.log(`  → Speedup: ${(orig2 / opt2).toFixed(2)}x\n`);
  
  // Three byte varint
  const orig3 = benchmark('Original - three byte varint', () => {
    const reader = new BinaryReader(threeByteVarint);
    reader.readVarint();
  }, 1000000);
  
  const opt3 = benchmark('Optimized - three byte varint', () => {
    const reader = new BinaryReaderOptimized(threeByteVarint);
    reader.readVarint();
  }, 1000000);
  
  console.log(`  → Speedup: ${(orig3 / opt3).toFixed(2)}x\n`);
  
  // Large varint
  const orig4 = benchmark('Original - large varint', () => {
    const reader = new BinaryReader(largeVarint);
    reader.readVarint();
  }, 1000000);
  
  const opt4 = benchmark('Optimized - large varint', () => {
    const reader = new BinaryReaderOptimized(largeVarint);
    reader.readVarint();
  }, 1000000);
  
  console.log(`  → Speedup: ${(orig4 / opt4).toFixed(2)}x\n`);
  
  console.log('\nString decoding:');
  console.log('='.repeat(70));
  
  // Small string
  const origStr1 = benchmark('Original - small string (5 chars)', () => {
    const reader = new BinaryReader(smallString);
    reader.readString();
  }, 100000);
  
  const optStr1 = benchmark('Optimized - small string (5 chars)', () => {
    const reader = new BinaryReaderOptimized(smallString);
    reader.readString();
  }, 100000);
  
  console.log(`  → Speedup: ${(origStr1 / optStr1).toFixed(2)}x\n`);
  
  // Medium string
  const origStr2 = benchmark('Original - medium string (20 chars)', () => {
    const reader = new BinaryReader(mediumString);
    reader.readString();
  }, 100000);
  
  const optStr2 = benchmark('Optimized - medium string (20 chars)', () => {
    const reader = new BinaryReaderOptimized(mediumString);
    reader.readString();
  }, 100000);
  
  console.log(`  → Speedup: ${(origStr2 / optStr2).toFixed(2)}x\n`);
  
  // Large string
  const origStr3 = benchmark('Original - large string (100 chars)', () => {
    const reader = new BinaryReader(largeString);
    reader.readString();
  }, 100000);
  
  const optStr3 = benchmark('Optimized - large string (100 chars)', () => {
    const reader = new BinaryReaderOptimized(largeString);
    reader.readString();
  }, 100000);
  
  console.log(`  → Speedup: ${(origStr3 / optStr3).toFixed(2)}x\n`);
  
  console.log('\nOther operations:');
  console.log('='.repeat(70));
  
  // Read bytes
  const bytesData = new Uint8Array([1, 2, 3, 4, 5, 6, 7, 8]);
  
  const origBytes = benchmark('Original - read 4 bytes', () => {
    const reader = new BinaryReader(bytesData);
    reader.readBytes(4);
  }, 1000000);
  
  const optBytes = benchmark('Optimized - read 4 bytes', () => {
    const reader = new BinaryReaderOptimized(bytesData);
    reader.readBytes(4);
  }, 1000000);
  
  console.log(`  → Speedup: ${(origBytes / optBytes).toFixed(2)}x\n`);
  
  // Summary
  console.log('\n📊 Summary:');
  console.log('='.repeat(70));
  const avgSpeedup = ((orig1/opt1 + orig2/opt2 + orig3/opt3 + orig4/opt4) / 4);
  console.log(`Average varint speedup: ${avgSpeedup.toFixed(2)}x`);
  const avgStringSpeedup = ((origStr1/optStr1 + origStr2/optStr2 + origStr3/optStr3) / 3);
  console.log(`Average string speedup: ${avgStringSpeedup.toFixed(2)}x`);
}

compareReaders();
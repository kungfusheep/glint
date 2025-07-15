/**
 * Detailed performance profiling tool
 */

import * as fs from 'fs';
import * as path from 'path';
import { GlintDecoder } from '../src/decoder';
import { BinaryReader } from '../src/reader';

// Performance timing utilities
class Timer {
  private timings: Map<string, number[]> = new Map();
  
  start(label: string): () => void {
    const startTime = process.hrtime.bigint();
    return () => {
      const endTime = process.hrtime.bigint();
      const durationNs = Number(endTime - startTime);
      
      if (!this.timings.has(label)) {
        this.timings.set(label, []);
      }
      this.timings.get(label)!.push(durationNs);
    };
  }
  
  getStats(label: string): { avg: number, min: number, max: number, count: number } {
    const times = this.timings.get(label) || [];
    if (times.length === 0) return { avg: 0, min: 0, max: 0, count: 0 };
    
    const sum = times.reduce((a, b) => a + b, 0);
    const avg = sum / times.length;
    const min = Math.min(...times);
    const max = Math.max(...times);
    
    return { avg, min, max, count: times.length };
  }
  
  getAllStats(): Map<string, { avg: number, min: number, max: number, count: number }> {
    const stats = new Map();
    for (const [label, _] of this.timings) {
      stats.set(label, this.getStats(label));
    }
    return stats;
  }
}

// Instrumented BinaryReader for profiling
class ProfiledBinaryReader extends BinaryReader {
  private timer: Timer;
  
  constructor(data: Uint8Array, timer: Timer) {
    super(data);
    this.timer = timer;
  }
  
  readVarint(): number {
    const end = this.timer.start('readVarint');
    const result = super.readVarint();
    end();
    return result;
  }
  
  readByte(): number {
    const end = this.timer.start('readByte');
    const result = super.readByte();
    end();
    return result;
  }
  
  readBytes(length: number): Uint8Array {
    const end = this.timer.start('readBytes');
    const result = super.readBytes(length);
    end();
    return result;
  }
  
  readString(maxLength?: number): string {
    const end = this.timer.start('readString');
    const result = super.readString(maxLength);
    end();
    return result;
  }
  
  readInt(): number {
    const end = this.timer.start('readInt');
    const result = super.readInt();
    end();
    return result;
  }
  
  // Add other read methods as needed
}

// Instrumented GlintDecoder for profiling
class ProfiledGlintDecoder extends GlintDecoder {
  private timer: Timer;
  
  constructor(timer: Timer) {
    super();
    this.timer = timer;
  }
  
  decode(data: Uint8Array): any {
    const end = this.timer.start('decode-total');
    
    // Use ProfiledBinaryReader
    const originalDecode = super.decode.bind(this);
    
    // We need to hook into the decoder's internal reader creation
    // For now, let's profile the high-level operations
    
    // Header parsing
    const headerEnd = this.timer.start('decode-header');
    const reader = new ProfiledBinaryReader(data, this.timer);
    const flags = reader.readByte();
    const crcBytes = reader.readBytes(4);
    const schemaSize = reader.readVarint();
    headerEnd();
    
    // Schema parsing
    const schemaEnd = this.timer.start('decode-schema');
    // Continue with normal decode...
    const result = originalDecode(data);
    schemaEnd();
    
    end();
    return result;
  }
}

async function profilePerformance(): Promise<void> {
  console.log('🔬 Detailed Performance Profiling\n');
  
  const testDir = path.join(__dirname, '..', '..', 'test');
  const timer = new Timer();
  
  // Test simple dataset many times
  const simpleData = new Uint8Array(fs.readFileSync(path.join(testDir, 'simple.glint')));
  const complexData = new Uint8Array(fs.readFileSync(path.join(testDir, 'complex.glint')));
  
  // Create decoders
  const normalDecoder = new GlintDecoder();
  const decoder = new ProfiledGlintDecoder(timer);
  
  // Warmup
  console.log('🔥 Warming up...');
  for (let i = 0; i < 1000; i++) {
    normalDecoder.decode(simpleData);
    normalDecoder.decode(complexData);
  }
  
  // Profile varint decoding specifically
  console.log('\n📊 Profiling varint decoding...');
  const varintTimer = new Timer();
  const testVarint = new Uint8Array([0x96, 0x01]); // 150
  const varintReader = new ProfiledBinaryReader(testVarint, varintTimer);
  
  for (let i = 0; i < 1000000; i++) {
    const reader = new ProfiledBinaryReader(testVarint, varintTimer);
    reader.readVarint();
  }
  
  // Profile string decoding
  console.log('📊 Profiling string decoding...');
  const stringTimer = new Timer();
  const testString = new Uint8Array([0x05, 0x48, 0x65, 0x6c, 0x6c, 0x6f]); // "Hello"
  const stringReader = new ProfiledBinaryReader(testString, stringTimer);
  
  for (let i = 0; i < 1000000; i++) {
    const reader = new ProfiledBinaryReader(testString, stringTimer);
    reader.readString();
  }
  
  // Profile full decoding
  console.log('📊 Profiling full decoding...');
  const iterations = 10000;
  
  for (let i = 0; i < iterations; i++) {
    decoder.decode(simpleData);
    decoder.decode(complexData);
  }
  
  // JSON comparison
  console.log('📊 Profiling JSON parsing...');
  const simpleJson = fs.readFileSync(path.join(testDir, 'simple.json'), 'utf8');
  const complexJson = fs.readFileSync(path.join(testDir, 'complex.json'), 'utf8');
  const jsonTimer = new Timer();
  
  for (let i = 0; i < iterations; i++) {
    const parseSimple = jsonTimer.start('json-simple');
    JSON.parse(simpleJson);
    parseSimple();
    
    const parseComplex = jsonTimer.start('json-complex');
    JSON.parse(complexJson);
    parseComplex();
  }
  
  // Print results
  console.log('\n📈 Results:\n');
  
  // Varint stats
  const varintStats = varintTimer.getStats('readVarint');
  console.log(`Varint decoding (1M iterations):`);
  console.log(`  Average: ${(varintStats.avg / 1000).toFixed(2)} µs`);
  console.log(`  Min: ${(varintStats.min / 1000).toFixed(2)} µs`);
  console.log(`  Max: ${(varintStats.max / 1000).toFixed(2)} µs`);
  
  // String stats
  const stringStats = stringTimer.getStats('readString');
  console.log(`\nString decoding (1M iterations):`);
  console.log(`  Average: ${(stringStats.avg / 1000).toFixed(2)} µs`);
  console.log(`  Min: ${(stringStats.min / 1000).toFixed(2)} µs`);
  console.log(`  Max: ${(stringStats.max / 1000).toFixed(2)} µs`);
  
  // Full decode stats
  console.log(`\nFull decoding stats (${iterations} iterations):`);
  const allStats = timer.getAllStats();
  for (const [label, stats] of allStats) {
    console.log(`\n${label}:`);
    console.log(`  Average: ${(stats.avg / 1000).toFixed(2)} µs`);
    console.log(`  Total calls: ${stats.count}`);
    console.log(`  Calls per decode: ${(stats.count / (iterations * 2)).toFixed(1)}`);
  }
  
  // JSON stats
  const jsonSimpleStats = jsonTimer.getStats('json-simple');
  const jsonComplexStats = jsonTimer.getStats('json-complex');
  console.log(`\nJSON parsing (${iterations} iterations):`);
  console.log(`  Simple: ${(jsonSimpleStats.avg / 1000).toFixed(2)} µs`);
  console.log(`  Complex: ${(jsonComplexStats.avg / 1000).toFixed(2)} µs`);
  
  // Hot spots analysis
  console.log('\n🔥 Hot spots:');
  const totalTime = timer.getStats('decode-total').avg;
  const sortedStats = Array.from(allStats.entries())
    .filter(([label]) => label !== 'decode-total')
    .sort((a, b) => (b[1].avg * b[1].count) - (a[1].avg * a[1].count));
  
  for (const [label, stats] of sortedStats) {
    const totalTimeInLabel = (stats.avg * stats.count) / (iterations * 2);
    const percentage = (totalTimeInLabel / totalTime) * 100;
    if (percentage > 1) {
      console.log(`  ${label}: ${percentage.toFixed(1)}% of total time`);
    }
  }
}

// Run profiling
profilePerformance().catch(console.error);
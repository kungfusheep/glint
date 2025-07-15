/**
 * Compare original vs optimized decoder performance
 */

import * as fs from 'fs';
import * as path from 'path';
import { GlintDecoder } from '../src/decoder';
import { GlintDecoderOptimized } from '../src/decoder-optimized';

function benchmark(name: string, fn: () => void, iterations: number): number {
  // Warmup
  for (let i = 0; i < 100; i++) fn();
  
  // Force GC if available
  if (global.gc) global.gc();
  
  const start = process.hrtime.bigint();
  for (let i = 0; i < iterations; i++) {
    fn();
  }
  const end = process.hrtime.bigint();
  
  const totalNs = Number(end - start);
  const avgNs = totalNs / iterations;
  const avgUs = avgNs / 1000;
  
  console.log(`${name.padEnd(50)} ${iterations.toString().padStart(8)} iterations  ${avgUs.toFixed(3).padStart(10)} µs/op`);
  return avgUs;
}

function compareDecoders(): void {
  console.log('🚀 Decoder Performance Comparison\n');
  
  const testDir = path.join(__dirname, '..', '..', 'test');
  const simpleData = new Uint8Array(fs.readFileSync(path.join(testDir, 'simple.glint')));
  const complexData = new Uint8Array(fs.readFileSync(path.join(testDir, 'complex.glint')));
  const simpleJson = fs.readFileSync(path.join(testDir, 'simple.json'), 'utf8');
  const complexJson = fs.readFileSync(path.join(testDir, 'complex.json'), 'utf8');
  
  const originalDecoder = new GlintDecoder();
  const optimizedDecoder = new GlintDecoderOptimized();
  
  console.log('Simple document decoding:');
  console.log('='.repeat(80));
  
  const origSimple = benchmark('Original decoder - simple', () => {
    originalDecoder.decode(simpleData);
  }, 10000);
  
  const optSimple = benchmark('Optimized decoder - simple', () => {
    optimizedDecoder.decode(simpleData);
  }, 10000);
  
  const jsonSimple = benchmark('JSON.parse - simple', () => {
    JSON.parse(simpleJson);
  }, 10000);
  
  console.log(`\n  → Optimization speedup: ${(origSimple / optSimple).toFixed(2)}x`);
  console.log(`  → vs JSON: Original is ${(origSimple / jsonSimple).toFixed(1)}x slower`);
  console.log(`  → vs JSON: Optimized is ${(optSimple / jsonSimple).toFixed(1)}x slower`);
  
  console.log('\n\nComplex document decoding:');
  console.log('='.repeat(80));
  
  const origComplex = benchmark('Original decoder - complex', () => {
    originalDecoder.decode(complexData);
  }, 10000);
  
  const optComplex = benchmark('Optimized decoder - complex', () => {
    optimizedDecoder.decode(complexData);
  }, 10000);
  
  const jsonComplex = benchmark('JSON.parse - complex', () => {
    JSON.parse(complexJson);
  }, 10000);
  
  console.log(`\n  → Optimization speedup: ${(origComplex / optComplex).toFixed(2)}x`);
  console.log(`  → vs JSON: Original is ${(origComplex / jsonComplex).toFixed(1)}x slower`);
  console.log(`  → vs JSON: Optimized is ${(optComplex / jsonComplex).toFixed(1)}x slower`);
  
  // Test with warm cache
  console.log('\n\nWith warm cache (100k iterations):');
  console.log('='.repeat(80));
  
  // Pre-warm the cache
  for (let i = 0; i < 100; i++) {
    originalDecoder.decode(simpleData);
    originalDecoder.decode(complexData);
    optimizedDecoder.decode(simpleData);
    optimizedDecoder.decode(complexData);
  }
  
  const origWarm = benchmark('Original decoder - warm cache', () => {
    originalDecoder.decode(simpleData);
    originalDecoder.decode(complexData);
  }, 50000);
  
  const optWarm = benchmark('Optimized decoder - warm cache', () => {
    optimizedDecoder.decode(simpleData);
    optimizedDecoder.decode(complexData);
  }, 50000);
  
  const jsonWarm = benchmark('JSON.parse - reference', () => {
    JSON.parse(simpleJson);
    JSON.parse(complexJson);
  }, 50000);
  
  console.log(`\n  → Optimization speedup: ${(origWarm / optWarm).toFixed(2)}x`);
  console.log(`  → vs JSON: Optimized is ${(optWarm / jsonWarm).toFixed(1)}x slower`);
  
  // Cache statistics
  console.log('\n\nCache Statistics:');
  console.log('='.repeat(80));
  const origStats = originalDecoder.getCacheStats();
  const optStats = optimizedDecoder.getCacheStats();
  
  console.log(`Original decoder - Hits: ${origStats.hits}, Misses: ${origStats.misses}, Hit rate: ${(origStats.hitRate * 100).toFixed(1)}%`);
  console.log(`Optimized decoder - Hits: ${optStats.hits}, Misses: ${optStats.misses}, Hit rate: ${(optStats.hitRate * 100).toFixed(1)}%`);
  
  // Final summary
  console.log('\n\n📊 Summary:');
  console.log('='.repeat(80));
  const avgSpeedup = ((origSimple/optSimple) + (origComplex/optComplex) + (origWarm/optWarm)) / 3;
  console.log(`Average speedup from optimization: ${avgSpeedup.toFixed(2)}x`);
  
  const percentFaster = ((origWarm - optWarm) / origWarm) * 100;
  console.log(`Optimized decoder is ${percentFaster.toFixed(0)}% faster`);
  
  console.log(`\nPerformance vs JSON:`);
  console.log(`  Simple: ${(optSimple / jsonSimple).toFixed(1)}x slower than JSON`);
  console.log(`  Complex: ${(optComplex / jsonComplex).toFixed(1)}x slower than JSON`);
  console.log(`  Goal: Get within 2x of JSON performance`);
}

// Add method to original decoder for cache stats
declare module '../src/decoder' {
  interface GlintDecoder {
    getCacheStats(): { hits: number, misses: number, hitRate: number };
  }
}

// Monkey patch for testing
(GlintDecoder.prototype as any).getCacheStats = function() {
  const stats = (this as any).cacheStats || { hits: 0, misses: 0 };
  const total = stats.hits + stats.misses;
  const hitRate = total > 0 ? stats.hits / total : 0;
  return { ...stats, hitRate };
};

// Run comparison
compareDecoders();
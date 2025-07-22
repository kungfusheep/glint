/**
 * Glint Decoder Performance Benchmark
 * 
 * Compares Glint vs JSON performance across different document sizes
 * and provides insights into what operations are slow
 */

import * as fs from 'fs';
import * as path from 'path';
import { GlintDecoder } from '../src/decoder';

interface BenchmarkResult {
  name: string;
  iterations: number;
  totalMs: number;
  avgNs: number;
  opsPerSec: number;
  dataSize: number;
  mbPerSec: number;
}

interface OperationTiming {
  operation: string;
  totalNs: number;
  count: number;
  avgNs: number;
  percentage: number;
}

class PerformanceProfiler {
  private timings: Map<string, { total: number; count: number }> = new Map();
  private startTime: bigint = 0n;
  
  start(): void {
    this.startTime = process.hrtime.bigint();
  }
  
  measure(operation: string, fn: () => void): void {
    const start = process.hrtime.bigint();
    fn();
    const duration = Number(process.hrtime.bigint() - start);
    
    const timing = this.timings.get(operation) || { total: 0, count: 0 };
    timing.total += duration;
    timing.count++;
    this.timings.set(operation, timing);
  }
  
  getResults(): OperationTiming[] {
    const totalTime = Number(process.hrtime.bigint() - this.startTime);
    const results: OperationTiming[] = [];
    
    for (const [operation, timing] of this.timings) {
      results.push({
        operation,
        totalNs: timing.total,
        count: timing.count,
        avgNs: timing.total / timing.count,
        percentage: (timing.total / totalTime) * 100
      });
    }
    
    return results.sort((a, b) => b.totalNs - a.totalNs);
  }
  
  clear(): void {
    this.timings.clear();
  }
}

// Note: Detailed profiling removed - use external profiling tools for analysis

function runBenchmark(name: string, data: Uint8Array | string, decoder: any, targetTime: number = 2000): BenchmarkResult {
  // Warmup
  for (let i = 0; i < 100; i++) {
    if (typeof data === 'string') {
      JSON.parse(data);
    } else {
      decoder.decode(data);
    }
  }
  
  // Force GC if available
  if (global.gc) global.gc();
  
  // Go-style adaptive benchmarking
  let iterations = 1;
  let duration = 0;
  
  // Keep doubling iterations until we get at least 1 second of runtime
  while (duration < 1000) {
    const start = process.hrtime.bigint();
    
    for (let i = 0; i < iterations; i++) {
      if (typeof data === 'string') {
        JSON.parse(data);
      } else {
        decoder.decode(data);
      }
    }
    
    const end = process.hrtime.bigint();
    duration = Number(end - start) / 1000000; // Convert to ms
    
    if (duration < 1000) {
      // Double the iterations for next attempt
      iterations *= 2;
    }
  }
  
  // Now run for the target time with progress updates
  const start = process.hrtime.bigint();
  let totalIterations = 0;
  let lastUpdate = 0;
  
  process.stdout.write(`${iterations.toLocaleString()}-`);
  
  while (true) {
    const batchStart = process.hrtime.bigint();
    
    for (let i = 0; i < iterations; i++) {
      if (typeof data === 'string') {
        JSON.parse(data);
      } else {
        decoder.decode(data);
      }
    }
    
    const batchEnd = process.hrtime.bigint();
    totalIterations += iterations;
    
    const elapsed = Number(batchEnd - start) / 1000000;
    
    // Show progress every 1000ms
    if (elapsed - lastUpdate > 1000) {
      process.stdout.write(`${totalIterations.toLocaleString()}-`);
      lastUpdate = elapsed;
    }
    
    // Stop if we've run long enough
    if (elapsed > targetTime) {
      break;
    }
  }
  
  const end = process.hrtime.bigint();
  const totalNs = Number(end - start);
  const totalMs = totalNs / 1000000;
  const avgNs = totalNs / totalIterations;
  const opsPerSec = 1000000000 / avgNs;
  
  const dataSize = typeof data === 'string' ? data.length : data.length;
  const mbPerSec = (dataSize * totalIterations / (1024 * 1024)) / (totalMs / 1000);
  
  return {
    name,
    iterations: totalIterations,
    totalMs,
    avgNs,
    opsPerSec,
    dataSize,
    mbPerSec
  };
}

function formatNumber(num: number): string {
  if (num >= 1000000) return `${(num / 1000000).toFixed(2)}M`;
  if (num >= 1000) return `${(num / 1000).toFixed(2)}K`;
  return num.toFixed(0);
}

function formatTime(ns: number): string {
  if (ns >= 1000000) return `${(ns / 1000000).toFixed(2)}ms`;
  if (ns >= 1000) return `${(ns / 1000).toFixed(2)}µs`;
  return `${ns.toFixed(0)}ns`;
}

async function main(): Promise<void> {
  console.log('🚀 Glint Performance Benchmark\n');
  
  const testDir = path.join(__dirname, '..', '..', 'test');
  const decoder = new GlintDecoder();
  
  // Test datasets
  const datasets = [
    { name: 'simple', glintFile: 'simple.glint', jsonFile: 'simple.json' },
    { name: 'complex', glintFile: 'complex.glint', jsonFile: 'complex.json' },
    { name: 'medium', glintFile: 'medium.glint', jsonFile: 'medium.json' },
    { name: 'large', glintFile: 'large.glint', jsonFile: 'large.json' },
    { name: 'huge', glintFile: 'huge.glint', jsonFile: 'huge.json' }
  ];
  
  const results: { dataset: string; glint?: BenchmarkResult; json?: BenchmarkResult; error?: string }[] = [];
  
  // Run benchmarks
  for (const dataset of datasets) {
    const glintPath = path.join(testDir, dataset.glintFile);
    const jsonPath = path.join(testDir, dataset.jsonFile);
    
    if (!fs.existsSync(glintPath) || !fs.existsSync(jsonPath)) {
      results.push({ dataset: dataset.name, error: 'Files not found' });
      continue;
    }
    
    const glintData = new Uint8Array(fs.readFileSync(glintPath));
    const jsonData = fs.readFileSync(jsonPath, 'utf8');
    
    try {
      console.log(`\nBenchmark${dataset.name.padEnd(8)} `);
      
      // Test if Glint decoding works first
      let glintResult;
      try {
        decoder.decode(glintData);
        process.stdout.write(`  Glint: `);
        glintResult = runBenchmark(`Glint-${dataset.name}`, glintData, decoder, 2000);
        console.log(` ${glintResult.iterations.toLocaleString()} iterations (${glintResult.totalMs.toFixed(0)}ms)`);
      } catch (decodeError) {
        console.log(`  Glint: ❌ Decode error - ${(decodeError as Error).message}`);
        results.push({ dataset: dataset.name, error: `Glint decode error: ${(decodeError as Error).message}` });
        continue;
      }
      
      process.stdout.write(`  JSON:  `);
      const jsonResult = runBenchmark(`JSON-${dataset.name}`, jsonData, null, 2000);
      console.log(` ${jsonResult.iterations.toLocaleString()} iterations (${jsonResult.totalMs.toFixed(0)}ms)`);
      
      results.push({ dataset: dataset.name, glint: glintResult, json: jsonResult });
    } catch (error) {
      console.log(`\n  Error: ${(error as Error).message}`);
      results.push({ dataset: dataset.name, error: (error as Error).message });
    }
  }
  
  // Display results
  console.log('📊 Performance Results');
  console.log('━'.repeat(80));
  console.log('Dataset     Size (B)    Format    Avg Time    Ops/sec     MB/s       vs JSON');
  console.log('━'.repeat(80));
  
  for (const result of results) {
    if (result.error) {
      console.log(`${result.dataset.padEnd(12)} ERROR: ${result.error}`);
      continue;
    }
    
    const { glint, json } = result;
    if (!glint || !json) continue;
    
    const ratio = glint.avgNs / json.avgNs;
    const sizeRatio = ((glint.dataSize - json.dataSize) / json.dataSize) * 100;
    
    // Glint row
    console.log(
      `${result.dataset.padEnd(12)}` +
      `${glint.dataSize.toString().padStart(8)} ` +
      `   Glint     ` +
      `${formatTime(glint.avgNs).padStart(10)} ` +
      `${formatNumber(glint.opsPerSec).padStart(10)} ` +
      `${glint.mbPerSec.toFixed(1).padStart(8)} ` +
      `${ratio.toFixed(1).padStart(7)}x slower`
    );
    
    // JSON row
    console.log(
      `${' '.repeat(12)}` +
      `${json.dataSize.toString().padStart(8)} ` +
      `   JSON      ` +
      `${formatTime(json.avgNs).padStart(10)} ` +
      `${formatNumber(json.opsPerSec).padStart(10)} ` +
      `${json.mbPerSec.toFixed(1).padStart(8)} ` +
      `        -`
    );
    
    // Size comparison
    console.log(
      `${' '.repeat(12)}` +
      `Size diff: ${sizeRatio > 0 ? '+' : ''}${sizeRatio.toFixed(1)}%`
    );
    console.log('─'.repeat(80));
  }
  
  // Cache statistics
  const cacheStats = decoder.getCacheStats();
  console.log('\n📈 Cache Statistics');
  console.log('━'.repeat(80));
  console.log(`Cache hits: ${cacheStats.hits.toLocaleString()}`);
  console.log(`Cache misses: ${cacheStats.misses}`);
  console.log(`Hit rate: ${(cacheStats.hitRate * 100).toFixed(1)}%`);
  
  // Summary
  console.log('\n📌 Summary');
  console.log('━'.repeat(80));
  
  const workingResults = results.filter(r => !r.error && r.glint && r.json);
  if (workingResults.length > 0) {
    const avgRatio = workingResults.reduce((sum, r) => sum + (r.glint!.avgNs / r.json!.avgNs), 0) / workingResults.length;
    console.log(`Average performance: Glint is ${avgRatio.toFixed(1)}x slower than JSON`);
    console.log(`Target: 2x slower than JSON (currently ${((avgRatio - 2) / 2 * 100).toFixed(0)}% over target)`);
  }
}

// Run with error handling
main().catch(console.error);
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

// Instrumented decoder for profiling
class ProfiledGlintDecoder extends GlintDecoder {
  private profiler?: PerformanceProfiler;
  
  setProfiler(profiler: PerformanceProfiler): void {
    this.profiler = profiler;
  }
  
  decode(data: Uint8Array): any {
    if (!this.profiler) {
      return super.decode(data);
    }
    
    let result: any;
    this.profiler.measure('decode-total', () => {
      result = super.decode(data);
    });
    return result;
  }
  
  protected decodeValue(reader: any, wireType: number, subSchema: any, context: any): any {
    if (!this.profiler) {
      return super.decodeValue(reader, wireType, subSchema, context);
    }
    
    const baseType = wireType & 0x1f;
    const typeName = this.getTypeName(baseType);
    
    let result: any;
    this.profiler.measure(`decode-${typeName}`, () => {
      result = super.decodeValue(reader, wireType, subSchema, context);
    });
    return result;
  }
  
  private getTypeName(wireType: number): string {
    const types: { [key: number]: string } = {
      1: 'bool', 2: 'int', 7: 'uint', 14: 'string', 16: 'struct'
    };
    return types[wireType] || 'unknown';
  }
}

function runBenchmark(name: string, data: Uint8Array | string, decoder: any, iterations: number): BenchmarkResult {
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
  
  // Actual benchmark
  const start = process.hrtime.bigint();
  
  for (let i = 0; i < iterations; i++) {
    if (typeof data === 'string') {
      JSON.parse(data);
    } else {
      decoder.decode(data);
    }
  }
  
  const end = process.hrtime.bigint();
  const totalNs = Number(end - start);
  const totalMs = totalNs / 1000000;
  const avgNs = totalNs / iterations;
  const opsPerSec = 1000000000 / avgNs;
  
  const dataSize = typeof data === 'string' ? data.length : data.length;
  const mbPerSec = (dataSize * iterations / (1024 * 1024)) / (totalMs / 1000);
  
  return {
    name,
    iterations,
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
  const profiledDecoder = new ProfiledGlintDecoder();
  
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
    
    // Determine iterations based on data size
    const iterations = glintData.length < 1000 ? 10000 : 
                      glintData.length < 100000 ? 1000 : 
                      glintData.length < 1000000 ? 100 : 10;
    
    try {
      const glintResult = runBenchmark(`Glint-${dataset.name}`, glintData, decoder, iterations);
      const jsonResult = runBenchmark(`JSON-${dataset.name}`, jsonData, null, iterations);
      results.push({ dataset: dataset.name, glint: glintResult, json: jsonResult });
    } catch (error) {
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
  
  // Profile simple dataset to show what's slow
  console.log('\n🔍 Performance Profile (Simple Dataset)');
  console.log('━'.repeat(80));
  
  const profiler = new PerformanceProfiler();
  profiledDecoder.setProfiler(profiler);
  
  const simpleData = new Uint8Array(fs.readFileSync(path.join(testDir, 'simple.glint')));
  
  profiler.start();
  for (let i = 0; i < 1000; i++) {
    profiledDecoder.decode(simpleData);
  }
  
  const timings = profiler.getResults();
  console.log('Operation                Count      Avg Time    Total %');
  console.log('─'.repeat(80));
  
  for (const timing of timings.slice(0, 10)) {
    console.log(
      `${timing.operation.padEnd(24)} ` +
      `${timing.count.toString().padStart(6)} ` +
      `${formatTime(timing.avgNs).padStart(12)} ` +
      `${timing.percentage.toFixed(1).padStart(8)}%`
    );
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
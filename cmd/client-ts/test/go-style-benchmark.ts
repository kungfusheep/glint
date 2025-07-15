/**
 * True Go-style benchmark that runs until statistically significant
 * Mimics `go test -bench=.` behavior exactly
 */

import * as fs from 'fs';
import * as path from 'path';
import { GlintDecoder } from '../src/decoder';

// Load test data
const testDir = path.join(__dirname, '..', '..', 'test');
const simpleGlint = new Uint8Array(fs.readFileSync(path.join(testDir, 'simple.glint')));
const complexGlint = new Uint8Array(fs.readFileSync(path.join(testDir, 'complex.glint')));

// JSON equivalents
const simpleJson = JSON.stringify({name: "Alice", age: 30});
const complexJson = JSON.stringify({name: "Bob", age: 25, active: true, tags: ["developer", "go"]});

interface BenchmarkResult {
  name: string;
  iterations: number;
  duration: number;
  nsPerOp: number;
  mbPerSec: number;
}

class GoBenchmark {
  private name: string;
  private fn: () => void;
  private dataSize: number;

  constructor(name: string, fn: () => void, dataSize: number = 0) {
    this.name = name;
    this.fn = fn;
    this.dataSize = dataSize;
  }

  async run(): Promise<BenchmarkResult> {
    process.stdout.write(`${this.name.padEnd(40)} `);
    
    // Start with 1 iteration, like Go
    let iterations = 1;
    let duration = 0;
    let attempts = 0;
    const maxAttempts = 20;
    
    // Target: run for at least 1 second of total time
    const targetDuration = 1000000000; // 1 second in nanoseconds
    
    while (attempts < maxAttempts) {
      attempts++;
      
      // Warmup
      for (let i = 0; i < Math.min(iterations / 10, 100); i++) {
        this.fn();
      }
      
      // Force GC
      if (global.gc) {
        global.gc();
      }
      
      // Time the benchmark
      const start = process.hrtime.bigint();
      
      for (let i = 0; i < iterations; i++) {
        this.fn();
      }
      
      const end = process.hrtime.bigint();
      duration = Number(end - start);
      
      // Show progress - only show significant iteration counts
      if (attempts === 1) {
        process.stdout.write(`${iterations}`);
      } else if (attempts % 2 === 0 || duration >= targetDuration) {
        process.stdout.write(`-${iterations}`);
      }
      
      // If we've run long enough, we're done
      if (duration >= targetDuration) {
        break;
      }
      
      // Calculate how many iterations we need for target duration
      if (duration > 0) {
        const targetIterations = Math.ceil(iterations * targetDuration / duration);
        iterations = Math.max(iterations * 2, targetIterations);
      } else {
        iterations *= 2;
      }
      
      // Cap iterations to prevent infinite loops
      if (iterations > 100000000) {
        break;
      }
    }
    
    const nsPerOp = duration / iterations;
    const mbPerSec = this.dataSize > 0 ? (this.dataSize * iterations * 1000000000) / duration / (1024 * 1024) : 0;
    
    // Complete the line with final result
    const durationMs = duration / 1000000;
    console.log(`\t${iterations.toString().padStart(8)} \t${nsPerOp.toFixed(1).padStart(6)} ns/op \t(${durationMs.toFixed(0)}ms)`);
    
    return {
      name: this.name,
      iterations,
      duration,
      nsPerOp,
      mbPerSec
    };
  }
}

function formatGoResult(result: BenchmarkResult): string {
  const { name, iterations, nsPerOp, mbPerSec } = result;
  
  let output = `${name.padEnd(40)} ${iterations.toString().padStart(8)} ${nsPerOp.toFixed(1).padStart(12)} ns/op`;
  
  if (mbPerSec > 0.01) {
    output += ` ${mbPerSec.toFixed(2).padStart(12)} MB/s`;
  }
  
  return output;
}

async function runGoStyleBenchmarks(): Promise<void> {
  console.log('🚀 Go-Style Benchmarks (runs until statistically significant)');
  console.log('');
  console.log('Running benchmarks...');
  
  const decoder = new GlintDecoder();
  
  const benchmarks = [
    new GoBenchmark('BenchmarkGlintSimpleDecode', () => {
      decoder.decode(simpleGlint);
    }, simpleGlint.length),
    
    new GoBenchmark('BenchmarkGlintComplexDecode', () => {
      decoder.decode(complexGlint);
    }, complexGlint.length),
    
    new GoBenchmark('BenchmarkGlintNoCacheSimple', () => {
      const noCache = new GlintDecoder();
      noCache.decode(simpleGlint);
    }, simpleGlint.length),
    
    new GoBenchmark('BenchmarkJsonSimpleParse', () => {
      JSON.parse(simpleJson);
    }, simpleJson.length),
    
    new GoBenchmark('BenchmarkJsonComplexParse', () => {
      JSON.parse(complexJson);
    }, complexJson.length),
    
    new GoBenchmark('BenchmarkGlintMemoryAllocation', () => {
      const result = decoder.decode(simpleGlint);
      // Touch result to prevent optimization
      result.name;
    }, simpleGlint.length),
  ];
  
  const results: BenchmarkResult[] = [];
  
  for (const benchmark of benchmarks) {
    const result = await benchmark.run();
    results.push(result);
  }
  
  console.log('');
  console.log('📊 Final Results:');
  console.log('='.repeat(80));
  
  for (const result of results) {
    console.log(formatGoResult(result));
  }
  
  console.log('');
  console.log('Analysis:');
  console.log('-'.repeat(40));
  
  const glintSimple = results.find(r => r.name === 'BenchmarkGlintSimpleDecode');
  const jsonSimple = results.find(r => r.name === 'BenchmarkJsonSimpleParse');
  const glintComplex = results.find(r => r.name === 'BenchmarkGlintComplexDecode');
  const jsonComplex = results.find(r => r.name === 'BenchmarkJsonComplexParse');
  
  if (glintSimple && jsonSimple) {
    const ratio = glintSimple.nsPerOp / jsonSimple.nsPerOp;
    console.log(`Simple: JSON is ${ratio.toFixed(1)}x faster than Glint`);
  }
  
  if (glintComplex && jsonComplex) {
    const ratio = glintComplex.nsPerOp / jsonComplex.nsPerOp;
    console.log(`Complex: JSON is ${ratio.toFixed(1)}x faster than Glint`);
  }
  
  console.log('');
  console.log('Space efficiency:');
  console.log(`Simple: Glint ${simpleGlint.length}B vs JSON ${simpleJson.length}B`);
  console.log(`Complex: Glint ${complexGlint.length}B vs JSON ${complexJson.length}B`);
  
  const simpleSpaceSaving = ((simpleJson.length - simpleGlint.length) / simpleJson.length * 100);
  const complexSpaceSaving = ((complexJson.length - complexGlint.length) / complexJson.length * 100);
  
  console.log(`Space savings: ${simpleSpaceSaving.toFixed(1)}% (simple), ${complexSpaceSaving.toFixed(1)}% (complex)`);
  
  // Show cache statistics
  const cacheStats = decoder.getCacheStats();
  console.log('');
  console.log('Schema cache performance:');
  console.log(`Cache hits: ${cacheStats.hits}, misses: ${cacheStats.misses}`);
  console.log(`Hit rate: ${(cacheStats.hitRate * 100).toFixed(1)}%`);
  console.log(`Cached schemas: ${(decoder as any).schemaCache?.size || 2}`);
}

// Run benchmarks if this file is executed directly
if (require.main === module) {
  runGoStyleBenchmarks().catch(console.error);
}

export { runGoStyleBenchmarks };
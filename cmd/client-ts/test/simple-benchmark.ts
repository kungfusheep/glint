/**
 * Simple, reliable benchmark for Glint TypeScript decoder
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

interface BenchResult {
  name: string;
  iterations: number;
  totalMs: number;
  nsPerOp: number;
  opsPerSec: number;
  dataSize: number;
  mbPerSec: number;
}

function benchmark(name: string, iterations: number, dataSize: number, fn: () => void): BenchResult {
  console.log(`Running ${name} with ${iterations} iterations...`);
  
  // Warmup
  const warmupRuns = Math.min(iterations / 10, 1000);
  for (let i = 0; i < warmupRuns; i++) {
    fn();
  }
  
  // Force GC
  if (global.gc) {
    global.gc();
  }
  
  // Run benchmark
  const start = process.hrtime.bigint();
  
  for (let i = 0; i < iterations; i++) {
    fn();
    
    // Show progress every 10k iterations
    if (i > 0 && i % 10000 === 0) {
      process.stdout.write(`${i}...`);
    }
  }
  
  const end = process.hrtime.bigint();
  const totalNs = Number(end - start);
  const totalMs = totalNs / 1000000;
  const nsPerOp = totalNs / iterations;
  const opsPerSec = 1000000000 / nsPerOp;
  const mbPerSec = (dataSize * opsPerSec) / (1024 * 1024);
  
  console.log(` done in ${totalMs.toFixed(2)}ms`);
  
  return {
    name,
    iterations,
    totalMs,
    nsPerOp,
    opsPerSec,
    dataSize,
    mbPerSec
  };
}

function formatResult(result: BenchResult): string {
  const { name, iterations, nsPerOp, opsPerSec, dataSize, mbPerSec } = result;
  
  return `${name.padEnd(35)} ${iterations.toString().padStart(8)} ` +
         `${Math.round(nsPerOp).toString().padStart(8)} ns/op ` +
         `${Math.round(opsPerSec).toString().padStart(10)} ops/sec ` +
         `${dataSize.toString().padStart(6)} bytes ` +
         `${mbPerSec.toFixed(2).padStart(8)} MB/s`;
}

async function runSimpleBenchmarks(customIterations?: number): Promise<void> {
  console.log('🚀 Glint TypeScript Simple Benchmarks');
  console.log('');
  
  const decoder = new GlintDecoder();
  const iterations = customIterations || 50000;
  
  console.log(`Running benchmarks with ${iterations} iterations each...`);
  console.log('');
  
  const results: BenchResult[] = [];
  
  // Glint benchmarks
  results.push(benchmark('Glint Simple Decode', iterations, simpleGlint.length, () => {
    decoder.decode(simpleGlint);
  }));
  
  results.push(benchmark('Glint Complex Decode', iterations, complexGlint.length, () => {
    decoder.decode(complexGlint);
  }));
  
  // JSON benchmarks
  results.push(benchmark('JSON Simple Parse', iterations, simpleJson.length, () => {
    JSON.parse(simpleJson);
  }));
  
  results.push(benchmark('JSON Complex Parse', iterations, complexJson.length, () => {
    JSON.parse(complexJson);
  }));
  
  // Memory allocation test
  results.push(benchmark('Glint Memory Allocation', iterations / 10, simpleGlint.length, () => {
    const result = decoder.decode(simpleGlint);
    // Touch the result to prevent optimization
    result.name;
  }));
  
  console.log('Results:');
  console.log('='.repeat(90));
  console.log('Name'.padEnd(35), 'Iterations'.padStart(8), 'ns/op'.padStart(8), 'ops/sec'.padStart(10), 'bytes'.padStart(6), 'MB/s'.padStart(8));
  console.log('-'.repeat(90));
  
  for (const result of results) {
    console.log(formatResult(result));
  }
  
  console.log('');
  console.log('📊 Analysis:');
  console.log('');
  
  const glintSimple = results.find(r => r.name === 'Glint Simple Decode');
  const jsonSimple = results.find(r => r.name === 'JSON Simple Parse');
  const glintComplex = results.find(r => r.name === 'Glint Complex Decode');
  const jsonComplex = results.find(r => r.name === 'JSON Complex Parse');
  
  if (glintSimple && jsonSimple) {
    const speedRatio = glintSimple.opsPerSec / jsonSimple.opsPerSec;
    console.log(`Simple data: Glint is ${speedRatio.toFixed(2)}x the speed of JSON`);
  }
  
  if (glintComplex && jsonComplex) {
    const speedRatio = glintComplex.opsPerSec / jsonComplex.opsPerSec;
    console.log(`Complex data: Glint is ${speedRatio.toFixed(2)}x the speed of JSON`);
  }
  
  console.log('');
  console.log('Size comparison:');
  console.log(`Simple: Glint ${simpleGlint.length}B vs JSON ${simpleJson.length}B (${((simpleJson.length - simpleGlint.length) / simpleJson.length * 100).toFixed(1)}% savings)`);
  console.log(`Complex: Glint ${complexGlint.length}B vs JSON ${complexJson.length}B (${((complexJson.length - complexGlint.length) / complexJson.length * 100).toFixed(1)}% savings)`);
}

// Run benchmarks if this file is executed directly
if (require.main === module) {
  // Parse command line arguments
  const args = process.argv.slice(2);
  let iterations: number | undefined;
  
  for (let i = 0; i < args.length; i++) {
    if (args[i] === '--iterations' || args[i] === '-n') {
      iterations = parseInt(args[i + 1], 10);
      if (isNaN(iterations) || iterations <= 0) {
        console.error('Invalid iterations value:', args[i + 1]);
        process.exit(1);
      }
      break;
    }
  }
  
  if (args.includes('--help') || args.includes('-h')) {
    console.log('Usage: node simple-benchmark.js [--iterations|-n <number>]');
    console.log('');
    console.log('Options:');
    console.log('  --iterations, -n  Number of iterations to run (default: 50000)');
    console.log('  --help, -h        Show this help message');
    process.exit(0);
  }
  
  if (iterations) {
    console.log(`Running with ${iterations} iterations per benchmark`);
  }
  
  runSimpleBenchmarks(iterations).catch(console.error);
}

export { runSimpleBenchmarks };
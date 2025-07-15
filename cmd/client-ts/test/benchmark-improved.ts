/**
 * Improved benchmark suite using Go-style benchmark runner
 * Provides statistically significant results with adaptive iteration counts
 */

import * as fs from 'fs';
import * as path from 'path';
import { GlintDecoder } from '../src/decoder';
import { runBenchmarks, BenchmarkFunction } from './benchmark-runner';

// Load test data
const testDir = path.join(__dirname, '..', '..', 'test');
const simpleGlintData = new Uint8Array(fs.readFileSync(path.join(testDir, 'simple.glint')));
const complexGlintData = new Uint8Array(fs.readFileSync(path.join(testDir, 'complex.glint')));

// JSON equivalents
const simpleJsonData = JSON.stringify({name: "Alice", age: 30});
const complexJsonData = JSON.stringify({name: "Bob", age: 25, active: true, tags: ["developer", "go"]});

// Larger test data
const largeData = generateLargeTestData();
const largeJsonData = JSON.stringify(largeData);

function generateLargeTestData() {
  return {
    users: Array.from({ length: 1000 }, (_, i) => ({
      id: i,
      name: `User${i}`,
      email: `user${i}@example.com`,
      active: i % 2 === 0,
      score: Math.random() * 100,
      metadata: {
        created: new Date().toISOString(),
        lastLogin: new Date().toISOString(),
        preferences: {
          theme: i % 2 === 0 ? 'dark' : 'light',
          notifications: i % 3 === 0
        }
      }
    })),
    summary: {
      total: 1000,
      active: 500,
      avgScore: 50
    }
  };
}

// Create decoder instances (reused across benchmarks)
const decoder = new GlintDecoder();
const decoderWithLimits = new GlintDecoder({
  maxStringLength: 1024 * 1024,
  maxArrayLength: 100000
});

// Benchmark functions
const benchmarkGlintSimpleDecode: BenchmarkFunction = (b) => {
  b.resetTimer();
  for (let i = 0; i < b.N; i++) {
    decoder.decode(simpleGlintData);
  }
  b.setBytes(simpleGlintData.length);
};

const benchmarkGlintComplexDecode: BenchmarkFunction = (b) => {
  b.resetTimer();
  for (let i = 0; i < b.N; i++) {
    decoder.decode(complexGlintData);
  }
  b.setBytes(complexGlintData.length);
};

const benchmarkJsonSimpleParse: BenchmarkFunction = (b) => {
  b.resetTimer();
  for (let i = 0; i < b.N; i++) {
    JSON.parse(simpleJsonData);
  }
  b.setBytes(simpleJsonData.length);
};

const benchmarkJsonComplexParse: BenchmarkFunction = (b) => {
  b.resetTimer();
  for (let i = 0; i < b.N; i++) {
    JSON.parse(complexJsonData);
  }
  b.setBytes(complexJsonData.length);
};

const benchmarkJsonLargeParse: BenchmarkFunction = (b) => {
  b.resetTimer();
  for (let i = 0; i < b.N; i++) {
    JSON.parse(largeJsonData);
  }
  b.setBytes(largeJsonData.length);
};

// Memory allocation benchmark
const benchmarkGlintMemoryAllocation: BenchmarkFunction = (b) => {
  const results: any[] = [];
  b.resetTimer();
  for (let i = 0; i < b.N; i++) {
    results.push(decoder.decode(simpleGlintData));
  }
  // Keep results in scope to prevent GC
  if (results.length > 0) {
    // Touch the data to prevent optimization
    results[0].name;
  }
};

// Decoder with security limits
const benchmarkGlintWithLimits: BenchmarkFunction = (b) => {
  b.resetTimer();
  for (let i = 0; i < b.N; i++) {
    decoderWithLimits.decode(complexGlintData);
  }
  b.setBytes(complexGlintData.length);
};

// Repeated decode with same decoder instance
const benchmarkGlintReusedDecoder: BenchmarkFunction = (b) => {
  b.resetTimer();
  for (let i = 0; i < b.N; i++) {
    decoder.decode(simpleGlintData);
  }
  b.setBytes(simpleGlintData.length);
};

// Stress test with multiple data types
const benchmarkGlintStressTest: BenchmarkFunction = (b) => {
  const dataTypes = [simpleGlintData, complexGlintData];
  let dataIndex = 0;
  
  b.resetTimer();
  for (let i = 0; i < b.N; i++) {
    decoder.decode(dataTypes[dataIndex]);
    dataIndex = (dataIndex + 1) % dataTypes.length;
  }
  b.setBytes((simpleGlintData.length + complexGlintData.length) / 2);
};

async function runImprovedBenchmarks(): Promise<void> {
  console.log('🚀 Glint TypeScript Benchmarks (Go-style Statistical Analysis)');
  console.log('');
  console.log('Test Data Sizes:');
  console.log(`Simple Glint: ${simpleGlintData.length} bytes`);
  console.log(`Complex Glint: ${complexGlintData.length} bytes`);
  console.log(`Simple JSON: ${simpleJsonData.length} bytes`);
  console.log(`Complex JSON: ${complexJsonData.length} bytes`);
  console.log(`Large JSON: ${largeJsonData.length} bytes`);
  console.log('');

  await runBenchmarks([
    { name: 'BenchmarkGlintSimpleDecode', fn: benchmarkGlintSimpleDecode },
    { name: 'BenchmarkGlintComplexDecode', fn: benchmarkGlintComplexDecode },
    { name: 'BenchmarkJsonSimpleParse', fn: benchmarkJsonSimpleParse },
    { name: 'BenchmarkJsonComplexParse', fn: benchmarkJsonComplexParse },
    { name: 'BenchmarkJsonLargeParse', fn: benchmarkJsonLargeParse },
    { name: 'BenchmarkGlintWithLimits', fn: benchmarkGlintWithLimits },
    { name: 'BenchmarkGlintReusedDecoder', fn: benchmarkGlintReusedDecoder },
    { name: 'BenchmarkGlintMemoryAllocation', fn: benchmarkGlintMemoryAllocation },
    { name: 'BenchmarkGlintStressTest', fn: benchmarkGlintStressTest },
  ]);

  // Performance analysis
  console.log('📊 Performance Analysis:');
  console.log('');
  console.log('Space Efficiency:');
  const simpleCompression = ((simpleJsonData.length - simpleGlintData.length) / simpleJsonData.length * 100);
  const complexCompression = ((complexJsonData.length - complexGlintData.length) / complexJsonData.length * 100);
  console.log(`Simple data: Glint is ${simpleCompression.toFixed(1)}% more compact than JSON`);
  console.log(`Complex data: Glint is ${complexCompression.toFixed(1)}% more compact than JSON`);
  console.log('');
  
  console.log('Notes:');
  console.log('- Benchmarks use adaptive iteration counts (like Go benchmarks)');
  console.log('- Each benchmark runs until statistically significant (≥1 second)');
  console.log('- Memory measurements include GC overhead');
  console.log('- Results are comparable to Go benchmark output format');
}

// Run benchmarks if this file is executed directly
if (require.main === module) {
  runImprovedBenchmarks().catch(console.error);
}

export { runImprovedBenchmarks };
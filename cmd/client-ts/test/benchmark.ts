/**
 * Benchmark suite for Glint TypeScript decoder
 * Measures performance against JSON and tests different data sizes
 */

import * as fs from 'fs';
import * as path from 'path';
import { GlintDecoder } from '../src/decoder';

interface BenchmarkResult {
  name: string;
  operations: number;
  totalTime: number;
  avgTime: number;
  opsPerSecond: number;
  memoryUsed?: number;
}

class Benchmark {
  private results: BenchmarkResult[] = [];

  async run(name: string, iterations: number, fn: () => void): Promise<BenchmarkResult> {
    // Warm up
    for (let i = 0; i < Math.min(iterations / 10, 100); i++) {
      fn();
    }

    // Force garbage collection if available
    if (global.gc) {
      global.gc();
    }

    const startMemory = process.memoryUsage().heapUsed;
    const startTime = process.hrtime.bigint();

    for (let i = 0; i < iterations; i++) {
      fn();
    }

    const endTime = process.hrtime.bigint();
    const endMemory = process.memoryUsage().heapUsed;

    const totalTime = Number(endTime - startTime) / 1000000; // Convert to milliseconds
    const avgTime = totalTime / iterations;
    const opsPerSecond = 1000 / avgTime;
    const memoryUsed = endMemory - startMemory;

    const result: BenchmarkResult = {
      name,
      operations: iterations,
      totalTime,
      avgTime,
      opsPerSecond,
      memoryUsed
    };

    this.results.push(result);
    return result;
  }

  printResults(): void {
    console.log('\n📊 Benchmark Results');
    console.log('═'.repeat(80));
    console.log('Name'.padEnd(30), 'Ops'.padEnd(10), 'Total(ms)'.padEnd(12), 'Avg(ms)'.padEnd(12), 'Ops/sec'.padEnd(12));
    console.log('─'.repeat(80));

    for (const result of this.results) {
      console.log(
        result.name.padEnd(30),
        result.operations.toString().padEnd(10),
        result.totalTime.toFixed(2).padEnd(12),
        result.avgTime.toFixed(4).padEnd(12),
        Math.round(result.opsPerSecond).toString().padEnd(12)
      );
    }

    console.log('─'.repeat(80));
    console.log();
  }

  getResult(name: string): BenchmarkResult | undefined {
    return this.results.find(r => r.name === name);
  }
}

// Test data generators
function generateTestData(size: 'small' | 'medium' | 'large'): any {
  const base = {
    id: Math.floor(Math.random() * 1000000),
    name: `User${Math.floor(Math.random() * 1000)}`,
    email: `user${Math.floor(Math.random() * 1000)}@example.com`,
    active: Math.random() > 0.5,
    score: Math.random() * 100,
    tags: ['user', 'active', 'verified']
  };

  switch (size) {
    case 'small':
      return base;
    
    case 'medium':
      return {
        ...base,
        profile: {
          firstName: 'John',
          lastName: 'Doe',
          age: Math.floor(Math.random() * 80) + 18,
          preferences: {
            theme: 'dark',
            notifications: true,
            language: 'en'
          }
        },
        friends: Array.from({ length: 10 }, (_, i) => ({
          id: i,
          name: `Friend${i}`,
          mutual: Math.random() > 0.5
        }))
      };
    
    case 'large':
      return {
        ...base,
        profile: {
          firstName: 'John',
          lastName: 'Doe',
          age: Math.floor(Math.random() * 80) + 18,
          bio: 'A'.repeat(500), // Large string
          preferences: {
            theme: 'dark',
            notifications: true,
            language: 'en'
          }
        },
        friends: Array.from({ length: 100 }, (_, i) => ({
          id: i,
          name: `Friend${i}`,
          mutual: Math.random() > 0.5
        })),
        posts: Array.from({ length: 50 }, (_, i) => ({
          id: i,
          title: `Post ${i}`,
          content: 'Lorem ipsum '.repeat(50),
          likes: Math.floor(Math.random() * 1000),
          comments: Array.from({ length: 5 }, (_, j) => ({
            id: j,
            text: `Comment ${j}`,
            author: `User${j}`
          }))
        }))
      };
  }
}

async function runBenchmarks(): Promise<void> {
  console.log('🚀 Starting Glint TypeScript Decoder Benchmarks\n');
  
  const benchmark = new Benchmark();
  const decoder = new GlintDecoder();

  // Load test files
  const testDir = path.join(__dirname, '..', '..', 'test');
  const simpleGlint = fs.readFileSync(path.join(testDir, 'simple.glint'));
  const complexGlint = fs.readFileSync(path.join(testDir, 'complex.glint'));
  
  const simpleData = new Uint8Array(simpleGlint);
  const complexData = new Uint8Array(complexGlint);

  // JSON equivalents for comparison
  const simpleJson = JSON.stringify({name: "Alice", age: 30});
  const complexJson = JSON.stringify({name: "Bob", age: 25, active: true, tags: ["developer", "go"]});

  console.log('Testing with real Glint documents...');

  // Benchmark real Glint data
  await benchmark.run('Glint Simple Decode', 10000, () => {
    decoder.decode(simpleData);
  });

  await benchmark.run('Glint Complex Decode', 10000, () => {
    decoder.decode(complexData);
  });

  // Benchmark JSON equivalents
  await benchmark.run('JSON Simple Parse', 10000, () => {
    JSON.parse(simpleJson);
  });

  await benchmark.run('JSON Complex Parse', 10000, () => {
    JSON.parse(complexJson);
  });

  console.log('\nTesting with synthetic data...');

  // Generate synthetic test data
  const smallData = generateTestData('small');
  const mediumData = generateTestData('medium');
  const largeData = generateTestData('large');

  const smallJson = JSON.stringify(smallData);
  const mediumJson = JSON.stringify(mediumData);
  const largeJson = JSON.stringify(largeData);

  // Benchmark JSON parsing on synthetic data
  await benchmark.run('JSON Small Parse', 5000, () => {
    JSON.parse(smallJson);
  });

  await benchmark.run('JSON Medium Parse', 2000, () => {
    JSON.parse(mediumJson);
  });

  await benchmark.run('JSON Large Parse', 500, () => {
    JSON.parse(largeJson);
  });

  console.log('\nTesting memory efficiency...');

  // Memory usage test
  const memoryTest = () => {
    const results = [];
    for (let i = 0; i < 1000; i++) {
      results.push(decoder.decode(simpleData));
    }
    return results;
  };

  const startMem = process.memoryUsage().heapUsed;
  const results = memoryTest();
  const endMem = process.memoryUsage().heapUsed;
  const memoryPerOp = (endMem - startMem) / 1000;

  console.log(`Memory per decode operation: ${memoryPerOp.toFixed(2)} bytes`);
  console.log(`Total objects created: ${results.length}`);

  // Print all results
  benchmark.printResults();

  // Performance analysis
  console.log('📈 Performance Analysis');
  console.log('═'.repeat(50));

  const glintSimple = benchmark.getResult('Glint Simple Decode');
  const jsonSimple = benchmark.getResult('JSON Simple Parse');
  const glintComplex = benchmark.getResult('Glint Complex Decode');
  const jsonComplex = benchmark.getResult('JSON Complex Parse');

  if (glintSimple && jsonSimple) {
    const ratio = jsonSimple.avgTime / glintSimple.avgTime;
    console.log(`Simple data: Glint is ${ratio.toFixed(2)}x ${ratio > 1 ? 'faster' : 'slower'} than JSON`);
  }

  if (glintComplex && jsonComplex) {
    const ratio = jsonComplex.avgTime / glintComplex.avgTime;
    console.log(`Complex data: Glint is ${ratio.toFixed(2)}x ${ratio > 1 ? 'faster' : 'slower'} than JSON`);
  }

  console.log(`\nData sizes:`);
  console.log(`Simple Glint: ${simpleData.length} bytes`);
  console.log(`Simple JSON: ${simpleJson.length} bytes`);
  console.log(`Complex Glint: ${complexData.length} bytes`);
  console.log(`Complex JSON: ${complexJson.length} bytes`);

  const simpleCompression = ((simpleJson.length - simpleData.length) / simpleJson.length * 100);
  const complexCompression = ((complexJson.length - complexData.length) / complexJson.length * 100);
  
  console.log(`\nCompression: Glint is ${simpleCompression.toFixed(1)}% smaller (simple), ${complexCompression.toFixed(1)}% smaller (complex)`);
}

// Run benchmarks if this file is executed directly
if (require.main === module) {
  runBenchmarks().catch(console.error);
}

export { runBenchmarks, Benchmark };
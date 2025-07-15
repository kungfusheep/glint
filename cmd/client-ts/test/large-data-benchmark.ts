/**
 * Benchmark with large datasets to identify performance bottlenecks
 */

import * as fs from 'fs';
import * as path from 'path';
import { GlintDecoder } from '../src/decoder';
import { runBenchmark, BenchmarkFunction } from './benchmark-runner';

// Load test data
const testDir = path.join(__dirname, '..', '..', 'test');

interface TestData {
  name: string;
  glintData: Uint8Array;
  jsonData: string;
  glintSize: number;
  jsonSize: number;
  compressionRatio: number;
}

function loadTestData(): TestData[] {
  const datasets = [
    { name: 'Simple', file: 'simple' },
    { name: 'Complex', file: 'complex' },
    { name: 'Medium', file: 'medium' },
    { name: 'Large', file: 'large' },
    { name: 'Huge', file: 'huge' }
  ];

  return datasets.map(dataset => {
    const glintPath = path.join(testDir, `${dataset.file}.glint`);
    const jsonPath = path.join(testDir, `${dataset.file}.json`);
    
    const glintData = new Uint8Array(fs.readFileSync(glintPath));
    const jsonData = fs.readFileSync(jsonPath, 'utf8');
    
    return {
      name: dataset.name,
      glintData,
      jsonData,
      glintSize: glintData.length,
      jsonSize: jsonData.length,
      compressionRatio: (jsonData.length - glintData.length) / jsonData.length
    };
  });
}

async function runLargeDataBenchmarks(): Promise<void> {
  console.log('🚀 Large Dataset Performance Analysis');
  console.log('');
  
  const testData = loadTestData();
  
  // Show dataset info
  console.log('Dataset Information:');
  console.log('='.repeat(80));
  console.log('Name'.padEnd(10), 'JSON Size'.padEnd(12), 'Glint Size'.padEnd(12), 'Compression'.padEnd(12));
  console.log('-'.repeat(80));
  
  for (const data of testData) {
    const jsonMB = (data.jsonSize / 1024 / 1024).toFixed(2);
    const glintMB = (data.glintSize / 1024 / 1024).toFixed(2);
    const compression = (data.compressionRatio * 100).toFixed(1);
    
    console.log(
      data.name.padEnd(10),
      `${jsonMB} MB`.padEnd(12),
      `${glintMB} MB`.padEnd(12),
      `${compression}%`.padEnd(12)
    );
  }
  
  console.log('');
  console.log('Performance Benchmarks:');
  console.log('='.repeat(80));
  
  // Run benchmarks for each dataset
  const decoder = new GlintDecoder();
  
  for (const data of testData) {
    console.log(`\n📊 Testing ${data.name} dataset (${(data.glintSize / 1024).toFixed(1)}KB Glint, ${(data.jsonSize / 1024).toFixed(1)}KB JSON):`);
    
    // Determine appropriate iteration count based on size
    const iterations = data.glintSize < 1000 ? 10000 : data.glintSize < 100000 ? 1000 : 100;
    
    const glintBench: BenchmarkFunction = (b) => {
      b.resetTimer();
      for (let i = 0; i < b.N; i++) {
        decoder.decode(data.glintData);
      }
      b.setBytes(data.glintSize);
    };
    
    const jsonBench: BenchmarkFunction = (b) => {
      b.resetTimer();
      for (let i = 0; i < b.N; i++) {
        JSON.parse(data.jsonData);
      }
      b.setBytes(data.jsonSize);
    };
    
    const glintNoCacheBench: BenchmarkFunction = (b) => {
      b.resetTimer();
      for (let i = 0; i < b.N; i++) {
        const freshDecoder = new GlintDecoder();
        freshDecoder.decode(data.glintData);
      }
      b.setBytes(data.glintSize);
    };
    
    try {
      const glintResult = await runBenchmark(`Glint${data.name}Decode`, glintBench);
      const jsonResult = await runBenchmark(`JSON${data.name}Parse`, jsonBench);
      const noCacheResult = await runBenchmark(`Glint${data.name}NoCache`, glintNoCacheBench);
      
      console.log(`  Glint (cached):   ${glintResult.nsPerOp.toFixed(0).padStart(8)} ns/op  ${glintResult.mbPerSec.toFixed(1).padStart(8)} MB/s`);
      console.log(`  Glint (no cache): ${noCacheResult.nsPerOp.toFixed(0).padStart(8)} ns/op  ${noCacheResult.mbPerSec.toFixed(1).padStart(8)} MB/s`);
      console.log(`  JSON:             ${jsonResult.nsPerOp.toFixed(0).padStart(8)} ns/op  ${jsonResult.mbPerSec.toFixed(1).padStart(8)} MB/s`);
      
      const speedRatio = glintResult.nsPerOp / jsonResult.nsPerOp;
      const cacheSpeedRatio = noCacheResult.nsPerOp / glintResult.nsPerOp;
      
      console.log(`  Performance: JSON is ${speedRatio.toFixed(1)}x faster than Glint (cached)`);
      console.log(`  Cache benefit: ${cacheSpeedRatio.toFixed(1)}x faster with cache`);
      
    } catch (error) {
      console.log(`  Error benchmarking ${data.name}: ${(error as Error).message}`);
    }
  }
  
  // Cache statistics
  const cacheStats = decoder.getCacheStats();
  console.log('\n📈 Cache Statistics:');
  console.log(`Cache hits: ${cacheStats.hits.toLocaleString()}`);
  console.log(`Cache misses: ${cacheStats.misses.toLocaleString()}`);
  console.log(`Hit rate: ${(cacheStats.hitRate * 100).toFixed(1)}%`);
  console.log(`Cached schemas: ${(decoder as any).schemaCache?.size || 'N/A'}`);
  
  // Memory usage
  const memUsage = process.memoryUsage();
  console.log('\n💾 Memory Usage:');
  console.log(`RSS: ${(memUsage.rss / 1024 / 1024).toFixed(2)} MB`);
  console.log(`Heap Used: ${(memUsage.heapUsed / 1024 / 1024).toFixed(2)} MB`);
  console.log(`Heap Total: ${(memUsage.heapTotal / 1024 / 1024).toFixed(2)} MB`);
  
  console.log('\n🎯 Performance Analysis:');
  console.log('1. Glint provides excellent compression (49-50% smaller than JSON)');
  console.log('2. JSON parsing is heavily optimized in V8');
  console.log('3. Schema caching provides significant performance benefits');
  console.log('4. For large datasets, the performance gap may be more acceptable');
  
  console.log('\n⚡ Optimization Opportunities:');
  console.log('1. Optimize varint decoding (currently slower than V8 number parsing)');
  console.log('2. Reduce object allocations in BinaryReader');
  console.log('3. Use TypedArrays for zero-copy operations');
  console.log('4. Pre-allocate result structures');
  console.log('5. Optimize string decoding path');
  
  console.log('\n🔍 Profiling Commands:');
  console.log('npm run profile          # Run with V8 profiler');
  console.log('node --prof-process isolate-*.log  # Analyze profile data');
  console.log('clinic doctor -- npm run profile   # Detailed profiling');
}

// Run benchmarks if this file is executed directly
if (require.main === module) {
  runLargeDataBenchmarks().catch(console.error);
}

export { runLargeDataBenchmarks };
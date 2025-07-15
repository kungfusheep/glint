/**
 * Debug profiling to identify specific bottlenecks and crashes
 */

import * as fs from 'fs';
import * as path from 'path';
import { GlintDecoder } from '../src/decoder';

// Simple profiling with detailed error reporting
class DebugProfiler {
  private timings: Map<string, number[]> = new Map();
  
  time<T>(name: string, fn: () => T): T {
    const start = process.hrtime.bigint();
    const result = fn();
    const end = process.hrtime.bigint();
    const duration = Number(end - start);
    
    if (!this.timings.has(name)) {
      this.timings.set(name, []);
    }
    this.timings.get(name)!.push(duration);
    
    return result;
  }
  
  report(): void {
    console.log('\n📊 Detailed Timing Report:');
    console.log('='.repeat(60));
    
    for (const [name, times] of this.timings) {
      if (times.length === 0) continue;
      
      const avg = times.reduce((a, b) => a + b, 0) / times.length;
      const min = Math.min(...times);
      const max = Math.max(...times);
      
      console.log(`${name.padEnd(25)} ${times.length.toString().padStart(6)} calls`);
      console.log(`${''.padEnd(25)} ${(avg / 1000000).toFixed(2).padStart(6)} ms avg`);
      console.log(`${''.padEnd(25)} ${(min / 1000000).toFixed(2).padStart(6)} ms min`);
      console.log(`${''.padEnd(25)} ${(max / 1000000).toFixed(2).padStart(6)} ms max`);
      console.log('');
    }
  }
}

async function debugProfile(): Promise<void> {
  console.log('🔍 Debug Profiling - Finding Bottlenecks & Crashes');
  console.log('');
  
  const profiler = new DebugProfiler();
  const testDir = path.join(__dirname, '..', '..', 'test');
  
  // Test each dataset systematically
  const datasets = [
    { name: 'Simple', file: 'simple' },
    { name: 'Complex', file: 'complex' },
    { name: 'Medium', file: 'medium' },
    { name: 'Large', file: 'large' },
    { name: 'Huge', file: 'huge' }
  ];
  
  for (const dataset of datasets) {
    console.log(`\n🧪 Testing ${dataset.name} dataset...`);
    
    try {
      const glintPath = path.join(testDir, `${dataset.file}.glint`);
      const jsonPath = path.join(testDir, `${dataset.file}.json`);
      
      if (!fs.existsSync(glintPath)) {
        console.log(`  ❌ Glint file not found: ${glintPath}`);
        continue;
      }
      
      if (!fs.existsSync(jsonPath)) {
        console.log(`  ❌ JSON file not found: ${jsonPath}`);
        continue;
      }
      
      const glintData = new Uint8Array(fs.readFileSync(glintPath));
      const jsonData = fs.readFileSync(jsonPath, 'utf8');
      
      console.log(`  📄 Glint: ${glintData.length} bytes, JSON: ${jsonData.length} bytes`);
      
      // Test JSON parsing first (baseline)
      let jsonResult: any;
      const jsonTime = profiler.time(`JSON-${dataset.name}`, () => {
        jsonResult = JSON.parse(jsonData);
      });
      
      console.log(`  ✅ JSON parse: ${(Number(jsonTime) / 1000000).toFixed(2)}ms`);
      
      // Test Glint decoding with detailed error reporting
      const decoder = new GlintDecoder({
        maxStringLength: 50 * 1024 * 1024,  // 50MB
        maxArrayLength: 100000,            // 100K elements
        maxMapSize: 50000,                 // 50K map entries
        maxNestingDepth: 200              // Deep nesting
      });
      
      let glintResult: any;
      const glintTime = profiler.time(`Glint-${dataset.name}`, () => {
        glintResult = decoder.decode(glintData);
      });
      
      console.log(`  ✅ Glint decode: ${(Number(glintTime) / 1000000).toFixed(2)}ms`);
      
      // Compare results structure
      const glintKeys = Object.keys(glintResult);
      const jsonKeys = Object.keys(jsonResult);
      
      console.log(`  🔍 Keys: JSON=${jsonKeys.length}, Glint=${glintKeys.length}`);
      
      if (glintKeys.length !== jsonKeys.length) {
        console.log(`  ⚠️  Key count mismatch!`);
        console.log(`    JSON keys: ${jsonKeys.slice(0, 5).join(', ')}...`);
        console.log(`    Glint keys: ${glintKeys.slice(0, 5).join(', ')}...`);
      }
      
      // Performance comparison
      const speedRatio = Number(glintTime) / Number(jsonTime);
      console.log(`  📊 Performance: JSON is ${speedRatio.toFixed(1)}x faster`);
      
      // Cache stats
      const cacheStats = decoder.getCacheStats();
      console.log(`  💾 Cache: ${cacheStats.hits} hits, ${cacheStats.misses} misses`);
      
    } catch (error) {
      console.log(`  ❌ Error processing ${dataset.name}:`);
      console.log(`     ${(error as Error).message}`);
      console.log(`     Stack: ${(error as Error).stack?.split('\n')[1]?.trim()}`);
      
      // Try to get more details about the error
      if (error instanceof Error) {
        if (error.message.includes('Unexpected end of data')) {
          console.log(`  🔍 Debugging "Unexpected end of data" error...`);
          
          try {
            const glintPath = path.join(testDir, `${dataset.file}.glint`);
            const glintData = new Uint8Array(fs.readFileSync(glintPath));
            
            console.log(`     Data length: ${glintData.length}`);
            console.log(`     First 20 bytes: ${Array.from(glintData.slice(0, 20)).map(b => b.toString(16).padStart(2, '0')).join(' ')}`);
            
            // Try to read header manually
            if (glintData.length >= 5) {
              const flags = glintData[0];
              const crc = new DataView(glintData.buffer, glintData.byteOffset + 1, 4).getUint32(0, true);
              console.log(`     Header - Flags: ${flags}, CRC: ${crc.toString(16)}`);
            }
            
          } catch (debugError) {
            console.log(`     Debug error: ${(debugError as Error).message}`);
          }
        }
      }
    }
  }
  
  profiler.report();
  
  console.log('\n🎯 Analysis Summary:');
  console.log('1. Identify which datasets cause crashes');
  console.log('2. Compare timing patterns between JSON and Glint');
  console.log('3. Focus optimization efforts on the slowest operations');
  console.log('4. Fix correctness issues before performance optimization');
}

// Run profiling if this file is executed directly
if (require.main === module) {
  debugProfile().catch(console.error);
}

export { debugProfile };
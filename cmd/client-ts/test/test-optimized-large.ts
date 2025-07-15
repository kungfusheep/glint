/**
 * Test optimized decoder with large datasets
 */

import * as fs from 'fs';
import * as path from 'path';
import { GlintDecoderOptimized } from '../src/decoder-optimized';

function testLargeDatasets(): void {
  console.log('🧪 Testing optimized decoder with large datasets\n');
  
  const testDir = path.join(__dirname, '..', '..', 'test');
  const decoder = new GlintDecoderOptimized();
  
  const datasets = ['simple', 'complex', 'medium', 'large', 'huge'];
  
  for (const dataset of datasets) {
    const glintPath = path.join(testDir, `${dataset}.glint`);
    const jsonPath = path.join(testDir, `${dataset}.json`);
    
    if (!fs.existsSync(glintPath)) {
      console.log(`❌ ${dataset}: File not found`);
      continue;
    }
    
    const glintData = new Uint8Array(fs.readFileSync(glintPath));
    
    try {
      console.log(`📊 Testing ${dataset} dataset (${glintData.length} bytes)...`);
      
      const start = process.hrtime.bigint();
      const result = decoder.decode(glintData);
      const end = process.hrtime.bigint();
      
      const timeMs = Number(end - start) / 1000000;
      
      // Count keys in result
      const keyCount = Object.keys(result).length;
      
      console.log(`  ✅ Success! Decoded in ${timeMs.toFixed(2)}ms`);
      console.log(`  📝 Top-level keys: ${keyCount}`);
      console.log(`  🔑 Keys: ${Object.keys(result).slice(0, 5).join(', ')}${keyCount > 5 ? '...' : ''}`);
      
      // Verify against JSON if available
      if (fs.existsSync(jsonPath)) {
        const jsonData = JSON.parse(fs.readFileSync(jsonPath, 'utf8'));
        const jsonKeys = Object.keys(jsonData).length;
        if (keyCount === jsonKeys) {
          console.log(`  ✓ Key count matches JSON (${jsonKeys})`);
        } else {
          console.log(`  ⚠️  Key count mismatch: Glint=${keyCount}, JSON=${jsonKeys}`);
        }
      }
      
    } catch (error) {
      console.log(`  ❌ Failed: ${(error as Error).message}`);
      console.log(`  📍 Stack: ${(error as Error).stack?.split('\n')[1]?.trim()}`);
    }
    
    console.log('');
  }
  
  // Cache stats
  const stats = decoder.getCacheStats();
  console.log('📊 Cache Statistics:');
  console.log(`Hits: ${stats.hits}, Misses: ${stats.misses}, Hit rate: ${(stats.hitRate * 100).toFixed(1)}%`);
}

// Run test
testLargeDatasets();
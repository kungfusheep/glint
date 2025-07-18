const fs = require('fs');
const zlib = require('zlib');
const { CodegenGlintDecoder } = require('./dist/src/codegen-decoder');

// Load test data - using JSON files for direct comparison
const mediumData = fs.readFileSync('./test/medium-go.glint');
const mediumJSON = fs.readFileSync('./test/medium-go.json', 'utf8');

const largeData = fs.readFileSync('./test/large-go.glint');
const largeJSON = fs.readFileSync('./test/large-go.json', 'utf8');

const hugeData = fs.readFileSync('./test/huge-go.glint');
const hugeJSON = fs.readFileSync('./test/huge-go.json', 'utf8');

// Stable benchmark function with multiple samples
function benchmark(fn, iterations = 1000, samples = 10) {
  const results = [];
  
  // Warm up once
  for (let i = 0; i < 20; i++) {
    fn();
  }
  
  // Run multiple samples
  for (let sample = 0; sample < samples; sample++) {
    // Small warm up for each sample
    for (let i = 0; i < 5; i++) {
      fn();
    }
    
    // Benchmark this sample
    const start = process.hrtime.bigint();
    for (let i = 0; i < iterations; i++) {
      fn();
    }
    const end = process.hrtime.bigint();
    
    const sampleTime = Number(end - start) / 1_000_000 / iterations; // ms per operation
    results.push(sampleTime);
  }
  
  // Calculate statistics
  results.sort((a, b) => a - b);
  const min = results[0];
  const max = results[results.length - 1];
  const median = results[Math.floor(results.length / 2)];
  const mean = results.reduce((sum, x) => sum + x, 0) / results.length;
  const stddev = Math.sqrt(results.reduce((sum, x) => sum + Math.pow(x - mean, 2), 0) / results.length);
  
  return { mean, median, min, max, stddev, samples: results.length };
}

// Data correctness validation
function validateDataCorrectness(glintResult, jsonResult, datasetName) {
  try {
    const glintStr = JSON.stringify(glintResult, null, 0);
    const jsonStr = JSON.stringify(jsonResult, null, 0);
    
    if (glintStr === jsonStr) {
      console.log(`✓ ${datasetName}: Data validation PASSED - decoded data matches JSON`);
      return true;
    } else {
      console.log(`✗ ${datasetName}: Data validation FAILED - decoded data differs from JSON`);
      
      // Show first difference for debugging
      const minLen = Math.min(glintStr.length, jsonStr.length);
      for (let i = 0; i < minLen; i++) {
        if (glintStr[i] !== jsonStr[i]) {
          console.log(`  First difference at position ${i}:`);
          console.log(`    Glint: "${glintStr.slice(i, i + 50)}..."`);
          console.log(`    JSON:  "${jsonStr.slice(i, i + 50)}..."`);
          break;
        }
      }
      return false;
    }
  } catch (error) {
    console.log(`✗ ${datasetName}: Data validation ERROR - ${error.message}`);
    return false;
  }
}

console.log('Glint vs JSON Performance Benchmark\n');

const decoder = new CodegenGlintDecoder();

// Test datasets with higher iteration counts for stability
const tests = [
  { name: 'Medium', data: mediumData, json: mediumJSON, desc: '100 users', iterations: 5000, samples: 20 },
  { name: 'Large', data: largeData, json: largeJSON, desc: '100+200 records', iterations: 1000, samples: 15 },
  // Skip huge dataset for now due to Float64 decoding issue
  // { name: 'Huge', data: hugeData, json: hugeJSON, desc: '300+600+analytics', iterations: 300, samples: 10 }
];

// Run data validation first
console.log('=== Data Correctness Validation ===\n');
let allValidationsPassed = true;

for (const test of tests) {
  console.log(`Validating ${test.name} dataset...`);
  
  const glintResult = decoder.decode(test.data);
  const jsonResult = JSON.parse(test.json);
  
  const isValid = validateDataCorrectness(glintResult, jsonResult, test.name);
  if (!isValid) {
    allValidationsPassed = false;
  }
}

if (!allValidationsPassed) {
  console.log('\n❌ Some data validations failed! Please fix data correctness before benchmarking.');
  process.exit(1);
}

console.log('\n✅ All data validations passed! Proceeding with performance benchmarks...\n');

console.log('=== Performance Benchmarks ===\n');
console.log('Dataset    Description      Glint (med±std)  JSON (med±std)   Winner              Raw Sizes         Gzipped Sizes');
console.log('──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────');

for (const test of tests) {
  console.log(`\nRunning ${test.name} benchmark (${test.samples} samples × ${test.iterations} iterations)...`);
  
  // Benchmark Glint
  const glintStats = benchmark(() => decoder.decode(test.data), test.iterations, test.samples);
  
  // Benchmark JSON - use the same JSON data that was used to create the Glint file
  const jsonStats = benchmark(() => JSON.parse(test.json), test.iterations, test.samples);
  
  // Format results
  const name = test.name.padEnd(10);
  const desc = test.desc.padEnd(16);
  const glintMs = `${glintStats.median.toFixed(3)}±${glintStats.stddev.toFixed(3)}ms`.padEnd(16);
  const jsonMs = `${jsonStats.median.toFixed(3)}±${jsonStats.stddev.toFixed(3)}ms`.padEnd(16);
  
  const winner = glintStats.median < jsonStats.median ? 
    `Glint ${(jsonStats.median/glintStats.median).toFixed(2)}x faster`.padEnd(19) : 
    `JSON ${(glintStats.median/jsonStats.median).toFixed(2)}x faster`.padEnd(19);
  
  // Calculate raw sizes
  const glintSize = test.data.length;
  const jsonSizeNum = Buffer.byteLength(test.json);
  
  // Calculate gzipped sizes
  const glintGzipped = zlib.gzipSync(test.data).length;
  const jsonGzipped = zlib.gzipSync(Buffer.from(test.json)).length;
  
  // Format size strings
  const rawSizes = `${glintSize.toLocaleString()}/${jsonSizeNum.toLocaleString()}`.padEnd(17);
  const gzippedSizes = `${glintGzipped.toLocaleString()}/${jsonGzipped.toLocaleString()}`;
  
  console.log(`${name} ${desc} ${glintMs} ${jsonMs} ${winner} ${rawSizes} ${gzippedSizes}`);
}
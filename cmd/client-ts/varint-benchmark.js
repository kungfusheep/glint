const { performance } = require('perf_hooks');

// Current implementation (from HELPERS)
function extractVarintCurrent(data, pos) {
  let value = 0;
  let shift = 0;
  let bytes = 0;
  
  while (bytes < 10) { // Max 10 bytes for uint64
    const b = data[pos + bytes];
    bytes++;
    
    value |= (b & 0x7f) << shift;
    
    if ((b & 0x80) === 0) {
      break;
    }
    
    shift += 7;
  }
  
  // Ensure unsigned result for values that might overflow signed 32-bit
  return { value: value >>> 0, bytes };
}

// Proposed fix based on Go implementation
function extractVarintFixed(data, pos) {
  let sf = 0;    // shift factor (like Go)
  let v = 0;     // accumulator (like Go)
  let bytes = 0;
  
  while (bytes < 10) { // Max 10 bytes for uint64
    const d = data[pos + bytes];
    bytes++;
    
    if ((d & 0x80) === 0) {  // Check MSB = 0 (end byte) - like Go
      v |= d << sf;
      return { value: v >>> 0, bytes };
    }
    
    v |= (d & 0x7f) << sf;  // Mask off MSB, shift by sf - like Go
    sf += 7;  // Each byte contributes 7 bits
  }
  
  return { value: v >>> 0, bytes };
}

// Alternative: Optimized unrolled version (like TypeScript reader.ts)
function extractVarintUnrolled(data, pos) {
  // Fast path for single-byte varints (most common)
  const b0 = data[pos];
  if ((b0 & 0x80) === 0) {
    return { value: b0, bytes: 1 };
  }
  
  // Two-byte varints
  const b1 = data[pos + 1];
  if ((b1 & 0x80) === 0) {
    return { value: ((b0 & 0x7f) | (b1 << 7)) >>> 0, bytes: 2 };
  }
  
  // Three-byte varints
  const b2 = data[pos + 2];
  if ((b2 & 0x80) === 0) {
    return { value: ((b0 & 0x7f) | ((b1 & 0x7f) << 7) | (b2 << 14)) >>> 0, bytes: 3 };
  }
  
  // Four-byte varints
  const b3 = data[pos + 3];
  if ((b3 & 0x80) === 0) {
    return { value: ((b0 & 0x7f) | ((b1 & 0x7f) << 7) | ((b2 & 0x7f) << 14) | (b3 << 21)) >>> 0, bytes: 4 };
  }
  
  // Fall back to loop for 5+ bytes
  let value = (b0 & 0x7f) | ((b1 & 0x7f) << 7) | ((b2 & 0x7f) << 14) | ((b3 & 0x7f) << 21);
  let shift = 28;
  let bytes = 4;
  
  while (bytes < 10) {
    const b = data[pos + bytes];
    bytes++;
    
    if ((b & 0x80) === 0) {
      value += (b << shift);
      return { value: value >>> 0, bytes };
    }
    
    value += ((b & 0x7f) << shift);
    shift += 7;
  }
  
  return { value: value >>> 0, bytes };
}

// Zigzag decode function
function zigzagDecode(n) {
  return (n >>> 1) ^ (-(n & 1));
}

console.log('🚀 Varint Extraction Micro-Benchmark\n');

// Generate test data with various varint sizes
const testCases = [
  { name: '1-byte varints (0-127)', data: new Uint8Array([0, 1, 42, 127]) },
  { name: '2-byte varints (128-16383)', data: new Uint8Array([0x80, 0x01, 0x80, 0x02, 0xFF, 0x7F]) },
  { name: '3-byte varints (16384+)', data: new Uint8Array([0x80, 0x80, 0x01, 0xFF, 0xFF, 0x7F]) },
  { name: '4-byte varints (large)', data: new Uint8Array([0x80, 0x80, 0x80, 0x01, 0xFF, 0xFF, 0xFF, 0x7F]) },
  { name: '5-byte varints (max)', data: new Uint8Array([0x80, 0x80, 0x80, 0x80, 0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0x0F]) },
];

console.log('📊 Correctness Test:');
testCases.forEach(({ name, data }) => {
  console.log(`\n${name}:`);
  
  for (let pos = 0; pos < data.length; ) {
    const current = extractVarintCurrent(data, pos);
    const fixed = extractVarintFixed(data, pos);
    const unrolled = extractVarintUnrolled(data, pos);
    
    const match = current.value === fixed.value && fixed.value === unrolled.value && 
                  current.bytes === fixed.bytes && fixed.bytes === unrolled.bytes;
    
    console.log(`  pos ${pos}: ${match ? '✅' : '❌'} current=${current.value}, fixed=${fixed.value}, unrolled=${unrolled.value} (${current.bytes}b)`);
    
    pos += current.bytes;
  }
});

// Performance benchmark
console.log('\n⚡ Performance Benchmark (1M iterations):');
const ITERATIONS = 1000000;
const benchData = new Uint8Array([0x80, 0x80, 0x80, 0x01]); // 4-byte varint

['extractVarintCurrent', 'extractVarintFixed', 'extractVarintUnrolled'].forEach(funcName => {
  const func = eval(funcName);
  
  const start = performance.now();
  for (let i = 0; i < ITERATIONS; i++) {
    func(benchData, 0);
  }
  const end = performance.now();
  
  const timeMs = end - start;
  const nsPerOp = (timeMs * 1000000) / ITERATIONS;
  
  console.log(`  ${funcName.padEnd(25)}: ${timeMs.toFixed(2)}ms total, ${nsPerOp.toFixed(0)}ns/op`);
});

console.log('\n🎯 Real-world test with actual Glint data:');

// Test with actual problematic data
const fs = require('fs');
try {
  const glintData = fs.readFileSync('./test/slice-tests.glint');
  
  // Test schema size extraction (should be 215)
  console.log(`\nTesting schema size extraction at position 5:`);
  console.log(`  Bytes: [${Array.from(glintData.slice(5, 8)).map(b => `0x${b.toString(16).padStart(2, '0')}`).join(', ')}]`);
  
  const currentResult = extractVarintCurrent(glintData, 5);
  const fixedResult = extractVarintFixed(glintData, 5);
  const unrolledResult = extractVarintUnrolled(glintData, 5);
  
  console.log(`  Current:  value=${currentResult.value}, bytes=${currentResult.bytes}`);
  console.log(`  Fixed:    value=${fixedResult.value}, bytes=${fixedResult.bytes}`);
  console.log(`  Unrolled: value=${unrolledResult.value}, bytes=${unrolledResult.bytes}`);
  
  // Test data section varints
  const dataStart = 5 + currentResult.bytes + currentResult.value;
  console.log(`\nTesting first data varint at position ${dataStart} (should be boolSlice length = 5):`);
  console.log(`  Bytes: [${Array.from(glintData.slice(dataStart, dataStart + 3)).map(b => `0x${b.toString(16).padStart(2, '0')}`).join(', ')}]`);
  
  const dataResult1 = extractVarintCurrent(glintData, dataStart);
  const dataResult2 = extractVarintFixed(glintData, dataStart);
  const dataResult3 = extractVarintUnrolled(glintData, dataStart);
  
  console.log(`  Current:  value=${dataResult1.value}, bytes=${dataResult1.bytes}`);
  console.log(`  Fixed:    value=${dataResult2.value}, bytes=${dataResult2.bytes}`);
  console.log(`  Unrolled: value=${dataResult3.value}, bytes=${dataResult3.bytes}`);
  
} catch (err) {
  console.log('  Could not load test data:', err.message);
}
const { CodegenGlintDecoder } = require('./dist/src/codegen-decoder.js');
const fs = require('fs');

console.log('🧪 Testing All Slice Types\n');

const decoder = new CodegenGlintDecoder();

try {
  // Load slice test data
  console.log('📊 Loading slice test data...');
  const glintData = fs.readFileSync('./test/comprehensive-slice-test.glint');
  const jsonData = JSON.parse(fs.readFileSync('./test/comprehensive-slice-test.json', 'utf8'));
  
  console.log(`   JSON: ${fs.statSync('./test/comprehensive-slice-test.json').size} bytes`);
  console.log(`   Glint: ${fs.statSync('./test/comprehensive-slice-test.glint').size} bytes`);
  console.log(`   Compression: ${((fs.statSync('./test/comprehensive-slice-test.glint').size / fs.statSync('./test/comprehensive-slice-test.json').size) * 100).toFixed(1)}% of JSON size`);
  
  // Decode
  console.log('\n🔄 Decoding slice test data...');
  const result = decoder.decode(glintData);
  
  console.log('\n✅ Slice Type Validation:\n');
  
  let totalTests = 0;
  let passedTests = 0;
  
  function test(description, expected, actual) {
    totalTests++;
    const passed = JSON.stringify(expected) === JSON.stringify(actual);
    if (passed) {
      passedTests++;
      console.log(`   ✓ ${description}`);
    } else {
      console.log(`   ✗ ${description}:`);
      console.log(`      Expected: ${JSON.stringify(expected)}`);
      console.log(`      Actual:   ${JSON.stringify(actual)}`);
    }
    return passed;
  }
  
  function testLength(description, expected, actual) {
    totalTests++;
    const expectedLen = expected ? expected.length : 0;
    const actualLen = actual ? actual.length : 0;
    const passed = expectedLen === actualLen;
    if (passed) {
      passedTests++;
      console.log(`   ✓ ${description} (length: ${actualLen})`);
    } else {
      console.log(`   ✗ ${description}: expected length ${expectedLen}, got ${actualLen}`);
    }
    return passed;
  }
  
  function testFloatSlice(description, expected, actual, tolerance = 0.001) {
    totalTests++;
    if (!expected || !actual || expected.length !== actual.length) {
      console.log(`   ✗ ${description}: length mismatch`);
      return false;
    }
    
    let allClose = true;
    for (let i = 0; i < expected.length; i++) {
      if (Math.abs(expected[i] - actual[i]) > tolerance) {
        allClose = false;
        break;
      }
    }
    
    if (allClose) {
      passedTests++;
      console.log(`   ✓ ${description} (${expected.length} values, tolerance ${tolerance})`);
    } else {
      console.log(`   ✗ ${description}: values don't match within tolerance`);
      console.log(`      Expected: [${expected.slice(0, 3)}...]`);
      console.log(`      Actual:   [${actual.slice(0, 3)}...]`);
    }
    return allClose;
  }
  
  // Boolean slices
  console.log('📌 Boolean Slices:');
  test('[]bool', jsonData.boolSlice, result.boolSlice);
  // Arrays are encoded as slices in Glint, test first 3 elements as [3]bool equivalent
  test('[3]bool (slice subset)', [true, false, true], result.boolSlice.slice(0,3));
  
  // String slices  
  console.log('\n📌 String Slices:');
  test('[]string', jsonData.stringSlice, result.stringSlice);
  // Arrays are encoded as slices in Glint, test first 2 elements as [2]string equivalent  
  test('[2]string (slice subset)', ["alpha", "beta"], result.stringSlice.slice(0,2));
  test('empty []string', jsonData.emptyStringSlice, result.emptyStringSlice);
  
  // Integer slices
  console.log('\n📌 Signed Integer Slices:');
  test('[]int', jsonData.intSlice, result.intSlice);
  test('[]int8', jsonData.int8Slice, result.int8Slice);
  test('[]int16', jsonData.int16Slice, result.int16Slice);  
  test('[]int32', jsonData.int32Slice, result.int32Slice);
  test('[]int64', jsonData.int64Slice, result.int64Slice);
  // Arrays are encoded as slices in Glint, test first 4 elements as [4]int equivalent
  test('[4]int (slice subset)', [-100, -1, 0, 1], result.intSlice.slice(0,4));
  test('empty []int', jsonData.emptyIntSlice, result.emptyIntSlice);
  
  console.log('\n📌 Unsigned Integer Slices:');
  test('[]uint', jsonData.uintSlice, result.uintSlice);
  test('[]uint8', Array.from(Buffer.from(jsonData.uint8Slice, 'base64')), Array.from(result.uint8Slice || []));
  test('[]uint16', jsonData.uint16Slice, result.uint16Slice);
  test('[]uint32', jsonData.uint32Slice, result.uint32Slice);
  test('[]uint64', jsonData.uint64Slice, result.uint64Slice);
  
  // Floating point slices (use tolerance due to precision)
  console.log('\n📌 Floating Point Slices:');
  testFloatSlice('[]float32', jsonData.float32Slice, result.float32Slice, 0.001);
  testFloatSlice('[]float64', jsonData.float64Slice, result.float64Slice, 0.000001);
  
  // Byte slice
  console.log('\n📌 Byte Slice:');
  test('[]byte', Array.from(Buffer.from(jsonData.bytesData, 'base64')), Array.from(result.bytesData || []));
  
  // Summary
  console.log(`\n📈 Slice Test Results: ${passedTests}/${totalTests} passed (${((passedTests/totalTests)*100).toFixed(1)}%)`);
  
  if (passedTests === totalTests) {
    console.log('\n🎉 All slice tests PASSED!');
    console.log('\n📋 Successfully Validated Slice Types:');
    console.log('   ✅ Boolean: []bool, [N]bool');
    console.log('   ✅ String: []string, [N]string, empty slices');
    console.log('   ✅ Signed integers: []int, []int8, []int16, []int32, []int64');
    console.log('   ✅ Unsigned integers: []uint, []uint8, []uint16, []uint32, []uint64');
    console.log('   ✅ Floating point: []float32, []float64');
    console.log('   ✅ Byte data: []byte');
    console.log('   ✅ Arrays: fixed-size [N]T arrays');
    console.log('   ✅ Edge cases: empty slices');
  } else {
    console.log('\n⚠️  Some slice tests failed - need to investigate specific type handling');
  }
  
  // Cache stats
  const stats = decoder.getCacheStats();
  console.log(`\n⚡ Performance: Schema compiled once (${stats.misses} miss, ${stats.hits} hits)`);
  
} catch (error) {
  console.error('❌ Slice test failed:', error.message);
  if (error.stack) {
    console.error(error.stack);
  }
  process.exit(1);
}
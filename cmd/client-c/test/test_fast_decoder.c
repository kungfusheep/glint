#include <stdio.h>
#include <time.h>
#include <sys/time.h>
#include <assert.h>
#include "../src/glint.h"
#include "../../../test/data/comprehensive_test_data.h"

// Declare the fast decoder functions
glint_document_t* glint_decode_fast(const uint8_t* data, size_t len);
void glint_document_free_fast(glint_document_t* doc);

double get_time_us() {
    struct timeval tv;
    gettimeofday(&tv, NULL);
    return tv.tv_sec * 1000000.0 + tv.tv_usec;
}

void benchmark_fast_decoder() {
    printf("=== Benchmarking Fast Instruction-Based Decoder ===\n");
    
    // Warmup - this will cache the schema
    for (int i = 0; i < 1000; i++) {
        glint_document_t* doc = glint_decode_fast(COMPREHENSIVE_TEST_DATA, COMPREHENSIVE_TEST_DATA_LEN);
        glint_document_free_fast(doc);
    }
    
    // Measure with cached schema
    int iterations = 100000;
    double start = get_time_us();
    
    for (int i = 0; i < iterations; i++) {
        glint_document_t* doc = glint_decode_fast(COMPREHENSIVE_TEST_DATA, COMPREHENSIVE_TEST_DATA_LEN);
        glint_document_free_fast(doc);
    }
    
    double end = get_time_us();
    double elapsed_us = end - start;
    double ns_per_op = (elapsed_us * 1000) / iterations;
    double mb_per_sec = (COMPREHENSIVE_TEST_DATA_LEN * iterations / (elapsed_us / 1000000.0)) / (1024 * 1024);
    
    printf("  Iterations: %d\n", iterations);
    printf("  Total time: %.3f seconds\n", elapsed_us / 1000000.0);
    printf("  Time per decode: %.0f ns\n", ns_per_op);
    printf("  Throughput: %.0f MB/s\n", mb_per_sec);
    
    // Compare with expected performance
    printf("\n📊 Performance vs Go:\n");
    printf("  Fast C decoder: %.0f ns\n", ns_per_op);
    printf("  Go decoder: 792 ns\n");
    
    if (ns_per_op < 792) {
        printf("  🚀 %.1fx FASTER than Go!\n", 792.0 / ns_per_op);
    } else if (ns_per_op < 1000) {
        printf("  ✓ Within competitive range (under 1μs)\n");
    } else {
        printf("  Go is %.1fx faster\n", ns_per_op / 792.0);
    }
}

void test_cache_effectiveness() {
    printf("\n=== Testing Schema Cache Effectiveness ===\n");
    
    // First decode - will parse and cache schema
    double start = get_time_us();
    glint_document_t* doc1 = glint_decode_fast(COMPREHENSIVE_TEST_DATA, COMPREHENSIVE_TEST_DATA_LEN);
    double first_time = get_time_us() - start;
    glint_document_free_fast(doc1);
    
    // Second decode - should use cached schema
    start = get_time_us();
    glint_document_t* doc2 = glint_decode_fast(COMPREHENSIVE_TEST_DATA, COMPREHENSIVE_TEST_DATA_LEN);
    double cached_time = get_time_us() - start;
    glint_document_free_fast(doc2);
    
    printf("  First decode (with schema parsing): %.0f ns\n", first_time * 1000);
    printf("  Second decode (cached schema): %.0f ns\n", cached_time * 1000);
    printf("  Speedup from caching: %.1fx\n", first_time / cached_time);
}

int main() {
    printf("=== Fast C Decoder Performance Test ===\n\n");
    
    // Test basic functionality
    printf("Testing basic decode...\n");
    glint_document_t* doc = glint_decode_fast(COMPREHENSIVE_TEST_DATA, COMPREHENSIVE_TEST_DATA_LEN);
    assert(doc != NULL);
    assert(doc->schema_crc32 == 0x3b767ba1);
    
    // Verify some decoded values to ensure we're actually decoding
    uint8_t* data = (uint8_t*)doc->root;
    bool first_bool = *(bool*)(data + 0);  // First field should be bool
    printf("  First bool value: %d (expected 1)\n", first_bool);
    assert(first_bool == 1);
    
    printf("✅ Basic decode successful\n\n");
    glint_document_free_fast(doc);
    
    // Test cache effectiveness
    test_cache_effectiveness();
    
    // Run benchmark
    benchmark_fast_decoder();
    
    printf("\n✅ Test complete!\n");
    return 0;
}
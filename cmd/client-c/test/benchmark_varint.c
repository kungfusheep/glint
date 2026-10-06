/*
 * Isolated varint benchmark to test optimization strategies
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <time.h>
#include <sys/time.h>
#include "../src/glint_varint.h"
#include "../src/glint_varint_opt.h"

// Test data structures
typedef struct {
    uint8_t data[10];
    size_t len;
    uint64_t expected;
    const char* name;
} varint_test_case_t;

// Generate test cases covering all varint sizes
static varint_test_case_t test_cases[] = {
    // 1-byte varints (most common)
    {{0x00}, 1, 0, "zero"},
    {{0x01}, 1, 1, "one"},
    {{0x7F}, 1, 127, "max_1byte"},
    
    // 2-byte varints  
    {{0x80, 0x01}, 2, 128, "min_2byte"},
    {{0xFF, 0x01}, 2, 255, "255"},
    {{0x80, 0x02}, 2, 256, "256"},
    {{0xFF, 0x7F}, 2, 16383, "max_2byte"},
    
    // 3-byte varints
    {{0x80, 0x80, 0x01}, 3, 16384, "min_3byte"},
    {{0xFF, 0xFF, 0x03}, 3, 65535, "65535"},
    {{0xFF, 0xFF, 0x7F}, 3, 2097151, "max_3byte"},
    
    // 4-byte varints
    {{0x80, 0x80, 0x80, 0x01}, 4, 2097152, "min_4byte"},
    {{0xFF, 0xFF, 0xFF, 0x7F}, 4, 268435455, "max_4byte"},
    
    // 5-byte varints
    {{0x80, 0x80, 0x80, 0x80, 0x01}, 5, 268435456, "min_5byte"},
    {{0xFF, 0xFF, 0xFF, 0xFF, 0x0F}, 5, 4294967295, "max_uint32"},
    
    // 10-byte varint (max)
    {{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x01}, 10, 
     UINT64_MAX, "max_uint64"},
};

static const int num_test_cases = sizeof(test_cases) / sizeof(test_cases[0]);

// Alternative varint implementation 1: Loop unrolling
static size_t glint_decode_varint_unrolled(const uint8_t* data, size_t len, uint64_t* out) {
    if (len == 0) return 0;
    
    // Fast path for single byte
    if ((data[0] & 0x80) == 0) {
        *out = data[0];
        return 1;
    }
    
    if (len < 2) return 0;
    uint64_t result = data[0] & 0x7F;
    
    // Unroll common cases
    if ((data[1] & 0x80) == 0) {
        *out = result | ((uint64_t)data[1] << 7);
        return 2;
    }
    
    if (len < 3) return 0;
    result |= ((uint64_t)(data[1] & 0x7F) << 7);
    
    if ((data[2] & 0x80) == 0) {
        *out = result | ((uint64_t)data[2] << 14);
        return 3;
    }
    
    if (len < 4) return 0;
    result |= ((uint64_t)(data[2] & 0x7F) << 14);
    
    if ((data[3] & 0x80) == 0) {
        *out = result | ((uint64_t)data[3] << 21);
        return 4;
    }
    
    if (len < 5) return 0;
    result |= ((uint64_t)(data[3] & 0x7F) << 21);
    
    if ((data[4] & 0x80) == 0) {
        *out = result | ((uint64_t)data[4] << 28);
        return 5;
    }
    
    // Continue for remaining bytes
    result |= ((uint64_t)(data[4] & 0x7F) << 28);
    size_t shift = 35;
    
    for (size_t i = 5; i < len && i < 10; i++) {
        uint8_t byte = data[i];
        result |= ((uint64_t)(byte & 0x7F)) << shift;
        if ((byte & 0x80) == 0) {
            *out = result;
            return i + 1;
        }
        shift += 7;
    }
    
    return 0;
}

// Alternative varint implementation 2: Branchless with lookup table
static const uint8_t continuation_mask[256] = {
    [0 ... 127] = 0,
    [128 ... 255] = 1,
};

static size_t glint_decode_varint_branchless(const uint8_t* data, size_t len, uint64_t* out) {
    if (len == 0) return 0;
    
    uint64_t result = 0;
    size_t bytes_read = 0;
    
    // Process up to 10 bytes
    for (size_t i = 0; i < len && i < 10; i++) {
        uint8_t byte = data[i];
        result |= ((uint64_t)(byte & 0x7F)) << (i * 7);
        bytes_read++;
        
        // Branchless check for continuation bit
        if (!continuation_mask[byte]) {
            *out = result;
            return bytes_read;
        }
    }
    
    return 0;
}

// Alternative varint implementation 3: SIMD-inspired (process multiple bytes at once)
static size_t glint_decode_varint_simd(const uint8_t* data, size_t len, uint64_t* out) {
    if (len == 0) return 0;
    
    // Fast path for 1-4 bytes (can read as uint32)
    if (len >= 4) {
        uint32_t chunk;
        memcpy(&chunk, data, 4);
        
        // Check continuation bits in parallel
        uint32_t cont_bits = chunk & 0x80808080;
        
        if (cont_bits == 0) {
            // Single byte
            *out = chunk & 0x7F;
            return 1;
        } else if ((cont_bits & 0x00008080) == 0) {
            // Two bytes
            *out = (chunk & 0x7F) | ((chunk & 0x7F00) >> 1);
            return 2;
        } else if ((cont_bits & 0x00808000) == 0) {
            // Three bytes
            *out = (chunk & 0x7F) | 
                   ((chunk & 0x7F00) >> 1) |
                   ((chunk & 0x7F0000) >> 2);
            return 3;
        } else if ((cont_bits & 0x80000000) == 0) {
            // Four bytes
            *out = (chunk & 0x7F) | 
                   ((chunk & 0x7F00) >> 1) |
                   ((chunk & 0x7F0000) >> 2) |
                   ((chunk & 0x7F000000) >> 3);
            return 4;
        }
    }
    
    // Fallback to standard implementation
    return glint_decode_varint(data, len, out);
}

// Benchmark runner
static uint64_t get_time_ns() {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return ts.tv_sec * 1000000000ULL + ts.tv_nsec;
}

typedef size_t (*varint_decoder_t)(const uint8_t*, size_t, uint64_t*);

static void benchmark_decoder(const char* name, varint_decoder_t decoder, int iterations) {
    // Prepare mixed test data (realistic distribution)
    // 70% 1-byte, 20% 2-byte, 8% 3-byte, 2% larger
    int total_decodes = iterations * num_test_cases;
    uint64_t dummy = 0;  // Prevent optimization
    
    uint64_t start = get_time_ns();
    
    for (int iter = 0; iter < iterations; iter++) {
        for (int i = 0; i < num_test_cases; i++) {
            uint64_t result;
            size_t bytes = decoder(test_cases[i].data, test_cases[i].len, &result);
            dummy += result;
            
            // Weighted repetition for realistic distribution
            if (i < 3) {  // 1-byte cases
                for (int j = 0; j < 7; j++) {
                    bytes = decoder(test_cases[i].data, test_cases[i].len, &result);
                    dummy += result;
                }
            } else if (i < 7) {  // 2-byte cases
                for (int j = 0; j < 2; j++) {
                    bytes = decoder(test_cases[i].data, test_cases[i].len, &result);
                    dummy += result;
                }
            }
        }
    }
    
    uint64_t end = get_time_ns();
    uint64_t elapsed = end - start;
    double ns_per_decode = (double)elapsed / total_decodes;
    double throughput = (total_decodes * 1000000000.0) / elapsed;
    
    printf("%-20s: %8.1f ns/decode, %10.0f decodes/sec (checksum: %llu)\n", 
           name, ns_per_decode, throughput, dummy);
}

// Correctness test
static int test_decoder(const char* name, varint_decoder_t decoder) {
    int passed = 0;
    int failed = 0;
    
    for (int i = 0; i < num_test_cases; i++) {
        uint64_t result;
        size_t bytes = decoder(test_cases[i].data, test_cases[i].len, &result);
        
        if (bytes != test_cases[i].len || result != test_cases[i].expected) {
            printf("FAIL %s on %s: got %llu (bytes=%zu), expected %llu (bytes=%zu)\n",
                   name, test_cases[i].name, result, bytes, 
                   test_cases[i].expected, test_cases[i].len);
            failed++;
        } else {
            passed++;
        }
    }
    
    if (failed == 0) {
        printf("✓ %s: All %d tests passed\n", name, passed);
        return 1;
    }
    return 0;
}

int main(int argc, char** argv) {
    int iterations = 1000000;
    if (argc > 1) {
        iterations = atoi(argv[1]);
    }
    
    printf("=== Varint Decoder Correctness Tests ===\n");
    test_decoder("Original", glint_decode_varint);
    test_decoder("Unrolled", glint_decode_varint_unrolled);
    test_decoder("Branchless", glint_decode_varint_branchless);
    test_decoder("SIMD-style", glint_decode_varint_simd);
    test_decoder("Prefetch", glint_decode_varint_prefetch);
    test_decoder("Predict", glint_decode_varint_predict);
#ifdef __GNUC__
    test_decoder("Computed Goto", glint_decode_varint_goto);
#endif
    
    printf("\n=== Varint Decoder Performance Benchmark ===\n");
    printf("Iterations: %d (with weighted distribution)\n\n", iterations);
    
    // Warmup
    benchmark_decoder("Warmup", glint_decode_varint, 1000);
    
    printf("\nResults:\n");
    benchmark_decoder("Original", glint_decode_varint, iterations);
    benchmark_decoder("Unrolled", glint_decode_varint_unrolled, iterations);
    benchmark_decoder("Branchless", glint_decode_varint_branchless, iterations);
    benchmark_decoder("SIMD-style", glint_decode_varint_simd, iterations);
    benchmark_decoder("Prefetch", glint_decode_varint_prefetch, iterations);
    benchmark_decoder("Predict", glint_decode_varint_predict, iterations);
#ifdef __GNUC__
    benchmark_decoder("Computed Goto", glint_decode_varint_goto, iterations);
#endif
    
    return 0;
}
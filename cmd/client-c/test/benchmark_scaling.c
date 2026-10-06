#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <sys/time.h>
#include "../src/glint_final.h"
#include "../../../test/data/scale_SMALL.h"
#include "../../../test/data/scale_MEDIUM.h"
#include "../../../test/data/scale_LARGE.h"
// #include "../../../test/data/scale_XLARGE.h" // Not generated yet

// High-resolution timer
static double get_time_ns() {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return ts.tv_sec * 1000000000.0 + ts.tv_nsec;
}

// Structure matching Go's LargeTestStruct
typedef struct {
    int32_t version;
    int64_t timestamp;
    glint_int32_slice_t int_array;
    glint_float64_slice_t float_array;
    glint_string_slice_t string_array;
    glint_bool_slice_t bool_array;
    // Records would be complex - skipping for now
} large_test_struct_t;

static double benchmark_decoder(const uint8_t* data, size_t len, int iterations) {
    // Warmup
    for (int i = 0; i < 10; i++) {
        large_test_struct_t result = {0};
        glint_decode_fast(data, len, &result);
    }

    // Actual benchmark
    double start = get_time_ns();
    for (int i = 0; i < iterations; i++) {
        large_test_struct_t result = {0};
        glint_decode_fast(data, len, &result);
    }
    double elapsed = get_time_ns() - start;

    return elapsed / iterations;
}

int main() {
    printf("C Decoder Performance Scaling Test\n");
    printf("=====================================\n");
    printf("%-20s %-15s %-15s %-15s\n", "Size", "Time/op", "Throughput", "ns/byte");
    printf("---------------------------------------------------------------------\n");

    struct {
        const char* name;
        const uint8_t* data;
        size_t len;
    } tests[] = {
        {"SMALL (2.3KB)", SCALE_SMALL_DATA, SCALE_SMALL_DATA_LEN},
        {"MEDIUM (20KB)", SCALE_MEDIUM_DATA, SCALE_MEDIUM_DATA_LEN},
        {"LARGE (175KB)", SCALE_LARGE_DATA, SCALE_LARGE_DATA_LEN},
        // {"XLARGE (841KB)", SCALE_XLARGE_DATA, SCALE_XLARGE_DATA_LEN},
    };

    for (int t = 0; t < 3; t++) {
        int iterations = 10000;
        if (tests[t].len > 100000) iterations = 1000;
        if (tests[t].len > 500000) iterations = 100;

        double ns_per_op = benchmark_decoder(tests[t].data, tests[t].len, iterations);
        double throughput_mbps = (tests[t].len / (ns_per_op / 1000000000.0)) / (1024.0 * 1024.0);
        double ns_per_byte = ns_per_op / tests[t].len;

        printf("%-20s %-15.0f ns %-15.1f MB/s %-15.2f ns/B\n",
               tests[t].name, ns_per_op, throughput_mbps, ns_per_byte);
    }

    return 0;
}
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <sys/time.h>
#include "../src/glint.h"
#include "../src/glint_decoder_fast.c"
#include "../../../test/data/scale_SMALL.h"
#include "../../../test/data/scale_MEDIUM.h"
#include "../../../test/data/scale_LARGE.h"

// High-resolution timer
static double get_time_ns() {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return ts.tv_sec * 1000000000.0 + ts.tv_nsec;
}

static double benchmark_decoder(const uint8_t* data, size_t len, int iterations) {
    // Warmup
    for (int i = 0; i < 10; i++) {
        glint_document_t* doc = glint_decode_fast(data, len);
        glint_document_free_fast(doc);
    }

    // Actual benchmark
    double start = get_time_ns();
    for (int i = 0; i < iterations; i++) {
        glint_document_t* doc = glint_decode_fast(data, len);
        glint_document_free_fast(doc);
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

    printf("\n\nComparison with Go:\n");
    printf("---------------------------------------------------------------------\n");
    printf("Go Results:\n");
    printf("SMALL (2.3KB)        2945 ns         777.2 MB/s      1.23 ns/B\n");
    printf("MEDIUM (20KB)        15692 ns        1220.4 MB/s     0.78 ns/B\n");
    printf("LARGE (175KB)        115864 ns       1473.5 MB/s     0.65 ns/B\n");

    return 0;
}
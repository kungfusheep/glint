/*
 * Glint C Decoder Benchmark
 * Matches the Go benchmark methodology for fair comparison
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <sys/time.h>
#include <assert.h>
#include <math.h>
#include "../src/glint.h"
#include "../../../test/data/comprehensive_test_data.h"

// External decoder functions
extern glint_document_t* glint_decode(const uint8_t* data, size_t len);
extern glint_document_t* glint_decode_fast(const uint8_t* data, size_t len);
extern void glint_document_free(glint_document_t* doc);
extern void glint_document_free_fast(glint_document_t* doc);

// Get high-precision time in nanoseconds
static uint64_t get_time_ns() {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return ts.tv_sec * 1000000000ULL + ts.tv_nsec;
}

// Statistics tracking
typedef struct {
    double* times;
    size_t count;
    size_t capacity;
    double min;
    double max;
    double mean;
    double median;
    double stddev;
    double p95;
    double p99;
} bench_stats_t;

static void stats_init(bench_stats_t* stats, size_t capacity) {
    stats->times = malloc(capacity * sizeof(double));
    stats->count = 0;
    stats->capacity = capacity;
    stats->min = 1e9;
    stats->max = 0;
}

static void stats_add(bench_stats_t* stats, double time_ns) {
    if (stats->count >= stats->capacity) {
        stats->capacity *= 2;
        stats->times = realloc(stats->times, stats->capacity * sizeof(double));
    }
    stats->times[stats->count++] = time_ns;
    if (time_ns < stats->min) stats->min = time_ns;
    if (time_ns > stats->max) stats->max = time_ns;
}

static int compare_double(const void* a, const void* b) {
    double da = *(double*)a;
    double db = *(double*)b;
    if (da < db) return -1;
    if (da > db) return 1;
    return 0;
}

static void stats_compute(bench_stats_t* stats) {
    if (stats->count == 0) return;
    
    // Sort for percentiles
    qsort(stats->times, stats->count, sizeof(double), compare_double);
    
    // Compute mean
    double sum = 0;
    for (size_t i = 0; i < stats->count; i++) {
        sum += stats->times[i];
    }
    stats->mean = sum / stats->count;
    
    // Compute median
    if (stats->count % 2 == 0) {
        stats->median = (stats->times[stats->count/2 - 1] + stats->times[stats->count/2]) / 2.0;
    } else {
        stats->median = stats->times[stats->count/2];
    }
    
    // Compute standard deviation
    double sq_sum = 0;
    for (size_t i = 0; i < stats->count; i++) {
        double diff = stats->times[i] - stats->mean;
        sq_sum += diff * diff;
    }
    stats->stddev = sqrt(sq_sum / stats->count);
    
    // Compute percentiles
    size_t p95_idx = (size_t)(stats->count * 0.95);
    size_t p99_idx = (size_t)(stats->count * 0.99);
    stats->p95 = stats->times[p95_idx];
    stats->p99 = stats->times[p99_idx];
}

static void stats_free(bench_stats_t* stats) {
    free(stats->times);
}

// Benchmark runner matching Go's testing.B methodology
typedef struct {
    const char* name;
    size_t iterations;
    uint64_t total_time_ns;
    bench_stats_t stats;
} benchmark_t;

static void benchmark_run(benchmark_t* b, void (*fn)(benchmark_t*)) {
    // Warmup phase - run for at least 100ms to determine iteration count
    printf("%-30s ", b->name);
    fflush(stdout);
    
    // Initial warmup
    size_t warmup_iters = 1;
    uint64_t warmup_start = get_time_ns();
    b->iterations = warmup_iters;
    fn(b);
    uint64_t warmup_time = get_time_ns() - warmup_start;
    
    // Scale up iterations to run for ~1 second
    if (warmup_time > 0) {
        b->iterations = (size_t)(1000000000.0 / (warmup_time / warmup_iters));
        if (b->iterations < 100) b->iterations = 100;
        if (b->iterations > 10000000) b->iterations = 10000000;
    } else {
        b->iterations = 1000000;
    }
    
    // Initialize stats
    stats_init(&b->stats, b->iterations);
    
    // Main benchmark run - collect individual operation times
    for (size_t i = 0; i < b->iterations; i++) {
        uint64_t start = get_time_ns();
        fn(b);  // Run single iteration
        uint64_t elapsed = get_time_ns() - start;
        stats_add(&b->stats, (double)elapsed);
    }
    
    // Compute statistics
    stats_compute(&b->stats);
    
    // Report results (similar to Go's benchmark output)
    printf("%8zu iterations  %7.0f ns/op", 
           b->iterations, b->stats.mean);
    
    // Show percentiles for consistency checking
    printf("  (p50=%.0fns p95=%.0fns p99=%.0fns)",
           b->stats.median, b->stats.p95, b->stats.p99);
    
    printf("\n");
    
    stats_free(&b->stats);
}

// Benchmark functions
static void bench_schema_decode(benchmark_t* b) {
    glint_document_t* doc = glint_decode(COMPREHENSIVE_TEST_DATA, COMPREHENSIVE_TEST_DATA_LEN);
    glint_document_free(doc);
}

static void bench_fast_decode(benchmark_t* b) {
    glint_document_t* doc = glint_decode_fast(COMPREHENSIVE_TEST_DATA, COMPREHENSIVE_TEST_DATA_LEN);
    glint_document_free_fast(doc);
}

// Partial decode benchmark - decode into smaller struct
typedef struct {
    bool bool_value;
    int32_t int_value;
    char* string_value;
    float float32_value;
    double float64_value;
} PartialData;

static void bench_partial_decode(benchmark_t* b) {
    // For now, just use the schema decoder
    // In a real implementation, this would decode only selected fields
    glint_document_t* doc = glint_decode(COMPREHENSIVE_TEST_DATA, COMPREHENSIVE_TEST_DATA_LEN);
    glint_document_free(doc);
}

// Memory allocation benchmark
static void bench_alloc_free(benchmark_t* b) {
    void* ptr = malloc(1024);
    free(ptr);
}

// Benchmark with different data sizes
static void run_size_benchmarks() {
    printf("\n=== Data Size Scaling ===\n");
    
    // Create different sized test data by repeating the comprehensive data
    size_t sizes[] = {100, 500, 1000, 5000, 10000};
    
    for (size_t i = 0; i < sizeof(sizes)/sizeof(sizes[0]); i++) {
        size_t data_size = sizes[i];
        uint8_t* large_data = malloc(data_size);
        
        // Fill with pattern from comprehensive data
        for (size_t j = 0; j < data_size; j++) {
            large_data[j] = COMPREHENSIVE_TEST_DATA[j % COMPREHENSIVE_TEST_DATA_LEN];
        }
        
        // Note: This won't produce valid Glint data, but tests raw throughput
        char name[64];
        snprintf(name, sizeof(name), "decode_%zu_bytes", data_size);
        
        benchmark_t b = {.name = name};
        // Would run benchmark here with large_data
        // For now just show the size
        printf("%-30s %zu bytes (not implemented)\n", name, data_size);
        
        free(large_data);
    }
}

// Compare with Go implementation
static void show_comparison() {
    printf("\n=== Performance Comparison ===\n");
    printf("Test data: %d bytes (comprehensive test data)\n", COMPREHENSIVE_TEST_DATA_LEN);
    printf("\n");
    printf("Implementation          Time/op      Relative\n");
    printf("------------------------------------------------\n");
    
    // Run quick benchmark for comparison
    benchmark_t c_schema = {.name = "C (schema)", .iterations = 100000};
    benchmark_t c_fast = {.name = "C (fast)", .iterations = 100000};
    
    // Collect timing for C implementations
    stats_init(&c_schema.stats, c_schema.iterations);
    stats_init(&c_fast.stats, c_fast.iterations);
    
    for (size_t i = 0; i < c_schema.iterations; i++) {
        uint64_t start = get_time_ns();
        bench_schema_decode(&c_schema);
        stats_add(&c_schema.stats, (double)(get_time_ns() - start));
    }
    
    for (size_t i = 0; i < c_fast.iterations; i++) {
        uint64_t start = get_time_ns();
        bench_fast_decode(&c_fast);
        stats_add(&c_fast.stats, (double)(get_time_ns() - start));
    }
    
    stats_compute(&c_schema.stats);
    stats_compute(&c_fast.stats);
    
    double go_time = 792.0; // From Go benchmark results
    
    printf("C (schema decoder)      %6.0f ns    %.1fx\n", 
           c_schema.stats.mean, go_time / c_schema.stats.mean);
    printf("C (fast decoder)        %6.0f ns    %.1fx\n", 
           c_fast.stats.mean, go_time / c_fast.stats.mean);
    printf("Go (reference)          %6.0f ns    1.0x\n", go_time);
    printf("TypeScript              %6.0f ns    %.1fx\n", 
           1842.0, go_time / 1842.0);
    
    stats_free(&c_schema.stats);
    stats_free(&c_fast.stats);
    
    // Throughput calculation
    double fast_throughput_mb = (COMPREHENSIVE_TEST_DATA_LEN * 1000.0) / c_fast.stats.mean;
    printf("\nThroughput (fast decoder): %.1f MB/s\n", fast_throughput_mb);
}

int main(int argc, char** argv) {
    printf("=== Glint C Decoder Benchmark ===\n");
    printf("Using comprehensive test data (%d bytes)\n\n", COMPREHENSIVE_TEST_DATA_LEN);
    
    // Validate decoders work before benchmarking
    glint_document_t* test_doc = glint_decode(COMPREHENSIVE_TEST_DATA, COMPREHENSIVE_TEST_DATA_LEN);
    if (!test_doc) {
        printf("ERROR: Schema decoder failed validation\n");
        return 1;
    }
    glint_document_free(test_doc);
    
    test_doc = glint_decode_fast(COMPREHENSIVE_TEST_DATA, COMPREHENSIVE_TEST_DATA_LEN);
    if (!test_doc) {
        printf("ERROR: Fast decoder failed validation\n");
        return 1;
    }
    glint_document_free_fast(test_doc);
    
    printf("=== Decode Benchmarks ===\n");
    
    // Main benchmarks
    benchmark_t benchmarks[] = {
        {.name = "schema-decode"},
        {.name = "fast-decode"},
        {.name = "partial-decode"},
        {.name = "alloc-free-baseline"},
    };
    
    void (*bench_fns[])(benchmark_t*) = {
        bench_schema_decode,
        bench_fast_decode,
        bench_partial_decode,
        bench_alloc_free,
    };
    
    for (size_t i = 0; i < sizeof(benchmarks)/sizeof(benchmarks[0]); i++) {
        benchmark_run(&benchmarks[i], bench_fns[i]);
    }
    
    // Show size scaling benchmarks
    run_size_benchmarks();
    
    // Show comparison with other implementations
    show_comparison();
    
    return 0;
}
/*
 * Glint C Decoder Profiler
 * Generates pprof-compatible CPU profiles
 */

#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <signal.h>
#include <time.h>
#include <unistd.h>
#include <sys/time.h>
#include <execinfo.h>
#include <dlfcn.h>
#include "../src/glint.h"
#include "../src/glint_reader.h"
#include "../../../test/data/comprehensive_test_data.h"

// External decoder functions
extern glint_document_t* glint_decode_fast(const uint8_t* data, size_t len);
extern void glint_document_free_fast(glint_document_t* doc);

// Profiling data structures
#define MAX_STACK_DEPTH 64
#define MAX_SAMPLES 1000000
#define SAMPLE_RATE_HZ 10000  // 10kHz sampling

typedef struct {
    void* stack[MAX_STACK_DEPTH];
    int depth;
    uint64_t count;
} stack_trace_t;

typedef struct {
    stack_trace_t* samples;
    size_t sample_count;
    size_t sample_capacity;
    uint64_t total_samples;
} profile_data_t;

static profile_data_t g_profile = {0};
static volatile int g_profiling = 0;
static struct itimerval g_timer;

// Compare stack traces for sorting/deduplication
static int compare_stacks(const void* a, const void* b) {
    const stack_trace_t* sa = (const stack_trace_t*)a;
    const stack_trace_t* sb = (const stack_trace_t*)b;
    
    if (sa->depth != sb->depth) {
        return sa->depth - sb->depth;
    }
    
    for (int i = 0; i < sa->depth; i++) {
        if (sa->stack[i] != sb->stack[i]) {
            return (sa->stack[i] < sb->stack[i]) ? -1 : 1;
        }
    }
    
    return 0;
}

// Signal handler for profiling timer
static void profile_handler(int sig) {
    (void)sig;
    
    if (!g_profiling) return;
    
    // Capture stack trace
    void* trace[MAX_STACK_DEPTH];
    int depth = backtrace(trace, MAX_STACK_DEPTH);
    
    if (depth <= 0) return;
    
    // Store sample
    if (g_profile.sample_count >= g_profile.sample_capacity) {
        // Aggregate duplicates when buffer is full
        // In real implementation, would use a hash table
        return;
    }
    
    stack_trace_t* sample = &g_profile.samples[g_profile.sample_count++];
    sample->depth = depth;
    sample->count = 1;
    memcpy(sample->stack, trace, depth * sizeof(void*));
    g_profile.total_samples++;
}

// Start profiling
static void start_profiling() {
    // Allocate sample buffer
    g_profile.sample_capacity = MAX_SAMPLES;
    g_profile.samples = calloc(MAX_SAMPLES, sizeof(stack_trace_t));
    g_profile.sample_count = 0;
    g_profile.total_samples = 0;
    
    // Set up signal handler
    struct sigaction sa;
    sa.sa_handler = profile_handler;
    sigemptyset(&sa.sa_mask);
    sa.sa_flags = SA_RESTART;
    sigaction(SIGPROF, &sa, NULL);
    
    // Start timer
    g_timer.it_interval.tv_sec = 0;
    g_timer.it_interval.tv_usec = 1000000 / SAMPLE_RATE_HZ;
    g_timer.it_value = g_timer.it_interval;
    
    g_profiling = 1;
    setitimer(ITIMER_PROF, &g_timer, NULL);
}

// Stop profiling
static void stop_profiling() {
    g_profiling = 0;
    
    // Stop timer
    g_timer.it_interval.tv_sec = 0;
    g_timer.it_interval.tv_usec = 0;
    g_timer.it_value = g_timer.it_interval;
    setitimer(ITIMER_PROF, &g_timer, NULL);
}

// Write pprof-compatible output
static void write_pprof_profile(const char* filename) {
    FILE* f = fopen(filename, "w");
    if (!f) {
        perror("Failed to open profile file");
        return;
    }
    
    // Write pprof text format header
    fprintf(f, "--- contention:\ncycles/second=1000000000\n");
    fprintf(f, "sampling period=1\n");
    fprintf(f, "ms since reset=1000\n");
    fprintf(f, "format=java\n");
    fprintf(f, "resolution=lines\n");
    fprintf(f, "discarded samples=0\n");
    fprintf(f, "\n");
    
    // Sort and aggregate samples
    qsort(g_profile.samples, g_profile.sample_count, sizeof(stack_trace_t), compare_stacks);
    
    // Aggregate identical stacks
    size_t unique_stacks = 0;
    for (size_t i = 0; i < g_profile.sample_count; i++) {
        if (i > 0 && compare_stacks(&g_profile.samples[i], &g_profile.samples[unique_stacks-1]) == 0) {
            g_profile.samples[unique_stacks-1].count++;
        } else {
            if (i != unique_stacks) {
                g_profile.samples[unique_stacks] = g_profile.samples[i];
            }
            unique_stacks++;
        }
    }
    
    // Write samples in pprof format
    for (size_t i = 0; i < unique_stacks; i++) {
        stack_trace_t* sample = &g_profile.samples[i];
        
        // Write sample count
        fprintf(f, "%llu", sample->count);
        
        // Write stack frames (bottom to top)
        for (int j = sample->depth - 1; j >= 0; j--) {
            Dl_info info;
            if (dladdr(sample->stack[j], &info) && info.dli_sname) {
                fprintf(f, " %s", info.dli_sname);
            } else {
                fprintf(f, " %p", sample->stack[j]);
            }
        }
        fprintf(f, "\n");
    }
    
    fclose(f);
    
    printf("Profile written to %s\n", filename);
    printf("Total samples: %llu\n", g_profile.total_samples);
    printf("Unique stacks: %zu\n", unique_stacks);
}

// Alternative: Generate a simpler flamegraph-compatible format
static void write_flamegraph_profile(const char* filename) {
    FILE* f = fopen(filename, "w");
    if (!f) {
        perror("Failed to open flamegraph file");
        return;
    }
    
    // Sort and aggregate samples
    qsort(g_profile.samples, g_profile.sample_count, sizeof(stack_trace_t), compare_stacks);
    
    // Write in flamegraph format: stack;frame;frame count
    for (size_t i = 0; i < g_profile.sample_count; i++) {
        stack_trace_t* sample = &g_profile.samples[i];
        
        // Skip duplicates (already counted)
        if (i > 0 && compare_stacks(&g_profile.samples[i], &g_profile.samples[i-1]) == 0) {
            g_profile.samples[i-1].count++;
            continue;
        }
        
        // Build stack string (bottom to top)
        for (int j = sample->depth - 1; j >= 0; j--) {
            if (j < sample->depth - 1) fprintf(f, ";");
            
            Dl_info info;
            if (dladdr(sample->stack[j], &info)) {
                if (info.dli_sname) {
                    fprintf(f, "%s", info.dli_sname);
                } else if (info.dli_fname) {
                    // Extract just the filename
                    const char* fname = strrchr(info.dli_fname, '/');
                    fprintf(f, "%s`%p", fname ? fname+1 : info.dli_fname, sample->stack[j]);
                } else {
                    fprintf(f, "unknown`%p", sample->stack[j]);
                }
            } else {
                fprintf(f, "unknown`%p", sample->stack[j]);
            }
        }
        fprintf(f, " %llu\n", sample->count);
    }
    
    fclose(f);
    printf("Flamegraph data written to %s\n", filename);
}

// Run workload while profiling
static void run_decode_workload(int iterations) {
    printf("Running %d decode iterations with profiling...\n", iterations);
    
    for (int i = 0; i < iterations; i++) {
        glint_document_t* doc = glint_decode_fast(
            COMPREHENSIVE_TEST_DATA, 
            COMPREHENSIVE_TEST_DATA_LEN
        );
        glint_document_free_fast(doc);
        
        // Print progress
        if ((i + 1) % 10000 == 0) {
            printf("  %d iterations completed\n", i + 1);
        }
    }
}

// Instrument specific functions manually (alternative to sampling)
typedef struct {
    const char* name;
    uint64_t calls;
    uint64_t total_ns;
    uint64_t min_ns;
    uint64_t max_ns;
} func_stats_t;

#define MAX_INSTRUMENTED_FUNCS 100
static func_stats_t g_func_stats[MAX_INSTRUMENTED_FUNCS];
static int g_func_count = 0;

// Manual instrumentation helpers
static inline uint64_t get_time_ns() {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return ts.tv_sec * 1000000000ULL + ts.tv_nsec;
}

static func_stats_t* get_func_stats(const char* name) {
    for (int i = 0; i < g_func_count; i++) {
        if (strcmp(g_func_stats[i].name, name) == 0) {
            return &g_func_stats[i];
        }
    }
    if (g_func_count < MAX_INSTRUMENTED_FUNCS) {
        func_stats_t* stats = &g_func_stats[g_func_count++];
        stats->name = name;
        stats->calls = 0;
        stats->total_ns = 0;
        stats->min_ns = UINT64_MAX;
        stats->max_ns = 0;
        return stats;
    }
    return NULL;
}

#define PROFILE_FUNC_START(name) \
    uint64_t _prof_start_##name = get_time_ns();

#define PROFILE_FUNC_END(name) do { \
    uint64_t _elapsed = get_time_ns() - _prof_start_##name; \
    func_stats_t* _stats = get_func_stats(#name); \
    if (_stats) { \
        _stats->calls++; \
        _stats->total_ns += _elapsed; \
        if (_elapsed < _stats->min_ns) _stats->min_ns = _elapsed; \
        if (_elapsed > _stats->max_ns) _stats->max_ns = _elapsed; \
    } \
} while(0)

// Write instrumentation stats
static void write_func_stats() {
    printf("\n=== Function Statistics ===\n");
    printf("%-30s %10s %10s %10s %10s %10s\n", 
           "Function", "Calls", "Total(ms)", "Avg(ns)", "Min(ns)", "Max(ns)");
    printf("--------------------------------------------------------------------------------\n");
    
    for (int i = 0; i < g_func_count; i++) {
        func_stats_t* stats = &g_func_stats[i];
        double total_ms = stats->total_ns / 1000000.0;
        uint64_t avg_ns = stats->calls > 0 ? stats->total_ns / stats->calls : 0;
        
        printf("%-30s %10llu %10.2f %10llu %10llu %10llu\n",
               stats->name, stats->calls, total_ms, avg_ns, 
               stats->min_ns, stats->max_ns);
    }
}

// Example of instrumented decode function
static void instrumented_decode_test(int iterations) {
    printf("\nRunning instrumented decode test (%d iterations)...\n", iterations);
    
    for (int i = 0; i < iterations; i++) {
        PROFILE_FUNC_START(full_decode);
        
        PROFILE_FUNC_START(decode_fast);
        glint_document_t* doc = glint_decode_fast(
            COMPREHENSIVE_TEST_DATA, 
            COMPREHENSIVE_TEST_DATA_LEN
        );
        PROFILE_FUNC_END(decode_fast);
        
        PROFILE_FUNC_START(document_free);
        glint_document_free_fast(doc);
        PROFILE_FUNC_END(document_free);
        
        PROFILE_FUNC_END(full_decode);
    }
    
    write_func_stats();
}

int main(int argc, char** argv) {
    int iterations = 100000;
    
    if (argc > 1) {
        iterations = atoi(argv[1]);
    }
    
    printf("=== Glint C Decoder Profiler ===\n");
    printf("Iterations: %d\n", iterations);
    printf("Sample rate: %d Hz\n\n", SAMPLE_RATE_HZ);
    
    // Method 1: Sampling profiler (like Go's pprof)
    printf("Starting sampling profiler...\n");
    start_profiling();
    run_decode_workload(iterations);
    stop_profiling();
    
    // Write profiles
    write_pprof_profile("glint_profile.pprof");
    write_flamegraph_profile("glint_profile.folded");
    
    // Method 2: Manual instrumentation
    instrumented_decode_test(iterations);
    
    // Cleanup
    free(g_profile.samples);
    
    printf("\nProfiling complete!\n");
    printf("To visualize:\n");
    printf("  1. For pprof: go tool pprof glint_profile.pprof\n");
    printf("  2. For flamegraph: cat glint_profile.folded | flamegraph.pl > glint.svg\n");
    
    return 0;
}
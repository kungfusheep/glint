/*
 * Hot benchmark for profiling - runs continuously for better profiling data
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <signal.h>
#include <unistd.h>
#include <time.h>
#include <sys/time.h>
#include "../src/glint.h"
#include "../../../test/data/comprehensive_test_data.h"

// External decoder functions
extern glint_document_t* glint_decode_fast(const uint8_t* data, size_t len);
extern void glint_document_free_fast(glint_document_t* doc);
extern glint_document_t* glint_decode_fast_profile(const uint8_t* data, size_t len);
extern void glint_document_free_fast_profile(glint_document_t* doc);
extern glint_document_t* glint_decode(const uint8_t* data, size_t len);
extern void glint_document_free(glint_document_t* doc);

static volatile int g_running = 1;
static uint64_t g_iterations = 0;

void signal_handler(int sig) {
    (void)sig;
    g_running = 0;
}

static uint64_t get_time_ms() {
    struct timeval tv;
    gettimeofday(&tv, NULL);
    return tv.tv_sec * 1000ULL + tv.tv_usec / 1000;
}

// Hot loop that we want to profile
__attribute__((noinline))
static void decode_hot_loop_fast() {
    glint_document_t* doc = glint_decode_fast(
        COMPREHENSIVE_TEST_DATA, 
        COMPREHENSIVE_TEST_DATA_LEN
    );
    glint_document_free_fast(doc);
    g_iterations++;
}

__attribute__((noinline))
static void decode_hot_loop_profile() {
    glint_document_t* doc = glint_decode_fast_profile(
        COMPREHENSIVE_TEST_DATA, 
        COMPREHENSIVE_TEST_DATA_LEN
    );
    glint_document_free_fast_profile(doc);
    g_iterations++;
}

__attribute__((noinline))
static void decode_hot_loop_schema() {
    glint_document_t* doc = glint_decode(
        COMPREHENSIVE_TEST_DATA, 
        COMPREHENSIVE_TEST_DATA_LEN
    );
    glint_document_free(doc);
    g_iterations++;
}

int main(int argc, char** argv) {
    int duration_sec = 10;  // Default 10 seconds
    int decoder_type = 0;    // 0=fast, 1=profile, 2=schema
    
    if (argc > 1) {
        duration_sec = atoi(argv[1]);
    }
    if (argc > 2) {
        if (strcmp(argv[2], "profile") == 0) {
            decoder_type = 1;
        } else if (strcmp(argv[2], "schema") == 0) {
            decoder_type = 2;
        }
    }
    
    // Setup signal handler for clean exit
    signal(SIGINT, signal_handler);
    signal(SIGTERM, signal_handler);
    
    const char* decoder_names[] = {"fast", "profile", "schema"};
    printf("=== Glint Hot Benchmark ===\n");
    printf("Decoder: %s\n", decoder_names[decoder_type]);
    printf("Duration: %d seconds\n", duration_sec);
    printf("Press Ctrl-C to stop early\n\n");
    printf("Running hot...\n");
    
    uint64_t start_time = get_time_ms();
    uint64_t last_report = start_time;
    uint64_t target_end = start_time + (duration_sec * 1000);
    
    // Main hot loop
    while (g_running && get_time_ms() < target_end) {
        // Run in tight batches for better performance
        for (int i = 0; i < 1000; i++) {
            switch (decoder_type) {
                case 0: decode_hot_loop_fast(); break;
                case 1: decode_hot_loop_profile(); break;
                case 2: decode_hot_loop_schema(); break;
            }
        }
        
        // Report progress every second
        uint64_t now = get_time_ms();
        if (now - last_report >= 1000) {
            uint64_t elapsed_sec = (now - start_time) / 1000;
            double rate = g_iterations / (double)elapsed_sec;
            printf("  %llu iterations in %llu sec (%.0f iter/sec)\n", 
                   g_iterations, elapsed_sec, rate);
            last_report = now;
        }
    }
    
    // Final stats
    uint64_t end_time = get_time_ms();
    uint64_t total_ms = end_time - start_time;
    double total_sec = total_ms / 1000.0;
    double rate = g_iterations / total_sec;
    double ns_per_op = (total_ms * 1000000.0) / g_iterations;
    double throughput_mb = (COMPREHENSIVE_TEST_DATA_LEN * g_iterations) / (total_sec * 1024 * 1024);
    
    printf("\n=== Results ===\n");
    printf("Total iterations: %llu\n", g_iterations);
    printf("Total time: %.2f seconds\n", total_sec);
    printf("Rate: %.0f iterations/sec\n", rate);
    printf("Time per decode: %.0f ns\n", ns_per_op);
    printf("Throughput: %.1f MB/s\n", throughput_mb);
    
    return 0;
}
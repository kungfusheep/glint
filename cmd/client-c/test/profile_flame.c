/*
 * Generate flame graph data for the Glint decoder
 * Outputs in folded stack format for flamegraph.pl
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include "../src/glint.h"
#include "../../../test/data/comprehensive_test_data.h"

// Track call stack manually for flame graph generation
#define MAX_STACK_DEPTH 20
typedef struct {
    const char* stack[MAX_STACK_DEPTH];
    int depth;
    uint64_t start_time;
} call_stack_t;

static call_stack_t g_stack = {0};
static FILE* g_trace_file = NULL;

static inline uint64_t get_time_ns() {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return ts.tv_sec * 1000000000ULL + ts.tv_nsec;
}

static void stack_push(const char* func) {
    if (g_stack.depth < MAX_STACK_DEPTH) {
        g_stack.stack[g_stack.depth++] = func;
        g_stack.start_time = get_time_ns();
    }
}

static void stack_pop() {
    if (g_stack.depth > 0) {
        uint64_t duration = get_time_ns() - g_stack.start_time;
        
        // Write stack in flamegraph format: func1;func2;func3 count
        for (int i = 0; i < g_stack.depth; i++) {
            if (i > 0) fprintf(g_trace_file, ";");
            fprintf(g_trace_file, "%s", g_stack.stack[i]);
        }
        fprintf(g_trace_file, " %llu\n", duration / 1000); // Convert to microseconds
        
        g_stack.depth--;
    }
}

// Instrumented decoder functions
static glint_document_t* decode_fast_instrumented(const uint8_t* data, size_t len) {
    stack_push("glint_decode_fast");
    
    // Simulate decoder stages
    stack_push("parse_header");
    // Header parsing work
    stack_pop();
    
    stack_push("cache_lookup");
    // Cache lookup work
    stack_pop();
    
    stack_push("execute_instructions");
    
    // Simulate instruction execution
    for (int i = 0; i < 10; i++) {
        if (i % 3 == 0) {
            stack_push("read_varint");
            stack_pop();
        } else if (i % 3 == 1) {
            stack_push("read_string");
            stack_pop();
        } else {
            stack_push("read_slice");
            stack_push("read_varint");
            stack_pop();
            stack_pop();
        }
    }
    
    stack_pop(); // execute_instructions
    
    extern glint_document_t* glint_decode_fast(const uint8_t*, size_t);
    glint_document_t* result = glint_decode_fast(data, len);
    
    stack_pop(); // glint_decode_fast
    return result;
}

int main(int argc, char** argv) {
    int iterations = 10000;
    if (argc > 1) {
        iterations = atoi(argv[1]);
    }
    
    g_trace_file = fopen("glint_flame.folded", "w");
    if (!g_trace_file) {
        perror("Failed to open trace file");
        return 1;
    }
    
    printf("Generating flame graph data for %d iterations...\n", iterations);
    
    // Run instrumented decodes
    for (int i = 0; i < iterations; i++) {
        stack_push("main");
        stack_push("decode_loop");
        
        glint_document_t* doc = decode_fast_instrumented(
            COMPREHENSIVE_TEST_DATA, 
            COMPREHENSIVE_TEST_DATA_LEN
        );
        
        stack_push("free_document");
        extern void glint_document_free_fast(glint_document_t*);
        glint_document_free_fast(doc);
        stack_pop();
        
        stack_pop(); // decode_loop
        stack_pop(); // main
        
        if ((i + 1) % 1000 == 0) {
            printf("  %d iterations completed\n", i + 1);
        }
    }
    
    fclose(g_trace_file);
    
    printf("\nFlame graph data written to glint_flame.folded\n");
    printf("To generate SVG flame graph:\n");
    printf("  1. Install flamegraph.pl:\n");
    printf("     git clone https://github.com/brendangregg/FlameGraph\n");
    printf("  2. Generate SVG:\n");
    printf("     cat glint_flame.folded | ./FlameGraph/flamegraph.pl > glint_flame.svg\n");
    printf("  3. Open glint_flame.svg in a browser\n");
    
    return 0;
}
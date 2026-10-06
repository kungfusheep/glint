/*
 * Glint C Decoder Profiling with Google Performance Tools (gperftools)
 * This is the de facto standard for C/C++ profiling, similar to Go's pprof
 * 
 * To use:
 * 1. Install gperftools: brew install gperftools (macOS) or apt-get install google-perftools (Linux)
 * 2. Compile: cc -O2 -g profile_gperftools.c -lprofiler -ltcmalloc
 * 3. Run: CPUPROFILE=glint.prof ./a.out
 * 4. Analyze: pprof --pdf ./a.out glint.prof > profile.pdf
 *           or: pprof --web ./a.out glint.prof
 */

#include <stdio.h>
#include <stdlib.h>
#include <gperftools/profiler.h>
#include "../src/glint.h"
#include "../../../test/data/comprehensive_test_data.h"

// External decoder functions
extern glint_document_t* glint_decode_fast(const uint8_t* data, size_t len);
extern void glint_document_free_fast(glint_document_t* doc);

static void run_decode_benchmark(int iterations) {
    for (int i = 0; i < iterations; i++) {
        glint_document_t* doc = glint_decode_fast(
            COMPREHENSIVE_TEST_DATA, 
            COMPREHENSIVE_TEST_DATA_LEN
        );
        glint_document_free_fast(doc);
    }
}

int main(int argc, char** argv) {
    int iterations = 1000000;
    if (argc > 1) {
        iterations = atoi(argv[1]);
    }
    
    printf("Running %d iterations with gperftools profiling...\n", iterations);
    
    // Start CPU profiling
    ProfilerStart("glint_cpu.prof");
    
    run_decode_benchmark(iterations);
    
    // Stop profiling
    ProfilerStop();
    
    printf("Profile written to glint_cpu.prof\n");
    printf("Analyze with: pprof --pdf %s glint_cpu.prof > profile.pdf\n", argv[0]);
    printf("         or: pprof --web %s glint_cpu.prof\n", argv[0]);
    printf("         or: pprof --text %s glint_cpu.prof\n", argv[0]);
    
    return 0;
}
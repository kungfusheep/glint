/*
 * Glint C Decoder Profiling with gprof
 * gprof is the traditional Unix profiler, available on most systems
 * 
 * To use:
 * 1. Compile with -pg flag: cc -O2 -pg profile_gprof.c src/*.c -o profile_gprof
 * 2. Run the program: ./profile_gprof
 * 3. Analyze: gprof profile_gprof gmon.out > analysis.txt
 *           or: gprof profile_gprof gmon.out | gprof2dot | dot -Tsvg -o profile.svg
 */

#include <stdio.h>
#include <stdlib.h>
#include "../src/glint.h"
#include "../../../test/data/comprehensive_test_data.h"

// External decoder functions
extern glint_document_t* glint_decode_fast(const uint8_t* data, size_t len);
extern void glint_document_free_fast(glint_document_t* doc);

// Mark as no inline to ensure it shows up in profile
__attribute__((noinline))
static void decode_one() {
    glint_document_t* doc = glint_decode_fast(
        COMPREHENSIVE_TEST_DATA, 
        COMPREHENSIVE_TEST_DATA_LEN
    );
    glint_document_free_fast(doc);
}

__attribute__((noinline))
static void run_batch(int count) {
    for (int i = 0; i < count; i++) {
        decode_one();
    }
}

int main(int argc, char** argv) {
    int iterations = 100000;
    if (argc > 1) {
        iterations = atoi(argv[1]);
    }
    
    printf("Running %d iterations for gprof profiling...\n", iterations);
    printf("(Compile with -pg flag for profiling)\n\n");
    
    // Run in batches to get better call graph
    int batch_size = 1000;
    int batches = iterations / batch_size;
    
    for (int i = 0; i < batches; i++) {
        run_batch(batch_size);
        if ((i + 1) % 10 == 0) {
            printf("  Completed %d/%d batches\n", i + 1, batches);
        }
    }
    
    printf("\nProfile data written to gmon.out\n");
    printf("Analyze with:\n");
    printf("  gprof %s gmon.out > analysis.txt\n", argv[0]);
    printf("  gprof %s gmon.out | less\n", argv[0]);
    
    return 0;
}
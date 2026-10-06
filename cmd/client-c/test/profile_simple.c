/*
 * Simple profiling test for the instrumented decoder
 */

#include <stdio.h>
#include <stdlib.h>
#include "../src/glint.h"
#include "../../../test/data/comprehensive_test_data.h"

// External functions from the profiling decoder
extern glint_document_t* glint_decode_fast_prof(const uint8_t* data, size_t len);
extern void glint_document_free_fast_prof(glint_document_t* doc);
extern void glint_print_prof_stats(void);
extern void glint_reset_prof_stats(void);

int main(int argc, char** argv) {
    int iterations = 100000;
    if (argc > 1) {
        iterations = atoi(argv[1]);
    }
    
    printf("=== Glint Decoder Profiling - %d iterations ===\n\n", iterations);
    
    // Reset stats
    glint_reset_prof_stats();
    
    // Run decodes
    for (int i = 0; i < iterations; i++) {
        glint_document_t* doc = glint_decode_fast_prof(
            COMPREHENSIVE_TEST_DATA, 
            COMPREHENSIVE_TEST_DATA_LEN
        );
        glint_document_free_fast_prof(doc);
        
        if ((i + 1) % 10000 == 0) {
            printf("  %d iterations completed\n", i + 1);
        }
    }
    
    // Print profiling results
    glint_print_prof_stats();
    
    return 0;
}
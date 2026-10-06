/*
 * Test to verify the fast decoder correctly decodes actual values
 * Not just consuming bytes, but producing correct results
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <stddef.h>
#include <math.h>
#include "../src/glint.h"
#include "../src/glint_reader.h"
#include "../src/glint_slice.h"
#include "../src/glint_field_map.h"

// External decoder functions
extern glint_document_t* glint_decode_fast(const uint8_t* data, size_t len);
extern glint_document_t* glint_decode_fast_with_registry(const uint8_t* data, size_t len,
                                                         const glint_field_registry_t* field_registry,
                                                         void* output_struct);
extern void glint_document_free_fast(glint_document_t* doc);

// Simple test structure matching our test data
typedef struct {
    bool bool_value;
    int32_t int_value;
    uint32_t uint_value;
    float float_value;
    double double_value;
    glint_string_view_t string_value;
    glint_int32_slice_t int_slice;
    glint_float32_slice_t float_slice;
    glint_string_slice_t string_slice;
} test_struct_t;

// Define field mappings for test_struct_t
static const glint_field_map_t test_struct_fields[] = {
    GLINT_FIELD(test_struct_t, bool_value, GLINT_WIRE_BOOL),
    GLINT_FIELD(test_struct_t, int_value, GLINT_WIRE_INT32),
    GLINT_FIELD(test_struct_t, uint_value, GLINT_WIRE_UINT32),
    GLINT_FIELD(test_struct_t, float_value, GLINT_WIRE_FLOAT32),
    GLINT_FIELD(test_struct_t, double_value, GLINT_WIRE_FLOAT64),
    GLINT_FIELD(test_struct_t, string_value, GLINT_WIRE_STRING),
    GLINT_FIELD(test_struct_t, int_slice, GLINT_WIRE_INT32 | GLINT_WIRE_SLICE_FLAG),
    GLINT_FIELD(test_struct_t, float_slice, GLINT_WIRE_FLOAT32 | GLINT_WIRE_SLICE_FLAG),
    GLINT_FIELD(test_struct_t, string_slice, GLINT_WIRE_STRING | GLINT_WIRE_SLICE_FLAG),
};

static const glint_field_registry_t test_struct_registry = {
    .fields = test_struct_fields,
    .field_count = sizeof(test_struct_fields) / sizeof(test_struct_fields[0])
};

// Create test data with known values
void create_test_data(uint8_t** out_data, size_t* out_len) {
    // For now, let's create a simple hand-crafted test
    // Format: flags(1) + hash(4) + schema_len(varint) + schema + data
    
    static uint8_t test_data[500];
    size_t pos = 0;
    
    // Header
    test_data[pos++] = 0x00;  // flags
    
    // Fake hash
    test_data[pos++] = 0x11;
    test_data[pos++] = 0x22;
    test_data[pos++] = 0x33;
    test_data[pos++] = 0x44;
    
    // Schema size (we'll fill this in later)
    size_t schema_size_pos = pos;
    test_data[pos++] = 0;  // placeholder
    
    size_t schema_start = pos;
    
    // Schema fields:
    // bool_value
    test_data[pos++] = 0x01;  // WireBool
    test_data[pos++] = 10;     // name length
    memcpy(&test_data[pos], "bool_value", 10);
    pos += 10;
    
    // int_value
    test_data[pos++] = 0x05;  // WireInt32
    test_data[pos++] = 9;
    memcpy(&test_data[pos], "int_value", 9);
    pos += 9;
    
    // uint_value
    test_data[pos++] = 0x0A;  // WireUint32
    test_data[pos++] = 10;
    memcpy(&test_data[pos], "uint_value", 10);
    pos += 10;
    
    // float_value
    test_data[pos++] = 0x0C;  // WireFloat32
    test_data[pos++] = 11;
    memcpy(&test_data[pos], "float_value", 11);
    pos += 11;
    
    // double_value
    test_data[pos++] = 0x0D;  // WireFloat64
    test_data[pos++] = 12;
    memcpy(&test_data[pos], "double_value", 12);
    pos += 12;
    
    // string_value
    test_data[pos++] = 0x0E;  // WireString
    test_data[pos++] = 12;
    memcpy(&test_data[pos], "string_value", 12);
    pos += 12;
    
    // int_slice (slice of int32)
    test_data[pos++] = 0x25;  // WireInt32 | WireSliceFlag
    test_data[pos++] = 9;
    memcpy(&test_data[pos], "int_slice", 9);
    pos += 9;
    
    // float_slice
    test_data[pos++] = 0x2C;  // WireFloat32 | WireSliceFlag
    test_data[pos++] = 11;
    memcpy(&test_data[pos], "float_slice", 11);
    pos += 11;
    
    // string_slice
    test_data[pos++] = 0x2E;  // WireString | WireSliceFlag
    test_data[pos++] = 12;
    memcpy(&test_data[pos], "string_slice", 12);
    pos += 12;
    
    // Update schema size
    size_t schema_size = pos - schema_start;
    test_data[schema_size_pos] = schema_size;
    
    // Now add the actual data values
    
    // bool_value = true
    test_data[pos++] = 0x01;
    
    // int_value = -42 (zigzag encoded)
    test_data[pos++] = 83;  // zigzag(-42) = 83
    
    // uint_value = 12345
    test_data[pos++] = 0xB9;
    test_data[pos++] = 0x60;
    
    // float_value = 3.14159 (as varint of uint32 bits)
    // float bits: 0x40490FDB
    test_data[pos++] = 0xDB;
    test_data[pos++] = 0x9F;
    test_data[pos++] = 0xA4;
    test_data[pos++] = 0x82;
    test_data[pos++] = 0x04;
    
    // double_value = 3.14 (as varint of uint64 bits)
    // double bits for 3.14: 0x40091EB851EB851F
    test_data[pos++] = 0x9F;
    test_data[pos++] = 0x8A;
    test_data[pos++] = 0xAE;
    test_data[pos++] = 0x8F;
    test_data[pos++] = 0x85;
    test_data[pos++] = 0xD7;
    test_data[pos++] = 0xC7;
    test_data[pos++] = 0x84;
    test_data[pos++] = 0x40;
    
    // string_value = "Hello"
    test_data[pos++] = 5;  // length
    memcpy(&test_data[pos], "Hello", 5);
    pos += 5;
    
    // int_slice = [1, -2, 3]
    test_data[pos++] = 3;  // count
    test_data[pos++] = 2;  // zigzag(1) = 2
    test_data[pos++] = 3;  // zigzag(-2) = 3
    test_data[pos++] = 6;  // zigzag(3) = 6
    
    // float_slice = [1.0, 2.0]
    test_data[pos++] = 2;  // count
    // 1.0 = 0x3F800000
    test_data[pos++] = 0x80;
    test_data[pos++] = 0x80;
    test_data[pos++] = 0x80;
    test_data[pos++] = 0xFC;
    test_data[pos++] = 0x03;
    // 2.0 = 0x40000000
    test_data[pos++] = 0x80;
    test_data[pos++] = 0x80;
    test_data[pos++] = 0x80;
    test_data[pos++] = 0x80;
    test_data[pos++] = 0x04;
    
    // string_slice = ["A", "B"]
    test_data[pos++] = 2;  // count
    test_data[pos++] = 1;  // length of "A"
    test_data[pos++] = 'A';
    test_data[pos++] = 1;  // length of "B"
    test_data[pos++] = 'B';
    
    *out_data = test_data;
    *out_len = pos;
}

int main() {
    printf("=== Fast Decoder Value Verification Test ===\n\n");
    
    uint8_t* test_data;
    size_t test_len;
    create_test_data(&test_data, &test_len);
    
    printf("Created test data: %zu bytes\n", test_len);
    
    // Create a test_struct_t to decode into
    test_struct_t result = {0};
    
    // Decode the data with field registry
    glint_document_t* doc = glint_decode_fast_with_registry(test_data, test_len, 
                                                           &test_struct_registry, &result);
    
    if (!doc) {
        printf("❌ Failed to decode test data\n");
        return 1;
    }
    
    printf("✅ Successfully decoded document\n");
    printf("  Schema CRC32: 0x%08x\n\n", doc->schema_crc32);
    
    printf("=== Checking Decoded Values ===\n");
    
    // Check bool
    printf("bool_value: %s", result.bool_value ? "true" : "false");
    if (result.bool_value == true) {
        printf(" ✅\n");
    } else {
        printf(" ❌ (expected true)\n");
    }
    
    // Check int
    printf("int_value: %d", result.int_value);
    if (result.int_value == -42) {
        printf(" ✅\n");
    } else {
        printf(" ❌ (expected -42)\n");
    }
    
    // Check uint
    printf("uint_value: %u", result.uint_value);
    if (result.uint_value == 12345) {
        printf(" ✅\n");
    } else {
        printf(" ❌ (expected 12345)\n");
    }
    
    // Check float
    printf("float_value: %f", result.float_value);
    if (fabs(result.float_value - 3.14159f) < 0.00001f) {
        printf(" ✅\n");
    } else {
        printf(" ❌ (expected 3.14159)\n");
    }
    
    // Check double
    printf("double_value: %f", result.double_value);
    if (fabs(result.double_value - 3.14) < 0.01) {
        printf(" ✅\n");
    } else {
        printf(" ❌ (expected 3.14)\n");
    }
    
    // Check string
    printf("string_value: \"");
    if (result.string_value.data && result.string_value.len > 0) {
        printf("%.*s\"", (int)result.string_value.len, result.string_value.data);
        if (result.string_value.len == 5 && 
            memcmp(result.string_value.data, "Hello", 5) == 0) {
            printf(" ✅\n");
        } else {
            printf(" ❌ (expected \"Hello\")\n");
        }
    } else {
        printf("NULL\" ❌\n");
    }
    
    // Check int slice
    printf("int_slice: [");
    if (result.int_slice.data && result.int_slice.len > 0) {
        for (size_t i = 0; i < result.int_slice.len; i++) {
            if (i > 0) printf(", ");
            printf("%d", result.int_slice.data[i]);
        }
        printf("]");
        if (result.int_slice.len == 3 &&
            result.int_slice.data[0] == 1 &&
            result.int_slice.data[1] == -2 &&
            result.int_slice.data[2] == 3) {
            printf(" ✅\n");
        } else {
            printf(" ❌ (expected [1, -2, 3])\n");
        }
    } else {
        printf("] ❌ (empty or null)\n");
    }
    
    // Check float slice
    printf("float_slice: [");
    if (result.float_slice.data && result.float_slice.len > 0) {
        for (size_t i = 0; i < result.float_slice.len; i++) {
            if (i > 0) printf(", ");
            printf("%.1f", result.float_slice.data[i]);
        }
        printf("]");
        if (result.float_slice.len == 2 &&
            fabs(result.float_slice.data[0] - 1.0f) < 0.01f &&
            fabs(result.float_slice.data[1] - 2.0f) < 0.01f) {
            printf(" ✅\n");
        } else {
            printf(" ❌ (expected [1.0, 2.0])\n");
        }
    } else {
        printf("] ❌ (empty or null)\n");
    }
    
    // Check string slice
    printf("string_slice: [");
    if (result.string_slice.data && result.string_slice.len > 0) {
        for (size_t i = 0; i < result.string_slice.len; i++) {
            if (i > 0) printf(", ");
            printf("\"%.*s\"", (int)result.string_slice.data[i].len, 
                   result.string_slice.data[i].data);
        }
        printf("]");
        if (result.string_slice.len == 2 &&
            result.string_slice.data[0].len == 1 &&
            result.string_slice.data[0].data[0] == 'A' &&
            result.string_slice.data[1].len == 1 &&
            result.string_slice.data[1].data[0] == 'B') {
            printf(" ✅\n");
        } else {
            printf(" ❌ (expected [\"A\", \"B\"])\n");
        }
    } else {
        printf("] ❌ (empty or null)\n");
    }
    
    printf("\n=== Summary ===\n");
    printf("Test completed. Using field registry for proper struct mapping.\n");
    
    glint_document_free_fast(doc);
    return 0;
}
/*
 * Quick verification test - pretty print and hex dump decoded structs
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

extern glint_document_t* glint_decode_fast_with_registry(const uint8_t* data, size_t len,
                                                         const glint_field_registry_t* field_registry,
                                                         void* output_struct);
extern void glint_document_free_fast(glint_document_t* doc);

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

// Hex dump of memory
void hex_dump(const char* label, const void* ptr, size_t len) {
    const uint8_t* p = (const uint8_t*)ptr;
    printf("\n%s (%zu bytes):\n", label, len);
    for (size_t i = 0; i < len; i++) {
        if (i % 16 == 0) printf("%04zx: ", i);
        printf("%02x ", p[i]);
        if (i % 16 == 15 || i == len - 1) {
            // Print ASCII representation
            int pad = 15 - (i % 16);
            for (int j = 0; j < pad; j++) printf("   ");
            printf(" |");
            for (size_t j = i - (i % 16); j <= i; j++) {
                printf("%c", (p[j] >= 32 && p[j] < 127) ? p[j] : '.');
            }
            printf("|\n");
        }
    }
}

// Pretty print the struct
void print_struct(const test_struct_t* s) {
    printf("\n=== Decoded Struct Contents ===\n");
    printf("bool_value:   %s\n", s->bool_value ? "true" : "false");
    printf("int_value:    %d\n", s->int_value);
    printf("uint_value:   %u\n", s->uint_value);
    printf("float_value:  %f\n", s->float_value);
    printf("double_value: %f\n", s->double_value);
    
    printf("string_value: \"");
    if (s->string_value.data && s->string_value.len > 0) {
        printf("%.*s", (int)s->string_value.len, s->string_value.data);
    }
    printf("\"\n");
    
    printf("int_slice:    [");
    if (s->int_slice.data && s->int_slice.len > 0) {
        for (size_t i = 0; i < s->int_slice.len; i++) {
            if (i > 0) printf(", ");
            printf("%d", s->int_slice.data[i]);
        }
    }
    printf("]\n");
    
    printf("float_slice:  [");
    if (s->float_slice.data && s->float_slice.len > 0) {
        for (size_t i = 0; i < s->float_slice.len; i++) {
            if (i > 0) printf(", ");
            printf("%.1f", s->float_slice.data[i]);
        }
    }
    printf("]\n");
    
    printf("string_slice: [");
    if (s->string_slice.data && s->string_slice.len > 0) {
        for (size_t i = 0; i < s->string_slice.len; i++) {
            if (i > 0) printf(", ");
            printf("\"%.*s\"", (int)s->string_slice.data[i].len, 
                   s->string_slice.data[i].data);
        }
    }
    printf("]\n");
}

// Create known test data
void create_test_data(uint8_t** out_data, size_t* out_len) {
    static uint8_t test_data[500];
    size_t pos = 0;
    
    // Header
    test_data[pos++] = 0x00;  // flags
    test_data[pos++] = 0x11; test_data[pos++] = 0x22; 
    test_data[pos++] = 0x33; test_data[pos++] = 0x44; // hash
    
    size_t schema_size_pos = pos;
    test_data[pos++] = 0;  // placeholder for schema size
    
    size_t schema_start = pos;
    
    // Schema fields
    test_data[pos++] = 0x01;  // WireBool
    test_data[pos++] = 10;
    memcpy(&test_data[pos], "bool_value", 10);
    pos += 10;
    
    test_data[pos++] = 0x05;  // WireInt32
    test_data[pos++] = 9;
    memcpy(&test_data[pos], "int_value", 9);
    pos += 9;
    
    test_data[pos++] = 0x0A;  // WireUint32
    test_data[pos++] = 10;
    memcpy(&test_data[pos], "uint_value", 10);
    pos += 10;
    
    test_data[pos++] = 0x0C;  // WireFloat32
    test_data[pos++] = 11;
    memcpy(&test_data[pos], "float_value", 11);
    pos += 11;
    
    test_data[pos++] = 0x0D;  // WireFloat64
    test_data[pos++] = 12;
    memcpy(&test_data[pos], "double_value", 12);
    pos += 12;
    
    test_data[pos++] = 0x0E;  // WireString
    test_data[pos++] = 12;
    memcpy(&test_data[pos], "string_value", 12);
    pos += 12;
    
    test_data[pos++] = 0x25;  // WireInt32 | WireSliceFlag
    test_data[pos++] = 9;
    memcpy(&test_data[pos], "int_slice", 9);
    pos += 9;
    
    test_data[pos++] = 0x2C;  // WireFloat32 | WireSliceFlag
    test_data[pos++] = 11;
    memcpy(&test_data[pos], "float_slice", 11);
    pos += 11;
    
    test_data[pos++] = 0x2E;  // WireString | WireSliceFlag
    test_data[pos++] = 12;
    memcpy(&test_data[pos], "string_slice", 12);
    pos += 12;
    
    size_t schema_size = pos - schema_start;
    test_data[schema_size_pos] = schema_size;
    
    // Data values
    test_data[pos++] = 0x01;  // bool = true
    test_data[pos++] = 83;     // int = -42 (zigzag)
    test_data[pos++] = 0xB9; test_data[pos++] = 0x60;  // uint = 12345
    
    // float = 3.14159
    test_data[pos++] = 0xDB; test_data[pos++] = 0x9F;
    test_data[pos++] = 0xA4; test_data[pos++] = 0x82; test_data[pos++] = 0x04;
    
    // double = 3.14
    test_data[pos++] = 0x9F; test_data[pos++] = 0x8A;
    test_data[pos++] = 0xAE; test_data[pos++] = 0x8F;
    test_data[pos++] = 0x85; test_data[pos++] = 0xD7;
    test_data[pos++] = 0xC7; test_data[pos++] = 0x84; test_data[pos++] = 0x40;
    
    // string = "Hello"
    test_data[pos++] = 5;
    memcpy(&test_data[pos], "Hello", 5);
    pos += 5;
    
    // int_slice = [1, -2, 3]
    test_data[pos++] = 3;
    test_data[pos++] = 2; test_data[pos++] = 3; test_data[pos++] = 6;
    
    // float_slice = [1.0, 2.0]
    test_data[pos++] = 2;
    test_data[pos++] = 0x80; test_data[pos++] = 0x80;
    test_data[pos++] = 0x80; test_data[pos++] = 0xFC; test_data[pos++] = 0x03;
    test_data[pos++] = 0x80; test_data[pos++] = 0x80;
    test_data[pos++] = 0x80; test_data[pos++] = 0x80; test_data[pos++] = 0x04;
    
    // string_slice = ["A", "B"]
    test_data[pos++] = 2;
    test_data[pos++] = 1; test_data[pos++] = 'A';
    test_data[pos++] = 1; test_data[pos++] = 'B';
    
    *out_data = test_data;
    *out_len = pos;
}

int main() {
    printf("=== Glint Decoder Verification Test ===\n");
    
    uint8_t* test_data;
    size_t test_len;
    create_test_data(&test_data, &test_len);
    
    printf("\nInput data: %zu bytes\n", test_len);
    hex_dump("Raw input bytes", test_data, test_len);
    
    // Create expected struct
    test_struct_t expected = {
        .bool_value = true,
        .int_value = -42,
        .uint_value = 12345,
        .float_value = 3.14159f,
        .double_value = 3.14,
        // Slices will be filled by decoder
    };
    
    printf("\n=== Expected Values ===\n");
    printf("bool_value:   true\n");
    printf("int_value:    -42\n");
    printf("uint_value:   12345\n");
    printf("float_value:  3.14159\n");
    printf("double_value: 3.14\n");
    printf("string_value: \"Hello\"\n");
    printf("int_slice:    [1, -2, 3]\n");
    printf("float_slice:  [1.0, 2.0]\n");
    printf("string_slice: [\"A\", \"B\"]\n");
    
    // Decode
    test_struct_t result = {0};
    glint_document_t* doc = glint_decode_fast_with_registry(test_data, test_len, 
                                                           &test_struct_registry, &result);
    
    if (!doc) {
        printf("\n❌ Failed to decode\n");
        return 1;
    }
    
    // Pretty print decoded struct
    print_struct(&result);
    
    // Hex dump the struct memory
    hex_dump("Decoded struct memory", &result, sizeof(result));
    
    // Verify each field
    printf("\n=== Field-by-Field Verification ===\n");
    printf("bool_value:   %s\n", result.bool_value == expected.bool_value ? "✅" : "❌");
    printf("int_value:    %s\n", result.int_value == expected.int_value ? "✅" : "❌");
    printf("uint_value:   %s\n", result.uint_value == expected.uint_value ? "✅" : "❌");
    printf("float_value:  %s\n", fabs(result.float_value - expected.float_value) < 0.00001 ? "✅" : "❌");
    printf("double_value: %s\n", fabs(result.double_value - expected.double_value) < 0.01 ? "✅" : "❌");
    printf("string_value: %s\n", (result.string_value.len == 5 && 
                                 memcmp(result.string_value.data, "Hello", 5) == 0) ? "✅" : "❌");
    printf("int_slice:    %s\n", (result.int_slice.len == 3 &&
                                 result.int_slice.data[0] == 1 &&
                                 result.int_slice.data[1] == -2 &&
                                 result.int_slice.data[2] == 3) ? "✅" : "❌");
    printf("float_slice:  %s\n", (result.float_slice.len == 2 &&
                                 fabs(result.float_slice.data[0] - 1.0f) < 0.01 &&
                                 fabs(result.float_slice.data[1] - 2.0f) < 0.01) ? "✅" : "❌");
    printf("string_slice: %s\n", (result.string_slice.len == 2 &&
                                 result.string_slice.data[0].len == 1 &&
                                 result.string_slice.data[0].data[0] == 'A' &&
                                 result.string_slice.data[1].len == 1 &&
                                 result.string_slice.data[1].data[0] == 'B') ? "✅" : "❌");
    
    glint_document_free_fast(doc);
    return 0;
}
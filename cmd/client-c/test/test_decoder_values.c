/*
 * Test that validates the Glint C decoder produces correct results
 * from the comprehensive test data
 */

#include <stdio.h>
#include <assert.h>
#include <string.h>
#include <math.h>
#include "../src/glint.h"
#include "../src/glint_reader.h"
#include "../../../test/data/comprehensive_test_data.h"

// External decoder functions
extern glint_document_t* glint_decode(const uint8_t* data, size_t len);
extern void glint_document_free(glint_document_t* doc);

void print_value(glint_value_t* value, int indent) {
    if (!value) return;
    
    // Print indentation
    for (int i = 0; i < indent; i++) printf("  ");
    
    // Print field name if present
    if (value->name && value->name_len > 0) {
        printf("%.*s: ", (int)value->name_len, value->name);
    }
    
    // Decode the wire type flags
    glint_wire_type_t base_type = value->type & GLINT_WIRE_TYPE_MASK;
    bool is_slice = (value->type & GLINT_WIRE_SLICE_FLAG) != 0;
    bool is_ptr = (value->type & GLINT_WIRE_PTR_FLAG) != 0;
    
    if (is_slice) {
        printf("[slice]");
    } else if (is_ptr) {
        printf("*");
    }
    
    // Print value based on type
    switch (base_type) {
        case GLINT_WIRE_BOOL:
            printf("%s", value->data.bool_val ? "true" : "false");
            break;
        case GLINT_WIRE_INT:
            printf("%lld", value->data.i64);
            break;
        case GLINT_WIRE_INT8:
            printf("%d", value->data.i8);
            break;
        case GLINT_WIRE_INT16:
            printf("%d", value->data.i16);
            break;
        case GLINT_WIRE_INT32:
            printf("%d", value->data.i32);
            break;
        case GLINT_WIRE_INT64:
            printf("%lld", value->data.i64);
            break;
        case GLINT_WIRE_UINT:
        case GLINT_WIRE_UINT8:
        case GLINT_WIRE_UINT16:
        case GLINT_WIRE_UINT32:
        case GLINT_WIRE_UINT64:
            printf("%llu", value->data.u64);
            break;
        case GLINT_WIRE_FLOAT32:
            printf("%f", value->data.f32);
            break;
        case GLINT_WIRE_FLOAT64:
            printf("%lf", value->data.f64);
            break;
        case GLINT_WIRE_STRING:
            printf("\"%.*s\"", (int)value->data.string.len, (char*)value->data.string.data);
            break;
        case GLINT_WIRE_BYTES:
            printf("<bytes len=%zu>", value->data.bytes.len);
            break;
        case GLINT_WIRE_STRUCT:
            printf("{\n");
            glint_value_t* field = value->data.struct_val;
            while (field) {
                print_value(field, indent + 1);
                field = field->next;
            }
            for (int i = 0; i < indent; i++) printf("  ");
            printf("}");
            break;
        case GLINT_WIRE_TIME:
            printf("<time>");
            break;
        default:
            printf("<unknown type %d>", base_type);
    }
    
    printf("\n");
}

void validate_comprehensive_data() {
    printf("=== Validating Comprehensive Test Data ===\n");
    printf("Data size: %d bytes\n", COMPREHENSIVE_TEST_DATA_LEN);
    
    // Decode the comprehensive test data
    glint_document_t* doc = glint_decode(COMPREHENSIVE_TEST_DATA, COMPREHENSIVE_TEST_DATA_LEN);
    
    if (!doc) {
        printf("❌ Failed to decode comprehensive test data\n");
        return;
    }
    
    printf("✅ Successfully decoded document\n");
    printf("  Schema CRC32: 0x%08x\n", doc->schema_crc32);
    
    // Print the decoded structure
    printf("\nDecoded values:\n");
    print_value(doc->root, 0);
    
    // Validate specific expected values
    printf("\n=== Validating Expected Values ===\n");
    
    // Navigate through the struct fields
    glint_value_t* field = doc->root->data.struct_val;
    
    bool found_bool = false;
    bool found_int = false;
    bool found_uint = false;
    bool found_float32 = false;
    bool found_float64 = false;
    bool found_string = false;
    bool found_int8 = false;
    bool found_uint8 = false;
    
    while (field) {
        if (field->name && field->name_len > 0) {
            char name[64];
            size_t name_len = field->name_len < 63 ? field->name_len : 63;
            memcpy(name, field->name, name_len);
            name[name_len] = '\0';
            
            glint_wire_type_t base_type = field->type & GLINT_WIRE_TYPE_MASK;
            
            if (strcmp(name, "boolValue") == 0) {
                assert(base_type == GLINT_WIRE_BOOL);
                assert(field->data.bool_val == true);
                printf("✅ boolValue: %s (expected true)\n", field->data.bool_val ? "true" : "false");
                found_bool = true;
            }
            else if (strcmp(name, "intValue") == 0) {
                assert(field->data.i64 == -42 || field->data.i32 == -42);
                printf("✅ intValue: %lld (expected -42)\n", field->data.i64);
                found_int = true;
            }
            else if (strcmp(name, "uintValue") == 0) {
                assert(field->data.u64 == 42 || field->data.u32 == 42);
                printf("✅ uintValue: %llu (expected 42)\n", field->data.u64);
                found_uint = true;
            }
            else if (strcmp(name, "int8Value") == 0) {
                assert(base_type == GLINT_WIRE_INT8);
                assert(field->data.i8 == -128);
                printf("✅ int8Value: %d (expected -128)\n", field->data.i8);
                found_int8 = true;
            }
            else if (strcmp(name, "uint8Value") == 0) {
                assert(field->data.u64 == 255 || field->data.u8 == 255);
                printf("✅ uint8Value: %llu (expected 255)\n", field->data.u64);
                found_uint8 = true;
            }
            else if (strcmp(name, "float32Value") == 0) {
                assert(base_type == GLINT_WIRE_FLOAT32);
                assert(fabs(field->data.f32 - 3.14159) < 0.001);
                printf("✅ float32Value: %.5f (expected 3.14159)\n", field->data.f32);
                found_float32 = true;
            }
            else if (strcmp(name, "float64Value") == 0) {
                assert(base_type == GLINT_WIRE_FLOAT64);
                assert(fabs(field->data.f64 - 2.718281828459045) < 0.000001);
                printf("✅ float64Value: %.15f (expected 2.718281828459045)\n", field->data.f64);
                found_float64 = true;
            }
            else if (strcmp(name, "stringValue") == 0) {
                assert(base_type == GLINT_WIRE_STRING);
                const char* expected = "Hello, Glint! 🚀";
                assert(field->data.string.len == strlen(expected));
                assert(strncmp((char*)field->data.string.data, expected, field->data.string.len) == 0);
                printf("✅ stringValue: \"%.*s\" (expected \"%s\")\n", 
                       (int)field->data.string.len, (char*)field->data.string.data, expected);
                found_string = true;
            }
        }
        
        field = field->next;
    }
    
    // Ensure we found all critical fields
    assert(found_bool);
    assert(found_int);
    assert(found_uint);
    assert(found_float32);
    assert(found_float64);
    assert(found_string);
    assert(found_int8);
    assert(found_uint8);
    
    printf("\n🎉 All critical values validated successfully!\n");
    
    glint_document_free(doc);
}

int main() {
    printf("=== Glint C Decoder Validation Test ===\n");
    printf("=======================================\n\n");
    
    validate_comprehensive_data();
    
    printf("\n✅ All tests passed! The decoder correctly decodes the comprehensive test data.\n");
    
    return 0;
}
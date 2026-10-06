#ifndef GLINT_H
#define GLINT_H

#include <stdint.h>
#include <stddef.h>
#include <stdbool.h>

#ifdef __cplusplus
extern "C" {
#endif

// Wire types matching Go implementation
typedef enum {
    GLINT_WIRE_BOOL    = 1,
    GLINT_WIRE_INT     = 2,
    GLINT_WIRE_INT8    = 3,
    GLINT_WIRE_INT16   = 4,
    GLINT_WIRE_INT32   = 5,
    GLINT_WIRE_INT64   = 6,
    GLINT_WIRE_UINT    = 7,
    GLINT_WIRE_UINT8   = 8,
    GLINT_WIRE_UINT16  = 9,
    GLINT_WIRE_UINT32  = 10,
    GLINT_WIRE_UINT64  = 11,
    GLINT_WIRE_FLOAT32 = 12,
    GLINT_WIRE_FLOAT64 = 13,
    GLINT_WIRE_STRING  = 14,
    GLINT_WIRE_BYTES   = 15,
    GLINT_WIRE_STRUCT  = 16,
    GLINT_WIRE_MAP     = 17,
    GLINT_WIRE_TIME    = 18,
    
    GLINT_WIRE_TYPE_MASK = 0x1F,
    GLINT_WIRE_SLICE_FLAG = 1 << 5,
    GLINT_WIRE_PTR_FLAG = 1 << 6,
    GLINT_WIRE_DELTA_FLAG = 1 << 7
} glint_wire_type_t;

// Error codes
typedef enum {
    GLINT_OK = 0,
    GLINT_ERR_INVALID_DATA = -1,
    GLINT_ERR_OUT_OF_MEMORY = -2,
    GLINT_ERR_BUFFER_TOO_SMALL = -3,
    GLINT_ERR_SCHEMA_MISMATCH = -4,
    GLINT_ERR_LIMIT_EXCEEDED = -5,
    GLINT_ERR_UNSUPPORTED = -6
} glint_error_t;

// String view for zero-copy strings
typedef struct {
    const uint8_t* data;
    size_t len;
} glint_string_view_t;

// Value types
typedef union {
    bool bool_val;
    int8_t i8;
    int16_t i16;
    int32_t i32;
    int64_t i64;
    uint8_t u8;
    uint16_t u16;
    uint32_t u32;
    uint64_t u64;
    float f32;
    double f64;
    glint_string_view_t string;
    struct {
        const uint8_t* data;
        size_t len;
    } bytes;
    struct {
        int64_t seconds;
        int32_t nanos;
    } time;
    struct glint_value* struct_val;
    struct glint_value* slice;
    struct glint_value* ptr;  // For nullable pointers
    struct {
        struct glint_value* keys;
        struct glint_value* values;
        size_t len;
    } map;
} glint_value_data_t;

// Generic value structure
typedef struct glint_value {
    glint_wire_type_t type;
    glint_value_data_t data;
    const char* name;         // Field name (points into schema)
    size_t name_len;          // Field name length
    struct glint_value* next; // For slices/arrays/struct fields
} glint_value_t;

// Document structure
typedef struct glint_document {
    glint_value_t* root;
    struct glint_arena* arena;
    uint32_t schema_crc32;
    bool owns_root;  // Whether document owns and should free root
} glint_document_t;

// Decoder options
typedef struct {
    size_t max_string_len;
    size_t max_bytes_len;
    size_t max_slice_len;
    size_t max_depth;
    bool use_arena;       // Use arena allocator (default: true)
    bool cache_schemas;   // Cache parsed schemas (default: true)
} glint_options_t;

// Forward declarations
typedef struct glint_decoder glint_decoder_t;
typedef struct glint_reader glint_reader_t;

// Default options
extern const glint_options_t GLINT_DEFAULT_OPTIONS;

// Simple API
glint_document_t* glint_decode(const uint8_t* data, size_t len);
void glint_document_free(glint_document_t* doc);

// Advanced API
glint_decoder_t* glint_decoder_new(const glint_options_t* opts);
glint_document_t* glint_decode_with_decoder(glint_decoder_t* decoder, const uint8_t* data, size_t len);
void glint_decoder_free(glint_decoder_t* decoder);

// Value access helpers
const char* glint_string_get(const glint_value_t* val);
size_t glint_string_len(const glint_value_t* val);
bool glint_bool_get(const glint_value_t* val);
int64_t glint_int_get(const glint_value_t* val);
uint64_t glint_uint_get(const glint_value_t* val);
double glint_float_get(const glint_value_t* val);

// Field access for structs
glint_value_t* glint_struct_get_field(const glint_value_t* struct_val, const char* field_name);
glint_value_t* glint_struct_get_field_by_index(const glint_value_t* struct_val, size_t index);

// Array/slice access
size_t glint_slice_len(const glint_value_t* slice);
glint_value_t* glint_slice_get(const glint_value_t* slice, size_t index);

// Error handling
const char* glint_error_string(glint_error_t err);

#ifdef __cplusplus
}
#endif

#endif // GLINT_H
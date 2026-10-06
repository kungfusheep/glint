/*
 * Glint Final API - The clean interface we actually agreed on
 * 
 * Usage:
 *   GLINT_STRUCT(UserProfile,
 *       G(uint32_t, id, "user_id"),
 *       G(char*, email, "email_address"),
 *       G(bool, verified, "is_verified"),
 *       // Internal fields not sent over wire
 *       void* db_connection,
 *       uint32_t cache_flags
 *   )
 * 
 * This generates:
 *   - struct UserProfile (pure data, no overhead)
 *   - UserProfile_decoder (holds metadata)
 *   - UserProfile_unmarshal(data, len, &user)
 */

#ifndef GLINT_FINAL_H
#define GLINT_FINAL_H

#include <stdint.h>
#include <stddef.h>
#include <stdbool.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <pthread.h>

#ifdef __cplusplus
extern "C" {
#endif

// ============================================================================
// Core Types
// ============================================================================

typedef enum {
    GLINT_OK = 0,
    GLINT_ERR_INVALID_DATA = -1,
    GLINT_ERR_SCHEMA_MISMATCH = -2,
} glint_error_t;

typedef enum {
    GLINT_BOOL = 1,
    GLINT_INT = 2,
    GLINT_INT8 = 3,
    GLINT_INT16 = 4,
    GLINT_INT32 = 5,
    GLINT_INT64 = 6,
    GLINT_UINT = 7,
    GLINT_UINT8 = 8,
    GLINT_UINT16 = 9,
    GLINT_UINT32 = 10,
    GLINT_UINT64 = 11,
    GLINT_FLOAT32 = 12,
    GLINT_FLOAT64 = 13,
    GLINT_STRING = 14,
    GLINT_BYTES = 15,
    GLINT_STRUCT = 16,
    GLINT_TIME = 18,
} glint_type_t;

// Field descriptor
typedef struct {
    const char* wire_name;
    size_t offset;
    glint_type_t type;
    void* nested_decoder;
} glint_field_t;

// Decoder holds metadata - structs stay pure
typedef struct glint_decoder {
    const char* type_name;
    size_t struct_size;
    const glint_field_t* fields;
    size_t field_count;
    
    // Schema cache (decoder holds it, not instances)
    struct {
        uint32_t hash;
        void* compiled;
        uint64_t last_used;
    } cache[4];
    
    // Performance stats
    uint64_t decode_count;
    uint64_t cache_hits;
    uint64_t cache_misses;
} glint_decoder_t;

// Forward declarations
glint_error_t glint_decode_with_decoder(
    glint_decoder_t* decoder,
    const uint8_t* data,
    size_t len,
    void* output
);

// From our fast decoder
glint_error_t glint_decode_fast_struct(
    const uint8_t* data,
    size_t len,
    void* output,
    const glint_field_t* fields,
    size_t field_count
);

// ============================================================================
// The G() Macro - Exactly as we designed
// ============================================================================

// G with 2 params: use field name as wire name
#define G2(type, name) \
    type name;

// G with 3 params: use custom wire name
#define G3(type, name, wire_name) \
    type name;

// Choose based on arg count
#define G(...) G_EXPAND(__VA_ARGS__)
#define G_EXPAND(...) G_SELECT(__VA_ARGS__, G3, G2)(__VA_ARGS__)
#define G_SELECT(_1, _2, _3, NAME, ...) NAME

// ============================================================================
// Helper Macros
// ============================================================================

#define GLINT_CAT(a, b) GLINT_CAT_IMPL(a, b)
#define GLINT_CAT_IMPL(a, b) a##b

#define GLINT_STRINGIFY(x) #x

// Type detection
#define GLINT_TYPE_OF(type) GLINT_TYPE_##type
#define GLINT_TYPE_bool GLINT_BOOL
#define GLINT_TYPE_int GLINT_INT
#define GLINT_TYPE_int8_t GLINT_INT8
#define GLINT_TYPE_int16_t GLINT_INT16
#define GLINT_TYPE_int32_t GLINT_INT32
#define GLINT_TYPE_int64_t GLINT_INT64
#define GLINT_TYPE_uint8_t GLINT_UINT8
#define GLINT_TYPE_uint16_t GLINT_UINT16
#define GLINT_TYPE_uint32_t GLINT_UINT32
#define GLINT_TYPE_uint64_t GLINT_UINT64
#define GLINT_TYPE_float GLINT_FLOAT32
#define GLINT_TYPE_double GLINT_FLOAT64
#define GLINT_TYPE_time_t GLINT_TIME

// Pointer types
#define GLINT_TYPE_char_ptr GLINT_STRING
#define GLINT_TYPE_void_ptr GLINT_BYTES

// ============================================================================
// GLINT_STRUCT - The Clean Macro We Agreed On
// ============================================================================

// Since C macros can't parse the struct definition to extract G() fields,
// we need a two-step process:

// Step 1: Define the struct
#define GLINT_STRUCT(name, ...) \
    typedef struct name { \
        __VA_ARGS__ \
    } name;

// Step 2: Define the decoder and functions
// This needs to list the G fields again (C macro limitation)
#define GLINT_IMPL(name, ...) \
    /* Field descriptors */ \
    static const glint_field_t GLINT_CAT(name, _glint_fields)[] = { \
        __VA_ARGS__ \
    }; \
    \
    /* Static decoder instance - holds all metadata */ \
    static glint_decoder_t GLINT_CAT(name, _decoder_storage) = { \
        .type_name = #name, \
        .struct_size = sizeof(name), \
        .fields = GLINT_CAT(name, _glint_fields), \
        .field_count = sizeof(GLINT_CAT(name, _glint_fields)) / sizeof(glint_field_t), \
        .cache = {{0}}, \
        .decode_count = 0, \
        .cache_hits = 0, \
        .cache_misses = 0 \
    }; \
    \
    /* Public decoder pointer */ \
    static glint_decoder_t* GLINT_CAT(name, _decoder) = &GLINT_CAT(name, _decoder_storage); \
    \
    /* Unmarshal function */ \
    static inline glint_error_t GLINT_CAT(name, _unmarshal)( \
        const uint8_t* data, \
        size_t len, \
        name* output \
    ) { \
        GLINT_CAT(name, _decoder_storage).decode_count++; \
        return glint_decode_fast_struct( \
            data, len, output, \
            GLINT_CAT(name, _glint_fields), \
            GLINT_CAT(name, _decoder_storage).field_count \
        ); \
    } \
    \
    /* Marshal function */ \
    static inline glint_error_t GLINT_CAT(name, _marshal)( \
        const name* input, \
        uint8_t** data, \
        size_t* len \
    ) { \
        /* TODO: Implement marshaling */ \
        return GLINT_OK; \
    } \
    \
    /* Get decoder */ \
    static inline glint_decoder_t* GLINT_CAT(name, _get_decoder)(void) { \
        return GLINT_CAT(name, _decoder); \
    }

// Helper to define a field descriptor
#define F(type, name, wire_name) \
    {wire_name, offsetof(GLINT_CURRENT_TYPE, name), GLINT_TYPE_OF(type), NULL}

// For nested structs
#define F_NESTED(type, name, wire_name, decoder) \
    {wire_name, offsetof(GLINT_CURRENT_TYPE, name), GLINT_STRUCT, decoder}

// ============================================================================
// Complete Example as We Designed It
// ============================================================================

/*
// Step 1: Define struct with G() for serialized fields
GLINT_STRUCT(UserProfile,
    G(uint32_t, id, "user_id"),              // Snake case in wire format
    G(char*, email, "email_address"),
    G(char*, display_name, "displayName"),   // Camel case in wire format
    G(bool, verified, "is_verified"),
    G(time_t, created, "created_at"),
    G(time_t, updated, "updated_at"),
    
    // Internal fields not sent over wire
    void* db_connection,
    uint32_t cache_flags,
    pthread_mutex_t lock
)

// Step 2: Implement the decoder (list G fields)
#define GLINT_CURRENT_TYPE UserProfile
GLINT_IMPL(UserProfile,
    F(uint32_t, id, "user_id"),
    F(char_ptr, email, "email_address"),
    F(char_ptr, display_name, "displayName"),
    F(bool, verified, "is_verified"),
    F(time_t, created, "created_at"),
    F(time_t, updated, "updated_at")
)
#undef GLINT_CURRENT_TYPE

// Step 3: Use it!
void example() {
    UserProfile profile = {0};  // Pure struct, no overhead!
    profile.cache_flags = 0xFF;  // Internal field
    
    UserProfile_unmarshal(data, len, &profile);
    
    printf("User %u: %s (%s)\n", 
           profile.id, profile.email, 
           profile.verified ? "verified" : "pending");
    
    // Internal fields untouched
    assert(profile.cache_flags == 0xFF);
}
*/

// ============================================================================
// Simpler Alternative Without Repetition (but less clean syntax)
// ============================================================================

#define GLINT_SIMPLE(name, fields, glint_fields) \
    typedef struct name { \
        fields \
    } name; \
    \
    static const glint_field_t GLINT_CAT(name, _glint_fields)[] = { \
        glint_fields \
    }; \
    \
    GLINT_IMPL_INTERNAL(name)

#define GLINT_IMPL_INTERNAL(name) \
    static glint_decoder_t GLINT_CAT(name, _decoder_storage) = { \
        .type_name = #name, \
        .struct_size = sizeof(name), \
        .fields = GLINT_CAT(name, _glint_fields), \
        .field_count = sizeof(GLINT_CAT(name, _glint_fields)) / sizeof(glint_field_t), \
        .cache = {{0}}, \
        .decode_count = 0, \
        .cache_hits = 0, \
        .cache_misses = 0 \
    }; \
    \
    static inline glint_error_t GLINT_CAT(name, _unmarshal)( \
        const uint8_t* data, \
        size_t len, \
        name* output \
    ) { \
        return glint_decode_fast_struct( \
            data, len, output, \
            GLINT_CAT(name, _glint_fields), \
            GLINT_CAT(name, _decoder_storage).field_count \
        ); \
    }

#ifdef __cplusplus
}
#endif

#endif // GLINT_FINAL_H
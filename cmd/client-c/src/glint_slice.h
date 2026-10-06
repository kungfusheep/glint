#ifndef GLINT_SLICE_H
#define GLINT_SLICE_H

#include <stdint.h>
#include <stddef.h>
#include <stdlib.h>
#include <string.h>

// Generic slice structure matching Go's slice layout
typedef struct {
    void* data;      // Pointer to elements
    size_t len;      // Number of elements
    size_t cap;      // Capacity
} glint_slice_t;

// Typed slice structures for each primitive type
typedef struct {
    bool* data;
    size_t len;
    size_t cap;
} glint_bool_slice_t;

typedef struct {
    int32_t* data;
    size_t len;
    size_t cap;
} glint_int32_slice_t;

typedef struct {
    int64_t* data;
    size_t len;
    size_t cap;
} glint_int64_slice_t;

typedef struct {
    uint32_t* data;
    size_t len;
    size_t cap;
} glint_uint32_slice_t;

typedef struct {
    uint64_t* data;
    size_t len;
    size_t cap;
} glint_uint64_slice_t;

typedef struct {
    float* data;
    size_t len;
    size_t cap;
} glint_float32_slice_t;

typedef struct {
    double* data;
    size_t len;
    size_t cap;
} glint_float64_slice_t;

typedef struct {
    glint_string_view_t* data;
    size_t len;
    size_t cap;
} glint_string_slice_t;

// Allocate a slice with given capacity
static inline void* glint_slice_alloc(size_t element_size, size_t capacity) {
    return calloc(capacity, element_size);
}

// Initialize a slice structure
static inline void glint_slice_init(glint_slice_t* slice, size_t element_size, size_t len) {
    slice->len = len;
    slice->cap = len > 0 ? len : 1;  // Always allocate at least 1 element
    slice->data = glint_slice_alloc(element_size, slice->cap);
}

// Free a slice
static inline void glint_slice_free(glint_slice_t* slice) {
    if (slice->data) {
        free(slice->data);
        slice->data = NULL;
        slice->len = 0;
        slice->cap = 0;
    }
}

// Get element size for a wire type
static inline size_t glint_wire_type_size(glint_wire_type_t type) {
    switch (type & GLINT_WIRE_TYPE_MASK) {
        case GLINT_WIRE_BOOL:
            return sizeof(bool);
        case GLINT_WIRE_INT:
        case GLINT_WIRE_UINT:
            return sizeof(uint64_t);
        case GLINT_WIRE_INT8:
        case GLINT_WIRE_UINT8:
            return sizeof(uint8_t);
        case GLINT_WIRE_INT16:
        case GLINT_WIRE_UINT16:
            return sizeof(uint16_t);
        case GLINT_WIRE_INT32:
        case GLINT_WIRE_UINT32:
            return sizeof(uint32_t);
        case GLINT_WIRE_INT64:
        case GLINT_WIRE_UINT64:
            return sizeof(uint64_t);
        case GLINT_WIRE_FLOAT32:
            return sizeof(float);
        case GLINT_WIRE_FLOAT64:
            return sizeof(double);
        case GLINT_WIRE_STRING:
            return sizeof(glint_string_view_t);
        case GLINT_WIRE_BYTES:
            return sizeof(glint_string_view_t);  // Same as string
        case GLINT_WIRE_TIME:
            return sizeof(uint64_t) * 2;  // Simplified time representation
        case GLINT_WIRE_STRUCT:
            return sizeof(void*);  // Pointer to struct
        default:
            return 0;
    }
}

#endif // GLINT_SLICE_H
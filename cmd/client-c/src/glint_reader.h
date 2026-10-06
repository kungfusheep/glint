#ifndef GLINT_READER_H
#define GLINT_READER_H

#include <stdint.h>
#include <stddef.h>
#include <stdbool.h>
#include <string.h>
#include "glint.h"
#include "glint_varint.h"

struct glint_reader {
    const uint8_t* data;
    size_t len;
    size_t pos;
};

static inline void glint_reader_init(glint_reader_t* r, const uint8_t* data, size_t len) {
    r->data = data;
    r->len = len;
    r->pos = 0;
}

static inline bool glint_reader_has_bytes(const glint_reader_t* r, size_t n) {
    return r->pos + n <= r->len;
}

static inline size_t glint_reader_remaining(const glint_reader_t* r) {
    return r->len - r->pos;
}

static inline uint8_t glint_reader_read_byte(glint_reader_t* r) {
    if (r->pos >= r->len) return 0;
    return r->data[r->pos++];
}

static inline bool glint_reader_read_bytes(glint_reader_t* r, uint8_t* out, size_t n) {
    if (!glint_reader_has_bytes(r, n)) return false;
    memcpy(out, r->data + r->pos, n);
    r->pos += n;
    return true;
}

// Get bytes without copying (zero-copy)
static inline const uint8_t* glint_reader_get_bytes(glint_reader_t* r, size_t n) {
    if (!glint_reader_has_bytes(r, n)) return NULL;
    const uint8_t* ptr = r->data + r->pos;
    r->pos += n;
    return ptr;
}

static inline bool glint_reader_read_varint(glint_reader_t* r, uint64_t* out) {
    size_t bytes = glint_decode_varint(r->data + r->pos, glint_reader_remaining(r), out);
    if (bytes == 0) return false;
    r->pos += bytes;
    return true;
}

static inline bool glint_reader_read_varint32(glint_reader_t* r, uint32_t* out) {
    size_t bytes = glint_decode_varint32(r->data + r->pos, glint_reader_remaining(r), out);
    if (bytes == 0) return false;
    r->pos += bytes;
    return true;
}

static inline bool glint_reader_read_zigzag(glint_reader_t* r, int64_t* out) {
    uint64_t val;
    if (!glint_reader_read_varint(r, &val)) return false;
    *out = glint_zigzag_decode(val);
    return true;
}

// Type-specific readers
static inline bool glint_reader_read_bool(glint_reader_t* r) {
    return glint_reader_read_byte(r) != 0;
}

static inline bool glint_reader_read_int8(glint_reader_t* r, int8_t* out) {
    if (!glint_reader_has_bytes(r, 1)) return false;
    *out = (int8_t)glint_reader_read_byte(r);
    return true;
}

static inline bool glint_reader_read_uint8(glint_reader_t* r, uint8_t* out) {
    if (!glint_reader_has_bytes(r, 1)) return false;
    *out = glint_reader_read_byte(r);
    return true;
}

static inline bool glint_reader_read_int16(glint_reader_t* r, int16_t* out) {
    int64_t val;
    if (!glint_reader_read_zigzag(r, &val)) return false;
    *out = (int16_t)val;
    return true;
}

static inline bool glint_reader_read_int32(glint_reader_t* r, int32_t* out) {
    int64_t val;
    if (!glint_reader_read_zigzag(r, &val)) return false;
    *out = (int32_t)val;
    return true;
}

static inline bool glint_reader_read_int64(glint_reader_t* r, int64_t* out) {
    uint64_t val;
    if (!glint_reader_read_varint(r, &val)) return false;
    *out = (int64_t)val;
    return true;
}

static inline bool glint_reader_read_uint16(glint_reader_t* r, uint16_t* out) {
    uint64_t val;
    if (!glint_reader_read_varint(r, &val)) return false;
    *out = (uint16_t)val;
    return true;
}

static inline bool glint_reader_read_uint32(glint_reader_t* r, uint32_t* out) {
    return glint_reader_read_varint32(r, out);
}

static inline bool glint_reader_read_uint64(glint_reader_t* r, uint64_t* out) {
    return glint_reader_read_varint(r, out);
}

static inline bool glint_reader_read_float32(glint_reader_t* r, float* out) {
    uint32_t val;
    if (!glint_reader_read_varint32(r, &val)) return false;
    *out = *(float*)&val;
    return true;
}

static inline bool glint_reader_read_float64(glint_reader_t* r, double* out) {
    uint64_t val;
    if (!glint_reader_read_varint(r, &val)) return false;
    *out = *(double*)&val;
    return true;
}

// String reading (zero-copy)
static inline const char* glint_reader_read_string(glint_reader_t* r, size_t* out_len) {
    uint64_t len;
    if (!glint_reader_read_varint(r, &len)) return NULL;
    
    if (!glint_reader_has_bytes(r, len)) return NULL;
    
    const char* str = (const char*)(r->data + r->pos);
    r->pos += len;
    
    if (out_len) *out_len = len;
    return str;
}

// Skip n bytes
static inline bool glint_reader_skip(glint_reader_t* r, size_t n) {
    if (!glint_reader_has_bytes(r, n)) return false;
    r->pos += n;
    return true;
}

// Read little-endian uint32
static inline uint32_t glint_reader_read_uint32_le(glint_reader_t* r) {
    if (!glint_reader_has_bytes(r, 4)) return 0;
    uint32_t val = (uint32_t)r->data[r->pos] |
                   ((uint32_t)r->data[r->pos + 1] << 8) |
                   ((uint32_t)r->data[r->pos + 2] << 16) |
                   ((uint32_t)r->data[r->pos + 3] << 24);
    r->pos += 4;
    return val;
}

#endif // GLINT_READER_H
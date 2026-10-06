#ifndef GLINT_VARINT_OPT_H
#define GLINT_VARINT_OPT_H

#include <stdint.h>
#include <stddef.h>
#include <string.h>

// Optimized varint decoder using computed goto (GCC/Clang extension)
// This eliminates branch misprediction in the main decode loop
#ifdef __GNUC__
static inline size_t glint_decode_varint_goto(const uint8_t* data, size_t len, uint64_t* out) {
    if (len == 0) return 0;
    
    // Fast path for single byte (most common)
    if ((data[0] & 0x80) == 0) {
        *out = data[0];
        return 1;
    }
    
    // Use computed goto for the rest
    static void* dispatch[] = {
        &&byte1, &&byte2, &&byte3, &&byte4, &&byte5,
        &&byte6, &&byte7, &&byte8, &&byte9, &&byte10
    };
    
    uint64_t result = 0;
    size_t i = 0;
    
    goto *dispatch[0];
    
byte1:
    if (i >= len) return 0;
    result |= (uint64_t)(data[i] & 0x7F);
    if ((data[i++] & 0x80) == 0) { *out = result; return i; }
    
byte2:
    if (i >= len) return 0;
    result |= (uint64_t)(data[i] & 0x7F) << 7;
    if ((data[i++] & 0x80) == 0) { *out = result; return i; }
    
byte3:
    if (i >= len) return 0;
    result |= (uint64_t)(data[i] & 0x7F) << 14;
    if ((data[i++] & 0x80) == 0) { *out = result; return i; }
    
byte4:
    if (i >= len) return 0;
    result |= (uint64_t)(data[i] & 0x7F) << 21;
    if ((data[i++] & 0x80) == 0) { *out = result; return i; }
    
byte5:
    if (i >= len) return 0;
    result |= (uint64_t)(data[i] & 0x7F) << 28;
    if ((data[i++] & 0x80) == 0) { *out = result; return i; }
    
byte6:
    if (i >= len) return 0;
    result |= (uint64_t)(data[i] & 0x7F) << 35;
    if ((data[i++] & 0x80) == 0) { *out = result; return i; }
    
byte7:
    if (i >= len) return 0;
    result |= (uint64_t)(data[i] & 0x7F) << 42;
    if ((data[i++] & 0x80) == 0) { *out = result; return i; }
    
byte8:
    if (i >= len) return 0;
    result |= (uint64_t)(data[i] & 0x7F) << 49;
    if ((data[i++] & 0x80) == 0) { *out = result; return i; }
    
byte9:
    if (i >= len) return 0;
    result |= (uint64_t)(data[i] & 0x7F) << 56;
    if ((data[i++] & 0x80) == 0) { *out = result; return i; }
    
byte10:
    if (i >= len) return 0;
    // Last byte can only use 1 bit for valid uint64
    if (data[i] > 1) return 0;
    result |= (uint64_t)(data[i] & 0x7F) << 63;
    *out = result;
    return i + 1;
}
#endif

// Optimized varint using prefetch and better memory access patterns
static inline size_t glint_decode_varint_prefetch(const uint8_t* data, size_t len, uint64_t* out) {
    if (len == 0) return 0;
    
    // Prefetch next cache line for better memory performance
    #ifdef __builtin_prefetch
    __builtin_prefetch(data + 64, 0, 1);
    #endif
    
    // Fast path for single byte
    uint8_t b0 = data[0];
    if ((b0 & 0x80) == 0) {
        *out = b0;
        return 1;
    }
    
    if (len < 2) return 0;
    uint8_t b1 = data[1];
    if ((b1 & 0x80) == 0) {
        *out = (b0 & 0x7F) | ((uint64_t)b1 << 7);
        return 2;
    }
    
    if (len < 3) return 0;
    uint8_t b2 = data[2];
    if ((b2 & 0x80) == 0) {
        *out = (b0 & 0x7F) | ((uint64_t)(b1 & 0x7F) << 7) | ((uint64_t)b2 << 14);
        return 3;
    }
    
    if (len < 4) return 0;
    uint8_t b3 = data[3];
    if ((b3 & 0x80) == 0) {
        *out = (b0 & 0x7F) | 
               ((uint64_t)(b1 & 0x7F) << 7) | 
               ((uint64_t)(b2 & 0x7F) << 14) |
               ((uint64_t)b3 << 21);
        return 4;
    }
    
    // For longer varints, use the standard approach
    uint64_t result = (b0 & 0x7F) | 
                      ((uint64_t)(b1 & 0x7F) << 7) | 
                      ((uint64_t)(b2 & 0x7F) << 14) |
                      ((uint64_t)(b3 & 0x7F) << 21);
    
    size_t shift = 28;
    for (size_t i = 4; i < len && i < 10; i++) {
        uint8_t byte = data[i];
        result |= ((uint64_t)(byte & 0x7F)) << shift;
        if ((byte & 0x80) == 0) {
            *out = result;
            return i + 1;
        }
        shift += 7;
    }
    
    return 0;
}

// Lookup table for fast byte count determination
static const uint8_t varint_byte_count[256] = {
    // 0x00-0x7F: single byte (no continuation bit)
    [0x00 ... 0x7F] = 1,
    // 0x80-0xFF: multi-byte (has continuation bit)
    [0x80 ... 0xFF] = 0,
};

// Fast varint with byte count prediction
static inline size_t glint_decode_varint_predict(const uint8_t* data, size_t len, uint64_t* out) {
    if (len == 0) return 0;
    
    // Check first byte for fast path
    uint8_t first = data[0];
    if (varint_byte_count[first] == 1) {
        *out = first;
        return 1;
    }
    
    // Count continuation bits to predict length
    size_t predicted_len = 10; // Default to max if all have continuation
    for (size_t i = 0; i < len && i < 10; i++) {
        if ((data[i] & 0x80) == 0) {
            predicted_len = i + 1;
            break;
        }
    }
    
    if (predicted_len > len) return 0; // Not enough data
    
    // Now decode knowing the length
    uint64_t result = 0;
    switch (predicted_len) {
        case 2:
            *out = (data[0] & 0x7F) | ((uint64_t)data[1] << 7);
            return 2;
        case 3:
            *out = (data[0] & 0x7F) | 
                   ((uint64_t)(data[1] & 0x7F) << 7) |
                   ((uint64_t)data[2] << 14);
            return 3;
        case 4:
            *out = (data[0] & 0x7F) | 
                   ((uint64_t)(data[1] & 0x7F) << 7) |
                   ((uint64_t)(data[2] & 0x7F) << 14) |
                   ((uint64_t)data[3] << 21);
            return 4;
        default:
            // General case
            for (size_t i = 0; i < predicted_len; i++) {
                result |= ((uint64_t)(data[i] & 0x7F)) << (i * 7);
            }
            *out = result;
            return predicted_len;
    }
}

#endif // GLINT_VARINT_OPT_H
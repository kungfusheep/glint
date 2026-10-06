/*
 * Profiling-optimized version of the fast decoder
 * Uses noinline attributes to preserve function boundaries for profiling
 * while maintaining most optimizations
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <stdbool.h>
#include <math.h>
#include "glint.h"
#include "glint_reader.h"
#include "glint_varint.h"

// Instruction opcodes matching wire types plus control flow
typedef enum {
    OP_BOOL = 1,
    OP_INT = 2,
    OP_INT8 = 3,
    OP_INT16 = 4,
    OP_INT32 = 5,
    OP_INT64 = 6,
    OP_UINT = 7,
    OP_UINT8 = 8,
    OP_UINT16 = 9,
    OP_UINT32 = 10,
    OP_UINT64 = 11,
    OP_FLOAT32 = 12,
    OP_FLOAT64 = 13,
    OP_STRING = 14,
    OP_BYTES = 15,
    OP_STRUCT = 16,
    OP_MAP = 17,
    OP_TIME = 18,
    
    // Control opcodes
    OP_SKIP_VARINT = 100,
    OP_SKIP_BYTES = 101,
    OP_SKIP_FIXED32 = 102,
    OP_SKIP_FIXED64 = 103,
    OP_SLICE_START = 104,
    OP_SLICE_END = 105,
    OP_PTR_CHECK = 106,
} opcode_t;

// Decode instruction - what to do with each field
typedef struct {
    opcode_t op;
    uint16_t offset;      // Offset into target struct (for non-skip ops)
    uint16_t slice_size;  // Size of slice element (for slices)
    uint16_t jump;        // Jump target for control flow
} decode_instruction_t;

// Schema cache entry
typedef struct schema_cache_entry {
    uint32_t hash;
    decode_instruction_t* instructions;
    size_t instruction_count;
    size_t ref_count;
    struct schema_cache_entry* next;  // For hash collision chaining
} schema_cache_entry_t;

// Simple hash table for schema cache
#define CACHE_SIZE 256
static schema_cache_entry_t* schema_cache[CACHE_SIZE];
static bool cache_initialized = false;

// Initialize cache if needed
__attribute__((noinline))
static void ensure_cache_initialized(void) {
    if (!cache_initialized) {
        memset(schema_cache, 0, sizeof(schema_cache));
        cache_initialized = true;
    }
}

// Look up schema in cache
__attribute__((noinline))
static schema_cache_entry_t* cache_lookup(uint32_t hash) {
    ensure_cache_initialized();
    uint32_t slot = hash % CACHE_SIZE;
    schema_cache_entry_t* entry = schema_cache[slot];
    
    while (entry) {
        if (entry->hash == hash) {
            return entry;
        }
        entry = entry->next;
    }
    return NULL;
}

// Add schema to cache
__attribute__((noinline))
static void cache_add(uint32_t hash, decode_instruction_t* instructions, size_t count) {
    ensure_cache_initialized();
    uint32_t slot = hash % CACHE_SIZE;
    
    schema_cache_entry_t* entry = calloc(1, sizeof(schema_cache_entry_t));
    entry->hash = hash;
    entry->instructions = instructions;
    entry->instruction_count = count;
    entry->ref_count = 1;
    entry->next = schema_cache[slot];
    schema_cache[slot] = entry;
}

// Compile schema into instructions
__attribute__((noinline))
static decode_instruction_t* compile_schema(glint_reader_t* schema, size_t* count) {
    // Start with space for 64 instructions
    size_t capacity = 64;
    decode_instruction_t* instructions = malloc(capacity * sizeof(decode_instruction_t));
    size_t n = 0;
    
    while (schema->pos < schema->len) {
        // Ensure capacity
        if (n >= capacity) {
            capacity *= 2;
            instructions = realloc(instructions, capacity * sizeof(decode_instruction_t));
        }
        
        // Read wire type and field name
        uint64_t wire_type;
        if (!glint_reader_read_varint(schema, &wire_type)) break;
        
        uint8_t name_len = glint_reader_read_byte(schema);
        if (name_len == 0 || name_len > 63) break;
        
        // Skip name for now - in real implementation we'd look up field offset
        schema->pos += name_len;
        
        // Decode wire type flags
        glint_wire_type_t base_type = wire_type & GLINT_WIRE_TYPE_MASK;
        bool is_slice = (wire_type & GLINT_WIRE_SLICE_FLAG) != 0;
        bool is_ptr = (wire_type & GLINT_WIRE_PTR_FLAG) != 0;
        
        // Handle different field types
        if (is_slice) {
            // Emit slice instructions
            instructions[n].op = OP_SLICE_START;
            instructions[n].offset = 0;  // Will be filled by field lookup
            n++;
            
            if (n >= capacity) {
                capacity *= 2;
                instructions = realloc(instructions, capacity * sizeof(decode_instruction_t));
            }
            
            // Element type instruction
            instructions[n].op = base_type;
            instructions[n].offset = 0;
            n++;
            
            if (n >= capacity) {
                capacity *= 2;
                instructions = realloc(instructions, capacity * sizeof(decode_instruction_t));
            }
            
            instructions[n].op = OP_SLICE_END;
            n++;
        } else if (is_ptr) {
            // Emit pointer check
            instructions[n].op = OP_PTR_CHECK;
            instructions[n].offset = 0;
            instructions[n].jump = 2;  // Skip next instruction if null
            n++;
            
            if (n >= capacity) {
                capacity *= 2;
                instructions = realloc(instructions, capacity * sizeof(decode_instruction_t));
            }
            
            instructions[n].op = base_type;
            instructions[n].offset = 0;
            n++;
        } else if (base_type == GLINT_WIRE_STRUCT) {
            // Handle nested struct schema
            uint64_t schema_len;
            glint_reader_read_varint(schema, &schema_len);
            
            // For now, skip nested schema
            // In real implementation, we'd recursively compile it
            schema->pos += schema_len;
            
            instructions[n].op = OP_STRUCT;
            instructions[n].offset = 0;
            n++;
        } else {
            // Simple field
            instructions[n].op = base_type;
            instructions[n].offset = 0;  // Will be filled by field lookup
            n++;
        }
    }
    
    *count = n;
    return instructions;
}

// Separate functions for each operation type to show in profiler
__attribute__((noinline))
static void decode_bool(glint_reader_t* reader, uint8_t* base, uint16_t offset) {
    *(bool*)(base + offset) = glint_reader_read_bool(reader);
}

__attribute__((noinline))
static void decode_int(glint_reader_t* reader, uint8_t* base, uint16_t offset) {
    int64_t val;
    glint_reader_read_zigzag(reader, &val);
    *(int64_t*)(base + offset) = val;
}

__attribute__((noinline))
static void decode_int32(glint_reader_t* reader, uint8_t* base, uint16_t offset) {
    int64_t val;
    glint_reader_read_zigzag(reader, &val);
    *(int32_t*)(base + offset) = (int32_t)val;
}

__attribute__((noinline))
static void decode_uint(glint_reader_t* reader, uint8_t* base, uint16_t offset) {
    uint64_t val;
    glint_reader_read_varint(reader, &val);
    *(uint64_t*)(base + offset) = val;
}

__attribute__((noinline))
static void decode_float64(glint_reader_t* reader, uint8_t* base, uint16_t offset) {
    double val;
    glint_reader_read_float64(reader, &val);
    *(double*)(base + offset) = val;
}

__attribute__((noinline))
static void decode_string(glint_reader_t* reader, uint8_t* base, uint16_t offset) {
    size_t len;
    const char* str = glint_reader_read_string(reader, &len);
    glint_string_view_t* sv = (glint_string_view_t*)(base + offset);
    sv->data = (const uint8_t*)str;
    sv->len = len;
}

__attribute__((noinline))
static void decode_bytes(glint_reader_t* reader, uint8_t* base, uint16_t offset) {
    uint64_t len;
    glint_reader_read_varint(reader, &len);
    typedef struct { const uint8_t* data; size_t len; } bytes_view_t;
    bytes_view_t* bv = (bytes_view_t*)(base + offset);
    bv->data = reader->data + reader->pos;
    bv->len = len;
    reader->pos += len;
}

__attribute__((noinline))
static size_t decode_slice_start(glint_reader_t* reader, decode_instruction_t* instructions, size_t i, size_t count) {
    uint64_t slice_count;
    glint_reader_read_varint(reader, &slice_count);
    
    // For now, skip slice elements
    i++; // Skip element instruction
    
    // Find matching SLICE_END
    while (i < count && instructions[i].op != OP_SLICE_END) {
        i++;
    }
    return i;
}

// Execute instructions to decode data - preserved as separate function
__attribute__((noinline))
static void execute_instructions(glint_reader_t* reader, 
                                decode_instruction_t* instructions, 
                                size_t count,
                                void* output) {
    uint8_t* base = (uint8_t*)output;
    
    for (size_t i = 0; i < count; i++) {
        decode_instruction_t* instr = &instructions[i];
        
        switch (instr->op) {
            case OP_BOOL:
                decode_bool(reader, base, instr->offset);
                break;
                
            case OP_INT:
                decode_int(reader, base, instr->offset);
                break;
            
            case OP_INT8: {
                int8_t val;
                glint_reader_read_int8(reader, &val);
                *(int8_t*)(base + instr->offset) = val;
                break;
            }
                
            case OP_INT16: {
                int16_t val;
                glint_reader_read_int16(reader, &val);
                *(int16_t*)(base + instr->offset) = val;
                break;
            }
                
            case OP_INT32:
                decode_int32(reader, base, instr->offset);
                break;
            
            case OP_INT64: {
                int64_t val;
                glint_reader_read_int64(reader, &val);
                *(int64_t*)(base + instr->offset) = val;
                break;
            }
                
            case OP_UINT:
                decode_uint(reader, base, instr->offset);
                break;
            
            case OP_UINT8: {
                uint8_t val;
                glint_reader_read_uint8(reader, &val);
                *(uint8_t*)(base + instr->offset) = val;
                break;
            }
                
            case OP_UINT16: {
                uint16_t val;
                glint_reader_read_uint16(reader, &val);
                *(uint16_t*)(base + instr->offset) = val;
                break;
            }
                
            case OP_UINT32: {
                uint32_t val;
                glint_reader_read_uint32(reader, &val);
                *(uint32_t*)(base + instr->offset) = val;
                break;
            }
                
            case OP_UINT64: {
                uint64_t val;
                glint_reader_read_uint64(reader, &val);
                *(uint64_t*)(base + instr->offset) = val;
                break;
            }
                
            case OP_FLOAT32: {
                float val;
                glint_reader_read_float32(reader, &val);
                *(float*)(base + instr->offset) = val;
                break;
            }
                
            case OP_FLOAT64:
                decode_float64(reader, base, instr->offset);
                break;
                
            case OP_STRING:
                decode_string(reader, base, instr->offset);
                break;
            
            case OP_BYTES:
                decode_bytes(reader, base, instr->offset);
                break;
            
            case OP_TIME: {
                uint64_t time_len;
                glint_reader_read_varint(reader, &time_len);
                reader->pos += time_len;
                break;
            }
            
            case OP_SLICE_START:
                i = decode_slice_start(reader, instructions, i, count);
                break;
            
            case OP_PTR_CHECK: {
                uint8_t present = glint_reader_read_byte(reader);
                if (!present) {
                    i += instr->jump;
                }
                break;
            }
            
            case OP_SKIP_VARINT: {
                uint64_t dummy;
                glint_reader_read_varint(reader, &dummy);
                break;
            }
            
            case OP_SKIP_BYTES: {
                uint64_t len;
                glint_reader_read_varint(reader, &len);
                reader->pos += len;
                break;
            }
            
            default:
                break;
        }
    }
}

// Header reading - separate function
__attribute__((noinline))
static bool read_header(glint_reader_t* reader, uint32_t* hash, uint64_t* schema_size) {
    uint8_t flags = glint_reader_read_byte(reader);
    if (flags != 0x00) return false;
    
    memcpy(hash, reader->data + reader->pos, 4);
    reader->pos += 4;
    
    return glint_reader_read_varint(reader, schema_size);
}

// Schema processing - separate function
__attribute__((noinline))
static decode_instruction_t* process_schema(glint_reader_t* reader, uint32_t hash, 
                                           uint64_t schema_size, size_t* instruction_count) {
    schema_cache_entry_t* cached = cache_lookup(hash);
    
    if (cached) {
        cached->ref_count++;
        reader->pos += schema_size;
        *instruction_count = cached->instruction_count;
        return cached->instructions;
    }
    
    glint_reader_t schema_reader;
    glint_reader_init(&schema_reader, reader->data + reader->pos, schema_size);
    reader->pos += schema_size;
    
    decode_instruction_t* instructions = compile_schema(&schema_reader, instruction_count);
    cache_add(hash, instructions, *instruction_count);
    
    return instructions;
}

// Fast decoder entry point - profiling version
glint_document_t* glint_decode_fast_profile(const uint8_t* data, size_t len) {
    if (!data || len < 7) return NULL;
    
    glint_reader_t reader;
    glint_reader_init(&reader, data, len);
    
    uint32_t hash;
    uint64_t schema_size;
    if (!read_header(&reader, &hash, &schema_size)) {
        return NULL;
    }
    
    size_t instruction_count;
    decode_instruction_t* instructions = process_schema(&reader, hash, schema_size, &instruction_count);
    
    glint_document_t* doc = calloc(1, sizeof(glint_document_t));
    doc->schema_crc32 = hash;
    
    void* output = calloc(1, 1024);
    
    glint_reader_t data_reader;
    glint_reader_init(&data_reader, reader.data + reader.pos, reader.len - reader.pos);
    execute_instructions(&data_reader, instructions, instruction_count, output);
    
    doc->root = (glint_value_t*)output;
    
    return doc;
}

// Free document - profiling version
void glint_document_free_fast_profile(glint_document_t* doc) {
    if (!doc) return;
    if (doc->root) free(doc->root);
    free(doc);
}
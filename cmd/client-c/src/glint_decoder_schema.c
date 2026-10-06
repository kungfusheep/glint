#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include "glint.h"
#include "glint_reader.h"

// Debug output control
#ifndef DEBUG
#define DEBUG 0
#endif
#if DEBUG
#define DBG_PRINT(...) fprintf(stderr, __VA_ARGS__)
#else
#define DBG_PRINT(...)
#endif

// Schema field definition
typedef struct schema_field {
    glint_wire_type_t wire_type;
    const char* name;  // Pointer to name in original buffer (zero-copy)
    size_t name_len;
    struct schema_field* next;
    // For nested structs, we store the nested fields
    struct schema_field* nested_fields;
    size_t nested_count;
} schema_field_t;

// Forward declarations
static glint_value_t* decode_value(glint_reader_t* reader, schema_field_t* field, int depth);
static schema_field_t* parse_schema_fields_limited(glint_reader_t* reader, size_t* field_count, size_t limit);
static void free_schema(schema_field_t* schema);

// Wrapper for unlimited parsing
static schema_field_t* parse_schema_fields(glint_reader_t* reader, size_t* field_count) {
    return parse_schema_fields_limited(reader, field_count, 0);
}

// Parse schema section into field list (with optional limit for nested structs)
static schema_field_t* parse_schema_fields_limited(glint_reader_t* reader, size_t* field_count, size_t limit) {
    schema_field_t* first = NULL;
    schema_field_t* current = NULL;
    size_t count = 0;
    
    while (reader->pos < reader->len && (limit == 0 || count < limit)) {
        // Read wire type
        uint64_t wire_type;
        if (!glint_reader_read_varint(reader, &wire_type)) break;
        
        // Read name length
        uint8_t name_len = glint_reader_read_byte(reader);
        if (name_len == 0 || name_len > 63) break;
        
        // Create field
        schema_field_t* field = calloc(1, sizeof(schema_field_t));
        if (!field) break;
        
        field->wire_type = (glint_wire_type_t)wire_type;
        field->name_len = name_len;
        
        // Point to name in original buffer (zero-copy)
        if (reader->pos + name_len > reader->len) {
            free(field);
            break;
        }
        field->name = (const char*)(reader->data + reader->pos);
        reader->pos += name_len;
        
        DBG_PRINT("Schema field %zu: %.*s (type=0x%02x)\n", count, (int)field->name_len, field->name, field->wire_type);
        
        // Handle nested struct schema
        glint_wire_type_t base_type = field->wire_type & GLINT_WIRE_TYPE_MASK;
        bool is_slice = (field->wire_type & GLINT_WIRE_SLICE_FLAG) != 0;
        
        if (base_type == GLINT_WIRE_STRUCT) {
            // Read the byte length of the nested schema
            uint64_t schema_len;
            if (glint_reader_read_varint(reader, &schema_len)) {
                DBG_PRINT("  Struct schema: %llu bytes\n", schema_len);
                
                // Create a sub-reader for the nested schema
                if (reader->pos + schema_len <= reader->len) {
                    glint_reader_t nested_reader;
                    glint_reader_init(&nested_reader, reader->data + reader->pos, schema_len);
                    
                    // Parse the nested schema
                    size_t nested_count;
                    field->nested_fields = parse_schema_fields_limited(&nested_reader, &nested_count, 0);
                    field->nested_count = nested_count;
                    
                    DBG_PRINT("  Parsed %zu nested fields\n", nested_count);
                    
                    // Advance main reader past the nested schema
                    reader->pos += schema_len;
                }
            }
        }
        
        // Link to list
        if (!first) {
            first = field;
            current = field;
        } else {
            current->next = field;
            current = field;
        }
        count++;
        
        // For nested structs, we've already advanced past their fields
        if (field->nested_fields) {
            // The nested fields were parsed recursively
        }
    }
    
    if (field_count) *field_count = count;
    return first;
}

// Free schema list
static void free_schema(schema_field_t* schema) {
    while (schema) {
        schema_field_t* next = schema->next;
        if (schema->nested_fields) {
            free_schema(schema->nested_fields);
        }
        free(schema);
        schema = next;
    }
}

// Decode a single value based on wire type
static glint_value_t* decode_value(glint_reader_t* reader, schema_field_t* field, int depth) {
    if (depth > 100) return NULL;
    
    glint_value_t* value = calloc(1, sizeof(glint_value_t));
    if (!value) return NULL;
    
    // Extract base type and flags
    glint_wire_type_t base_type = field->wire_type & GLINT_WIRE_TYPE_MASK;
    bool is_slice = (field->wire_type & GLINT_WIRE_SLICE_FLAG) != 0;
    bool is_ptr = (field->wire_type & GLINT_WIRE_PTR_FLAG) != 0;
    
    value->type = field->wire_type;
    value->name = field->name;
    value->name_len = field->name_len;
    
    DBG_PRINT("  Decoding %.*s (type=0x%02x, base=%d, slice=%d, ptr=%d)\n", 
              (int)field->name_len, field->name, field->wire_type, base_type, is_slice, is_ptr);
    
    // Handle slices
    if (is_slice) {
        DBG_PRINT("    About to read slice count at pos=%zu, byte=0x%02x\n", 
                  reader->pos, reader->pos < reader->len ? reader->data[reader->pos] : 0xFF);
        uint64_t count;
        if (!glint_reader_read_varint(reader, &count)) {
            DBG_PRINT("    ERROR: Failed to read slice count at pos=%zu\n", reader->pos);
            free(value);
            return NULL;
        }
        DBG_PRINT("    Slice with %llu elements (reader pos=%zu after reading count)\n", count, reader->pos);
        
        // Decode slice elements
        glint_value_t* first_elem = NULL;
        glint_value_t* current_elem = NULL;
        
        for (uint64_t i = 0; i < count; i++) {
            // Create a temporary field for each element
            schema_field_t elem_field = *field;
            elem_field.wire_type = base_type; // Remove slice flag
            
            glint_value_t* elem = NULL;
            
            // For slice of structs, use nested schema
            if (base_type == GLINT_WIRE_STRUCT && field->nested_fields) {
                // Decode struct using nested schema
                elem = calloc(1, sizeof(glint_value_t));
                if (elem) {
                    elem->type = GLINT_WIRE_STRUCT;
                    glint_value_t* first_field = NULL;
                    glint_value_t* current_field = NULL;
                    
                    schema_field_t* nested = field->nested_fields;
                    while (nested) {
                        glint_value_t* field_val = decode_value(reader, nested, depth + 1);
                        if (!field_val) break;
                        
                        if (!first_field) {
                            first_field = field_val;
                            current_field = field_val;
                        } else {
                            current_field->next = field_val;
                            current_field = field_val;
                        }
                        nested = nested->next;
                    }
                    elem->data.struct_val = first_field;
                }
            } else {
                // Decode primitive element
                DBG_PRINT("      Decoding slice element %llu (type=%d)\n", i, base_type);
                elem = decode_value(reader, &elem_field, depth + 1);
            }
            
            if (!elem) {
                DBG_PRINT("      ERROR: Failed to decode element %llu\n", i);
                break;
            }
            
            if (!first_elem) {
                first_elem = elem;
                current_elem = elem;
            } else {
                current_elem->next = elem;
                current_elem = elem;
            }
        }
        
        value->data.slice = first_elem;
        return value;
    }
    
    // Handle nullable pointers
    if (is_ptr) {
        uint8_t present = glint_reader_read_byte(reader);
        DBG_PRINT("    Pointer present=%d\n", present);
        if (!present) {
            value->data.ptr = NULL;
            return value;
        }
        // Continue to decode the value
    }
    
    // Decode based on type
    switch (base_type) {
        case GLINT_WIRE_BOOL:
            value->data.bool_val = glint_reader_read_bool(reader);
            DBG_PRINT("    Bool value: %d\n", value->data.bool_val);
            break;
            
        case GLINT_WIRE_INT: {
            int64_t val;
            glint_reader_read_zigzag(reader, &val);
            value->data.i64 = val;  // INT is full size int
            DBG_PRINT("    Int value: %lld\n", val);
            break;
        }
        
        case GLINT_WIRE_INT32: {
            int64_t val;
            glint_reader_read_zigzag(reader, &val);
            value->data.i32 = (int32_t)val;
            DBG_PRINT("    Int32 value: %d\n", value->data.i32);
            break;
        }
            
        case GLINT_WIRE_INT8: {
            int8_t val;
            glint_reader_read_int8(reader, &val);
            value->data.i8 = val;
            DBG_PRINT("    Int8 value: %d\n", val);
            break;
        }
            
        case GLINT_WIRE_INT16: {
            int16_t val;
            glint_reader_read_int16(reader, &val);
            value->data.i16 = val;
            DBG_PRINT("    Int16 value: %d\n", val);
            break;
        }
            
        case GLINT_WIRE_INT64: {
            int64_t val;
            glint_reader_read_int64(reader, &val);
            value->data.i64 = val;
            DBG_PRINT("    Int64 value: %lld\n", val);
            break;
        }
            
        case GLINT_WIRE_UINT: {
            uint64_t val;
            glint_reader_read_varint(reader, &val);
            value->data.u64 = val;  // UINT is full size uint
            DBG_PRINT("    Uint value: %llu\n", val);
            break;
        }
        
        case GLINT_WIRE_UINT32: {
            uint32_t val;
            glint_reader_read_uint32(reader, &val);
            value->data.u32 = val;
            DBG_PRINT("    Uint32 value: %u\n", val);
            break;
        }
            
        case GLINT_WIRE_UINT8: {
            uint8_t val;
            glint_reader_read_uint8(reader, &val);
            value->data.u8 = val;
            DBG_PRINT("    Uint8 value: %u\n", val);
            break;
        }
            
        case GLINT_WIRE_UINT16: {
            uint16_t val;
            glint_reader_read_uint16(reader, &val);
            value->data.u16 = val;
            DBG_PRINT("    Uint16 value: %u\n", val);
            break;
        }
            
        case GLINT_WIRE_UINT64: {
            uint64_t val;
            glint_reader_read_uint64(reader, &val);
            value->data.u64 = val;
            DBG_PRINT("    Uint64 value: %llu\n", val);
            break;
        }
            
        case GLINT_WIRE_FLOAT32: {
            float val;
            glint_reader_read_float32(reader, &val);
            value->data.f32 = val;
            DBG_PRINT("    Float32 value: %f\n", value->data.f32);
            break;
        }
            
        case GLINT_WIRE_FLOAT64: {
            double val;
            glint_reader_read_float64(reader, &val);
            value->data.f64 = val;
            DBG_PRINT("    Float64 value: %f\n", value->data.f64);
            break;
        }
            
        case GLINT_WIRE_STRING: {
            size_t len;
            const char* str = glint_reader_read_string(reader, &len);
            value->data.string.data = (const uint8_t*)str;
            value->data.string.len = len;
            DBG_PRINT("    String value: '%.*s'\n", (int)len, str);
            break;
        }
            
        case GLINT_WIRE_BYTES: {
            uint64_t len;
            glint_reader_read_varint(reader, &len);
            value->data.bytes.data = reader->data + reader->pos;
            value->data.bytes.len = len;
            reader->pos += len;
            DBG_PRINT("    Bytes value: %llu bytes\n", len);
            break;
        }
            
        case GLINT_WIRE_STRUCT: {
            DBG_PRINT("    Decoding nested struct\n");
            // Decode nested struct using its schema
            glint_value_t* first_field = NULL;
            glint_value_t* current_field = NULL;
            
            schema_field_t* nested = field->nested_fields;
            while (nested) {
                glint_value_t* field_val = decode_value(reader, nested, depth + 1);
                if (!field_val) break;
                
                if (!first_field) {
                    first_field = field_val;
                    current_field = field_val;
                } else {
                    current_field->next = field_val;
                    current_field = field_val;
                }
                nested = nested->next;
            }
            
            value->data.struct_val = first_field;
            break;
        }
            
        case GLINT_WIRE_TIME: {
            // Time is encoded as varint length + marshaled bytes
            DBG_PRINT("    Reading time at pos=%zu\n", reader->pos);
            uint64_t time_len;
            glint_reader_read_varint(reader, &time_len);
            
            // For now, just skip the time bytes
            // TODO: Properly decode Go's time.Time binary format
            reader->pos += time_len;
            value->data.time.seconds = 0;
            value->data.time.nanos = 0;
            
            DBG_PRINT("    Time: skipped %llu bytes (now at pos=%zu)\n", 
                      time_len, reader->pos);
            break;
        }
            
        default:
            DBG_PRINT("    Unknown type %d\n", base_type);
            free(value);
            return NULL;
    }
    
    return value;
}

// Main decode function
glint_document_t* glint_decode(const uint8_t* data, size_t len) {
    if (!data || len < 7) return NULL;
    
    glint_reader_t reader;
    glint_reader_init(&reader, data, len);
    
    // Read flags byte
    uint8_t flags = glint_reader_read_byte(&reader);
    if (flags != 0x00) return NULL;
    
    // Read hash (4 bytes, little endian)
    uint32_t hash = 0;
    glint_reader_read_bytes(&reader, (uint8_t*)&hash, 4);
    
    DBG_PRINT("Document hash: 0x%08x\n", hash);
    
    // Read schema size
    uint64_t schema_size;
    if (!glint_reader_read_varint(&reader, &schema_size)) return NULL;
    
    DBG_PRINT("Schema size: %llu bytes\n", schema_size);
    
    // Parse schema
    glint_reader_t schema_reader;
    glint_reader_init(&schema_reader, reader.data + reader.pos, schema_size);
    size_t field_count = 0;
    schema_field_t* schema = parse_schema_fields(&schema_reader, &field_count);
    if (!schema) return NULL;
    
    DBG_PRINT("Parsed %zu top-level fields\n", field_count);
    
    // Skip past schema in main reader
    reader.pos += schema_size;
    
    DBG_PRINT("Data section starts at position %zu\n", reader.pos);
    
    // Create document
    glint_document_t* doc = calloc(1, sizeof(glint_document_t));
    if (!doc) {
        free_schema(schema);
        return NULL;
    }
    
    doc->schema_crc32 = hash;
    
    // Create root struct value
    glint_value_t* root = calloc(1, sizeof(glint_value_t));
    if (!root) {
        free_schema(schema);
        free(doc);
        return NULL;
    }
    
    root->type = GLINT_WIRE_STRUCT;
    doc->root = root;
    
    // Decode data section using schema
    glint_reader_t data_reader;
    glint_reader_init(&data_reader, reader.data + reader.pos, reader.len - reader.pos);
    
    DBG_PRINT("Decoding data section (%zu bytes)\n", data_reader.len);
    
    // Decode each field in schema order
    glint_value_t* first_field = NULL;
    glint_value_t* current_field = NULL;
    
    schema_field_t* schema_field = schema;
    while (schema_field) {
        glint_value_t* field_val = decode_value(&data_reader, schema_field, 0);
        if (!field_val) {
            DBG_PRINT("Failed to decode field %s\n", schema_field->name);
            break;
        }
        
        if (!first_field) {
            first_field = field_val;
            current_field = field_val;
        } else {
            current_field->next = field_val;
            current_field = field_val;
        }
        
        schema_field = schema_field->next;
    }
    
    root->data.struct_val = first_field;
    
    DBG_PRINT("Decoding complete. Bytes remaining: %zu\n", 
              data_reader.len - data_reader.pos);
    
    // Clean up schema
    free_schema(schema);
    
    return doc;
}

// Free a value and all its children
static void free_value(glint_value_t* value) {
    if (!value) return;
    
    glint_wire_type_t base_type = value->type & GLINT_WIRE_TYPE_MASK;
    bool is_slice = (value->type & GLINT_WIRE_SLICE_FLAG) != 0;
    
    if (is_slice) {
        // Free all slice elements
        glint_value_t* elem = value->data.slice;
        while (elem) {
            glint_value_t* next = elem->next;
            free_value(elem);
            elem = next;
        }
    } else if (base_type == GLINT_WIRE_STRUCT) {
        // Free all struct fields
        glint_value_t* field = value->data.struct_val;
        while (field) {
            glint_value_t* next = field->next;
            free_value(field);
            field = next;
        }
    }
    
    free(value);
}

// Free document
void glint_document_free(glint_document_t* doc) {
    if (!doc) return;
    
    if (doc->root) {
        // Free all fields in root struct
        glint_value_t* field = doc->root->data.struct_val;
        while (field) {
            glint_value_t* next = field->next;
            free_value(field);
            field = next;
        }
        free(doc->root);
    }
    
    free(doc);
}

// Get error string
const char* glint_error_string(glint_error_t err) {
    switch (err) {
        case GLINT_OK: return "Success";
        case GLINT_ERR_INVALID_DATA: return "Invalid data";
        case GLINT_ERR_OUT_OF_MEMORY: return "Out of memory";
        case GLINT_ERR_BUFFER_TOO_SMALL: return "Buffer too small";
        case GLINT_ERR_SCHEMA_MISMATCH: return "Schema mismatch";
        case GLINT_ERR_LIMIT_EXCEEDED: return "Limit exceeded";
        case GLINT_ERR_UNSUPPORTED: return "Unsupported operation";
        default: return "Unknown error";
    }
}
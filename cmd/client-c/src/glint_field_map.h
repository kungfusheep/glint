#ifndef GLINT_FIELD_MAP_H
#define GLINT_FIELD_MAP_H

#include <stddef.h>
#include <string.h>
#include <stdint.h>

// Field mapping structure to associate field names with struct offsets
typedef struct {
    const char* name;
    size_t name_len;
    size_t offset;
    uint8_t wire_type;
} glint_field_map_t;

// Field map registry for a struct type
typedef struct {
    const glint_field_map_t* fields;
    size_t field_count;
} glint_field_registry_t;

// Find field offset by name
static inline size_t glint_find_field_offset(const glint_field_registry_t* registry,
                                            const char* name, size_t name_len) {
    if (!registry || !registry->fields) {
        return 0;
    }
    
    for (size_t i = 0; i < registry->field_count; i++) {
        const glint_field_map_t* field = &registry->fields[i];
        if (field->name_len == name_len && 
            memcmp(field->name, name, name_len) == 0) {
            return field->offset;
        }
    }
    
    // Field not found - return 0 (will write to start of struct)
    return 0;
}

// Macro to simplify field registration
#define GLINT_FIELD(struct_type, field_name, wire_type) \
    { #field_name, sizeof(#field_name) - 1, offsetof(struct_type, field_name), wire_type }

// Example usage for test_struct_t:
// static const glint_field_map_t test_struct_fields[] = {
//     GLINT_FIELD(test_struct_t, bool_value, GLINT_WIRE_BOOL),
//     GLINT_FIELD(test_struct_t, int_value, GLINT_WIRE_INT32),
//     GLINT_FIELD(test_struct_t, uint_value, GLINT_WIRE_UINT32),
//     // ... etc
// };

#endif // GLINT_FIELD_MAP_H
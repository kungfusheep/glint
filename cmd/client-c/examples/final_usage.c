/*
 * Final Glint API - Exactly as we designed it
 */

#include <stdio.h>
#include <stddef.h>
#include <assert.h>
#include <pthread.h>
#include "../src/glint_final.h"

// ============================================================================
// Example 1: The exact example we agreed on
// ============================================================================

// Step 1: Define with G() macros - exactly as designed
GLINT_STRUCT(UserProfile,
    G(uint32_t, id, "user_id"),              // Snake case in wire format
    G(char*, email, "email_address"),
    G(char*, display_name, "displayName"),   // Camel case in wire format
    G(bool, verified, "is_verified"),
    G(time_t, created, "created_at"),
    G(time_t, updated, "updated_at"),
    
    // Internal fields not sent over wire - no G()
    void* db_connection,
    uint32_t cache_flags,
    pthread_mutex_t lock
)

// Step 2: Implement decoder (C macro limitation - need to list fields)
#define GLINT_CURRENT_TYPE UserProfile
GLINT_IMPL(UserProfile,
    F(uint32_t, id, "user_id"),
    F(char_ptr, email, "email_address"),
    F(char_ptr, display_name, "displayName"),
    F(bool, verified, "is_verified"),
    F(time_t, created, "created_at"),
    F(time_t, updated, "updated_at")
    // Internal fields NOT listed - won't be serialized
)
#undef GLINT_CURRENT_TYPE

// ============================================================================
// Example 2: Simple struct
// ============================================================================

GLINT_STRUCT(Point,
    G(int32_t, x, "x"),
    G(int32_t, y, "y")
)

#define GLINT_CURRENT_TYPE Point
GLINT_IMPL(Point,
    F(int32_t, x, "x"),
    F(int32_t, y, "y")
)
#undef GLINT_CURRENT_TYPE

// ============================================================================
// Example 3: Complex struct with nested types
// ============================================================================

GLINT_STRUCT(Entity,
    G(uint64_t, entity_id, "entityId"),      // Camel case wire
    G(char*, name, "name"),
    G(Point, position, "pos"),               // Nested struct
    G(Point, velocity, "vel"),               // Nested struct
    G(bool, active, "isActive"),             // Different wire name
    G(double, health, "health"),
    
    // Internal game engine fields - not serialized
    void* sprite_texture,
    uint32_t animation_frame,
    float* vertex_buffer
)

#define GLINT_CURRENT_TYPE Entity
GLINT_IMPL(Entity,
    F(uint64_t, entity_id, "entityId"),
    F(char_ptr, name, "name"),
    F_NESTED(Point, position, "pos", Point_decoder),
    F_NESTED(Point, velocity, "vel", Point_decoder),
    F(bool, active, "isActive"),
    F(double, health, "health")
)
#undef GLINT_CURRENT_TYPE

// ============================================================================
// Usage Examples
// ============================================================================

void test_user_profile() {
    printf("Testing UserProfile with exact API we designed...\n");
    
    // Create user - pure struct, no overhead!
    UserProfile profile = {0};
    
    // Set internal fields that won't be touched by unmarshal
    profile.cache_flags = 0xDEADBEEF;
    profile.db_connection = (void*)0x12345678;
    pthread_mutex_init(&profile.lock, NULL);
    
    // Simulated Glint data
    uint8_t data[1000] = {/* ... */};
    size_t len = sizeof(data);
    
    // Decode using our clean API
    glint_error_t err = UserProfile_unmarshal(data, len, &profile);
    
    if (err == GLINT_OK) {
        printf("Decoded profile:\n");
        printf("  ID: %u\n", profile.id);
        printf("  Email: %s\n", profile.email);
        printf("  Display Name: %s\n", profile.display_name);
        printf("  Verified: %s\n", profile.verified ? "Yes" : "No");
        
        // Check internal fields are untouched
        assert(profile.cache_flags == 0xDEADBEEF);
        assert(profile.db_connection == (void*)0x12345678);
        printf("  Internal fields preserved ✓\n");
    }
    
    // The decoder holds metadata, not the struct
    printf("\nUserProfile struct size: %zu bytes (pure data)\n", 
           sizeof(UserProfile));
    printf("UserProfile_decoder size: %zu bytes (metadata)\n",
           sizeof(*UserProfile_decoder));
}

void test_nested_entity() {
    printf("\nTesting Entity with nested structs...\n");
    
    Entity entity = {0};
    entity.animation_frame = 42;  // Internal field
    
    uint8_t data[1000] = {/* ... */};
    size_t len = sizeof(data);
    
    Entity_unmarshal(data, len, &entity);
    
    printf("Entity decoded:\n");
    printf("  ID: %llu\n", entity.entity_id);
    printf("  Name: %s\n", entity.name);
    printf("  Position: (%d, %d)\n", entity.position.x, entity.position.y);
    printf("  Velocity: (%d, %d)\n", entity.velocity.x, entity.velocity.y);
    printf("  Active: %s\n", entity.active ? "Yes" : "No");
    printf("  Health: %.1f\n", entity.health);
    printf("  Animation frame (internal): %u\n", entity.animation_frame);
}

void show_memory_efficiency() {
    printf("\nMemory Efficiency:\n");
    printf("==================\n");
    
    // Create many instances - they're just data!
    UserProfile profiles[1000] = {0};
    
    printf("1000 UserProfile instances:\n");
    printf("  Struct memory: %zu bytes (pure data)\n", sizeof(profiles));
    printf("  Decoder memory: %zu bytes (shared metadata)\n", 
           sizeof(*UserProfile_decoder));
    printf("  Total: %zu bytes\n", 
           sizeof(profiles) + sizeof(*UserProfile_decoder));
    
    printf("\nIf metadata was per-instance:\n");
    printf("  Would be: %zu bytes\n", 
           sizeof(profiles) + (1000 * sizeof(*UserProfile_decoder)));
    printf("  We save: %zu bytes!\n", 
           (999 * sizeof(*UserProfile_decoder)));
}

void show_clean_api() {
    printf("\nOur Clean API Design:\n");
    printf("=====================\n\n");
    
    printf("1. Define struct with G() for wire fields:\n");
    printf("   GLINT_STRUCT(UserProfile,\n");
    printf("       G(uint32_t, id, \"user_id\"),\n");
    printf("       G(char*, email, \"email_address\"),\n");
    printf("       G(bool, verified, \"is_verified\"),\n");
    printf("       void* internal_ptr,  // Not serialized\n");
    printf("       uint32_t cache_flags // Not serialized\n");
    printf("   )\n\n");
    
    printf("2. Use the generated functions:\n");
    printf("   UserProfile profile = {0};\n");
    printf("   UserProfile_unmarshal(data, len, &profile);\n\n");
    
    printf("Benefits:\n");
    printf("  ✓ Clean struct-like syntax with G()\n");
    printf("  ✓ Wire name mapping (C name != wire name)\n");
    printf("  ✓ Internal fields not serialized\n");
    printf("  ✓ Structs have zero overhead\n");
    printf("  ✓ Decoder holds all metadata\n");
    printf("  ✓ Schema caching in decoder\n");
}

int main() {
    printf("Glint Final API - Exactly As Designed\n");
    printf("======================================\n\n");
    
    test_user_profile();
    test_nested_entity();
    show_memory_efficiency();
    show_clean_api();
    
    return 0;
}
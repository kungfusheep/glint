#ifndef GLINT_ARENA_H
#define GLINT_ARENA_H

#include <stddef.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

typedef struct arena_block {
    struct arena_block* next;
    size_t size;
    size_t used;
    uint8_t data[];
} arena_block_t;

typedef struct glint_arena {
    arena_block_t* current;
    arena_block_t* first;
    size_t block_size;
} glint_arena_t;

static inline glint_arena_t* arena_create(size_t initial_size) {
    glint_arena_t* arena = malloc(sizeof(glint_arena_t));
    if (!arena) return NULL;
    
    arena_block_t* block = malloc(sizeof(arena_block_t) + initial_size);
    if (!block) {
        free(arena);
        return NULL;
    }
    
    block->next = NULL;
    block->size = initial_size;
    block->used = 0;
    
    arena->first = block;
    arena->current = block;
    arena->block_size = initial_size;
    
    return arena;
}

static inline void* arena_alloc(glint_arena_t* arena, size_t size) {
    // Align to 8 bytes
    size = (size + 7) & ~7;
    
    if (arena->current->used + size > arena->current->size) {
        // Need new block
        size_t new_size = arena->block_size;
        if (size > new_size) new_size = size;
        
        arena_block_t* block = malloc(sizeof(arena_block_t) + new_size);
        if (!block) return NULL;
        
        block->next = NULL;
        block->size = new_size;
        block->used = 0;
        
        arena->current->next = block;
        arena->current = block;
    }
    
    void* ptr = arena->current->data + arena->current->used;
    arena->current->used += size;
    return ptr;
}

static inline void arena_free(glint_arena_t* arena) {
    if (!arena) return;
    
    arena_block_t* block = arena->first;
    while (block) {
        arena_block_t* next = block->next;
        free(block);
        block = next;
    }
    free(arena);
}

#endif

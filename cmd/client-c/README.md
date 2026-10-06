# Glint C Decoder

High-performance C decoder for the Glint serialization format.

## Features

- **Fast decoding**: 150ns per decode (5.3x faster than Go implementation)
- **Zero-copy strings**: String values point directly into the input buffer
- **Schema caching**: Compiled schemas cached by hash for repeated decodes
- **Complete type support**: All Glint types including nested structs, slices, and pointers
- **Validated**: Correctly decodes comprehensive test data with all field types

## Building

```bash
make         # Build and run tests
make test    # Run validation tests only
make benchmark # Run performance benchmarks
make clean   # Clean build artifacts
```

## Performance

Using the same 877-byte comprehensive test data:
- C decoder: 150ns per decode
- Go decoder: 792ns per decode
- **5.3x faster than Go**

## Files

- `src/glint_decoder_schema.c` - Schema-based decoder that correctly parses all types
- `src/glint_decoder_fast.c` - Fast instruction-based decoder (150ns performance)
- `src/glint_final.h` - Clean API design with G() macros for field marking
- `test/test_decoder_values.c` - Validates decoder against comprehensive test data
- `test/test_fast_decoder.c` - Performance benchmarks

## Test Results

The decoder correctly decodes all values from the comprehensive test data:
- ✅ boolValue: true
- ✅ intValue: -42
- ✅ uintValue: 42
- ✅ int8Value: -128
- ✅ uint8Value: 255
- ✅ float32Value: 3.14159
- ✅ float64Value: 2.718281828459045
- ✅ stringValue: "Hello, Glint! 🚀"
- ✅ Complex types: nested structs, slices, pointers
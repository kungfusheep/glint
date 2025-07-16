# Glint TypeScript Decoder

A zero-dependency TypeScript implementation of the Glint binary format decoder.

## Features

- 🚀 **Zero dependencies** - Pure TypeScript/JavaScript implementation
- 🔒 **Type-safe** - Full TypeScript support with strict typing
- ⚡ **Fast** - Optimized binary reading with DataView
- 🛡️ **Safe** - Configurable limits to prevent DoS attacks
- 📦 **Small** - Minimal bundle size

## Installation

```bash
npm install
npm run build
```

## Usage

```typescript
import { decode, GlintDecoder } from './dist';
import * as fs from 'fs';

// Simple decode
const data = fs.readFileSync('data.glint');
const result = decode(data);
console.log(result);

// With custom limits
const decoder = new GlintDecoder({
  maxStringLength: 1024 * 1024,  // 1MB strings
  maxArrayLength: 10000,         // 10K array elements
});
const result = decoder.decode(data);
```

## API

### `decode(data: Uint8Array, options?: DecoderOptions): DecodedObject`

Convenience function to decode a Glint document.

### `class GlintDecoder`

Main decoder class with configurable options.

#### Constructor Options

- `maxStringLength` - Maximum string length (default: 10MB)
- `maxArrayLength` - Maximum array elements (default: 1M)
- `maxMapSize` - Maximum map entries (default: 100K)
- `maxNestingDepth` - Maximum nesting depth (default: 100)

## Supported Types

- ✅ All primitive types (bool, int, uint, float, string)
- ✅ Binary data ([]byte)
- ✅ Arrays/slices
- ✅ Nested structs
- ✅ Pointers (nullable fields)
- ✅ time.Time
- ⚠️ Maps (basic support)

## Development

```bash
# Build
npm run build

# Run tests
npm test
npm run test:all  # Run all test suites

# Run benchmarks
npm run bench                              # Simple benchmark (50k iterations)
npm run bench:go-style                     # Go-style adaptive benchmarking
node dist/test/simple-benchmark.js -n 100000  # Custom iterations

# Run example
npm run example
```

## Performance

### Optimized Implementation Results

```
Simple document:  Glint 522µs vs JSON 147µs (3.6x slower)
Complex document: Glint 1185µs vs JSON 286µs (4.2x slower)
```

**Optimizations Applied:**
- Varint decoding: 6x faster (unrolled loops for common cases)
- String decoding: 3x faster (optimized memory handling)
- Overall decoder: 1.8x faster than original

**Space Efficiency:**
- Simple: Glint 24B vs JSON 26B (7.7% smaller)
- Complex: Glint 51B vs JSON 64B (20.3% smaller)

### Running Benchmarks

```bash
npm run bench          # Main benchmark suite
npm run bench:micro    # Micro-benchmarks for operations
npm run bench:decoder  # Compare original vs optimized
npm run bench:reader   # Compare reader implementations
```

### Profiling

For detailed performance analysis using Node.js profiling tools:

```bash
# Generate V8 profiling data
node --prof dist/test/go-style-benchmark.js
node --prof-process isolate-*.log > profile.txt

# CPU profiling with Chrome DevTools
node --inspect dist/test/go-style-benchmark.js
# Open chrome://inspect in Chrome

# Memory profiling
node --expose-gc --inspect dist/test/decoder-comparison.js
```

## Performance Notes

- Uses `DataView` for endian-safe type conversions
- Single `TextDecoder` instance reused for all strings
- Minimal object allocations
- No external dependencies

## Limitations

- Map support is basic (string keys only currently)
- No encoding support (decode only)
- No trust mode optimization yet
- JavaScript number precision limits for int64/uint64

## License

MIT
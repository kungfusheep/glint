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

## Performance Benchmarks

Current TypeScript implementation results (M3 Pro, statistically significant):

```
BenchmarkGlintSimpleDecode      957,630       1,502.6 ns/op
BenchmarkGlintComplexDecode     809,388       2,244.5 ns/op
BenchmarkJsonSimpleParse     13,382,098         140.6 ns/op
BenchmarkJsonComplexParse     7,524,760         264.2 ns/op
```

**Performance Analysis:**
- Simple data: JSON is 10.7x faster than Glint (140.6ns vs 1,502.6ns)
- Complex data: JSON is 8.5x faster than Glint (264.2ns vs 2,244.5ns)

**Space Efficiency:**
- Simple: Glint 24B vs JSON 25B (4.0% savings)
- Complex: Glint 51B vs JSON 63B (19.0% savings)

The TypeScript implementation prioritizes correctness and safety over raw speed, making it ~10x slower than JSON but with significant space savings on complex data.

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
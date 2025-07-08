![logo](./logo.svg)


# Glint

Glint brings binary serialization performance to Go without the complexity. Encode and decode your existing structs at speeds that match or exceed code-generated solutions.

## Performance at a Glance

<table>
<tr>
<td>

**Encoding** (ns/op)
```
Glint:     60 - 493
Protobuf:  174 - 1,509  
JSON:      694 - 84,330
```

</td>
<td>

**Decoding** (ns/op)
```
Glint:     185 - 775
Protobuf:  213 - 1,971
JSON:      3,921 - 510,590
```

</td>
<td>

**Key Benefits**
- Zero allocations
- Zero code generation
- ~70% smaller than JSON
- ~40% smaller than Protobuf
- Works with existing structs

</td>
</tr>
</table>

See [detailed benchmarks](PERFORMANCE.md) for comprehensive performance data.

## Quick Start

```go
import "github.com/kungfusheep/glint"

// Define your struct - no special annotations required
type Person struct {
    Name string `glint:"name"`
    Age  int    `glint:"age"`
}

// Create encoder/decoder once, reuse many times
var (
    encoder = glint.NewEncoder[Person]()
    decoder = glint.NewDecoder[Person]()
)

// Encode
buffer := glint.NewBufferFromPool()
defer buffer.ReturnToPool()
encoder.Marshal(&Person{Name: "Alice", Age: 30}, buffer)

// Decode  
var person Person
err := decoder.Unmarshal(buffer.Bytes, &person)
```

## Why Choose Glint?

### 🚀 Exceptional Performance
- **Sub-microsecond operations**: As low as 60ns for simple structs
- **Zero allocations**: Careful design eliminates heap allocations
- **Compact format**: 70% smaller than JSON, 40% smaller than Protobuf

### 🛠️ Developer Friendly
- **No code generation**: Work directly with your Go structs
- **No schema files**: Self-describing format includes schemas
- **Simple API**: Familiar Marshal/Unmarshal pattern

### 🔄 Production Ready
- **Backward compatibility**: Add fields freely, remove fields safely
- **Forward compatibility**: Older versions ignore unknown fields  
- **Type safety**: Schema validation ensures type compatibility
- **Memory protection**: Configurable limits prevent DoS attacks

## Backward Compatibility

Glint's compatibility strategy is simple and powerful:

- ✅ **Add new fields** - Older versions ignore them automatically
- ✅ **Remove fields** - Newer versions handle missing fields gracefully  
- ❌ **Change field types** - The only breaking change (schema mismatch)

This approach gives you flexibility to evolve schemas naturally:
```go
// Version 1
type User struct {
    ID   string `glint:"id"`
    Name string `glint:"name"`
}

// Version 2 - Safe to deploy alongside v1
type User struct {
    ID       string    `glint:"id"`
    Name     string    `glint:"name"`
    Email    string    `glint:"email"`     // New field - ignored by v1
    Created  time.Time `glint:"created"`   // New field - ignored by v1
    // Age int        `glint:"age"`       // Removed - v2 handles absence
}
```

No version numbers, no migration scripts - just natural schema evolution.

## Installation

```bash
go get github.com/kungfusheep/glint
```

## Core Concepts

### Supported Types

Glint works with standard Go types out of the box:

- **Basic types**: int/8/16/32/64, uint/8/16/32/64, float32/64, string, bool, time.Time
- **Composite types**: structs, slices, maps
- **Pointers**: Automatic nil handling
- **Custom types**: Via `MarshalBinary`/`UnmarshalBinary` interfaces

### Memory Protection

Glint provides configurable limits to prevent malicious inputs from exhausting memory:

```go
// Custom limits for untrusted data
decoder := glint.NewDecoderWithLimits[MyStruct](glint.DecodeLimits{
    MaxByteSliceLen: 10 * 1024 * 1024,  // 10MB
    MaxStringLen:    1 * 1024 * 1024,   // 1MB
})
```

### Struct Tags

Control field encoding with struct tags:

```go
type User struct {
    ID        string    `glint:"id"`
    Secret    string    `glint:"-"`              // Skip this field
    Data      []byte    `glint:"data,copy"`      // Copy bytes instead of referencing
    CreatedAt time.Time `glint:"created_at"`
}
```

### Custom Types

Implement custom encoding for your types:

```go
type UUID [16]byte

func (u UUID) MarshalBinary() []byte {
    return u[:]
}

func (u *UUID) UnmarshalBinary(data []byte) {
    copy(u[:], data)
}

type User struct {
    ID UUID `glint:"id,encoder"`  // Uses custom encoding
}
```


### Trust Mode (Schema Optimization)

For high-frequency communication between services, skip schema transmission:

```go
// Server side - get trust header after first decode
trustHeader := glint.NewTrustHeader(decoder.impl)
response.Header.Set(trustHeader.Key(), trustHeader.Value())

// Client side - use trusted encoding
trustee := glint.HTTPTrustee(request)
buffer := glint.NewBufferWithTrust(trustee, encoder.impl)
encoder.Marshal(&data, buffer)  // Smaller payload, no schema
```

### Manual Document Building

For dynamic document construction without structs:

```go
doc := &glint.DocumentBuilder{}
doc.AppendString("name", "Alice").
    AppendInt("age", 30).
    AppendSlice("tags", glint.SliceBuilder{}.
        AppendStringSlice([]string{"admin", "user"}))

data := doc.Bytes()
```

### Debugging Tools

Inspect Glint documents without decoding:

```go
// Pretty print any Glint document
glint.Print(encodedData)

// Use with fmt for different formats
doc := glint.Document(encodedData)
fmt.Printf("%s", doc)   // Tree view
fmt.Printf("%+v", doc)  // Tree + hex
fmt.Printf("%x", doc)   // Raw hex
```

## CLI Tool

Glint includes a powerful CLI for working with binary data:

```bash
go install github.com/kungfusheep/glint/cmd/glint@latest

# Inspect binary files
cat mydata.glint | glint

# Decode API responses
curl -s https://api.example.com/user | glint

# Extract specific fields
cat mydata.glint | glint get user.name

# Generate Go structs from data
cat mydata.glint | glint generate go package.StructName
```

See the [CLI documentation](cmd/glint/README.md) for more features.

## Use Cases

Glint excels in scenarios where:

- **Performance matters**: High-throughput services, real-time systems
- **Schema flexibility is needed**: Microservices with independent deployment
- **Storage is constrained**: IoT devices, embedded systems, high-volume logging
- **Go-native solution preferred**: No external tooling or code generation

## License

See [LICENSE](LICENSE) for details.


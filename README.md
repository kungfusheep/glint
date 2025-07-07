![logo](./logo.svg)


# Glint

Glint is a hierarchical binary data format designed for efficient encoding and decoding of structured data. It provides a flexible schema-based approach to serialise and deserialise data, making it suitable for high-performance applications.

## Why Glint?

- **High Performance**: Optimised for speed and transfer size whilst retaining the flexibility of JSON-like structures.
- **No Code Generation**: Glint does not require code generation
- **Schema Evolution**: Built-in versioning with CRC32 hashing allows safe schema changes and backward compatibility.
- **Type Safety**: Only types are checked during schema parsing, giving you the freedom to evolve your data structures without breaking changes.
- **Self-Describing**: Embedded schemas make data portable and understandable by intermediates without external schema files
- **Compact**: Efficient binary encoding reduces data size when compared to JSON or Protocol Buffers
- **Zero Dependencies**: Pure Go implementation with no external dependencies
- **Compatibility**: Compatible with existing Go types and structures

**Use Cases:**
- Real-time data processing pipelines with evolving schemas
- Distributed caching where schema compatibility is critical
- Game networking protocols with complex nested data
- Database storage engines with evolving record formats

## Features

- **Schema-based encoding**: Binary schemas are included in the encoded data, allowing for easy versioning and compatibility.
- **Dynamic typing**: Support for various data types including integers, floats, strings, slices, maps, and structs.
- **Nillable and slice handling**: Encode and decode nillable fields and slices seamlessly.
- **Custom encoding/decoding**: Implement custom binary encoders and decoders for your types.
- **Trusted schema mode**: Optimise performance by omitting schema transmission when the schema is already known.
- **Memory protection**: Configurable decode limits prevent memory exhaustion attacks from malicious data.

## Usage

### Type-Safe Generic API (Recommended)

Glint provides a modern type-safe API using Go generics. This is the recommended approach for new code as it provides compile-time type safety and cleaner syntax.

### Security and Memory Protection

Glint includes configurable decode limits to prevent memory exhaustion attacks during decoding. These limits provide protection against malicious data that could attempt to allocate excessive memory.

#### Basic Usage with Default Limits

```go
// Default decoder with sensible limits
decoder := glint.NewDecoder[MyStruct]()
```

#### Custom Limits Configuration

```go
// Configure custom limits for strict environments
limits := glint.DecodeLimits{
    MaxByteSliceLen: 10 * 1024 * 1024,  // 10MB max byte slices
    MaxSliceInitCap: 1000,               // Cap initial slice allocations
    MaxSchemaSize:   512 * 1024,         // 512KB max schema size
    MaxStringLen:    5 * 1024 * 1024,    // 5MB max string length
}

decoder := glint.NewDecoderWithLimits[MyStruct](limits)
```

**Default limits provide reasonable protection for most use cases:**
- MaxByteSliceLen: 100MB
- MaxSliceInitCap: 10,000 elements  
- MaxSchemaSize: 1MB
- MaxStringLen: 50MB

Setting a limit to 0 disables that specific check.

### Encoding Data

To encode data, create an `Encoder` instance and use it to serialise your data:

```go
package main

import (
	"fmt"
	"github.com/kungfusheep/glint"
)

type MyStruct struct {
	Name string `glint:"name"`
	Age  int    `glint:"age"`
}
var encoder = glint.NewEncoder[MyStruct]()

func main() {
	data := MyStruct{Name: "Alice", Age: 30}

	buffer := glint.NewBufferFromPool()
	defer buffer.ReturnToPool()

	encoder.Marshal(&data, buffer)
	fmt.Printf("Encoded %d bytes: %x\n", len(buffer.Bytes), buffer.Bytes)
}
```

### Decoding Data

To decode data, create a `Decoder` instance and use it to deserialise your data:

```go
package main

import (
	"fmt"
	"github.com/kungfusheep/glint"
)

type MyStruct struct {
	Name string `glint:"name"`
	Age  int    `glint:"age"`
}
var decoder = glint.NewDecoder[MyStruct]()

func main() {
	// Example with actual encoded data from previous example
	encodedData := []byte{0x00, 0xec, 0x95, 0x66, 0x5d, 0x0b, 0x0e, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x02, 0x03, 0x61, 0x67, 0x65, 0x05, 0x41, 0x6c, 0x69, 0x63, 0x65, 0x3c}
	
	var data MyStruct

	err := decoder.Unmarshal(encodedData, &data)
	if err != nil {
		fmt.Println("Error decoding data:", err)
		return
	}

	fmt.Printf("Decoded data: %+v\n", data) // Output: {Name:Alice Age:30}
}
```

## Documentation

### Core Types

- `Encoder[T]`: Type-safe encoder that encodes values of type T into binary format.
- `Decoder[T]`: Type-safe decoder that decodes binary data into Go structs of type T.
- `Buffer`: A reusable buffer for encoding and decoding operations.

### Supported Types
    
- **Primitive Types**: `int`, `int8`, `int16`, `int32`, `int64`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `float32`, `float64`, `string` 
- **Composite Types**: `struct`, `map`, `slice`
- **Pointers**: Pointers to any of the above types are supported.
- **Custom Types**: You can define your own types and implement the `binaryEncoder` and `binaryDecoder` interfaces for custom encoding/decoding.
- **Trusted Schema**: When using the `TrustedSchema` option, the schema is not transmitted with the data, which can improve performance in scenarios where the schema is already known.
- **Schema Versioning**: Glint supports schema versioning, allowing you to evolve your data structures over time without breaking compatibility.

### Struct Tags

You can use struct tags to specify the field names and options for encoding/decoding:

```go 
type MyStruct struct {
    Name string `glint:"name"`
    Age  int    `glint:"age"`
}
```

The `glint` tag specifies the name of the field in the encoded data. You can also use additional options in the tag to control encoding behavior.  

### Encoding Options 

You can customise the encoding behavior using options in struct tags. Options are specified after the field name, separated by commas:

```go
type MyStruct struct {
    Field1 int    `glint:"field1"`
    Field2 string `glint:"field2,copy"`
}
```

Available options:

- **`encoder`**: Use custom binary encoding for the field. The field type must implement the `binaryEncoder` interface (with a `MarshalBinary() []byte` method) for encoding and `binaryDecoder` interface (with an `UnmarshalBinary([]byte)` method) for decoding.
  ```go
  CustomField MyCustomType `glint:"custom,encoder"`
  ```

- **`stringer`**: Encode structs as strings using their `String()` method. The field type must implement the `fmt.Stringer` interface.
  ```go
  StringableField MyStringType `glint:"stringable,stringer"`
  ```

- **`copy`**: Used with string fields to enable copying behavior during encoding/decoding operations, by default strings are read by reference to the document that contains them.
  ```go
  CopiedString string `glint:"cstring,copy"`
  ```


### Custom Types
Glint allows you to define custom types and implement your own encoding/decoding logic. This is useful when you want to control how a specific type is serialised or deserialised.

To implement custom encoding/decoding for a field, your type should implement the `binaryEncoder` and `binaryDecoder` interfaces:

```go
type MyType struct {
	Value string
}

func (m MyType) MarshalBinary() []byte {
	return []byte(m.Value)
}

func (m *MyType) UnmarshalBinary(data []byte) {
	m.Value = string(data)
}
```

### Document Builder

For manual document construction without Go structs, use `DocumentBuilder`:

```go
func main() {
	doc := &glint.DocumentBuilder{}
	
	// Chain method calls to build document
	doc.AppendString("name", "Alice").
		AppendInt("age", 30).
		AppendBool("active", true)
	
	// Get the bytes
	data := doc.Bytes()
	fmt.Printf("Document: %x\n", data)
}
```

For building slices manually:

```go
func main() {
	slice := &glint.SliceBuilder{}
	slice.AppendStringSlice([]string{"item1", "item2", "item3"})
	
	// Use slice in document
	doc := &glint.DocumentBuilder{}
	doc.AppendSlice("items", *slice)
	
	data := doc.Bytes()
}
```

### Document Walker

To inspect glint documents without deserialising, implement the `Visitor` interface:

```go
type MyVisitor struct{}

func (v MyVisitor) VisitFlags(flags byte) error                                              { return nil }
func (v MyVisitor) VisitSchemaHash(hash []byte) error                                        { return nil }
func (v MyVisitor) VisitArrayStart(name string, wire glint.WireType, length int) error   { return nil }
func (v MyVisitor) VisitArrayEnd(name string) error                                          { return nil }
func (v MyVisitor) VisitStructStart(name string) error                                       { return nil }
func (v MyVisitor) VisitStructEnd(name string) error                                         { return nil }

func (v MyVisitor) VisitField(name string, wire glint.WireType, body glint.Reader) (glint.Reader, error) {
	// Handle each field as needed
	fmt.Printf("Field: %s\n", name)
	return body, nil
}

func main() {
	data := []byte{...} // your glint document
	visitor := MyVisitor{}
	
	err := glint.Walk(data, visitor)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
```

### Debugging

For debugging glint documents in Go code, use the `Print` function:

```go
func main() {
	// Create some test data
	doc := &glint.DocumentBuilder{}
	doc.AppendString("name", "Alice").AppendInt("age", 30)
	data := doc.Bytes()
	
	// Print human-readable representation
	glint.Print(data)
}
```

This outputs a tree-like structure showing the schema and values, similar to the CLI tool output.

### Document Type for Printf

The `Document` type alias provides integration with Go's `fmt` package for easy pretty-printing:

```go
func main() {
	// Create document
	doc := &glint.DocumentBuilder{}
	doc.AppendString("name", "Alice").AppendInt("age", 30)
	
	// Convert to Document type
	data := glint.Document(doc.Bytes())
	
	// Use with fmt functions
	fmt.Printf("Document: %s\n", data)        // Pretty tree view
	fmt.Printf("Verbose: %+v\n", data)        // Tree view + hex
	fmt.Printf("Hex: %x\n", data)             // Raw hex
	fmt.Println(data)                         // Uses String() method
}
```

**Supported format verbs:**
- `%s`, `%v` - Pretty-printed tree structure
- `%+v` - Verbose format with both tree and hex representation  
- `%x`, `%X` - Hex representation (lowercase/uppercase)
- `%q` - Quoted byte representation

## CLI Tool

The `glint` command-line tool provides utilities for inspecting and manipulating glint documents.

### Installation

```bash
go install github.com/kungfusheep/glint/cmd/glint@latest
```

### Features

- Inspect glint documents with tree visualization
- Extract specific fields using dot notation
- Validate document integrity
- Debug wire format with varint/zigzag decoders
- Convert between formats (JSON, YAML, etc.)

For detailed documentation and examples, see the [glint CLI documentation](cmd/glint/README.md).


## Testing

To run the tests:

```bash
go test
go test -bench=.
```


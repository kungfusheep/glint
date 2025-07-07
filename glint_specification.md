# Glint Binary Serialization Format Specification

## Overview

The Glint format is a high-performance, schema-driven binary serialization protocol for hierarchical data. It is designed for use with structured data, like Go's `struct`, supporting complex types (maps, slices, pointers, time), with self-describing schemas and versioning via CRC32. The format prioritizes efficient encoding/decoding, minimal allocations, and forward/backward compatibility.

---

## Table of Contents

- [1. Document Structure](#1-document-structure)
- [2. Schema Section](#2-schema-section)
- [3. Data Section](#3-data-section)
- [4. Wire Types](#4-wire-types)
- [5. Field and Type Encoding](#5-field-and-type-encoding)
- [6. Slices, Maps, and Pointers](#6-slices-maps-and-pointers)
- [7. Trusted Schema Optimization](#7-trusted-schema-optimization)
- [8. Dynamic Values](#8-dynamic-values)
- [9. Example](#9-example)
- [10. Versioning & Compatibility](#10-versioning--compatibility)

---

## 1. Document Structure

A Glint document consists of the following sections, in order:

| Offset      | Field         | Size        | Description                          |
|-------------|--------------|-------------|--------------------------------------|
| 0           | Flags        | 1 byte      | Bit flags for format features        |
| 1           | CRC32 Hash   | 4 bytes     | CRC32 of the schema section          |
| 5           | Schema Size  | varint      | Length of schema section             |
| ...         | Schema       | variable    | Self-describing schema definition    |
| ...         | Data         | variable    | Encoded values (body)                |

- **Flags:** Reserved for future use (e.g., trusted schema).
- **CRC32:** Little-endian. Used to identify and trust schema.
- **Schema Size:** Unsigned LEB128 varint.
- **Schema:** See below for encoding details.
- **Data:** Values encoded according to the schema.

---

## 2. Schema Section

The schema is a TLV (Type-Length-Value) list describing all fields of the encoded struct:

For each field:
```
[WireType (varint)][FieldNameLen (1 byte)][FieldName (bytes)][Subschema (if needed)]
```

- **WireType:** (see [Wire Types](#4-wire-types))
- **FieldNameLen:** Unsigned 8-bit
- **FieldName:** ASCII/UTF-8
- **Subschema:** Included for fields that are structs, slices, or maps. The subschema is itself a Glint schema.

### Example Schema Entry

For a field `age int32`:
```
[WireInt32][3][97 103 101]
```

For a field `friends []Person`:
```
[WireSliceFlag|WireStruct][7][102 114 105 101 110 100 115][Person schema...]
```

---

## 3. Data Section

For each field, the value is encoded in the order specified by the schema. Types are encoded as follows:

- **Integral / Float / Bool / String / Time:** Standard binary encoding, with length prefix for variable-length types.
- **Structs:** Field values in schema order, recurse into subschema.
- **Slices/Arrays:** `[Length (varint)][Element1][Element2]...`
- **Maps:** `[Length (varint)][Key1][Value1][Key2][Value2]...`
- **Pointers:** `[Present (1 byte)][Value?]` (1 = present, 0 = nil)
- **nil Pointers/Slices/Maps:** Length or present byte is 0.

---

## 4. Wire Types

Wire types are used to identify the on-the-wire encoding for each field.

| Name         | Value  | Description           |
|--------------|--------|----------------------|
| WireBool     | 1      | bool                 |
| WireInt      | 2      | int                  |
| WireInt8     | 3      | int8                 |
| WireInt16    | 4      | int16                |
| WireInt32    | 5      | int32                |
| WireInt64    | 6      | int64                |
| WireUint     | 7      | uint                 |
| WireUint8    | 8      | uint8                |
| WireUint16   | 9      | uint16               |
| WireUint32   | 10     | uint32               |
| WireUint64   | 11     | uint64               |
| WireFloat32  | 12     | float32              |
| WireFloat64  | 13     | float64              |
| WireString   | 14     | string               |
| WireBytes    | 15     | []byte               |
| WireStruct   | 16     | struct               |
| WireMap      | 17     | map                  |
| WireTime     | 18     | time.Time            |

Modifiers:
- `WireSliceFlag` (0x20): Field is a slice/array
- `WirePtrFlag`   (0x40): Field is a pointer
- `WireSliceElemPtr` (0x80): Field is a slice of pointers

**Composite:** Modifiers are bitwise OR'ed with base type.

---

## 5. Field and Type Encoding

### Scalar Types

- **Int/Uint:** Zigzag varint encoding for signed, LEB128 for unsigned.
- **Bool:** 1 byte; 0 = false, 1 = true.
- **Float32/64:** IEEE-754, stored as varint of raw bits.
- **String/Bytes:** `[Length (varint)][Data]`
- **time.Time:** Encoded via `time.Time.MarshalBinary`.

### Structs

- Recursively encoded using their own schema.

### Slices

- `[Length (varint)][Elem1][Elem2]...`

### Maps

- `[Length (varint)][Key1][Value1][Key2][Value2]...`
- Map key and value types are described in the schema.

### Pointers

- `[Present (1 byte)][Value?]`
- If present byte is 0, value is nil and omitted.

---

## 6. Slices, Maps, and Pointers

- Slices/arrays: WireSliceFlag | element wire type
- Maps: WireMap, followed by key and value types in schema
- Slices of structs/maps: Nested schema describes element type
- Multi-dimensional slices: Nested WireSliceFlag as needed
- nil slices/maps: length is 0

---

## 7. Trusted Schema Optimization

If the CRC32 hash in the document matches a "trusted" schema known to the decoder, the schema section can be omitted from the data, reducing overhead for repeated types.

To use this:
- Client sends a custom header (e.g., `X-Glint-Trust: <hash>`)
- Server omits schema if hash matches

---

## 8. Dynamic Values

Glint can encode individual values with type tags for dynamic (interface{}) storage:

```
[WireType (varint)][Value]
```

Supports scalars, slices, and pointers.

---

## 9. Example

### Example Go Struct

```go
type User struct {
    Name    string    `glint:"name"`
    Age     int32     `glint:"age"`
    Friends []string  `glint:"friends"`
}
```

### Schema (pseudo-TLV):

- Field 1: WireString, "name"
- Field 2: WireInt32, "age"
- Field 3: WireString|WireSliceFlag , "friends"

### Data:

- [name string][age int32][friends slice]

---

## 10. Versioning & Compatibility

- Field order and names are set by schema, not struct order.
- Unknown fields in input are skipped.
- Fields missing from input are left as zero values.
- Schema changes (e.g., adding/removing fields) are supported as long as field names and types are changed in a compatible way.

---

## References

- [Glint Source Code](https://github.com/kungfusheep/glint/blob/main/glint.go)
- Go Reflection, `unsafe` package documentation

---

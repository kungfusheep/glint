package glint

import (
	"fmt"
	"hash/crc32"
	"reflect"
	"time"
	"unsafe"
)

// Encoder handles type-safe encoding of type T
type Encoder[T any] struct {
	impl *encoderImpl
}

// NewEncoder builds an Encoder using type information extracted from a blank struct instance.
// Create only ONE encoder per type - the encoder is safe for concurrent use.
// The blank struct's type must exactly match what you'll pass to Marshal.
// e.g mystructEncoder := glint.NewEncoder(mystruct{})
//
// Fields that we wish to encode, on `mystruct` in the example above, carry tags that look like this
//
//	type mystruct struct {
//		name string `glint:"name"`
//		age  uint8  `glint:"age"`
//	}
//
// When `Marshal` encodes data into the Buffer, these tag names are used as the schema field names
func NewEncoder[T any]() *Encoder[T] {
	var zero T
	impl := newEncoder(zero)
	return &Encoder[T]{impl: impl}
}

// Marshal encodes a value of type T into the supplied buffer
func (e *Encoder[T]) Marshal(v *T, buf *Buffer) {
	e.impl.Marshal(v, buf)
}

// Schema retrieves this encoder's schema, excluding version and hash bytes
func (e *Encoder[T]) Schema() *Buffer {
	return e.impl.Schema()
}

// ClearSchema wipes all schema data from this encoder.
func (e *Encoder[T]) ClearSchema() {
	e.impl.ClearSchema()
}

// encoderImpl holds the internal encoding state - always construct via `newEncoder`
type encoderImpl struct {
	instructions []encodeInstruction // encoding operations to execute for this struct
	header       Buffer              // header bytes (1 flag, 4 crc32, 1 zero) for trusted schema mode
	schema       Buffer              // complete schema data with header included
}

// encoder defines the required methods for all encoder types (Encoder, SliceEncoder, MapEncoder)
type encoder interface {
	Schema() *Buffer      // get schema without version/hash bytes
	ClearSchema()         // reset schema buffer to empty
	Marshal(any, *Buffer) // encode a complete document
}

// encodeInstruction represents a single encoding operation that can be executed at runtime.
// Either offset or fun is used, never both. Using a concrete type improves performance.
type encodeInstruction struct {
	wire   WireType                      // determines fast path selection in Marshal (e.g. strings)
	offset uintptr                       // field offset for fast path operations
	fun    func(unsafe.Pointer, *Buffer) // fallback encoder when fast paths don't apply
	tag    string                        // struct field name from tag
	subenc *encoderImpl                  // encoder for nested struct types
}

// Schema extracts the raw schema data, stripping version and hash prefixes
func (e *encoderImpl) Schema() *Buffer {
	if len(e.schema.Bytes) < 5 {
		return &Buffer{}
	}
	return &Buffer{Bytes: e.schema.Bytes[5:]}
}

// ClearSchema resets the schema buffer, discarding all data
func (e *encoderImpl) ClearSchema() {
	e.schema.Bytes = nil
}

func newEncoder(t any) *encoderImpl {
	return newEncoderUsingTag(t, "glint")
}

// newEncoderUsingTag is primarily for internal use.
//
// Like newEncoder but accepts a custom struct tag name for framework integration (e.g. "rpc").
func newEncoderUsingTag(t any, tagName string) *encoderImpl {
	e := &encoderImpl{}

	tt := reflect.TypeOf(t)

	e.schema.Bytes = append(e.schema.Bytes, 0)                     // flag byte (8 bits)
	e.schema.Bytes = append(e.schema.Bytes, []byte{0, 0, 0, 0}...) // placeholder for 32-bit checksum

	switch tt.Kind() {
	case reflect.Slice:

		panic("must be of type struct")

	case reflect.Struct:
		e.buildStruct(tt, tagName)
	}

	// embed the schema's checksum within itself
	crc := crc32.ChecksumIEEE(e.schema.Bytes[5:])
	b := e.schema.Bytes[1:5]
	b[0] = byte(crc)
	b[1] = byte(crc >> 8)
	b[2] = byte(crc >> 16)
	b[3] = byte(crc >> 24)

	e.header.Bytes = make([]byte, 6) // 6 bytes: 5 for header + 1 zero-length schema marker
	copy(e.header.Bytes, e.schema.Bytes[:5])
	return e
}

// binaryEncoder allows types to handle their own encoding when tagged with 'encoder'.
// The type converts itself to bytes for inclusion in the glint buffer.
type binaryEncoder interface {
	MarshalBinary() []byte
}

// binaryDecoder enables types to handle their own decoding when tagged with 'encoder'.
// The type reconstructs itself from a byte slice.
type binaryDecoder interface {
	UnmarshalBinary(bytes []byte)
}

// buildStruct generates encoding instructions based on the struct type's fields.
func (e *encoderImpl) buildStruct(t reflect.Type, usingTagName string) {

	bytes := []byte{}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		tag, opts := parseTag(f.Tag.Get(usingTagName))
		if tag == "" {
			continue
		}

		var fun func(unsafe.Pointer, *Buffer)
		var wire WireType
		var enc encoder

		pointerWrap := false

		k := f.Type.Kind()
		if k == reflect.Pointer {
			pointerWrap = true
			k = f.Type.Elem().Kind()
		}

		switch k {
		case reflect.Uint8:
			wire = WireUint8
			fun = func(p unsafe.Pointer, b *Buffer) {
				b.AppendUint8(*(*uint8)(p))
			}
		case reflect.Uint16:
			wire = WireUint16
			fun = func(p unsafe.Pointer, b *Buffer) {
				b.AppendUint16(*(*uint16)(p))
			}
		case reflect.Uint32:
			wire = WireUint32
			fun = func(p unsafe.Pointer, b *Buffer) {
				b.AppendUint32(*(*uint32)(p))
			}
		case reflect.Uint64:
			wire = WireUint64
			fun = func(p unsafe.Pointer, b *Buffer) {
				b.AppendUint64(*(*uint64)(p))
			}
		case reflect.Uint:
			wire = WireUint
			fun = func(p unsafe.Pointer, b *Buffer) {
				b.AppendUint(*(*uint)(p))
			}
		case reflect.Int8:
			wire = WireInt8
			fun = func(p unsafe.Pointer, b *Buffer) {
				b.AppendInt8(*(*int8)(p))
			}

		case reflect.Int16:
			wire = WireInt16
			fun = func(p unsafe.Pointer, b *Buffer) {
				b.AppendInt16(*(*int16)(p))
			}
		case reflect.Int32:
			wire = WireInt32
			fun = func(p unsafe.Pointer, b *Buffer) {
				b.AppendInt32(*(*int32)(p))
			}
		case reflect.Int64:
			wire = WireInt64
			fun = func(p unsafe.Pointer, b *Buffer) {
				b.AppendInt64(*(*int64)(p))
			}

		case reflect.Float32:
			wire = WireFloat32
			fun = func(p unsafe.Pointer, b *Buffer) {
				b.AppendFloat32(*(*float32)(p))
			}

		case reflect.Float64:
			wire = WireFloat64
			fun = func(p unsafe.Pointer, b *Buffer) {
				b.AppendFloat64(*(*float64)(p))
			}

		case reflect.Bool:
			wire = WireBool
			fun = func(p unsafe.Pointer, b *Buffer) {
				b.AppendBool(*(*bool)(p))
			}

		case reflect.Int:
			wire = WireInt
			fun = func(p unsafe.Pointer, b *Buffer) {
				b.AppendInt(*(*int)(p))
			}

		case reflect.String:
			wire = WireString
			fun = func(p unsafe.Pointer, b *Buffer) {
				b.AppendString(*(*string)(p))
			}

		case reflect.Map:

			mpEnc := newMapEncoderUsingTagWithSchemaAndOpts(reflect.New(f.Type).Elem().Interface(), usingTagName, &Buffer{}, opts)
			fun = func(p unsafe.Pointer, b *Buffer) {
				var em = p
				mpEnc.Marshal(em, b)
			}
			wire = WireMap
			enc = mpEnc

		case reflect.Slice:

			// create a slice encoder to handle the slice type then hand off to it in the fun
			slEnc := newSliceEncoderUsingTagWithSchemaAndOpts(reflect.New(f.Type).Elem().Interface(), usingTagName, &Buffer{}, opts)
			fun = func(p unsafe.Pointer, b *Buffer) {
				var em = p
				slEnc.Marshal(em, b)
			}
			wire = slEnc.wire
			enc = slEnc

		case reflect.Struct:

			// check first if we're a stringer field because we have a bespoke method for encoding stringers
			if opts.Contains("stringer") {
				if reflect.ValueOf(f.Type).MethodByName("String").Kind() == reflect.Invalid {
					panic("stringer option requires a String method on the struct") // we need a String method
				}

				wire = WireString
				t := f.Type
				fun = func(p unsafe.Pointer, b *Buffer) {
					e, ok := reflect.NewAt(t, p).Interface().(fmt.Stringer)
					if !ok {
						return
					}

					b.AppendString(e.String())
				}
				break
			}

			if opts.Contains("encoder") {
				wire = WireBytes
				t := f.Type

				if pointerWrap {
					t = t.Elem()
				}

				if !reflect.PointerTo(t).Implements(reflect.TypeOf((*binaryEncoder)(nil)).Elem()) {
					panic("usage of encoder option with an Encoder requires a MarshalBinary method on the target field type struct. see glint.binaryEncoder")
				}

				fun = func(p unsafe.Pointer, b *Buffer) {
					e, ok := reflect.NewAt(t, p).Interface().(binaryEncoder)
					if !ok {
						b.AppendBytes([]byte{})
						return
					}
					b.AppendBytes(e.MarshalBinary())
				}
				break
			}

			// check first if we're a time field because we have a bespoke method for encoding time
			if f.Type == timeType || (pointerWrap && f.Type.Elem() == timeType) {
				wire = WireTime
				fun = func(p unsafe.Pointer, b *Buffer) {
					b.AppendTime(*(*time.Time)(p))
				}
				break
			}

			// or a standard struct encoding
			wire = WireStruct

			// create a new encoder to handle the sub-type and hand off to it in the fun
			var inf any
			if pointerWrap {
				inf = reflect.New(f.Type.Elem()).Elem().Interface()
			} else {
				inf = reflect.New(f.Type).Elem().Interface()
			}

			se := newEncoderUsingTag(inf, usingTagName)
			enc = se

			fun = func(p unsafe.Pointer, b *Buffer) {
				var em any = p
				se.Marshal(em, b)
			}

		default:
			continue
		}

		if pointerWrap {
			// pointer fields require wrapping in a dereference function that handles nil values.
			// A leading byte indicates presence: 1 for value present, 0 for nil.
			wire |= WirePtrFlag
			fun = derefAppend(fun)
		}

		bytes = appendField(bytes, tag, wire)

		// sub-encoders append their own schema data when needed.
		// Examples: slice encoders for multi-dimensional arrays,
		// or struct encoders for embedded fields.
		if enc != nil {
			schema := enc.Schema()
			bytes = append(bytes, schema.Bytes...)
			enc.ClearSchema()
		}

		if opts.Contains("stringer") || opts.Contains("encoder") {
			wire = 0 // we don't want to use fast paths in marshal for stringer or encoder
		}

		var encd *encoderImpl

		switch ee := enc.(type) {
		case *encoderImpl:
			encd = ee
		}

		e.instructions = append(e.instructions, encodeInstruction{
			wire:   wire,
			offset: f.Offset,
			tag:    tag,
			fun:    fun,
			subenc: encd,
		})
	}

	e.schema.AppendBytes(bytes)
}

// Marshal executes the encoding instructions built during NewEncoder to write the struct data
// into the provided Buffer.
// Fields tagged as `glint:"name"` map to "name" in the schema, with types inferred from
// the Go struct field types.
// Requirements: v must be a pointer and its type must exactly match what NewEncoder received.
func (e *encoderImpl) Marshal(v any, b *Buffer) {

	p := (*iface)(unsafe.Pointer(&v)).Data

	if !b.TrustedSchema {
		b.Bytes = append(b.Bytes, e.schema.Bytes...)
	} else if len(b.Bytes) == 0 {
		// For recursive Marshal calls (nested structs), only the top level
		// gets the minimal header - not the nested ones.

		// Trusted schema mode needs only the hash for validation
		b.Bytes = append(b.Bytes, e.header.Bytes...)
	}

	for i := 0; i < len(e.instructions); i++ {
		switch e.instructions[i].wire {
		// inlinable fast paths - const cases required for jump table optimization
		case WireBool:
			b.AppendBool(*(*bool)(unsafe.Add(p, e.instructions[i].offset)))
		case WireInt:
			b.AppendInt(*(*int)(unsafe.Add(p, e.instructions[i].offset)))
		case WireInt8:
			b.AppendInt8(*(*int8)(unsafe.Add(p, e.instructions[i].offset)))
		case WireInt16:
			b.AppendInt16(*(*int16)(unsafe.Add(p, e.instructions[i].offset)))
		case WireInt32:
			b.AppendInt32(*(*int32)(unsafe.Add(p, e.instructions[i].offset)))
		case WireInt64:
			b.AppendInt64(*(*int64)(unsafe.Add(p, e.instructions[i].offset)))
		case WireUint:
			b.AppendUint(*(*uint)(unsafe.Add(p, e.instructions[i].offset)))
		case WireUint8:
			b.AppendUint8(*(*uint8)(unsafe.Add(p, e.instructions[i].offset)))
		case WireUint16:
			b.AppendUint16(*(*uint16)(unsafe.Add(p, e.instructions[i].offset)))
		case WireUint32:
			b.AppendUint32(*(*uint32)(unsafe.Add(p, e.instructions[i].offset)))
		case WireUint64:
			b.AppendUint64(*(*uint64)(unsafe.Add(p, e.instructions[i].offset)))
		case WireFloat32:
			b.AppendFloat32(*(*float32)(unsafe.Add(p, e.instructions[i].offset)))
		case WireFloat64:
			b.AppendFloat64(*(*float64)(unsafe.Add(p, e.instructions[i].offset)))
		case WireString:
			b.AppendString(*(*string)(unsafe.Add(p, e.instructions[i].offset)))
		case WireBytes:
			b.AppendBytes((*(*[]byte)(unsafe.Add(p, e.instructions[i].offset))))
		case WireTime:
			b.AppendTime(*(*time.Time)(unsafe.Add(p, e.instructions[i].offset)))
		case WireStruct:
			e.instructions[i].subenc.Marshal(unsafe.Add(p, e.instructions[i].offset), b)
		default:
			e.instructions[i].fun(unsafe.Add(p, e.instructions[i].offset), b)
		}
	}

}

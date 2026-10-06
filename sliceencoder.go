package glint

import (
	"fmt"
	"reflect"
	"time"
	"unsafe"
)

type SliceEncoder struct {
	instruction func(t unsafe.Pointer, w *Buffer) // the function we'll run to encode our date
	wire        WireType                          // the wire type of the data we're encoding
	offset      uintptr                           // used in the pointer arrhythmic to traverse through the arrays at runtime
	subenc      *encoderImpl                      // if we're reliant on an encoderImpl instance to encode our data
	schema      *Buffer                           // if our data requires us to define more schema data it will be written here
}

// Schema returns the schema without any header information
func (s *SliceEncoder) Schema() *Buffer {
	return s.schema
}

// ClearSchema removes all data from the schema of this instance
func (s *SliceEncoder) ClearSchema() {
	s.schema.Bytes = nil
}

// Marshal executes the instruction set built up by NewSliceEncoder
func (s *SliceEncoder) Marshal(v any, w *Buffer) {

	p := unsafe.Pointer(reflect.ValueOf(v).Pointer())
	s.instruction(p, w)
}

func NewSliceEncoder(t any) *SliceEncoder {
	return NewSliceEncoderUsingTagWithSchema(t, "glint", &Buffer{})
}

func NewSliceEncoderUsingTag(t any, usingTagName string) *SliceEncoder {
	return NewSliceEncoderUsingTagWithSchema(t, usingTagName, &Buffer{})
}

// NewSliceEncoderUsingTagWithSchema allows us to create an instruction which can iterate over a slice of different data types at runtime
func NewSliceEncoderUsingTagWithSchema(t any, usingTagName string, sc *Buffer) *SliceEncoder {
	return newSliceEncoderUsingTagWithSchemaAndOpts(t, usingTagName, sc, tagOptions(""))
}

// NewSliceEncoderUsingTagWithSchemaAndOpts allows us to create an instruction which can iterate over a slice of different data types at runtime
func newSliceEncoderUsingTagWithSchemaAndOpts(t any, usingTagName string, sc *Buffer, opts tagOptions) *SliceEncoder {

	s := &SliceEncoder{}
	s.schema = sc

	tt := reflect.TypeOf(t)
	s.offset = tt.Elem().Size()
	eoffset := s.offset

	// used for iterating through the slice elements
	/// yeah we could save a few lines of code here if we went and added the 'AppendString' bit
	/// below into a map based on the type, but then we wouldn't get any inlining from the compiler
	/// ... so we're doing it the hard way.

	pointerWrapped := false

	k := tt.Elem()

	// if the slice is a pointer we need to dereference it.
	if tt.Elem().Kind() == reflect.Pointer {
		pointerWrapped = true
		k = k.Elem()
	}

	switch k.Kind() {
	case reflect.String:
		s.wire = WireSliceFlag | WireString
		s.instruction = func(p unsafe.Pointer, b *Buffer) {
			sl := *(*sliceHeader)(p)
			b.AppendUint(uint(sl.Len))

			for i := uintptr(0); i < uintptr(sl.Len); i++ {
				b.AppendString(*(*string)(unsafe.Add(sl.Data, (i * eoffset))))
			}
		}

	case reflect.Int:

		// standard encoding.
		s.wire = WireSliceFlag | WireInt
		if opts.Contains("delta") {
			s.wire |= WireDeltaFlag
		}
		s.instruction = func(p unsafe.Pointer, b *Buffer) {
			sl := *(*sliceHeader)(p)
			b.AppendUint(uint(sl.Len))
			if sl.Len == 0 {
				return
			}

			if s.wire&WireDeltaFlag != 0 {
				// Delta encoding: first value + zigzag deltas
				prev := *(*int)(sl.Data)
				b.AppendInt(prev)
				for i := uintptr(1); i < uintptr(sl.Len); i++ {
					curr := *(*int)(unsafe.Add(sl.Data, (i * eoffset)))
					delta := int64(curr) - int64(prev)
					appendVarintZigzag(b, delta)
					prev = curr
				}
			} else {
				// Standard encoding
				for i := uintptr(0); i < uintptr(sl.Len); i++ {
					b.AppendInt(*(*int)(unsafe.Add(sl.Data, (i * eoffset))))
				}
			}
		}

	case reflect.Int8:
		s.wire = WireSliceFlag | WireInt8
		s.instruction = func(p unsafe.Pointer, b *Buffer) {
			sl := *(*sliceHeader)(p)
			b.AppendUint(uint(sl.Len))
			for i := uintptr(0); i < uintptr(sl.Len); i++ {
				b.AppendInt8(*(*int8)(unsafe.Add(sl.Data, (i * eoffset))))
			}
		}

	case reflect.Int16:
		s.wire = WireSliceFlag | WireInt16
		if opts.Contains("delta") {
			s.wire |= WireDeltaFlag
		}
		s.instruction = func(p unsafe.Pointer, b *Buffer) {
			sl := *(*sliceHeader)(p)
			b.AppendUint(uint(sl.Len))
			if sl.Len == 0 {
				return
			}

			if s.wire&WireDeltaFlag != 0 {
				// Delta encoding: first value + zigzag deltas
				prev := *(*int16)(sl.Data)
				b.AppendInt16(prev)
				for i := uintptr(1); i < uintptr(sl.Len); i++ {
					curr := *(*int16)(unsafe.Add(sl.Data, (i * eoffset)))
					delta := int64(curr) - int64(prev)
					appendVarintZigzag(b, delta)
					prev = curr
				}
			} else {
				// Standard encoding
				for i := uintptr(0); i < uintptr(sl.Len); i++ {
					b.AppendInt16(*(*int16)(unsafe.Add(sl.Data, (i * eoffset))))
				}
			}
		}

	case reflect.Int32:
		s.wire = WireSliceFlag | WireInt32
		if opts.Contains("delta") {
			s.wire |= WireDeltaFlag
		}
		s.instruction = func(p unsafe.Pointer, b *Buffer) {
			sl := *(*sliceHeader)(p)
			b.AppendUint(uint(sl.Len))
			if sl.Len == 0 {
				return
			}

			if s.wire&WireDeltaFlag != 0 {
				// Delta encoding: first value + zigzag deltas
				prev := *(*int32)(sl.Data)
				b.AppendInt32(prev)
				for i := uintptr(1); i < uintptr(sl.Len); i++ {
					curr := *(*int32)(unsafe.Add(sl.Data, (i * eoffset)))
					delta := int64(curr) - int64(prev)
					appendVarintZigzag(b, delta)
					prev = curr
				}
			} else {
				// Standard encoding
				for i := uintptr(0); i < uintptr(sl.Len); i++ {
					b.AppendInt32(*(*int32)(unsafe.Add(sl.Data, (i * eoffset))))
				}
			}
		}

	case reflect.Int64:
		s.wire = WireSliceFlag | WireInt64
		if opts.Contains("delta") {
			s.wire |= WireDeltaFlag
		}
		s.instruction = func(p unsafe.Pointer, b *Buffer) {
			sl := *(*sliceHeader)(p)
			b.AppendUint(uint(sl.Len))
			if sl.Len == 0 {
				return
			}

			if s.wire&WireDeltaFlag != 0 {
				// Delta encoding: first value + zigzag deltas
				prev := *(*int64)(sl.Data)
				b.AppendInt64(prev)
				for i := uintptr(1); i < uintptr(sl.Len); i++ {
					curr := *(*int64)(unsafe.Add(sl.Data, (i * eoffset)))
					delta := curr - prev
					appendVarint(b, uint64((delta>>63)^(delta<<1)))
					prev = curr
				}
			} else {
				// Standard encoding
				for i := uintptr(0); i < uintptr(sl.Len); i++ {
					b.AppendInt64(*(*int64)(unsafe.Add(sl.Data, (i * eoffset))))
				}
			}
		}

	case reflect.Uint:
		s.wire = WireSliceFlag | WireUint
		if opts.Contains("delta") {
			s.wire |= WireDeltaFlag
		}
		s.instruction = func(p unsafe.Pointer, b *Buffer) {
			sl := *(*sliceHeader)(p)
			b.AppendUint(uint(sl.Len))
			if sl.Len == 0 {
				return
			}

			if s.wire&WireDeltaFlag != 0 {
				// Delta encoding: first value + zigzag deltas
				prev := *(*uint)(sl.Data)
				b.AppendUint(prev)
				for i := uintptr(1); i < uintptr(sl.Len); i++ {
					curr := *(*uint)(unsafe.Add(sl.Data, (i * eoffset)))
					delta := int64(curr) - int64(prev)
					appendVarintZigzag(b, delta)
					prev = curr
				}
			} else {
				// Standard encoding
				for i := uintptr(0); i < uintptr(sl.Len); i++ {
					b.AppendUint(*(*uint)(unsafe.Add(sl.Data, (i * eoffset))))
				}
			}
		}

	case reflect.Uint8:
		s.wire = WireBytes
		s.instruction = func(p unsafe.Pointer, b *Buffer) {
			sl := *(*[]byte)(p)
			b.AppendBytes(sl)
		}

	case reflect.Uint16:
		s.wire = WireSliceFlag | WireUint16
		if opts.Contains("delta") {
			s.wire |= WireDeltaFlag
		}
		s.instruction = func(p unsafe.Pointer, b *Buffer) {
			sl := *(*sliceHeader)(p)
			b.AppendUint(uint(sl.Len))
			if sl.Len == 0 {
				return
			}

			if s.wire&WireDeltaFlag != 0 {
				// Delta encoding: first value + zigzag deltas
				prev := *(*uint16)(sl.Data)
				b.AppendUint16(prev)
				for i := uintptr(1); i < uintptr(sl.Len); i++ {
					curr := *(*uint16)(unsafe.Add(sl.Data, (i * eoffset)))
					delta := int64(curr) - int64(prev)
					appendVarintZigzag(b, delta)
					prev = curr
				}
			} else {
				// Standard encoding
				for i := uintptr(0); i < uintptr(sl.Len); i++ {
					b.AppendUint16(*(*uint16)(unsafe.Add(sl.Data, (i * eoffset))))
				}
			}
		}

	case reflect.Uint32:
		s.wire = WireSliceFlag | WireUint32
		if opts.Contains("delta") {
			s.wire |= WireDeltaFlag
		}
		s.instruction = func(p unsafe.Pointer, b *Buffer) {
			sl := *(*sliceHeader)(p)
			b.AppendUint(uint(sl.Len))
			if sl.Len == 0 {
				return
			}

			if s.wire&WireDeltaFlag != 0 {
				// Delta encoding: first value + zigzag deltas
				prev := *(*uint32)(sl.Data)
				b.AppendUint32(prev)
				for i := uintptr(1); i < uintptr(sl.Len); i++ {
					curr := *(*uint32)(unsafe.Add(sl.Data, (i * eoffset)))
					delta := int64(curr) - int64(prev)
					appendVarintZigzag(b, delta)
					prev = curr
				}
			} else {
				// Standard encoding
				for i := uintptr(0); i < uintptr(sl.Len); i++ {
					b.AppendUint32(*(*uint32)(unsafe.Add(sl.Data, (i * eoffset))))
				}
			}
		}

	case reflect.Uint64:
		s.wire = WireSliceFlag | WireUint64
		if opts.Contains("delta") {
			s.wire |= WireDeltaFlag
		}
		s.instruction = func(p unsafe.Pointer, b *Buffer) {
			sl := *(*sliceHeader)(p)
			b.AppendUint(uint(sl.Len))
			if sl.Len == 0 {
				return
			}

			if s.wire&WireDeltaFlag != 0 {
				// Delta encoding: first value + zigzag deltas
				prev := *(*uint64)(sl.Data)
				b.AppendUint64(prev)
				for i := uintptr(1); i < uintptr(sl.Len); i++ {
					curr := *(*uint64)(unsafe.Add(sl.Data, (i * eoffset)))
					delta := int64(curr) - int64(prev)
					appendVarintZigzag(b, delta)
					prev = curr
				}
			} else {
				// Standard encoding
				for i := uintptr(0); i < uintptr(sl.Len); i++ {
					b.AppendUint64(*(*uint64)(unsafe.Add(sl.Data, (i * eoffset))))
				}
			}
		}

	case reflect.Float32:
		s.wire = WireSliceFlag | WireFloat32
		s.instruction = func(p unsafe.Pointer, b *Buffer) {
			sl := *(*sliceHeader)(p)
			b.AppendUint(uint(sl.Len))
			for i := uintptr(0); i < uintptr(sl.Len); i++ {
				b.AppendFloat32(*(*float32)(unsafe.Add(sl.Data, (i * eoffset))))
			}
		}

	case reflect.Float64:
		s.wire = WireSliceFlag | WireFloat64
		s.instruction = func(p unsafe.Pointer, b *Buffer) {
			sl := *(*sliceHeader)(p)
			b.AppendUint(uint(sl.Len))
			for i := uintptr(0); i < uintptr(sl.Len); i++ {
				b.AppendFloat64(*(*float64)(unsafe.Add(sl.Data, (i * eoffset))))
			}
		}

	case reflect.Bool:
		s.wire = WireSliceFlag | WireBool
		s.instruction = func(p unsafe.Pointer, b *Buffer) {
			sl := *(*sliceHeader)(p)
			b.AppendUint(uint(sl.Len))
			for i := uintptr(0); i < uintptr(sl.Len); i++ {
				b.AppendBool(*(*bool)(unsafe.Add(sl.Data, (i * eoffset))))
			}
		}

	case reflect.Map:

		panic("not yet")

	case reflect.Slice:
		s.wire = WireSliceFlag // slice on its own denotes slice of slice

		var inf = reflect.New(tt.Elem()).Elem().Interface()
		enc := newSliceEncoderUsingTagWithSchemaAndOpts(inf, usingTagName, &Buffer{}, opts)
		s.instruction = func(p unsafe.Pointer, b *Buffer) {
			sl := *(*sliceHeader)(p)
			b.AppendUint(uint(sl.Len))
			for i := uintptr(0); i < uintptr(sl.Len); i++ {
				var em any = unsafe.Add(sl.Data, (i * eoffset))
				enc.Marshal(em, b)
			}
		}

		if enc.wire > 0 {
			s.schema.AppendUint(uint(enc.wire))
		}
		if len(enc.Schema().Bytes) > 0 {
			s.schema.Bytes = append(s.schema.Bytes, enc.Schema().Bytes...)
		}

	case reflect.Struct:

		if tt.Elem() == timeType {
			s.wire = WireSliceFlag | WireTime
			s.instruction = func(p unsafe.Pointer, b *Buffer) {
				sl := *(*sliceHeader)(p)
				b.AppendUint(uint(sl.Len))
				for i := uintptr(0); i < uintptr(sl.Len); i++ {
					b.AppendTime(*(*time.Time)(unsafe.Add(sl.Data, (i * eoffset))))
				}
			}
			break
		}

		s.wire = WireSliceFlag | WireStruct

		var inf = reflect.New(k).Elem().Interface()
		s.subenc = newEncoderUsingTag(inf, usingTagName)
		enc := s.subenc

		s.schema.Bytes = append(s.schema.Bytes, s.subenc.Schema().Bytes...)
		s.subenc.schema.Reset()

		if pointerWrapped {
			// if pointerWrapped, we need to dereference the pointer and add the pointer bytes
			s.instruction = func(p unsafe.Pointer, b *Buffer) {
				sl := *(*sliceHeader)(p)
				b.AppendUint(uint(sl.Len))

				for i := uintptr(0); i < uintptr(sl.Len); i++ {
					em := *(*unsafe.Pointer)(unsafe.Add(sl.Data, (i * eoffset)))
					if em == unsafe.Pointer(nil) {
						b.AppendUint8(0)
						continue
					}

					b.AppendUint8(1)
					enc.Marshal(em, b)
				}
			}

			break
		}

		s.instruction = func(p unsafe.Pointer, b *Buffer) {
			sl := *(*sliceHeader)(p)
			b.AppendUint(uint(sl.Len))

			for i := uintptr(0); i < uintptr(sl.Len); i++ {
				var em any = unsafe.Add(sl.Data, (i * eoffset))
				enc.Marshal(em, b)
			}
		}

	default:
		panic(fmt.Sprintf("unsupported type %v", tt))
	}

	return s
}

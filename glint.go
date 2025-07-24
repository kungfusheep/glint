// Package glint implements a hierarchical binary serialization format
package glint

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"
)

// iface represents the memory layout of interface{} to bypass reflect.ValueOf overhead
type iface struct {
	_, Data unsafe.Pointer
}

// sliceHeader replaces reflect.SliceHeader with inline pointer conversion
// for compatibility with vet and unsafe pointer rules
type sliceHeader struct {
	Data unsafe.Pointer
	Len  int
	Cap  int
}

type WireType uint

const (
	WireBool    WireType = 1
	WireInt     WireType = 2
	WireInt8    WireType = 3
	WireInt16   WireType = 4
	WireInt32   WireType = 5
	WireInt64   WireType = 6
	WireUint    WireType = 7
	WireUint8   WireType = 8
	WireUint16  WireType = 9
	WireUint32  WireType = 10
	WireUint64  WireType = 11
	WireFloat32 WireType = 12
	WireFloat64 WireType = 13
	WireString  WireType = 14
	WireBytes   WireType = 15
	WireStruct  WireType = 16
	WireMap     WireType = 17
	WireTime    WireType = 18
	// maximum value 31 (5-bit limit)
	WireTypeMask = 0b00011111

	WireSliceFlag WireType = 1 << 5 // marks slice fields
	WirePtrFlag   WireType = 1 << 6 // marks nullable fields
	WireDeltaFlag WireType = 1 << 7 // delta encoding for numeric slices

	wireSkip WireType = 1 << 8 // internal only
)

func (w WireType) String() string {

	switch w {
	case WireBool:
		return "WireBool"
	case WireInt:
		return "WireInt"
	case WireInt8:
		return "WireInt8"
	case WireInt16:
		return "WireInt16"
	case WireInt32:
		return "WireInt32"
	case WireInt64:
		return "WireInt64"
	case WireUint:
		return "WireUint"
	case WireUint8:
		return "WireUint8"
	case WireUint16:
		return "WireUint16"
	case WireUint32:
		return "WireUint32"
	case WireUint64:
		return "WireUint64"
	case WireFloat32:
		return "WireFloat32"
	case WireFloat64:
		return "WireFloat64"
	case WireString:
		return "WireString"
	case WireBytes:
		return "WireBytes"
	case WireStruct:
		return "WireStruct"
	case WireMap:
		return "WireMap"
	case WireTime:
		return "WireTime"

	default:

		var prefix string
		if w&wireSkip > 0 {
			prefix += "(skip)"
		}
		if w&WireSliceFlag > 0 {
			prefix += "[]"
		}
		if w&WirePtrFlag > 0 {
			prefix += "*"
		}
		if w&WireDeltaFlag > 0 {
			prefix += "(delta)"
		}
		if prefix != "" {
			return prefix + (w & WireTypeMask).String()
		}
		return "invalid WireType"
	}
}

// ReflectKindToWireType converts Go reflection types to glint wire types
func ReflectKindToWireType(k reflect.Type) WireType {
	switch k.Kind() {
	case reflect.Bool:
		return WireBool
	case reflect.Int:
		return WireInt
	case reflect.Int8:
		return WireInt8
	case reflect.Int16:
		return WireInt16
	case reflect.Int32:
		return WireInt32
	case reflect.Int64:
		return WireInt64
	case reflect.Uint:
		return WireUint
	case reflect.Uint8:
		return WireUint8
	case reflect.Uint16:
		return WireUint16
	case reflect.Uint32:
		return WireUint32
	case reflect.Uint64:
		return WireUint64
	case reflect.Float32:
		return WireFloat32
	case reflect.Float64:
		return WireFloat64
	case reflect.String:
		return WireString
	case reflect.Slice:
		if k.Elem().Kind() == reflect.Uint8 {
			return WireBytes
		}
		return WireSliceFlag | ReflectKindToWireType(k.Elem())
	case reflect.Pointer:
		return WirePtrFlag | ReflectKindToWireType(k.Elem())
	case reflect.Struct:
		if k == timeType {
			return WireTime
		}
		return WireStruct
	case reflect.Map:
		return WireMap
	}

	panic(fmt.Sprintf("unable to create a wire type for %v", k))
}

// WireTypeToReflectType converts glint wire types back to Go reflection types
func WireTypeToReflectType(k WireType) reflect.Type {

	switch {
	case k&WirePtrFlag > 0:
		return reflect.PointerTo(WireTypeToReflectType(k ^ WirePtrFlag))
	case k&WireSliceFlag > 0:
		return reflect.SliceOf(WireTypeToReflectType(k ^ WireSliceFlag))
	}

	switch k {
	case WireBool:
		return reflect.TypeOf(false)
	case WireInt:
		return reflect.TypeOf(int(0))
	case WireInt8:
		return reflect.TypeOf(int8(0))
	case WireInt16:
		return reflect.TypeOf(int16(0))
	case WireInt32:
		return reflect.TypeOf(int32(0))
	case WireInt64:
		return reflect.TypeOf(int64(0))
	case WireUint:
		return reflect.TypeOf(uint(0))
	case WireUint8:
		return reflect.TypeOf(uint8(0))
	case WireUint16:
		return reflect.TypeOf(uint16(0))
	case WireUint32:
		return reflect.TypeOf(uint32(0))
	case WireUint64:
		return reflect.TypeOf(uint64(0))
	case WireFloat32:
		return reflect.TypeOf(float32(0))
	case WireFloat64:
		return reflect.TypeOf(float64(0))
	case WireString:
		return reflect.TypeOf("")
	case WireStruct:
		return reflect.StructOf([]reflect.StructField{})

	case WireTime:
		return reflect.TypeOf(time.Time{})

	case WireMap:
		panic("use mapWireTypesToReflectKind")
	}

	panic(fmt.Sprintf("unable to create a reflect.Type for %v", k))
}

// mapWireTypesToReflectKind builds a map type from key and value wire types
func mapWireTypesToReflectKind(key, value WireType) reflect.Type {
	k := WireTypeToReflectType(key)
	v := WireTypeToReflectType(value)
	return reflect.MapOf(k, v)
}

func derefAppend(f func(unsafe.Pointer, *Buffer)) func(unsafe.Pointer, *Buffer) {
	// pointer fields require wrapping in a dereference function that handles nil values.
	// A leading byte indicates presence: 1 for value present, 0 for nil.
	return func(p unsafe.Pointer, b *Buffer) {
		p = *(*unsafe.Pointer)(p)
		if p == unsafe.Pointer(nil) {
			b.AppendUint8(0)
			return
		}
		b.AppendUint8(1)
		f(p, b)
	}
}

// decodeInstruction specifies how to decode and store a field value
type decodeInstruction struct {
	fun         func(unsafe.Pointer, Reader) Reader // fallback decoder when fast path unavailable
	offset      uintptr                             // field location in struct
	kind        WireType                            // wire type for fast path selection
	subdec      decoder                             // nested decoder for struct fields
	subType     reflect.Type                        // type information for nested decoder
	tag         string                              // field name from struct tag
	subinstr    []decodeInstruction                 // nested instructions for inlined decoding
	optimizable bool                                // true if this slice-of-structs can use fast path
}

// TrustHeader enables HTTP-based trusted schema mode.
// Implements the KVP interface for HTTP headers.
type TrustHeader struct {
	key   string
	value string
}

// Key provides the header name
func (t TrustHeader) Key() string {
	return t.key
}

// Value provides the header content
func (t TrustHeader) Value() string {
	return t.value
}

// NewTrustHeader creates an HTTP header for schema trust negotiation.
// Use with NewBufferWithTrust to skip schema transmission when both sides have matching schemas.
func NewTrustHeader(d *decoderImpl) TrustHeader {
	lastHash := atomic.LoadUint32(&d.lastHash)
	return TrustHeader{"X-Glint-Trust", strconv.FormatUint(uint64(lastHash), 10)}
}

// deref creates a wrapper that dereferences pointers and handles nil checks.
// Reads the presence byte (1=value, 0=nil) before processing.
func deref(wrap func(unsafe.Pointer, Reader) Reader, _ WireType, t reflect.Type) func(unsafe.Pointer, Reader) Reader {
	return func(p unsafe.Pointer, r Reader) Reader {

		if r.ReadByte() == 0 {
			return r
		}

		if *(*unsafe.Pointer)(p) == unsafe.Pointer(nil) {
			*(*unsafe.Pointer)(p) = reflect.New(t.Elem()).UnsafePointer()
		}

		return wrap(*(*unsafe.Pointer)(p), r)
	}
}

type assigner struct {
	subDecoder decoder                             // decoder for nested types
	fun        func(unsafe.Pointer, Reader) Reader // field assignment function
	rkind      reflect.Kind                        // final type after pointer unwrapping
	wire       WireType                            // glint type for this field
	pointer    bool                                // indicates pointer field
}

func reflectKindToAssigner(k reflect.Type, usingTagName string, opts tagOptions, limits DecodeLimits) assigner {

	pointerWrap := false
	var fun func(unsafe.Pointer, Reader) Reader
	var sub decoder
	var wire WireType

	kind := k.Kind()
	if kind == reflect.Pointer {
		pointerWrap = true
		kind = k.Elem().Kind()
	}

	switch kind {
	case reflect.Uint8:
		wire = WireUint8
		fun = func(p unsafe.Pointer, r Reader) Reader {
			*(*uint8)(p) = r.ReadUint8()
			return r
		}
	case reflect.Uint16:
		wire = WireUint16
		fun = func(p unsafe.Pointer, r Reader) Reader {
			*(*uint16)(p) = r.ReadUint16()
			return r
		}
	case reflect.Uint32:
		wire = WireUint32
		fun = func(p unsafe.Pointer, r Reader) Reader {
			*(*uint32)(p) = r.ReadUint32()
			return r
		}
	case reflect.Uint64:
		wire = WireUint64
		fun = func(p unsafe.Pointer, r Reader) Reader {
			*(*uint64)(p) = r.ReadUint64()
			return r
		}
	case reflect.Uint:
		wire = WireUint
		fun = func(p unsafe.Pointer, r Reader) Reader {
			*(*uint)(p) = r.ReadUint()
			return r
		}
	case reflect.Int8:

		wire = WireInt8
		fun = func(p unsafe.Pointer, r Reader) Reader {
			*(*int8)(p) = r.ReadInt8()
			return r
		}

	case reflect.Int16:
		wire = WireInt16
		fun = func(p unsafe.Pointer, r Reader) Reader {
			*(*int16)(p) = r.ReadInt16()
			return r
		}
	case reflect.Int32:
		wire = WireInt32
		fun = func(p unsafe.Pointer, r Reader) Reader {
			*(*int32)(p) = r.ReadInt32()
			return r
		}
	case reflect.Int64:
		wire = WireInt64
		fun = func(p unsafe.Pointer, r Reader) Reader {
			*(*int64)(p) = r.ReadInt64()
			return r
		}

	case reflect.Float32:
		wire = WireFloat32
		fun = func(p unsafe.Pointer, r Reader) Reader {
			*(*float32)(p) = r.ReadFloat32()
			return r
		}

	case reflect.Float64:
		wire = WireFloat64
		fun = func(p unsafe.Pointer, r Reader) Reader {
			*(*float64)(p) = r.ReadFloat64()
			return r
		}

	case reflect.Bool:
		wire = WireBool
		fun = func(p unsafe.Pointer, r Reader) Reader {
			*(*bool)(p) = r.ReadBool()
			return r
		}

	case reflect.Int:
		wire = WireInt
		fun = func(p unsafe.Pointer, r Reader) Reader {
			*(*int)(p) = r.ReadInt()
			return r
		}

	case reflect.String:
		wire = WireString
		fun = func(p unsafe.Pointer, r Reader) Reader {
			*(*string)(p) = r.ReadString()
			return r
		}

	case reflect.Map:

		// map fields require specialized decoder
		wire = WireMap
		mpDec := newMapDecoderUsingTagAndOptsWithLimits(reflect.New(k).Elem().Interface(), usingTagName, opts, limits)
		mpDec.subType = k
		fun = func(p unsafe.Pointer, r Reader) Reader {
			var em any = p
			return mpDec.unmarshal(r, nil, em)
		}

		sub = mpDec

	case reflect.Slice:

		// slice fields need custom handling
		slDec := newSliceDecoderUsingTagAndOptsWithLimits(reflect.New(k).Elem().Interface(), usingTagName, opts, limits)
		slDec.subType = k
		fun = func(p unsafe.Pointer, r Reader) Reader {
			var em any = p
			return slDec.unmarshal(r, nil, em)
		}

		sub = slDec
		wire = slDec.kind

	case reflect.Struct:

		if k == timeType || (k.Kind() == reflect.Pointer && k.Elem() == timeType) {
			wire = WireTime
			fun = func(p unsafe.Pointer, r Reader) Reader {
				*(*time.Time)(p) = r.ReadTime()
				return r
			}
			break
		}

		if opts.Contains("encoder") {

			wire = WireBytes

			ft := k
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}

			if !reflect.PointerTo(ft).Implements(reflect.TypeOf((*binaryDecoder)(nil)).Elem()) {
				panic("usage of encoder option with a Decoder requires an UnmarshalBinary method on the target field type struct. see glint.binaryDecoder")
			}

			fun = func(p unsafe.Pointer, r Reader) Reader {
				// consume bytes even if reflection fails
				bytes := r.ReadUint8Slice()

				e, ok := reflect.NewAt(ft, p).Interface().(binaryDecoder)
				if !ok {
					return r
				}

				e.UnmarshalBinary(bytes)

				return r
			}

			break
		}

		wire = WireStruct

		// nested structs use recursive decoder
		var inf = reflect.New(k).Elem().Interface() // create instance for schema
		dec := newDecoderUsingTag(inf, usingTagName)
		sub = dec // store for schema parsing phase

		fun = func(p unsafe.Pointer, r Reader) Reader {
			return dec.unmarshal(r, dec.instr, p)
		}
	}

	if fun == nil && sub == nil {
		panic(fmt.Sprintf("unsupported tye passed to reflectKindToAssigner: %v", k))
	}
	if wire == 0 {
		panic(fmt.Sprintf("unsupported wire passed to reflectKindToAssigner: %v", k))
	}

	if pointerWrap {
		wire |= WirePtrFlag
		fun = deref(fun, 0, k)
	}

	return assigner{fun: fun, wire: wire, pointer: pointerWrap, subDecoder: sub, rkind: kind}
}

type reflectAssigner struct {
	fun        func(Reader) (reflect.Value, Reader)
	subDecoder decoder
	assigner   assigner
}

func reflectKindToReflectValue(k reflect.Type, usingTagName string, opts tagOptions, limits DecodeLimits) reflectAssigner {

	assigner := reflectKindToAssigner(k, usingTagName, opts, limits)
	var fun func(Reader) (reflect.Value, Reader)

	switch k.Kind() {
	case reflect.Ptr:

		reflectKindToReflectValue(k.Elem(), usingTagName, opts, limits)
		a := reflectKindToReflectValue(k.Elem(), usingTagName, opts, limits)
		assigner = a.assigner

		fun = func(r Reader) (reflect.Value, Reader) {
			if r.ReadByte() == 0 { // read the nil check byte
				return reflect.ValueOf(nil), r
			}

			vv, re := a.fun(r)
			return vv.Addr(), re
		}

	case reflect.Uint8:
		fun = func(r Reader) (reflect.Value, Reader) {
			var val uint8
			r = assigner.fun(unsafe.Pointer(&val), r)
			return reflect.ValueOf(val), r
		}

	case reflect.Uint16:
		fun = func(r Reader) (reflect.Value, Reader) {
			var val uint16
			r = assigner.fun(unsafe.Pointer(&val), r)
			return reflect.ValueOf(val), r
		}

	case reflect.Uint32:
		fun = func(r Reader) (reflect.Value, Reader) {
			var val uint32
			r = assigner.fun(unsafe.Pointer(&val), r)
			return reflect.ValueOf(val), r
		}

	case reflect.Uint64:
		fun = func(r Reader) (reflect.Value, Reader) {
			var val uint64
			r = assigner.fun(unsafe.Pointer(&val), r)
			return reflect.ValueOf(val), r
		}

	case reflect.Uint:
		fun = func(r Reader) (reflect.Value, Reader) {
			var val uint
			r = assigner.fun(unsafe.Pointer(&val), r)
			return reflect.ValueOf(val), r
		}

	case reflect.Int8:
		fun = func(r Reader) (reflect.Value, Reader) {
			var val int8
			r = assigner.fun(unsafe.Pointer(&val), r)
			return reflect.ValueOf(val), r
		}

	case reflect.Int16:
		fun = func(r Reader) (reflect.Value, Reader) {
			var val int16
			r = assigner.fun(unsafe.Pointer(&val), r)
			return reflect.ValueOf(val), r
		}

	case reflect.Int32:
		fun = func(r Reader) (reflect.Value, Reader) {
			var val int32
			r = assigner.fun(unsafe.Pointer(&val), r)
			return reflect.ValueOf(val), r
		}

	case reflect.Int64:
		fun = func(r Reader) (reflect.Value, Reader) {
			var val int64
			r = assigner.fun(unsafe.Pointer(&val), r)
			return reflect.ValueOf(val), r
		}

	case reflect.Float32:
		fun = func(r Reader) (reflect.Value, Reader) {
			var val float32
			r = assigner.fun(unsafe.Pointer(&val), r)
			return reflect.ValueOf(val), r
		}

	case reflect.Float64:
		fun = func(r Reader) (reflect.Value, Reader) {
			var val float64
			r = assigner.fun(unsafe.Pointer(&val), r)
			return reflect.ValueOf(val), r
		}

	case reflect.Bool:
		fun = func(r Reader) (reflect.Value, Reader) {
			var val bool
			r = assigner.fun(unsafe.Pointer(&val), r)
			return reflect.ValueOf(val), r
		}

	case reflect.Int:
		fun = func(r Reader) (reflect.Value, Reader) {
			var val int
			r = assigner.fun(unsafe.Pointer(&val), r)
			return reflect.ValueOf(val), r
		}

	case reflect.String:
		fun = func(r Reader) (reflect.Value, Reader) {
			var val string
			r = assigner.fun(unsafe.Pointer(&val), r)
			return reflect.ValueOf(val), r
		}

	case reflect.Map:
		fun = func(r Reader) (reflect.Value, Reader) {
			m := reflect.MakeMap(k)
			p := m.Pointer()
			r = assigner.fun(unsafe.Pointer(&p), r)
			return m, r
		}

	case reflect.Slice:
		fun = func(r Reader) (reflect.Value, Reader) {

			b := r.position
			sl := int(r.ReadVarint())
			r.Unread(r.position - b)

			a := reflect.MakeSlice(k, sl, sl)

			p := a.Interface()
			pp := (*(*iface)(unsafe.Pointer(&p))).Data
			sh := *(*sliceHeader)(unsafe.Pointer(pp))

			r = assigner.fun(unsafe.Pointer(&sh), r)
			return a, r
		}

	case reflect.Struct:
		fun = func(r Reader) (reflect.Value, Reader) {
			a := reflect.New(k)
			r = assigner.fun(unsafe.Pointer(a.Pointer()), r)

			return reflect.Indirect(a), r
		}

	}

	if fun == nil {
		panic(fmt.Sprintf("reflectKindToReflectValue could not resolve type %v", k))
	}

	return reflectAssigner{
		fun:        fun,
		subDecoder: assigner.subDecoder,
		assigner:   assigner,
	}
}

// AppendDynamicValue encodes a value with type prefix into the buffer
func AppendDynamicValue(v any, b *Buffer) {
	switch val := v.(type) {
	case string:
		appendVarint(b, uint64(WireString))
		b.AppendString(val)
	case int:
		appendVarint(b, uint64(WireInt))
		b.AppendInt(val)
	case int8:
		appendVarint(b, uint64(WireInt8))
		b.AppendInt8(val)
	case int16:
		appendVarint(b, uint64(WireInt16))
		b.AppendInt16(val)
	case int32:
		appendVarint(b, uint64(WireInt32))
		b.AppendInt32(val)
	case int64:
		appendVarint(b, uint64(WireInt64))
		b.AppendInt64(val)
	case uint:
		appendVarint(b, uint64(WireUint))
		b.AppendUint(val)
	case uint8:
		appendVarint(b, uint64(WireUint8))
		b.AppendUint8(val)
	case uint16:
		appendVarint(b, uint64(WireUint16))
		b.AppendUint16(val)
	case uint32:
		appendVarint(b, uint64(WireUint32))
		b.AppendUint32(val)
	case uint64:
		appendVarint(b, uint64(WireUint64))
		b.AppendUint64(val)
	case float32:
		appendVarint(b, uint64(WireFloat32))
		b.AppendFloat32(val)
	case float64:
		appendVarint(b, uint64(WireFloat64))
		b.AppendFloat64(val)
	case bool:
		appendVarint(b, uint64(WireBool))
		b.AppendBool(val)
	case time.Time:
		appendVarint(b, uint64(WireTime))
		b.AppendTime(val)

	case *string:
		appendVarint(b, uint64(WireString|WirePtrFlag))
		if appendNil(val, b) {
			return
		}
		b.AppendString(*val)

	case *int:
		appendVarint(b, uint64(WireInt|WirePtrFlag))
		if appendNil(val, b) {
			return
		}
		b.AppendInt(*val)

	case *int8:
		appendVarint(b, uint64(WireInt8|WirePtrFlag))
		if appendNil(val, b) {
			return
		}
		b.AppendInt8(*val)

	case *int16:
		appendVarint(b, uint64(WireInt16|WirePtrFlag))
		if appendNil(val, b) {
			return
		}
		b.AppendInt16(*val)

	case *int32:
		appendVarint(b, uint64(WireInt32|WirePtrFlag))
		if appendNil(val, b) {
			return
		}
		b.AppendInt32(*val)

	case *int64:
		appendVarint(b, uint64(WireInt64|WirePtrFlag))
		if appendNil(val, b) {
			return
		}
		b.AppendInt64(*val)

	case *uint:
		appendVarint(b, uint64(WireUint|WirePtrFlag))
		if appendNil(val, b) {
			return
		}
		b.AppendUint(*val)

	case *uint8:
		appendVarint(b, uint64(WireUint8|WirePtrFlag))
		if appendNil(val, b) {
			return
		}
		b.AppendUint8(*val)

	case *uint16:
		appendVarint(b, uint64(WireUint16|WirePtrFlag))
		if appendNil(val, b) {
			return
		}
		b.AppendUint16(*val)

	case *uint32:
		appendVarint(b, uint64(WireUint32|WirePtrFlag))
		if appendNil(val, b) {
			return
		}
		b.AppendUint32(*val)

	case *uint64:
		appendVarint(b, uint64(WireUint64|WirePtrFlag))
		if appendNil(val, b) {
			return
		}
		b.AppendUint64(*val)

	case *float32:
		appendVarint(b, uint64(WireFloat32|WirePtrFlag))
		if appendNil(val, b) {
			return
		}
		b.AppendFloat32(*val)

	case *float64:
		appendVarint(b, uint64(WireFloat64|WirePtrFlag))
		if appendNil(val, b) {
			return
		}
		b.AppendFloat64(*val)

	case *bool:
		appendVarint(b, uint64(WireBool|WirePtrFlag))
		if appendNil(val, b) {
			return
		}
		b.AppendBool(*val)

	case *time.Time:
		appendVarint(b, uint64(WireTime|WirePtrFlag))
		if appendNil(val, b) {
			return
		}
		b.AppendTime(*val)

	case []string:
		appendVarint(b, uint64(WireString|WireSliceFlag))
		b.AppendUint(uint(len(val)))
		for i := 0; i < len(val); i++ {
			b.AppendString(val[i])
		}

	case []int:
		appendVarint(b, uint64(WireInt|WireSliceFlag))
		b.AppendUint(uint(len(val)))
		for i := 0; i < len(val); i++ {
			b.AppendInt(val[i])
		}

	case []int8:
		appendVarint(b, uint64(WireInt8|WireSliceFlag))
		b.AppendUint(uint(len(val)))
		for i := 0; i < len(val); i++ {
			b.AppendInt8(val[i])
		}

	case []int16:
		appendVarint(b, uint64(WireInt16|WireSliceFlag))
		b.AppendUint(uint(len(val)))
		for i := 0; i < len(val); i++ {
			b.AppendInt16(val[i])
		}

	case []int32:
		appendVarint(b, uint64(WireInt32|WireSliceFlag))
		b.AppendUint(uint(len(val)))
		for i := 0; i < len(val); i++ {
			b.AppendInt32(val[i])
		}

	case []int64:
		appendVarint(b, uint64(WireInt64|WireSliceFlag))
		b.AppendUint(uint(len(val)))
		for i := 0; i < len(val); i++ {
			b.AppendInt64(val[i])
		}

	case []uint:
		appendVarint(b, uint64(WireUint|WireSliceFlag))
		b.AppendUint(uint(len(val)))
		for i := 0; i < len(val); i++ {
			b.AppendUint(val[i])
		}

	case []uint8:
		appendVarint(b, uint64(WireUint8|WireSliceFlag))
		b.AppendUint(uint(len(val)))
		b.AppendBytes(val)

	case []uint16:
		appendVarint(b, uint64(WireUint16|WireSliceFlag))
		b.AppendUint(uint(len(val)))
		for i := 0; i < len(val); i++ {
			b.AppendUint16(val[i])
		}

	case []uint32:
		appendVarint(b, uint64(WireUint32|WireSliceFlag))
		b.AppendUint(uint(len(val)))
		for i := 0; i < len(val); i++ {
			b.AppendUint32(val[i])
		}

	case []uint64:
		appendVarint(b, uint64(WireUint64|WireSliceFlag))
		b.AppendUint(uint(len(val)))

		for i := 0; i < len(val); i++ {
			b.AppendUint64(val[i])
		}

	case []float32:
		appendVarint(b, uint64(WireFloat32|WireSliceFlag))
		b.AppendUint(uint(len(val)))
		for i := 0; i < len(val); i++ {
			b.AppendFloat32(val[i])
		}

	case []float64:
		appendVarint(b, uint64(WireFloat64|WireSliceFlag))
		b.AppendUint(uint(len(val)))
		for i := 0; i < len(val); i++ {
			b.AppendFloat64(val[i])
		}

	case []bool:
		appendVarint(b, uint64(WireBool|WireSliceFlag))
		b.AppendUint(uint(len(val)))
		for i := 0; i < len(val); i++ {
			b.AppendBool(val[i])
		}

	case []time.Time:
		appendVarint(b, uint64(WireTime|WireSliceFlag))
		b.AppendUint(uint(len(val)))
		for i := 0; i < len(val); i++ {
			b.AppendTime(val[i])
		}

	case [][]byte:
		appendVarint(b, uint64(WireBytes|WireSliceFlag))
		b.AppendUint(uint(len(val)))
		for i := 0; i < len(val); i++ {
			b.AppendBytes(val[i])
		}

	default:
		panic(fmt.Sprintf("unsupported type %T", val))
		// Handle other types as needed
	}
}

// appendNil writes a nil marker (0) or presence marker (1), returning true for nil
func appendNil(val any, b *Buffer) bool {
	if val == nil {
		b.Bytes = append(b.Bytes, 0)
		return true
	}
	b.Bytes = append(b.Bytes, 1)
	return false
}

// DynamicValue encodes any value to bytes with type information
func DynamicValue(value any) []byte {
	b := Buffer{}
	AppendDynamicValue(value, &b)
	return b.Bytes
}

// ReadDynamicValue decodes a value using the wire type from the first varint
func ReadDynamicValue(b []byte) any {
	r := NewReader(b)
	return ReadDynamicValueFromReader(&r)
}

// ReadDynamicValueFromReader extracts a typed value after reading the wire type indicator
func ReadDynamicValueFromReader(r *Reader) any {

	wire := WireType(r.ReadVarint())

	if wire&WirePtrFlag > 0 {
		p := r.ReadByte()
		if p == 0 {
			return nil
		}

		wire &= ^WirePtrFlag
	}

	if wire&WireSliceFlag > 0 {
		return ReadDynamicSlice(r, wire)
	}

	switch wire {
	case WireString:
		return r.ReadString()
	case WireInt:
		return r.ReadInt()
	case WireInt8:
		return r.ReadInt8()
	case WireInt16:
		return r.ReadInt16()
	case WireInt32:
		return r.ReadInt32()
	case WireInt64:
		return r.ReadInt64()
	case WireUint:
		return r.ReadUint()
	case WireUint8:
		return r.ReadUint8()
	case WireUint16:
		return r.ReadUint16()
	case WireUint32:
		return r.ReadUint32()
	case WireUint64:
		return r.ReadUint64()
	case WireFloat32:
		return r.ReadFloat32()
	case WireFloat64:
		return r.ReadFloat64()
	case WireBool:
		return r.ReadBool()
	case WireTime:
		return r.ReadTime()
	default:
		return nil
	}
}

// ReadDynamicSlice decodes a typed slice after examining the wire type
func ReadDynamicSlice(r *Reader, wire WireType) any {

	switch wire {
	case WireString | WireSliceFlag:
		return r.ReadStringSlice()

	case WireInt | WireSliceFlag:
		return r.ReadIntSlice()

	case WireInt8 | WireSliceFlag:
		return r.ReadInt8Slice()

	case WireInt16 | WireSliceFlag:
		return r.ReadInt16Slice()

	case WireInt32 | WireSliceFlag:
		return r.ReadInt32Slice()

	case WireInt64 | WireSliceFlag:
		return r.ReadInt64Slice()

	case WireUint | WireSliceFlag:
		return r.ReadUintSlice()

	case WireUint8 | WireSliceFlag:
		return r.ReadUint8Slice()

	case WireUint16 | WireSliceFlag:
		return r.ReadUint16Slice()

	case WireUint32 | WireSliceFlag:
		return r.ReadUint32Slice()

	case WireUint64 | WireSliceFlag:
		return r.ReadUint64Slice()

	case WireFloat32 | WireSliceFlag:
		return r.ReadFloat32Slice()

	case WireFloat64 | WireSliceFlag:
		return r.ReadFloat64Slice()

	case WireBool | WireSliceFlag:
		return r.ReadBoolSlice()

	case WireTime | WireSliceFlag:
		return r.ReadTimeSlice()

	case WireBytes | WireSliceFlag:
		return r.ReadBytesSlice()

	default:
		return nil
	}
}

// ReadDynamicString reads a string from a byte array based on the wire type provided by reading the first varint, it
// will return false if the wire type was not a string
func ReadDynamicString(b []byte) (string, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireString) {
		return "", false
	}

	return r.ReadString(), true
}

// ReadDynamicInt reads an int from a byte array based on the wire type provided by reading the first varint, it
// will return false if the wire type was not an int
func ReadDynamicInt(b []byte) (int, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireInt) {
		return 0, false
	}

	return r.ReadInt(), true
}

// ReadDynamicInt8 reads an int8 from a byte array based on the wire type provided by reading the first varint, it
// will return false if the wire type was not an int8
func ReadDynamicInt8(b []byte) (int8, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireInt8) {
		return 0, false
	}

	return r.ReadInt8(), true
}

// ReadDynamicInt16 reads an int16 from a byte array based on the wire type provided by reading the first varint, it
// will return false if the wire type was not an int16
func ReadDynamicInt16(b []byte) (int16, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireInt16) {
		return 0, false
	}

	return r.ReadInt16(), true
}

// ReadDynamicInt32 reads an int32 from a byte array based on the wire type provided by reading the first varint, it
// will return false if the wire type was not an int32
func ReadDynamicInt32(b []byte) (int32, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireInt32) {
		return 0, false
	}

	return r.ReadInt32(), true
}

// ReadDynamicInt64 reads an int64 from a byte array based on the wire type provided by reading the first varint, it
// will return false if the wire type was not an int64
func ReadDynamicInt64(b []byte) (int64, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireInt64) {
		return 0, false
	}

	return r.ReadInt64(), true
}

// ReadDynamicUint reads an uint from a byte array based on the wire type provided by reading the first varint, it
// will return false if the wire type was not an uint
func ReadDynamicUint(b []byte) (uint, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireUint) {
		return 0, false
	}

	return r.ReadUint(), true
}

// ReadDynamicUint8 reads an uint8 from a byte array based on the wire type provided by reading the first varint, it
// will return false if the wire type was not an uint8
func ReadDynamicUint8(b []byte) (uint8, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireUint8) {
		return 0, false
	}

	return r.ReadUint8(), true
}

// ReadDynamicUint16 reads an uint16 from a byte array based on the wire type provided by reading the first varint, it
// will return false if the wire type was not an uint16
func ReadDynamicUint16(b []byte) (uint16, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireUint16) {
		return 0, false
	}

	return r.ReadUint16(), true
}

// ReadDynamicUint32 reads an uint32 from a byte array based on the wire type provided by reading the first varint, it
// will return false if the wire type was not an uint32
func ReadDynamicUint32(b []byte) (uint32, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireUint32) {
		return 0, false
	}

	return r.ReadUint32(), true
}

// ReadDynamicUint64 reads an uint64 from a byte array based on the wire type provided by reading the first varint, it
// will return false if the wire type was not an uint64
func ReadDynamicUint64(b []byte) (uint64, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireUint64) {
		return 0, false
	}

	return r.ReadUint64(), true
}

// ReadDynamicFloat32 reads a float32 from a byte array based on the wire type provided by reading the first varint, it
// will return false if the wire type was not a float32
func ReadDynamicFloat32(b []byte) (float32, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireFloat32) {
		return 0, false
	}

	return r.ReadFloat32(), true
}

// ReadDynamicFloat64 reads a float64 from a byte array based on the wire type provided by reading the first varint, it
// will return false if the wire type was not a float64
func ReadDynamicFloat64(b []byte) (float64, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireFloat64) {
		return 0, false
	}

	return r.ReadFloat64(), true
}

// ReadDynamicBool reads a bool from a byte array based on the wire type provided by reading the first varint, it
// will return false if the wire type was not a bool
func ReadDynamicBool(b []byte) (bool, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireBool) {
		return false, false
	}

	return r.ReadBool(), true
}

// ReadDynamicTime reads a time.Time from a byte array based on the wire type provided by reading the first varint, it
// will return false if the wire type was not a time.Time
func ReadDynamicTime(b []byte) (time.Time, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireTime) {
		return time.Time{}, false
	}

	return r.ReadTime(), true
}

// ReadDynamicBytes reads a []byte from a byte array based on the wire type provided by reading the first varint, it
// will return false if the wire type was not a []byte
func ReadDynamicBytes(b []byte) ([]byte, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireBytes) {
		return nil, false
	}

	return r.Read(r.ReadVarint()), true
}

// ReadDynamicStringSlice reads a []string from a byte array based on the wire type provided by reading the first varint, it
// will return false if the wire type was not a []string
func ReadDynamicStringSlice(b []byte) ([]string, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireString|WireSliceFlag) {
		return nil, false
	}

	return r.ReadStringSlice(), true
}

// ReadDynamicIntSlice reads a []int from a byte array based on the wire type provided by reading the first varint, it
// will return false if the wire type was not a []int
func ReadDynamicIntSlice(b []byte) ([]int, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireInt|WireSliceFlag) {
		return nil, false
	}

	return r.ReadIntSlice(), true
}

// ReadDynamicInt8Slice reads a []int8 from a byte array based on the wire type provided by reading the first varint, it
// will return false if the wire type was not a []int8
func ReadDynamicInt8Slice(b []byte) ([]int8, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireInt8|WireSliceFlag) {
		return nil, false
	}

	return r.ReadInt8Slice(), true
}

// ReadDynamicInt16Slice reads a []int16 from a byte array based on the wire type provided by reading the first varint, it
// will return false if the wire type was not a []int16
func ReadDynamicInt16Slice(b []byte) ([]int16, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireInt16|WireSliceFlag) {
		return nil, false
	}

	return r.ReadInt16Slice(), true
}

// ReadDynamicInt32Slice reads a []int32 from a byte array based on the wire type provided by reading the first varint, it
// will return false if the wire type was not a []int32
func ReadDynamicInt32Slice(b []byte) ([]int32, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireInt32|WireSliceFlag) {
		return nil, false
	}

	return r.ReadInt32Slice(), true
}

// ReadDynamicInt64Slice reads a []int64 from a byte array based on the wire type provided by reading the first varint, it
// will return false if the wire type was not a []int64
func ReadDynamicInt64Slice(b []byte) ([]int64, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireInt64|WireSliceFlag) {
		return nil, false
	}

	return r.ReadInt64Slice(), true
}

// ReadDynamicUintSlice reads a []uint from a byte array based on the wire type provided by reading the first varint, it
// will return false if the wire type was not a []uint
func ReadDynamicUintSlice(b []byte) ([]uint, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireUint|WireSliceFlag) {
		return nil, false
	}

	return r.ReadUintSlice(), true
}

// ReadDynamicUint8Slice reads a []uint8 from a byte array based on the wire type provided by reading the first varint, it
// will return false if the wire type was not a []uint8
func ReadDynamicUint8Slice(b []byte) ([]uint8, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireUint8|WireSliceFlag) {
		return nil, false
	}

	return r.ReadUint8Slice(), true
}

// ReadDynamicUint16Slice reads a []uint16 from a byte array based on the wire type provided by reading the first varint, it
// will return false if the wire type was not a []uint16
func ReadDynamicUint16Slice(b []byte) ([]uint16, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireUint16|WireSliceFlag) {
		return nil, false
	}

	return r.ReadUint16Slice(), true
}

// ReadDynamicUint32Slice reads a []uint32 from a byte array based on the wire type provided by reading the first varint, it
// will return false if the wire type was not a []uint32
func ReadDynamicUint32Slice(b []byte) ([]uint32, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireUint32|WireSliceFlag) {
		return nil, false
	}

	return r.ReadUint32Slice(), true
}

// ReadDynamicUint64Slice attempts to decode a []uint64 after checking the wire type.
// Returns false if the data doesn't represent a uint64 slice.
func ReadDynamicUint64Slice(b []byte) ([]uint64, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireUint64|WireSliceFlag) {
		return nil, false
	}

	return r.ReadUint64Slice(), true
}

// ReadDynamicFloat32Slice attempts to decode a []float32 after checking the wire type.
// Returns false if the data doesn't represent a float32 slice.
func ReadDynamicFloat32Slice(b []byte) ([]float32, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireFloat32|WireSliceFlag) {
		return nil, false
	}

	return r.ReadFloat32Slice(), true
}

// ReadDynamicFloat64Slice attempts to decode a []float64 after checking the wire type.
// Returns false if the data doesn't represent a float64 slice.
func ReadDynamicFloat64Slice(b []byte) ([]float64, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireFloat64|WireSliceFlag) {
		return nil, false
	}

	return r.ReadFloat64Slice(), true
}

// ReadDynamicBoolSlice attempts to decode a []bool after checking the wire type.
// Returns false if the data doesn't represent a bool slice.
func ReadDynamicBoolSlice(b []byte) ([]bool, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireBool|WireSliceFlag) {
		return nil, false
	}

	return r.ReadBoolSlice(), true
}

// ReadDynamicTimeSlice attempts to decode a []time.Time after checking the wire type.
// Returns false if the data doesn't represent a time slice.
func ReadDynamicTimeSlice(b []byte) ([]time.Time, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireTime|WireSliceFlag) {
		return nil, false
	}

	return r.ReadTimeSlice(), true
}

// ReadDynamicBytesSlice attempts to decode a [][]byte after checking the wire type.
// Returns false if the data doesn't represent a bytes slice.
func ReadDynamicBytesSlice(b []byte) ([][]byte, bool) {
	r := NewReader(b)
	if r.ReadVarint() != uint(WireBytes|WireSliceFlag) {
		return nil, false
	}

	return r.ReadBytesSlice(), true
}

// appendField adds a field to the schema using TLV encoding.
// Complex types like structs and maps are handled elsewhere.
func appendField(bytes []byte, name string, wire WireType) []byte {
	bytes = appendVarintb(bytes, uint64(wire))       // type identifier
	bytes = append(bytes, byte(len(name)))           // field name size
	bytes = append(bytes, name[:byte(len(name))]...) // field name data
	return bytes
}

var timeType = reflect.TypeOf(time.Time{})

// DecodeLimits configures bounds checking during decoding to prevent memory exhaustion attacks
type DecodeLimits struct {
	MaxByteSliceLen uint // Maximum byte slice length (0 = unlimited)
	MaxSliceInitCap uint // Cap initial slice allocations to prevent huge upfront allocations
	MaxSchemaSize   uint // Maximum schema size in bytes
	MaxStringLen    uint // Maximum string length
}

// DefaultLimits provides sensible defaults for most use cases
var DefaultLimits = DecodeLimits{
	MaxByteSliceLen: 100 * 1024 * 1024, // 100MB
	MaxSliceInitCap: 10000,             // 10K elements initial cap
	MaxSchemaSize:   1024 * 1024,       // 1MB schema max
	MaxStringLen:    50 * 1024 * 1024,  // 50MB string max
}

// checkLimit validates a length against a limit, with 0 meaning unlimited
func checkLimit(length, limit uint, name string) {
	if limit > 0 && length > limit {
		panic(fmt.Sprintf("%s length %d exceeds limit %d", name, length, limit))
	}
}

// min returns the smaller of two uints
func min(a, b uint) uint {
	if a < b {
		return a
	}
	return b
}

// tagOptions represents the comma-separated options in a struct tag.
// Empty string if no options present.
//
// this is jacked from the stdlib to remain compatible with that syntax.
type tagOptions string

// parseTag extracts the name and options from a struct field tag.
// Returns name and comma-separated options.
func parseTag(tag string) (string, tagOptions) {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx], tagOptions(tag[idx+1:])
	}
	return tag, tagOptions("")
}

// Contains reports whether a comma-separated list of options
// contains a particular substr flag. substr must be surrounded by a
// string boundary or commas.
func (o tagOptions) Contains(optionName string) bool {
	if len(o) == 0 {
		return false
	}
	s := string(o)
	for s != "" {
		var next string
		i := strings.Index(s, ",")
		if i >= 0 {
			s, next = s[:i], s[i+1:]
		}
		if s == optionName {
			return true
		}
		s = next
	}
	return false
}

// SchemaBytes generates a binary schema from struct tags marked 'glint'
func SchemaBytes(t any) []byte {
	return SchemaBytesUsingTag(t, "glint")
}

// SchemaBytesUsingTag creates a schema using custom field tag names
func SchemaBytesUsingTag(t any, tagName string) []byte {
	enc := newEncoderUsingTag(t, tagName)
	return enc.schema.Bytes
}

// HashBytes extracts the 4-byte schema hash from a document header
func HashBytes(document []byte) []byte {
	return document[1:5]
}

// Flags extracts the first byte containing document flags
func Flags(document []byte) uint {
	return uint(document[0])
}

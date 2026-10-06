package glint

import (
	"fmt"
	"reflect"
	"time"
	"unsafe"
)

// sliceDecoder is a specialized decoder for parsing and decoding slice data from the binary format.
type sliceDecoder struct {
	instruction func(t unsafe.Pointer, r Reader) Reader
	subType     reflect.Type
	subdec      decoder
	kind        WireType     // this is the wire type we were created to parse
	wireType    WireType     // this is the wire type that was actually sent
	limits      DecodeLimits // bounds checking configuration
}

// setWireType allows this instance to have its wireType set, which is the type information pulled from the schema
func (s *sliceDecoder) setWireType(wt WireType) {
	s.wireType = wt
}

// Unmarshal
func (s *sliceDecoder) Unmarshal(bytes []byte, v any) error {
	panic("sliceDecoder Unmarshal")
}

// unmarshal executes the instruction we've created for the type we're to decode.
func (s *sliceDecoder) unmarshal(r Reader, instructions []decodeInstruction, v any) Reader {
	p := unsafe.Pointer(reflect.ValueOf(v).Pointer())
	return s.instruction(p, r)
}

// parseSchema reads the schema information sent and creates an instruction
func (s *sliceDecoder) parseSchema(r Reader, instructions []decodeInstruction) ([]decodeInstruction, Reader, error) {

	// there are slightly different ways we need to parse the schema depending on the type of data we've been sent.

	if s.subdec == nil && s.instruction == nil {

		// in this block we're only interested in creating instructions which simply read past fields due to
		// us not being interested in them. We'll end up here if something was sent in the schema we were not
		// expecting.

		switch {
		case s.wireType == WireSliceFlag:

			dec := sliceDecoder{wireType: WireType(r.ReadVarint())}
			var err error
			_, r, err = dec.parseSchema(r, nil)
			if err != nil {
				return nil, r, err
			}
			slic := []struct{}{}

			s.instruction = func(p unsafe.Pointer, r Reader) Reader {
				for i, sl := uint(0), r.ReadVarint(); i < sl; i++ {
					r = dec.unmarshal(r, nil, slic)
				}
				return r
			}

			if d, ok := dec.subdec.(*sliceDecoder); d == nil && ok {
				r.ReadVarint()
			}

		case s.wireType&WireTypeMask == WireStruct:

			sl := r.ReadVarint()
			sb := r.Read(sl)

			dec := newDecoder(struct{}{})

			var err error
			instructions, _, err = dec.parseSchema(NewReader(sb), nil) // parse the schema using the sub-decoder we set up in the decodeInstruction
			if err != nil {
				return nil, r, err
			}
			s.instruction = func(p unsafe.Pointer, r Reader) Reader {

				sl := int(r.ReadVarint()) // array length

				for i := uintptr(0); i < uintptr(sl); i++ {
					r = dec.unmarshal(r, instructions, nil)
				}

				return r
			}

		default:
			// skip past slices of basic types

			switch s.wireType & WireTypeMask {

			case WireInt, WireInt16, WireInt32, WireInt64,
				WireUint, WireUint16, WireUint32, WireUint64,
				WireFloat32, WireFloat64:

				s.instruction = func(t unsafe.Pointer, r Reader) Reader {
					for i, l := uint(0), r.ReadVarint(); i < l; i++ {
						r.SkipVarint()
					}
					return r
				}

			case WireString, WireTime:
				s.instruction = func(t unsafe.Pointer, r Reader) Reader {
					for i, l := uint(0), r.ReadVarint(); i < l; i++ {
						r.Skip(r.ReadVarint())
					}
					return r
				}

			case WireBool, WireBytes, WireInt8, WireUint8:
				s.instruction = func(t unsafe.Pointer, r Reader) Reader {
					r.Skip(r.ReadVarint())
					return r
				}
			}

		}

	} else if s.wireType != s.kind {
		// if the wire type we were sent does not match the kind we were created for we fail here.
		return nil, r, fmt.Errorf("slice wire type mismatch: %v != %v", s.wireType, s.kind)
	}

	switch d := s.subdec.(type) {
	case *sliceDecoder:
		if s.wireType != s.kind {
			break
		}

		d.setWireType(WireType(r.ReadVarint()))
		var err error
		_, r, err = d.parseSchema(r, instructions) // we need to pass the schema down and back up here to read the type byte
		if err != nil {
			return nil, r, err
		}

		if dd, ok := d.subdec.(*sliceDecoder); dd == nil && ok {
			r.ReadVarint()
		}

	case *mapDecoder:
		if d.wireType != s.kind {
			break
		}

		d.setWireType(WireType(r.ReadVarint()))
		var err error
		_, r, err = d.parseSchema(r, instructions) // we need to pass the schema down and back up here to read the type byte
		if err != nil {
			return nil, r, err
		}

	case *decoderImpl:
		if WireType(s.wireType) != WireSliceFlag|WireStruct {
			break
		}

		sl := r.ReadVarint()
		sb := r.Read(sl)

		var err error
		instructions, _, err = d.parseSchema(NewReader(sb), nil) // parse the schema using the sub-decoder we set up in the decodeInstruction
		if err != nil {
			return nil, r, err
		}

		// Check if this slice-of-structs can be optimized (small structs with only basic types)
		canOptimize := len(instructions) <= 4 && len(instructions) > 0
		if canOptimize {
			for _, instr := range instructions {
				if instr.kind != WireString && instr.kind != WireInt && instr.kind != WireInt32 &&
					instr.kind != WireBool && instr.kind != WireTime && instr.kind != WireFloat64 {
					canOptimize = false
					break
				}
			}
		}

		s.instruction = func(p unsafe.Pointer, r Reader) Reader {

			sl := int(r.ReadVarint()) // array length

			if sl == 0 {
				sli := reflect.MakeSlice(s.subType, 0, 1)
				*(*sliceHeader)(unsafe.Pointer(uintptr(p))) = sliceHeader{
					Data: unsafe.Pointer(sli.Pointer()),
					Len:  0,
					Cap:  1,
				}
			}

			if sheader := (*sliceHeader)(unsafe.Pointer(uintptr(p))); sheader == nil || sheader.Cap < sl {
				sli := reflect.MakeSlice(s.subType, sl, sl)
				*(*sliceHeader)(unsafe.Pointer(uintptr(p))) = sliceHeader{
					Data: unsafe.Pointer(sli.Pointer()),
					Len:  sl,
					Cap:  sl,
				}
			} else {
				sheader.Len = sl // we're reusing the slice, so we need to reset the length
			}

			sp := reflect.NewAt(s.subType, unsafe.Pointer(uintptr(p)))
			elem := (*(*sliceHeader)(unsafe.Pointer(sp.Pointer()))).Data
			size := s.subType.Elem().Size()

			if canOptimize {
				// Fast path: process basic-type structs directly without function pointer overhead
				for i := uintptr(0); i < uintptr(sl); i++ {
					structPtr := unsafe.Add(elem, i*size)

					// Process each field directly using the same logic as the decoder fast path
					for j := 0; j < len(instructions); j++ {
						switch instructions[j].kind {
						case WireString:
							l := r.ReadVarint()
							if l > r.BytesLeft() {
								panic(fmt.Sprintf("string length %d exceeds remaining bytes %d", l, r.BytesLeft()))
							}
							b := r.Read(l)
							*(*string)(unsafe.Add(structPtr, instructions[j].offset)) = *(*string)(unsafe.Pointer(&b))
						case WireInt:
							*(*int)(unsafe.Add(structPtr, instructions[j].offset)) = r.ReadInt()
						case WireInt32:
							*(*int32)(unsafe.Add(structPtr, instructions[j].offset)) = r.ReadInt32()
						case WireBool:
							*(*bool)(unsafe.Add(structPtr, instructions[j].offset)) = r.ReadBool()
						case WireTime:
							*(*time.Time)(unsafe.Add(structPtr, instructions[j].offset)) = r.ReadTime()
						case WireFloat64:
							*(*float64)(unsafe.Add(structPtr, instructions[j].offset)) = r.ReadFloat64()
						}
					}
				}
			} else {
				// General case with potential function pointers
				for i := uintptr(0); i < uintptr(sl); i++ {
					var em any = unsafe.Add(elem, (i * size))
					r = d.unmarshal(r, instructions, em)
				}
			}

			return r
		}
	}

	return instructions, r, nil
}

func newSliceDecoderUsingTagAndOpts(t any, usingTagName string, opts tagOptions) *sliceDecoder {
	return newSliceDecoderUsingTagAndOptsWithLimits(t, usingTagName, opts, DefaultLimits)
}

func newSliceDecoderUsingTagAndOptsWithLimits(t any, usingTagName string, opts tagOptions, limits DecodeLimits) *sliceDecoder {

	s := &sliceDecoder{}
	s.limits = limits

	tt := reflect.TypeOf(t)

	switch tt.Elem().Kind() {
	case reflect.String:
		s.kind = WireSliceFlag | WireString
		s.instruction = func(p unsafe.Pointer, r Reader) Reader {

			sl := r.ReadVarint() // array length

			var slice []string
			// Cap initial allocation to prevent memory bombs
			initialCap := min(sl, s.limits.MaxSliceInitCap)
			if cap(*(*[]string)(unsafe.Pointer(uintptr(p)))) < int(initialCap) {
				slice = make([]string, 0, initialCap)
			} else if sl == 0 {
				slice = make([]string, 0, 1)
			} else {
				slice = (*(*[]string)(unsafe.Pointer(uintptr(p))))[:0]
			}
			for i := uint(0); i < sl; i++ {
				l := r.ReadVarint() // r.ReadString() can't be inlined

				// Bounds checking for individual strings
				checkLimit(l, s.limits.MaxStringLen, "string")
				if l > r.BytesLeft() {
					panic(fmt.Sprintf("string length %d exceeds remaining bytes %d", l, r.BytesLeft()))
				}

				v := r.Read(l)

				slice = append(slice, *(*string)(unsafe.Pointer(&v)))
			}
			*(*[]string)(unsafe.Pointer(uintptr(p))) = slice

			return r
		}

	case reflect.Int:

		s.kind = WireSliceFlag | WireInt
		if opts.Contains("delta") {
			s.kind |= WireDeltaFlag
		}
		s.instruction = func(p unsafe.Pointer, r Reader) Reader {

			sl := r.ReadVarint() // array length
			var slice []int
			if cap(*(*[]int)(unsafe.Pointer(uintptr(p)))) < int(sl) {
				slice = make([]int, 0, sl)
			} else if sl == 0 {
				slice = make([]int, 0, 1)
			} else {
				slice = (*(*[]int)(unsafe.Pointer(uintptr(p))))[:0]
			}
			
			if sl > 0 {
				if s.wireType&WireDeltaFlag != 0 {
					// Delta decoding: first value + zigzag deltas
					prev := r.ReadInt()
					slice = append(slice, prev)
					for i := uint(1); i < sl; i++ {
						delta := int64(r.ReadZigzagVarint())
						prev = int(int64(prev) + delta)
						slice = append(slice, prev)
					}
				} else {
					// Standard decoding
					for i := uint(0); i < sl; i++ {
						slice = append(slice, r.ReadInt())
					}
				}
			}
			*(*[]int)(unsafe.Pointer(uintptr(p))) = slice

			return r
		}

	case reflect.Int8:
		s.kind = WireSliceFlag | WireInt8
		s.instruction = func(p unsafe.Pointer, r Reader) Reader {

			sl := r.ReadVarint() // array length
			var slice []int8
			if cap(*(*[]int8)(unsafe.Pointer(uintptr(p)))) < int(sl) {
				slice = make([]int8, 0, sl)
			} else if sl == 0 {
				slice = make([]int8, 0, 1)
			} else {
				slice = (*(*[]int8)(unsafe.Pointer(uintptr(p))))[:0]
			}
			for i := uint(0); i < sl; i++ {
				slice = append(slice, r.ReadInt8())
			}
			*(*[]int8)(unsafe.Pointer(uintptr(p))) = slice

			return r
		}

	case reflect.Int16:
		s.kind = WireSliceFlag | WireInt16
		if opts.Contains("delta") {
			s.kind |= WireDeltaFlag
		}
		s.instruction = func(p unsafe.Pointer, r Reader) Reader {

			sl := r.ReadVarint() // array length
			var slice []int16
			if cap(*(*[]int16)(unsafe.Pointer(uintptr(p)))) < int(sl) {
				slice = make([]int16, 0, sl)
			} else if sl == 0 {
				slice = make([]int16, 0, 1)
			} else {
				slice = (*(*[]int16)(unsafe.Pointer(uintptr(p))))[:0]
			}
			
			if sl > 0 {
				if s.wireType&WireDeltaFlag != 0 {
					// Delta decoding: first value + zigzag deltas
					prev := r.ReadInt16()
					slice = append(slice, prev)
					for i := uint(1); i < sl; i++ {
						delta := int64(r.ReadZigzagVarint())
						prev = int16(int64(prev) + delta)
						slice = append(slice, prev)
					}
				} else {
					// Standard decoding
					for i := uint(0); i < sl; i++ {
						slice = append(slice, r.ReadInt16())
					}
				}
			}
			*(*[]int16)(unsafe.Pointer(uintptr(p))) = slice

			return r
		}

	case reflect.Int32:
		s.kind = WireSliceFlag | WireInt32
		if opts.Contains("delta") {
			s.kind |= WireDeltaFlag
		}
		s.instruction = func(p unsafe.Pointer, r Reader) Reader {

			sl := r.ReadVarint() // array length
			var slice []int32
			if cap(*(*[]int32)(unsafe.Pointer(uintptr(p)))) < int(sl) {
				slice = make([]int32, 0, sl)
			} else if sl == 0 {
				slice = make([]int32, 0, 1)
			} else {
				slice = (*(*[]int32)(unsafe.Pointer(uintptr(p))))[:0]
			}
			
			if sl > 0 {
				if s.wireType&WireDeltaFlag != 0 {
					// Delta decoding: first value + zigzag deltas
					prev := r.ReadInt32()
					slice = append(slice, prev)
					for i := uint(1); i < sl; i++ {
						delta := int64(r.ReadZigzagVarint())
						prev = int32(int64(prev) + delta)
						slice = append(slice, prev)
					}
				} else {
					// Standard decoding
					for i := uint(0); i < sl; i++ {
						slice = append(slice, r.ReadInt32())
					}
				}
			}
			*(*[]int32)(unsafe.Pointer(uintptr(p))) = slice

			return r
		}

	case reflect.Int64:
		s.kind = WireSliceFlag | WireInt64
		if opts.Contains("delta") {
			s.kind |= WireDeltaFlag
		}
		s.instruction = func(p unsafe.Pointer, r Reader) Reader {

			sl := r.ReadVarint() // array length
			var slice []int64
			if cap(*(*[]int64)(unsafe.Pointer(uintptr(p)))) < int(sl) {
				slice = make([]int64, 0, sl)
			} else if sl == 0 {
				slice = make([]int64, 0, 1)
			} else {
				slice = (*(*[]int64)(unsafe.Pointer(uintptr(p))))[:0]
			}
			
			// Check if delta encoding is used
			if s.wireType&WireDeltaFlag != 0 {
				if sl > 0 {
					// First value is stored as-is
					prev := r.ReadInt64()
					slice = append(slice, prev)
					// Subsequent values are zigzag-encoded deltas
					for i := uint(1); i < sl; i++ {
						// Decode zigzag delta
						zigzag := r.ReadVarint()
						delta := int64((zigzag >> 1) ^ -(zigzag & 1))
						prev += delta
						slice = append(slice, prev)
					}
				}
			} else {
				// Standard encoding
				for i := uint(0); i < sl; i++ {
					slice = append(slice, r.ReadInt64())
				}
			}
			*(*[]int64)(unsafe.Pointer(uintptr(p))) = slice

			return r
		}

	case reflect.Uint:
		s.kind = WireSliceFlag | WireUint
		if opts.Contains("delta") {
			s.kind |= WireDeltaFlag
		}
		s.instruction = func(p unsafe.Pointer, r Reader) Reader {

			sl := r.ReadVarint() // array length
			var slice []uint
			if cap(*(*[]uint)(unsafe.Pointer(uintptr(p)))) < int(sl) {
				slice = make([]uint, 0, sl)
			} else if sl == 0 {
				slice = make([]uint, 0, 1)
			} else {
				slice = (*(*[]uint)(unsafe.Pointer(uintptr(p))))[:0]
			}
			
			if sl > 0 {
				if s.wireType&WireDeltaFlag != 0 {
					// Delta decoding: first value + zigzag deltas
					prev := r.ReadUint()
					slice = append(slice, prev)
					for i := uint(1); i < sl; i++ {
						delta := int64(r.ReadZigzagVarint())
						prev = uint(int64(prev) + delta)
						slice = append(slice, prev)
					}
				} else {
					// Standard decoding
					for i := uint(0); i < sl; i++ {
						slice = append(slice, r.ReadUint())
					}
				}
			}
			*(*[]uint)(unsafe.Pointer(uintptr(p))) = slice

			return r
		}

	case reflect.Uint8:
		s.kind = WireBytes
		s.instruction = func(p unsafe.Pointer, r Reader) Reader {
			sl := r.ReadVarint()

			// Bounds checking for byte slice length
			checkLimit(sl, s.limits.MaxByteSliceLen, "byte slice")
			if sl > r.BytesLeft() {
				panic(fmt.Sprintf("byte slice length %d exceeds remaining bytes %d", sl, r.BytesLeft()))
			}

			if sl == 0 {
				*(*[]byte)(unsafe.Pointer(uintptr(p))) = make([]byte, 0, 1)
				return r
			}
			*(*[]byte)(unsafe.Pointer(uintptr(p))) = r.Read(sl)
			return r
		}

	case reflect.Uint16:
		s.kind = WireSliceFlag | WireUint16
		if opts.Contains("delta") {
			s.kind |= WireDeltaFlag
		}
		s.instruction = func(p unsafe.Pointer, r Reader) Reader {

			sl := r.ReadVarint() // array length
			var slice []uint16
			if cap(*(*[]uint16)(unsafe.Pointer(uintptr(p)))) < int(sl) {
				slice = make([]uint16, 0, sl)
			} else if sl == 0 {
				slice = make([]uint16, 0, 1)
			} else {
				slice = (*(*[]uint16)(unsafe.Pointer(uintptr(p))))[:0]
			}
			
			if sl > 0 {
				if s.wireType&WireDeltaFlag != 0 {
					// Delta decoding: first value + zigzag deltas
					prev := r.ReadUint16()
					slice = append(slice, prev)
					for i := uint(1); i < sl; i++ {
						delta := int64(r.ReadZigzagVarint())
						prev = uint16(int64(prev) + delta)
						slice = append(slice, prev)
					}
				} else {
					// Standard decoding
					for i := uint(0); i < sl; i++ {
						slice = append(slice, uint16(r.ReadVarint()))
					}
				}
			}
			*(*[]uint16)(unsafe.Pointer(uintptr(p))) = slice

			return r
		}

	case reflect.Uint32:
		s.kind = WireSliceFlag | WireUint32
		if opts.Contains("delta") {
			s.kind |= WireDeltaFlag
		}
		s.instruction = func(p unsafe.Pointer, r Reader) Reader {

			sl := r.ReadVarint() // array length
			var slice []uint32
			if cap(*(*[]uint32)(unsafe.Pointer(uintptr(p)))) < int(sl) {
				slice = make([]uint32, 0, sl)
			} else if sl == 0 {
				slice = make([]uint32, 0, 1)
			} else {
				slice = (*(*[]uint32)(unsafe.Pointer(uintptr(p))))[:0]
			}
			
			if sl > 0 {
				if s.wireType&WireDeltaFlag != 0 {
					// Delta decoding: first value + zigzag deltas
					prev := r.ReadUint32()
					slice = append(slice, prev)
					for i := uint(1); i < sl; i++ {
						delta := int64(r.ReadZigzagVarint())
						prev = uint32(int64(prev) + delta)
						slice = append(slice, prev)
					}
				} else {
					// Standard decoding
					for i := uint(0); i < sl; i++ {
						slice = append(slice, r.ReadUint32())
					}
				}
			}
			*(*[]uint32)(unsafe.Pointer(uintptr(p))) = slice

			return r
		}

	case reflect.Uint64:
		s.kind = WireSliceFlag | WireUint64
		if opts.Contains("delta") {
			s.kind |= WireDeltaFlag
		}
		s.instruction = func(p unsafe.Pointer, r Reader) Reader {

			sl := r.ReadVarint() // array length
			var slice []uint64
			if cap(*(*[]uint64)(unsafe.Pointer(uintptr(p)))) < int(sl) {
				slice = make([]uint64, 0, sl)
			} else if sl == 0 {
				slice = make([]uint64, 0, 1)
			} else {
				slice = (*(*[]uint64)(unsafe.Pointer(uintptr(p))))[:0]
			}
			
			if sl > 0 {
				if s.wireType&WireDeltaFlag != 0 {
					// Delta decoding: first value + zigzag deltas
					prev := r.ReadUint64()
					slice = append(slice, prev)
					for i := uint(1); i < sl; i++ {
						delta := int64(r.ReadZigzagVarint())
						prev = uint64(int64(prev) + delta)
						slice = append(slice, prev)
					}
				} else {
					// Standard decoding
					for i := uint(0); i < sl; i++ {
						slice = append(slice, r.ReadUint64())
					}
				}
			}
			*(*[]uint64)(unsafe.Pointer(uintptr(p))) = slice

			return r
		}

	case reflect.Float32:
		s.kind = WireSliceFlag | WireFloat32
		s.instruction = func(p unsafe.Pointer, r Reader) Reader {

			sl := r.ReadVarint() // array length
			var slice []float32
			if cap(*(*[]float32)(unsafe.Pointer(uintptr(p)))) < int(sl) {
				slice = make([]float32, 0, sl)
			} else if sl == 0 {
				slice = make([]float32, 0, 1)
			} else {
				slice = (*(*[]float32)(unsafe.Pointer(uintptr(p))))[:0]
			}
			for i := uint(0); i < sl; i++ {
				slice = append(slice, r.ReadFloat32())
			}
			*(*[]float32)(unsafe.Pointer(uintptr(p))) = slice

			return r
		}

	case reflect.Float64:
		s.kind = WireSliceFlag | WireFloat64
		s.instruction = func(p unsafe.Pointer, r Reader) Reader {

			sl := r.ReadVarint() // array length
			var slice []float64
			if cap(*(*[]float64)(unsafe.Pointer(uintptr(p)))) < int(sl) {
				slice = make([]float64, 0, sl)
			} else if sl == 0 {
				slice = make([]float64, 0, 1)
			} else {
				slice = (*(*[]float64)(unsafe.Pointer(uintptr(p))))[:0]
			}
			for i := uint(0); i < sl; i++ {
				slice = append(slice, r.ReadFloat64())
			}
			*(*[]float64)(unsafe.Pointer(uintptr(p))) = slice

			return r
		}

	case reflect.Bool:
		s.kind = WireSliceFlag | WireBool
		s.instruction = func(p unsafe.Pointer, r Reader) Reader {

			sl := r.ReadVarint() // array length

			var slice []bool
			if cap(*(*[]bool)(unsafe.Pointer(uintptr(p)))) < int(sl) {
				slice = make([]bool, 0, sl)
			} else if sl == 0 {
				slice = make([]bool, 0, 1)
			} else {
				slice = (*(*[]bool)(unsafe.Pointer(uintptr(p))))[:0]
			}

			for i := uint(0); i < sl; i++ {
				slice = append(slice, r.ReadBool())
			}
			*(*[]bool)(unsafe.Pointer(uintptr(p))) = slice

			return r
		}

	case reflect.Struct:

		if tt.Elem() == timeType {
			s.kind = WireSliceFlag | WireTime
			s.instruction = func(p unsafe.Pointer, r Reader) Reader {

				sl := r.ReadVarint() // array length
				var slice []time.Time
				if cap(*(*[]time.Time)(unsafe.Pointer(uintptr(p)))) < int(sl) {
					slice = make([]time.Time, 0, sl)
				} else if sl == 0 {
					slice = make([]time.Time, 0, 1)
				} else {
					slice = (*(*[]time.Time)(unsafe.Pointer(uintptr(p))))[:0]
				}

				for i := uint(0); i < sl; i++ {
					slice = append(slice, time.Time(r.ReadTime()))
				}
				*(*[]time.Time)(unsafe.Pointer(uintptr(p))) = slice

				return r
			}
			break
		}

		s.kind = WireSliceFlag | WireStruct
		var inf = reflect.New(tt.Elem()).Elem().Interface()

		dec := newDecoderUsingTag(inf, usingTagName)
		s.subdec = dec

	case reflect.Map:

		panic("not yet")

	case reflect.Slice:

		s.kind = WireSliceFlag
		var inf = reflect.New(tt.Elem()).Elem().Interface()

		dec := newSliceDecoderUsingTagAndOpts(inf, usingTagName, opts)
		dec.subType = tt

		s.instruction = func(p unsafe.Pointer, r Reader) Reader {

			sl := int(r.ReadVarint()) // array length

			if sl == 0 {
				sli := reflect.MakeSlice(s.subType, 0, 1)
				*(*sliceHeader)(unsafe.Pointer(uintptr(p))) = sliceHeader{
					Data: unsafe.Pointer(sli.Pointer()),
					Len:  sl,
					Cap:  sl,
				}
				return r
			}

			if sheader := (*sliceHeader)(unsafe.Pointer(uintptr(p))); sheader == nil || sheader.Cap < sl {
				sli := reflect.MakeSlice(s.subType, sl, sl)
				*(*sliceHeader)(unsafe.Pointer(uintptr(p))) = sliceHeader{
					Data: unsafe.Pointer(sli.Pointer()),
					Len:  sl,
					Cap:  sl,
				}
			}

			sp := reflect.NewAt(s.subType, unsafe.Pointer(uintptr(p)))
			elem := (*(*sliceHeader)(unsafe.Pointer(sp.Pointer()))).Data
			size := s.subType.Elem().Size()

			for i := uintptr(0); i < uintptr(sl); i++ {
				var em any = unsafe.Add(elem, (i * size))
				r = s.subdec.unmarshal(r, nil, em)
			}

			return r
		}
		s.subdec = dec

	default:

		panic(fmt.Sprintf("slicedecoder unsupported type %v", tt))
	}

	return s
}

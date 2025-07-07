package glint

import (
	"fmt"
	"reflect"
	"unsafe"
)

// mapDecoder is a specialized decoder for parsing and decoding maps data from the binary format.
type mapDecoder struct {
	subdec      decoder
	instruction func(t unsafe.Pointer, r Reader) Reader
	subType     reflect.Type
	keyKind     WireType     // this is the key type we were created to parse
	valueKind   WireType     // this is the value type we were created to parse
	wireType    WireType     // this is the wire type that was actually sent
	keyWire     WireType     // the wire type of the map keys
	valueWire   WireType     // the wire type of the map values
	limits      DecodeLimits // bounds checking configuration
}

func newMapDecoderUsingTagAndOpts(t any, usingTagName string, opts tagOptions) *mapDecoder {
	return newMapDecoderUsingTagAndOptsWithLimits(t, usingTagName, opts, DefaultLimits)
}

func newMapDecoderUsingTagAndOptsWithLimits(t any, usingTagName string, opts tagOptions, limits DecodeLimits) *mapDecoder {

	m := &mapDecoder{}
	m.limits = limits

	tt := reflect.TypeOf(t)

	key := tt.Key()
	value := tt.Elem()

	m.keyKind = ReflectKindToWireType(key)
	m.valueKind = ReflectKindToWireType(value)

	switch {
	case key.Kind() == reflect.String && value.Kind() == reflect.String:

		m.instruction = func(t unsafe.Pointer, r Reader) Reader {

			m := reflect.NewAt(tt, t).Elem()

			ml := r.ReadUint() // number of items we expect to decode from the document

			if *(*unsafe.Pointer)(t) == unsafe.Pointer(nil) {
				m.Set(reflect.MakeMapWithSize(tt, int(ml)))
			}
			mapp := *(*map[string]string)(t)

			for i := uint(0); i < ml; i++ {
				key := r.ReadString()
				value := r.ReadString()

				mapp[key] = value
			}
			return r
		}

	case key.Kind() == reflect.String && value.Kind() == reflect.Int:

		m.instruction = func(t unsafe.Pointer, r Reader) Reader {

			m := reflect.NewAt(tt, t).Elem()

			ml := r.ReadUint() // number of items we expect to decode from the document

			if *(*unsafe.Pointer)(t) == unsafe.Pointer(nil) {
				m.Set(reflect.MakeMapWithSize(tt, int(ml)))
			}
			mapp := *(*map[string]int)(t)

			for i := uint(0); i < ml; i++ {
				key := r.ReadString()
				value := r.ReadInt()

				mapp[key] = value
			}
			return r
		}

	default:

		k := reflectKindToReflectValue(key, usingTagName, opts, m.limits)
		v := reflectKindToReflectValue(value, usingTagName, opts, m.limits)
		if v.subDecoder != nil {
			m.subdec = v.subDecoder
		}

		m.instruction = func(t unsafe.Pointer, r Reader) Reader {

			m := reflect.NewAt(tt, t).Elem()

			ml := r.ReadUint() // number of items we expect to decode from the document

			if *(*unsafe.Pointer)(t) == unsafe.Pointer(nil) {
				m.Set(reflect.MakeMapWithSize(tt, int(ml)))
			}

			for i := uint(0); i < ml; i++ {
				var key, value reflect.Value
				key, r = k.fun(r)
				value, r = v.fun(r)

				if v.assigner.pointer {
					value = toPointer(value)
				}

				m.SetMapIndex(key, value)
			}

			return r
		}
	}

	return m
}

func toPointer(value reflect.Value) reflect.Value {
	if value.IsValid() {
		// Check if the value is addressable
		if value.CanAddr() {
			// If it's addressable, return a pointer to the value
			return value.Addr()
		}

		// If it's not addressable, create a new pointer to a copy of the value
		ptrType := reflect.PointerTo(value.Type())
		ptrValue := reflect.New(ptrType.Elem())
		ptrValue.Elem().Set(value)

		return ptrValue
	}

	// If the value is invalid, return a zero value
	return reflect.Value{}
}

// setWireType allows this instance to have its wireType set, which is the type information pulled from the schema
func (m *mapDecoder) setWireType(wt WireType) {
	m.wireType = wt
}

// Unmarshal
func (m *mapDecoder) Unmarshal(bytes []byte, v any) error {
	panic("mapDecoder Unmarshal")
}

// unmarshal executes the instruction we've created for the type we're to decode.
func (m *mapDecoder) unmarshal(r Reader, instructions []decodeInstruction, v any) Reader {
	p := unsafe.Pointer(reflect.ValueOf(v).Pointer())
	return m.instruction(p, r)
}

// parseSchema reads the schema information sent and creates an instruction
func (m *mapDecoder) parseSchema(r Reader, instructions []decodeInstruction) ([]decodeInstruction, Reader, error) {

	// there are slightly different ways we need to parse the schema depending on the type of data we've been sent.

	m.keyWire = WireType(r.ReadVarint())
	m.valueWire = WireType(r.ReadVarint())

	if m.instruction == nil && m.subdec == nil {

		m.keyKind = m.keyWire
		m.valueKind = m.valueWire

		tt := mapWireTypesToReflectKind(m.keyWire, m.valueWire)

		k := reflectKindToReflectValue(tt.Key(), "", "", m.limits)
		v := reflectKindToReflectValue(tt.Elem(), "", "", m.limits)
		if v.subDecoder != nil {
			m.subdec = v.subDecoder
		}

		m.instruction = func(t unsafe.Pointer, r Reader) Reader {

			m := reflect.NewAt(tt, t).Elem()

			if *(*unsafe.Pointer)(t) == unsafe.Pointer(nil) {
				m.Set(reflect.MakeMap(tt))
			}

			ml := r.ReadUint()
			for i := uint(0); i < ml; i++ {
				_, r = k.fun(r)
				_, r = v.fun(r)
				// we're not interested in storing the values, just making sure they're read past by the
			}

			return r
		}
	}

	if m.keyKind != m.keyWire || m.valueKind != m.valueWire {
		return nil, r, fmt.Errorf("schema mismatch for map, expected id %v[%v] got %v[%v]", m.keyKind, m.valueKind, m.keyWire, m.valueWire)
	}

	if m.subdec != nil {
		m.subdec.setWireType(m.valueWire)
	}

	switch d := m.subdec.(type) {
	case *sliceDecoder:
		var err error
		_, r, err = d.parseSchema(r, instructions) // we need to pass the schema down and back up here to read the type byte
		if err != nil {
			return nil, r, err
		}

		if dec, ok := d.subdec.(*sliceDecoder); dec == nil && ok {
			r.ReadVarint()
		}

	case *mapDecoder:
		var err error
		_, r, err = d.parseSchema(r, instructions) // we need to pass the schema down and back up here to read the type byte
		if err != nil {
			return nil, r, err
		}

	case *decoderImpl:
		sl := r.ReadVarint()
		sb := r.Read(sl)

		var err error
		instructions, _, err = d.parseSchema(NewReader(sb), nil) // parse the schema using the sub-decoder we set up in the decodeInstruction
		if err != nil {
			return nil, r, err
		}

		d.instr = instructions // this is one of the few occasions instr is needed.
	}

	return instructions, r, nil
}

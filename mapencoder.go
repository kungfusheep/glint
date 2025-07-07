package glint

import (
	"reflect"
	"time"
	"unsafe"
)

type mapEncoder struct {
	instruction func(t unsafe.Pointer, w *Buffer) // the function we'll run to encode our date
	tt          reflect.Type                      // the type of data we're encoding
	offset      uintptr                           // used in the pointer arrhythmic to traverse through the arrays at runtime
	schema      *Buffer                           // if our data requires us to define more schema data it will be written here
}

func newMapEncoderUsingTagWithSchemaAndOpts(t any, usingTagName string, sc *Buffer, opts tagOptions) *mapEncoder {

	m := &mapEncoder{}
	m.schema = sc

	m.tt = reflect.TypeOf(t)
	m.offset = m.tt.Elem().Size()

	// used for iterating through the slice elements
	/// yeah we could save a few lines of code here if we went and added the 'AppendString' bit
	/// below into a map based on the type, but then we wouldn't get any inlining from the compiler
	/// ... so we're doing it the hard way.

	key := m.tt.Key()
	value := m.tt.Elem()

	keyType := ReflectKindToWireType(m.tt.Key())
	valueType := ReflectKindToWireType(m.tt.Elem())

	m.schema.AppendUint(uint(keyType))
	m.schema.AppendUint(uint(valueType))

	switch {
	case key.Kind() == reflect.String && value.Kind() == reflect.String:

		m.instruction = func(t unsafe.Pointer, w *Buffer) {
			m := *(*map[string]string)(t)

			if *(*unsafe.Pointer)(t) == unsafe.Pointer(nil) {
				w.AppendUint(0) // zero length
				return
			}

			w.AppendUint(uint(len(m))) // length
			for k, v := range m {
				w.AppendString(k)
				w.AppendString(v)
			}
		}

	case key.Kind() == reflect.String && value.Kind() == reflect.Int:

		m.instruction = func(t unsafe.Pointer, w *Buffer) {
			m := *(*map[string]int)(t)

			if *(*unsafe.Pointer)(t) == unsafe.Pointer(nil) {
				w.AppendUint(0) // zero length
				return
			}

			w.AppendUint(uint(len(m))) // length
			for k, v := range m {
				w.AppendString(k)
				w.AppendInt(v)
			}
		}

	default:

		k := reflectKindToAppender(key, usingTagName, opts)
		if k.subenc != nil {
			m.schema.Bytes = append(m.schema.Bytes, k.subenc.Schema().Bytes...)
			k.subenc.ClearSchema()
		}
		v := reflectKindToAppender(value, usingTagName, opts)
		if v.subenc != nil {
			m.schema.Bytes = append(m.schema.Bytes, v.subenc.Schema().Bytes...)
			v.subenc.ClearSchema()
		}

		m.instruction = func(t unsafe.Pointer, w *Buffer) {
			m := reflect.NewAt(m.tt, t).Elem()

			if *(*unsafe.Pointer)(t) == unsafe.Pointer(nil) {
				w.AppendUint(0) // zero length
				return
			}

			w.AppendUint(uint(m.Len())) // length
			iter := m.MapRange()

			for iter.Next() {
				key := iter.Key()
				value := iter.Value()

				var keyPtr, valuePtr unsafe.Pointer
				if key.CanAddr() {
					keyPtr = unsafe.Pointer(key.Addr().Pointer())
				} else {
					// Handle the case when key cannot be addressed
					tempVal := reflect.New(key.Type()).Elem()
					tempVal.Set(key)
					keyPtr = unsafe.Pointer(tempVal.Addr().Pointer())
				}

				if value.CanAddr() {
					valuePtr = unsafe.Pointer(value.Addr().Pointer())
				} else {
					// Handle the case when value cannot be addressed
					tempVal := reflect.New(value.Type()).Elem()
					tempVal.Set(value)
					valuePtr = unsafe.Pointer(tempVal.Addr().Pointer())
				}

				k.fun(keyPtr, w)
				v.fun(valuePtr, w)
			}
		}
	}

	return m
}

type appender struct {
	fun     func(unsafe.Pointer, *Buffer)
	pointer bool
	subenc  encoder
}

// reflectKindToAppender returns a function that can be used to append the data
func reflectKindToAppender(k reflect.Type, usingTagName string, opts tagOptions) appender {

	pointerWrap := false
	var fun func(unsafe.Pointer, *Buffer)
	var sub encoder

	kind := k.Kind()
	if kind == reflect.Pointer {
		pointerWrap = true
		kind = k.Elem().Kind()
	}

	switch kind {
	case reflect.Uint8:
		fun = func(p unsafe.Pointer, b *Buffer) {
			b.AppendUint8(*(*uint8)(p))
		}
	case reflect.Uint16:
		fun = func(p unsafe.Pointer, b *Buffer) {
			b.AppendUint16(*(*uint16)(p))
		}
	case reflect.Uint32:
		fun = func(p unsafe.Pointer, b *Buffer) {
			b.AppendUint32(*(*uint32)(p))
		}
	case reflect.Uint64:
		fun = func(p unsafe.Pointer, b *Buffer) {
			b.AppendUint64(*(*uint64)(p))
		}
	case reflect.Uint:
		fun = func(p unsafe.Pointer, b *Buffer) {
			b.AppendUint(*(*uint)(p))
		}
	case reflect.Int8:

		// only needs to be defined if we're a pointer field due to the fast path in marshal
		fun = func(p unsafe.Pointer, b *Buffer) {
			b.AppendInt8(*(*int8)(p))
		}

	case reflect.Int16:
		fun = func(p unsafe.Pointer, b *Buffer) {
			b.AppendInt16(*(*int16)(p))
		}
	case reflect.Int32:
		fun = func(p unsafe.Pointer, b *Buffer) {
			b.AppendInt32(*(*int32)(p))
		}
	case reflect.Int64:
		fun = func(p unsafe.Pointer, b *Buffer) {
			b.AppendInt64(*(*int64)(p))
		}

	case reflect.Float32:
		fun = func(p unsafe.Pointer, b *Buffer) {
			b.AppendFloat32(*(*float32)(p))
		}

	case reflect.Float64:
		fun = func(p unsafe.Pointer, b *Buffer) {
			b.AppendFloat64(*(*float64)(p))
		}

	case reflect.Bool:
		fun = func(p unsafe.Pointer, b *Buffer) {
			b.AppendBool(*(*bool)(p))
		}

	case reflect.Int:

		// doesn't need to be defined unless its a pointer field due to the fast path in marshal
		fun = func(p unsafe.Pointer, b *Buffer) {
			b.AppendInt(*(*int)(p))
		}

	case reflect.String:

		// this doesn't need to be defined unless we know its going to be a pointer, due to the fast path in marshal
		fun = func(p unsafe.Pointer, b *Buffer) {
			b.AppendString(*(*string)(p))
		}

	case reflect.Map:

		mpEnc := newMapEncoderUsingTagWithSchemaAndOpts(reflect.New(k).Elem().Interface(), usingTagName, &Buffer{}, opts)
		fun = func(p unsafe.Pointer, b *Buffer) {
			var em = p
			mpEnc.Marshal(em, b)
		}
		sub = mpEnc

	case reflect.Slice:

		// create a slice encoder to handle the slice type then hand off to it in the fun
		slEnc := newSliceEncoderUsingTagWithSchemaAndOpts(reflect.New(k).Elem().Interface(), usingTagName, &Buffer{}, opts)
		fun = func(p unsafe.Pointer, b *Buffer) {
			var em = p
			slEnc.Marshal(em, b)
		}
		sub = slEnc

	case reflect.Struct:

		// check first if we're a time field because we have a bespoke method for encoding time
		if k == timeType {
			fun = func(p unsafe.Pointer, b *Buffer) {
				b.AppendTime(*(*time.Time)(p))
			}
			break
		}

		// create a new encoder to handle the sub-type and hand off to it in the fun
		var inf any
		if pointerWrap {
			inf = reflect.New(k.Elem()).Elem().Interface()
		} else {
			inf = reflect.New(k).Elem().Interface()
		}

		se := newEncoderUsingTag(inf, usingTagName)

		fun = func(p unsafe.Pointer, b *Buffer) {
			var em any = p
			se.Marshal(em, b)
		}

		sub = se
	}

	if fun == nil {
		panic("invalid type passed to reflectKindToAppender")
	}

	if pointerWrap {
		fun = derefAppend(fun)
	}

	return appender{fun: fun, pointer: pointerWrap, subenc: sub}
}

func (m *mapEncoder) Marshal(v any, b *Buffer) {
	p := unsafe.Pointer(reflect.ValueOf(v).Pointer())
	m.instruction(p, b)
}

func (m *mapEncoder) Schema() *Buffer {
	return m.schema
}

func (m *mapEncoder) ClearSchema() {
	m.schema.Reset()
}

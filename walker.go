package glint

import (
	"errors"
	"unsafe"
)

// Visitor is an interface that can be implemented to walk a document
type Visitor interface {
	VisitFlags(flags byte) error
	VisitSchemaHash(hash []byte) error
	VisitField(name string, wire WireType, body Reader) (Reader, error)
	VisitArrayStart(name string, wire WireType, length int) error
	VisitArrayEnd(name string) error
	VisitStructStart(name string) error
	VisitStructEnd(name string) error
}

func Walk(doc []byte, visitor Visitor) error {
	w := NewWalker(doc)
	return w.Walk(visitor)
}

// Walker walks a document
type Walker struct {
	r Reader
}

// NewWalker creates a new walker
func NewWalker(doc []byte) Walker {
	return Walker{r: NewReader(doc)}
}

// Walk walks the document
func (w *Walker) Walk(visitor Visitor) error {

	visitor.VisitFlags(w.r.ReadByte())
	visitor.VisitSchemaHash(w.r.Read(4))

	schema := NewReader(w.r.Read(w.r.ReadVarint()))
	body := NewReader(w.r.Remaining())
	_, body = w.walk(visitor, schema, body)

	if body.BytesLeft() > 0 {
		panic("underparse")
	}

	return nil
}

// ErrSkipVisit is returned by a visitor to indicate that the walker should skip visiting the current field
var ErrSkipVisit = errors.New("skip visit")

// walk walks the schema and body in parallel, calling the visitor as it goes.
func (w *Walker) walk(visitor Visitor, schema, body Reader) (Reader, Reader) {
	for schema.BytesLeft() > 0 {

		typeID := WireType(schema.ReadVarint())
		nameb := schema.Read(schema.ReadVarint())
		name := *(*string)(unsafe.Pointer(&nameb)) //avoids allocation of a new string for each field

		// do we need to do something more specific here?
		// sigh
		var ok bool
		schema, body, ok = w.walkSubschema(typeID, schema, body, visitor, name)
		if ok {
			continue
		}

		var err error
		body, err = visitor.VisitField(name, typeID, body) // no, just a normal field
		switch err {
		case ErrSkipVisit:
			fieldBytes(&body, typeID) // allows the visitor to return an error an we skip over the field

			// if we don't read the field here, everything that comes after this will break, including reading
			// array lengths etc.

		case nil:
		default:
			return schema, body
		}
	}

	return schema, body
}

// walkSubschema walks a subschema, calling the visitor as it goes.
func (w *Walker) walkSubschema(typeID WireType, schema, body Reader, visitor Visitor, name string) (Reader, Reader, bool) {
	switch {
	case typeID == WireStruct:
		schema, body = w.walkStruct(visitor, name, schema, body)
		return schema, body, true

	case typeID&WireSliceFlag > 0:
		schema, body = w.walkArray(visitor, name, typeID, schema, body)
		return schema, body, true

	case typeID == WireMap:
		return schema, body, true
	}
	return schema, body, false
}

// walkArray walks an array, calling the visitor as it goes.
func (w *Walker) walkArray(visitor Visitor, name string, typeID WireType, schema, body Reader) (Reader, Reader) {
	visitor.VisitArrayStart(name, WireType(typeID), 0) // start of a slice
	name = ""

	switch typeID {
	case WireSliceFlag:
		nextLevelType := WireType(schema.ReadVarint())

		// read the length of the slice
		length := body.ReadVarint()

		first := schema

		schema, body = w.walkArray(visitor, name, nextLevelType, schema, body)
		for i := uint(1); i < length; i++ {
			_, body = w.walkArray(visitor, name, nextLevelType, first, body)
		}

	default:
		typeID = WireType(typeID & WireTypeMask)

		// read the length of the slice
		length := body.ReadVarint()

		// if length == 0 {
		// 	break
		// }

		first := schema // we need to reset the schema for each element in the array
		schema, body, _ = w.walkSubschema(typeID, schema, body, visitor, name)
		for i := uint(1); i < length; i++ {
			_, body, _ = w.walkSubschema(typeID, first, body, visitor, name)
		}
	}

	visitor.VisitArrayEnd(name) // end of a slice
	return schema, body
}

// walkStruct walks a struct, calling the visitor as it goes.
func (w *Walker) walkStruct(visitor Visitor, name string, schema, body Reader) (Reader, Reader) {
	visitor.VisitStructStart(name) // start of a struct

	s := NewReader(schema.Read(schema.ReadVarint()))
	_, body = w.walk(visitor, s, body)

	visitor.VisitStructEnd(name) // end of a struct

	return schema, body
}

// fieldBytes returns the raw bytes that represent a field of a given wire type
func fieldBytes(body *Reader, typeID WireType) []byte {

	switch typeID {
	case WireInt8, WireUint8, WireBool:
		return body.Read(1)

	case WireInt, WireInt16, WireInt32, WireInt64,
		WireUint, WireUint16, WireUint32, WireUint64,
		WireFloat32, WireFloat64:

		body.SetMark()
		body.SkipVarint()
		return body.BytesFromMark()

	case WireString, WireBytes, WireTime:
		return body.Read(body.ReadVarint())
	}

	length := body.ReadVarint()
	return body.Read(length)
}

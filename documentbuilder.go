package glint

import (
	"hash/crc32"
	"time"
)

// DocumentBuilder is a simple inline progressive builder. You add your properties and it builds the document up as you go along.
type DocumentBuilder struct {
	schema Buffer
	body   Buffer
}

// AppendNestedDocument appends another document within this one. Equivalent of a nested struct.
func (d *DocumentBuilder) AppendNestedDocument(name string, value *DocumentBuilder) *DocumentBuilder {
	d.schema.Bytes = appendField(d.schema.Bytes, name, WireStruct)
	d.schema.AppendBytes(value.schema.Bytes)
	d.body.Bytes = append(d.body.Bytes, value.body.Bytes...)
	return d
}

// AppendSlice adds a slice field to the document against a given name
func (d *DocumentBuilder) AppendSlice(name string, value SliceBuilder) *DocumentBuilder {
	if value.wire == 0 {
		return d // if the slice was empty then it has no type so theres nothing to encode
	}

	d.schema.Bytes = appendField(d.schema.Bytes, name, WireSliceFlag|value.wire)
	if len(value.schema.Bytes) > 0 {
		d.schema.Bytes = append(d.schema.Bytes, value.schema.Bytes...)
	}
	d.body.Bytes = append(d.body.Bytes, value.body.Bytes...)
	return d
}

// AppendString adds a string field to the document against a given name
func (d *DocumentBuilder) AppendString(name, value string) *DocumentBuilder {
	d.schema.Bytes = appendField(d.schema.Bytes, name, WireString)
	d.body.AppendString(value)
	return d
}

// AppendInt adds a int field to the document against a given name
func (d *DocumentBuilder) AppendInt(name string, value int) *DocumentBuilder {
	d.schema.Bytes = appendField(d.schema.Bytes, name, WireInt)
	d.body.AppendInt(value)
	return d
}

// AppendBytes adds a bytes field to the document against a given name
func (d *DocumentBuilder) AppendBytes(name string, value []byte) *DocumentBuilder {
	d.schema.Bytes = appendField(d.schema.Bytes, name, WireBytes)
	d.body.AppendBytes(value)
	return d
}

// AppendUint8 adds a uint8 field to the document against a given name
func (d *DocumentBuilder) AppendUint8(name string, value uint8) *DocumentBuilder {
	d.schema.Bytes = appendField(d.schema.Bytes, name, WireUint8)
	d.body.AppendUint8(value)
	return d
}

// AppendUint16 adds a uint16 field to the document against a given name
func (d *DocumentBuilder) AppendUint16(name string, value uint16) *DocumentBuilder {
	d.schema.Bytes = appendField(d.schema.Bytes, name, WireUint16)
	d.body.AppendUint16(value)
	return d
}

// AppendUint32 adds a uint32 field to the document against a given name
func (d *DocumentBuilder) AppendUint32(name string, value uint32) *DocumentBuilder {
	d.schema.Bytes = appendField(d.schema.Bytes, name, WireUint32)
	d.body.AppendUint32(value)
	return d
}

// AppendUint64 adds a uint64 field to the document against a given name
func (d *DocumentBuilder) AppendUint64(name string, value uint64) *DocumentBuilder {
	d.schema.Bytes = appendField(d.schema.Bytes, name, WireUint64)
	d.body.AppendUint64(value)
	return d
}

// AppendUint adds a uint field to the document against a given name
func (d *DocumentBuilder) AppendUint(name string, value uint) *DocumentBuilder {
	d.schema.Bytes = appendField(d.schema.Bytes, name, WireUint)
	d.body.AppendUint(value)
	return d
}

// AppendInt8 adds a int8 field to the document against a given name
func (d *DocumentBuilder) AppendInt8(name string, value int8) *DocumentBuilder {
	d.schema.Bytes = appendField(d.schema.Bytes, name, WireInt8)
	d.body.AppendInt8(value)
	return d
}

// AppendInt16 adds a int16 field to the document against a given name
func (d *DocumentBuilder) AppendInt16(name string, value int16) *DocumentBuilder {
	d.schema.Bytes = appendField(d.schema.Bytes, name, WireInt16)
	d.body.AppendInt16(value)
	return d
}

// AppendInt32 adds a int32 field to the document against a given name
func (d *DocumentBuilder) AppendInt32(name string, value int32) *DocumentBuilder {
	d.schema.Bytes = appendField(d.schema.Bytes, name, WireInt32)
	d.body.AppendInt32(value)
	return d
}

// AppendInt64 adds a int64 field to the document against a given name
func (d *DocumentBuilder) AppendInt64(name string, value int64) *DocumentBuilder {
	d.schema.Bytes = appendField(d.schema.Bytes, name, WireInt64)
	d.body.AppendInt64(value)
	return d
}

// AppendFloat32 adds a float32 field to the document against a given name
func (d *DocumentBuilder) AppendFloat32(name string, value float32) *DocumentBuilder {
	d.schema.Bytes = appendField(d.schema.Bytes, name, WireFloat32)
	d.body.AppendFloat32(value)
	return d
}

// AppendFloat64 adds a float64 field to the document against a given name
func (d *DocumentBuilder) AppendFloat64(name string, value float64) *DocumentBuilder {
	d.schema.Bytes = appendField(d.schema.Bytes, name, WireFloat64)
	d.body.AppendFloat64(value)
	return d
}

// AppendTime adds a time field to the document against a given name
func (d *DocumentBuilder) AppendTime(name string, value time.Time) *DocumentBuilder {
	d.schema.Bytes = appendField(d.schema.Bytes, name, WireTime)
	d.body.AppendTime(value)
	return d
}

// AppendBool adds a bool field to the document against a given name
func (d *DocumentBuilder) AppendBool(name string, value bool) *DocumentBuilder {
	d.schema.Bytes = appendField(d.schema.Bytes, name, WireBool)
	d.body.AppendBool(value)
	return d
}

// WriteTo writes the document to a buffer
func (d *DocumentBuilder) WriteTo(b *Buffer) {

	// 8 bits reserved for flags
	// 32 bits reserved for schema checksum (below)
	b.Bytes = append(b.Bytes, []byte{0, 0, 0, 0, 0}...)

	b.AppendBytes(d.schema.Bytes)

	// encode a hash of the schema into the schema
	crc := crc32.ChecksumIEEE(b.Bytes[5:])
	h := b.Bytes[1:5]
	h[0] = byte(crc)
	h[1] = byte(crc >> 8)
	h[2] = byte(crc >> 16)
	h[3] = byte(crc >> 24)

	b.Bytes = append(b.Bytes, d.body.Bytes...)
}

// Bytes returns the document as a byte array
func (d *DocumentBuilder) Bytes() []byte {
	b := Buffer{}
	d.WriteTo(&b)
	return b.Bytes
}

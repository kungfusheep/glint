package glint

import (
	"time"
	"unsafe"
)

// Reader provides sequential access to encoded data with position tracking.
type Reader struct {
	position uint // current read position (first for alignment)
	bytes    []byte
	mark     uint // saved position for later reference
}

func NewReader(b []byte) Reader {
	return Reader{bytes: b}
}
func (r *Reader) ReadVarint() uint {
	var sf uint8
	var v uint
	b := r.bytes

	for ; r.position < uint(len(b)); r.position++ {
		d := b[r.position]

		if d&0b10000000 == 0 {
			v |= uint(d) << sf
			r.advance(1)
			return v
		}

		v |= uint(d&0b01111111) << sf
		sf += 7
	}

	return 0
}

// ReadZigzagVarint decodes a zigzag-encoded variable integer.
func (r *Reader) ReadZigzagVarint() int {
	i := r.ReadVarint()
	return (int(i) >> 1) ^ -(int(i) & 1)
}

// ReadUint8 extracts a single byte
func (r *Reader) ReadUint8() uint8 {
	return r.ReadByte()
}

// ReadUint16 decodes a variable-length uint16
func (r *Reader) ReadUint16() uint16 {
	return uint16(r.ReadVarint())
}

// ReadUint32 decodes a variable-length uint32
func (r *Reader) ReadUint32() uint32 {
	return uint32(r.ReadVarint())
}

// ReadUint64 decodes a variable-length uint64
func (r *Reader) ReadUint64() uint64 {
	return uint64(r.ReadVarint())
}

// ReadUint decodes a variable-length uint
func (r *Reader) ReadUint() uint {
	return r.ReadVarint()
}

// ReadInt8 extracts a signed byte
func (r *Reader) ReadInt8() int8 {
	return int8(r.ReadByte())
}

// ReadInt16 decodes a zigzag-encoded int16
func (r *Reader) ReadInt16() int16 {
	return int16(r.ReadZigzagVarint())
}

// ReadInt32 decodes a zigzag-encoded int32
func (r *Reader) ReadInt32() int32 {
	return int32(r.ReadZigzagVarint())
}

// ReadInt64 decodes a variable-length int64
func (r *Reader) ReadInt64() int64 {
	return int64(r.ReadVarint())
}

// ReadFloat32 decodes a float32 from its uint32 bit representation
func (r *Reader) ReadFloat32() float32 {
	v := uint32(r.ReadVarint())
	return *(*float32)(unsafe.Pointer(&v))
}

// ReadFloat64 decodes a float64 from its uint64 bit representation
func (r *Reader) ReadFloat64() float64 {
	v := r.ReadVarint()
	return *(*float64)(unsafe.Pointer(&v))
}

// ReadInt decodes a zigzag-encoded int
func (r *Reader) ReadInt() int {
	return r.ReadZigzagVarint()
}

// ReadString decodes a length-prefixed string
func (r *Reader) ReadString() string {
	l := r.ReadVarint()
	if r.position+l > uint(len(r.bytes)) {
		panic("read out of bounds")
	}

	b := r.Read(l)
	return *(*string)(unsafe.Pointer(&b))
}

// ReadBool interprets a byte as boolean: 1 = true, 0 = false.
func (r *Reader) ReadBool() bool {
	return r.ReadByte() == 1
}

// ReadTime decodes a binary-marshaled time value
func (r *Reader) ReadTime() time.Time {
	l := r.ReadVarint()
	t := time.Time{}
	_ = t.UnmarshalBinary(r.Read(l))
	return t
}

// ReadStringSlice decodes a length-prefixed array of strings
func (r *Reader) ReadStringSlice() []string {
	length := r.ReadUint()
	s := make([]string, length)
	for i := uint(0); i < length; i++ {
		s[i] = r.ReadString()
	}

	return s
}

// ReadUintSlice extracts an array of variable-length uints
func (r *Reader) ReadUintSlice() []uint {
	length := r.ReadUint()
	s := make([]uint, length)
	for i := uint(0); i < length; i++ {
		s[i] = r.ReadUint()
	}

	return s
}

// ReadIntSlice decodes an array of zigzag-encoded ints
func (r *Reader) ReadIntSlice() []int {
	length := r.ReadUint()
	s := make([]int, length)
	for i := uint(0); i < length; i++ {
		s[i] = r.ReadInt()
	}

	return s
}

// ReadUint8Slice returns a byte slice of the specified length
func (r *Reader) ReadUint8Slice() []uint8 {
	length := r.ReadUint()
	return r.Read(length)
}

// ReadUint16Slice decodes an array of variable-length uint16s
func (r *Reader) ReadUint16Slice() []uint16 {
	length := r.ReadUint()
	s := make([]uint16, length)
	for i := uint(0); i < length; i++ {
		s[i] = r.ReadUint16()
	}

	return s
}

// ReadUint32Slice extracts multiple uint32 values
func (r *Reader) ReadUint32Slice() []uint32 {
	length := r.ReadUint()
	s := make([]uint32, length)
	for i := uint(0); i < length; i++ {
		s[i] = r.ReadUint32()
	}

	return s
}

// ReadUint64Slice decodes a sequence of uint64 values
func (r *Reader) ReadUint64Slice() []uint64 {
	length := r.ReadUint()
	s := make([]uint64, length)
	for i := uint(0); i < length; i++ {
		s[i] = r.ReadUint64()
	}

	return s
}

// ReadInt8Slice extracts an array of signed bytes
func (r *Reader) ReadInt8Slice() []int8 {
	length := r.ReadUint()
	s := make([]int8, length)
	for i := uint(0); i < length; i++ {
		s[i] = r.ReadInt8()
	}

	return s
}

// ReadInt16Slice retrieves multiple zigzag-encoded int16s
func (r *Reader) ReadInt16Slice() []int16 {
	length := r.ReadUint()
	s := make([]int16, length)
	for i := uint(0); i < length; i++ {
		s[i] = r.ReadInt16()
	}

	return s
}

// ReadInt32Slice decodes a collection of zigzag-encoded int32s
func (r *Reader) ReadInt32Slice() []int32 {
	length := r.ReadUint()
	s := make([]int32, length)
	for i := uint(0); i < length; i++ {
		s[i] = r.ReadInt32()
	}

	return s
}

// ReadInt64Slice extracts an array of variable-length int64s
func (r *Reader) ReadInt64Slice() []int64 {
	length := r.ReadUint()
	s := make([]int64, length)
	for i := uint(0); i < length; i++ {
		s[i] = r.ReadInt64()
	}

	return s
}

// ReadFloat32Slice decodes multiple float32 values
func (r *Reader) ReadFloat32Slice() []float32 {
	length := r.ReadUint()
	s := make([]float32, length)
	for i := uint(0); i < length; i++ {
		s[i] = r.ReadFloat32()
	}

	return s
}

// ReadFloat64Slice retrieves an array of float64 values
func (r *Reader) ReadFloat64Slice() []float64 {
	length := r.ReadUint()
	s := make([]float64, length)
	for i := uint(0); i < length; i++ {
		s[i] = r.ReadFloat64()
	}

	return s
}

// ReadBoolSlice decodes an array of boolean values
func (r *Reader) ReadBoolSlice() []bool {
	length := r.ReadUint()
	s := make([]bool, length)
	for i := uint(0); i < length; i++ {
		s[i] = r.ReadBool()
	}

	return s
}

// ReadTimeSlice extracts multiple binary-encoded time values
func (r *Reader) ReadTimeSlice() []time.Time {
	length := r.ReadUint()
	s := make([]time.Time, length)
	for i := uint(0); i < length; i++ {
		s[i] = r.ReadTime()
	}

	return s
}

// ReadBytesSlice decodes a collection of length-prefixed byte arrays
func (r *Reader) ReadBytesSlice() [][]byte {
	length := r.ReadUint()
	s := make([][]byte, length)
	for i := uint(0); i < length; i++ {
		s[i] = r.Read(r.ReadVarint())
	}

	return s
}

// SkipVarint bypasses a variable-length integer without decoding
func (r *Reader) SkipVarint() {

	index := r.position
	b := r.bytes

loop:
	if b[index]&0b10000000 == 0 {
		r.move(index + 1)
		return
	}
	index++

	goto loop
}

// SetMark saves the current position for later reference
func (r *Reader) SetMark() {
	r.mark = r.position
}

// Mark retrieves the saved position value
func (r *Reader) Mark() uint {
	return r.mark
}

// ResetMark jumps back to the saved position
func (r *Reader) ResetMark() {
	r.position = r.mark
}

// BytesFromMark extracts data between the saved position and current location
func (r *Reader) BytesFromMark() []byte {
	return r.bytes[r.mark:r.position]
}

// ReadByte extracts the next byte
func (r *Reader) ReadByte() byte {
	p := r.position
	r.advance(1)
	return r.bytes[p]
}

// Read extracts the specified number of bytes
func (r *Reader) Read(l uint) []byte {
	if r.position+l > uint(len(r.bytes)) {
		panic("read out of bounds")
	}

	p := r.position
	r.advance(l)
	return r.bytes[p : p+l]
}

// Unread moves backward by the specified amount
func (r *Reader) Unread(l uint) {
	r.position -= l
}

// Skip moves forward without extracting data
func (r *Reader) Skip(l uint) {
	r.advance(l)
}

func (r *Reader) move(p uint) {
	r.position = p
}

func (r *Reader) advance(a uint) {
	r.position += a
}

// BytesLeft calculates remaining unread bytes
func (r *Reader) BytesLeft() uint {
	return uint(len(r.bytes)) - r.position
}

// Remaining provides all unread data as a slice
func (r *Reader) Remaining() []byte {
	return r.bytes[r.position:]
}

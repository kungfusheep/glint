package glint

import (
	"encoding/binary"
	"net/http"
	"strconv"
	"sync"
	"time"
	"unsafe"
)

// Buffer accumulates encoded data during serialization. Supports only append operations
// for efficiency.
type Buffer struct {
	Bytes         []byte
	TrustedSchema bool // when true, omits schema body for trusted connections
}

// Reset clears the buffer contents but preserves allocated memory
func (b *Buffer) Reset() {
	b.Bytes = b.Bytes[:0]
	b.TrustedSchema = false
}

var bufpool = sync.Pool{
	New: func() any { return &Buffer{} },
}

// NewBufferFromPool obtains a reset Buffer from the pool. Call ReturnToPool when finished.
// For existing memory, create directly: `buf := Buffer{mySlice[:0]}` - pooling is optional.
func NewBufferFromPool() *Buffer {
	b := bufpool.Get().(*Buffer)
	b.Reset()
	return b
}

// NewBufferFromPoolWithCap acquires a pooled Buffer with guaranteed capacity.
// Call ReturnToPool after use.
func NewBufferFromPoolWithCap(size int) *Buffer {
	b := bufpool.Get().(*Buffer)

	if c := cap(b.Bytes); c < size {
		b.Bytes = make([]byte, 0, size)
	} else if c > 0 {
		b.Reset()
	}

	return b
}

// Trustee interface enables schema trust verification. HTTPTrustee provides a default HTTP-based implementation.
type Trustee interface {
	Hash() uint32
}

// HTTPTrustee implements trust verification via the X-Glint-Trust HTTP header.
// Use with NewBufferWithTrust to enable schema omission for trusted requests.
func HTTPTrustee(r *http.Request) Trustee {
	return httpTrustee{Request: r}
}

// httpTrustee is a default implmentation of a Trustee that uses the X-Glint-Trust header to determine if the schema can be trusted
type httpTrustee struct {
	Request *http.Request
}

// Hash returns the hash in the X-Glint-Trust header
func (h httpTrustee) Hash() uint32 {
	header := h.Request.Header.Get("X-Glint-Trust")
	if header == "" {
		return 0
	}
	trustUint, _ := strconv.ParseUint(header, 10, 32)
	return uint32(trustUint)
}

// NewBufferWithTrust acquires a pooled Buffer and enables trust mode if schema hashes match.
// Remember to call ReturnToPool after use.
func NewBufferWithTrust(r Trustee, e *encoderImpl) *Buffer {
	b := bufpool.Get().(*Buffer)
	b.Reset()

	if binary.LittleEndian.Uint32(e.header.Bytes[1:5]) == r.Hash() {
		b.TrustedSchema = true
	}

	return b
}

// ReturnToPool releases the buffer back to the pool. Using the buffer after this call
// results in undefined behavior.
func (b *Buffer) ReturnToPool() {
	bufpool.Put(b)
}

// AppendString encodes a string with length prefix into the buffer.
func (b *Buffer) AppendString(value string) {
	appendVarint(b, uint64(len(value)))
	b.Bytes = append(b.Bytes, value...)
}

// AppendBytes encodes a byte slice with length prefix into the buffer.
func (b *Buffer) AppendBytes(value []byte) {
	appendVarint(b, uint64(len(value)))
	b.Bytes = append(b.Bytes, value...)
}

// AppendUint8 adds a single byte to the buffer.
func (b *Buffer) AppendUint8(value uint8) {
	b.Bytes = append(b.Bytes, value)
}

// AppendUint16 encodes a uint16 as a variable-length integer.
func (b *Buffer) AppendUint16(value uint16) {
	appendVarint(b, uint64(value))
}

// AppendUint32 encodes a uint32 as a variable-length integer.
func (b *Buffer) AppendUint32(value uint32) {
	appendVarint(b, uint64(value))
}

// AppendUint64 encodes a uint64 as a variable-length integer.
func (b *Buffer) AppendUint64(value uint64) {
	appendVarint(b, value)
}

// AppendUint encodes a uint as a variable-length integer.
func (b *Buffer) AppendUint(value uint) {
	appendVarint(b, uint64(value))
}

// appendVarint uses variable-length encoding to minimize bytes used for the value
func appendVarint(b *Buffer, value uint64) {
	b.Bytes = appendVarintb(b.Bytes, value)
}

// appendVarintb performs variable-length encoding directly to a byte slice
func appendVarintb(b []byte, value uint64) []byte {
	for value >= 0b10000000 {
		b = append(b, byte((value&0b01111111)|0b10000000))
		value >>= 7
	}
	b = append(b, byte(value))
	return b
}

func appendVarintZigzag(b *Buffer, value int64) {
	appendVarint(b, uint64((value>>63)^(value<<1)))
}

// AppendInt8 adds a signed byte to the buffer.
func (b *Buffer) AppendInt8(value int8) {
	b.Bytes = append(b.Bytes, byte(value))
}

// AppendInt16 encodes a signed int16 using zigzag encoding.
func (b *Buffer) AppendInt16(value int16) {
	appendVarintZigzag(b, int64(value))
}

// AppendInt32 encodes a signed int32 using zigzag encoding.
func (b *Buffer) AppendInt32(value int32) {
	appendVarintZigzag(b, int64(value))
}

// AppendInt64 encodes a signed int64 as a variable-length integer.
func (b *Buffer) AppendInt64(value int64) {
	appendVarint(b, uint64(value))
}

// AppendInt encodes a signed int using zigzag encoding.
func (b *Buffer) AppendInt(value int) {
	appendVarintZigzag(b, int64(value))

}

// AppendFloat32 encodes a float32 by converting to uint32 bits
func (b *Buffer) AppendFloat32(value float32) {
	appendVarint(b, uint64(*(*uint32)(unsafe.Pointer(&value))))
}

// AppendFloat64 encodes a float64 by converting to uint64 bits
func (b *Buffer) AppendFloat64(value float64) {
	appendVarint(b, *(*uint64)(unsafe.Pointer(&value)))
}

// AppendTime encodes a time value using Go's binary marshaling
func (b *Buffer) AppendTime(t time.Time) {
	buf, err := t.MarshalBinary()
	if err != nil {
		return
	}

	b.AppendBytes(buf)
}

// AppendBool encodes a boolean as a single byte: 1 for true, 0 for false.
func (b *Buffer) AppendBool(value bool) {
	if value {
		b.Bytes = append(b.Bytes, 1)
	} else {
		b.Bytes = append(b.Bytes, 0)
	}
}

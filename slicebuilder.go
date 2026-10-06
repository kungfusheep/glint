package glint

import "time"

// SliceBuilder can build various types of glint slices and be appended into a DocumentBuilder
type SliceBuilder struct {
	schema Buffer
	body   Buffer
	wire   WireType
}

// AppendNestedDocumentSlice appends a nesteddocument slice
func (s *SliceBuilder) AppendNestedDocumentSlice(value []DocumentBuilder) {
	if len(value) == 0 {
		return
	}

	s.wire = WireStruct
	s.schema.AppendBytes(value[0].schema.Bytes)

	s.body.AppendUint(uint(len(value)))
	for i := 0; i < len(value); i++ {
		s.body.Bytes = append(s.body.Bytes, value[i].body.Bytes...)
	}
}

func (s *SliceBuilder) AppendSlice(value []SliceBuilder) {
	if len(value) == 0 {
		return
	}

	s.wire = WireSliceFlag
	s.schema.Bytes = appendVarintb(s.schema.Bytes, uint64(WireSliceFlag|value[0].wire))

	if len(value[0].schema.Bytes) > 0 {
		s.schema.Bytes = append(s.schema.Bytes, value[0].schema.Bytes...)
	}

	s.body.AppendUint(uint(len(value)))
	for i := 0; i < len(value); i++ {
		s.body.Bytes = append(s.body.Bytes, value[i].body.Bytes...)
	}
}

// AppendStringSlice appends a string slice to this slice builder
func (s *SliceBuilder) AppendStringSlice(value []string) {
	s.wire = WireString
	s.body.AppendUint(uint(len(value)))
	for i := 0; i < len(value); i++ {
		s.body.AppendString(value[i])
	}
}

// AppendIntSlice appends a int slice to this slice builder
func (s *SliceBuilder) AppendIntSlice(value []int) {
	s.wire = WireInt
	s.body.AppendUint(uint(len(value)))
	for i := 0; i < len(value); i++ {
		s.body.AppendInt(value[i])
	}
}

// AppendIntSliceDelta appends a int slice using delta encoding
func (s *SliceBuilder) AppendIntSliceDelta(value []int) {
	s.wire = WireInt | WireDeltaFlag
	s.body.AppendUint(uint(len(value)))
	if len(value) == 0 {
		return
	}
	
	// First value as-is
	s.body.AppendInt(value[0])
	
	// Subsequent values as zigzag-encoded deltas
	prev := value[0]
	for i := 1; i < len(value); i++ {
		delta := int64(value[i] - prev)
		appendVarintZigzag(&s.body, delta)
		prev = value[i]
	}
}

// AppendBytesSlice appends a bytes slice to this slice builder
func (s *SliceBuilder) AppendBytesSlice(value [][]byte) {
	s.wire = WireBytes
	s.body.AppendUint(uint(len(value)))
	for i := 0; i < len(value); i++ {
		s.body.AppendBytes(value[i])
	}
}

// AppendUint8Slice appends a uint8 slice to this slice builder
func (s *SliceBuilder) AppendUint8Slice(value []uint8) {
	s.wire = WireUint8
	s.body.AppendUint(uint(len(value)))
	s.body.AppendBytes(value)
}


// AppendUint16Slice appends a uint16 slice to this slice builder
func (s *SliceBuilder) AppendUint16Slice(value []uint16) {
	s.wire = WireUint16
	s.body.AppendUint(uint(len(value)))
	for i := 0; i < len(value); i++ {
		s.body.AppendUint16(value[i])
	}
}

// AppendUint16SliceDelta appends a uint16 slice using delta encoding
func (s *SliceBuilder) AppendUint16SliceDelta(value []uint16) {
	s.wire = WireUint16 | WireDeltaFlag
	s.body.AppendUint(uint(len(value)))
	if len(value) == 0 {
		return
	}
	
	// First value as-is
	s.body.AppendUint16(value[0])
	
	// Subsequent values as zigzag-encoded deltas
	prev := value[0]
	for i := 1; i < len(value); i++ {
		delta := int64(value[i] - prev)
		appendVarintZigzag(&s.body, delta)
		prev = value[i]
	}
}

// AppendUint32Slice appends a uint32 slice to this slice builder
func (s *SliceBuilder) AppendUint32Slice(value []uint32) {
	s.wire = WireUint32
	s.body.AppendUint(uint(len(value)))
	for i := 0; i < len(value); i++ {
		s.body.AppendUint32(value[i])
	}
}

// AppendUint32SliceDelta appends a uint32 slice using delta encoding
func (s *SliceBuilder) AppendUint32SliceDelta(value []uint32) {
	s.wire = WireUint32 | WireDeltaFlag
	s.body.AppendUint(uint(len(value)))
	if len(value) == 0 {
		return
	}
	
	// First value as-is
	s.body.AppendUint32(value[0])
	
	// Subsequent values as zigzag-encoded deltas
	prev := value[0]
	for i := 1; i < len(value); i++ {
		delta := int64(value[i] - prev)
		appendVarintZigzag(&s.body, delta)
		prev = value[i]
	}
}

// AppendUint64Slice appends a uint64 slice to this slice builder
func (s *SliceBuilder) AppendUint64Slice(value []uint64) {
	s.wire = WireUint64
	s.body.AppendUint(uint(len(value)))
	for i := 0; i < len(value); i++ {
		s.body.AppendUint64(value[i])
	}
}

// AppendUint64SliceDelta appends a uint64 slice using delta encoding
func (s *SliceBuilder) AppendUint64SliceDelta(value []uint64) {
	s.wire = WireUint64 | WireDeltaFlag
	s.body.AppendUint(uint(len(value)))
	if len(value) == 0 {
		return
	}
	
	// First value as-is
	s.body.AppendUint64(value[0])
	
	// Subsequent values as zigzag-encoded deltas
	prev := value[0]
	for i := 1; i < len(value); i++ {
		delta := int64(value[i] - prev)
		appendVarintZigzag(&s.body, delta)
		prev = value[i]
	}
}

// AppendUintSlice appends a uint slice to this slice builder
func (s *SliceBuilder) AppendUintSlice(value []uint) {
	s.wire = WireUint
	s.body.AppendUint(uint(len(value)))
	for i := 0; i < len(value); i++ {
		s.body.AppendUint(value[i])
	}
}

// AppendUintSliceDelta appends a uint slice using delta encoding
func (s *SliceBuilder) AppendUintSliceDelta(value []uint) {
	s.wire = WireUint | WireDeltaFlag
	s.body.AppendUint(uint(len(value)))
	if len(value) == 0 {
		return
	}
	
	// First value as-is
	s.body.AppendUint(value[0])
	
	// Subsequent values as zigzag-encoded deltas
	prev := value[0]
	for i := 1; i < len(value); i++ {
		delta := int64(value[i] - prev)
		appendVarintZigzag(&s.body, delta)
		prev = value[i]
	}
}

// AppendInt8Slice appends a int8 slice to this slice builder
func (s *SliceBuilder) AppendInt8Slice(value []int8) {
	s.wire = WireInt8
	s.body.AppendUint(uint(len(value)))
	for i := 0; i < len(value); i++ {
		s.body.AppendInt8(value[i])
	}
}


// AppendInt16Slice appends a int16 slice to this slice builder
func (s *SliceBuilder) AppendInt16Slice(value []int16) {
	s.wire = WireInt16
	s.body.AppendUint(uint(len(value)))
	for i := 0; i < len(value); i++ {
		s.body.AppendInt16(value[i])
	}
}

// AppendInt16SliceDelta appends a int16 slice using delta encoding
func (s *SliceBuilder) AppendInt16SliceDelta(value []int16) {
	s.wire = WireInt16 | WireDeltaFlag
	s.body.AppendUint(uint(len(value)))
	if len(value) == 0 {
		return
	}
	
	// First value as-is
	s.body.AppendInt16(value[0])
	
	// Subsequent values as zigzag-encoded deltas
	prev := value[0]
	for i := 1; i < len(value); i++ {
		delta := int64(value[i] - prev)
		appendVarintZigzag(&s.body, delta)
		prev = value[i]
	}
}

// AppendInt32Slice appends a int32 slice to this slice builder
func (s *SliceBuilder) AppendInt32Slice(value []int32) {
	s.wire = WireInt32
	s.body.AppendUint(uint(len(value)))
	for i := 0; i < len(value); i++ {
		s.body.AppendInt32(value[i])
	}
}

// AppendInt32SliceDelta appends a int32 slice using delta encoding
func (s *SliceBuilder) AppendInt32SliceDelta(value []int32) {
	s.wire = WireInt32 | WireDeltaFlag
	s.body.AppendUint(uint(len(value)))
	if len(value) == 0 {
		return
	}
	
	// First value as-is
	s.body.AppendInt32(value[0])
	
	// Subsequent values as zigzag-encoded deltas
	prev := value[0]
	for i := 1; i < len(value); i++ {
		delta := int64(value[i] - prev)
		appendVarintZigzag(&s.body, delta)
		prev = value[i]
	}
}

// AppendInt64Slice appends a int64 slice to this slice builder
func (s *SliceBuilder) AppendInt64Slice(value []int64) {
	s.wire = WireInt64
	s.body.AppendUint(uint(len(value)))
	for i := 0; i < len(value); i++ {
		s.body.AppendInt64(value[i])
	}
}

// AppendInt64SliceDelta appends a int64 slice using delta encoding
func (s *SliceBuilder) AppendInt64SliceDelta(value []int64) {
	s.wire = WireInt64 | WireDeltaFlag
	s.body.AppendUint(uint(len(value)))
	if len(value) == 0 {
		return
	}
	
	// First value as-is
	s.body.AppendInt64(value[0])
	
	// Subsequent values as zigzag-encoded deltas
	prev := value[0]
	for i := 1; i < len(value); i++ {
		delta := value[i] - prev
		appendVarintZigzag(&s.body, delta)
		prev = value[i]
	}
}

// AppendFloat32Slice appends a float32 slice to this slice builder
func (s *SliceBuilder) AppendFloat32Slice(value []float32) {
	s.wire = WireFloat32
	s.body.AppendUint(uint(len(value)))
	for i := 0; i < len(value); i++ {
		s.body.AppendFloat32(value[i])
	}
}

// AppendFloat64Slice appends a float64 slice to this slice builder
func (s *SliceBuilder) AppendFloat64Slice(value []float64) {
	s.wire = WireFloat64
	s.body.AppendUint(uint(len(value)))
	for i := 0; i < len(value); i++ {
		s.body.AppendFloat64(value[i])
	}
}

// AppendTimeSlice appends a time slice to this slice builder
func (s *SliceBuilder) AppendTimeSlice(value []time.Time) {
	s.wire = WireTime
	s.body.AppendUint(uint(len(value)))
	for i := 0; i < len(value); i++ {
		s.body.AppendTime(value[i])
	}
}

// AppendBoolSlice appends a bool slice to this slice builder
func (s *SliceBuilder) AppendBoolSlice(value []bool) {
	s.wire = WireBool
	s.body.AppendUint(uint(len(value)))
	for i := 0; i < len(value); i++ {
		s.body.AppendBool(value[i])
	}
}

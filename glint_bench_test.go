package glint

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"testing"
	"time"
)

// This file contains all benchmark tests for the glint package.
// Benchmarks are separated from regular tests for clarity and performance testing.

// Helper functions for benchmarks
func switchCase(n int) int {
	switch n {
	case 0:
		return 10
	case 1:
		return 20
	case 2:
		return 30
	case 3:
		return 40
	default:
		return -1
	}
}

func ifElse(n int) int {
	switch n {
	case 0:
		return 10
	case 1:
		return 20
	case 2:
		return 30
	case 3:
		return 40
	default:
		return -1
	}
}

// Time series types for delta encoding benchmarks
type TimeSeriesData struct {
	Timestamps []int64 `glint:"timevals"`
	Values     []int   `glint:"values"`
}

type DeltaTimeSeriesData struct {
	Timestamps []int64 `glint:"timestamps,delta"`
	Values     []int   `glint:"values,delta"`
}

func generateTimeSeriesData(n int) ([]int64, []int) {
	timestamps := make([]int64, n)
	values := make([]int, n)

	// Generate timestamps with 1-second intervals
	base := int64(1700000000)
	for i := 0; i < n; i++ {
		timestamps[i] = base + int64(i)
	}

	// Generate values with small variations
	baseValue := 1000
	for i := 0; i < n; i++ {
		values[i] = baseValue + (i%10) - 5
	}

	return timestamps, values
}

// JSONContext and JSONTranscodeVisitor for BenchmarkWalker
type JSONContext struct {
	Index int
}

type JSONTranscodeVisitor struct {
	b        *Buffer
	ctxStack []JSONContext
}

func (v *JSONTranscodeVisitor) PushContext()                 { v.ctxStack = append(v.ctxStack, JSONContext{}) }
func (v *JSONTranscodeVisitor) PopContext()                  { v.ctxStack = v.ctxStack[:len(v.ctxStack)-1] }
func (v *JSONTranscodeVisitor) CurrentContext() *JSONContext { return &v.ctxStack[len(v.ctxStack)-1] }
func (v *JSONTranscodeVisitor) VisitFlags(flags byte) error  { return nil }
func (v *JSONTranscodeVisitor) VisitSchemaHash(hash []byte) error {
	v.PushContext()
	return nil
}

func (v *JSONTranscodeVisitor) VisitField(name string, wire WireType, body Reader) (Reader, error) {
	if v.CurrentContext().Index > 0 {
		v.b.Bytes = append(v.b.Bytes, ',')
	}
	v.CurrentContext().Index++

	v.b.Bytes = append(v.b.Bytes, '"')
	v.b.Bytes = append(v.b.Bytes, name...)
	v.b.Bytes = append(v.b.Bytes, "\":"...)

	if wire&WirePtrFlag > 0 {
		nullCheck := body.ReadByte()
		if nullCheck == 0 {
			v.b.Bytes = append(v.b.Bytes, []byte("null")...)
		}

		wire ^= WirePtrFlag
	}

	switch wire {
	case WireInt:
		v.b.Bytes = strconv.AppendInt(v.b.Bytes, int64(body.ReadInt()), 10)
	case WireInt8:
		v.b.Bytes = strconv.AppendInt(v.b.Bytes, int64(body.ReadInt8()), 10)
	case WireInt16:
		v.b.Bytes = strconv.AppendInt(v.b.Bytes, int64(body.ReadInt16()), 10)
	case WireInt32:
		v.b.Bytes = strconv.AppendInt(v.b.Bytes, int64(body.ReadInt32()), 10)
	case WireInt64:
		v.b.Bytes = strconv.AppendInt(v.b.Bytes, int64(body.ReadInt64()), 10)
	case WireUint:
		v.b.Bytes = strconv.AppendUint(v.b.Bytes, uint64(body.ReadUint()), 10)
	case WireUint8:
		v.b.Bytes = strconv.AppendUint(v.b.Bytes, uint64(body.ReadUint8()), 10)
	case WireUint16:
		v.b.Bytes = strconv.AppendUint(v.b.Bytes, uint64(body.ReadUint16()), 10)
	case WireUint32:
		v.b.Bytes = strconv.AppendUint(v.b.Bytes, uint64(body.ReadUint32()), 10)
	case WireUint64:
		v.b.Bytes = strconv.AppendUint(v.b.Bytes, body.ReadUint64(), 10)
	case WireFloat32:
		v.b.Bytes = strconv.AppendFloat(v.b.Bytes, float64(body.ReadFloat32()), 'f', -1, 32)
	case WireFloat64:
		v.b.Bytes = strconv.AppendFloat(v.b.Bytes, body.ReadFloat64(), 'f', -1, 64)
	case WireString:
		v.b.Bytes = append(v.b.Bytes, "\""...)
		v.b.Bytes = append(v.b.Bytes, body.ReadString()...)
		v.b.Bytes = append(v.b.Bytes, "\""...)

	case WireBytes:
		// fmt.Sprintf("%v", body.Read(body.ReadVarint()))
	case WireTime:
		v.b.Bytes = body.ReadTime().AppendFormat(v.b.Bytes, time.RFC3339Nano)
	case WireBool:
		v.b.Bytes = strconv.AppendBool(v.b.Bytes, body.ReadBool())
	}

	return body, nil
}

func (v *JSONTranscodeVisitor) VisitArrayStart(name string, wire WireType, length int) error {
	if v.CurrentContext().Index > 0 {
		v.b.Bytes = append(v.b.Bytes, ',')
	}
	v.CurrentContext().Index++

	v.PushContext()

	if name != "" {
		v.b.Bytes = append(v.b.Bytes, '"')
		v.b.Bytes = append(v.b.Bytes, name...)
		v.b.Bytes = append(v.b.Bytes, "\":["...)
		return nil
	}

	v.b.Bytes = append(v.b.Bytes, "["...)
	return nil
}

func (v *JSONTranscodeVisitor) VisitArrayEnd(name string) error {
	v.b.Bytes = append(v.b.Bytes, ']')
	v.PopContext()
	return nil
}

func (v *JSONTranscodeVisitor) VisitStructStart(name string) error {
	if v.CurrentContext().Index > 0 {
		v.b.Bytes = append(v.b.Bytes, ',')
	}
	v.CurrentContext().Index++

	v.PushContext()

	if name != "" {
		v.b.Bytes = append(v.b.Bytes, '"')
		v.b.Bytes = append(v.b.Bytes, name...)
		v.b.Bytes = append(v.b.Bytes, "\":{"...)
		return nil
	}

	v.b.Bytes = append(v.b.Bytes, '{')
	return nil
}

func (v *JSONTranscodeVisitor) VisitStructEnd(name string) error {
	v.PopContext()
	v.b.Bytes = append(v.b.Bytes, '}')
	return nil
}

func (v *JSONTranscodeVisitor) Bytes() []byte {
	return v.b.Bytes
}

// Benchmark Functions

func BenchmarkComprehensiveStructEncoding(b *testing.B) {
	// Prepare pointer values.
	pInt := 100
	pStr := "reference text"
	pTime := time.Date(2021, time.December, 31, 23, 59, 59, 0, time.UTC)
	pChild := Child{A: 10, B: "nested reference"}
	_ = pTime
	_ = pStr
	_ = pInt
	_ = pChild

	orig := Comprehensive{
		// Primitives
		Bool:    true,
		Int:     -123,
		Int8:    -12,
		Int16:   -1234,
		Int32:   -123456,
		Int64:   -1234567890,
		Uint:    123,
		Uint8:   12,
		Uint16:  1234,
		Uint32:  123456,
		Uint64:  1234567890,
		Float32: 3.14,
		Float64: 6.28,
		String:  "test data comprehensive",
		Bytes:   []byte{1, 2, 3, 4},
		Time:    time.Date(2020, time.May, 10, 12, 34, 56, 789, time.UTC),

		Child: Child{A: 200, B: "nested complete"},

		// Pointer fields
		PtrInt:    &pInt,
		PtrString: &pStr,
		PtrTime:   &pTime,
		PtrChild:  &pChild,

		// Slices of primitives
		BoolSlice:    []bool{true, false, true},
		IntSlice:     []int{-1, -2, -3},
		Int8Slice:    []int8{-1, -2, -3},
		Int16Slice:   []int16{-100, -200},
		Int32Slice:   []int32{-1000, -2000},
		Int64Slice:   []int64{-10000, -20000},
		UintSlice:    []uint{1, 2, 3},
		Uint8Slice:   []uint8{1, 2, 3},
		Uint16Slice:  []uint16{100, 200},
		Uint32Slice:  []uint32{1000, 2000},
		Uint64Slice:  []uint64{10000, 20000},
		Float32Slice: []float32{3.14, 2.71},
		Float64Slice: []float64{6.28, 3.1415},
		StringSlice:  []string{"x", "y", "z"},
		BytesSlice:   [][]byte{{1}, {2, 2}, {3, 3, 3}},
		TimeSlice: []time.Time{
			time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC),
		},

		ChildSlice: []Child{
			{A: 300, B: "nested array 1"},
			{A: 400, B: "nested array 2"},
		},

		MapStringInt:    map[string]int{"alpha": 1, "beta": 2},
		MapStringString: map[string]string{"key1": "value1", "key2": "value2"},

		ZigzagInt:      -999,
		ZigzagIntSlice: []int{-1, -2, 3, 4},

		SimpleEnd: true,
	}

	// Encode the original Comprehensive struct.
	enc := newEncoder(Comprehensive{})
	origBuf := NewBufferFromPool()
	enc.Marshal(&orig, origBuf)

	b.Run("all-encode", func(b *testing.B) {
		buf := NewBufferFromPool()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			enc.Marshal(&orig, buf)

		}
	})

	dec := newDecoder(Comprehensive{})

	b.Run("all-decode", func(b *testing.B) {
		var decoded Comprehensive
		for i := 0; i < b.N; i++ {
			dec.Unmarshal(origBuf.Bytes, &decoded)
		}
	})

	b.Run("gob-all-encode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			gob.NewEncoder(bytes.NewBuffer(nil)).Encode(orig)
		}
	})

	gobBuf := bytes.NewBuffer(nil)
	gob.NewEncoder(gobBuf).Encode(orig)

	b.Run("gob-all-decode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buf := bytes.NewBuffer(gobBuf.Bytes())
			dec := gob.NewDecoder(buf)
			var decoded Comprehensive
			dec.Decode(&decoded)
		}
	})

	encPartial := newEncoder(PartialComprehensive{})
	origPartial := PartialComprehensive{
		Bool:      orig.Bool,
		Int:       orig.Int,
		String:    orig.String,
		Time:      orig.Time,
		Child:     orig.Child,
		PtrInt:    orig.PtrInt,
		BoolSlice: orig.BoolSlice,
		// IntSlice: orig.IntSlice,
		Int8Slice: orig.Int8Slice,
		// StringSlice: orig.StringSlice,
		SimpleEnd: orig.SimpleEnd,
	}

	b.Run("partial-encode", func(b *testing.B) {

		buf := NewBufferFromPool()

		for i := 0; i < b.N; i++ {
			buf.Reset()
			encPartial.Marshal(&origPartial, buf)
		}
	})

	partDec := newDecoder(PartialComprehensive{})
	decoderContext := DecoderContext{
		InstructionCache: &DecodeInstructionLookup{},
		ID:               3,
	}

	b.Run("partial-decode", func(b *testing.B) {
		var decoded PartialComprehensive
		for i := 0; i < b.N; i++ {
			partDec.UnmarshalWithContext(origBuf.Bytes, &decoded, decoderContext)
		}
	})

	b.Run("gob-partial-encode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			gob.NewEncoder(bytes.NewBuffer(nil)).Encode(origPartial)
		}
	})

	gobPartialBuf := bytes.NewBuffer(nil)
	gob.NewEncoder(gobPartialBuf).Encode(origPartial)

	b.Run("gob-partial-decode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buf := bytes.NewBuffer(gobPartialBuf.Bytes())
			dec := gob.NewDecoder(buf)
			var decoded PartialComprehensive
			dec.Decode(&decoded)
		}
	})

}

func BenchmarkSwitchStatementPerformance(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = switchCase(i % 4)
	}
}

func BenchmarkIfElseStatementPerformance(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ifElse(i % 4)
	}
}

func BenchmarkGlintVsJSONVsGobEncoding(b *testing.B) {

	type CChild struct {
		Name string `glint:"name" json:"name"`
		Age  int    `glint:"age" json:"age"`
	}

	type Child struct {
		Name string `json:"f" glint:"f" nlv:"f"`
		Age  int    `json:"g" glint:"g" nlv:"g"`

		C CChild `glint:"item1" json:"child"`
	}

	var _ = &Child{}

	type Parent struct {
		Name  string `glint:"name" json:"name"`
		Age8  int8   `glint:"age8" json:"age8"`
		Age   int    `glint:"age" json:"age"`
		Age32 int32  `glint:"age32" json:"age32"`
		Age64 int64  `glint:"age64" json:"age64"`

		Children Child `json:"e" glint:"e" nlv:"e"`

		List []string `json:"list" glint:"lst1"`

		ChildList []Child `json:"childlist" glint:"items1"`
	}

	type Parent2 struct {
		Name  string `glint:"name" json:"name"`
		Age8  int8   `glint:"age8" json:"age8"`
		Age   int    `glint:"age" json:"age"`
		Age32 int32  `glint:"age32" json:"age32"`
		Age64 int64  `glint:"age64" json:"age64"`

		Children Child `json:"e" glint:"e" nlv:"e"`

		List []string `json:"list" glint:"lst1"`

		ChildList []Child `json:"childlist" glint:"items1"`
	}

	var _ = Parent{}

	v := Parent{
		Name: "sample test value",

		Age8:  127,
		Age:   21239871235,
		Age32: 31923987,
		Age64: 41263,
		Children: Child{
			Name: "nested name",
			Age:  13,
			C: CChild{
				Name: "nested name",
				Age:  13,
			},
		},

		List: []string{"first", "second", "third", "first", "second", "third", "first", "second", "third", "first", "second", "third", "first", "second", "third"},

		ChildList: []Child{
			{Name: "First", Age: 11},
			{Name: "Second", Age: 22},
		},
	}

	b.Run("table-encode", func(b *testing.B) {
		enc := newEncoder(Parent{})

		buf := &Buffer{}

		for i := 0; i < b.N; i++ {

			buf.Reset()

			enc.Marshal(&v, buf)
		}
	})
	// return

	b.Run("table-decode", func(b *testing.B) {
		enc := newEncoder(Parent{})

		buf := &Buffer{}
		enc.Marshal(&v, buf)

		dec := newDecoder(Parent2{}) // decode into a different struct to test skips

		var p Parent
		for i := 0; i < b.N; i++ {
			dec.Unmarshal(buf.Bytes, &p)
		}

	})
	// return

	b.Run("std-gob-encode", func(b *testing.B) {
		e := gob.NewEncoder(io.Discard)

		for i := 0; i < b.N; i++ {
			e.Encode(&v)
		}
	})

	b.Run("std-gob-decode", func(b *testing.B) {
		buf := &bytes.Buffer{}
		e := gob.NewEncoder(buf)
		e.Encode(&v)

		// var p Parent
		// d.Decode(&p)
		// fmt.Printf("hyuya: %v\n", p)

		for i := 0; i < b.N; i++ {
			d := gob.NewDecoder(bytes.NewReader(buf.Bytes()))
			var p Parent
			d.Decode(&p)
		}
	})

	b.Run("std-json-encode", func(b *testing.B) {

		for i := 0; i < b.N; i++ {
			json.Marshal(&v)
		}
	})

	b.Run("std-json-decode", func(b *testing.B) {

		by, _ := json.Marshal(&v)

		var p Parent

		for i := 0; i < b.N; i++ {

			json.Unmarshal(by, &p)
		}
	})
}

func BenchmarkLargeDatasetEncoding(b *testing.B) {

	type Child struct {
		Name string `json:"f" glint:"f" nlv:"f"`
		Age  int    `json:"g" glint:"g" nlv:"g"`
	}

	type Parent struct {
		Name  string `glint:"name" json:"name"`
		Age8  int8   `glint:"age8" json:"age8"`
		Age   int    `glint:"age" json:"age"`
		Age32 int32  `glint:"age32" json:"age32"`
		Age64 int64  `glint:"age64" json:"age64"`

		ChildList []Child `json:"childlist" glint:"items1"`
	}

	type Parent2 struct {
		Name  string `glint:"name" json:"name"`
		Age8  int8   `glint:"age8" json:"age8"`
		Age   int    `glint:"age" json:"age"`
		Age32 int32  `glint:"age32" json:"age32"`
		Age64 int64  `glint:"age64" json:"age64"`

		ChildList []Child `json:"childlist" glint:"items1"`
	}

	var _ = Parent{}

	v := Parent{
		Name: "sample test value",

		Age8:  127,
		Age:   21239871235,
		Age32: 31923987,
		Age64: 41263,

		ChildList: []Child{
			{Name: "First", Age: 11},
			{Name: "Second", Age: 22},
		},
	}

	/// buff out the rows
	for i := 0; i < 1000; i++ {
		v.ChildList = append(v.ChildList, Child{Name: "First", Age: 11})
		v.ChildList = append(v.ChildList, Child{Name: "Second", Age: 123})
	}

	//
	enc := newEncoder(Parent{})
	buf := &Buffer{}
	enc.Marshal(&v, buf)
	fmt.Println("table rows", len(v.ChildList), "doc len", len(buf.Bytes))
	//
	by, _ := json.Marshal(&v)
	fmt.Println("json rows", len(v.ChildList), "doc len", len(by))
	//
	buff := &bytes.Buffer{}
	genc := gob.NewEncoder(buff)
	genc.Encode(&v)
	fmt.Println("gob rows", len(v.ChildList), "doc len", buff.Len())

	b.Run("table-encode", func(b *testing.B) {
		enc := newEncoder(Parent{})

		buf := &Buffer{}

		for i := 0; i < b.N; i++ {

			buf.Reset()

			enc.Marshal(&v, buf)
		}
	})
	// return

	b.Run("table-decode", func(b *testing.B) {
		enc := newEncoder(Parent{})

		buf := &Buffer{}
		enc.Marshal(&v, buf)

		dec := newDecoder(Parent2{}) // decode into a different struct to test skips

		b.ResetTimer()

		var p Parent
		for i := 0; i < b.N; i++ {
			dec.Unmarshal(buf.Bytes, &p)
		}

	})
	// return

	b.Run("std-gob-encode", func(b *testing.B) {
		buff := &bytes.Buffer{}
		e := gob.NewEncoder(buff)

		for i := 0; i < b.N; i++ {
			buff.Reset()
			e.Encode(&v)
		}
	})

	b.Run("std-gob-decode", func(b *testing.B) {
		buf := &bytes.Buffer{}
		e := gob.NewEncoder(buf)

		e.Encode(&v)

		for i := 0; i < b.N; i++ {
			var p Parent
			d := gob.NewDecoder(bytes.NewReader(buf.Bytes()))
			d.Decode(&p)
		}
	})

	b.Run("std-json-encode", func(b *testing.B) {

		for i := 0; i < b.N; i++ {
			json.Marshal(&v)
		}
	})

	b.Run("std-json-decode", func(b *testing.B) {

		by, _ := json.Marshal(&v)
		b.ResetTimer()

		var p Parent

		for i := 0; i < b.N; i++ {

			json.Unmarshal(by, &p)
		}
	})
}

func BenchmarkDynamicValueSerialization(b *testing.B) {
	// Example benchmark for appending a string
	b.Run("String", func(b *testing.B) {
		buffer := &Buffer{}
		for i := 0; i < b.N; i++ {
			buffer.Reset()
			AppendDynamicValue("example string", buffer)
		}
	})

	b.Run("Bool", func(b *testing.B) {
		buffer := &Buffer{}
		for i := 0; i < b.N; i++ {
			buffer.Reset()
			AppendDynamicValue(true, buffer)
		}
	})

	b.Run("[]string", func(b *testing.B) {
		buffer := &Buffer{}
		for i := 0; i < b.N; i++ {
			buffer.Reset()
			AppendDynamicValue([]string{"x", "y", "z"}, buffer)
		}
	})

	b.Run("[]Time", func(b *testing.B) {
		buffer := &Buffer{}
		for i := 0; i < b.N; i++ {
			buffer.Reset()
			AppendDynamicValue([]time.Time{
				time.Date(1969, 4, 20, 4, 20, 4, 20, time.UTC),
				time.Date(1969, 4, 20, 4, 20, 4, 20, time.UTC),
			}, buffer)
		}
	})

	b.Run("[]Bool", func(b *testing.B) {
		buffer := &Buffer{}
		for i := 0; i < b.N; i++ {
			buffer.Reset()
			AppendDynamicValue([]bool{true, true}, buffer)
		}
	})

	// Additional benchmarks for other types
	// ...
}

// Benchmark for ReadDynamicValue
func BenchmarkDynamicValueDeserialization(b *testing.B) {
	// Example benchmark for reading a string
	b.Run("String", func(b *testing.B) {
		buffer := &Buffer{}
		AppendDynamicValue("test content", buffer)

		for i := 0; i < b.N; i++ {
			_ = ReadDynamicValue(buffer.Bytes)
		}
	})

	b.Run("[]Time", func(b *testing.B) {
		buffer := &Buffer{}
		AppendDynamicValue([]time.Time{
			time.Date(1969, 4, 20, 4, 20, 4, 20, time.UTC),
			time.Date(1969, 4, 20, 4, 20, 4, 20, time.UTC),
		}, buffer)

		for i := 0; i < b.N; i++ {
			_ = ReadDynamicValue(buffer.Bytes)
		}
	})

	b.Run("[]Bool", func(b *testing.B) {
		buffer := &Buffer{}
		AppendDynamicValue([]bool{true, true}, buffer)

		for i := 0; i < b.N; i++ {
			_ = ReadDynamicValue(buffer.Bytes)
		}
	})

	// Additional benchmarks for other types
	// ...
}

func BenchmarkDocumentWalkerVsUnmarshal(b *testing.B) {

	// Glint Document
	// ├─ Schema
	// │  ├─ String: name
	// │  ├─ Int: age
	// │  └─ Struct: wife
	// │    ├─ String: name
	// │    └─ Int: age
	// │  └─ []Struct: items1
	// │    ├─ String: name
	// │    └─ Int: age
	// │  └─ [][]Struct: nested slice
	// │    ├─ String: name
	// │    └─ Int: age

	// expect := []byte{0, 1, 91, 80, 87, 78, 14, 4, 110, 97, 109, 101, 2, 3, 97, 103, 101, 16, 4, 119, 105, 102, 101, 11, 14, 4, 110, 97, 109, 101, 2, 3, 97, 103, 101, 48, 8, 99, 104, 105, 108, 100, 114, 101, 110, 11, 14, 4, 110, 97, 109, 101, 2, 3, 97, 103, 101, 32, 12, 110, 101, 115, 116, 101, 100, 32, 115, 108, 105, 99, 101, 48, 11, 14, 4, 110, 97, 109, 101, 2, 3, 97, 103, 101, 4, 74, 111, 104, 110, 80, 5, 66, 101, 116, 116, 121, 78, 2, 6, 67, 97, 115, 112, 101, 114, 16, 8, 74, 117, 108, 105, 101, 116, 116, 101, 28, 2, 2, 6, 67, 97, 115, 112, 101, 114, 16, 8, 74, 117, 108, 105, 101, 116, 116, 101, 28, 2, 6, 67, 97, 115, 112, 101, 114, 16, 8, 74, 117, 108, 105, 101, 116, 116, 101, 28}
	// expect = []byte{0, 134, 190, 208, 128, 29, 14, 4, 110, 97, 109, 101, 2, 3, 97, 103, 101, 16, 4, 119, 105, 102, 101, 11, 14, 4, 110, 97, 109, 101, 2, 3, 97, 103, 101, 4, 74, 111, 104, 110, 80, 5, 66, 101, 116, 116, 121, 78}

	bu := DocumentBuilder{}
	bu.AppendString("name", "TestUser")
	bu.AppendInt("age", 40)

	wifeDoc := DocumentBuilder{}
	wifeDoc.AppendString("name", "Partner")
	wifeDoc.AppendInt("age", 39)
	bu.AppendNestedDocument("wife", &wifeDoc)

	children := []DocumentBuilder{}
	children = append(children, *(&DocumentBuilder{}).AppendString("name", "ChildA").AppendInt("age", 8))
	// children = append(children, *(&DocumentBuilder{}).AppendString("name", "TestChild2").AppendInt("age", 14))
	sb := SliceBuilder{}
	sb.AppendNestedDocumentSlice(children)
	bu.AppendSlice("items1", sb)

	expect := bu.Bytes()

	b.Run("Walker", func(b *testing.B) {
		v := &JSONTranscodeVisitor{b: &Buffer{}}
		for i := 0; i < b.N; i++ {
			v.b.Reset()
			w := NewWalker(expect)
			w.Walk(v)
		}

		// fmt.Println(string(v.b.Bytes))
	})

	b.Run("Unmarshal", func(b *testing.B) {

		type db struct {
			Name string `glint:"name"`
			Age  int    `glint:"age"`
			Wife struct {
				Name string `glint:"name"`
				Age  int    `glint:"age"`
			} `glint:"rel1"`
			Children []struct {
				Name string `glint:"name"`
				Age  int    `glint:"age"`
			} `glint:"items1"`
		}
		dec := newDecoder(db{})

		v := &db{}
		for i := 0; i < b.N; i++ {
			dec.Unmarshal(expect, v)
		}
	})

}

func BenchmarkDeltaEncodingPerformance(b *testing.B) {
	timestamps, values := generateTimeSeriesData(1000)

	b.Run("Standard", func(b *testing.B) {
		data := TimeSeriesData{
			Timestamps: timestamps,
			Values:     values,
		}
		encoder := NewEncoder[TimeSeriesData]()
		buf := NewBufferFromPool()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			encoder.Marshal(&data, buf)
		}
		b.ReportMetric(float64(len(buf.Bytes)), "bytes")
	})

	b.Run("Delta", func(b *testing.B) {
		data := DeltaTimeSeriesData{
			Timestamps: timestamps,
			Values:     values,
		}
		encoder := NewEncoder[DeltaTimeSeriesData]()
		buf := NewBufferFromPool()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			encoder.Marshal(&data, buf)
		}
		b.ReportMetric(float64(len(buf.Bytes)), "bytes")
	})
}

func BenchmarkDeltaDecodingPerformance(b *testing.B) {
	timestamps, values := generateTimeSeriesData(1000)

	b.Run("Standard", func(b *testing.B) {
		data := TimeSeriesData{
			Timestamps: timestamps,
			Values:     values,
		}
		encoder := NewEncoder[TimeSeriesData]()
		decoder := NewDecoder[TimeSeriesData]()
		buf := NewBufferFromPool()
		encoder.Marshal(&data, buf)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var decoded TimeSeriesData
			decoder.Unmarshal(buf.Bytes, &decoded)
		}
	})

	b.Run("Delta", func(b *testing.B) {
		data := DeltaTimeSeriesData{
			Timestamps: timestamps,
			Values:     values,
		}
		encoder := NewEncoder[DeltaTimeSeriesData]()
		decoder := NewDecoder[DeltaTimeSeriesData]()
		buf := NewBufferFromPool()
		encoder.Marshal(&data, buf)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var decoded DeltaTimeSeriesData
			decoder.Unmarshal(buf.Bytes, &decoded)
		}
	})
}
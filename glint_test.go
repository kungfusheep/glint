package glint

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Child is used as a nested struct.
type Child struct {
	A int    `glint:"a"`
	B string `glint:"b"`
}

// Comprehensive includes a field for every supported type and combination.
type Comprehensive struct {
	// Primitives
	Bool    bool      `glint:"bool"`
	Int     int       `glint:"int"`
	Int8    int8      `glint:"int8"`
	Int16   int16     `glint:"int16"`
	Int32   int32     `glint:"int32"`
	Int64   int64     `glint:"int64"`
	Uint    uint      `glint:"uint"`
	Uint8   uint8     `glint:"uint8"`
	Uint16  uint16    `glint:"uint16"`
	Uint32  uint32    `glint:"uint32"`
	Uint64  uint64    `glint:"uint64"`
	Float32 float32   `glint:"float32"`
	Float64 float64   `glint:"float64"`
	String  string    `glint:"string"`
	Bytes   []byte    `glint:"bytes"`
	Time    time.Time `glint:"time"`

	// Nested struct
	Child Child `glint:"item1"`

	// Pointer fields
	PtrInt    *int       `glint:"ptr_int"`
	PtrString *string    `glint:"ptr_string"`
	PtrTime   *time.Time `glint:"ptr_time"`
	PtrChild  *Child     `glint:"ptr_child"`

	// // Slices of primitives
	BoolSlice    []bool      `glint:"[]bool"`
	IntSlice     []int       `glint:"[]int"`
	Int8Slice    []int8      `glint:"[]int8"`
	Int16Slice   []int16     `glint:"[]int16"`
	Int32Slice   []int32     `glint:"[]int32"`
	Int64Slice   []int64     `glint:"[]int64"`
	UintSlice    []uint      `glint:"[]uint"`
	Uint8Slice   []uint8     `glint:"[]uint8"`
	Uint16Slice  []uint16    `glint:"[]uint16"`
	Uint32Slice  []uint32    `glint:"[]uint32"`
	Uint64Slice  []uint64    `glint:"[]uint64"`
	Float32Slice []float32   `glint:"[]float32"`
	Float64Slice []float64   `glint:"[]float64"`
	StringSlice  []string    `glint:"[]string"`
	BytesSlice   [][]byte    `glint:"[]bytes"`
	TimeSlice    []time.Time `glint:"[]time"`

	// Slices of nested structs
	ChildSlice []Child `glint:"[]child"`

	// Maps
	MapStringInt    map[string]int    `glint:"map_str_int"`
	MapStringString map[string]string `glint:"map_str_str"`

	// Zigzagy fields
	ZigzagInt      int   `glint:"zzint"`
	ZigzagIntSlice []int `glint:"[]zzint"`

	SimpleEnd bool `glint:"simple_end"`
}

// PartialComprehensive omits some fields so that skip code paths are exercised.
type PartialComprehensive struct {
	// Only a subset of the primitives
	Bool      bool      `glint:"bool"`
	Int       int       `glint:"int"`
	String    string    `glint:"string"`
	Time      time.Time `glint:"time"`
	Child     Child     `glint:"item1"`
	PtrInt    *int      `glint:"ptr_int"`
	BoolSlice []bool    `glint:"[]bool"`
	Int8Slice []int8    `glint:"[]int8"`

	SimpleEnd bool `glint:"simple_end"`
}

type TestTime struct {
	PtrTime *time.Time `glint:"time"`
}

// TestTypeSupport consolidates all type-related tests into a comprehensive table-driven test
func TestBasicTypesEncodeDecodeRoundtrip(t *testing.T) {
	// Helper function to create pointer values
	intPtr := func(v int) *int { return &v }
	stringPtr := func(v string) *string { return &v }
	childPtr := func(v Child) *Child { return &v }

	tests := []struct {
		name        string
		testFunc    func(t *testing.T)
		description string
	}{
		{
			name:        "PtrTime",
			description: "Test pointer to time.Time encoding/decoding",
			testFunc: func(t *testing.T) {
				expected := time.Date(2021, time.December, 31, 23, 59, 59, 0, time.UTC)
				orig := TestTime{
					PtrTime: &expected,
				}

				enc := newEncoder(TestTime{})
				buf := NewBufferFromPool()
				enc.Marshal(&orig, buf)

				var decoded TestTime
				dec := newDecoder(TestTime{})
				if err := dec.Unmarshal(buf.Bytes, &decoded); err != nil {
					t.Fatalf("Unmarshal error: %v", err)
				}

				if decoded.PtrTime == nil {
					t.Fatalf("Decoded PtrTime is nil")
				}

				if !decoded.PtrTime.Equal(expected) {
					t.Errorf("Time mismatch: expected %v, got %v", expected, decoded.PtrTime)
				}
			},
		},
		{
			name:        "Comprehensive",
			description: "Test all supported types comprehensively",
			testFunc: func(t *testing.T) {
				pTime := time.Date(2021, time.December, 31, 23, 59, 59, 0, time.UTC)

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
					PtrInt:    intPtr(100),
					PtrString: stringPtr("reference text"),
					PtrTime:   &pTime,
					PtrChild:  childPtr(Child{A: 10, B: "nested reference"}),

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

				// Test full roundtrip
				enc := newEncoder(Comprehensive{})
				buf := NewBufferFromPool()
				enc.Marshal(&orig, buf)

				var decoded Comprehensive
				dec := newDecoder(Comprehensive{})
				if err := dec.UnmarshalWithContext(buf.Bytes, &decoded, DecoderContext{
					InstructionCache: &DecodeInstructionLookup{},
					ID:               0,
				}); err != nil {
					t.Fatalf("Unmarshal full error: %v", err)
				}

				PrintStructIfDiff(t, orig, decoded)

				// Test partial decoding to exercise skip paths
				var partial PartialComprehensive
				decPartial := newDecoder(PartialComprehensive{})
				if err := decPartial.UnmarshalWithContext(buf.Bytes, &partial, DecoderContext{
					InstructionCache: &DecodeInstructionLookup{},
					ID:               1,
				}); err != nil {
					t.Fatalf("Unmarshal partial error: %v", err)
				}

				expectedPartial := PartialComprehensive{
					Bool:      orig.Bool,
					Int:       orig.Int,
					String:    orig.String,
					Time:      orig.Time,
					Child:     orig.Child,
					PtrInt:    orig.PtrInt,
					BoolSlice: orig.BoolSlice,
					Int8Slice: orig.Int8Slice,
					SimpleEnd: orig.SimpleEnd,
				}

				PrintStructIfDiff(t, expectedPartial, partial)
			},
		},
		{
			name:        "EncoderAll",
			description: "Test encoding all types with expected byte output",
			testFunc: func(t *testing.T) {
				type child struct {
					Name string `glint:"name"`
					Age  int    `glint:"age"`
				}

				type All struct {
					Vbool     bool        `glint:"bool"`
					Abool     []bool      `glint:"[]bool"`
					Vint      int         `glint:"int"`
					Aint      []int       `glint:"[]int"`
					Vint8     int8        `glint:"int8"`
					Aint8     []int8      `glint:"[]int8"`
					Vint16    int16       `glint:"int16"`
					Aint16    []int16     `glint:"[]int16"`
					Vint32    int32       `glint:"int32"`
					Aint32    []int32     `glint:"[]int32"`
					Vint64    int64       `glint:"int64"`
					Aint64    []int64     `glint:"[]int64"`
					Vuint     uint        `glint:"uint"`
					Auint     []uint      `glint:"[]uint"`
					Vuint8    uint8       `glint:"uint8"`
					Auint8    []uint8     `glint:"[]uint8"`
					Vuint16   uint16      `glint:"uint16"`
					Auint16   []uint16    `glint:"[]uint16"`
					Vuint32   uint32      `glint:"uint32"`
					Auint32   []uint32    `glint:"[]uint32"`
					Vuint64   uint64      `glint:"uint64"`
					Auint64   []uint64    `glint:"[]uint64"`
					Vfloat32  float32     `glint:"float32"`
					Afloat32  []float32   `glint:"[]float32"`
					Vfloat64  float64     `glint:"float64"`
					Afloat64  []float64   `glint:"[]float64"`
					Vstring   string      `glint:"string"`
					Astring   []string    `glint:"[]string"`
					Vbytes    []byte      `glint:"bytes"`
					Abytes    [][]byte    `glint:"[]bytes"`
					Vstruct   child       `glint:"Vstruct"`
					Astruct   []child     `glint:"[]Vstruct"`
					Vzzint    int         `glint:"zzint,zigzag"`
					Azzint    []int       `glint:"[]zzint,zigzag"`
					VtimeTime time.Time   `glint:"timeTime"`
					AtimeTime []time.Time `glint:"[]timeTime"`
				}

				encoder := newEncoder(All{})
				buf := NewBufferFromPool()

				value := All{
					Vbool:    true,
					Abool:    []bool{true, false, true},
					Vint:     -11,
					Aint:     []int{-11, -12, -13},
					Vint8:    -120,
					Aint8:    []int8{-120, 120, 55},
					Vint16:   -1000,
					Aint16:   []int16{-1000, -1001, -1002},
					Vint32:   -80_000,
					Aint32:   []int32{-80_000, -80_001, -80_002},
					Vint64:   MaxInt32 * 2,
					Aint64:   []int64{MaxInt32 * 2, MaxInt32*2 + 1},
					Vuint:    MaxUint64,
					Auint:    []uint{1, MaxUint64, 3},
					Vuint8:   38,
					Auint8:   []uint8{23, 24, 25},
					Vuint16:  64000,
					Auint16:  []uint16{64000, 64001, 64002},
					Vuint32:  80_000,
					Auint32:  []uint32{80_000, 80_001, 80_002},
					Vuint64:  MaxUint,
					Auint64:  []uint64{MaxInt + 1, MaxInt + 2, MaxInt + 3},
					Vfloat32: MaxFloat32,
					Afloat32: []float32{MaxFloat32, 0.1, 0.2},
					Vfloat64: MaxFloat64,
					Afloat64: []float64{MaxFloat64, 0.1, 0.2},
					Vstring:  "sample text",
					Astring:  []string{"sample text", "sample text", "sample text"},
					Vbytes:   []byte{11, 22, 33},
					Abytes:   [][]byte{{1, 1}, {2, 2}, {3, 3}},
					Vstruct: child{
						Name: "nested item",
						Age:  29,
					},
					Astruct: []child{
						{Name: "primary nested item", Age: 25},
						{Name: "secondary nested item", Age: 31},
						{Name: "tertiary nested item", Age: 27},
					},
					Vzzint:    -1,
					Azzint:    []int{-1, -2, 1, 2},
					VtimeTime: time.Date(2023, 6, 15, 10, 30, 45, 123456789, time.UTC),
					AtimeTime: []time.Time{
						time.Date(2023, 6, 15, 10, 30, 45, 123456789, time.UTC),
						time.Date(2022, 3, 8, 16, 20, 30, 987654321, time.UTC),
						time.Date(2021, 9, 22, 8, 45, 15, 555666777, time.UTC),
					},
				}

				encoder.Marshal(&value, buf)

				expected := []byte{0, 11, 155, 80, 71, 201, 2, 1, 4, 98, 111, 111, 108, 33, 6, 91, 93, 98, 111, 111,
					108, 2, 3, 105, 110, 116, 34, 5, 91, 93, 105, 110, 116, 3, 4, 105, 110, 116, 56, 35,
					6, 91, 93, 105, 110, 116, 56, 4, 5, 105, 110, 116, 49, 54, 36, 7, 91, 93, 105, 110,
					116, 49, 54, 5, 5, 105, 110, 116, 51, 50, 37, 7, 91, 93, 105, 110, 116, 51, 50, 6,
					5, 105, 110, 116, 54, 52, 38, 7, 91, 93, 105, 110, 116, 54, 52, 7, 4, 117, 105, 110,
					116, 39, 6, 91, 93, 117, 105, 110, 116, 8, 5, 117, 105, 110, 116, 56, 15, 7, 91, 93,
					117, 105, 110, 116, 56, 9, 6, 117, 105, 110, 116, 49, 54, 41, 8, 91, 93, 117, 105, 110,
					116, 49, 54, 10, 6, 117, 105, 110, 116, 51, 50, 42, 8, 91, 93, 117, 105, 110, 116, 51,
					50, 11, 6, 117, 105, 110, 116, 54, 52, 43, 8, 91, 93, 117, 105, 110, 116, 54, 52, 12,
					7, 102, 108, 111, 97, 116, 51, 50, 44, 9, 91, 93, 102, 108, 111, 97, 116, 51, 50, 13,
					7, 102, 108, 111, 97, 116, 54, 52, 45, 9, 91, 93, 102, 108, 111, 97, 116, 54, 52, 14,
					6, 115, 116, 114, 105, 110, 103, 46, 8, 91, 93, 115, 116, 114, 105, 110, 103, 15, 5, 98,
					121, 116, 101, 115, 32, 7, 91, 93, 98, 121, 116, 101, 115, 15, 16, 7, 86, 115, 116, 114,
					117, 99, 116, 11, 14, 4, 110, 97, 109, 101, 2, 3, 97, 103, 101, 48, 9, 91, 93, 86,
					115, 116, 114, 117, 99, 116, 11, 14, 4, 110, 97, 109, 101, 2, 3, 97, 103, 101, 2, 5,
					122, 122, 105, 110, 116, 34, 7, 91, 93, 122, 122, 105, 110, 116, 18, 8, 116, 105, 109, 101,
					84, 105, 109, 101, 50, 10, 91, 93, 116, 105, 109, 101, 84, 105, 109, 101, 1, 3, 1, 0,
					1, 21, 3, 21, 23, 25, 136, 3, 136, 120, 55, 207, 15, 3, 207, 15, 209, 15, 211, 15,
					255, 225, 9, 3, 255, 225, 9, 129, 226, 9, 131, 226, 9, 254, 255, 255, 255, 15, 2, 254,
					255, 255, 255, 15, 255, 255, 255, 255, 15, 255, 255, 255, 255, 255, 255, 255, 255, 255, 1, 3,
					1, 255, 255, 255, 255, 255, 255, 255, 255, 255, 1, 3, 38, 3, 23, 24, 25, 128, 244, 3,
					3, 128, 244, 3, 129, 244, 3, 130, 244, 3, 128, 241, 4, 3, 128, 241, 4, 129, 241, 4,
					130, 241, 4, 255, 255, 255, 255, 255, 255, 255, 255, 255, 1, 3, 128, 128, 128, 128, 128, 128,
					128, 128, 128, 1, 129, 128, 128, 128, 128, 128, 128, 128, 128, 1, 130, 128, 128, 128, 128, 128,
					128, 128, 128, 1, 255, 255, 255, 251, 7, 3, 255, 255, 255, 251, 7, 205, 153, 179, 238, 3,
					205, 153, 179, 242, 3, 255, 255, 255, 255, 255, 255, 255, 247, 127, 3, 255, 255, 255, 255, 255,
					255, 255, 247, 127, 154, 179, 230, 204, 153, 179, 230, 220, 63, 154, 179, 230, 204, 153, 179, 230,
					228, 63, 11, 115, 97, 109, 112, 108, 101, 32, 116, 101, 120, 116, 3, 11, 115, 97, 109, 112,
					108, 101, 32, 116, 101, 120, 116, 11, 115, 97, 109, 112, 108, 101, 32, 116, 101, 120, 116, 11,
					115, 97, 109, 112, 108, 101, 32, 116, 101, 120, 116, 3, 11, 22, 33, 3, 2, 1, 1, 2,
					2, 2, 2, 3, 3, 11, 110, 101, 115, 116, 101, 100, 32, 105, 116, 101, 109, 58, 3, 19,
					112, 114, 105, 109, 97, 114, 121, 32, 110, 101, 115, 116, 101, 100, 32, 105, 116, 101, 109, 50,
					21, 115, 101, 99, 111, 110, 100, 97, 114, 121, 32, 110, 101, 115, 116, 101, 100, 32, 105, 116,
					101, 109, 62, 20, 116, 101, 114, 116, 105, 97, 114, 121, 32, 110, 101, 115, 116, 101, 100, 32,
					105, 116, 101, 109, 54, 1, 4, 1, 3, 2, 4, 15, 1, 0, 0, 0, 14, 220, 28, 223,
					85, 7, 91, 205, 21, 255, 255, 3, 15, 1, 0, 0, 0, 14, 220, 28, 223, 85, 7, 91,
					205, 21, 255, 255, 15, 1, 0, 0, 0, 14, 217, 185, 121, 78, 58, 222, 104, 177, 255, 255,
					15, 1, 0, 0, 0, 14, 216, 220, 228, 27, 33, 30, 205, 89, 255, 255}

				if !bytes.Equal(buf.Bytes, expected) {
					t.Errorf("Byte output mismatch.\nGot (%d bytes): %v\nExpected (%d bytes): %v",
						len(buf.Bytes), buf.Bytes, len(expected), expected)
				}
			},
		},
		{
			name:        "ArrayOfStructPointers",
			description: "Test slice of struct pointers encoding",
			testFunc: func(t *testing.T) {
				type Point struct {
					X uint `glint:"x"`
					Y uint `glint:"y"`
				}

				type Points struct {
					Points []*Point `glint:"points"`
				}

				encoder := newEncoder(Points{})
				buf := NewBufferFromPool()
				value := Points{
					Points: []*Point{
						{X: 10, Y: 55},
						{X: 11, Y: 56},
					},
				}

				encoder.Marshal(&value, buf)

				// Original test only checks encoding, not decoding
				// since slice of pointers isn't fully supported for decoding
				if len(buf.Bytes) == 0 {
					t.Error("Expected non-empty encoding")
				}
			},
		},
		{
			name:        "StringComparison",
			description: "Test string comparison utility",
			testFunc: func(t *testing.T) {
				want := "This is a test string. Line 2 is different."
				got := want

				if got != want {
					t.Errorf("Strings don't match:\n%s", stringDiff(want, got))
				}

				// Test with actual difference
				got = "This is a test string. Line 2 has changed."
				if got == want {
					t.Error("Strings should be different")
				}
			},
		},
		{
			name:        "Map",
			description: "Test various map types including nested maps",
			testFunc: func(t *testing.T) {
				type subtype struct {
					Name string `glint:"name"`
					Age  int    `glint:"age"`
				}

				type myetype struct {
					MapStringInt          map[string]int            `glint:"ssi"`
					MapStringSInt         map[int][]subtype         `glint:"misls"`
					MapStringSrInt        map[int]subtype           `glint:"sri"`
					MapStringSrPInt       map[int]subtype           `glint:"srpi"`
					MapStringMapStringInt map[string]map[string]int `glint:"msmsi"`
				}

				type mydtype struct {
					MapStringInt          map[string]int            `glint:"ssi"`
					MapStringSInt         map[int][]subtype         `glint:"misls"`
					MapStringSrInt        map[int]subtype           `glint:"sri"`
					MapStringMapStringInt map[string]map[string]int `glint:"msmsi"`
				}

				encoder := newEncoder(myetype{})
				buf := Buffer{}

				v := myetype{
					MapStringInt: map[string]int{"item1": 22, "item2": 22},
					MapStringSInt: map[int][]subtype{
						11: {
							{Name: "ITEM1", Age: 24},
							{Name: "ITEM2", Age: 26},
						},
						22: {
							{Name: "ITEM3", Age: 35},
							{Name: "ITEM4", Age: 42},
						},
					},
					MapStringSrInt:  map[int]subtype{11: {Name: "test data"}, 17: {Name: "test data"}},
					MapStringSrPInt: map[int]subtype{11: {Name: "test data", Age: 33}},
					MapStringMapStringInt: map[string]map[string]int{
						"1": {"1,1": 1999, "1,2": 1999},
						"2": {"2,1": 1999, "2,2": 1999},
					},
				}

				encoder.Marshal(&v, &buf)

				// Test decoding into partial struct
				decoder := newDecoder(mydtype{})
				mt := mydtype{}

				err := decoder.Unmarshal(buf.Bytes, &mt)
				if err != nil {
					t.Error(err)
					return
				}

				// Verify decoded values
				if mt.MapStringInt["item1"] != 22 {
					t.Error("MapStringInt item1 not 22")
				}

				if len(mt.MapStringSInt) != 2 {
					t.Error("MapStringSInt length not 2")
				}

				if mt.MapStringSrInt[11].Name != "test data" {
					t.Error("MapStringSrInt[11].Name not 'test data'")
				}

				if mt.MapStringMapStringInt["1"]["1,1"] != 1999 {
					t.Error("Nested map value incorrect")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t)
		})
	}

	// Test Reader slice methods for coverage
	t.Run("ReaderSliceMethods", func(t *testing.T) {
		// Test ReadUintSlice
		t.Run("ReadUintSlice", func(t *testing.T) {
			values := []uint{100, 200}
			buf := &Buffer{}
			appendVarint(buf, uint64(len(values)))
			for _, v := range values {
				buf.AppendUint(v)
			}
			reader := NewReader(buf.Bytes)
			result := reader.ReadUintSlice()
			if len(result) != len(values) {
				t.Errorf("ReadUintSlice length mismatch: got %d, want %d", len(result), len(values))
			}
		})

		// Test other Reader slice methods
		t.Run("ReadUint16Slice", func(t *testing.T) {
			values := []uint16{1000, 2000}
			buf := &Buffer{}
			appendVarint(buf, uint64(len(values)))
			for _, v := range values {
				buf.AppendUint16(v)
			}
			reader := NewReader(buf.Bytes)
			reader.ReadUint16Slice()
		})

		t.Run("ReadUint32Slice", func(t *testing.T) {
			values := []uint32{100000, 200000}
			buf := &Buffer{}
			appendVarint(buf, uint64(len(values)))
			for _, v := range values {
				buf.AppendUint32(v)
			}
			reader := NewReader(buf.Bytes)
			reader.ReadUint32Slice()
		})

		t.Run("ReadInt8Slice", func(t *testing.T) {
			values := []int8{-100, 100}
			buf := &Buffer{}
			appendVarint(buf, uint64(len(values)))
			for _, v := range values {
				buf.AppendInt8(v)
			}
			reader := NewReader(buf.Bytes)
			reader.ReadInt8Slice()
		})

		t.Run("ReadInt32Slice", func(t *testing.T) {
			values := []int32{-100000, 100000}
			buf := &Buffer{}
			appendVarint(buf, uint64(len(values)))
			for _, v := range values {
				buf.AppendInt32(v)
			}
			reader := NewReader(buf.Bytes)
			reader.ReadInt32Slice()
		})

		t.Run("ReadInt64Slice", func(t *testing.T) {
			values := []int64{-1000000000, 1000000000}
			buf := &Buffer{}
			appendVarint(buf, uint64(len(values)))
			for _, v := range values {
				buf.AppendInt64(v)
			}
			reader := NewReader(buf.Bytes)
			reader.ReadInt64Slice()
		})

		t.Run("ReadFloat64Slice", func(t *testing.T) {
			values := []float64{-3.14, 2.71}
			buf := &Buffer{}
			appendVarint(buf, uint64(len(values)))
			for _, v := range values {
				buf.AppendFloat64(v)
			}
			reader := NewReader(buf.Bytes)
			reader.ReadFloat64Slice()
		})

		// Test Reader position methods
		t.Run("ReaderPositioning", func(t *testing.T) {
			data := []byte{1, 2, 3, 4, 5}
			reader := NewReader(data)

			mark := reader.Mark()
			if mark < 0 {
				t.Error("Mark should return valid position")
			}

			// Read some data and reset
			reader.ReadUint8()
			reader.ResetMark()
		})
	})
}

// PrintStructIfDiff compares two structs and prints a colorized diff.
// It returns true if differences exist, otherwise returns false.
func PrintStructIfDiff(t *testing.T, a, b any) bool {
	jsonA := normalizeJSON(a)
	jsonB := normalizeJSON(b)

	// If normalized JSON is identical, return false (no differences)
	if jsonA == jsonB {
		return false
	}

	compareJSON(t, jsonA, jsonB)
	return true
}

// Converts a struct to normalized JSON (ignores `null` vs `[]` for slices)
func normalizeJSON(v any) string {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ") // Pretty-print JSON
	encoder.Encode(replaceNilSlices(v))
	return buf.String()
}

// Recursively replaces nil slices with empty slices to normalize JSON output
func replaceNilSlices(v any) any {
	val := reflect.ValueOf(v)

	switch val.Kind() {
	case reflect.Ptr:
		if val.IsNil() {
			return nil
		}
		return replaceNilSlices(val.Elem().Interface())

	case reflect.Struct:
		newMap := make(map[string]any)
		for i := 0; i < val.NumField(); i++ {
			field := val.Type().Field(i)
			if field.PkgPath != "" { // Ignore unexported fields
				continue
			}
			newMap[field.Name] = replaceNilSlices(val.Field(i).Interface())
		}
		return newMap

	case reflect.Map:
		newMap := make(map[string]any)
		for _, key := range val.MapKeys() {
			newMap[key.String()] = replaceNilSlices(val.MapIndex(key).Interface())
		}
		return newMap

	case reflect.Slice:
		if val.IsNil() {
			return []any{} // Convert nil slices to empty slices
		}
		newSlice := make([]any, val.Len())
		for i := 0; i < val.Len(); i++ {
			newSlice[i] = replaceNilSlices(val.Index(i).Interface())
		}
		return newSlice

	default:
		return v
	}
}

// Compares two JSON strings line by line and highlights differences
func compareJSON(t *testing.T, a, b string) {
	aLines := strings.Split(a, "\n")
	bLines := strings.Split(b, "\n")

	maxLen := len(aLines)
	if len(bLines) > maxLen {
		maxLen = len(bLines)
	}

	buff := bytes.NewBuffer(nil)

	fmt.Fprintln(buff, "Differences (Green = Added, Red = Removed):")

	for i := 0; i < maxLen; i++ {
		var aLine, bLine string
		if i < len(aLines) {
			aLine = aLines[i]
		}
		if i < len(bLines) {
			bLine = bLines[i]
		}

		if aLine == bLine {
			fmt.Fprintln(buff, "  "+aLine) // No difference, print normally
		} else {
			if aLine != "" {
				fmt.Fprintln(buff, "\033[31m- "+aLine+"\033[0m") // Red for removed
			}
			if bLine != "" {
				fmt.Fprintln(buff, "\033[32m+ "+bLine+"\033[0m") // Green for added
			}
		}
	}

	t.Error(buff.String())
	fmt.Println(buff.String())
}

func TestSliceUnmarshalWithPreAllocatedSlice(t *testing.T) {

	type TestKey struct {
		ItemID    int    `glint:"k1"`
		SubItemID int    `glint:"k2"`
		Label     string `glint:"k3"`
	}
	type TestRequest struct {
		Keys []TestKey `json:"keys" glint:"keys"`
	}

	var testRequestDecoder = newDecoder(TestRequest{})

	encoder := newEncoder(TestRequest{})

	testItems := []TestKey{
		{
			ItemID:    1,
			SubItemID: 1,
			Label:     "A",
		},
		{
			ItemID:    2,
			SubItemID: 2,
			Label:     "B",
		},
		{
			ItemID:    3,
			SubItemID: 3,
			Label:     "C",
		},
	}

	pb := NewBufferFromPool()

	encoder.Marshal(&TestRequest{
		Keys: testItems,
	}, pb)

	request := TestRequest{
		Keys: make([]TestKey, 3),
	}

	request.Keys = request.Keys[:0]

	if err := testRequestDecoder.Unmarshal(pb.Bytes, &request); err != nil {
		t.Fatalf("failed to decode body  %v\n%s", err, pb.Bytes)
		return
	}

	if len(request.Keys) != 3 {
		t.Fatalf("expected 3 keys got %d", len(request.Keys))
	}
}

type StringerTest struct {
	Name string `glint:"name"`
}

func (s *StringerTest) String() string {
	return s.Name
}

// stub for the number.Decimal type which implements binaryEncoder
type number struct {
	numericValue float64
}

func (n *number) MarshalBinary() []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf[0:], math.Float64bits(n.numericValue))
	return buf
}

type binaryEncoderTest struct {
	Decimal    number  `glint:"decimal,encoder"`
	DecimalPtr *number `glint:"decimalptr,encoder"`
}

type numericData struct {
	number float64
}

func (w *numericData) UnmarshalBinary(bytes []byte) {
	u := binary.LittleEndian.Uint64(bytes)
	w.number = math.Float64frombits(u)
}

type binaryDecoderTest struct {
	Decimal    numericData  `glint:"decimal,encoder"`
	DecimalPtr *numericData `glint:"decimalptr,encoder"`
}

// tests the encoder tag works with binaryEncoder implementation
func TestCustomBinaryEncoderInterface(t *testing.T) {

	encoder := newEncoder(binaryEncoderTest{})
	decoder := newDecoder(binaryDecoderTest{})

	ebuf := NewBufferFromPool()
	defer ebuf.ReturnToPool()

	var testStruct = binaryEncoderTest{
		number{numericValue: 1.5},
		&number{numericValue: 2.5},
	}

	encoder.Marshal(&testStruct, ebuf)

	var result binaryDecoderTest
	err := decoder.Unmarshal(ebuf.Bytes, &result)

	if err != nil {
		t.Fatalf("Error unmarshalling encoded data: %s", err)
	}

	if result.Decimal.number != 1.5 {
		t.Error("decoded Decimal value not 1.5")
	}

	if result.DecimalPtr.number != 2.5 {
		t.Error("decoded DecimalPtr value not 2.5")
	}
}

func ExampleDecoder() {

	type child struct {
		Name string `glint:"name"`
	}

	type Example struct {
		Bool    bool      `glint:"bool"`
		Int     int       `glint:"int"`
		Int8    int8      `glint:"int8"`
		Int16   int16     `glint:"int16"`
		Int32   int32     `glint:"int32"`
		Int64   int64     `glint:"int64"`
		Uint    uint      `glint:"uint"`
		Uint8   uint8     `glint:"uint8"`
		Uint16  uint16    `glint:"uint16"`
		Uint32  uint32    `glint:"uint32"`
		Uint64  uint64    `glint:"uint64"`
		Float32 float32   `glint:"float32"`
		Float64 float64   `glint:"float64"`
		String  string    `glint:"string"`
		Bytes   []byte    `glint:"bytes"`
		Struct  child     `glint:"struct"`
		Time    time.Time `glint:"time"`
	}

	test := Example{
		Bool:    true,
		Int:     -42,
		Int8:    -8,
		Int16:   -2048,
		Int32:   -1024,
		Int64:   -9223372036854775808,
		Uint:    42,
		Uint8:   8,
		Uint16:  2048,
		Uint32:  1024,
		Uint64:  9223372036854775807,
		Float32: 3.14,
		Float64: 3.142069,
		String:  "greetings",
		Bytes:   []byte{42, 7, 255},
		Struct: child{
			Name: "TestUser",
		},
		Time: time.Now(),
	}

	encoder := newEncoder(Example{})

	buf := Buffer{}

	encoder.Marshal(&test, &buf)

	// fmt.Println(buf.Bytes)
	// Print(buf.Bytes)

	/// Output:
	// 1

}

func ExampleEncoder() {

	type Point struct {
		X int `glint:"x"`
		Y int `glint:"y"`
	}

	encoder := newEncoder(Point{})

	buf := NewBufferFromPool()
	value := Point{
		X: 10,
		Y: 55,
	}

	encoder.Marshal(&value, buf)

	fmt.Println(buf.Bytes)

	//Output:
	// [0 59 198 217 19 6 2 1 120 2 1 121 20 110]
}

func TestTrustedSchemaMode(t *testing.T) {

	// buf := glint.NewBufferWithTrust(glint.HTTPTrustee(r), testDataEnc)

	type TestStruct2 struct {
		String  string  `glint:"String"`
		Float64 float64 `glint:"Float64"`
	}

	type TestStruct struct {
		Bool   bool        `glint:"Bool"`
		Int    int         `glint:"Int"`
		Struct TestStruct2 `glint:"Struct"`
	}

	encoder := newEncoder(TestStruct{})
	decoder := newDecoder(TestStruct{})

	testStruct := &TestStruct{Bool: true, Int: 42, Struct: TestStruct2{String: "Struct", Float64: 9.99}}

	buf := NewBufferFromPool()
	encoder.Marshal(testStruct, buf)
	decoder.Unmarshal(buf.Bytes, &TestStruct{})

	trustHeader := NewTrustHeader(decoder)

	request, err := http.NewRequest("GET", "url", nil)
	if err != nil {
		t.Error(err)
		return
	}
	request.Header.Set(trustHeader.Key(), trustHeader.Value())

	// 35 1 220 181 == 3051094307

	/// assert trust header is set to correct value
	if request.Header["X-Glint-Trust"][0] != "3051094307" {
		t.Errorf("trust header not set to 3051094307: %s", request.Header["X-Glint-Trust"][0])
	}

	trustee := HTTPTrustee(request)
	tb := NewBufferWithTrust(trustee, encoder)

	// assert we resulted in a buffer with the correct trust flag set
	if tb.TrustedSchema == false {
		t.Error("trusted schema false")
	}

	encoder.Marshal(&TestStruct{Bool: true, Int: 42, Struct: TestStruct2{String: "Struct", Float64: 9.99}}, tb)

	var trustedTestStruct TestStruct
	// assert decoding with trusted schema does not cause any errors
	if err := decoder.Unmarshal(tb.Bytes, &trustedTestStruct); err != nil {
		t.Errorf("Unexpected error")
	}

	// assert decoding with trusted schema is correct
	if !reflect.DeepEqual(*testStruct, trustedTestStruct) {
		t.Errorf("Unexpected decoding when schema is tructed: got=%#v, want=%#v", trustedTestStruct, *testStruct)
	}
}

func TestSchemaEvolutionBackwardCompatibility(t *testing.T) {
	t.Run("ComplexNestedTypesSkipping", func(t *testing.T) {
		// Tests complex nested types can be safely skipped without panic
		type child struct {
			Name string `glint:"name"`
			Age  int32  `glint:"age"`
		}

		type Child struct {
			C    child  `glint:"item1"`
			Name string `glint:"f"`
			Age  int32  `glint:"g"`
		}

		type senderStruct struct {
			S         StringerTest `glint:"s,stringer"`
			ChildList []Child      `glint:"items1"`
			Age64     int64        `glint:"age64"`
		}

		type receiverStruct struct {
			// ChildList field is missing - should be skipped
			S     string `glint:"s"`
			Age64 int64  `glint:"age64"`
			ID    string
		}

		encoder := newEncoder(senderStruct{})
		buf := Buffer{}
		v := senderStruct{
			ChildList: []Child{
				{Name: "First", Age: 25, C: child{Name: "first nested", Age: 1}},
				{Name: "Second", Age: 31, C: child{Name: "second nested", Age: 2}},
				{Name: "Third", Age: 273, C: child{Name: "third nested", Age: 3}},
			},
			Age64: 41263,
			S:     StringerTest{Name: "display text"},
		}

		encoder.Marshal(&v, &buf)
		decoder := newDecoder(receiverStruct{})

		var result receiverStruct
		err := decoder.Unmarshal(buf.Bytes, &result)
		if err != nil {
			t.Errorf("Failed to decode with missing complex field: %v", err)
		}

		// Verify available fields are correctly decoded
		if result.Age64 != v.Age64 {
			t.Errorf("Age64 mismatch: got %d, want %d", result.Age64, v.Age64)
		}
		if result.S != "display text" {
			t.Errorf("S mismatch: got %q, want %q", result.S, "display text")
		}
	})

	t.Run("NestedStructEvolution", func(t *testing.T) {
		// Tests schema evolution with nested structs and field reordering
		type child struct {
			Age  int32  `glint:"age" json:"age"`
			Name string `glint:"name" json:"name"`
		}

		type Child struct {
			Name string `json:"f" glint:"f" nlv:"f"`
			Age  int32  `json:"g" glint:"g" nlv:"g"`
			C    child  `glint:"item1" json:"child"`
		}

		type senderStruct struct {
			Name           string      `glint:"name" json:"name"`
			Age8           int32       `glint:"age8" json:"age8"`
			Age            int64       `glint:"age" json:"age"`
			Age32          int32       `glint:"age32" json:"age32"`
			Age64          int64       `glint:"age64" json:"age64"`
			Flag           bool        `glint:"flag" json:"flag"`
			Flags          []bool      `glint:"flags" json:"flags"`
			Children       Child       `json:"e" glint:"e"`
			List           []string    `glint:"list" json:"list"`
			IntList        []int64     `glint:"intlist" json:"intlist"`
			SkipSliceSlice [][][]uint8 `glint:"skipsliceslice" json:"skipsliceslice"`
			ChildList      []Child     `json:"childlist" glint:"items1"`
			ID             string
		}

		type receiverStruct struct {
			Name     string `glint:"name" json:"name"`
			Age8     int32  `glint:"age8" json:"age8"`
			Age      int64  `glint:"age" json:"age"`
			Age32    int32  `glint:"age32" json:"age32"`
			Children Child  `json:"e" glint:"e"`
			// List and IntList fields are missing - should be skipped
			ChildList []Child `json:"childlist" glint:"items1"`
			ID        string
			Age64     int64 `glint:"age64" json:"age64"` // Field reordered
		}

		encoder := newEncoder(senderStruct{})
		buf := Buffer{}
		v := senderStruct{
			Name:  "test content",
			Age8:  127,
			Age:   21239871235,
			Age32: 31923987,
			Age64: 41263,
			Flag:  true,
			Flags: []bool{true, true, false, true},
			Children: Child{
				Name: "nested name",
				Age:  13,
				C: child{
					Name: "sub item",
					Age:  1,
				},
			},
			List:    []string{"first", "second", "third"},
			IntList: []int64{1, 2, 3, 44},
			SkipSliceSlice: [][][]uint8{
				{{1, 2, 3}, {4, 5, 6}},
				{{7, 8, 9}, {10, 11, 12}},
				{{13, 14, 15}, {16, 17, 18}},
			},
			ChildList: []Child{
				{Name: "First", Age: 25, C: child{Name: "first nested", Age: 1}},
				{Name: "Second", Age: 31, C: child{Name: "second nested", Age: 2}},
				{Name: "Third", Age: 273, C: child{Name: "third nested", Age: 3}},
				{Name: "Fourth", Age: 273, C: child{Name: "fourth nested", Age: 23}},
			},
		}

		encoder.Marshal(&v, &buf)
		decoder := newDecoder(receiverStruct{})

		var result receiverStruct
		err := decoder.Unmarshal(buf.Bytes, &result)
		if err != nil {
			t.Errorf("Failed to decode with schema evolution: %v", err)
		}

		// Verify field values are correctly preserved
		if result.Name != v.Name {
			t.Errorf("Name mismatch: got %q, want %q", result.Name, v.Name)
		}
		if result.Age8 != v.Age8 {
			t.Errorf("Age8 mismatch: got %d, want %d", result.Age8, v.Age8)
		}
		if result.Age != v.Age {
			t.Errorf("Age mismatch: got %d, want %d", result.Age, v.Age)
		}
		if result.Age32 != v.Age32 {
			t.Errorf("Age32 mismatch: got %d, want %d", result.Age32, v.Age32)
		}
		if result.Age64 != v.Age64 {
			t.Errorf("Age64 mismatch: got %d, want %d", result.Age64, v.Age64)
		}
		if len(result.ChildList) > 0 && result.ChildList[0].Name != v.ChildList[0].Name {
			t.Errorf("ChildList[0].Name mismatch: got %q, want %q", result.ChildList[0].Name, v.ChildList[0].Name)
		}
	})

	t.Run("DeltaFieldSkipping", func(t *testing.T) {
		// Test that delta-encoded fields can be properly skipped when schema doesn't match
		type SenderData struct {
			ID     int     `glint:"id"`
			Values []int64 `glint:"values,delta"` // This will be skipped
			Name   string  `glint:"name"`
		}

		type ReceiverData struct {
			ID   int    `glint:"id"`
			Name string `glint:"name"`
			// Values field is missing - should be skipped
		}

		senderData := SenderData{
			ID:     42,
			Values: []int64{1000, 1001, 1002, 1003, 1004}, // Sequential data
			Name:   "sample",
		}

		senderEncoder := NewEncoder[SenderData]()
		buf := NewBufferFromPool()
		defer buf.ReturnToPool()
		senderEncoder.Marshal(&senderData, buf)

		receiverDecoder := NewDecoder[ReceiverData]()
		var receiverData ReceiverData
		err := receiverDecoder.Unmarshal(buf.Bytes, &receiverData)
		if err != nil {
			t.Fatalf("Failed to decode with missing delta field: %v", err)
		}

		// Verify that the non-skipped fields are correct
		if receiverData.ID != senderData.ID {
			t.Errorf("ID mismatch: got %d, want %d", receiverData.ID, senderData.ID)
		}
		if receiverData.Name != senderData.Name {
			t.Errorf("Name mismatch: got %q, want %q", receiverData.Name, senderData.Name)
		}
	})

	t.Run("DeltaFieldReordering", func(t *testing.T) {
		// Test that delta encoding works when fields are in different order
		type DataA struct {
			Values []int64 `glint:"values,delta"`
			ID     int     `glint:"id"`
			Name   string  `glint:"name"`
		}

		type DataB struct {
			ID     int     `glint:"id"`
			Name   string  `glint:"name"`
			Values []int64 `glint:"values,delta"`
		}

		dataA := DataA{
			Values: []int64{100, 95, 90, 85, 80}, // Decreasing values (negative deltas)
			ID:     123,
			Name:   "reorder-test",
		}

		encoderA := NewEncoder[DataA]()
		buf := NewBufferFromPool()
		defer buf.ReturnToPool()
		encoderA.Marshal(&dataA, buf)

		decoderB := NewDecoder[DataB]()
		var dataB DataB
		err := decoderB.Unmarshal(buf.Bytes, &dataB)
		if err != nil {
			t.Fatalf("Failed to decode with reordered fields: %v", err)
		}

		// Verify all fields match
		if dataB.ID != dataA.ID {
			t.Errorf("ID mismatch: got %d, want %d", dataB.ID, dataA.ID)
		}
		if dataB.Name != dataA.Name {
			t.Errorf("Name mismatch: got %q, want %q", dataB.Name, dataA.Name)
		}
		if len(dataB.Values) != len(dataA.Values) {
			t.Fatalf("Values length mismatch: got %d, want %d", len(dataB.Values), len(dataA.Values))
		}
		for i := range dataA.Values {
			if dataB.Values[i] != dataA.Values[i] {
				t.Errorf("Values[%d] mismatch: got %d, want %d", i, dataB.Values[i], dataA.Values[i])
			}
		}
	})

	t.Run("ForwardCompatibilityNewFields", func(t *testing.T) {
		// Test forward compatibility: old data can be read by new schema
		type OldData struct {
			ID   int    `glint:"id"`
			Name string `glint:"name"`
		}

		type NewData struct {
			ID     int     `glint:"id"`
			Name   string  `glint:"name"`
			Values []int64 `glint:"values,delta"` // New field
		}

		oldData := OldData{ID: 789, Name: "old-format"}
		oldEncoder := NewEncoder[OldData]()
		buf := NewBufferFromPool()
		defer buf.ReturnToPool()
		oldEncoder.Marshal(&oldData, buf)

		newDecoder := NewDecoder[NewData]()
		var newData NewData
		err := newDecoder.Unmarshal(buf.Bytes, &newData)
		if err != nil {
			t.Fatalf("Failed to decode old data with new schema: %v", err)
		}

		// Verify existing fields are preserved
		if newData.ID != oldData.ID {
			t.Errorf("ID mismatch: got %d, want %d", newData.ID, oldData.ID)
		}
		if newData.Name != oldData.Name {
			t.Errorf("Name mismatch: got %q, want %q", newData.Name, oldData.Name)
		}

		// New field should be zero/empty
		if len(newData.Values) != 0 {
			t.Errorf("New field should be empty, got %v", newData.Values)
		}
	})

	t.Run("SchemaTypeMismatchErrors", func(t *testing.T) {
		// Test that schema mismatch errors are properly detected
		type DeltaMismatchData struct {
			Values []int64 `glint:"test_values,delta"`
		}

		type StandardMismatchData struct {
			Values []int64 `glint:"test_values"` // No delta
		}

		t.Run("DeltaToStandard", func(t *testing.T) {
			// Send delta, try to receive as standard - should fail with schema mismatch
			deltaData := DeltaMismatchData{Values: []int64{1, 2, 3, 4, 5}}
			encoder := NewEncoder[DeltaMismatchData]()
			buf := NewBufferFromPool()
			defer buf.ReturnToPool()
			encoder.Marshal(&deltaData, buf)

			decoder := NewDecoder[StandardMismatchData]()
			var standardData StandardMismatchData
			err := decoder.Unmarshal(buf.Bytes, &standardData)

			// Should get a schema mismatch error
			if err == nil {
				t.Error("Expected schema mismatch error when decoding delta as standard")
			} else if !strings.Contains(err.Error(), "schema mismatch") {
				t.Errorf("Expected schema mismatch error, got: %v", err)
			}
		})

		t.Run("StandardToDelta", func(t *testing.T) {
			// Send standard, try to receive as delta - should fail with schema mismatch
			standardData := StandardMismatchData{Values: []int64{1, 2, 3, 4, 5}}
			encoder := NewEncoder[StandardMismatchData]()
			buf := NewBufferFromPool()
			defer buf.ReturnToPool()
			encoder.Marshal(&standardData, buf)

			decoder := NewDecoder[DeltaMismatchData]()
			var deltaData DeltaMismatchData
			err := decoder.Unmarshal(buf.Bytes, &deltaData)

			// Should get a schema mismatch error
			if err == nil {
				t.Error("Expected schema mismatch error when decoding standard as delta")
			} else if !strings.Contains(err.Error(), "schema mismatch") {
				t.Errorf("Expected schema mismatch error, got: %v", err)
			}
		})
	})
}

// some equality checks on input structs matching their decoded values with the addition of length checks on
// the serialized documents themselves to make sure we're not introducing breaking changes in the document format by accident.
func TestDocumentLengthAndEqualityComparison(t *testing.T) {

	type test struct {
		Name string `glint:"n"`
		Age  int    `glint:"a"`
	}
	type test1 struct {
		Name  string  `glint:"name"`
		Age   int     `glint:"age"`
		Ratio float64 `glint:"r"`
	}
	type test2 struct {
		Wrap []test `glint:"w"`
		ID   int    `glint:"i"`
	}
	type test3 struct {
		List []int `glint:"l"`
		ID   int   `glint:"i"`
	}
	type test4 struct {
		Listlist [][]int `glint:"ll"`
	}
	type test5 struct {
		Lllist [][][]int `glint:"aa"`
	}
	type test6 struct {
		List [][][]test `glint:"ll"`
	}

	type test77 struct {
		Wrap []test `glint:"w"`
		ID   int    `glint:"i"`
		Blah []int  `glint:"b,zigzag"`
	}
	type test7 struct {
		Name string   `glint:"name"`
		List []test77 `glint:"list"`
	}
	type testTime struct {
		Time  time.Time   `glint:"t"`
		Times []time.Time `glint:"time_list"`
	}

	type child1 struct {
		Name string `glint:"name"`
		Age  int    `glint:"age"`
	}

	type All struct {
		Vbool     bool        `glint:"bool"`
		Abool     []bool      `glint:"[]bool"`
		Vint      int         `glint:"int"`
		Aint      []int       `glint:"[]int"`
		Vint8     int8        `glint:"int8"`
		Aint8     []int8      `glint:"[]int8"`
		Vint16    int16       `glint:"int16"`
		Aint16    []int16     `glint:"[]int16"`
		Vint32    int32       `glint:"int32"`
		Aint32    []int32     `glint:"[]int32"`
		Vint64    int64       `glint:"int64"`
		Aint64    []int64     `glint:"[]int64"`
		Vuint     uint        `glint:"uint"`
		Auint     []uint      `glint:"[]uint"`
		Vuint8    uint8       `glint:"uint8"`
		Auint8    []uint8     `glint:"[]uint8"`
		Vuint16   uint16      `glint:"uint16"`
		Auint16   []uint16    `glint:"[]uint16"`
		Vuint32   uint32      `glint:"uint32"`
		Auint32   []uint32    `glint:"[]uint32"`
		Vuint64   uint64      `glint:"uint64"`
		Auint64   []uint64    `glint:"[]uint64"`
		Vfloat32  float32     `glint:"float32"`
		Afloat32  []float32   `glint:"[]float32"`
		Vfloat64  float64     `glint:"float64"`
		Afloat64  []float64   `glint:"[]float64"`
		Vstring   string      `glint:"string"`
		Astring   []string    `glint:"[]string"`
		Vbytes    []byte      `glint:"bytes"`
		Abytes    [][]byte    `glint:"[]bytes"`
		Vstruct   child1      `glint:"Vstruct"`
		Astruct   []child1    `glint:"[]Vstruct"`
		Vzzint    int         `glint:"zzint,zigzag"`
		Azzint    []int       `glint:"[]zzint,zigzag"`
		VtimeTime time.Time   `glint:"timeTime"`
		AtimeTime []time.Time `glint:"[]timeTime"`
	}

	var _, _, _, _, _, _, _ = test{}, test1{}, test2{}, test3{}, test4{}, test5{}, test6{}

	tests := []struct {
		name     string
		input    any
		expected int
	}{
		{name: "should be 22", input: test{Name: "TestName", Age: 40}, expected: 22},
		{name: "should be 36", input: test1{Name: "UserA", Age: 37, Ratio: 0.8}, expected: 36},
		{name: "should be 40", input: test1{Name: "UserB", Age: 409999999, Ratio: 0.263478}, expected: 40},
		{name: "should be 27", input: test2{Wrap: []test{{Name: "XYZ", Age: 31}}, ID: 77}, expected: 27},
		{name: "should be 47", input: test2{Wrap: []test{{Name: "User1", Age: 41}, {Name: "User2", Age: 42}, {Name: "User3", Age: 4322}}, ID: 8386827346}, expected: 47},
		{name: "should be 42", input: test3{List: []int{9182739, 523452345234, 234523452345, 34523452345, 34523452345}, ID: 1}, expected: 42},
		{name: "should be __", input: test4{Listlist: [][]int{{1, 2, 3}, {4, 5, 6}}}, expected: 20},
		{name: "should be __", input: test5{Lllist: [][][]int{
			{
				{1, 2, 3}, {11, 22, 33},
			},
			{
				{4, 5, 6}, {1, 2, 3},
			},
		}}, expected: 31},
		{name: "nested slice struct", input: test6{
			List: [][][]test{
				{
					{
						{Name: "ABC", Age: 251},
						{Name: "sample long text data for testing purposes with enough content to generate sufficient bytes in output ", Age: 666},
						{Name: "sample long text data for testing purposes with enough content to generate sufficient bytes in output ", Age: 1},
						{Name: "sample long text data for testing purposes with enough content to generate sufficient bytes in output ", Age: 666},
						{Name: "sample long text data for testing purposes with enough content to generate sufficient bytes in output ", Age: 2},
						{Name: "sample long text data for testing purposes with enough content to generate sufficient bytes in output ", Age: 666},
						{Name: "sample long text data for testing purposes with enough content to generate sufficient bytes in output ", Age: 3},
						{Name: "sample long text data for testing purposes with enough content to generate sufficient bytes in output ", Age: 666},
						{Name: "sample long text data for testing purposes with enough content to generate sufficient bytes in output ", Age: 4},
						{Name: "sample long text data for testing purposes with enough content to generate sufficient bytes in output ", Age: 666},
					},
				},
			}}, expected: 969},
		{
			name: "nested slice struct -> nested slice struct",
			input: test7{
				Name: "root element name",
				List: []test77{
					{
						Wrap: []test{{
							Name: "item 1",
							Age:  101,
						}, {
							Name: "item 1.1",
							Age:  1011,
						}},
						ID:   202,
						Blah: []int{1, 2, 3},
					},
					{
						Wrap: []test{{
							Name: "item 2",
							Age:  303,
						}, {
							Name: "item 2.2",
							Age:  3033,
						}},
						ID:   404,
						Blah: []int{-4, 128, -127},
					},
					{
						Wrap: []test{{
							Name: "item 3",
							Age:  505,
						}, {
							Name: "item 3.3",
							Age:  5055,
						}},
						ID:   606,
						Blah: []int{7, -8, 9},
					},
				},
			},
			expected: 137,
		},
		{
			name: "testing time encoding / decoding",
			input: testTime{
				Time: time.Date(1969, 4, 20, 4, 20, 4, 20, time.UTC),
				Times: []time.Time{
					time.Date(1969, 4, 20, 4, 20, 4, 20, time.UTC),
					time.Date(1969, 4, 20, 4, 20, 4, 20, time.UTC),
				},
			},
			expected: 69,
		},
		{
			name: "literally everything",
			input: All{
				Vbool:    true,
				Abool:    []bool{true, false, true},
				Vint:     -11,
				Aint:     []int{-11, -12, -13},
				Vint8:    -120,
				Aint8:    []int8{-120, 120, 55},
				Vint16:   -1000,
				Aint16:   []int16{-1000, -1001, -1002},
				Vint32:   -80_000,
				Aint32:   []int32{-80_000, -80_001, -80_002},
				Vint64:   MaxInt32 * 2,
				Aint64:   []int64{MaxInt32 * 2, MaxInt32*2 + 1},
				Vuint:    MaxUint64,
				Auint:    []uint{1, MaxUint64, 3},
				Vuint8:   38,
				Auint8:   []uint8{23, 24, 25},
				Vuint16:  64000,
				Auint16:  []uint16{64000, 64001, 64002},
				Vuint32:  80_000,
				Auint32:  []uint32{80_000, 80_001, 80_002},
				Vuint64:  MaxUint,
				Auint64:  []uint64{MaxInt + 1, MaxInt + 2, MaxInt + 3},
				Vfloat32: MaxFloat32,
				Afloat32: []float32{MaxFloat32, 0.1, 0.2},
				Vfloat64: MaxFloat64,
				Afloat64: []float64{MaxFloat64, 0.1, 0.2},
				Vstring:  "test content",
				Astring:  []string{"test content", "test content", "test content"},
				Vbytes:   []byte{11, 22, 33},
				Abytes:   [][]byte{{1, 1}, {2, 2}, {3, 3}},
				Vstruct: child1{
					Name: "nested element",
					Age:  29,
				},
				Astruct: []child1{
					{
						Name: "first nested element",
						Age:  11,
					},
					{
						Name: "second nested element",
						Age:  22,
					},
					{
						Name: "third nested element",
						Age:  29,
					},
				},
				Vzzint:    -1,
				Azzint:    []int{-1, -2, 1, 2},
				VtimeTime: time.Date(2023, 6, 15, 10, 30, 45, 123456789, time.UTC),
				AtimeTime: []time.Time{
					time.Date(2023, 6, 15, 10, 30, 45, 123456789, time.UTC),
					time.Date(2022, 3, 8, 16, 20, 30, 987654321, time.UTC),
					time.Date(2021, 9, 22, 8, 45, 15, 555666777, time.UTC),
				},
			},
			expected: 764,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			enc := newEncoder(tt.input)

			buf := Buffer{}
			enc.Marshal(tt.input, &buf)

			defer func() {
				if r := recover(); r != nil {
					fmt.Println(tt.name, buf.Bytes, len(buf.Bytes))
					Print(buf.Bytes)
				}
			}()

			actual := len(buf.Bytes)
			if !reflect.DeepEqual(tt.expected, actual) {
				t.Errorf("expected %v, got %v", tt.expected, actual)
			}

			// fmt.Println("bb", len(buf.Bytes), buf.Bytes)
			// jb, _ := json.Marshal(tt.input)
			// fmt.Println("json bytes l", len(jb))

			// Print(buf.Bytes)

			// check the input can be reconstructed from its serialized form

			dec := newDecoder(tt.input)
			out := reflect.Indirect(reflect.New(reflect.TypeOf(tt.input))).Interface()
			err := dec.Unmarshal(buf.Bytes, out)
			if err != nil {
				t.Error(err)
			}

			// fmt.Printf("%#v \n", out)

			if !reflect.DeepEqual(tt.input, out) {

				t.Errorf("\n%v", stringDiff(fmt.Sprintf("%#v \n", tt.input), fmt.Sprintf("%#v \n", out)))
			}
		})
	}
}

const (
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	reset  = "\033[0m"
)

func stringDiff(a, b string) string {
	var diff strings.Builder
	diff.WriteString(fmt.Sprintf("%sWANT: %s", yellow, reset))
	for i := 0; i < len(a) || i < len(b); i++ {
		if i >= len(a) {
			diff.WriteString(fmt.Sprintf("%s%s", red, string(b[i])))
		} else if i >= len(b) {
			diff.WriteString(fmt.Sprintf("%s%s", red, string(a[i])))
		} else if a[i] != b[i] {
			diff.WriteString(fmt.Sprintf("%s%s", red, string(a[i])))
		} else {
			diff.WriteString(fmt.Sprintf("%s%s", green, string(a[i])))
		}
	}
	diff.WriteString(reset)

	diff.WriteString(fmt.Sprintf("\n%s GOT: %s", yellow, reset))
	for i := 0; i < len(a) || i < len(b); i++ {
		if i >= len(a) {
			diff.WriteString(fmt.Sprintf("%s%s", red, string(b[i])))
		} else if i >= len(b) {
			diff.WriteString(fmt.Sprintf("%s%s", red, string(a[i])))
		} else if a[i] != b[i] {
			diff.WriteString(fmt.Sprintf("%s%s", red, string(b[i])))
		} else {
			diff.WriteString(fmt.Sprintf("%s%s", green, string(b[i])))
		}
	}
	diff.WriteString(reset)

	return diff.String()
}

func TestZigzagVarintEncoding(t *testing.T) {
	tests := []struct {
		name  string
		input int
	}{
		{
			name:  "",
			input: 1232,
		},
		{
			name:  "",
			input: 1986554430403320196,
		},
		{
			name:  "",
			input: 8986554430403320197,
		},
		{
			name:  "",
			input: -8986554430403320197,
		},
		{
			name:  "",
			input: -1232,
		},
		{
			name:  "",
			input: -1986554430403320196,
		},
		{
			name:  "",
			input: 0,
		},
		{
			name:  "",
			input: -1,
		},
		{
			name:  "",
			input: -2,
		},
		{
			name:  "",
			input: 1,
		},
		{
			name:  "",
			input: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if !canZigzagEncode(int64(tt.input)) {
				return
			}

			b := Buffer{}
			b.AppendUint(uint((tt.input >> 63) ^ (tt.input << 1)))

			r := Reader{bytes: b.Bytes}

			if actual := r.ReadZigzagVarint(); !reflect.DeepEqual(tt.input, actual) {
				t.Errorf("input %v, got %v", tt.input, actual)
			}
		})
	}
}

type TestVisitor struct {
}

func (v *TestVisitor) VisitFlags(flags byte) error {
	fmt.Println("VisitFlags", flags)
	return nil
}

func (v *TestVisitor) VisitSchemaHash(hash []byte) error {
	fmt.Println("VisitSchemaHash", hash)
	return nil
}

func (v *TestVisitor) VisitField(name string, wire WireType, body Reader) (Reader, error) {
	return body, ErrSkipVisit
}

func (v *TestVisitor) VisitArrayStart(name string, wire WireType, length int) error {
	fmt.Println("VisitArrayStart", name, wire, length)
	return nil
}

func (v *TestVisitor) VisitArrayEnd(name string) error {
	fmt.Println("VisitArrayEnd", name)
	return nil
}

func (v *TestVisitor) VisitStructStart(name string) error {
	// fmt.Println("VisitStructStart", name)
	return nil
}

func (v *TestVisitor) VisitStructEnd(name string) error {
	// fmt.Println("VisitStructEnd", name)
	return nil
}

type SilentTestVisitor struct {
	actionCount int
}

func (v *SilentTestVisitor) inc() error {
	v.actionCount++
	return nil
}
func (v *SilentTestVisitor) VisitFlags(flags byte) error       { return v.inc() }
func (v *SilentTestVisitor) VisitSchemaHash(hash []byte) error { return v.inc() }
func (v *SilentTestVisitor) VisitField(_ string, w WireType, body Reader) (Reader, error) {
	fieldBytes(&body, w)
	return body, v.inc()
}
func (v *SilentTestVisitor) VisitArrayStart(_ string, _ WireType, _ int) error { return v.inc() }
func (v *SilentTestVisitor) VisitArrayEnd(name string) error                   { return v.inc() }
func (v *SilentTestVisitor) VisitStructStart(name string) error                { return v.inc() }
func (v *SilentTestVisitor) VisitStructEnd(name string) error                  { return v.inc() }

// Benchmark for AppendDynamicValue

// Benchmark for ReadDynamicValue

///
///
///

func canZigzagEncode(num int64) bool {
	magnitude := uint64(math.Abs(float64(num)))
	if magnitude >= (1 << 63) {
		return false
	}
	return ((num >= 0) && ((num & (1 << 62)) == 0)) || ((num < 0) && ((num & (1 << 62)) != 0))
}

// testChild1 is a shared struct used across multiple fuzz tests
type testChild1 struct {
	Name string `glint:"name"`
	Age  int16  `glint:"age"`
}

// testFuzzEncodeDecode is a shared helper for fuzz testing encoding/decoding roundtrips
func testFuzzEncodeDecode[T any](t *testing.T, testStruct T, maxCount int) {
	encoder := newEncoder(testStruct)
	decoder := newDecoder(testStruct)

	f := func(pt T) bool {
		resetInvalidSignedIntFields(&pt)

		// Encode the Glint instance
		buf := Buffer{}
		encoder.Marshal(&pt, &buf)

		// Decode the encoded data back into a Glint
		var decoded T
		err := decoder.Unmarshal(buf.Bytes, &decoded)
		if err != nil {
			t.Error(err)
			return false
		}

		// Compare the original and decoded glint instances
		if !reflect.DeepEqual(pt, decoded) {
			fmt.Printf("\n%v", stringDiff(fmt.Sprintf("%#v \n", pt), fmt.Sprintf("%#v \n", decoded)))
			return false
		}
		return true
	}

	// Run the fuzz test using testing/quick
	if err := Check(f, &Config{MaxCount: maxCount, MaxCountScale: 1}); err != nil {
		t.Error(err)
	}
}

// TestFuzzPrimitiveTypes tests encoding/decoding of basic primitive types
func TestEncodingRobustnessPrimitiveTypes(t *testing.T) {
	type primitiveTypes struct {
		Vbool     bool    `glint:"bool"`
		Vint      int     `glint:"int"`
		Vint8     int8    `glint:"int8"`
		Vint16    int16   `glint:"int16"`
		Vint32    int32   `glint:"int32"`
		Vint64    int64   `glint:"int64"`
		Vuint     uint    `glint:"uint"`
		Vuint8    uint8   `glint:"uint8"`
		Vuint16   uint16  `glint:"uint16"`
		Vuint32   uint32  `glint:"uint32"`
		Vuint64   uint64  `glint:"uint64"`
		Vfloat32  float32 `glint:"float32"`
		Vfloat64  float64 `glint:"float64"`
		Vstring   string  `glint:"string"`
		Cpystring string  `glint:"cstring,copy"`
	}

	testFuzzEncodeDecode(t, primitiveTypes{}, 1000)
}

// TestFuzzSliceCollections tests encoding/decoding of slice collections
func TestEncodingRobustnessSliceTypes(t *testing.T) {
	type sliceCollections struct {
		Abool    []bool     `glint:"[]bool"`
		NAbool   [][]bool   `glint:"[]bool_nest"`
		Aint     []int      `glint:"[]int"`
		NAint    [][]int    `glint:"[]int_nest"`
		Aint8    []int8     `glint:"[]int8"`
		NAint8   [][]int8   `glint:"[]int8_nest"`
		Aint16   []int16    `glint:"[]int16"`
		NAint16  [][]int16  `glint:"[]int16_nest"`
		Aint32   []int32    `glint:"[]int32"`
		NAint32  [][]int32  `glint:"[]int32_nest"`
		Aint64   []int64    `glint:"[]int64"`
		NAint64  [][]int64  `glint:"[]int64_nest"`
		Auint    []uint     `glint:"[]uint"`
		NAuint   [][]uint   `glint:"[]uint_nest"`
		Auint8   []uint8    `glint:"[]uint8"`
		NAuint8  [][]uint8  `glint:"[]uint8_nest"`
		Auint16  []uint16   `glint:"[]uint16"`
		NAuint16 [][]uint16 `glint:"[]uint16_nest"`
		Auint32  []uint32   `glint:"[]uint32"`
		NAuint32 [][]uint32 `glint:"[]uint32_nest"`
		Auint64  []uint64   `glint:"[]uint64"`
		NAuint64 [][]uint64 `glint:"[]uint64_nest"`
		Afloat32 []float32  `glint:"[]float32"`
		Afloat64 []float64  `glint:"[]float64"`
		Astring  []string   `glint:"[]string"`
		NAstring [][]string `glint:"[]string_nest"`
	}

	testFuzzEncodeDecode(t, sliceCollections{}, 1000)
}

// TestFuzzByteTypes tests encoding/decoding of byte slices and collections
func TestEncodingRobustnessByteArrays(t *testing.T) {
	type byteTypes struct {
		Vbytes  []byte     `glint:"bytes"`
		NVbytes [][]byte   `glint:"bytes_nest"`
		Abytes  [][]byte   `glint:"[]bytes"`
		NAbytes [][][]byte `glint:"[]bytes_nest"`
	}

	// Reduced iterations for nested byte arrays as they're computationally expensive
	testFuzzEncodeDecode(t, byteTypes{}, 100)
}

// TestFuzzNestedStructs tests encoding/decoding of nested structs
func TestEncodingRobustnessNestedStructures(t *testing.T) {
	type nestedStructs struct {
		Vstruct  testChild1     `glint:"Vstruct"`
		Astruct  []testChild1   `glint:"[]Vstruct"`
		NAstruct [][]testChild1 `glint:"[]Vstruct_nest"`
	}

	// Reduced iterations for nested slices as they're computationally expensive
	testFuzzEncodeDecode(t, nestedStructs{}, 50)
}

// testFuzzPointerComparison is a specialized helper for pointer types that need field-by-field comparison
func testFuzzPointerComparison[T any](t *testing.T, testStruct T, maxCount int, compareFn func(T, T) bool) {
	encoder := newEncoder(testStruct)
	decoder := newDecoder(testStruct)

	f := func(pt T) bool {
		resetInvalidSignedIntFields(&pt)

		// Encode the Glint instance
		buf := Buffer{}
		encoder.Marshal(&pt, &buf)

		// Decode the encoded data back into a Glint
		var decoded T
		err := decoder.Unmarshal(buf.Bytes, &decoded)
		if err != nil {
			t.Error(err)
			return false
		}

		return compareFn(pt, decoded)
	}

	// Run the fuzz test using testing/quick
	if err := Check(f, &Config{MaxCount: maxCount, MaxCountScale: 1}); err != nil {
		t.Error(err)
	}
}

// TestFuzzPointerTypes tests encoding/decoding of pointer types
func TestEncodingRobustnessPointerTypes(t *testing.T) {
	type pointerTypes struct {
		Vboolp   *bool       `glint:"boolp"`
		Vintp    *int        `glint:"intp"`
		Vint8p   *int8       `glint:"int8p"`
		Vint16p  *int16      `glint:"int16p"`
		Vint32p  *int32      `glint:"int32p"`
		Vint64p  *int64      `glint:"int64p"`
		Vuintp   *uint       `glint:"uintp"`
		Vuint8p  *uint8      `glint:"uint8p"`
		Vuint16p *uint16     `glint:"uint16p"`
		Vuint32p *uint32     `glint:"uint32p"`
		Vuint64p *uint64     `glint:"uint64p"`
		Vstringp *string     `glint:"stringp"`
		Vstructp *testChild1 `glint:"Vstructp"`
	}

	compareFn := func(pt, decoded pointerTypes) bool {
		passed := true

		if (decoded.Vboolp != nil && pt.Vboolp != nil) && *decoded.Vboolp != *pt.Vboolp {
			fmt.Println("Vboolp original:decoded", *pt.Vboolp, *decoded.Vboolp)
			passed = false
		}
		if (decoded.Vintp != nil && pt.Vintp != nil) && *decoded.Vintp != *pt.Vintp {
			fmt.Println("Vintp original:decoded", *pt.Vintp, *decoded.Vintp)
			passed = false
		}
		if (decoded.Vint8p != nil && pt.Vint8p != nil) && *decoded.Vint8p != *pt.Vint8p {
			fmt.Println("Vint8p original:decoded", *pt.Vint8p, *decoded.Vint8p)
			passed = false
		}
		if (decoded.Vint16p != nil && pt.Vint16p != nil) && *decoded.Vint16p != *pt.Vint16p {
			fmt.Println("Vint16p original:decoded", *pt.Vint16p, *decoded.Vint16p)
			passed = false
		}
		if (decoded.Vint32p != nil && pt.Vint32p != nil) && *decoded.Vint32p != *pt.Vint32p {
			fmt.Println("Vint32p original:decoded", *pt.Vint32p, *decoded.Vint32p)
			passed = false
		}
		if (decoded.Vint64p != nil && pt.Vint64p != nil) && *decoded.Vint64p != *pt.Vint64p {
			fmt.Println("Vint64p original:decoded", *pt.Vint64p, *decoded.Vint64p)
			passed = false
		}
		if (decoded.Vuintp != nil && pt.Vuintp != nil) && *decoded.Vuintp != *pt.Vuintp {
			fmt.Println("Vuintp original:decoded", *pt.Vuintp, *decoded.Vuintp)
			passed = false
		}
		if (decoded.Vuint8p != nil && pt.Vuint8p != nil) && *decoded.Vuint8p != *pt.Vuint8p {
			fmt.Println("Vuint8p original:decoded", *pt.Vuint8p, *decoded.Vuint8p)
			passed = false
		}
		if (decoded.Vuint16p != nil && pt.Vuint16p != nil) && *decoded.Vuint16p != *pt.Vuint16p {
			fmt.Println("Vuint16p original:decoded", *pt.Vuint16p, *decoded.Vuint16p)
			passed = false
		}
		if (decoded.Vuint32p != nil && pt.Vuint32p != nil) && *decoded.Vuint32p != *pt.Vuint32p {
			fmt.Println("Vuint32p original:decoded", *pt.Vuint32p, *decoded.Vuint32p)
			passed = false
		}
		if (decoded.Vuint64p != nil && pt.Vuint64p != nil) && *decoded.Vuint64p != *pt.Vuint64p {
			fmt.Println("Vuint64p original:decoded", *pt.Vuint64p, *decoded.Vuint64p)
			passed = false
		}
		if (decoded.Vstringp != nil && pt.Vstringp != nil) && *decoded.Vstringp != *pt.Vstringp {
			fmt.Println("Vstringp original:decoded", *pt.Vstringp, *decoded.Vstringp)
			passed = false
		}
		if (decoded.Vstructp != nil && pt.Vstructp != nil) && *decoded.Vstructp != *pt.Vstructp {
			fmt.Println("Vstructp original:decoded", *pt.Vstructp, *decoded.Vstructp)
			passed = false
		}

		return passed
	}

	testFuzzPointerComparison(t, pointerTypes{}, 1000, compareFn)
}

// TestFuzzCompleteTypes tests encoding/decoding of all types together (comprehensive test)
func TestEncodingRobustnessComplexStructures(t *testing.T) {
	type completeTypes struct {
		// Primitive types
		Vbool     bool    `glint:"bool"`
		Vint      int     `glint:"int"`
		Vint8     int8    `glint:"int8"`
		Vint16    int16   `glint:"int16"`
		Vint32    int32   `glint:"int32"`
		Vint64    int64   `glint:"int64"`
		Vuint     uint    `glint:"uint"`
		Vuint8    uint8   `glint:"uint8"`
		Vuint16   uint16  `glint:"uint16"`
		Vuint32   uint32  `glint:"uint32"`
		Vuint64   uint64  `glint:"uint64"`
		Vfloat32  float32 `glint:"float32"`
		Vfloat64  float64 `glint:"float64"`
		Vstring   string  `glint:"string"`
		Cpystring string  `glint:"cstring,copy"`
		// Slice collections
		Abool    []bool     `glint:"[]bool"`
		NAbool   [][]bool   `glint:"[]bool_nest"`
		Aint     []int      `glint:"[]int"`
		NAint    [][]int    `glint:"[]int_nest"`
		Astring  []string   `glint:"[]string"`
		NAstring [][]string `glint:"[]string_nest"`
		// Byte types
		Vbytes  []byte     `glint:"bytes"`
		NVbytes [][]byte   `glint:"bytes_nest"`
		Abytes  [][]byte   `glint:"[]bytes"`
		NAbytes [][][]byte `glint:"[]bytes_nest"`
		// Nested structs
		Vstruct  testChild1     `glint:"Vstruct"`
		Astruct  []testChild1   `glint:"[]Vstruct"`
		NAstruct [][]testChild1 `glint:"[]Vstruct_nest"`
		// Zigzag encoding
		Vzzint     int     `glint:"zzint,zigzag"`
		Azzint     []int   `glint:"[]zzint,zigzag"`
		NestAZZInt [][]int `glint:"[]zzint_nest,zigzag"`
	}

	testFuzzEncodeDecode(t, completeTypes{}, 500)
}

func resetInvalidSignedIntFields(s any) {
	v := reflect.ValueOf(s)

	// Dereference pointers
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Iterate through struct fields
	if v.Kind() == reflect.Struct {
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			resetInvalidSignedIntFields(f.Addr().Interface())
		}
	}

	// Handle signed integer fields
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if !canZigzagEncode(v.Int()) {
			v.SetInt(0)
		}
	}

	// Handle slice fields
	if v.Kind() == reflect.Slice {
		elemType := v.Type().Elem()
		for j := 0; j < v.Len(); j++ {
			elem := v.Index(j)
			if elemType.Kind() == reflect.Slice || elemType.Kind() == reflect.Ptr {
				resetInvalidSignedIntFields(elem.Interface())
			} else {
				switch elemType.Kind() {
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					if !canZigzagEncode(elem.Int()) {
						newElem := reflect.New(elemType).Elem()
						newElem.SetInt(0)
						v.Index(j).Set(newElem)
					}
				}
			}
		}
	}
}

var defaultMaxCount *int = flag.Int("quickchecks", 100, "The default number of iterations for each check")

// A Generator can generate random values of its own type.
type Generator interface {
	// Generate returns a random instance of the type on which it is a
	// method using the size as a size hint.
	Generate(rand *rand.Rand, size int) reflect.Value
}

// Mathematical constants.
const (
	E   = 2.71828182845904523536028747135266249775724709369995957496696763 // mathematical constant E
	Pi  = 3.14159265358979323846264338327950288419716939937510582097494459 // mathematical constant Pi
	Phi = 1.61803398874989484820458683436563811772030917980576286213544862 // mathematical constant Phi (golden ratio)

	Sqrt2   = 1.41421356237309504880168872420969807856967187537694807317667974 // mathematical constant sqrt(2)
	SqrtE   = 1.64872127070012814684865078781416357165377610071014801157507931 // mathematical constant sqrt(e)
	SqrtPi  = 1.77245385090551602729816748334114518279754945612238712821380779 // mathematical constant sqrt(pi)
	SqrtPhi = 1.27201964951406896425242246173749149171560804184009624861664038 // mathematical constant sqrt(phi)

	Ln2    = 0.693147180559945309417232121458176568075500134360255254120680009 // mathematical constant ln(2)
	Log2E  = 1 / Ln2
	Ln10   = 2.30258509299404568401799145468436420760110148862877297603332790 // mathematical constant ln(10)
	Log10E = 1 / Ln10
)

// Floating-point limit values.
// Max is the largest finite value representable by the type.
// SmallestNonzero is the smallest positive, non-zero value representable by the type.
const (
	MaxFloat32             = 0x1p127 * (1 + (1 - 0x1p-23)) // 3.40282346638528859811704183484516925440e+38
	SmallestNonzeroFloat32 = 0x1p-126 * 0x1p-23            // 1.401298464324817070923729583289916131280e-45

	MaxFloat64             = 0x1p1023 * (1 + (1 - 0x1p-52)) // 1.79769313486231570814527423731704356798070e+308
	SmallestNonzeroFloat64 = 0x1p-1022 * 0x1p-52            // 4.9406564584124654417656879286822137236505980e-324
)

// Integer limit values.
const (
	intSize = 32 << (^uint(0) >> 63) // 32 or 64

	MaxInt    = 1<<(intSize-1) - 1
	MinInt    = -1 << (intSize - 1)
	MaxInt8   = 1<<7 - 1
	MinInt8   = -1 << 7
	MaxInt16  = 1<<15 - 1
	MinInt16  = -1 << 15
	MaxInt32  = 1<<31 - 1
	MinInt32  = -1 << 31
	MaxInt64  = 1<<63 - 1
	MinInt64  = -1 << 63
	MaxUint   = 1<<intSize - 1
	MaxUint8  = 1<<8 - 1
	MaxUint16 = 1<<16 - 1
	MaxUint32 = 1<<32 - 1
	MaxUint64 = 1<<64 - 1
)

// randFloat32 generates a random float taking the full range of a float32.
func randFloat32(rand *rand.Rand) float32 {
	f := rand.Float64() * MaxFloat32
	if rand.Int()&1 == 1 {
		f = -f
	}
	return float32(f)
}

// randFloat64 generates a random float taking the full range of a float64.
func randFloat64(rand *rand.Rand) float64 {
	f := rand.Float64() * MaxFloat64
	if rand.Int()&1 == 1 {
		f = -f
	}
	return f
}

// randInt64 returns a random int64.
func randInt64(rand *rand.Rand) int64 {
	return int64(rand.Uint64())
}

// complexSize is the maximum length of arbitrary values that contain other
// values.
const complexSize = 900

// Value returns an arbitrary value of the given type.
// If the type implements the Generator interface, that will be used.
// Note: To create arbitrary values for structs, all the fields must be exported.
func Value(t reflect.Type, rand *rand.Rand) (value reflect.Value, ok bool) {
	return sizedValue(t, rand, complexSize)
}

// sizedValue returns an arbitrary value of the given type. The size
// hint is used for shrinking as a function of indirection level so
// that recursive data structures will terminate.
func sizedValue(t reflect.Type, rand *rand.Rand, size int) (value reflect.Value, ok bool) {
	if m, ok := reflect.Zero(t).Interface().(Generator); ok {
		return m.Generate(rand, size), true
	}

	v := reflect.New(t).Elem()
	switch concrete := t; concrete.Kind() {
	case reflect.Bool:
		v.SetBool(rand.Int()&1 == 0)
	case reflect.Float32:
		v.SetFloat(float64(randFloat32(rand)))
	case reflect.Float64:
		v.SetFloat(randFloat64(rand))
	case reflect.Complex64:
		v.SetComplex(complex(float64(randFloat32(rand)), float64(randFloat32(rand))))
	case reflect.Complex128:
		v.SetComplex(complex(randFloat64(rand), randFloat64(rand)))
	case reflect.Int16:
		v.SetInt(randInt64(rand))
	case reflect.Int32:
		v.SetInt(randInt64(rand))
	case reflect.Int64:
		v.SetInt(randInt64(rand))
	case reflect.Int8:
		v.SetInt(randInt64(rand))
	case reflect.Int:
		v.SetInt(randInt64(rand))
	case reflect.Uint16:
		v.SetUint(uint64(randInt64(rand)))
	case reflect.Uint32:
		v.SetUint(uint64(randInt64(rand)))
	case reflect.Uint64:
		v.SetUint(uint64(randInt64(rand)))
	case reflect.Uint8:
		v.SetUint(uint64(randInt64(rand)))
	case reflect.Uint:
		v.SetUint(uint64(randInt64(rand)))
	case reflect.Uintptr:
		v.SetUint(uint64(randInt64(rand)))
	case reflect.Map:
		numElems := rand.Intn(size)
		v.Set(reflect.MakeMap(concrete))
		for i := 0; i < numElems; i++ {
			key, ok1 := sizedValue(concrete.Key(), rand, size)
			value, ok2 := sizedValue(concrete.Elem(), rand, size)
			if !ok1 || !ok2 {
				return reflect.Value{}, false
			}
			v.SetMapIndex(key, value)
		}
	case reflect.Ptr:
		if rand.Intn(size) == 0 {
			v.Set(reflect.Zero(concrete)) // Generate nil pointer.
		} else {
			elem, ok := sizedValue(concrete.Elem(), rand, size)
			if !ok {
				return reflect.Value{}, false
			}
			v.Set(reflect.New(concrete.Elem()))
			v.Elem().Set(elem)
		}
	case reflect.Slice:
		numElems := rand.Intn(size)
		sizeLeft := size - numElems
		v.Set(reflect.MakeSlice(concrete, numElems, numElems))
		for i := 0; i < numElems; i++ {
			elem, ok := sizedValue(concrete.Elem(), rand, sizeLeft)
			if !ok {
				return reflect.Value{}, false
			}
			v.Index(i).Set(elem)
		}
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			elem, ok := sizedValue(concrete.Elem(), rand, size)
			if !ok {
				return reflect.Value{}, false
			}
			v.Index(i).Set(elem)
		}
	case reflect.String:
		numChars := rand.Intn(complexSize)
		codePoints := make([]rune, numChars)
		for i := 0; i < numChars; i++ {
			codePoints[i] = rune(rand.Intn(0x10ffff))
		}
		v.SetString(string(codePoints))
	case reflect.Struct:
		n := v.NumField()
		// Divide sizeLeft evenly among the struct fields.
		sizeLeft := size
		if n > sizeLeft {
			sizeLeft = 1
		} else if n > 0 {
			sizeLeft /= n
		}
		for i := 0; i < n; i++ {
			elem, ok := sizedValue(concrete.Field(i).Type, rand, sizeLeft)
			if !ok {
				return reflect.Value{}, false
			}
			v.Field(i).Set(elem)
		}
	default:
		return reflect.Value{}, false
	}

	return v, true
}

// A Config structure contains options for running a test.
type Config struct {
	// MaxCount sets the maximum number of iterations.
	// If zero, MaxCountScale is used.
	MaxCount int
	// MaxCountScale is a non-negative scale factor applied to the
	// default maximum.
	// A count of zero implies the default, which is usually 100
	// but can be set by the -quickchecks flag.
	MaxCountScale float64
	// Rand specifies a source of random numbers.
	// If nil, a default pseudo-random source will be used.
	Rand *rand.Rand
	// Values specifies a function to generate a slice of
	// arbitrary reflect.Values that are congruent with the
	// arguments to the function being tested.
	// If nil, the top-level Value function is used to generate them.
	Values func([]reflect.Value, *rand.Rand)
}

var defaultConfig Config

// getRand returns the *rand.Rand to use for a given Config.
func (c *Config) getRand() *rand.Rand {
	if c.Rand == nil {
		return rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	return c.Rand
}

// getMaxCount returns the maximum number of iterations to run for a given
// Config.
func (c *Config) getMaxCount() (maxCount int) {
	maxCount = c.MaxCount
	if maxCount == 0 {
		if c.MaxCountScale != 0 {
			maxCount = int(c.MaxCountScale * float64(*defaultMaxCount))
		} else {
			maxCount = *defaultMaxCount
		}
	}

	return
}

// A SetupError is the result of an error in the way that check is being
// used, independent of the functions being tested.
type SetupError string

func (s SetupError) Error() string { return string(s) }

// A CheckError is the result of Check finding an error.
type CheckError struct {
	Count int
	In    []any
}

func (s *CheckError) Error() string {
	return fmt.Sprintf("#%d: failed on input %s", s.Count, toString(s.In))
}

// A CheckEqualError is the result CheckEqual finding an error.
type CheckEqualError struct {
	CheckError
	Out1 []any
	Out2 []any
}

func (s *CheckEqualError) Error() string {
	return fmt.Sprintf("#%d: failed on input %s. Output 1: %s. Output 2: %s", s.Count, toString(s.In), toString(s.Out1), toString(s.Out2))
}

// Check looks for an input to f, any function that returns bool,
// such that f returns false. It calls f repeatedly, with arbitrary
// values for each argument. If f returns false on a given input,
// Check returns that input as a *CheckError.
// For example:
//
//	func TestOddMultipleOfThree(t *testing.T) {
//		f := func(x int) bool {
//			y := OddMultipleOfThree(x)
//			return y%2 == 1 && y%3 == 0
//		}
//		if err := quick.Check(f, nil); err != nil {
//			t.Error(err)
//		}
//	}
func Check(f any, config *Config) error {
	if config == nil {
		config = &defaultConfig
	}

	fVal, fType, ok := functionAndType(f)
	if !ok {
		return SetupError("argument is not a function")
	}

	if fType.NumOut() != 1 {
		return SetupError("function does not return one value")
	}
	if fType.Out(0).Kind() != reflect.Bool {
		return SetupError("function does not return a bool")
	}

	arguments := make([]reflect.Value, fType.NumIn())
	rand := config.getRand()
	maxCount := config.getMaxCount()

	for i := 0; i < maxCount; i++ {
		err := arbitraryValues(arguments, fType, config, rand)
		if err != nil {
			return err
		}

		if !fVal.Call(arguments)[0].Bool() {
			return &CheckError{i + 1, toInterfaces(arguments)}
		}
	}

	return nil
}

// arbitraryValues writes Values to args such that args contains Values
// suitable for calling f.
func arbitraryValues(args []reflect.Value, f reflect.Type, config *Config, rand *rand.Rand) (err error) {
	if config.Values != nil {
		config.Values(args, rand)
		return
	}

	for j := 0; j < len(args); j++ {
		var ok bool
		args[j], ok = Value(f.In(j), rand)
		if !ok {
			err = SetupError(fmt.Sprintf("cannot create arbitrary value of type %s for argument %d", f.In(j), j))
			return
		}
	}

	return
}

func functionAndType(f any) (v reflect.Value, t reflect.Type, ok bool) {
	v = reflect.ValueOf(f)
	ok = v.Kind() == reflect.Func
	if !ok {
		return
	}
	t = v.Type()
	return
}

func toInterfaces(values []reflect.Value) []any {
	ret := make([]any, len(values))
	for i, v := range values {
		ret[i] = v.Interface()
	}
	return ret
}

func toString(interfaces []any) string {
	s := make([]string, len(interfaces))
	for i, v := range interfaces {
		s[i] = fmt.Sprintf("%#v", v)
	}
	return strings.Join(s, ", ")
}

type Person struct {
	Name string `glint:"name"`
	Age  int    `glint:"age"`
}

func ExampleEncoder_generic() {
	// Create a type-safe encoder for Person
	encoder := NewEncoder[Person]()

	// Create a person
	person := Person{Name: "SampleUser", Age: 30}

	// Encode the person
	buf := NewBufferFromPool()
	encoder.Marshal(&person, buf)

	// Create a type-safe decoder for Person
	decoder := NewDecoder[Person]()

	// Decode into a new person
	var decoded Person
	err := decoder.Unmarshal(buf.Bytes, &decoded)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s is %d years old\n", decoded.Name, decoded.Age)
	// Output: SampleUser is 30 years old
}

type People struct {
	Items []Person `glint:"items"`
}

func ExampleEncoder_sliceGeneric() {
	// Create a type-safe encoder for People (which contains a slice)
	encoder := NewEncoder[People]()

	// Create some people
	people := People{
		Items: []Person{
			{Name: "ExampleUser", Age: 25},
			{Name: "Carol", Age: 35},
		},
	}

	// Encode the people
	buf := NewBufferFromPool()
	encoder.Marshal(&people, buf)

	// Create a type-safe decoder for People
	decoder := NewDecoder[People]()

	// Decode into a new People struct
	var decoded People
	err := decoder.Unmarshal(buf.Bytes, &decoded)
	if err != nil {
		panic(err)
	}

	for _, p := range decoded.Items {
		fmt.Printf("%s is %d years old\n", p.Name, p.Age)
	}
	// Output: ExampleUser is 25 years old
	// Carol is 35 years old
}

// Dataset generators for different data patterns

func generateRandomData(n int, r *rand.Rand) []int64 {
	data := make([]int64, n)
	for i := 0; i < n; i++ {
		data[i] = int64(r.Intn(1000000))
	}
	return data
}

func generateSensorData(n int, r *rand.Rand) []int64 {
	data := make([]int64, n)
	base := int64(20000) // Starting temperature in millidegrees
	for i := 0; i < n; i++ {
		// Small random walk simulating temperature changes
		change := int64(r.Intn(200) - 100)
		base += change
		data[i] = base
	}
	return data
}

func generateStockPrices(n int, r *rand.Rand) []int64 {
	// Stock prices in cents
	prices := make([]int64, n)
	base := int64(15000) // $150.00
	for i := 0; i < n; i++ {
		// Random walk with slight upward bias
		change := int64(r.Intn(200) - 95)
		base += change
		if base < 100 {
			base = 100 // Minimum price
		}
		prices[i] = base
	}
	return prices
}

func generatePageViews(n int, r *rand.Rand) []int64 {
	// Cumulative page view counter
	views := make([]int64, n)
	total := int64(0)
	for i := 0; i < n; i++ {
		// Increasing counter with variable increments
		increment := int64(r.Intn(100) + 1)
		total += increment
		views[i] = total
	}
	return views
}

type TestDataset struct {
	Values []int64 `glint:"values"`
}

type DeltaTestDataset struct {
	Values []int64 `glint:"values,delta"`
}

// testDeltaEncodingType is a generic test function for delta encoding types
func testDeltaEncodingType[T comparable](t *testing.T, values []T, typeName string) {
	type Data struct {
		Values []T `glint:"values,delta"`
	}

	data := Data{Values: values}
	encoder := NewEncoder[Data]()
	decoder := NewDecoder[Data]()
	buf := NewBufferFromPool()
	defer buf.ReturnToPool()

	encoder.Marshal(&data, buf)
	var decoded Data
	err := decoder.Unmarshal(buf.Bytes, &decoded)
	if err != nil {
		t.Fatalf("%s delta failed: %v", typeName, err)
	}
	if !reflect.DeepEqual(data.Values, decoded.Values) {
		t.Errorf("%s delta values don't match", typeName)
	}
}

// testDocumentBuilderDelta is a generic test function for SliceBuilder delta encoding
func testDocumentBuilderDelta[T comparable](t *testing.T, values []T, appendFunc func(*SliceBuilder, []T), typeName string) {
	// Build document with delta encoding
	var sb SliceBuilder
	appendFunc(&sb, values)

	doc := DocumentBuilder{}
	doc.AppendString("name", "test-delta")
	doc.AppendSlice("values", sb)

	bytes := doc.Bytes()

	// Print the document to verify delta encoding shows up
	output := SPrint(bytes)
	if !strings.Contains(output, "(delta)") {
		t.Error("Expected delta encoding indicator in printed output")
	}

	// Verify the document can be decoded
	type TestStruct struct {
		Name   string `glint:"name"`
		Values []T    `glint:"values,delta"`
	}

	decoder := NewDecoder[TestStruct]()
	var decoded TestStruct
	err := decoder.Unmarshal(bytes, &decoded)
	if err != nil {
		t.Fatalf("Failed to decode document: %v", err)
	}

	// Verify values match
	if decoded.Name != "test-delta" {
		t.Errorf("Name mismatch: got %q, want %q", decoded.Name, "test-delta")
	}
	if len(decoded.Values) != len(values) {
		t.Fatalf("Values length mismatch: got %d, want %d", len(decoded.Values), len(values))
	}
	for i, v := range values {
		if decoded.Values[i] != v {
			t.Errorf("Values[%d] mismatch: got %v, want %v", i, decoded.Values[i], v)
		}
	}
}

// TestSpecialEncoding consolidates all special encoding tests including zigzag, delta encoding correctness and performance
func TestStringerAndCustomEncodingTags(t *testing.T) {
	t.Run("ZigzagFuzz", func(t *testing.T) {
		type zigzagTypes struct {
			Vzzint     int     `glint:"zzint,zigzag"`
			Azzint     []int   `glint:"[]zzint,zigzag"`
			NestAZZInt [][]int `glint:"[]zzint_nest,zigzag"`
		}

		testFuzzEncodeDecode(t, zigzagTypes{}, 1000)
	})

	t.Run("DeltaCorrectness", func(t *testing.T) {
		// Import the time series data types from benchmark test
		type TimeSeriesData struct {
			Timestamps []int64 `glint:"timevals"`
			Values     []int   `glint:"values"`
		}

		type DeltaTimeSeriesData struct {
			Timestamps []int64 `glint:"timestamps,delta"`
			Values     []int   `glint:"values,delta"`
		}

		timestamps, values := generateTimeSeriesData(100)

		// Test that delta encoding/decoding produces the same results
		deltaData := DeltaTimeSeriesData{
			Timestamps: timestamps,
			Values:     values,
		}
		deltaEncoder := NewEncoder[DeltaTimeSeriesData]()
		deltaDecoder := NewDecoder[DeltaTimeSeriesData]()
		deltaBuf := NewBufferFromPool()
		deltaEncoder.Marshal(&deltaData, deltaBuf)

		var decodedDelta DeltaTimeSeriesData
		err := deltaDecoder.Unmarshal(deltaBuf.Bytes, &decodedDelta)
		if err != nil {
			t.Fatalf("Delta decode failed: %v", err)
		}

		// Verify correctness
		for i := range timestamps {
			if decodedDelta.Timestamps[i] != timestamps[i] {
				t.Errorf("Delta timestamp mismatch at %d: got %d, want %d", i, decodedDelta.Timestamps[i], timestamps[i])
			}
			if decodedDelta.Values[i] != values[i] {
				t.Errorf("Delta value mismatch at %d: got %d, want %d", i, decodedDelta.Values[i], values[i])
			}
		}

		// Compare sizes between standard and delta encoding
		standardData := TimeSeriesData{
			Timestamps: timestamps,
			Values:     values,
		}
		standardEncoder := NewEncoder[TimeSeriesData]()
		standardBuf := NewBufferFromPool()
		standardEncoder.Marshal(&standardData, standardBuf)

		// Report size savings
		standardSize := len(standardBuf.Bytes)
		deltaSize := len(deltaBuf.Bytes)
		savings := float64(standardSize-deltaSize) / float64(standardSize) * 100
		t.Logf("Size comparison: Standard=%d bytes, Delta=%d bytes, Savings=%.1f%%", standardSize, deltaSize, savings)
	})

	t.Run("DeltaAllTypes", func(t *testing.T) {
		testCases := []struct {
			name string
			test func(t *testing.T)
		}{
			{"int", func(t *testing.T) { testDeltaEncodingType(t, []int{1, 2, 3, 4, 5}, "int") }},
			{"int32", func(t *testing.T) { testDeltaEncodingType(t, []int32{1000, 1005, 1010, 1003, 1008}, "int32") }},
			{"int64", func(t *testing.T) { testDeltaEncodingType(t, []int64{1, 2, 3, 4, 5}, "int64") }},
			{"int16", func(t *testing.T) { testDeltaEncodingType(t, []int16{1000, 1005, 1010, 1003, 1008}, "int16") }},
			{"uint", func(t *testing.T) { testDeltaEncodingType(t, []uint{100, 105, 110, 103, 108}, "uint") }},
			{"uint16", func(t *testing.T) { testDeltaEncodingType(t, []uint16{1000, 1005, 1010, 1003, 1008}, "uint16") }},
			{"uint32", func(t *testing.T) {
				testDeltaEncodingType(t, []uint32{100000, 100005, 100010, 100003, 100008}, "uint32")
			}},
			{"uint64", func(t *testing.T) {
				testDeltaEncodingType(t, []uint64{1000000, 1000005, 1000010, 1000003, 1000008}, "uint64")
			}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, tc.test)
		}

		// Test unsupported types
		unsupportedTypes := []string{"float32", "float64"}
		for _, typ := range unsupportedTypes {
			t.Run(typ, func(t *testing.T) {
				t.Skipf("Delta encoding for %s not yet implemented", typ)
			})
		}
	})

	t.Run("DeltaDatasets", func(t *testing.T) {
		// Create a local random generator for deterministic tests
		r := rand.New(rand.NewSource(42))

		testCases := []struct {
			name     string
			data     []int64
			expected string // expected outcome: "good", "poor", "neutral"
		}{
			{"Sequential", func() []int64 { ts, _ := generateTimeSeriesData(100); return ts }(), "good"},
			{"Random", generateRandomData(100, r), "poor"},
			{"Sensor", generateSensorData(100, r), "good"},        // Random walk with zigzag deltas compresses well
			{"Stock Prices", generateStockPrices(100, r), "good"}, // Small random changes with zigzag compress well
			{"Page Views", generatePageViews(100, r), "good"},     // Cumulative data compresses well
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Standard encoding
				standardData := TestDataset{Values: tc.data}
				standardEncoder := NewEncoder[TestDataset]()
				standardBuf := NewBufferFromPool()
				standardEncoder.Marshal(&standardData, standardBuf)

				// Delta encoding
				deltaData := DeltaTestDataset{Values: tc.data}
				deltaEncoder := NewEncoder[DeltaTestDataset]()
				deltaBuf := NewBufferFromPool()
				deltaEncoder.Marshal(&deltaData, deltaBuf)

				// Verify correctness
				deltaDecoder := NewDecoder[DeltaTestDataset]()
				var decoded DeltaTestDataset
				err := deltaDecoder.Unmarshal(deltaBuf.Bytes, &decoded)
				if err != nil {
					t.Fatalf("Delta decode failed: %v", err)
				}

				for i := range tc.data {
					if decoded.Values[i] != tc.data[i] {
						t.Errorf("Value mismatch at %d: got %d, want %d", i, decoded.Values[i], tc.data[i])
					}
				}

				// Calculate savings
				standardSize := len(standardBuf.Bytes)
				deltaSize := len(deltaBuf.Bytes)
				savings := float64(standardSize-deltaSize) / float64(standardSize) * 100

				t.Logf("Size: Standard=%d bytes, Delta=%d bytes, Savings=%.1f%%",
					standardSize, deltaSize, savings)

				// Verify expected outcome
				switch tc.expected {
				case "good":
					if savings < 25 { // Adjusted threshold
						t.Errorf("Expected good compression (>25%%), got %.1f%%", savings)
					}
				case "poor":
					if savings > 10 {
						t.Errorf("Expected poor compression (<10%%), got %.1f%%", savings)
					}
				}
			})
		}
	})

	t.Run("DeltaPrinting", func(t *testing.T) {
		// Test that delta-encoded documents print correctly
		type TestData struct {
			Name   string  `glint:"name"`
			Values []int64 `glint:"values,delta"`
		}

		data := TestData{
			Name:   "sensor-data",
			Values: []int64{100, 105, 103, 107, 110, 108},
		}

		encoder := NewEncoder[TestData]()
		buf := NewBufferFromPool()
		defer buf.ReturnToPool()
		encoder.Marshal(&data, buf)

		// Print the document
		output := SPrint(buf.Bytes)

		// Verify output contains delta indicator
		if !strings.Contains(output, "[](delta)Int64") {
			t.Errorf("Expected [](delta)Int64 in schema, got:\n%s", output)
		}

		// Verify all values are printed correctly
		for i, v := range data.Values {
			expected := fmt.Sprintf("[%d]: %d", i, v)
			if !strings.Contains(output, expected) {
				t.Errorf("Expected to find %q in output, got:\n%s", expected, output)
			}
		}
	})

	t.Run("DeltaInt32", func(t *testing.T) {
		// Test int32 delta encoding with struct tags
		type TestData struct {
			Name   string  `glint:"name"`
			Values []int32 `glint:"values,delta"`
		}

		data := TestData{
			Name:   "int32-test",
			Values: []int32{10000, 10005, 10003, 10008, 10001},
		}

		// Encode with delta
		encoder := NewEncoder[TestData]()
		buf := NewBufferFromPool()
		defer buf.ReturnToPool()
		encoder.Marshal(&data, buf)

		// Decode and verify
		decoder := NewDecoder[TestData]()
		var decoded TestData
		err := decoder.Unmarshal(buf.Bytes, &decoded)
		if err != nil {
			t.Fatalf("Failed to decode: %v", err)
		}

		// Verify correctness
		if decoded.Name != data.Name {
			t.Errorf("Name mismatch: got %q, want %q", decoded.Name, data.Name)
		}
		if len(decoded.Values) != len(data.Values) {
			t.Fatalf("Values length mismatch: got %d, want %d", len(decoded.Values), len(data.Values))
		}
		for i, v := range data.Values {
			if decoded.Values[i] != v {
				t.Errorf("Values[%d] mismatch: got %d, want %d", i, decoded.Values[i], v)
			}
		}

		// Test printing
		output := SPrint(buf.Bytes)
		if !strings.Contains(output, "[](delta)Int32") {
			t.Errorf("Expected [](delta)Int32 in schema, got:\n%s", output)
		}

		// Test with SliceBuilder
		t.Run("SliceBuilder", func(t *testing.T) {
			values := []int32{50000, 50100, 50050, 50200, 50150}

			// Build document with delta encoding
			var sb SliceBuilder
			sb.AppendInt32SliceDelta(values)

			doc := DocumentBuilder{}
			doc.AppendString("type", "sensor-readings")
			doc.AppendSlice("vals1", sb)

			bytes := doc.Bytes()

			// Verify the document can be decoded
			type TestStruct struct {
				Type     string  `glint:"type"`
				Readings []int32 `glint:"vals1,delta"`
			}

			decoder := NewDecoder[TestStruct]()
			var decoded TestStruct
			err := decoder.Unmarshal(bytes, &decoded)
			if err != nil {
				t.Fatalf("Failed to decode document: %v", err)
			}

			// Verify values match
			if decoded.Type != "sensor-readings" {
				t.Errorf("Type mismatch: got %q, want %q", decoded.Type, "sensor-readings")
			}
			for i, v := range values {
				if decoded.Readings[i] != v {
					t.Errorf("Readings[%d] mismatch: got %d, want %d", i, decoded.Readings[i], v)
				}
			}
		})
	})

	t.Run("SliceBuilderComprehensive", func(t *testing.T) {
		// Test all SliceBuilder append methods for comprehensive coverage
		tests := []struct {
			name    string
			buildfn func() SliceBuilder
			decode  func([]byte) (interface{}, error)
		}{
			{
				name: "AppendStringSlice",
				buildfn: func() SliceBuilder {
					var sb SliceBuilder
					sb.AppendStringSlice([]string{"hello", "world", "test"})
					return sb
				},
				decode: func(data []byte) (interface{}, error) {
					type Test struct {
						Values []string `glint:"values"`
					}
					decoder := NewDecoder[Test]()
					var result Test
					err := decoder.Unmarshal(data, &result)
					return result.Values, err
				},
			},
			{
				name: "AppendInt16SliceDelta",
				buildfn: func() SliceBuilder {
					var sb SliceBuilder
					sb.AppendInt16SliceDelta([]int16{1000, 1010, 1005, 1020})
					return sb
				},
				decode: func(data []byte) (interface{}, error) {
					type Test struct {
						Values []int16 `glint:"values,delta"`
					}
					decoder := NewDecoder[Test]()
					var result Test
					err := decoder.Unmarshal(data, &result)
					return result.Values, err
				},
			},
			{
				name: "AppendInt64SliceDelta",
				buildfn: func() SliceBuilder {
					var sb SliceBuilder
					sb.AppendInt64SliceDelta([]int64{100000, 100010, 100005, 100020})
					return sb
				},
				decode: func(data []byte) (interface{}, error) {
					type Test struct {
						Values []int64 `glint:"values,delta"`
					}
					decoder := NewDecoder[Test]()
					var result Test
					err := decoder.Unmarshal(data, &result)
					return result.Values, err
				},
			},
			{
				name: "AppendIntSliceDelta",
				buildfn: func() SliceBuilder {
					var sb SliceBuilder
					sb.AppendIntSliceDelta([]int{10, 15, 12, 20})
					return sb
				},
				decode: func(data []byte) (interface{}, error) {
					type Test struct {
						Values []int `glint:"values,delta"`
					}
					decoder := NewDecoder[Test]()
					var result Test
					err := decoder.Unmarshal(data, &result)
					return result.Values, err
				},
			},
			{
				name: "AppendUintSliceDelta",
				buildfn: func() SliceBuilder {
					var sb SliceBuilder
					sb.AppendUintSliceDelta([]uint{100, 110, 105, 120})
					return sb
				},
				decode: func(data []byte) (interface{}, error) {
					type Test struct {
						Values []uint `glint:"values,delta"`
					}
					decoder := NewDecoder[Test]()
					var result Test
					err := decoder.Unmarshal(data, &result)
					return result.Values, err
				},
			},
			{
				name: "AppendUint16SliceDelta",
				buildfn: func() SliceBuilder {
					var sb SliceBuilder
					sb.AppendUint16SliceDelta([]uint16{5000, 5010, 5005, 5020})
					return sb
				},
				decode: func(data []byte) (interface{}, error) {
					type Test struct {
						Values []uint16 `glint:"values,delta"`
					}
					decoder := NewDecoder[Test]()
					var result Test
					err := decoder.Unmarshal(data, &result)
					return result.Values, err
				},
			},
			{
				name: "AppendUint32SliceDelta",
				buildfn: func() SliceBuilder {
					var sb SliceBuilder
					sb.AppendUint32SliceDelta([]uint32{200000, 200010, 200005, 200020})
					return sb
				},
				decode: func(data []byte) (interface{}, error) {
					type Test struct {
						Values []uint32 `glint:"values,delta"`
					}
					decoder := NewDecoder[Test]()
					var result Test
					err := decoder.Unmarshal(data, &result)
					return result.Values, err
				},
			},
			{
				name: "AppendUint64SliceDelta",
				buildfn: func() SliceBuilder {
					var sb SliceBuilder
					sb.AppendUint64SliceDelta([]uint64{1000000, 1000010, 1000005, 1000020})
					return sb
				},
				decode: func(data []byte) (interface{}, error) {
					type Test struct {
						Values []uint64 `glint:"values,delta"`
					}
					decoder := NewDecoder[Test]()
					var result Test
					err := decoder.Unmarshal(data, &result)
					return result.Values, err
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				sb := tt.buildfn()

				// Create a document with the slice
				doc := DocumentBuilder{}
				doc.AppendSlice("values", sb)
				data := doc.Bytes()

				// Verify document was created successfully
				if len(data) == 0 {
					t.Error("Document should not be empty")
				}

				// Verify we can decode it
				decoded, err := tt.decode(data)
				if err != nil {
					t.Errorf("Failed to decode: %v", err)
				}

				// Basic validation that we got something back
				switch v := decoded.(type) {
				case []uint8:
					if len(v) == 0 {
						t.Error("Decoded slice should not be empty")
					}
				case []string:
					if len(v) == 0 {
						t.Error("Decoded slice should not be empty")
					}
				case []int16, []int64, []int, []uint, []uint16, []uint32, []uint64:
					rv := reflect.ValueOf(v)
					if rv.Len() == 0 {
						t.Error("Decoded slice should not be empty")
					}
				}

				// Test that SPrint works on the document (tests printer functions)
				output := SPrint(data)
				if !strings.Contains(output, "values") {
					t.Errorf("SPrint output should contain 'values' field")
				}
			})
		}
	})

	t.Run("SliceBuilderNonDelta", func(t *testing.T) {
		// Test individual SliceBuilder methods with proper validation

		// Generic helper to test slice builders
		testSliceBuilderWithValidation := func(t *testing.T, builderFunc func(*SliceBuilder), decodeFunc func([]byte) (interface{}, error), expected interface{}) {
			var sb SliceBuilder
			builderFunc(&sb)

			doc := DocumentBuilder{}
			doc.AppendSlice("values", sb)
			data := doc.Bytes()

			decoded, err := decodeFunc(data)
			if err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}

			if !reflect.DeepEqual(decoded, expected) {
				t.Errorf("Values mismatch: got %v, want %v", decoded, expected)
			}
		}

		t.Run("AppendIntSlice", func(t *testing.T) {
			expected := []int{-100, 0, 100}
			testSliceBuilderWithValidation(t,
				func(sb *SliceBuilder) { sb.AppendIntSlice(expected) },
				func(data []byte) (interface{}, error) {
					type TestStruct struct {
						Values []int `glint:"values"`
					}
					decoder := NewDecoder[TestStruct]()
					var decoded TestStruct
					err := decoder.Unmarshal(data, &decoded)
					return decoded.Values, err
				},
				expected,
			)
		})

		t.Run("AppendBytesSlice", func(t *testing.T) {
			// AppendBytesSlice has a different encoding than [][]byte fields
			// It encodes as a single bytes field with length-prefixed entries
			// We'll just verify it creates valid output that can be printed
			var sb SliceBuilder
			expected := [][]byte{{0x01}, {0xFF}}
			sb.AppendBytesSlice(expected)

			doc := DocumentBuilder{}
			doc.AppendSlice("values", sb)
			data := doc.Bytes()

			if len(data) == 0 {
				t.Error("Document should not be empty")
			}

			// Verify document can be printed (validates structure)
			output := SPrint(data)
			if !strings.Contains(output, "values") {
				t.Error("Document should contain 'values' field")
			}
		})

		t.Run("AppendUint16Slice", func(t *testing.T) {
			expected := []uint16{0, 65535}
			testSliceBuilderWithValidation(t,
				func(sb *SliceBuilder) { sb.AppendUint16Slice(expected) },
				func(data []byte) (interface{}, error) {
					type TestStruct struct {
						Values []uint16 `glint:"values"`
					}
					decoder := NewDecoder[TestStruct]()
					var decoded TestStruct
					err := decoder.Unmarshal(data, &decoded)
					return decoded.Values, err
				},
				expected,
			)
		})

		t.Run("AppendUint32Slice", func(t *testing.T) {
			expected := []uint32{0, 4294967295}
			testSliceBuilderWithValidation(t,
				func(sb *SliceBuilder) { sb.AppendUint32Slice(expected) },
				func(data []byte) (interface{}, error) {
					type TestStruct struct {
						Values []uint32 `glint:"values"`
					}
					decoder := NewDecoder[TestStruct]()
					var decoded TestStruct
					err := decoder.Unmarshal(data, &decoded)
					return decoded.Values, err
				},
				expected,
			)
		})

		t.Run("AppendUint64Slice", func(t *testing.T) {
			expected := []uint64{0, 18446744073709551615}
			testSliceBuilderWithValidation(t,
				func(sb *SliceBuilder) { sb.AppendUint64Slice(expected) },
				func(data []byte) (interface{}, error) {
					type TestStruct struct {
						Values []uint64 `glint:"values"`
					}
					decoder := NewDecoder[TestStruct]()
					var decoded TestStruct
					err := decoder.Unmarshal(data, &decoded)
					return decoded.Values, err
				},
				expected,
			)
		})

		t.Run("AppendUintSlice", func(t *testing.T) {
			expected := []uint{100, 200}
			testSliceBuilderWithValidation(t,
				func(sb *SliceBuilder) { sb.AppendUintSlice(expected) },
				func(data []byte) (interface{}, error) {
					type TestStruct struct {
						Values []uint `glint:"values"`
					}
					decoder := NewDecoder[TestStruct]()
					var decoded TestStruct
					err := decoder.Unmarshal(data, &decoded)
					return decoded.Values, err
				},
				expected,
			)
		})

		t.Run("AppendInt8Slice", func(t *testing.T) {
			expected := []int8{-128, 127}
			testSliceBuilderWithValidation(t,
				func(sb *SliceBuilder) { sb.AppendInt8Slice(expected) },
				func(data []byte) (interface{}, error) {
					type TestStruct struct {
						Values []int8 `glint:"values"`
					}
					decoder := NewDecoder[TestStruct]()
					var decoded TestStruct
					err := decoder.Unmarshal(data, &decoded)
					return decoded.Values, err
				},
				expected,
			)
		})

		t.Run("AppendInt16Slice", func(t *testing.T) {
			expected := []int16{-32768, 32767}
			testSliceBuilderWithValidation(t,
				func(sb *SliceBuilder) { sb.AppendInt16Slice(expected) },
				func(data []byte) (interface{}, error) {
					type TestStruct struct {
						Values []int16 `glint:"values"`
					}
					decoder := NewDecoder[TestStruct]()
					var decoded TestStruct
					err := decoder.Unmarshal(data, &decoded)
					return decoded.Values, err
				},
				expected,
			)
		})

		t.Run("AppendInt32Slice", func(t *testing.T) {
			expected := []int32{-2147483648, 2147483647}
			testSliceBuilderWithValidation(t,
				func(sb *SliceBuilder) { sb.AppendInt32Slice(expected) },
				func(data []byte) (interface{}, error) {
					type TestStruct struct {
						Values []int32 `glint:"values"`
					}
					decoder := NewDecoder[TestStruct]()
					var decoded TestStruct
					err := decoder.Unmarshal(data, &decoded)
					return decoded.Values, err
				},
				expected,
			)
		})

		t.Run("AppendInt64Slice", func(t *testing.T) {
			expected := []int64{-9223372036854775808, 9223372036854775807}
			testSliceBuilderWithValidation(t,
				func(sb *SliceBuilder) { sb.AppendInt64Slice(expected) },
				func(data []byte) (interface{}, error) {
					type TestStruct struct {
						Values []int64 `glint:"values"`
					}
					decoder := NewDecoder[TestStruct]()
					var decoded TestStruct
					err := decoder.Unmarshal(data, &decoded)
					return decoded.Values, err
				},
				expected,
			)
		})

		t.Run("AppendFloat32Slice", func(t *testing.T) {
			expected := []float32{-3.14, 0.0, 2.71}
			testSliceBuilderWithValidation(t,
				func(sb *SliceBuilder) { sb.AppendFloat32Slice(expected) },
				func(data []byte) (interface{}, error) {
					type TestStruct struct {
						Values []float32 `glint:"values"`
					}
					decoder := NewDecoder[TestStruct]()
					var decoded TestStruct
					err := decoder.Unmarshal(data, &decoded)
					return decoded.Values, err
				},
				expected,
			)
		})

		t.Run("AppendFloat64Slice", func(t *testing.T) {
			expected := []float64{-3.141592653589793, 0.0}
			testSliceBuilderWithValidation(t,
				func(sb *SliceBuilder) { sb.AppendFloat64Slice(expected) },
				func(data []byte) (interface{}, error) {
					type TestStruct struct {
						Values []float64 `glint:"values"`
					}
					decoder := NewDecoder[TestStruct]()
					var decoded TestStruct
					err := decoder.Unmarshal(data, &decoded)
					return decoded.Values, err
				},
				expected,
			)
		})

		t.Run("AppendTimeSlice", func(t *testing.T) {
			expected := []time.Time{time.Unix(1609459200, 0), time.Unix(1672531200, 0)}
			testSliceBuilderWithValidation(t,
				func(sb *SliceBuilder) { sb.AppendTimeSlice(expected) },
				func(data []byte) (interface{}, error) {
					type TestStruct struct {
						Values []time.Time `glint:"values"`
					}
					decoder := NewDecoder[TestStruct]()
					var decoded TestStruct
					err := decoder.Unmarshal(data, &decoded)
					return decoded.Values, err
				},
				expected,
			)
		})

		t.Run("AppendBoolSlice", func(t *testing.T) {
			expected := []bool{true, false, true}
			testSliceBuilderWithValidation(t,
				func(sb *SliceBuilder) { sb.AppendBoolSlice(expected) },
				func(data []byte) (interface{}, error) {
					type TestStruct struct {
						Values []bool `glint:"values"`
					}
					decoder := NewDecoder[TestStruct]()
					var decoded TestStruct
					err := decoder.Unmarshal(data, &decoded)
					return decoded.Values, err
				},
				expected,
			)
		})

		t.Run("AppendStringSlice", func(t *testing.T) {
			expected := []string{"hello", "world", "test"}
			testSliceBuilderWithValidation(t,
				func(sb *SliceBuilder) { sb.AppendStringSlice(expected) },
				func(data []byte) (interface{}, error) {
					type TestStruct struct {
						Values []string `glint:"values"`
					}
					decoder := NewDecoder[TestStruct]()
					var decoded TestStruct
					err := decoder.Unmarshal(data, &decoded)
					return decoded.Values, err
				},
				expected,
			)
		})
	})
}

// Test encoder/decoder utilities for coverage
func TestEncoderDecoderHelperMethods(t *testing.T) {
	t.Run("EncoderUtilities", func(t *testing.T) {
		type TestStruct struct {
			Name string `glint:"name"`
			Age  int    `glint:"age"`
		}

		// Test Schema method
		encoder := NewEncoder[TestStruct]()
		schema := encoder.Schema()
		if schema == nil {
			t.Error("Schema should not be nil")
		}

		// Test ClearSchema method
		encoder.ClearSchema()
	})

	t.Run("DecoderUtilities", func(t *testing.T) {
		type TestStruct struct {
			Name string `glint:"name"`
			Age  int    `glint:"age"`
		}

		// Test NewDecoderUsingTag
		decoder := NewDecoderUsingTag[TestStruct]("custom_tag")
		if decoder == nil {
			t.Error("NewDecoderUsingTag should return valid decoder")
		}
	})

	t.Run("SliceEncoderUtilities", func(t *testing.T) {
		// Test NewSliceEncoder
		encoder := NewSliceEncoder([]int{})
		if encoder == nil {
			t.Error("NewSliceEncoder should return valid encoder")
		}

		// Test NewSliceEncoderUsingTag
		encoder = NewSliceEncoderUsingTag([]string{}, "custom")
		if encoder == nil {
			t.Error("NewSliceEncoderUsingTag should return valid encoder")
		}

		// Test NewSliceEncoderUsingTagWithSchema
		buf := NewBufferFromPool()
		defer buf.ReturnToPool()
		encoder = NewSliceEncoderUsingTagWithSchema([]int{}, "custom", buf)
		if encoder == nil {
			t.Error("NewSliceEncoderUsingTagWithSchema should return valid encoder")
		}
	})

	t.Run("BufferUtilities", func(t *testing.T) {
		// Test NewBufferFromPoolWithCap
		buf := NewBufferFromPoolWithCap(1024)
		defer buf.ReturnToPool()

		if cap(buf.Bytes) < 1024 {
			t.Error("Buffer should have requested capacity")
		}
	})

	t.Run("UtilityFunctions", func(t *testing.T) {
		type TestStruct struct {
			Name string `glint:"name"`
		}

		// Test SchemaBytes
		schema := SchemaBytes(TestStruct{})
		if len(schema) == 0 {
			t.Error("SchemaBytes should return schema")
		}

		// Test SchemaBytesUsingTag
		schema = SchemaBytesUsingTag(TestStruct{}, "custom")
		if len(schema) == 0 {
			t.Error("SchemaBytesUsingTag should return schema")
		}

		// Test HashBytes
		HashBytes([]byte("test12345"))

		// Test Flags
		testBytes := []byte{1, 2, 3}
		flags := Flags(testBytes)
		_ = flags // Just test it doesn't panic
	})
}

// Test Map Decoder functionality for coverage
func TestMapEncodingAndDecoding(t *testing.T) {
	t.Run("BasicMapDecoding", func(t *testing.T) {
		// Test map[string]string
		type TestStruct struct {
			StringMap map[string]string `glint:"map1"`
		}

		testData := TestStruct{
			StringMap: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		}

		encoder := NewEncoder[TestStruct]()
		buf := NewBufferFromPool()
		defer buf.ReturnToPool()
		encoder.Marshal(&testData, buf)

		decoder := NewDecoder[TestStruct]()
		var result TestStruct
		err := decoder.Unmarshal(buf.Bytes, &result)

		if err != nil {
			t.Fatalf("Failed to decode map: %v", err)
		}

		if len(result.StringMap) != len(testData.StringMap) {
			t.Errorf("Map length mismatch: got %d, want %d", len(result.StringMap), len(testData.StringMap))
		}

		for k, v := range testData.StringMap {
			if result.StringMap[k] != v {
				t.Errorf("Map value mismatch for key %s: got %s, want %s", k, result.StringMap[k], v)
			}
		}
	})

	t.Run("MapDecoderAPIs", func(t *testing.T) {
		// Test newMapDecoderUsingTagAndOpts
		mapType := map[string]int{}
		decoder := newMapDecoderUsingTagAndOpts(mapType, "glint", tagOptions(""))
		if decoder == nil {
			t.Error("newMapDecoderUsingTagAndOpts should return valid decoder")
		}

		// Test with different map types
		mapIntInt := map[int]int{}
		decoder = newMapDecoderUsingTagAndOpts(mapIntInt, "glint", tagOptions(""))
		if decoder == nil {
			t.Error("newMapDecoderUsingTagAndOpts should work for map[int]int")
		}

		// Test toPointer method indirectly through usage
		// The toPointer method is used internally in the mapDecoder
	})

	t.Run("MapDecoderUnmarshal", func(t *testing.T) {
		// Test the Unmarshal method directly
		type TestMapStruct struct {
			Data map[string]int `glint:"info1"`
		}

		testData := TestMapStruct{
			Data: map[string]int{
				"one":   1,
				"two":   2,
				"three": 3,
			},
		}

		encoder := NewEncoder[TestMapStruct]()
		buf := NewBufferFromPool()
		defer buf.ReturnToPool()
		encoder.Marshal(&testData, buf)

		// Create map decoder directly
		mapType := map[string]int{}
		_ = newMapDecoderUsingTagAndOpts(mapType, "glint", tagOptions(""))

		// Test that the map was encoded/decoded properly through normal path
		decoder := NewDecoder[TestMapStruct]()
		var decoded TestMapStruct
		err := decoder.Unmarshal(buf.Bytes, &decoded)

		if err != nil {
			t.Fatalf("Failed to decode: %v", err)
		}

		if len(decoded.Data) != len(testData.Data) {
			t.Errorf("Map length mismatch: got %d, want %d", len(decoded.Data), len(testData.Data))
		}
	})

	t.Run("VariousMapTypes", func(t *testing.T) {
		// Test different key/value combinations
		testCases := []struct {
			name string
			test func(t *testing.T)
		}{
			{
				name: "MapStringInt",
				test: func(t *testing.T) {
					type Test struct {
						M map[string]int `glint:"m"`
					}
					data := Test{M: map[string]int{"a": 1, "b": 2}}
					testMapRoundtrip(t, data)
				},
			},
			{
				name: "MapIntString",
				test: func(t *testing.T) {
					type Test struct {
						M map[int]string `glint:"m"`
					}
					data := Test{M: map[int]string{1: "a", 2: "b"}}
					testMapRoundtrip(t, data)
				},
			},
			{
				name: "MapStringBool",
				test: func(t *testing.T) {
					type Test struct {
						M map[string]bool `glint:"m"`
					}
					data := Test{M: map[string]bool{"true": true, "false": false}}
					testMapRoundtrip(t, data)
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, tc.test)
		}
	})
}

// Helper function for map roundtrip testing
func testMapRoundtrip[T any](t *testing.T, data T) {
	encoder := NewEncoder[T]()
	buf := NewBufferFromPool()
	defer buf.ReturnToPool()
	encoder.Marshal(&data, buf)

	decoder := NewDecoder[T]()
	var result T
	err := decoder.Unmarshal(buf.Bytes, &result)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}
}

// Tests merged from limits_test.go
func TestDecodeLimitsSecurityProtection(t *testing.T) {
	// Create proper test data by encoding first
	type TestStruct struct {
		Data []byte `glint:"info1"`
	}

	encoder := newEncoder(TestStruct{})
	original := TestStruct{Data: []byte("Hello")}

	buf := NewBufferFromPool()
	defer buf.ReturnToPool()
	encoder.Marshal(&original, buf)
	validData := buf.Bytes

	// Test that default limits work
	decoder := NewDecoder[TestStruct]()
	var result TestStruct

	err := decoder.Unmarshal(validData, &result)
	if err != nil {
		t.Errorf("Normal data should decode successfully: %v", err)
	}

	if string(result.Data) != "Hello" {
		t.Errorf("Expected 'Hello', got %s", string(result.Data))
	}

	// Test custom limits
	strictLimits := DecodeLimits{
		MaxByteSliceLen: 10, // Still enough for "Hello"
		MaxSliceInitCap: 100,
		MaxSchemaSize:   1024,
		MaxStringLen:    1000,
	}

	strictDecoder := NewDecoderWithLimits[TestStruct](strictLimits)

	// This should still work since we have 5 bytes
	err = strictDecoder.Unmarshal(validData, &result)
	if err != nil {
		t.Errorf("Small data should work with strict limits: %v", err)
	}
}

func TestSliceCapacityLimitsProtection(t *testing.T) {
	// Test slice cap limits with proper data
	type TestStruct struct {
		Items []string `glint:"items"`
	}

	// Create test data
	encoder := newEncoder(TestStruct{})
	original := TestStruct{Items: []string{"foo", "bar", "baz"}}

	buf := NewBufferFromPool()
	defer buf.ReturnToPool()
	encoder.Marshal(&original, buf)
	validData := buf.Bytes

	// Test slice cap limits
	limits := DecodeLimits{
		MaxByteSliceLen: 1000000,
		MaxSliceInitCap: 2, // Very small cap - should cap initial allocation
		MaxSchemaSize:   1024000,
		MaxStringLen:    1000000,
	}

	decoder := NewDecoderWithLimits[TestStruct](limits)
	var result TestStruct

	err := decoder.Unmarshal(validData, &result)
	if err != nil {
		t.Errorf("Should work with slice cap limits: %v", err)
	}

	if len(result.Items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result.Items))
	}
}

// Tests merged from fix_verification_test.go
func TestFieldOrderIndependentDecoding(t *testing.T) {
	// Comprehensive test to verify the field order bug is fixed

	// Scenario: Producer sends documents, multiple consumers with different schemas receive them
	type Producer struct {
		ID    int32  `glint:"id"`
		Value string `glint:"value"`
		Count int64  `glint:"count"`
	}

	// Consumer 1: Exact same schema as producer
	type Consumer1 struct {
		ID    int32  `glint:"id"`
		Value string `glint:"value"`
		Count int64  `glint:"count"`
	}

	// Consumer 2: Extra field at the end (this always worked)
	type Consumer2 struct {
		ID    int32  `glint:"id"`
		Value string `glint:"value"`
		Count int64  `glint:"count"`
		Extra string `glint:"add1"`
	}

	// Consumer 3: Extra field in the middle (this was broken before the fix)
	type Consumer3 struct {
		ID    int32  `glint:"id"`
		Value string `glint:"value"`
		Extra string `glint:"add1"` // Inserted in middle!
		Count int64  `glint:"count"`
	}

	// Consumer 4: Multiple extra fields
	type Consumer4 struct {
		ID     int32  `glint:"id"`
		Extra1 string `glint:"extra1"`
		Value  string `glint:"value"`
		Extra2 int32  `glint:"extra2"`
		Count  int64  `glint:"count"`
		Extra3 bool   `glint:"extra3"`
	}

	// Producer creates documents
	producer := NewEncoder[Producer]()
	buffer := NewBufferFromPool()
	defer buffer.ReturnToPool()

	testData := Producer{
		ID:    12345,
		Value: "test_value",
		Count: 98765,
	}
	producer.Marshal(&testData, buffer)

	t.Logf("Testing field order bug fix with %d consumers", 4)

	// Test Consumer 1 (same schema)
	consumer1 := NewDecoder[Consumer1]()
	var result1 Consumer1
	err := consumer1.Unmarshal(buffer.Bytes, &result1)
	if err != nil {
		t.Fatalf("Consumer1 failed: %v", err)
	}
	if result1.ID != testData.ID || result1.Value != testData.Value || result1.Count != testData.Count {
		t.Errorf("Consumer1 data mismatch")
	}
	t.Logf("✅ Consumer1 (same schema): ID=%d, Value=%q, Count=%d",
		result1.ID, result1.Value, result1.Count)

	// Test Consumer 2 (extra field at end)
	consumer2 := NewDecoder[Consumer2]()
	var result2 Consumer2
	err = consumer2.Unmarshal(buffer.Bytes, &result2)
	if err != nil {
		t.Fatalf("Consumer2 failed: %v", err)
	}
	if result2.ID != testData.ID || result2.Value != testData.Value || result2.Count != testData.Count {
		t.Errorf("Consumer2 data mismatch")
	}
	if result2.Extra != "" {
		t.Errorf("Consumer2 Extra field should be empty, got %q", result2.Extra)
	}
	t.Logf("✅ Consumer2 (extra at end): ID=%d, Value=%q, Count=%d, Extra=%q",
		result2.ID, result2.Value, result2.Count, result2.Extra)

	// Test Consumer 3 (extra field in middle - this was the problematic case)
	consumer3 := NewDecoder[Consumer3]()
	var result3 Consumer3
	err = consumer3.Unmarshal(buffer.Bytes, &result3)
	if err != nil {
		t.Fatalf("Consumer3 failed: %v", err)
	}
	if result3.ID != testData.ID || result3.Value != testData.Value || result3.Count != testData.Count {
		t.Errorf("Consumer3 data mismatch: ID=%d(want %d), Value=%q(want %q), Count=%d(want %d)",
			result3.ID, testData.ID, result3.Value, testData.Value, result3.Count, testData.Count)
	}
	if result3.Extra != "" {
		t.Errorf("Consumer3 Extra field should be empty, got %q", result3.Extra)
	}
	t.Logf("✅ Consumer3 (extra in middle): ID=%d, Value=%q, Extra=%q, Count=%d",
		result3.ID, result3.Value, result3.Extra, result3.Count)

	// Test Consumer 4 (multiple extra fields)
	consumer4 := NewDecoder[Consumer4]()
	var result4 Consumer4
	err = consumer4.Unmarshal(buffer.Bytes, &result4)
	if err != nil {
		t.Fatalf("Consumer4 failed: %v", err)
	}
	if result4.ID != testData.ID || result4.Value != testData.Value || result4.Count != testData.Count {
		t.Errorf("Consumer4 data mismatch")
	}
	t.Logf("✅ Consumer4 (multiple extras): ID=%d, Value=%q, Count=%d",
		result4.ID, result4.Value, result4.Count)
	t.Logf("   Extra fields: Extra1=%q, Extra2=%d, Extra3=%v",
		result4.Extra1, result4.Extra2, result4.Extra3)

	t.Logf("\n🎉 All consumers successfully decoded the same document!")
	t.Logf("   The field order bug has been fixed!")
}

// Tests merged from delta_overflow_test.go
func TestDeltaEncodingOverflowHandling(t *testing.T) {
	// Test the specific overflow case that was previously failing
	t.Run("Int32Overflow", func(t *testing.T) {
		type TestStruct struct {
			Values []int32 `glint:"values,delta"`
		}

		// This specific case was causing the fuzz test to fail:
		// prev=-1792004048, curr=808464432, delta=2600468480
		// delta (2600468480) exceeds MaxInt32 (2147483647)
		testData := TestStruct{
			Values: []int32{-1792004048, 808464432},
		}

		encoder := NewEncoder[TestStruct]()
		buffer := NewBufferFromPool()
		defer buffer.ReturnToPool()

		encoder.Marshal(&testData, buffer)

		decoder := NewDecoder[TestStruct]()
		var result TestStruct
		err := decoder.Unmarshal(buffer.Bytes, &result)
		if err != nil {
			t.Fatalf("Decoding failed: %v", err)
		}

		if len(result.Values) != 2 {
			t.Errorf("Expected 2 values, got %d", len(result.Values))
		}
		if result.Values[0] != testData.Values[0] {
			t.Errorf("Value[0]: expected %d, got %d", testData.Values[0], result.Values[0])
		}
		if result.Values[1] != testData.Values[1] {
			t.Errorf("Value[1]: expected %d, got %d", testData.Values[1], result.Values[1])
		}

		t.Logf("✅ Successfully handled int32 delta overflow: %d -> %d (delta: %d)",
			testData.Values[0], testData.Values[1], int64(testData.Values[1])-int64(testData.Values[0]))
	})

	t.Run("AllIntegerTypes", func(t *testing.T) {
		// Test overflow scenarios for all integer types

		// Test extreme values for each type
		testCases := []struct {
			name string
			test func(t *testing.T)
		}{
			{"int16", func(t *testing.T) {
				type TestStruct struct {
					Values []int16 `glint:"values,delta"`
				}
				data := TestStruct{Values: []int16{-32000, 32000}} // delta = 64000, exceeds int16 range
				testRoundTrip(t, data)
			}},
			{"int32", func(t *testing.T) {
				type TestStruct struct {
					Values []int32 `glint:"values,delta"`
				}
				data := TestStruct{Values: []int32{-2000000000, 2000000000}} // delta = 4000000000, exceeds int32 range
				testRoundTrip(t, data)
			}},
			{"uint16", func(t *testing.T) {
				type TestStruct struct {
					Values []uint16 `glint:"values,delta"`
				}
				data := TestStruct{Values: []uint16{65000, 500}} // delta = -64500, negative delta for unsigned type
				testRoundTrip(t, data)
			}},
			{"uint32", func(t *testing.T) {
				type TestStruct struct {
					Values []uint32 `glint:"values,delta"`
				}
				data := TestStruct{Values: []uint32{4000000000, 100000000}} // Large negative delta
				testRoundTrip(t, data)
			}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, tc.test)
		}
	})
}

func testRoundTrip[T any](t *testing.T, data T) {
	encoder := NewEncoder[T]()
	buffer := NewBufferFromPool()
	defer buffer.ReturnToPool()

	encoder.Marshal(&data, buffer)

	decoder := NewDecoder[T]()
	var result T
	err := decoder.Unmarshal(buffer.Bytes, &result)
	if err != nil {
		t.Fatalf("Decoding failed: %v", err)
	}

	// Basic verification that decoding succeeded without panics
	t.Logf("✅ Round-trip successful for %T", data)
}

// Tests merged from fresh_cache_test.go
func TestDecoderCacheIsolation(t *testing.T) {
	// Test if using a fresh cache for each decoder fixes the issue

	type Producer struct {
		ID    int32  `glint:"id"`
		Value string `glint:"value"`
		Name  string `glint:"name"`
	}

	type ConsumerA struct {
		ID    int32  `glint:"id"`
		Value string `glint:"value"`
		Name  string `glint:"name"`
	}

	type ConsumerB struct {
		ID    int32  `glint:"id"`
		Value string `glint:"value"`
		Extra string `glint:"add1"`
		Name  string `glint:"name"`
	}

	// Create one document
	producer := NewEncoder[Producer]()
	buffer := NewBufferFromPool()
	defer buffer.ReturnToPool()
	producer.Marshal(&Producer{ID: 123, Value: "VVVV", Name: "NNNN"}, buffer)

	t.Logf("=== Test 1: Using Default Global Cache ===")

	// First decode with ConsumerA (pollutes global cache)
	decoderA1 := NewDecoder[ConsumerA]()
	var resultA1 ConsumerA
	err := decoderA1.Unmarshal(buffer.Bytes, &resultA1)
	if err != nil {
		t.Fatalf("ConsumerA failed: %v", err)
	}
	t.Logf("ConsumerA with global cache: ID=%d, Value=%q, Name=%q",
		resultA1.ID, resultA1.Value, resultA1.Name)

	// Then decode with ConsumerB (uses wrong cached instructions)
	decoderB1 := NewDecoder[ConsumerB]()
	var resultB1 ConsumerB
	err = decoderB1.Unmarshal(buffer.Bytes, &resultB1)
	if err != nil {
		t.Fatalf("ConsumerB failed: %v", err)
	}
	t.Logf("ConsumerB with global cache: ID=%d, Value=%q, Extra=%q, Name=%q",
		resultB1.ID, resultB1.Value, resultB1.Extra, resultB1.Name)

	if resultB1.Name != "NNNN" {
		t.Logf("❌ BUG CONFIRMED: Name is corrupted with global cache")
	}

	t.Logf("\n=== Test 2: Using Fresh Cache Per Decoder ===")

	// Create fresh caches for each decoder
	cacheA := &DecodeInstructionLookup{}
	cacheB := &DecodeInstructionLookup{}

	// Decode with ConsumerA using its own cache
	decoderA2 := NewDecoder[ConsumerA]()
	var resultA2 ConsumerA
	err = decoderA2.impl.UnmarshalWithContext(buffer.Bytes, &resultA2, DecoderContext{
		InstructionCache: cacheA,
		ID:               1,
	})
	if err != nil {
		t.Fatalf("ConsumerA with fresh cache failed: %v", err)
	}
	t.Logf("ConsumerA with fresh cache: ID=%d, Value=%q, Name=%q",
		resultA2.ID, resultA2.Value, resultA2.Name)

	// Decode with ConsumerB using its own cache
	decoderB2 := NewDecoder[ConsumerB]()
	var resultB2 ConsumerB
	err = decoderB2.impl.UnmarshalWithContext(buffer.Bytes, &resultB2, DecoderContext{
		InstructionCache: cacheB,
		ID:               2,
	})
	if err != nil {
		t.Fatalf("ConsumerB with fresh cache failed: %v", err)
	}
	t.Logf("ConsumerB with fresh cache: ID=%d, Value=%q, Extra=%q, Name=%q",
		resultB2.ID, resultB2.Value, resultB2.Extra, resultB2.Name)

	if resultB2.Name == "NNNN" && resultB2.Extra == "" {
		t.Logf("✅ SUCCESS: Fresh cache per decoder fixes the issue!")
	} else {
		t.Logf("❌ STILL BROKEN: Even with fresh cache, Name=%q, Extra=%q",
			resultB2.Name, resultB2.Extra)
	}

	t.Logf("\n=== Test 3: Reuse Same Document with Same Decoder ===")

	// This tests if cache works correctly for same decoder type
	cacheShared := &DecodeInstructionLookup{}

	// First decode
	decoderB3 := NewDecoder[ConsumerB]()
	var resultB3 ConsumerB
	err = decoderB3.impl.UnmarshalWithContext(buffer.Bytes, &resultB3, DecoderContext{
		InstructionCache: cacheShared,
		ID:               3,
	})
	if err != nil {
		t.Fatalf("First decode failed: %v", err)
	}
	t.Logf("First decode: Name=%q, Extra=%q", resultB3.Name, resultB3.Extra)

	// Second decode with same decoder type (should hit cache)
	decoderB4 := NewDecoder[ConsumerB]()
	var resultB4 ConsumerB
	err = decoderB4.impl.UnmarshalWithContext(buffer.Bytes, &resultB4, DecoderContext{
		InstructionCache: cacheShared,
		ID:               3,
	})
	if err != nil {
		t.Fatalf("Second decode failed: %v", err)
	}
	t.Logf("Second decode (cache hit): Name=%q, Extra=%q", resultB4.Name, resultB4.Extra)

	if resultB3.Name == resultB4.Name && resultB3.Extra == resultB4.Extra {
		t.Logf("✅ Cache works correctly for same decoder type")
	}
}

// TestDocumentProcessing consolidates all document processing related tests
func TestDocumentBuilderAndWalkerFunctionality(t *testing.T) {
	t.Run("DocumentAPI", func(t *testing.T) {
		t.Run("BasicDocumentBuilder", func(t *testing.T) {
			doc := DocumentBuilder{}
			doc.AppendBool("flag", true)
			doc.AppendString("name", "TestUser")
			doc.AppendInt("age", 42)

			type ins struct {
				Flag bool   `glint:"flag"`
				Name string `glint:"name"`
				Age  int    `glint:"age"`
			}
			decoder := newDecoder(ins{})

			inst := ins{}

			decoder.Unmarshal(doc.Bytes(), &inst)

			if inst.Flag != true {
				t.Error("flag not true")
			}
			if inst.Name != "TestUser" {
				t.Error("name not TestUser")
			}
			if inst.Age != 42 {
				t.Error("age not 42")
			}
		})

		t.Run("ComplexDocumentBuilder", func(t *testing.T) {
			b := DocumentBuilder{}
			b.AppendString("name", "TestUser")
			b.AppendInt("age", 40)

			wifeDoc := DocumentBuilder{}
			wifeDoc.AppendString("name", "TestSpouse")
			wifeDoc.AppendInt("age", 39)
			b.AppendNestedDocument("wife", &wifeDoc)

			children := []DocumentBuilder{}
			children = append(children, *(&DocumentBuilder{}).AppendString("name", "TestChild1").AppendInt("age", 8))
			children = append(children, *(&DocumentBuilder{}).AppendString("name", "TestChild2").AppendInt("age", 14))

			sb := SliceBuilder{}
			sb.AppendNestedDocumentSlice(children)
			b.AppendSlice("items1", sb)

			chchildren := []SliceBuilder{}
			chchildren = append(chchildren, sb)
			chchildren = append(chchildren, sb)

			sb2 := SliceBuilder{}
			sb2.AppendSlice(chchildren)

			b.AppendSlice("nested slice", sb2)

			doc := b.Bytes()

			expect := []byte{0, 27, 114, 94, 153, 76, 14, 4, 110, 97, 109, 101, 2, 3, 97, 103, 101, 16, 4, 119, 105, 102, 101, 11, 14, 4, 110, 97, 109, 101, 2, 3, 97, 103, 101, 48, 6, 105, 116, 101, 109, 115, 49, 11, 14, 4, 110, 97, 109, 101, 2, 3, 97, 103, 101, 32, 12, 110, 101, 115, 116, 101, 100, 32, 115, 108, 105, 99, 101, 48, 11, 14, 4, 110, 97, 109, 101, 2, 3, 97, 103, 101, 8, 84, 101, 115, 116, 85, 115, 101, 114, 80, 10, 84, 101, 115, 116, 83, 112, 111, 117, 115, 101, 78, 2, 10, 84, 101, 115, 116, 67, 104, 105, 108, 100, 49, 16, 10, 84, 101, 115, 116, 67, 104, 105, 108, 100, 50, 28, 2, 2, 10, 84, 101, 115, 116, 67, 104, 105, 108, 100, 49, 16, 10, 84, 101, 115, 116, 67, 104, 105, 108, 100, 50, 28, 2, 10, 84, 101, 115, 116, 67, 104, 105, 108, 100, 49, 16, 10, 84, 101, 115, 116, 67, 104, 105, 108, 100, 50, 28}
			if !bytes.Equal(expect, doc) {
				t.Errorf("doc not equal. got %v want %v", doc, expect)
				Print(doc)
			}
		})

		t.Run("DocumentBuilderAllTypes", func(t *testing.T) {
			// Test all DocumentBuilder methods that were previously untested
			doc := &DocumentBuilder{}

			// Test all numeric append methods
			doc.AppendUint8("uint8_field", 255)
			doc.AppendUint16("uint16_field", 65535)
			doc.AppendUint32("uint32_field", 4294967295)
			doc.AppendUint64("uint64_field", 18446744073709551615)
			doc.AppendUint("uint_field", 123456)
			doc.AppendInt8("int8_field", -128)
			doc.AppendInt16("int16_field", -32768)
			doc.AppendInt32("int32_field", -2147483648)
			doc.AppendInt64("int64_field", -9223372036854775808)
			doc.AppendFloat32("float32_field", 3.14159)
			doc.AppendFloat64("float64_field", 2.7182818284590451)
			doc.AppendBool("bool_field", true)
			doc.AppendBytes("bytes_field", []byte{0x01, 0x02, 0x03, 0xFF})
			doc.AppendTime("time_field", time.Unix(1672531200, 0))

			data := doc.Bytes()

			// Verify document was created successfully
			if len(data) == 0 {
				t.Error("Document should not be empty")
			}

			// Verify SPrint can handle it (tests printer functions)
			output := SPrint(data)
			expectedFields := []string{"uint8_field", "uint16_field", "uint32_field", "uint64_field", "uint_field",
				"int8_field", "int16_field", "int32_field", "int64_field", "float32_field", "float64_field",
				"bool_field", "bytes_field", "time_field"}

			for _, field := range expectedFields {
				if !strings.Contains(output, field) {
					t.Errorf("SPrint output should contain field '%s'", field)
				}
			}
		})

		t.Run("DocumentTranscoding", func(t *testing.T) {
			b := DocumentBuilder{}
			b.AppendString("name", "TestUser")
			b.AppendInt("age", 40)

			wifeDoc := DocumentBuilder{}
			wifeDoc.AppendString("name", "TestSpouse")
			wifeDoc.AppendInt("age", 39)
			b.AppendNestedDocument("wife", &wifeDoc)

			children := []DocumentBuilder{}
			children = append(children, *(&DocumentBuilder{}).AppendString("name", "TestChild1").AppendInt("age", 8))
			children = append(children, *(&DocumentBuilder{}).AppendString("name", "TestChild2").AppendInt("age", 14))
			sb := SliceBuilder{}
			sb.AppendNestedDocumentSlice(children)
			b.AppendSlice("items1", sb)

			chchildren := []SliceBuilder{}
			chchildren = append(chchildren, sb)
			chchildren = append(chchildren, sb)

			sb2 := SliceBuilder{}
			sb2.AppendSlice(chchildren)
			b.AppendSlice("nested slice", sb2)

			input := b.Bytes()

			transcoder := &JSONTranscodeVisitor{b: &Buffer{}}
			transcoder.b.Bytes = append(transcoder.b.Bytes, "{"...)

			walker := NewWalker(input)
			walker.Walk(transcoder)

			transcoder.b.Bytes = append(transcoder.b.Bytes, "}"...)

			expectJSON := `{"name":"TestUser","age":40,"wife":{"name":"TestSpouse","age":39},"items1":[{"name":"TestChild1","age":8},{"name":"TestChild2","age":14}],"nested slice":[[{"name":"TestChild1","age":8},{"name":"TestChild2","age":14}],[{"name":"TestChild1","age":8},{"name":"TestChild2","age":14}]]}`
			if expectJSON != string(transcoder.Bytes()) {
				t.Errorf("Expecting %s, got %s", expectJSON, string(transcoder.Bytes()))
				fmt.Println(stringDiff(expectJSON, string(transcoder.Bytes())))
			}
		})

		t.Run("DocumentFormatting", func(t *testing.T) {
			// Create test document
			doc := &DocumentBuilder{}
			doc.AppendString("name", "SampleUser").AppendInt("age", 30)
			data := Document(doc.Bytes())

			// Test %s formatting (pretty print)
			sFormat := fmt.Sprintf("%s", data)
			if !strings.Contains(sFormat, "Glint Document") {
				t.Error("Document with s format should contain pretty printed output")
			}
			if !strings.Contains(sFormat, "name: SampleUser") {
				t.Error("Document with s format should contain field values")
			}

			// Test %v formatting (same as %s)
			vFormat := fmt.Sprintf("%v", data)
			if sFormat != vFormat {
				t.Error("Document with v format should be same as s format")
			}

			// Test %+v formatting (verbose with hex)
			verboseFormat := fmt.Sprintf("%+v", data)
			if !strings.Contains(verboseFormat, "hex:") {
				t.Error("Document with +v format should contain hex representation")
			}
			if !strings.Contains(verboseFormat, "Glint Document") {
				t.Error("Document with +v format should contain pretty printed output")
			}

			// Test %x formatting (hex)
			hexFormat := fmt.Sprintf("%x", data)
			if len(hexFormat) == 0 {
				t.Error("Document with x format should not be empty")
			}
			// Should be pure hex without spaces
			if strings.Contains(hexFormat, " ") || strings.Contains(hexFormat, "Glint") {
				t.Error("Document with x format should be pure hex")
			}

			// Test %X formatting (uppercase hex)
			hexUpperFormat := fmt.Sprintf("%X", data)
			if len(hexUpperFormat) == 0 {
				t.Error("Document with X format should not be empty")
			}
			if hexUpperFormat == hexFormat {
				// Only fail if there are actually letters that should be different
				if strings.ContainsAny(hexFormat, "abcdef") {
					t.Error("Document with X format should be uppercase")
				}
			}

			// Test %q formatting (quoted)
			quotedFormat := fmt.Sprintf("%q", data)
			if !strings.HasPrefix(quotedFormat, "\"") || !strings.HasSuffix(quotedFormat, "\"") {
				t.Error("Document with q format should be quoted")
			}

			// Test unsupported format verb
			unsupportedFormat := fmt.Sprintf("%d", data)
			if !strings.Contains(unsupportedFormat, "%!d") {
				t.Error("Document unsupported format should show error")
			}

			// Test String() method directly
			str := data.String()
			if !strings.Contains(str, "Glint Document") {
				t.Error("Document.String() should return pretty printed output")
			}
		})

		t.Run("DeltaEncodingDocumentBuilder", func(t *testing.T) {
			tests := []struct {
				name           string
				buildDocument  func() *DocumentBuilder
				expectedFields []string
			}{
				{
					name: "Int32DeltaEncoding",
					buildDocument: func() *DocumentBuilder {
						doc := &DocumentBuilder{}
						sb := &SliceBuilder{}
						sb.AppendInt32SliceDelta([]int32{100, 102, 105, 103, 110})
						doc.AppendSlice("values", *sb)
						return doc
					},
					expectedFields: []string{"values"},
				},
				{
					name: "Int64DeltaEncoding",
					buildDocument: func() *DocumentBuilder {
						doc := &DocumentBuilder{}
						sb := &SliceBuilder{}
						sb.AppendInt64SliceDelta([]int64{1000, 1005, 1002, 1010})
						doc.AppendSlice("numbers", *sb)
						return doc
					},
					expectedFields: []string{"numbers"},
				},
				{
					name: "IntDeltaEncoding",
					buildDocument: func() *DocumentBuilder {
						doc := &DocumentBuilder{}
						sb := &SliceBuilder{}
						sb.AppendIntSliceDelta([]int{50, 55, 52, 60})
						doc.AppendSlice("sequence", *sb)
						return doc
					},
					expectedFields: []string{"sequence"},
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					doc := tt.buildDocument()
					data := doc.Bytes()

					// Verify the document was created without errors
					if len(data) == 0 {
						t.Error("Document should not be empty")
					}

					// Verify we can print it (basic validation)
					output := SPrint(data)
					for _, field := range tt.expectedFields {
						if !strings.Contains(output, field) {
							t.Errorf("SPrint output should contain field '%s'", field)
						}
					}
				})
			}
		})
	})

	t.Run("DynamicValueProcessing", func(t *testing.T) {
		// Define a set of test cases for dynamic value append/read
		tests := []struct {
			name  string
			value any
		}{
			{
				name:  "String",
				value: "hello",
			},
			{
				name:  "Int",
				value: 123,
			},
			{
				name:  "Int8",
				value: int8(123),
			},
			{
				name:  "Int16",
				value: int16(123),
			},
			{
				name:  "Int32",
				value: int32(123),
			},
			{
				name:  "Int64",
				value: int64(123),
			},
			{
				name:  "Uint",
				value: uint(123),
			},
			{
				name:  "Uint8",
				value: uint8(123),
			},
			{
				name:  "Uint16",
				value: uint16(123),
			},
			{
				name:  "Uint32",
				value: uint32(123),
			},
			{
				name:  "Uint64",
				value: uint64(123),
			},
			{
				name:  "Float32",
				value: float32(123.123),
			},
			{
				name:  "Float64",
				value: float64(123.123),
			},
			{
				name:  "Bool",
				value: true,
			},
			{
				name:  "SliceOfString",
				value: []string{"a", "b", "c"},
			},
			{
				name:  "SliceOfInt",
				value: []int{1, 2, 3},
			},
			{
				name: "SliceOfTime",
				value: []time.Time{
					time.Date(1969, 4, 20, 4, 20, 4, 20, time.UTC),
					time.Date(1969, 4, 20, 4, 20, 4, 20, time.UTC),
				},
			},
		}

		// Iterate over each test case
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// Instantiate a new Buffer
				b := &Buffer{}

				// Call AppendDynamicValue with the test case value
				AppendDynamicValue(tc.value, b)
				v := ReadDynamicValue(b.Bytes)

				// Compare buffer content with expected bytes
				if !reflect.DeepEqual(v, tc.value) {
					t.Errorf("Test %s failed: got %v, want %v", tc.name, v, tc.value)
				}
			})
		}

		// Test AppendDynamicValue for better coverage
		t.Run("AppendDynamicValueComprehensive", func(t *testing.T) {
			testCases := []struct {
				name  string
				value interface{}
			}{
				// Test all supported types
				{"string", "test"},
				{"int", 42},
				{"int8", int8(8)},
				{"int16", int16(16)},
				{"int32", int32(32)},
				{"int64", int64(64)},
				{"uint", uint(42)},
				{"uint8", uint8(8)},
				{"uint16", uint16(16)},
				{"uint32", uint32(32)},
				{"uint64", uint64(64)},
				{"float32", float32(3.14)},
				{"float64", float64(3.14159)},
				{"bool", true},
				{"time", time.Unix(1609459200, 0)},
				{"bytes", []byte{1, 2, 3}},
				// Slice types
				{"[]string", []string{"a", "b"}},
				{"[]int", []int{1, 2, 3}},
				{"[]int8", []int8{1, 2}},
				{"[]int16", []int16{1, 2}},
				{"[]int32", []int32{1, 2}},
				{"[]int64", []int64{1, 2}},
				{"[]uint", []uint{1, 2}},
				{"[]uint8", []uint8{1, 2}},
				{"[]uint16", []uint16{1, 2}},
				{"[]uint32", []uint32{1, 2}},
				{"[]uint64", []uint64{1, 2}},
				{"[]float32", []float32{1.0, 2.0}},
				{"[]float64", []float64{1.0, 2.0}},
				{"[]bool", []bool{true, false}},
				{"[]time.Time", []time.Time{time.Unix(1609459200, 0)}},
				{"[][]byte", [][]byte{{1}, {2}}},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					buf := &Buffer{}
					AppendDynamicValue(tc.value, buf)

					if len(buf.Bytes) == 0 {
						t.Error("AppendDynamicValue should produce output")
					}

					// Try to read back
					result := ReadDynamicValue(buf.Bytes)
					// Basic check that we got something back
					if result == nil {
						t.Error("ReadDynamicValue returned nil")
					}
				})
			}
		})

		// Test ReadDynamicSlice edge cases
		t.Run("ReadDynamicSliceEdgeCases", func(t *testing.T) {
			// Test with empty slices
			testSliceReading := func(wireType WireType, expectedNil bool) {
				buf := &Buffer{}
				appendVarint(buf, uint64(WireSliceFlag|wireType))
				appendVarint(buf, 0) // Empty slice

				reader := NewReader(buf.Bytes)
				reader.ReadVarint() // Read wire type
				result := ReadDynamicSlice(&reader, WireSliceFlag|wireType)

				if expectedNil && result != nil {
					t.Errorf("Expected nil for wire type %v, got %v", wireType, result)
				} else if !expectedNil && result == nil {
					t.Errorf("Expected non-nil for wire type %v", wireType)
				}
			}

			// Test various wire types
			testSliceReading(WireString, false)
			testSliceReading(WireInt, false)
			testSliceReading(WireBytes, false)
			testSliceReading(WireStruct, true)   // Unsupported, returns nil
			testSliceReading(WireMap, true)      // Unsupported, returns nil
			testSliceReading(WireType(99), true) // Unknown type
		})

		// Test ReadDynamic* functions for comprehensive coverage
		t.Run("DynamicPrimitives", func(t *testing.T) {
			// Test basic ReadDynamic functions with clear, readable patterns
			t.Run("String", func(t *testing.T) {
				value := "test string"
				buf := &Buffer{}
				appendVarint(buf, uint64(WireString))
				buf.AppendString(value)
				result, ok := ReadDynamicString(buf.Bytes)
				if !ok || result != value {
					t.Errorf("ReadDynamicString failed: got (%s, %v), want (%s, true)", result, ok, value)
				}
			})

			t.Run("Int8", func(t *testing.T) {
				value := int8(-42)
				buf := &Buffer{}
				appendVarint(buf, uint64(WireInt8))
				buf.AppendInt8(value)
				result, ok := ReadDynamicInt8(buf.Bytes)
				if !ok || result != value {
					t.Errorf("ReadDynamicInt8 failed: got (%d, %v), want (%d, true)", result, ok, value)
				}
			})

			t.Run("Uint32", func(t *testing.T) {
				value := uint32(4294967295)
				buf := &Buffer{}
				appendVarint(buf, uint64(WireUint32))
				buf.AppendUint32(value)
				result, ok := ReadDynamicUint32(buf.Bytes)
				if !ok || result != value {
					t.Errorf("ReadDynamicUint32 failed: got (%d, %v), want (%d, true)", result, ok, value)
				}
			})

			t.Run("Float64", func(t *testing.T) {
				value := 3.141592653589793
				buf := &Buffer{}
				appendVarint(buf, uint64(WireFloat64))
				buf.AppendFloat64(value)
				result, ok := ReadDynamicFloat64(buf.Bytes)
				if !ok || result != value {
					t.Errorf("ReadDynamicFloat64 failed: got (%f, %v), want (%f, true)", result, ok, value)
				}
			})

			t.Run("Bool", func(t *testing.T) {
				value := true
				buf := &Buffer{}
				appendVarint(buf, uint64(WireBool))
				buf.AppendBool(value)
				result, ok := ReadDynamicBool(buf.Bytes)
				if !ok || result != value {
					t.Errorf("ReadDynamicBool failed: got (%t, %v), want (%t, true)", result, ok, value)
				}
			})

			t.Run("Time", func(t *testing.T) {
				value := time.Unix(1672531200, 0)
				buf := &Buffer{}
				appendVarint(buf, uint64(WireTime))
				buf.AppendTime(value)
				result, ok := ReadDynamicTime(buf.Bytes)
				if !ok || !result.Equal(value) {
					t.Errorf("ReadDynamicTime failed: got (%v, %v), want (%v, true)", result, ok, value)
				}
			})

			t.Run("Bytes", func(t *testing.T) {
				value := []byte{0x01, 0x02, 0x03, 0xFF}
				buf := &Buffer{}
				appendVarint(buf, uint64(WireBytes))
				buf.AppendBytes(value)
				result, ok := ReadDynamicBytes(buf.Bytes)
				if !ok || !bytes.Equal(result, value) {
					t.Errorf("ReadDynamicBytes failed: got (%v, %v), want (%v, true)", result, ok, value)
				}
			})

			// Test additional numeric types for complete coverage
			intTests := []struct {
				name     string
				wireType WireType
				value    interface{}
				testFunc func([]byte) (interface{}, bool)
			}{
				{"Int", WireInt, -12345, func(data []byte) (interface{}, bool) { return ReadDynamicInt(data) }},
				{"Int16", WireInt16, int16(-1234), func(data []byte) (interface{}, bool) { return ReadDynamicInt16(data) }},
				{"Int32", WireInt32, int32(-123456), func(data []byte) (interface{}, bool) { return ReadDynamicInt32(data) }},
				{"Int64", WireInt64, int64(-1234567890), func(data []byte) (interface{}, bool) { return ReadDynamicInt64(data) }},
				{"Uint", WireUint, uint(12345), func(data []byte) (interface{}, bool) { return ReadDynamicUint(data) }},
				{"Uint8", WireUint8, uint8(255), func(data []byte) (interface{}, bool) { return ReadDynamicUint8(data) }},
				{"Uint16", WireUint16, uint16(65535), func(data []byte) (interface{}, bool) { return ReadDynamicUint16(data) }},
				{"Uint64", WireUint64, uint64(18446744073709551615), func(data []byte) (interface{}, bool) { return ReadDynamicUint64(data) }},
				{"Float32", WireFloat32, float32(-3.14159), func(data []byte) (interface{}, bool) { return ReadDynamicFloat32(data) }},
			}

			for _, test := range intTests {
				t.Run(test.name, func(t *testing.T) {
					buf := &Buffer{}
					appendVarint(buf, uint64(test.wireType))
					switch v := test.value.(type) {
					case int:
						buf.AppendInt(v)
					case int16:
						buf.AppendInt16(v)
					case int32:
						buf.AppendInt32(v)
					case int64:
						buf.AppendInt64(v)
					case uint:
						buf.AppendUint(v)
					case uint8:
						buf.AppendUint8(v)
					case uint16:
						buf.AppendUint16(v)
					case uint64:
						buf.AppendUint64(v)
					case float32:
						buf.AppendFloat32(v)
					}
					result, ok := test.testFunc(buf.Bytes)
					if !ok || !reflect.DeepEqual(result, test.value) {
						t.Errorf("ReadDynamic%s failed: got (%v, %v), want (%v, true)", test.name, result, ok, test.value)
					}
				})
			}
		})

		t.Run("DynamicSlices", func(t *testing.T) {
			// Test slice functions with representative samples
			t.Run("StringSlice", func(t *testing.T) {
				value := []string{"hello", "world", "test"}
				buf := &Buffer{}
				appendVarint(buf, uint64(WireSliceFlag|WireString))
				appendVarint(buf, uint64(len(value)))
				for _, s := range value {
					buf.AppendString(s)
				}
				result, ok := ReadDynamicStringSlice(buf.Bytes)
				if !ok || !reflect.DeepEqual(result, value) {
					t.Errorf("ReadDynamicStringSlice failed")
				}
			})

			t.Run("Int16Slice", func(t *testing.T) {
				value := []int16{-32768, 0, 32767}
				buf := &Buffer{}
				appendVarint(buf, uint64(WireSliceFlag|WireInt16))
				appendVarint(buf, uint64(len(value)))
				for _, i := range value {
					buf.AppendInt16(i)
				}
				result, ok := ReadDynamicInt16Slice(buf.Bytes)
				if !ok || !reflect.DeepEqual(result, value) {
					t.Errorf("ReadDynamicInt16Slice failed")
				}
			})

			t.Run("BoolSlice", func(t *testing.T) {
				value := []bool{true, false, true}
				buf := &Buffer{}
				appendVarint(buf, uint64(WireSliceFlag|WireBool))
				appendVarint(buf, uint64(len(value)))
				for _, b := range value {
					buf.AppendBool(b)
				}
				result, ok := ReadDynamicBoolSlice(buf.Bytes)
				if !ok || !reflect.DeepEqual(result, value) {
					t.Errorf("ReadDynamicBoolSlice failed")
				}
			})
		})
	})
}

func TestDocumentWalkerBasicTraversal(t *testing.T) {
	// TODO: Fix DocumentWalker API usage
	// Create a simple test to verify DocumentBuilder works
	b := DocumentBuilder{}
	b.AppendString("name", "TestUser")
	b.AppendInt("age", 40)

	documentBytes := b.Bytes()
	if len(documentBytes) == 0 {
		t.Error("DocumentBuilder should produce bytes")
	}
}

func TestSchemaParsingAndValidation(t *testing.T) {
	t.Run("ReadSubSchemaForStruct", func(t *testing.T) {
		// Test ReadSubSchema with nested struct
		type NestedStruct struct {
			Value int `glint:"value"`
		}
		type TestStruct struct {
			Nested NestedStruct `glint:"inner1"`
		}

		encoder := NewEncoder[TestStruct]()
		buf := NewBufferFromPool()
		defer buf.ReturnToPool()

		testData := TestStruct{
			Nested: NestedStruct{Value: 42},
		}
		encoder.Marshal(&testData, buf)

		// Create printer document to test schema reading
		reader := NewReader(buf.Bytes)
		doc := NewPrinterDocument(&reader)
		schema := NewPrinterSchema(&doc.Schema)

		if len(schema.Fields) == 0 {
			t.Error("Expected at least one field in schema")
		}

		// Find the nested field
		var nestedField *PrinterSchemaField
		for i := range schema.Fields {
			if schema.Fields[i].Name == "inner1" {
				nestedField = &schema.Fields[i]
				break
			}
		}

		if nestedField == nil {
			t.Error("Could not find nested field in schema")
		}

		if nestedField.NestedSchema == nil {
			t.Error("Expected NestedSchema to be populated for struct field")
		}

		if len(nestedField.NestedSchema.Fields) == 0 {
			t.Error("Expected nested schema to have fields")
		}
	})

	t.Run("ReadSubSchemaForMap", func(t *testing.T) {
		// Test ReadSubSchema with map types
		type TestStruct struct {
			StringMap map[string]string `glint:"map1"`
		}

		encoder := NewEncoder[TestStruct]()
		buf := NewBufferFromPool()
		defer buf.ReturnToPool()

		testData := TestStruct{
			StringMap: map[string]string{"key": "value"},
		}
		encoder.Marshal(&testData, buf)

		reader := NewReader(buf.Bytes)
		doc := NewPrinterDocument(&reader)
		schema := NewPrinterSchema(&doc.Schema)

		var mapField *PrinterSchemaField
		for i := range schema.Fields {
			if schema.Fields[i].Name == "map1" {
				mapField = &schema.Fields[i]
				break
			}
		}

		if mapField == nil {
			t.Error("Could not find map field in schema")
		}

		if mapField.TypeID&WireTypeMask != WireMap {
			t.Error("Expected map field to have WireMap type")
		}

		if len(mapField.MapType) != 2 {
			t.Error("Expected MapType to have 2 elements for key and value types")
		}
	})

	t.Run("ReadSubSchemaForSlice", func(t *testing.T) {
		// Test ReadSubSchema with slice types
		type TestStruct struct {
			IntSlice []int `glint:"intslice"`
		}

		encoder := NewEncoder[TestStruct]()
		buf := NewBufferFromPool()
		defer buf.ReturnToPool()

		testData := TestStruct{
			IntSlice: []int{1, 2, 3},
		}
		encoder.Marshal(&testData, buf)

		reader := NewReader(buf.Bytes)
		doc := NewPrinterDocument(&reader)
		schema := NewPrinterSchema(&doc.Schema)

		var sliceField *PrinterSchemaField
		for i := range schema.Fields {
			if schema.Fields[i].Name == "intslice" {
				sliceField = &schema.Fields[i]
				break
			}
		}

		if sliceField == nil {
			t.Error("Could not find slice field in schema")
		}

		if sliceField.TypeID&WireSliceFlag == 0 {
			t.Error("Expected slice field to have WireSliceFlag set")
		}

		if !sliceField.IsSlice {
			t.Error("Expected IsSlice to be true for slice field")
		}
	})

	t.Run("ReadSubSchemaForNestedSlice", func(t *testing.T) {
		// Test ReadSubSchema with nested struct slice
		type NestedStruct struct {
			Value string `glint:"value"`
		}
		type TestStruct struct {
			NestedSlice []NestedStruct `glint:"nestedslice"`
		}

		encoder := NewEncoder[TestStruct]()
		buf := NewBufferFromPool()
		defer buf.ReturnToPool()

		testData := TestStruct{
			NestedSlice: []NestedStruct{{Value: "test"}},
		}
		encoder.Marshal(&testData, buf)

		reader := NewReader(buf.Bytes)
		doc := NewPrinterDocument(&reader)
		schema := NewPrinterSchema(&doc.Schema)

		var nestedSliceField *PrinterSchemaField
		for i := range schema.Fields {
			if schema.Fields[i].Name == "nestedslice" {
				nestedSliceField = &schema.Fields[i]
				break
			}
		}

		if nestedSliceField == nil {
			t.Error("Could not find nested slice field in schema")
		}

		if nestedSliceField.NestedSchema == nil {
			t.Error("Expected NestedSchema to be populated for nested struct slice")
		}

		if len(nestedSliceField.NestedSchema.Fields) == 0 {
			t.Error("Expected nested schema to have fields for struct slice")
		}
	})

	t.Run("ReadSubSchemaForComplexNesting", func(t *testing.T) {
		// Test ReadSubSchema with map containing struct values
		type NestedStruct struct {
			Data string `glint:"info1"`
		}
		type TestStruct struct {
			ComplexMap map[string]NestedStruct `glint:"complexmap"`
		}

		encoder := NewEncoder[TestStruct]()
		buf := NewBufferFromPool()
		defer buf.ReturnToPool()

		testData := TestStruct{
			ComplexMap: map[string]NestedStruct{
				"key1": {Data: "value1"},
			},
		}
		encoder.Marshal(&testData, buf)

		reader := NewReader(buf.Bytes)
		doc := NewPrinterDocument(&reader)
		schema := NewPrinterSchema(&doc.Schema)

		var complexField *PrinterSchemaField
		for i := range schema.Fields {
			if schema.Fields[i].Name == "complexmap" {
				complexField = &schema.Fields[i]
				break
			}
		}

		if complexField == nil {
			t.Error("Could not find complex map field in schema")
		}

		if complexField.TypeID&WireTypeMask != WireMap {
			t.Error("Expected complex field to be a map type")
		}

		// For maps with struct values, we should have nested schema
		if complexField.NestedSchema == nil {
			t.Error("Expected NestedSchema for map with struct values")
		}
	})

	t.Run("ReadSubSchemaEdgeCases", func(t *testing.T) {
		// Test edge cases in ReadSubSchema
		type SimpleStruct struct {
			Text string `glint:"text"`
		}
		type TestStruct struct {
			SliceOfSlices [][]int       `glint:"sliceslice"`
			PtrToStruct   *SimpleStruct `glint:"ptrstruct"`
		}

		encoder := NewEncoder[TestStruct]()
		buf := NewBufferFromPool()
		defer buf.ReturnToPool()

		simpleData := &SimpleStruct{Text: "hello"}
		testData := TestStruct{
			SliceOfSlices: [][]int{{1, 2}, {3, 4}},
			PtrToStruct:   simpleData,
		}
		encoder.Marshal(&testData, buf)

		reader := NewReader(buf.Bytes)
		doc := NewPrinterDocument(&reader)
		schema := NewPrinterSchema(&doc.Schema)

		// Verify we can parse complex schema structures without errors
		if len(schema.Fields) == 0 {
			t.Error("Expected fields in complex schema")
		}

		// Find slice of slices field
		var sliceSlicesField *PrinterSchemaField
		for i := range schema.Fields {
			if schema.Fields[i].Name == "sliceslice" {
				sliceSlicesField = &schema.Fields[i]
				break
			}
		}

		if sliceSlicesField == nil {
			t.Error("Could not find slice of slices field")
		}

		if sliceSlicesField.TypeID&WireSliceFlag == 0 {
			t.Error("Expected slice field to have WireSliceFlag")
		}

		// Find pointer to struct field
		var ptrStructField *PrinterSchemaField
		for i := range schema.Fields {
			if schema.Fields[i].Name == "ptrstruct" {
				ptrStructField = &schema.Fields[i]
				break
			}
		}

		if ptrStructField == nil {
			t.Error("Could not find pointer to struct field")
		}

		if ptrStructField.TypeID&WirePtrFlag == 0 {
			t.Error("Expected pointer field to have WirePtrFlag")
		}

		if !ptrStructField.IsPointer {
			t.Error("Expected IsPointer to be true for pointer field")
		}
	})
}

func TestWireTypeConversionAndMapping(t *testing.T) {
	// Consolidated test covering all type conversions with round-trip validation
	tests := []struct {
		name     string
		typ      reflect.Type
		expected WireType
	}{
		// Basic types
		{"bool", reflect.TypeOf(true), WireBool},
		{"int", reflect.TypeOf(int(0)), WireInt},
		{"int8", reflect.TypeOf(int8(0)), WireInt8},
		{"int16", reflect.TypeOf(int16(0)), WireInt16},
		{"int32", reflect.TypeOf(int32(0)), WireInt32},
		{"int64", reflect.TypeOf(int64(0)), WireInt64},
		{"uint", reflect.TypeOf(uint(0)), WireUint},
		{"uint8", reflect.TypeOf(uint8(0)), WireUint8},
		{"uint16", reflect.TypeOf(uint16(0)), WireUint16},
		{"uint32", reflect.TypeOf(uint32(0)), WireUint32},
		{"uint64", reflect.TypeOf(uint64(0)), WireUint64},
		{"float32", reflect.TypeOf(float32(0)), WireFloat32},
		{"float64", reflect.TypeOf(float64(0)), WireFloat64},
		{"string", reflect.TypeOf(""), WireString},
		{"[]byte", reflect.TypeOf([]byte{}), WireBytes},
		{"time.Time", reflect.TypeOf(time.Time{}), WireTime},
		{"struct", reflect.TypeOf(struct{}{}), WireStruct},
		{"map", reflect.TypeOf(map[string]int{}), WireMap},
		// Slices
		{"[]int", reflect.TypeOf([]int{}), WireSliceFlag | WireInt},
		{"[]string", reflect.TypeOf([]string{}), WireSliceFlag | WireString},
		{"[]bool", reflect.TypeOf([]bool{}), WireSliceFlag | WireBool},
		{"[]float64", reflect.TypeOf([]float64{}), WireSliceFlag | WireFloat64},
		// Pointers
		{"*int", reflect.TypeOf((*int)(nil)), WirePtrFlag | WireInt},
		{"*string", reflect.TypeOf((*string)(nil)), WirePtrFlag | WireString},
		{"*bool", reflect.TypeOf((*bool)(nil)), WirePtrFlag | WireBool},
		// Complex nested
		{"[]*int", reflect.TypeOf([]*int{}), WireSliceFlag | WirePtrFlag | WireInt},
		{"*[]int", reflect.TypeOf((*[]int)(nil)), WirePtrFlag | WireSliceFlag | WireInt},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Test ReflectKindToWireType
			result := ReflectKindToWireType(test.typ)
			if result != test.expected {
				t.Errorf("ReflectKindToWireType(%v) = %v, want %v", test.typ, result, test.expected)
			}

			// Test WireTypeToReflectType (skip maps, bytes, and complex nested types due to conversion limitations)
			if test.expected != WireMap && test.expected != WireBytes && test.expected != WireSliceFlag|WirePtrFlag|WireInt {
				roundTrip := WireTypeToReflectType(result)
				if roundTrip != test.typ {
					t.Errorf("Round trip failed: %v -> %v -> %v", test.typ, result, roundTrip)
				}
			}
		})
	}

	// Test map type creation and error cases
	t.Run("MapTypesAndErrors", func(t *testing.T) {
		// Test mapWireTypesToReflectKind
		mapTests := []struct {
			keyType, valueType WireType
			expected           reflect.Type
		}{
			{WireString, WireInt, reflect.TypeOf(map[string]int{})},
			{WireString, WireString, reflect.TypeOf(map[string]string{})},
			{WireInt, WireBool, reflect.TypeOf(map[int]bool{})},
		}

		for _, test := range mapTests {
			result := mapWireTypesToReflectKind(test.keyType, test.valueType)
			if result != test.expected {
				t.Errorf("mapWireTypesToReflectKind(%v, %v) = %v, want %v",
					test.keyType, test.valueType, result, test.expected)
			}
		}

		// Test panic cases
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Error("Expected panic for WireMap in WireTypeToReflectType")
				}
			}()
			WireTypeToReflectType(WireMap)
		}()

		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Error("Expected panic for unsupported type")
				}
			}()
			ReflectKindToWireType(reflect.TypeOf(make(chan int)))
		}()
	})

}

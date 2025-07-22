package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kungfusheep/glint"
)

// Test struct with both slices AND arrays
type SliceAndArrayTests struct {
	// Slices (dynamic length)
	BoolSlice    []bool    `json:"boolSlice"`
	StringSlice  []string  `json:"stringSlice"`
	IntSlice     []int     `json:"intSlice"`
	Int8Slice    []int8    `json:"int8Slice"`
	Int16Slice   []int16   `json:"int16Slice"`
	Int32Slice   []int32   `json:"int32Slice"`
	Int64Slice   []int64   `json:"int64Slice"`
	UintSlice    []uint    `json:"uintSlice"`
	Uint8Slice   []uint8   `json:"uint8Slice"`
	Uint16Slice  []uint16  `json:"uint16Slice"`
	Uint32Slice  []uint32  `json:"uint32Slice"`
	Uint64Slice  []uint64  `json:"uint64Slice"`
	Float32Slice []float32 `json:"float32Slice"`
	Float64Slice []float64 `json:"float64Slice"`
	BytesData    []byte    `json:"bytesData"`

	// Arrays (fixed length) - this is what was missing!
	BoolArray   [3]bool   `json:"boolArray"`
	IntArray    [4]int    `json:"intArray"`
	StringArray [2]string `json:"stringArray"`

	// Empty slices
	EmptyStringSlice []string `json:"emptyStringSlice"`
	EmptyIntSlice    []int    `json:"emptyIntSlice"`
}

func main() {
	data := SliceAndArrayTests{
		// Slices
		BoolSlice:    []bool{true, false, true, false, true},
		StringSlice:  []string{"alpha", "beta", "gamma", "delta", "epsilon"},
		IntSlice:     []int{-100, -1, 0, 1, 100, 1000, -1000},
		Int8Slice:    []int8{-128, -1, 0, 1, 127},
		Int16Slice:   []int16{-32768, -1000, -1, 0, 1, 1000, 32767},
		Int32Slice:   []int32{-2147483648, -1000000, -1, 0, 1, 1000000, 2147483647},
		Int64Slice:   []int64{-9223372036854775808, -1000000000000, -1, 0, 1, 1000000000000, 9223372036854775807},
		UintSlice:    []uint{0, 1, 10, 100, 1000, 4294967295},
		Uint8Slice:   []uint8{0, 1, 4, 100, 255},
		Uint16Slice:  []uint16{0, 1, 100, 1000, 65535},
		Uint32Slice:  []uint32{0, 1, 1000, 1000000, 4294967295},
		Uint64Slice:  []uint64{0, 1, 1000000000, 18446744073709551615},
		Float32Slice: []float32{-3.14159, 0, 1, 2.71828, 3.14159, 1e10, -1e10},
		Float64Slice: []float64{-3.141592653589793, 0, 1, 2.718281828459045, 3.141592653589793, 1e100, -1e100},
		BytesData:    []byte{0x00, 0x01, 0xFF, 0x7F, 0x80, 0xAA, 0x55, 0xDE, 0xAD, 0xBE, 0xEF},

		// Arrays (fixed length)
		BoolArray:   [3]bool{true, false, true},
		IntArray:    [4]int{10, 20, 30, 40},
		StringArray: [2]string{"first", "second"},

		// Empty slices
		EmptyStringSlice: []string{},
		EmptyIntSlice:    []int{},
	}

	// Generate JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		panic(err)
	}
	
	err = os.WriteFile("./test/array-tests.json", jsonData, 0644)
	if err != nil {
		panic(err)
	}

	// Generate Glint binary
	glintData, err := glint.Marshal(data)
	if err != nil {
		panic(err)
	}
	
	err = os.WriteFile("./test/array-tests.glint", glintData, 0644)
	if err != nil {
		panic(err)
	}

	fmt.Printf("✅ Generated test data with %d slices + 3 arrays\n", 15)
	fmt.Printf("   JSON: %d bytes\n", len(jsonData))
	fmt.Printf("   Glint: %d bytes\n", len(glintData))
}
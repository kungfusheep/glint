package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kungfusheep/glint"
)

// SimpleSliceTests contains basic slice types only
type SimpleSliceTests struct {
	// Basic slice types
	BoolSlice   []bool   `json:"boolSlice" glint:"boolSlice"`
	StringSlice []string `json:"stringSlice" glint:"stringSlice"`

	// Integer slices (all variants)
	IntSlice   []int   `json:"intSlice" glint:"intSlice"`
	Int8Slice  []int8  `json:"int8Slice" glint:"int8Slice"`
	Int16Slice []int16 `json:"int16Slice" glint:"int16Slice"`
	Int32Slice []int32 `json:"int32Slice" glint:"int32Slice"`
	Int64Slice []int64 `json:"int64Slice" glint:"int64Slice"`

	UintSlice   []uint   `json:"uintSlice" glint:"uintSlice"`
	Uint8Slice  []uint8  `json:"uint8Slice" glint:"uint8Slice"`
	Uint16Slice []uint16 `json:"uint16Slice" glint:"uint16Slice"`
	Uint32Slice []uint32 `json:"uint32Slice" glint:"uint32Slice"`
	Uint64Slice []uint64 `json:"uint64Slice" glint:"uint64Slice"`

	// Floating point slices
	Float32Slice []float32 `json:"float32Slice" glint:"float32Slice"`
	Float64Slice []float64 `json:"float64Slice" glint:"float64Slice"`

	// Byte slice (special case)
	BytesData []byte `json:"bytesData" glint:"bytesData"`

	// Arrays (fixed size)
	BoolArray   [3]bool   `json:"boolArray" glint:"boolArray"`
	IntArray    [4]int    `json:"intArray" glint:"intArray"`
	StringArray [2]string `json:"stringArray" glint:"stringArray"`

	// Edge cases
	EmptyStringSlice []string `json:"emptyStringSlice" glint:"emptyStringSlice"`
	EmptyIntSlice    []int    `json:"emptyIntSlice" glint:"emptyIntSlice"`
}

func main() {
	testData := SimpleSliceTests{
		// Basic slices
		BoolSlice:   []bool{true, false, true, false, true},
		StringSlice: []string{"alpha", "beta", "gamma", "delta", "epsilon"},

		// Integer slices with diverse values
		IntSlice:   []int{-100, -1, 0, 1, 100, 1000, -1000},
		Int8Slice:  []int8{-128, -1, 0, 1, 127},
		Int16Slice: []int16{-32768, -1000, -1, 0, 1, 1000, 32767},
		Int32Slice: []int32{-2147483648, -1000000, -1, 0, 1, 1000000, 2147483647},
		Int64Slice: []int64{-9223372036854775808, -1000000000000, -1, 0, 1, 1000000000000, 9223372036854775807},

		UintSlice:   []uint{0, 1, 10, 100, 1000, 4294967295},
		Uint8Slice:  []uint8{0, 1, 10, 100, 255},
		Uint16Slice: []uint16{0, 1, 100, 1000, 65535},
		Uint32Slice: []uint32{0, 1, 1000, 1000000, 4294967295},
		Uint64Slice: []uint64{0, 1, 1000000000, 18446744073709551615},

		// Floating point slices
		Float32Slice: []float32{-3.14159, 0.0, 1.0, 2.71828, 3.14159, 1e10, -1e10},
		Float64Slice: []float64{-3.141592653589793, 0.0, 1.0, 2.718281828459045, 3.141592653589793, 1e100, -1e100},

		// Byte slice with various byte values
		BytesData: []byte{0x00, 0x01, 0xFF, 0x7F, 0x80, 0xAA, 0x55, 0xDE, 0xAD, 0xBE, 0xEF},

		// Arrays
		BoolArray:   [3]bool{true, false, true},
		IntArray:    [4]int{10, 20, 30, 40},
		StringArray: [2]string{"first", "second"},

		// Edge cases
		EmptyStringSlice: []string{},
		EmptyIntSlice:    []int{},
	}

	// Create output directory
	outputDir := "../../cmd/client-ts/test"

	// Generate JSON file
	jsonData, err := json.MarshalIndent(testData, "", "  ")
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal JSON: %v", err))
	}

	jsonPath := filepath.Join(outputDir, "slice-tests.json")
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		panic(fmt.Sprintf("Failed to write JSON file: %v", err))
	}

	// Generate Glint file using struct encoder
	encoder := glint.NewEncoder[SimpleSliceTests]()
	buffer := glint.NewBufferFromPool()
	defer buffer.ReturnToPool()

	encoder.Marshal(&testData, buffer)

	// Copy the bytes since buffer.Bytes is a slice that gets reused
	glintData := make([]byte, len(buffer.Bytes))
	copy(glintData, buffer.Bytes)

	glintPath := filepath.Join(outputDir, "slice-tests.glint")
	if err := os.WriteFile(glintPath, glintData, 0644); err != nil {
		panic(fmt.Sprintf("Failed to write Glint file: %v", err))
	}

	fmt.Printf("✅ Generated slice test files:\n")
	fmt.Printf("   JSON: %s (%d bytes)\n", jsonPath, len(jsonData))
	fmt.Printf("   Glint: %s (%d bytes)\n", glintPath, len(glintData))
	fmt.Printf("   Compression ratio: %.1f%% of JSON size\n", float64(len(glintData))/float64(len(jsonData))*100)

	fmt.Printf("\n📋 Basic Slice Types Covered:\n")
	fmt.Printf("   ✓ Boolean slices: []bool + [3]bool\n")
	fmt.Printf("   ✓ String slices: []string + [2]string\n")
	fmt.Printf("   ✓ Integer slices: []int, []int8→int64, []uint, []uint8→uint64\n")
	fmt.Printf("   ✓ Floating slices: []float32, []float64\n")
	fmt.Printf("   ✓ Byte slice: []byte\n")
	fmt.Printf("   ✓ Edge cases: empty slices\n")
}
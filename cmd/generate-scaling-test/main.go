package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"time"

	"github.com/kungfusheep/glint"
)

// LargeTestStruct contains arrays of different sizes for scaling tests
type LargeTestStruct struct {
	// Metadata
	Version   int32    `glint:"1"`
	Timestamp int64    `glint:"2"`

	// Variable-size data
	IntArray     []int32   `glint:"3"`
	FloatArray   []float64 `glint:"4"`
	StringArray  []string  `glint:"5"`
	BoolArray    []bool    `glint:"6"`

	// Some nested structs
	Records []Record `glint:"7"`
}

type Record struct {
	ID          int64   `glint:"1"`
	Name        string  `glint:"2"`
	Value       float64 `glint:"3"`
	Active      bool    `glint:"4"`
	Tags        []string `glint:"5"`
	Scores      []int32  `glint:"6"`
}

func generateTestDocument(arraySize int, recordCount int) []byte {
	// Create test data
	data := LargeTestStruct{
		Version:   1,
		Timestamp: time.Now().Unix(),
		IntArray:  make([]int32, arraySize),
		FloatArray: make([]float64, arraySize),
		StringArray: make([]string, arraySize/10), // Fewer strings as they're larger
		BoolArray: make([]bool, arraySize),
		Records: make([]Record, recordCount),
	}

	// Fill with deterministic but varied data
	rand.Seed(42) // Fixed seed for reproducibility

	for i := 0; i < arraySize; i++ {
		data.IntArray[i] = int32(rand.Intn(1000000) - 500000)
		data.FloatArray[i] = rand.Float64() * 1000.0
		data.BoolArray[i] = rand.Intn(2) == 1
	}

	for i := 0; i < arraySize/10; i++ {
		data.StringArray[i] = fmt.Sprintf("test_string_%d_with_some_padding", i)
	}

	// Generate records
	for i := 0; i < recordCount; i++ {
		record := Record{
			ID:     int64(1000 + i),
			Name:   fmt.Sprintf("Record_%d_Name_With_Some_Length", i),
			Value:  rand.Float64() * 10000.0,
			Active: i%2 == 0,
			Tags:   []string{"tag1", "tag2", "tag3", fmt.Sprintf("custom_%d", i)},
			Scores: []int32{int32(rand.Intn(100)), int32(rand.Intn(100)), int32(rand.Intn(100))},
		}
		data.Records[i] = record
	}

	// Encode to Glint
	encoder := glint.NewEncoder[LargeTestStruct]()
	buffer := &glint.Buffer{}
	encoder.Marshal(&data, buffer)

	return buffer.Bytes
}

func generateCHeader(name string, data []byte) string {
	header := fmt.Sprintf(`// Auto-generated test data: %s
#ifndef %s_H
#define %s_H

#include <stdint.h>
#include <stddef.h>

const uint8_t %s_DATA[] = {
`, name, name, name, name)

	// Write bytes in hex format
	for i, b := range data {
		if i%16 == 0 && i > 0 {
			header += "\n    "
		}
		header += fmt.Sprintf("0x%02X", b)
		if i < len(data)-1 {
			header += ", "
		}
	}

	header += fmt.Sprintf(`
};
const size_t %s_DATA_LEN = %d;

#endif // %s_H
`, name, len(data), name)

	return header
}

func main() {
	// Test configurations: (array_size, record_count)
	configs := []struct {
		name       string
		arraySize  int
		recordCount int
	}{
		{"SMALL", 100, 10},        // ~5 KB
		{"MEDIUM", 1000, 50},      // ~50 KB
		{"LARGE", 10000, 200},     // ~500 KB
		{"XLARGE", 50000, 500},    // ~2.5 MB
		{"XXLARGE", 100000, 1000}, // ~5 MB
	}

	fmt.Println("Generating scaled test documents...")
	fmt.Println("=====================================")

	for _, config := range configs {
		fmt.Printf("\nGenerating %s test document...\n", config.name)

		// Generate document
		data := generateTestDocument(config.arraySize, config.recordCount)

		// Save binary file
		binFile := fmt.Sprintf("test/data/scale_%s.glint", config.name)
		err := ioutil.WriteFile(binFile, data, 0644)
		if err != nil {
			log.Fatalf("Failed to write %s: %v", binFile, err)
		}

		// Save C header file
		headerFile := fmt.Sprintf("test/data/scale_%s.h", config.name)
		header := generateCHeader(fmt.Sprintf("SCALE_%s", config.name), data)
		err = ioutil.WriteFile(headerFile, []byte(header), 0644)
		if err != nil {
			log.Fatalf("Failed to write %s: %v", headerFile, err)
		}

		fmt.Printf("  - Array size: %d elements\n", config.arraySize)
		fmt.Printf("  - Record count: %d records\n", config.recordCount)
		fmt.Printf("  - Document size: %d bytes (%.2f KB)\n", len(data), float64(len(data))/1024.0)
		fmt.Printf("  - Files: %s, %s\n", binFile, headerFile)
	}

	fmt.Println("\n✅ All test documents generated successfully!")
}
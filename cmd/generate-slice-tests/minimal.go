package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/kungfusheep/glint"
)

// Minimal test with just 3 slice types
type MinimalSliceTest struct {
	BoolSlice   []bool  `json:"boolSlice"`
	IntSlice    []int   `json:"intSlice"`
	StringSlice []string `json:"stringSlice"`
}

func main() {
	// Create minimal test data
	test := MinimalSliceTest{
		BoolSlice:   []bool{true, false, true},
		IntSlice:    []int{10, 20, 30},
		StringSlice: []string{"hello", "world"},
	}

	// Write JSON for comparison
	jsonPath := "./cmd/client-ts/test/minimal-slice.json"
	glintPath := "./cmd/client-ts/test/minimal-slice.glint"

	jsonData, err := json.MarshalIndent(test, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(jsonPath, jsonData, 0644)
	if err != nil {
		log.Fatal(err)
	}

	// Encode as Glint using struct encoder
	encoder := glint.NewEncoder[MinimalSliceTest]()
	buffer := glint.NewBufferFromPool()
	defer buffer.ReturnToPool()

	encoder.Marshal(&test, buffer)

	// Copy the bytes since buffer.Bytes is a slice that gets reused
	encoded := make([]byte, len(buffer.Bytes))
	copy(encoded, buffer.Bytes)

	err = os.WriteFile(glintPath, encoded, 0644)
	if err != nil {
		log.Fatal(err)
	}

	jsonSize := len(jsonData)
	glintSize := len(encoded)
	compressionRatio := (float64(glintSize) / float64(jsonSize)) * 100

	fmt.Printf("✅ Generated minimal test files:\n")
	fmt.Printf("   JSON: %s (%d bytes)\n", path.Base(jsonPath), jsonSize)
	fmt.Printf("   Glint: %s (%d bytes)\n", path.Base(glintPath), glintSize)
	fmt.Printf("   Compression ratio: %.1f%% of JSON size\n", compressionRatio)
}
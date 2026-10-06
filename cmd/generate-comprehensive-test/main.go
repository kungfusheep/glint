package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kungfusheep/glint"
)

// ComprehensiveTest includes all supported Go types
type ComprehensiveTest struct {
	// Basic types (wire types 1-2, 7)
	BoolValue bool `json:"boolValue" glint:"boolValue"`
	IntValue  int  `json:"intValue" glint:"intValue"`
	UintValue uint `json:"uintValue" glint:"uintValue"`

	// Specific integer sizes (wire types 3-6, 8-11)
	Int8Value   int8   `json:"int8Value" glint:"int8Value"`
	Int16Value  int16  `json:"int16Value" glint:"int16Value"`
	Int32Value  int32  `json:"int32Value" glint:"int32Value"`
	Int64Value  int64  `json:"int64Value" glint:"int64Value"`
	Uint8Value  uint8  `json:"uint8Value" glint:"uint8Value"`
	Uint16Value uint16 `json:"uint16Value" glint:"uint16Value"`
	Uint32Value uint32 `json:"uint32Value" glint:"uint32Value"`
	Uint64Value uint64 `json:"uint64Value" glint:"uint64Value"`

	// Floating point (wire types 12-13)
	Float32Value float32 `json:"float32Value" glint:"float32Value"`
	Float64Value float64 `json:"float64Value" glint:"float64Value"`

	// String and bytes (wire types 14-15)
	StringValue string `json:"stringValue" glint:"stringValue"`
	BytesValue  []byte `json:"bytesValue" glint:"bytesValue"`

	// Time (wire type 18)
	TimeValue time.Time `json:"timeValue" glint:"timeValue"`

	// Arrays and slices
	IntArray     [3]int     `json:"intArray" glint:"intArray"`
	StringSlice  []string   `json:"stringSlice" glint:"stringSlice"`
	Float32Slice []float32  `json:"float32Slice" glint:"float32Slice"`
	BoolSlice    []bool     `json:"boolSlice" glint:"boolSlice"`
	ByteSlice    []uint8    `json:"byteSlice" glint:"byteSlice"`
	Int16Slice   []int16    `json:"int16Slice" glint:"int16Slice"`
	Uint64Slice  []uint64   `json:"uint64Slice" glint:"uint64Slice"`

	// Maps with different key/value types (wire type 17)
	StringToIntMap     map[string]int     `json:"stringToIntMap" glint:"stringToIntMap"`
	StringToFloatMap   map[string]float64 `json:"stringToFloatMap" glint:"stringToFloatMap"`
	StringToBoolMap    map[string]bool    `json:"stringToBoolMap" glint:"stringToBoolMap"`
	IntToStringMap     map[int]string     `json:"intToStringMap" glint:"intToStringMap"`
	StringToStringMap  map[string]string  `json:"stringToStringMap" glint:"stringToStringMap"`

	// Nested struct (wire type 16)
	NestedStruct NestedData `json:"nestedStruct" glint:"nestedStruct"`

	// Pointer types (nullable)
	StringPtr  *string  `json:"stringPtr" glint:"stringPtr"`
	IntPtr     *int     `json:"intPtr" glint:"intPtr"`
	BoolPtr    *bool    `json:"boolPtr" glint:"boolPtr"`
	Float64Ptr *float64 `json:"float64Ptr" glint:"float64Ptr"`

	// Null pointers
	NullStringPtr *string `json:"nullStringPtr" glint:"nullStringPtr"`
	NullIntPtr    *int    `json:"nullIntPtr" glint:"nullIntPtr"`

	// Complex nested data
	UserProfiles []UserProfile `json:"userProfiles" glint:"userProfiles"`
}

type NestedData struct {
	ID          int64             `json:"id" glint:"id"`
	Name        string            `json:"name" glint:"name"`
	Active      bool              `json:"active" glint:"active"`
	Metadata    map[string]string `json:"metadata" glint:"metadata"`
	Scores      []float32         `json:"scores" glint:"scores"`
	LastUpdated time.Time         `json:"lastUpdated" glint:"lastUpdated"`
}

type UserProfile struct {
	UserID       uint32            `json:"userId" glint:"userId"`
	Username     string            `json:"username" glint:"username"`
	Age          uint8             `json:"age" glint:"age"`
	Height       float32           `json:"height" glint:"height"`
	IsVerified   bool              `json:"isVerified" glint:"isVerified"`
	JoinDate     time.Time         `json:"joinDate" glint:"joinDate"`
	Preferences  map[string]bool   `json:"preferences" glint:"preferences"`
	Tags         []string          `json:"tags" glint:"tags"`
	Scores       map[string]int16  `json:"scores" glint:"scores"`
	LastActivity *time.Time        `json:"lastActivity" glint:"lastActivity"`
}

func main() {
	// Create comprehensive test data
	now := time.Now()
	lastWeek := now.AddDate(0, 0, -7)
	
	// Helper function to create pointers
	stringPtr := func(s string) *string { return &s }
	intPtr := func(i int) *int { return &i }
	boolPtr := func(b bool) *bool { return &b }
	float64Ptr := func(f float64) *float64 { return &f }
	timePtr := func(t time.Time) *time.Time { return &t }

	testData := ComprehensiveTest{
		// Basic types
		BoolValue: true,
		IntValue:  -42,
		UintValue: 42,

		// Specific integer sizes
		Int8Value:   -128,
		Int16Value:  -32000,
		Int32Value:  -2000000000,
		Int64Value:  -9000000000000000000,
		Uint8Value:  255,
		Uint16Value: 65000,
		Uint32Value: 4000000000,
		Uint64Value: 18000000000000000000,

		// Floating point
		Float32Value: 3.14159,
		Float64Value: 2.718281828459045,

		// String and bytes
		StringValue: "Hello, Glint! 🚀",
		BytesValue:  []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD},

		// Time
		TimeValue: now,

		// Arrays and slices
		IntArray:     [3]int{1, 2, 3},
		StringSlice:  []string{"alpha", "beta", "gamma", "delta"},
		Float32Slice: []float32{1.1, 2.2, 3.3, 4.4, 5.5},
		BoolSlice:    []bool{true, false, true, false},
		ByteSlice:    []uint8{10, 20, 30, 40, 50},
		Int16Slice:   []int16{-100, -200, 300, 400},
		Uint64Slice:  []uint64{1000000, 2000000, 3000000},

		// Maps with different types
		StringToIntMap: map[string]int{
			"one":   1,
			"two":   2,
			"three": 3,
			"negative": -10,
		},
		StringToFloatMap: map[string]float64{
			"pi":  3.14159,
			"e":   2.71828,
			"phi": 1.61803,
		},
		StringToBoolMap: map[string]bool{
			"enabled":  true,
			"disabled": false,
			"active":   true,
		},
		IntToStringMap: map[int]string{
			1: "first",
			2: "second",
			3: "third",
		},
		StringToStringMap: map[string]string{
			"color":    "blue",
			"size":     "large",
			"material": "cotton",
		},

		// Nested struct
		NestedStruct: NestedData{
			ID:     12345,
			Name:   "NestedExample",
			Active: true,
			Metadata: map[string]string{
				"version": "1.0",
				"env":     "test",
			},
			Scores:      []float32{95.5, 87.2, 91.8},
			LastUpdated: lastWeek,
		},

		// Pointer types
		StringPtr:  stringPtr("pointer string"),
		IntPtr:     intPtr(999),
		BoolPtr:    boolPtr(false),
		Float64Ptr: float64Ptr(123.456),

		// Null pointers
		NullStringPtr: nil,
		NullIntPtr:    nil,

		// Complex nested data
		UserProfiles: []UserProfile{
			{
				UserID:      1001,
				Username:    "alice_dev",
				Age:         28,
				Height:      165.5,
				IsVerified:  true,
				JoinDate:    lastWeek,
				Preferences: map[string]bool{
					"dark_mode":     true,
					"notifications": false,
					"beta_features": true,
				},
				Tags: []string{"developer", "go", "typescript"},
				Scores: map[string]int16{
					"coding":      95,
					"design":      75,
					"leadership":  88,
				},
				LastActivity: timePtr(now),
			},
			{
				UserID:      1002,
				Username:    "bob_designer",
				Age:         32,
				Height:      178.2,
				IsVerified:  false,
				JoinDate:    now.AddDate(0, -2, 0),
				Preferences: map[string]bool{
					"dark_mode":     false,
					"notifications": true,
					"beta_features": false,
				},
				Tags: []string{"designer", "ui", "ux"},
				Scores: map[string]int16{
					"coding":      60,
					"design":      98,
					"leadership":  82,
				},
				LastActivity: nil,
			},
		},
	}

	// Create output directory
	outputDir := "../../cmd/client-ts/test"
	
	// Generate JSON file
	jsonData, err := json.MarshalIndent(testData, "", "  ")
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal JSON: %v", err))
	}
	
	jsonPath := filepath.Join(outputDir, "comprehensive.json")
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		panic(fmt.Sprintf("Failed to write JSON file: %v", err))
	}
	
	// Generate Glint file using struct encoder
	encoder := glint.NewEncoder[ComprehensiveTest]()
	buffer := glint.NewBufferFromPool()
	defer buffer.ReturnToPool()
	
	encoder.Marshal(&testData, buffer)
	
	// Copy the bytes since buffer.Bytes is a slice that gets reused
	glintData := make([]byte, len(buffer.Bytes))
	copy(glintData, buffer.Bytes)
	
	glintPath := filepath.Join(outputDir, "comprehensive.glint")
	if err := os.WriteFile(glintPath, glintData, 0644); err != nil {
		panic(fmt.Sprintf("Failed to write Glint file: %v", err))
	}
	
	fmt.Printf("✅ Generated comprehensive test files:\n")
	fmt.Printf("   JSON: %s (%d bytes)\n", jsonPath, len(jsonData))
	fmt.Printf("   Glint: %s (%d bytes)\n", glintPath, len(glintData))
	fmt.Printf("   Compression ratio: %.1f%% of JSON size\n", float64(len(glintData))/float64(len(jsonData))*100)
	
	// Print type coverage summary
	fmt.Printf("\n📋 Type Coverage:\n")
	fmt.Printf("   ✓ All basic types: bool, int, uint, int8, uint8\n")
	fmt.Printf("   ✓ Extended integers: int16, int32, int64, uint16, uint32, uint64\n")
	fmt.Printf("   ✓ Floating point: float32, float64\n")
	fmt.Printf("   ✓ Collections: string, []byte, arrays, slices\n")
	fmt.Printf("   ✓ Maps: string→int, string→float64, int→string, etc.\n")
	fmt.Printf("   ✓ Nested structs and complex data\n")
	fmt.Printf("   ✓ Pointers and nullable types\n")
	fmt.Printf("   ✓ Time values with nanosecond precision\n")
}
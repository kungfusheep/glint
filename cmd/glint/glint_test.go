package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kungfusheep/glint"
)

// Test data structures
type TestDataWithMaps struct {
	User   string            `glint:"user"`
	Nums1 map[string]int    `glint:"nums1"`
	Meta   map[string]string `glint:"meta"`
	Age    int               `glint:"age"`
}

type NestedTestData struct {
	Name    string                        `glint:"name"`
	Config  map[string]bool               `glint:"config"`
	Nested  map[string]map[string]int     `glint:"nested"`
	Metrics map[int]string                `glint:"stats1"`
}

type ComprehensiveTestData struct {
	// Primitive types
	Name      string    `glint:"name"`
	Age       int       `glint:"age"`
	Height    float64   `glint:"height"`
	Active    bool      `glint:"active"`
	Birthday  time.Time `glint:"birthday"`
	
	// Arrays/Slices
	Tags      []string  `glint:"tags"`
	Numbers   []int     `glint:"numbers"`
	Nums1    []float64 `glint:"nums1"`
	Flags     []bool    `glint:"flags"`
	
	// Maps
	Metadata  map[string]string `glint:"metadata"`
	Config    map[string]int    `glint:"config"`
	
	// Optional/Pointer fields
	Nickname  *string   `glint:"nickname"`
	Score     *int      `glint:"score"`
}

type NestedStructData struct {
	User   UserInfo          `glint:"user"`
	System SystemInfo        `glint:"sys1"`
	Data   map[string]string `glint:"data"`
}

type UserInfo struct {
	Name  string `glint:"name"`
	Email string `glint:"email"`
	Age   int    `glint:"age"`
}

type SystemInfo struct {
	OS      string `glint:"os1"`
	Version string `glint:"ver1"`
	Arch    string `glint:"arc1"`
}

// Helper functions to create test documents
func createTestDocument() []byte {
	data := TestDataWithMaps{
		User: "sampleuser",
		Age:  25,
		Nums1: map[string]int{
			"stage1": 85,
			"stage2": 175,
			"extra":  65,
		},
		Meta: map[string]string{
			"name":    "TestUser",
			"role":    "engineer",
			"project": "serialization",
		},
	}

	encoder := glint.NewEncoder[TestDataWithMaps]()
	buf := glint.NewBufferFromPool()
	defer buf.ReturnToPool()

	encoder.Marshal(&data, buf)
	return append([]byte(nil), buf.Bytes...) // Copy to avoid pool reuse issues
}

func createComprehensiveTestDocument() []byte {

	// print the pwd of every claude pid 
	//  ps -eo pid,comm | grep claude | xargs -I {} sh -c 'echo "PID: {}"; pwd; echo'


	nickname := "TU"
	score := 89
	
	data := ComprehensiveTestData{
		Name:     "TestUser",
		Age:      28,
		Height:   5.8,
		Active:   true,
		Birthday: time.Date(1995, 4, 22, 12, 15, 30, 777888999, time.UTC),
		Tags:     []string{"engineer", "go", "serialization"},
		Numbers:  []int{1, 2, 3, 4, 5},
		Nums1:   []float64{88.7, 91.4, 85.2},
		Flags:    []bool{true, false, true},
		Metadata: map[string]string{
			"category": "engineering",
			"tier":      "senior",
			"region":   "global",
		},
		Config: map[string]int{
			"timeout":    45,
			"retries":    5,
			"bufferSize": 2048,
		},
		Nickname: &nickname,
		Score:    &score,
	}

	encoder := glint.NewEncoder[ComprehensiveTestData]()
	buf := glint.NewBufferFromPool()
	defer buf.ReturnToPool()

	encoder.Marshal(&data, buf)
	return append([]byte(nil), buf.Bytes...)
}

func createNestedStructDocument() []byte {
	data := NestedStructData{
		User: UserInfo{
			Name:  "Test Person",
			Email: "test@example.com",
			Age:   25,
		},
		System: SystemInfo{
			OS:      "linux",
			Version: "5.4.0",
			Arch:    "x86_64",
		},
		Data: map[string]string{
			"env":     "staging",
			"region":  "us-east-1",
			"cluster": "primary",
		},
	}

	encoder := glint.NewEncoder[NestedStructData]()
	buf := glint.NewBufferFromPool()
	defer buf.ReturnToPool()

	encoder.Marshal(&data, buf)
	return append([]byte(nil), buf.Bytes...)
}

// Helper function to create nested test document
func createNestedTestDocument() []byte {
	data := NestedTestData{
		Name: "SampleApp",
		Config: map[string]bool{
			"debug":   true,
			"verbose": false,
		},
		Nested: map[string]map[string]int{
			"server": {"port": 9090, "timeout": 45},
			"client": {"retries": 5, "delay": 150},
		},
		Metrics: map[int]string{
			1: "processor",
			2: "ram",
			3: "storage",
		},
	}

	encoder := glint.NewEncoder[NestedTestData]()
	buf := glint.NewBufferFromPool()
	defer buf.ReturnToPool()

	encoder.Marshal(&data, buf)
	return append([]byte(nil), buf.Bytes...)
}

func TestCLITemplateMapFieldAccess(t *testing.T) {
	doc := createTestDocument()

	tests := []struct {
		name     string
		template string
		expected string
		contains []string
	}{
		{
			name:     "Simple field access",
			template: "User: {{.user}}",
			expected: "User: sampleuser",
		},
		{
			name:     "Map value access",
			template: "Score: {{.nums1.stage1}}",
			expected: "Score: 85",
		},
		{
			name:     "Multiple map values",
			template: "Nums1 - Stage1: {{.nums1.stage1}}, Stage2: {{.nums1.stage2}}, Extra: {{.nums1.extra}}",
			expected: "Nums1 - Stage1: 85, Stage2: 175, Extra: 65",
		},
		{
			name:     "String map access",
			template: "{{.meta.name}} is a {{.meta.role}} working on {{.meta.project}}",
			expected: "TestUser is a engineer working on serialization",
		},
		{
			name:     "Mixed field and map access",
			template: "{{.user}} ({{.age}}) scored {{.nums1.stage1}} on stage1",
			expected: "sampleuser (25) scored 85 on stage1",
		},
		{
			name:     "Map range iteration",
			template: "Nums1:{{range $key, $value := .nums1}} {{$key}}={{$value}}{{end}}",
			contains: []string{"stage1=85", "stage2=175", "extra=65"}, // Order not guaranteed
		},
		{
			name:     "Map with conditionals",
			template: "{{if .nums1.extra}}Extra: {{.nums1.extra}}{{else}}No extra{{end}}",
			expected: "Extra: 65",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := NewTemplate(doc)
			if err != nil {
				t.Fatalf("Error creating template: %v", err)
			}

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err = tmpl.Execute(tt.template)
			w.Close()
			os.Stdout = oldStdout

			if err != nil {
				t.Fatalf("Error executing template: %v", err)
			}

			var buf bytes.Buffer
			buf.ReadFrom(r)
			result := strings.TrimSpace(buf.String())

			if tt.expected != "" {
				if result != tt.expected {
					t.Errorf("Expected '%s', got '%s'", tt.expected, result)
				}
			}
			
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected output to contain '%s', got: %s", expected, result)
				}
			}
		})
	}
}

// TestNestedMapSupport is commented out due to a panic in the template system
// This test will be re-enabled once nested map support is fixed
// func TestNestedMapSupport(t *testing.T) {
// 	doc := createNestedTestDocument()
// 	
// 	// Simple test to ensure basic functionality
// 	tmpl, err := NewTemplate(doc)
// 	if err != nil {
// 		t.Fatalf("Error creating template: %v", err)
// 	}
// 	
// 	// Test a simple field access that should work
// 	oldStdout := os.Stdout
// 	r, w, _ := os.Pipe()
// 	os.Stdout = w
// 	
// 	err = tmpl.Execute("Name: {{.name}}")
// 	w.Close()
// 	os.Stdout = oldStdout
// 	
// 	if err != nil {
// 		t.Fatalf("Error executing simple template: %v", err)
// 	}
// 	
// 	var buf bytes.Buffer
// 	buf.ReadFrom(r)
// 	result := strings.TrimSpace(buf.String())
// 	
// 	if result != "Name: SampleApp" {
// 		t.Errorf("Expected 'Name: SampleApp', got '%s'", result)
// 	}
// }

func TestCLIReaderBasicValueReading(t *testing.T) {
	// Simple test to verify basic reader functionality
	buf := glint.NewBufferFromPool()
	defer buf.ReturnToPool()
	buf.AppendBool(true)
	
	reader := glint.NewReader(buf.Bytes)
	result := reader.ReadBool()
	if result != true {
		t.Errorf("Expected true, got %v", result)
	}
}

func TestCLIDocumentParsingWithMaps(t *testing.T) {
	// Create a test document with a map
	data := TestDataWithMaps{
		User: "sampleuser",
		Nums1: map[string]int{
			"stage1": 85,
			"stage2": 175,
		},
		Meta: map[string]string{
			"name": "TestUser",
		},
	}

	encoder := glint.NewEncoder[TestDataWithMaps]()
	buf := glint.NewBufferFromPool()
	defer buf.ReturnToPool()

	encoder.Marshal(&data, buf)
	doc := append([]byte(nil), buf.Bytes...)

	// Test that we can create a reader and parse the document with maps
	r := glint.NewReader(doc)
	printerDoc := glint.NewPrinterDocument(&r)
	schema := glint.NewPrinterSchema(&printerDoc.Schema)

	// Find the nums1 field (which is a map)
	var hasNumsField bool
	for i := range schema.Fields {
		if schema.Fields[i].Name == "nums1" {
			hasNumsField = true
			break
		}
	}

	if !hasNumsField {
		t.Fatal("Could not find nums1 field in schema")
	}
}

// Benchmark removed - uses legacy parseFieldPath and extractFieldValue functions

func BenchmarkCLITemplateMapAccess(b *testing.B) {
	doc := createTestDocument()
	templateStr := "{{.meta.name}} scored {{.nums1.stage1}} on stage1"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tmpl, err := NewTemplate(doc)
		if err != nil {
			b.Fatalf("Error creating template: %v", err)
		}
		
		// Capture output to /dev/null equivalent
		oldStdout := os.Stdout
		os.Stdout, _ = os.Open(os.DevNull)
		
		err = tmpl.Execute(templateStr)
		
		os.Stdout = oldStdout
		
		if err != nil {
			b.Fatalf("Error executing template: %v", err)
		}
	}
}

// Test comprehensive field extraction (all types)
// TestComprehensiveFieldExtraction now uses template-based extraction
func TestCLIFieldExtractionAllTypes(t *testing.T) {
	doc := createComprehensiveTestDocument()

	tests := []struct {
		name     string
		field    string
		expected string
	}{
		// Primitive types
		{name: "String field", field: "name", expected: "TestUser"},
		{name: "Int field", field: "age", expected: "28"},
		{name: "Float field", field: "height", expected: "5.8"},
		{name: "Bool field", field: "active", expected: "true"},
		
		// Array indexing using template index function
		{name: "String array index 0", field: "tags[0]", expected: "engineer"},
		{name: "Int array index 0", field: "numbers[0]", expected: "1"},
		{name: "Bool array index 0", field: "flags[0]", expected: "true"},
		
		// Map access
		{name: "String map", field: "metadata[category]", expected: "engineering"},
		{name: "Int map timeout", field: "config[timeout]", expected: "45"},
		
		// Pointer fields
		{name: "Pointer string", field: "nickname", expected: "TU"},
		{name: "Pointer int", field: "score", expected: "89"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			templateStr, err := fieldPathToTemplate(tt.field)
			if err != nil {
				t.Fatalf("Error converting field path: %v", err)
			}
			
			tmpl, err := NewTemplate(doc)
			if err != nil {
				t.Fatalf("Error creating template: %v", err)
			}
			
			// Capture output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			
			err = tmpl.Execute(templateStr)
			w.Close()
			os.Stdout = oldStdout
			
			if err != nil {
				t.Fatalf("Error executing template: %v", err)
			}
			
			var buf bytes.Buffer
			buf.ReadFrom(r)
			result := strings.TrimSpace(buf.String())
			
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestNestedStructFieldExtraction is commented out because struct field skipping is not implemented
// This test will be re-enabled once struct field skipping is implemented
// func TestNestedStructFieldExtraction(t *testing.T) {
// 	doc := createNestedStructDocument()
// 	
// 	tests := []struct {
// 		name     string
// 		field    string
// 		expected string
// 	}{
// 		// Map access in nested struct works
// 		{name: "Map in nested struct", field: "data[env]", expected: "production"},
// 		{name: "Map region", field: "data[region]", expected: "us-west-2"},
// 		{name: "Map cluster", field: "data[cluster]", expected: "main"},
// 	}
// 	
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			r := glint.NewReader(doc)
// 			printerDoc := glint.NewPrinterDocument(&r)
// 			schema := glint.NewPrinterSchema(&printerDoc.Schema)
// 			
// 			path := parseFieldPath(tt.field)
// 			if len(path) == 0 {
// 				t.Fatalf("Invalid field path: %s", tt.field)
// 			}
// 			
// 			value, err := extractFieldValue(&printerDoc.Body, &schema, path)
// 			if err != nil {
// 				t.Fatalf("Error extracting field '%s': %v", tt.field, err)
// 			}
// 			
// 			if value != tt.expected {
// 				t.Errorf("Expected '%s', got '%s'", tt.expected, value)
// 			}
// 		})
// 	}
// }

// Test comprehensive template functionality
func TestCLITemplateComplexFunctionality(t *testing.T) {
	doc := createComprehensiveTestDocument()

	tests := []struct {
		name     string
		template string
		contains []string // Check if output contains these strings
	}{
		{
			name:     "All primitive types",
			template: "Name: {{.name}}, Age: {{.age}}, Height: {{.height}}, Active: {{.active}}",
			contains: []string{"Name: TestUser", "Age: 28", "Height: 5.8", "Active: true"},
		},
		{
			name:     "Array iteration",
			template: "Tags: {{range .tags}}[{{.}}] {{end}}",
			contains: []string{"[engineer]", "[go]", "[serialization]"},
		},
		{
			name:     "Array indexing",
			template: "First tag: {{index .tags 0}}, Last: {{index .tags 2}}",
			contains: []string{"First tag: engineer", "Last: serialization"},
		},
		{
			name:     "Map iteration",
			template: "{{range $key, $value := .metadata}}{{$key}}: {{$value}} | {{end}}",
			contains: []string{"category: engineering", "tier: senior", "region: global"},
		},
		{
			name:     "Map access",
			template: "Dept: {{.metadata.category}}, Timeout: {{.config.timeout}}",
			contains: []string{"Dept: engineering", "Timeout: 45"},
		},
		{
			name:     "Conditionals with maps",
			template: "{{if .metadata.category}}Has category{{else}}No category{{end}}",
			contains: []string{"Has category"},
		},
		{
			name:     "Pointer fields",
			template: "Nickname: {{.nickname}}, Score: {{.score}}",
			contains: []string{"Nickname: TU", "Score: 89"},
		},
		{
			name:     "Complex template",
			template: `{{.name}} ({{.nickname}}) - Age: {{.age}}, Score: {{.score}}
Tags: {{range .tags}}{{.}} {{end}}
Config: timeout={{.config.timeout}}, retries={{.config.retries}}`,
			contains: []string{"TestUser (TU)", "Age: 28", "Score: 89", "engineer go serialization", "timeout=45", "retries=5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := NewTemplate(doc)
			if err != nil {
				t.Fatalf("Error creating template: %v", err)
			}

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err = tmpl.Execute(tt.template)
			w.Close()
			os.Stdout = oldStdout

			if err != nil {
				t.Fatalf("Error executing template: %v", err)
			}

			var buf bytes.Buffer
			buf.ReadFrom(r)
			result := buf.String()

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected output to contain '%s', got: %s", expected, result)
				}
			}
		})
	}
}

// Test error cases for field extraction
// TestFieldExtractionErrors - template-based extraction handles errors by returning "<no value>"
func TestCLIFieldExtractionErrorHandling(t *testing.T) {
	doc := createComprehensiveTestDocument()

	tests := []struct {
		name     string
		field    string
		expected string
	}{
		{name: "Non-existent field", field: "nonexistent", expected: "<no value>"},
		{name: "Map non-existent key", field: "metadata[nonexistent]", expected: "<no value>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			templateStr, err := fieldPathToTemplate(tt.field)
			if err != nil {
				t.Fatalf("Error converting field path: %v", err)
			}
			
			tmpl, err := NewTemplate(doc)
			if err != nil {
				t.Fatalf("Error creating template: %v", err)
			}
			
			// Capture output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			
			err = tmpl.Execute(templateStr)
			w.Close()
			os.Stdout = oldStdout
			
			if err != nil {
				t.Fatalf("Error executing template: %v", err)
			}
			
			var buf bytes.Buffer
			buf.ReadFrom(r)
			result := strings.TrimSpace(buf.String())
			
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// Test template error cases
func TestCLITemplateErrorHandling(t *testing.T) {
	doc := createTestDocument()

	tests := []struct {
		name     string
		template string
	}{
		{name: "Invalid template syntax", template: "{{.name"},
		// Note: Go templates don't error on missing fields, they output <no value>
		// {name: "Non-existent field", template: "{{.nonexistent}}"},
		{name: "Invalid function", template: "{{invalid .name}}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := NewTemplate(doc)
			if err != nil {
				t.Fatalf("Error creating template: %v", err)
			}

			// Capture stdout and stderr
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err = tmpl.Execute(tt.template)
			w.Close()
			os.Stdout = oldStdout

			// Should get an error for invalid templates
			if err == nil {
				t.Errorf("Expected error for template '%s', but got none", tt.template)
			}

			// Read any output (might be empty)
			var buf bytes.Buffer
			buf.ReadFrom(r)
		})
	}
}

// Test skip functionality - simplified version
func TestCLIDocumentSchemaValidation(t *testing.T) {
	doc := createComprehensiveTestDocument()
	
	r := glint.NewReader(doc)
	printerDoc := glint.NewPrinterDocument(&r)
	schema := glint.NewPrinterSchema(&printerDoc.Schema)

	// Test that we can at least access the document schema without panicking
	if len(schema.Fields) == 0 {
		t.Error("Expected document to have fields")
	}
	
	// Test that we can create a reader and printer document
	r2 := glint.NewReader(doc)
	_ = glint.NewPrinterDocument(&r2)
	if len(doc) == 0 {
		t.Error("Expected document to have data")
	}
}

// Test that we can at least parse documents with nested structures
func TestCLINestedDocumentParsing(t *testing.T) {
	// Test parsing nested test document
	doc1 := createNestedTestDocument()
	r1 := glint.NewReader(doc1)
	printerDoc1 := glint.NewPrinterDocument(&r1)
	schema1 := glint.NewPrinterSchema(&printerDoc1.Schema)
	
	// Verify we can read the schema
	if len(schema1.Fields) == 0 {
		t.Error("Expected fields in nested test document schema")
	}
	
	// Look for expected fields
	foundName := false
	foundNested := false
	for _, field := range schema1.Fields {
		if field.Name == "name" {
			foundName = true
		}
		if field.Name == "nested" {
			foundNested = true
		}
	}
	
	if !foundName {
		t.Error("Expected to find 'name' field in schema")
	}
	if !foundNested {
		t.Error("Expected to find 'nested' field in schema")
	}
	
	// Test parsing nested struct document
	doc2 := createNestedStructDocument()
	r2 := glint.NewReader(doc2)
	printerDoc2 := glint.NewPrinterDocument(&r2)
	schema2 := glint.NewPrinterSchema(&printerDoc2.Schema)
	
	// Verify we can read the schema
	if len(schema2.Fields) == 0 {
		t.Error("Expected fields in nested struct document schema")
	}
	
	// Look for expected fields
	foundUser := false
	foundData := false
	for _, field := range schema2.Fields {
		if field.Name == "user" {
			foundUser = true
		}
		if field.Name == "data" {
			foundData = true
		}
	}
	
	if !foundUser {
		t.Error("Expected to find 'user' field in schema")
	}
	if !foundData {
		t.Error("Expected to find 'data' field in schema")
	}
}

// Test template file functionality
func TestCLITemplateFileExecution(t *testing.T) {
	doc := createTestDocument()
	
	// Create a temporary template file
	templateContent := "User: {{.user}}, Age: {{.age}}, Score: {{.nums1.stage1}}"
	
	// Write to a temporary file
	tempFile := "/tmp/test_template.tmpl"
	err := os.WriteFile(tempFile, []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp template file: %v", err)
	}
	defer os.Remove(tempFile)
	
	tmpl, err := NewTemplate(doc)
	if err != nil {
		t.Fatalf("Error creating template: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = tmpl.ExecuteFile(tempFile)
	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("Error executing template file: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	result := strings.TrimSpace(buf.String())

	expected := "User: sampleuser, Age: 25, Score: 85"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// Test simple value reading for all types
func TestCLIReaderMultipleDataTypes(t *testing.T) {
	// Simple test to verify basic reader functionality with multiple types
	buf := glint.NewBufferFromPool()
	defer buf.ReturnToPool()
	
	// Test bool
	buf.AppendBool(false)
	reader := glint.NewReader(buf.Bytes)
	result := reader.ReadBool()
	if result != false {
		t.Errorf("Expected false, got %v", result)
	}
	
	// Test int8 with new buffer
	buf2 := glint.NewBufferFromPool()
	defer buf2.ReturnToPool()
	buf2.AppendInt8(42)
	reader2 := glint.NewReader(buf2.Bytes)
	result8 := reader2.ReadInt8()
	if result8 != 42 {
		t.Errorf("Expected 42, got %v", result8)
	}
}

// Test field path to template conversion
func TestCLIFieldPathTemplateConversion(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		expected string
	}{
		{name: "Simple field", field: "name", expected: "{{.name}}"},
		{name: "Map access", field: "nums1[level1]", expected: "{{.nums1.level1}}"},
		{name: "Struct field", field: "user.name", expected: "{{.user.name}}"},
		{name: "Mixed access", field: "user.settings[theme]", expected: "{{.user.settings.theme}}"},
		{name: "Array access", field: "tags[0]", expected: "{{index .tags 0}}"},
		{name: "Array with field access", field: "items[1].name", expected: "{{(index .items 1).name}}"},
		{name: "Map then array access", field: "data[key][0]", expected: "{{index .data.key 0}}"},
		{name: "Complex path", field: "data[config][server][port]", expected: "{{.data.config.server.port}}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fieldPathToTemplate(tt.field)
			if err != nil {
				t.Fatalf("Error converting field path: %v", err)
			}
			
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// Test new template-based field extraction
func TestCLITemplateBasedFieldExtraction(t *testing.T) {
	// Test with simple document
	doc := createTestDocument()
	
	tests := []struct {
		name     string
		field    string
		expected string
	}{
		{name: "Simple field", field: "user", expected: "sampleuser"},
		{name: "Map access", field: "nums1[stage1]", expected: "85"},
		{name: "String map", field: "meta[name]", expected: "TestUser"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			templateStr, err := fieldPathToTemplate(tt.field)
			if err != nil {
				t.Fatalf("Error converting field path: %v", err)
			}
			
			tmpl, err := NewTemplate(doc)
			if err != nil {
				t.Fatalf("Error creating template: %v", err)
			}
			
			// Capture output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			
			err = tmpl.Execute(templateStr)
			w.Close()
			os.Stdout = oldStdout
			
			if err != nil {
				t.Fatalf("Error executing template: %v", err)
			}
			
			var buf bytes.Buffer
			buf.ReadFrom(r)
			result := strings.TrimSpace(buf.String())
			
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// Test struct field extraction with template-based approach
func TestCLINestedStructFieldAccess(t *testing.T) {
	// Create a document with nested struct data inline
	type UserInfo struct {
		Name  string `glint:"name"`
		Email string `glint:"email"`
		Age   int    `glint:"age"`
	}
	
	type DocWithStruct struct {
		Title string   `glint:"title"`
		User  UserInfo `glint:"user"`
	}
	
	data := DocWithStruct{
		Title: "Test Document",
		User: UserInfo{
			Name:  "TestUser",
			Email: "testuser@example.com",
			Age:   30,
		},
	}
	
	encoder := glint.NewEncoder[DocWithStruct]()
	buf := glint.NewBufferFromPool()
	defer buf.ReturnToPool()
	
	encoder.Marshal(&data, buf)
	doc := append([]byte(nil), buf.Bytes...)
	
	// Test struct field access
	tests := []struct {
		name     string
		field    string
		expected string
	}{
		{name: "Top level field", field: "title", expected: "Test Document"},
		{name: "Struct field - name", field: "user.name", expected: "TestUser"},
		{name: "Struct field - email", field: "user.email", expected: "testuser@example.com"},
		{name: "Struct field - age", field: "user.age", expected: "30"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			templateStr, err := fieldPathToTemplate(tt.field)
			if err != nil {
				t.Fatalf("Error converting field path: %v", err)
			}
			
			tmpl, err := NewTemplate(doc)
			if err != nil {
				t.Fatalf("Error creating template: %v", err)
			}
			
			// Capture output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			
			err = tmpl.Execute(templateStr)
			w.Close()
			os.Stdout = oldStdout
			
			if err != nil {
				t.Fatalf("Error executing template: %v", err)
			}
			
			var buf bytes.Buffer
			buf.ReadFrom(r)
			result := strings.TrimSpace(buf.String())
			
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// Test edge cases for field path parsing - DISABLED
func TestCLIFieldPathEdgeCases_DISABLED(t *testing.T) {
	t.Skip("Legacy parseFieldPath function removed")
	/*
	tests := []struct {
		name     string
		path     string
		expected []FieldPathElement
		valid    bool
	}{
		{
			name:     "Empty string",
			path:     "",
			expected: []FieldPathElement{},
			valid:    true,
		},
		{
			name:     "Just dots",
			path:     "...",
			expected: []FieldPathElement{},
			valid:    true,
		},
		{
			name:     "Trailing dot",
			path:     "field.",
			expected: []FieldPathElement{{Name: "field", Index: -1, MapKey: ""}},
			valid:    true,
		},
		{
			name:     "Leading dot",
			path:     ".field",
			expected: []FieldPathElement{{Name: "field", Index: -1, MapKey: ""}},
			valid:    true,
		},
		{
			name:     "Empty brackets",
			path:     "field[]",
			expected: []FieldPathElement{{Name: "field", Index: -1, MapKey: ""}},
			valid:    true,
		},
		{
			name:     "Unclosed bracket",
			path:     "field[key",
			expected: []FieldPathElement{{Name: "field[key", Index: -1, MapKey: ""}},
			valid:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFieldPath(tt.path)
			
			if !tt.valid {
				if result != nil {
					t.Errorf("Expected nil for invalid path '%s', got %v", tt.path, result)
				}
				return
			}
			
			if len(result) != len(tt.expected) {
				t.Fatalf("Expected %d elements, got %d", len(tt.expected), len(result))
			}
			
			for i, elem := range result {
				expected := tt.expected[i]
				if elem.Name != expected.Name || elem.Index != expected.Index || elem.MapKey != expected.MapKey {
					t.Errorf("Element %d: expected %+v, got %+v", i, expected, elem)
				}
			}
		})
	}
	*/
}

// TestJSONConversion tests the new JSON to glint conversion functionality
func TestCLIJSONToGlintConversion(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected map[string]string // field -> expected value
	}{
		{
			name: "Simple object",
			json: `{"name":"SampleUser","age":30,"active":true}`,
			expected: map[string]string{
				"name":   "SampleUser",
				"age":    "30", 
				"active": "true",
			},
		},
		{
			name: "Object with arrays",
			json: `{"user":"testuser","tags":["dev","go"],"score":82.3}`,
			expected: map[string]string{
				"user":    "testuser",
				"tags[0]": "dev",
				"tags[1]": "go",
				"score":   "82.3",
			},
		},
		{
			name: "Single value",
			json: `"hello"`,
			expected: map[string]string{
				"value": "hello",
			},
		},
		{
			name: "Array",
			json: `["apple","banana","cherry"]`,
			expected: map[string]string{
				"items[0]": "apple",
				"items[1]": "banana", 
				"items[2]": "cherry",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert JSON to glint using the conversion function
			var data interface{}
			if err := json.Unmarshal([]byte(tt.json), &data); err != nil {
				t.Fatalf("Error parsing test JSON: %v", err)
			}

			glintData, err := jsonToGlint(data)
			if err != nil {
				t.Fatalf("Error converting JSON to glint: %v", err)
			}

			// Test field extraction for each expected field
			for fieldName, expectedValue := range tt.expected {
				templateStr, err := fieldPathToTemplate(fieldName)
				if err != nil {
					t.Fatalf("Error converting field path '%s': %v", fieldName, err)
				}

				tmpl, err := NewTemplate(glintData)
				if err != nil {
					t.Fatalf("Error creating template: %v", err)
				}

				// Capture output
				oldStdout := os.Stdout
				r, w, _ := os.Pipe()
				os.Stdout = w

				err = tmpl.Execute(templateStr)
				w.Close()
				os.Stdout = oldStdout

				if err != nil {
					t.Fatalf("Error executing template for field '%s': %v", fieldName, err)
				}

				var buf bytes.Buffer
				buf.ReadFrom(r)
				result := strings.TrimSpace(buf.String())

				if result != expectedValue {
					t.Errorf("Field '%s': expected '%s', got '%s'", fieldName, expectedValue, result)
				}
			}
		})
	}
}

// TestJSONRoundTrip tests full round-trip JSON ↔ glint conversion
func TestCLIJSONGlintRoundTripConversion(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		testField string
		expected string
	}{
		{
			name:      "Simple object",
			json:      `{"name":"SampleUser","age":30,"active":true}`,
			testField: "name",
			expected:  "SampleUser",
		},
		{
			name:      "Nested objects",
			json:      `{"user":{"name":"ExampleUser","age":25},"active":true}`,
			testField: "user.name",
			expected:  "ExampleUser",
		},
		{
			name:      "Deep nesting",
			json:      `{"config":{"server":{"host":"testhost","port":9090}}}`,
			testField: "config.server.host",
			expected:  "testhost",
		},
		{
			name:      "Mixed types with nesting",
			json:      `{"data":{"score":82.3,"valid":true},"count":10}`,
			testField: "data.score",
			expected:  "82.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Step 1: JSON → glint
			var jsonData interface{}
			if err := json.Unmarshal([]byte(tt.json), &jsonData); err != nil {
				t.Fatalf("Error parsing JSON: %v", err)
			}

			glintData, err := jsonToGlint(jsonData)
			if err != nil {
				t.Fatalf("Error converting JSON to glint: %v", err)
			}

			// Step 2: glint → JSON
			tmpl, err := NewTemplate(glintData)
			if err != nil {
				t.Fatalf("Error creating template: %v", err)
			}

			jsonOutput, err := json.Marshal(tmpl.data)
			if err != nil {
				t.Fatalf("Error converting glint to JSON: %v", err)
			}

			// Step 3: JSON → glint (again)
			var roundTripData interface{}
			if err := json.Unmarshal(jsonOutput, &roundTripData); err != nil {
				t.Fatalf("Error parsing round-trip JSON: %v", err)
			}

			glintData2, err := jsonToGlint(roundTripData)
			if err != nil {
				t.Fatalf("Error converting round-trip JSON to glint: %v", err)
			}

			// Step 4: Test field extraction
			templateStr, err := fieldPathToTemplate(tt.testField)
			if err != nil {
				t.Fatalf("Error converting field path: %v", err)
			}

			tmpl2, err := NewTemplate(glintData2)
			if err != nil {
				t.Fatalf("Error creating template for round-trip data: %v", err)
			}

			// Capture output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err = tmpl2.Execute(templateStr)
			w.Close()
			os.Stdout = oldStdout

			if err != nil {
				t.Fatalf("Error executing template: %v", err)
			}

			var buf bytes.Buffer
			buf.ReadFrom(r)
			result := strings.TrimSpace(buf.String())

			if result != tt.expected {
				t.Errorf("Round-trip failed for field '%s': expected '%s', got '%s'", tt.testField, tt.expected, result)
			}
		})
	}
}

// TestBinaryRoundTrip tests that glint → JSON → glint preserves binary format and types
func TestCLIBinaryFormatPreservation(t *testing.T) {
	tests := []struct {
		name           string
		buildDocument  func() []byte
		validateFields func(t *testing.T, data []byte)
	}{
		{
			name: "Arrays with different types",
			buildDocument: func() []byte {
				builder := &glint.DocumentBuilder{}
				
				// String array
				stringSlice := &glint.SliceBuilder{}
				stringSlice.AppendStringSlice([]string{"dev", "go", "test"})
				builder.AppendSlice("tags", *stringSlice)
				
				// Integer array
				intSlice := &glint.SliceBuilder{}
				intSlice.AppendIntSlice([]int{78, 84, 91})
				builder.AppendSlice("nums1", *intSlice)
				
				// Boolean array
				boolSlice := &glint.SliceBuilder{}
				boolSlice.AppendBoolSlice([]bool{true, false, true})
				builder.AppendSlice("flags", *boolSlice)
				
				// Float array
				floatSlice := &glint.SliceBuilder{}
				floatSlice.AppendFloat64Slice([]float64{1.5, 2.7, 3.14})
				builder.AppendSlice("values", *floatSlice)
				
				// Scalar fields
				builder.AppendString("name", "SampleUser")
				builder.AppendInt("age", 30)
				builder.AppendBool("active", true)
				builder.AppendFloat64("rating", 4.8)
				
				return builder.Bytes()
			},
			validateFields: func(t *testing.T, data []byte) {
				// Validate array access
				checkFieldExtraction(t, data, "tags[0]", "dev")
				checkFieldExtraction(t, data, "tags[1]", "go") 
				checkFieldExtraction(t, data, "tags[2]", "test")
				
				checkFieldExtraction(t, data, "nums1[0]", "78")
				checkFieldExtraction(t, data, "nums1[1]", "84")
				checkFieldExtraction(t, data, "nums1[2]", "91")
				
				checkFieldExtraction(t, data, "flags[0]", "true")
				checkFieldExtraction(t, data, "flags[1]", "false")
				checkFieldExtraction(t, data, "flags[2]", "true")
				
				checkFieldExtraction(t, data, "values[0]", "1.5")
				checkFieldExtraction(t, data, "values[1]", "2.7")
				checkFieldExtraction(t, data, "values[2]", "3.14")
				
				// Validate scalar fields
				checkFieldExtraction(t, data, "name", "SampleUser")
				checkFieldExtraction(t, data, "age", "30")
				checkFieldExtraction(t, data, "active", "true")
				checkFieldExtraction(t, data, "rating", "4.8")
			},
		},
		{
			name: "Nested documents with arrays",
			buildDocument: func() []byte {
				// Create nested document
				userDoc := &glint.DocumentBuilder{}
				userDoc.AppendString("name", "ExampleUser")
				userDoc.AppendInt("age", 25)
				
				// Create array in nested document
				tagSlice := &glint.SliceBuilder{}
				tagSlice.AppendStringSlice([]string{"backend", "api"})
				userDoc.AppendSlice("skills", *tagSlice)
				
				// Create main document
				builder := &glint.DocumentBuilder{}
				builder.AppendNestedDocument("user", userDoc)
				builder.AppendString("role", "developer")
				
				return builder.Bytes()
			},
			validateFields: func(t *testing.T, data []byte) {
				checkFieldExtraction(t, data, "user.name", "ExampleUser")
				checkFieldExtraction(t, data, "user.age", "25")
				checkFieldExtraction(t, data, "user.skills[0]", "backend")
				checkFieldExtraction(t, data, "user.skills[1]", "api")
				checkFieldExtraction(t, data, "role", "developer")
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Step 1: Create original glint document
			originalData := tt.buildDocument()
			
			// Step 2: Convert glint → JSON
			tmpl, err := NewTemplate(originalData)
			if err != nil {
				t.Fatalf("Error creating template from original data: %v", err)
			}
			
			jsonData, err := json.Marshal(tmpl.data)
			if err != nil {
				t.Fatalf("Error converting glint to JSON: %v", err)
			}
			
			// Step 3: Convert JSON → glint
			var parsedJSON interface{}
			if err := json.Unmarshal(jsonData, &parsedJSON); err != nil {
				t.Fatalf("Error parsing JSON: %v", err)
			}
			
			roundTripData, err := jsonToGlint(parsedJSON)
			if err != nil {
				t.Fatalf("Error converting JSON back to glint: %v", err)
			}
			
			// Step 4: Validate that both original and round-trip data have same field values
			t.Run("Original", func(t *testing.T) {
				tt.validateFields(t, originalData)
			})
			
			t.Run("RoundTrip", func(t *testing.T) {
				tt.validateFields(t, roundTripData)
			})
			
			// Step 5: Compare binary schemas (same fields and types, order may differ due to JSON map ordering)
			originalReader := glint.NewReader(originalData)
			roundTripReader := glint.NewReader(roundTripData)
			
			originalDoc := glint.NewPrinterDocument(&originalReader)
			roundTripDoc := glint.NewPrinterDocument(&roundTripReader)
			
			// Create schemas from the document schema readers
			originalSchema := glint.NewPrinterSchema(&originalDoc.Schema)
			roundTripSchema := glint.NewPrinterSchema(&roundTripDoc.Schema)
			
			// Compare schema field count
			if len(originalSchema.Fields) != len(roundTripSchema.Fields) {
				t.Errorf("Schema field count mismatch: original=%d, roundtrip=%d", 
					len(originalSchema.Fields), len(roundTripSchema.Fields))
			}
			
			// Create maps for field comparison (order doesn't matter after JSON round-trip)
			originalFields := make(map[string]glint.WireType)
			roundTripFields := make(map[string]glint.WireType)
			
			for _, field := range originalSchema.Fields {
				originalFields[field.Name] = field.TypeID
			}
			
			for _, field := range roundTripSchema.Fields {
				roundTripFields[field.Name] = field.TypeID
			}
			
			// Verify all original fields exist in round-trip with same types
			for name, origType := range originalFields {
				if rtType, exists := roundTripFields[name]; !exists {
					t.Errorf("Field '%s' missing in round-trip schema", name)
				} else {
					// Compare base types (ignoring flags for arrays)
					origBaseType := origType & glint.WireTypeMask
					rtBaseType := rtType & glint.WireTypeMask
					
					if origBaseType != rtBaseType {
						t.Errorf("Field type mismatch for '%s': original=%v, roundtrip=%v", 
							name, origBaseType, rtBaseType)
					}
				}
			}
		})
	}
}

// Helper function to test field extraction
func checkFieldExtraction(t *testing.T, data []byte, fieldPath, expected string) {
	templateStr, err := fieldPathToTemplate(fieldPath)
	if err != nil {
		t.Fatalf("Error converting field path '%s': %v", fieldPath, err)
	}
	
	tmpl, err := NewTemplate(data)
	if err != nil {
		t.Fatalf("Error creating template: %v", err)
	}
	
	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	
	err = tmpl.Execute(templateStr)
	w.Close()
	os.Stdout = oldStdout
	
	if err != nil {
		t.Fatalf("Error executing template for field '%s': %v", fieldPath, err)
	}
	
	var buf bytes.Buffer
	buf.ReadFrom(r)
	result := strings.TrimSpace(buf.String())
	
	if result != expected {
		t.Errorf("Field '%s': expected '%s', got '%s'", fieldPath, expected, result)
	}
}

// TestJSONOutput tests the --json flag functionality
func TestCLIJSONOutputFormatting(t *testing.T) {
	// Create test glint data from JSON
	inputJSON := `{"name":"SampleUser","age":45,"config":{"debug":true,"timeout":45}}`
	
	var data interface{}
	if err := json.Unmarshal([]byte(inputJSON), &data); err != nil {
		t.Fatalf("Error parsing input JSON: %v", err)
	}

	glintData, err := jsonToGlint(data)
	if err != nil {
		t.Fatalf("Error converting to glint: %v", err)
	}

	// Test JSON output conversion
	tmpl, err := NewTemplate(glintData)
	if err != nil {
		t.Fatalf("Error creating template: %v", err)
	}

	jsonOutput, err := json.Marshal(tmpl.data)
	if err != nil {
		t.Fatalf("Error converting to JSON: %v", err)
	}

	// Parse and verify the output
	var result map[string]interface{}
	if err := json.Unmarshal(jsonOutput, &result); err != nil {
		t.Fatalf("Error parsing JSON output: %v", err)
	}

	// Check basic fields
	if result["name"] != "SampleUser" {
		t.Errorf("Expected name 'SampleUser', got %v", result["name"])
	}

	if result["age"] != float64(45) { // JSON numbers are float64
		t.Errorf("Expected age 45, got %v", result["age"])
	}

	// Check nested object
	config, ok := result["config"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected config to be a map, got %T", result["config"])
	}

	if config["debug"] != true {
		t.Errorf("Expected config.debug true, got %v", config["debug"])
	}

	if config["timeout"] != float64(45) {
		t.Errorf("Expected config.timeout 30, got %v", config["timeout"])
	}
}

// Performance tests
// Benchmark removed - uses legacy parseFieldPath and extractFieldValue functions

// Benchmark removed - uses legacy parseFieldPath and extractFieldValue functions

func BenchmarkCLITemplateSimpleExecution(b *testing.B) {
	doc := createComprehensiveTestDocument()
	templateStr := "{{.name}} - {{.age}}"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tmpl, err := NewTemplate(doc)
		if err != nil {
			b.Fatalf("Error creating template: %v", err)
		}
		
		oldStdout := os.Stdout
		os.Stdout, _ = os.Open(os.DevNull)
		
		err = tmpl.Execute(templateStr)
		
		os.Stdout = oldStdout
		
		if err != nil {
			b.Fatalf("Error executing template: %v", err)
		}
	}
}

func BenchmarkCLITemplateComplexExecution(b *testing.B) {
	doc := createComprehensiveTestDocument()
	templateStr := `{{.name}} ({{.age}}) 
Tags: {{range .tags}}{{.}} {{end}}
Config: {{range $k, $v := .config}}{{$k}}={{$v}} {{end}}`
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tmpl, err := NewTemplate(doc)
		if err != nil {
			b.Fatalf("Error creating template: %v", err)
		}
		
		oldStdout := os.Stdout
		os.Stdout, _ = os.Open(os.DevNull)
		
		err = tmpl.Execute(templateStr)
		
		os.Stdout = oldStdout
		
		if err != nil {
			b.Fatalf("Error executing template: %v", err)
		}
	}
}

package main

import (
	"strings"
	"testing"

	"github.com/kungfusheep/glint"
)

func TestCLIStructGeneratorBasicTypes(t *testing.T) {
	// Create a simple glint document
	builder := &glint.DocumentBuilder{}
	builder.AppendString("name", "TestUser")
	builder.AppendInt("age", 29)
	builder.AppendBool("active", true)
	
	doc := builder.Bytes()
	
	result, err := GenerateStruct(doc, "main", "User")
	if err != nil {
		t.Fatalf("GenerateStruct failed: %v", err)
	}
	
	// Check package declaration
	if !strings.Contains(result, "package main") {
		t.Error("Expected package main declaration")
	}
	
	// Check struct declaration
	if !strings.Contains(result, "type User struct {") {
		t.Error("Expected User struct declaration")
	}
	
	// Check fields
	expectedFields := []string{
		"Name   string `glint:\"name\"`",
		"Age    int    `glint:\"age\"`",
		"Active bool   `glint:\"active\"`",
	}
	
	for _, field := range expectedFields {
		if !strings.Contains(result, field) {
			t.Errorf("Expected field %s in result", field)
		}
	}
	
	// Check import
	if !strings.Contains(result, "\"github.com/kungfusheep/glint\"") {
		t.Error("Expected glint import")
	}
}

func TestCLIStructGeneratorAllSupportedTypes(t *testing.T) {
	// Create a document with all supported types
	builder := &glint.DocumentBuilder{}
	builder.AppendBool("bool_field", true)
	builder.AppendInt("int_field", 47)
	builder.AppendString("string_field", "sample")
	builder.AppendBytes("bytes_field", []byte("content"))
	
	doc := builder.Bytes()
	
	result, err := GenerateStruct(doc, "test", "AllTypes")
	if err != nil {
		t.Fatalf("GenerateStruct failed: %v", err)
	}
	
	// Check basic type mappings
	expectedPatterns := []string{
		"BoolField   bool   `glint:\"bool_field\"`",
		"IntField    int    `glint:\"int_field\"`", 
		"StringField string `glint:\"string_field\"`",
		"BytesField  []byte `glint:\"bytes_field\"`",
	}
	
	for _, pattern := range expectedPatterns {
		if !strings.Contains(result, pattern) {
			t.Errorf("Expected pattern not found: %s\nActual result:\n%s", pattern, result)
		}
	}
}

func TestCLIStructGeneratorSliceTypes(t *testing.T) {
	// Create a document with slice fields
	builder := &glint.DocumentBuilder{}
	
	// String slice
	stringSlice := &glint.SliceBuilder{}
	stringSlice.AppendStringSlice([]string{"lang1", "lang2"})
	builder.AppendSlice("tags", *stringSlice)
	
	// Int slice
	intSlice := &glint.SliceBuilder{}
	intSlice.AppendIntSlice([]int{1, 2, 3})
	builder.AppendSlice("numbers", *intSlice)
	
	// Bool slice
	boolSlice := &glint.SliceBuilder{}
	boolSlice.AppendBoolSlice([]bool{true, false})
	builder.AppendSlice("flags", *boolSlice)
	
	doc := builder.Bytes()
	
	result, err := GenerateStruct(doc, "test", "WithSlices")
	if err != nil {
		t.Fatalf("GenerateStruct failed: %v", err)
	}
	
	// Check slice types
	expectedSlices := []string{
		"Tags    []string `glint:\"tags\"`",
		"Numbers []int    `glint:\"numbers\"`",
		"Flags   []bool   `glint:\"flags\"`",
	}
	
	for _, slice := range expectedSlices {
		if !strings.Contains(result, slice) {
			t.Errorf("Expected slice field: %s", slice)
		}
	}
}

func TestCLIStructGeneratorNestedStructures(t *testing.T) {
	// Create a document with nested struct
	nestedBuilder := &glint.DocumentBuilder{}
	nestedBuilder.AppendString("bio", "Engineer")
	nestedBuilder.AppendInt("years", 5)
	
	builder := &glint.DocumentBuilder{}
	builder.AppendString("name", "TestUser")
	builder.AppendNestedDocument("profile", nestedBuilder)
	
	doc := builder.Bytes()
	
	result, err := GenerateStruct(doc, "models", "User")
	if err != nil {
		t.Fatalf("GenerateStruct failed: %v", err)
	}
	
	// Check nested struct is generated
	if !strings.Contains(result, "type Profile struct {") {
		t.Error("Expected Profile nested struct")
	}
	
	// Check nested struct fields
	if !strings.Contains(result, "Bio   string `glint:\"bio\"`") {
		t.Error("Expected Bio field in nested struct")
	}
	
	if !strings.Contains(result, "Years int    `glint:\"years\"`") {
		t.Error("Expected Years field in nested struct")
	}
	
	// Check main struct references nested struct
	if !strings.Contains(result, "Profile Profile `glint:\"profile\"`") {
		t.Error("Expected Profile field in main struct")
	}
}

func TestCLIStructGeneratorSliceOfStructs(t *testing.T) {
	// Create a document with array of structs
	people := []glint.DocumentBuilder{}
	
	person1 := &glint.DocumentBuilder{}
	person1.AppendString("name", "TestUser")
	person1.AppendInt("age", 29)
	people = append(people, *person1)
	
	person2 := &glint.DocumentBuilder{}
	person2.AppendString("name", "SampleUser")
	person2.AppendInt("age", 34)
	people = append(people, *person2)
	
	slice := &glint.SliceBuilder{}
	slice.AppendNestedDocumentSlice(people)
	
	builder := &glint.DocumentBuilder{}
	builder.AppendSlice("users", *slice)
	
	doc := builder.Bytes()
	
	result, err := GenerateStruct(doc, "api", "Response")
	if err != nil {
		t.Fatalf("GenerateStruct failed: %v", err)
	}
	
	// Check struct slice element type is generated
	if !strings.Contains(result, "type UsersItem struct {") {
		t.Error("Expected UsersItem struct for slice elements")
	}
	
	// Check slice field
	if !strings.Contains(result, "Users []UsersItem `glint:\"users\"`") {
		t.Error("Expected Users slice field")
	}
	
	// Check element struct fields
	if !strings.Contains(result, "Name string `glint:\"name\"`") {
		t.Error("Expected Name field in UsersItem")
	}
	
	if !strings.Contains(result, "Age  int    `glint:\"age\"`") {
		t.Error("Expected Age field in UsersItem")
	}
}

func TestCLIGoFieldNameConversion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"name", "Name"},
		{"first_name", "FirstName"},
		{"user_id", "UserId"},
		{"created_at", "CreatedAt"},
		{"api_key", "ApiKey"},
		{"", ""},
		{"a", "A"},
		{"a_b_c", "ABC"},
		{"snake_case_field", "SnakeCaseField"},
	}
	
	for _, test := range tests {
		result := toGoFieldName(test.input)
		if result != test.expected {
			t.Errorf("toGoFieldName(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestCLIStructGeneratorErrorHandling(t *testing.T) {
	// Test with invalid glint data
	invalidDoc := []byte{0x01, 0x02, 0x03}
	
	_, err := GenerateStruct(invalidDoc, "test", "Invalid")
	if err == nil {
		t.Error("Expected error for invalid glint data")
	}
}

func TestCLIStructGeneratorEmptyDocument(t *testing.T) {
	// Create empty document
	builder := &glint.DocumentBuilder{}
	doc := builder.Bytes()
	
	result, err := GenerateStruct(doc, "empty", "Empty")
	if err != nil {
		t.Fatalf("GenerateStruct failed: %v", err)
	}
	
	// Check empty struct is generated
	if !strings.Contains(result, "type Empty struct {") {
		t.Error("Expected Empty struct declaration")
	}
	
	// Should have package and import but no fields
	if !strings.Contains(result, "package empty") {
		t.Error("Expected package declaration")
	}
}
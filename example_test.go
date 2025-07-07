package glint_test

import (
	"fmt"
	"github.com/kungfusheep/glint"
)

func Example() {
	// Define your struct
	type Person struct {
		Name string `glint:"name"`
		Age  int    `glint:"age"`
		Tags []string `glint:"tags"`
	}

	// Create encoder and decoder once (thread-safe, reusable)
	encoder := glint.NewEncoder[Person]()
	decoder := glint.NewDecoder[Person]()

	// Create some data
	alice := Person{
		Name: "TestUser",
		Age:  32,
		Tags: []string{"engineer", "go", "serialization"},
	}

	// Encode to binary
	buffer := glint.NewBufferFromPool()
	defer buffer.ReturnToPool()
	
	encoder.Marshal(&alice, buffer)
	encoded := buffer.Bytes
	
	fmt.Printf("Encoded %d bytes\n", len(encoded))

	// Decode from binary
	var decoded Person
	err := decoder.Unmarshal(encoded, &decoded)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Decoded: %+v\n", decoded)
	// Output:
	// Encoded 60 bytes
	// Decoded: {Name:TestUser Age:32 Tags:[engineer go serialization]}
}

func ExampleDocumentBuilder() {
	// Build documents without structs
	doc := &glint.DocumentBuilder{}
	
	doc.AppendString("name", "SampleUser").
		AppendInt("age", 25).
		AppendBool("active", true)
	
	data := doc.Bytes()
	
	// Print human-readable format
	glint.Print(data)
	
	// Output shows tree structure (actual output may vary):
	// Glint Document
	// ├─ Schema
	// │  ├─ String: name
	// │  ├─ Int: age
	// │  └─ Bool: active
	// └─ Values
	//    ├─ name: SampleUser
	//    ├─ age: 25
	//    └─ active: true
}

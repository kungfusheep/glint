package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/kungfusheep/glint"
)

// Command interface for all pt commands
type Command interface {
	Name() string
	DefineFlags(fs *flag.FlagSet)
	Execute(args []string) error
}

// CommandRegistry holds all available commands
type CommandRegistry struct {
	commands map[string]Command
}

func NewCommandRegistry() *CommandRegistry {
	registry := &CommandRegistry{
		commands: make(map[string]Command),
	}

	// Register all commands
	registry.Register(&ConvertCmd{})
	registry.Register(&GenerateCmd{})
	registry.Register(&StatsCmd{})
	registry.Register(&SchemaCmd{})
	registry.Register(&CompatCmd{})
	registry.Register(&GetCmd{})
	registry.Register(&PrintfCmd{})
	registry.Register(&DebugCmd{})
	registry.Register(&InspectCmd{})

	return registry
}

func (r *CommandRegistry) Register(cmd Command) {
	r.commands[cmd.Name()] = cmd
}

func (r *CommandRegistry) Get(name string) (Command, bool) {
	cmd, exists := r.commands[name]
	return cmd, exists
}

func (r *CommandRegistry) ListCommands() []string {
	var names []string
	for name := range r.commands {
		if name != "" { // Skip empty names (like inspect which is default)
			names = append(names, name)
		}
	}
	return names
}

func (r *CommandRegistry) ExecuteCommand(cmdName string, args []string) error {
	cmd, exists := r.Get(cmdName)
	if !exists {
		return fmt.Errorf("unknown command: %s", cmdName)
	}

	// Create a new flag set for this command
	fs := flag.NewFlagSet(fmt.Sprintf("glint %s", cmdName), flag.ExitOnError)
	cmd.DefineFlags(fs)

	// Set custom usage function
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: glint %s [flags] [args...]\n", cmdName)
		if fs.NFlag() > 0 {
			fmt.Fprintf(os.Stderr, "\nFlags:\n")
			fs.PrintDefaults()
		}
		fmt.Fprintf(os.Stderr, "\nUse 'glint --help' for general help.\n")
	}

	// Parse command-specific flags
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Execute the command with remaining args
	return cmd.Execute(fs.Args())
}

// Main entry point
func main() {
	// Handle global help flag
	if len(os.Args) > 1 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		printGlobalHelp()
		return
	}

	// Create command registry
	registry := NewCommandRegistry()

	// If no arguments, default to inspect mode
	if len(os.Args) == 1 {
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			os.Exit(1)
		}
		
		if err := inspectDocument(input); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Get command name
	cmdName := os.Args[1]
	args := os.Args[2:]

	// Handle global help for specific commands
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		if cmd, exists := registry.Get(cmdName); exists {
			fs := flag.NewFlagSet(fmt.Sprintf("glint %s", cmdName), flag.ContinueOnError)
			cmd.DefineFlags(fs)
			fs.Usage()
			return
		}
	}

	// Execute command
	if err := registry.ExecuteCommand(cmdName, args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printGlobalHelp() {
	fmt.Print(`glint - Glint CLI Tool

A command-line utility for inspecting and manipulating glint binary documents.

Usage:
  glint [command] [flags] [args...]
  glint < document.glint                    # inspect document (default)

Conversion Commands:
  convert --from json                 # convert JSON to glint
  convert --to json                   # convert glint to JSON  
  convert --to csv                    # convert glint to CSV

Code Generation:
  generate go package.StructName      # generate Go struct from glint

Analysis Commands:
  stats                              # analyze document structure
  schema                             # show document schema only
  compat <old-file>                  # check schema compatibility

Data Extraction:
  get <field-path>                   # extract field (e.g., user.name, items[0])
  printf "<template>"                # format output with Go template
  printf -f <template-file>          # format using template file

Debugging:
  debug varint <bytes...>            # decode unsigned varint
  debug zigzag <bytes...>            # decode zigzag-encoded varint  
  debug ascii "<space-separated>"    # parse ASCII byte sequence

Examples:
  echo '{"name":"SampleUser"}' | glint convert --from json | glint
  glint get user.name < data.glint
  glint printf "Hello {{.name}}" < data.glint
  glint stats < data.glint
  glint compat old.glint < new.glint
  glint debug varint 172 2

Use 'glint <command> --help' for command-specific help.
`)
}

// ===============================
// COMMAND IMPLEMENTATIONS
// ===============================

// ConvertCmd handles format conversion
type ConvertCmd struct {
	from string
	to   string
}

func (c *ConvertCmd) Name() string { return "convert" }

func (c *ConvertCmd) DefineFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.from, "from", "", "Convert from format (json)")
	fs.StringVar(&c.to, "to", "", "Convert to format (json, csv)")
}

func (c *ConvertCmd) Execute(args []string) error {
	if c.from != "" && c.to != "" {
		return fmt.Errorf("cannot specify both --from and --to")
	}

	if c.from == "" && c.to == "" {
		return fmt.Errorf("must specify either --from or --to")
	}

	if c.from == "json" {
		return convertJSONToGlint()
	}

	if c.to == "json" {
		return convertGlintToJSON()
	}

	if c.to == "csv" {
		return convertGlintToCSV()
	}

	return fmt.Errorf("unsupported conversion: from=%s to=%s", c.from, c.to)
}

// DebugCmd handles debugging operations
type DebugCmd struct{}

func (d *DebugCmd) Name() string { return "debug" }

func (d *DebugCmd) DefineFlags(fs *flag.FlagSet) {
	// No flags - we use positional args
}

func (d *DebugCmd) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: pt debug <type> [args...]\n  pt debug varint 172 2\n  pt debug zigzag 3\n  pt debug ascii \"0 1 2 3\"")
	}

	switch args[0] {
	case "varint":
		return d.parseVarintsFromArgs(args[1:], false)
	case "zigzag":
		return d.parseVarintsFromArgs(args[1:], true)
	case "ascii":
		if len(args) != 2 {
			return fmt.Errorf("usage: pt debug ascii \"space separated bytes\"")
		}
		return d.parseASCIIBytes(args[1])
	default:
		return fmt.Errorf("unsupported debug type: %s", args[0])
	}
}

func (d *DebugCmd) parseVarintsFromArgs(args []string, zigzag bool) error {
	var input string
	if len(args) > 0 {
		input = strings.Join(args, " ")
	} else {
		// Read from stdin if no args provided
		bytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("error reading input: %v", err)
		}
		input = string(bytes)
	}

	return parseVarints(input, zigzag)
}

func (d *DebugCmd) parseASCIIBytes(asciiInput string) error {
	// Parse the input as space-separated byte values
	parts := strings.Fields(asciiInput)
	if len(parts) == 0 {
		return fmt.Errorf("no input provided")
	}

	var doc []byte
	for _, part := range parts {
		v, err := strconv.ParseUint(part, 10, 8)
		if err != nil {
			return fmt.Errorf("error parsing byte %q: %v", part, err)
		}
		doc = append(doc, byte(v))
	}

	// Print the glint document
	glint.Print(doc)
	return nil
}

// GetCmd handles field extraction
type GetCmd struct{}

func (g *GetCmd) Name() string { return "get" }

func (g *GetCmd) DefineFlags(fs *flag.FlagSet) {
	// No flags - we use positional args
}

func (g *GetCmd) Execute(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: pt get <field-path>\n  pt get user.name\n  pt get items[0].id")
	}

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("error reading input: %v", err)
	}

	return extractField(input, args[0])
}

// PrintfCmd handles template formatting
type PrintfCmd struct {
	templateFile string
}

func (p *PrintfCmd) Name() string { return "printf" }

func (p *PrintfCmd) DefineFlags(fs *flag.FlagSet) {
	fs.StringVar(&p.templateFile, "f", "", "Read template from file")
}

func (p *PrintfCmd) Execute(args []string) error {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("error reading input: %v", err)
	}

	if p.templateFile != "" {
		// Use template from file
		if len(args) > 0 {
			return fmt.Errorf("cannot specify both template string and template file")
		}
		return executeTemplate(input, "", p.templateFile)
	} else {
		// Use template from args
		if len(args) != 1 {
			return fmt.Errorf("usage: pt printf \"<template>\" or pt printf -f <template-file>")
		}
		return executeTemplate(input, args[0], "")
	}
}

// GenerateCmd handles code generation
type GenerateCmd struct {
	goStruct string
}

func (g *GenerateCmd) Name() string { return "generate" }

func (g *GenerateCmd) DefineFlags(fs *flag.FlagSet) {
	// No flags - we use positional args for generate commands
}

func (g *GenerateCmd) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: pt generate <type> [args...]\n  pt generate go package.StructName")
	}

	switch args[0] {
	case "go":
		if len(args) != 2 {
			return fmt.Errorf("usage: pt generate go package.StructName")
		}
		return g.generateGoStruct(args[1])
	default:
		return fmt.Errorf("unsupported generation type: %s", args[0])
	}
}

func (g *GenerateCmd) generateGoStruct(packageStruct string) error {
	// Parse the package.StructName format
	parts := strings.Split(packageStruct, ".")
	if len(parts) != 2 {
		return fmt.Errorf("invalid format: %s. Expected format: package.StructName", packageStruct)
	}

	packageName := parts[0]
	structName := parts[1]

	// Validate names
	if packageName == "" || structName == "" {
		return fmt.Errorf("package name and struct name cannot be empty")
	}

	// Read glint input from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("error reading glint input: %v", err)
	}

	// Generate the Go struct
	return generateGoStructFromDocument(input, packageName, structName)
}

// InspectCmd handles the default document inspection
type InspectCmd struct{}

func (i *InspectCmd) Name() string { return "inspect" }

func (i *InspectCmd) DefineFlags(fs *flag.FlagSet) {
	// No flags for basic inspection
}

func (i *InspectCmd) Execute(args []string) error {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("error reading input: %v", err)
	}

	// Print a human-readable representation of the glint document
	glint.Print(input)
	return nil
}

// SchemaCmd handles schema extraction
type SchemaCmd struct{}

func (s *SchemaCmd) Name() string { return "schema" }

func (s *SchemaCmd) DefineFlags(fs *flag.FlagSet) {
	// No flags for schema extraction
}

func (s *SchemaCmd) Execute(args []string) error {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("error reading input: %v", err)
	}

	return printSchema(input)
}

// StatsCmd handles document analysis
type StatsCmd struct{}

func (s *StatsCmd) Name() string { return "stats" }

func (s *StatsCmd) DefineFlags(fs *flag.FlagSet) {
	// No flags for stats
}

func (s *StatsCmd) Execute(args []string) error {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("error reading input: %v", err)
	}

	analyzeDocument(input)
	return nil
}

// CompatCmd handles schema compatibility checking
type CompatCmd struct{}

func (c *CompatCmd) Name() string { return "compat" }

func (c *CompatCmd) DefineFlags(fs *flag.FlagSet) {
	// No flags - we use positional args
}

func (c *CompatCmd) Execute(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: glint compat <old-file>\n  glint compat old.glint < new.glint")
	}

	// Read old schema from file
	oldData, err := os.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("error reading old file: %v", err)
	}

	// Read new schema from stdin
	newData, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("error reading new input: %v", err)
	}

	return checkSchemaCompatibility(oldData, newData)
}

// ===============================
// HELPER FUNCTIONS
// ===============================

// inspectDocument is the default behavior when no command is specified
func inspectDocument(input []byte) error {
	// Print a human-readable representation of the glint document
	glint.Print(input)
	return nil
}

func parseVarints(input string, zigzag bool) error {
	// Parse space-separated byte values
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return fmt.Errorf("no input provided")
	}

	var bytes []byte
	for _, part := range parts {
		v, err := strconv.ParseUint(part, 10, 8)
		if err != nil {
			return fmt.Errorf("error parsing byte %q: %v", part, err)
		}
		bytes = append(bytes, byte(v))
	}

	// Create reader and parse the varint
	reader := glint.NewReader(bytes)

	if zigzag {
		value := reader.ReadZigzagVarint()
		fmt.Printf("%d\n", value)
	} else {
		value := reader.ReadVarint()
		fmt.Printf("%d\n", value)
	}

	return nil
}

func printSchema(input []byte) error {
	if len(input) < 5 {
		return fmt.Errorf("document too short")
	}

	fmt.Printf("Glint Schema\n")
	// Use the glint Print function to show the schema
	glint.Print(input)
	return nil
}

// analyzeDocument performs basic structural analysis of a glint document
func analyzeDocument(doc []byte) {
	if len(doc) < 5 {
		fmt.Println("Error: Document too short for analysis")
		return
	}

	// Parse document structure
	reader := glint.NewReader(doc)
	printerDoc := glint.NewPrinterDocument(&reader)
	schema := glint.NewPrinterSchema(&printerDoc.Schema)

	// Calculate size breakdown
	totalSize := len(doc)
	headerSize := 5 // Fixed header size
	
	// Calculate schema size
	schemaReader := glint.NewReader(doc[5:])
	schemaLength := schemaReader.ReadVarint()
	varintLength := len(doc[5:]) - len(schemaReader.Remaining())
	schemaSize := varintLength + int(schemaLength)
	dataSize := totalSize - headerSize - schemaSize

	// Analyze schema structure
	fieldCount, maxDepth, wireTypes := analyzeSchemaStructure(&schema)

	// Print results
	fmt.Printf("Total size: %d bytes\n", totalSize)
	fmt.Printf("Header: %d bytes\n", headerSize)
	fmt.Printf("Schema: %d bytes (%.1f%%)\n", schemaSize, float64(schemaSize)/float64(totalSize)*100)
	fmt.Printf("Data: %d bytes (%.1f%%)\n", dataSize, float64(dataSize)/float64(totalSize)*100)
	fmt.Printf("Fields: %d\n", fieldCount)
	fmt.Printf("Max depth: %d\n", maxDepth)
	fmt.Printf("Wire types: %s\n", formatWireTypes(wireTypes))
}

// analyzeSchemaStructure examines schema structure and returns basic metrics
func analyzeSchemaStructure(schema *glint.PrinterSchema) (int, int, map[string]int) {
	wireTypes := make(map[string]int)
	fieldCount := 0
	maxDepth := 1

	// Simple field counting for now - this could be enhanced
	for _, field := range schema.Fields {
		fieldCount++
		wireTypes[field.TypeID.String()]++
	}

	return fieldCount, maxDepth, wireTypes
}

func formatWireTypes(wireTypes map[string]int) string {
	if len(wireTypes) == 0 {
		return "none"
	}

	var parts []string
	for wireType, count := range wireTypes {
		parts = append(parts, fmt.Sprintf("%s(%d)", wireType, count))
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

// ===============================
// CONVERSION FUNCTIONS
// ===============================

// convertJSONToGlint reads JSON from stdin and converts it to glint format
func convertJSONToGlint() error {
	// Read JSON input from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("error reading JSON input: %v", err)
	}

	// Parse JSON into generic interface{}
	var data interface{}
	if err := json.Unmarshal(input, &data); err != nil {
		return fmt.Errorf("error parsing JSON: %v", err)
	}

	// Convert to glint format and write to stdout
	glintData, err := jsonToGlint(data)
	if err != nil {
		return fmt.Errorf("error converting to glint: %v", err)
	}

	// Write binary glint data to stdout
	os.Stdout.Write(glintData)
	return nil
}

// convertGlintToJSON reads glint data from stdin and converts it to JSON format
func convertGlintToJSON() error {
	// Read glint input from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("error reading glint input: %v", err)
	}

	// Create template processor to convert glint to map[string]interface{}
	tmpl, err := NewTemplate(input)
	if err != nil {
		return fmt.Errorf("error parsing glint document: %v", err)
	}

	// Convert the template data to JSON
	jsonData, err := json.MarshalIndent(tmpl.data, "", "  ")
	if err != nil {
		return fmt.Errorf("error converting to JSON: %v", err)
	}

	// Write JSON to stdout
	os.Stdout.Write(jsonData)
	fmt.Println() // Add newline for cleaner output
	return nil
}

// convertGlintToCSV reads glint data from stdin and converts it to CSV format
func convertGlintToCSV() error {
	// Read glint input from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("error reading glint input: %v", err)
	}

	// Create template processor to convert glint to map[string]interface{}
	tmpl, err := NewTemplate(input)
	if err != nil {
		return fmt.Errorf("error parsing glint document: %v", err)
	}

	// Convert the template data to CSV
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	if err := convertDataToCSV(writer, tmpl.data); err != nil {
		return fmt.Errorf("error converting to CSV: %v", err)
	}

	return nil
}

// convertDataToCSV converts glint data to CSV format with flattening for nested structures
func convertDataToCSV(writer *csv.Writer, data map[string]interface{}) error {
	// Check if this is array-like data (single field containing a slice)
	// Common patterns: {"items": [...]} or {"value": [...]}
	if len(data) == 1 {
		for _, value := range data {
			if slice, ok := value.([]interface{}); ok {
				return convertArrayToCSV(writer, slice)
			}
		}
	}

	// For object data, flatten it and write as rows
	headers, rows := flattenData(data)
	
	// Write headers
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("error writing CSV headers: %v", err)
	}

	// Write data rows
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("error writing CSV row: %v", err)
		}
	}

	return nil
}

// convertArrayToCSV converts an array of items to CSV format
func convertArrayToCSV(writer *csv.Writer, data []interface{}) error {
	if len(data) == 0 {
		return nil
	}

	// Check if array contains objects (maps)
	if len(data) > 0 {
		if firstMap, ok := data[0].(map[string]interface{}); ok {
			// Array of objects - extract headers from first object
			var headers []string
			for key := range firstMap {
				headers = append(headers, key)
			}

			// Write headers
			if err := writer.Write(headers); err != nil {
				return fmt.Errorf("error writing CSV headers: %v", err)
			}

			// Write each object as a row
			for _, item := range data {
				if itemMap, ok := item.(map[string]interface{}); ok {
					var row []string
					for _, header := range headers {
						if value, exists := itemMap[header]; exists {
							row = append(row, formatValue(value))
						} else {
							row = append(row, "")
						}
					}
					if err := writer.Write(row); err != nil {
						return fmt.Errorf("error writing CSV row: %v", err)
					}
				}
			}
			return nil
		}
	}

	// Array of primitives - write as single column
	writer.Write([]string{"value"}) // Header
	for _, item := range data {
		if err := writer.Write([]string{formatValue(item)}); err != nil {
			return fmt.Errorf("error writing CSV row: %v", err)
		}
	}

	return nil
}

// flattenData flattens nested data structures for CSV output
func flattenData(data map[string]interface{}) ([]string, [][]string) {
	flattened := make(map[string]interface{})
	flattenMap("", data, flattened)

	// Extract headers and create single row
	var headers []string
	var values []string

	for key, value := range flattened {
		headers = append(headers, key)
		values = append(values, formatValue(value))
	}

	return headers, [][]string{values}
}

// flattenMap recursively flattens nested maps with dot notation
func flattenMap(prefix string, data map[string]interface{}, result map[string]interface{}) {
	for key, value := range data {
		newKey := key
		if prefix != "" {
			newKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]interface{}:
			// Recursively flatten nested objects
			flattenMap(newKey, v, result)
		case []interface{}:
			// Handle arrays by flattening each element or summarizing
			if len(v) == 0 {
				result[newKey] = "[]"
			} else if len(v) <= 5 {
				// Small arrays - enumerate elements
				for i, item := range v {
					result[fmt.Sprintf("%s[%d]", newKey, i)] = item
				}
			} else {
				// Large arrays - summarize
				result[newKey] = fmt.Sprintf("[%d items]", len(v))
			}
		default:
			result[newKey] = value
		}
	}
}

// formatValue converts any value to string for CSV output
func formatValue(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case bool:
		return strconv.FormatBool(v)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%g", v)
	case []interface{}:
		if len(v) <= 3 {
			var items []string
			for _, item := range v {
				items = append(items, formatValue(item))
			}
			return "[" + strings.Join(items, ", ") + "]"
		}
		return fmt.Sprintf("[%d items]", len(v))
	case map[string]interface{}:
		return fmt.Sprintf("{%d fields}", len(v))
	default:
		return fmt.Sprintf("%v", v)
	}
}

// jsonToGlint converts a parsed JSON value to glint binary format
func jsonToGlint(data interface{}) ([]byte, error) {
	// Process the data based on its type
	switch v := data.(type) {
	case map[string]interface{}:
		// JSON object - process as glint document with fields
		return buildGlintFromObject(v)
	case []interface{}:
		// JSON array - create a document with a single array field
		return buildGlintFromArray(v)
	default:
		// Single value - create a document with a single field
		return buildGlintFromValue(v)
	}
}

// buildGlintFromObject converts a JSON object to glint format
func buildGlintFromObject(obj map[string]interface{}) ([]byte, error) {
	builder := &glint.DocumentBuilder{}

	// Add fields to the builder based on the JSON object
	for key, value := range obj {
		if err := addFieldToBuilder(builder, key, value); err != nil {
			return nil, fmt.Errorf("error adding field '%s': %v", key, err)
		}
	}

	return builder.Bytes(), nil
}

// buildGlintFromArray converts a JSON array to glint format
func buildGlintFromArray(arr []interface{}) ([]byte, error) {
	builder := &glint.DocumentBuilder{}

	// Create a single field called "items" containing the array
	if err := addFieldToBuilder(builder, "items", arr); err != nil {
		return nil, fmt.Errorf("error adding array field: %v", err)
	}

	return builder.Bytes(), nil
}

// buildGlintFromValue converts a single JSON value to glint format
func buildGlintFromValue(value interface{}) ([]byte, error) {
	builder := &glint.DocumentBuilder{}

	// Create a single field called "value" containing the data
	if err := addFieldToBuilder(builder, "value", value); err != nil {
		return nil, fmt.Errorf("error adding value field: %v", err)
	}

	return builder.Bytes(), nil
}

// addFieldToBuilder adds a field with the given name and value to the glint builder
func addFieldToBuilder(builder *glint.DocumentBuilder, name string, value interface{}) error {
	switch v := value.(type) {
	case nil:
		// Handle null values as strings for now (could be enhanced to use pointers)
		builder.AppendString(name, "")

	case bool:
		builder.AppendBool(name, v)

	case float64:
		// JSON numbers are always float64, but we can try to detect integers
		if v == float64(int64(v)) {
			// It's an integer value
			builder.AppendInt(name, int(v))
		} else {
			// It's a floating point value
			builder.AppendFloat64(name, v)
		}

	case string:
		builder.AppendString(name, v)

	case []interface{}:
		// Handle arrays using proper glint array types
		if err := addArrayToBuilder(builder, name, v); err != nil {
			return fmt.Errorf("error adding array field '%s': %v", name, err)
		}

	case map[string]interface{}:
		// Handle nested objects by creating a nested document
		nestedDoc := &glint.DocumentBuilder{}
		for nestedKey, nestedValue := range v {
			if err := addFieldToBuilder(nestedDoc, nestedKey, nestedValue); err != nil {
				return fmt.Errorf("error adding nested field '%s.%s': %v", name, nestedKey, err)
			}
		}
		builder.AppendNestedDocument(name, nestedDoc)

	default:
		return fmt.Errorf("unsupported JSON value type %T for field '%s'", value, name)
	}

	return nil
}

// addArrayToBuilder adds a JSON array to the glint builder with proper type detection
func addArrayToBuilder(builder *glint.DocumentBuilder, name string, arr []interface{}) error {
	if len(arr) == 0 {
		// Empty array - create empty string slice
		slice := &glint.SliceBuilder{}
		slice.AppendStringSlice([]string{})
		builder.AppendSlice(name, *slice)
		return nil
	}

	// Determine the array type from the first element
	firstElem := arr[0]

	switch firstElem.(type) {
	case string:
		// String array - collect all strings
		var strings []string
		for _, elem := range arr {
			if str, ok := elem.(string); ok {
				strings = append(strings, str)
			} else {
				// Mixed types - convert everything to strings
				strings = append(strings, fmt.Sprintf("%v", elem))
			}
		}
		slice := &glint.SliceBuilder{}
		slice.AppendStringSlice(strings)
		builder.AppendSlice(name, *slice)

	case bool:
		// Boolean array - collect all booleans, fallback to strings for mixed types
		var bools []bool
		allBools := true
		for _, elem := range arr {
			if b, ok := elem.(bool); ok {
				bools = append(bools, b)
			} else {
				allBools = false
				break
			}
		}

		slice := &glint.SliceBuilder{}
		if allBools {
			slice.AppendBoolSlice(bools)
		} else {
			// Mixed types - convert everything to strings
			var strings []string
			for _, elem := range arr {
				strings = append(strings, fmt.Sprintf("%v", elem))
			}
			slice.AppendStringSlice(strings)
		}
		builder.AppendSlice(name, *slice)

	case float64:
		// Numeric array - check if all are integers or if we need floats
		allNumbers := true
		allInts := true
		for _, elem := range arr {
			if num, ok := elem.(float64); ok {
				if num != float64(int64(num)) {
					allInts = false
				}
			} else {
				// Not all numbers - fall back to string array
				allNumbers = false
				break
			}
		}

		slice := &glint.SliceBuilder{}
		if !allNumbers {
			// Mixed types including non-numbers - convert to strings
			var strings []string
			for _, elem := range arr {
				strings = append(strings, fmt.Sprintf("%v", elem))
			}
			slice.AppendStringSlice(strings)
		} else if allInts {
			// Integer array - collect all integers
			var ints []int
			for _, elem := range arr {
				if num, ok := elem.(float64); ok {
					ints = append(ints, int(num))
				}
			}
			slice.AppendIntSlice(ints)
		} else {
			// Float array - collect all floats
			var floats []float64
			for _, elem := range arr {
				if num, ok := elem.(float64); ok {
					floats = append(floats, num)
				}
			}
			slice.AppendFloat64Slice(floats)
		}
		builder.AppendSlice(name, *slice)

	case nil:
		// Array with null values - treat as string array with empty strings
		var strings []string
		for _, elem := range arr {
			if elem == nil {
				strings = append(strings, "")
			} else {
				strings = append(strings, fmt.Sprintf("%v", elem))
			}
		}
		slice := &glint.SliceBuilder{}
		slice.AppendStringSlice(strings)
		builder.AppendSlice(name, *slice)

	case map[string]interface{}:
		// Array of objects - use proper nested document slice
		var documents []glint.DocumentBuilder
		for _, elem := range arr {
			if objMap, ok := elem.(map[string]interface{}); ok {
				nestedDoc := &glint.DocumentBuilder{}
				for nestedKey, nestedValue := range objMap {
					if err := addFieldToBuilder(nestedDoc, nestedKey, nestedValue); err != nil {
						return fmt.Errorf("error adding nested field '%s': %v", nestedKey, err)
					}
				}
				documents = append(documents, *nestedDoc)
			}
		}

		if len(documents) > 0 {
			slice := &glint.SliceBuilder{}
			slice.AppendNestedDocumentSlice(documents)
			builder.AppendSlice(name, *slice)
		}

	default:
		// Mixed or unsupported types - convert everything to strings
		var strings []string
		for _, elem := range arr {
			strings = append(strings, fmt.Sprintf("%v", elem))
		}
		slice := &glint.SliceBuilder{}
		slice.AppendStringSlice(strings)
		builder.AppendSlice(name, *slice)
	}

	return nil
}

// ===============================
// UTILITY FUNCTION WRAPPERS
// ===============================

// These functions call implementations in separate utility files
// per the footprint rules - keeping complex utilities separate

func generateGoStructFromDocument(input []byte, packageName, structName string) error {
	// Call the GenerateStruct function from structgenerator.go
	result, err := GenerateStruct(input, packageName, structName)
	if err != nil {
		return err
	}
	
	fmt.Print(result)
	return nil
}

func extractField(doc []byte, fieldPath string) error {
	// Convert field path to template syntax
	templateStr, err := fieldPathToTemplate(fieldPath)
	if err != nil {
		return fmt.Errorf("invalid field path: %s", fieldPath)
	}

	// Use template system to extract the field
	tmpl, err := NewTemplate(doc)
	if err != nil {
		return fmt.Errorf("error parsing document: %v", err)
	}

	err = tmpl.Execute(templateStr)
	if err != nil {
		return fmt.Errorf("error extracting field: %v", err)
	}

	return nil
}

func executeTemplate(doc []byte, templateStr, templateFile string) error {
	tmpl, err := NewTemplate(doc)
	if err != nil {
		return fmt.Errorf("error creating template: %v", err)
	}

	if templateFile != "" {
		err = tmpl.ExecuteFile(templateFile)
	} else {
		err = tmpl.Execute(templateStr)
	}

	if err != nil {
		return fmt.Errorf("error executing template: %v", err)
	}

	return nil
}

func checkSchemaCompatibility(oldData, newData []byte) error {
	// Parse old document
	if len(oldData) < 5 {
		return fmt.Errorf("old document too short")
	}
	oldReader := glint.NewReader(oldData)
	oldDoc := glint.NewPrinterDocument(&oldReader)
	oldSchema := glint.NewPrinterSchema(&oldDoc.Schema)

	// Parse new document  
	if len(newData) < 5 {
		return fmt.Errorf("new document too short")
	}
	newReader := glint.NewReader(newData)
	newDoc := glint.NewPrinterDocument(&newReader)
	newSchema := glint.NewPrinterSchema(&newDoc.Schema)

	// Compare schemas
	result := compareSchemas(&oldSchema, &newSchema)

	// Print results
	fmt.Printf("Schema compatibility: old → new\n")
	fmt.Println()

	if result.BackwardCompatible && result.ForwardCompatible {
		fmt.Println("✓ Fully compatible")
	} else if result.BackwardCompatible {
		fmt.Println("✓ Backward compatible")
		fmt.Println("✗ Forward compatible: NO")
	} else if result.ForwardCompatible {
		fmt.Println("✗ Backward compatible: NO")
		fmt.Println("✓ Forward compatible")
	} else {
		fmt.Println("✗ Backward compatible: NO")
		fmt.Println("✗ Forward compatible: NO")
	}

	if len(result.Changes) > 0 {
		fmt.Println("\nChanges detected:")
		for _, change := range result.Changes {
			fmt.Printf("%s %s\n", change.Icon(), change.Description())
		}
	} else {
		fmt.Println("\nNo changes detected")
	}

	return nil
}

// CompatibilityResult represents the result of a schema compatibility check
type CompatibilityResult struct {
	BackwardCompatible bool
	ForwardCompatible  bool
	Changes           []SchemaChange
}

// SchemaChange represents a single change between schemas
type SchemaChange struct {
	Type        string // "added", "removed", "type_changed"
	FieldPath   string
	OldType     string
	NewType     string
	Breaking    bool
}

func (c SchemaChange) Icon() string {
	switch c.Type {
	case "added":
		return "+"
	case "removed":
		return "-"
	case "type_changed":
		return "~"
	default:
		return "?"
	}
}

func (c SchemaChange) Description() string {
	switch c.Type {
	case "added":
		return fmt.Sprintf("Added field '%s' (%s)", c.FieldPath, c.NewType)
	case "removed":
		return fmt.Sprintf("Removed field '%s' (%s)", c.FieldPath, c.OldType)
	case "type_changed":
		return fmt.Sprintf("Changed field '%s' from %s to %s", c.FieldPath, c.OldType, c.NewType)
	default:
		return fmt.Sprintf("Unknown change: %s", c.FieldPath)
	}
}

// compareSchemas compares two schemas and returns compatibility information
func compareSchemas(oldSchema, newSchema *glint.PrinterSchema) CompatibilityResult {
	result := CompatibilityResult{
		BackwardCompatible: true,
		ForwardCompatible:  true,
		Changes:           []SchemaChange{},
	}

	// Build field maps for easy comparison
	oldFields := buildFieldMap(oldSchema, "")
	newFields := buildFieldMap(newSchema, "")

	// Check for type changes and removals
	for fieldPath, oldField := range oldFields {
		if newField, exists := newFields[fieldPath]; exists {
			// Field exists in both - check for type changes
			if !areTypesCompatible(oldField, newField) {
				result.Changes = append(result.Changes, SchemaChange{
					Type:      "type_changed",
					FieldPath: fieldPath,
					OldType:   getFieldTypeName(oldField),
					NewType:   getFieldTypeName(newField),
					Breaking:  true,
				})
				result.BackwardCompatible = false
				result.ForwardCompatible = false
			}
		} else {
			// Field was removed
			result.Changes = append(result.Changes, SchemaChange{
				Type:      "removed",
				FieldPath: fieldPath,
				OldType:   getFieldTypeName(oldField),
				Breaking:  true,
			})
			result.ForwardCompatible = false
		}
	}

	// Check for additions
	for fieldPath, newField := range newFields {
		if _, exists := oldFields[fieldPath]; !exists {
			// Field was added
			result.Changes = append(result.Changes, SchemaChange{
				Type:      "added",
				FieldPath: fieldPath,
				NewType:   getFieldTypeName(newField),
				Breaking:  false, // Field additions don't break backward compatibility
			})
			result.BackwardCompatible = false
		}
	}

	return result
}

// buildFieldMap creates a flat map of field paths to field definitions
func buildFieldMap(schema *glint.PrinterSchema, pathPrefix string) map[string]glint.PrinterSchemaField {
	fields := make(map[string]glint.PrinterSchemaField)
	
	for _, field := range schema.Fields {
		fieldPath := field.Name
		if pathPrefix != "" {
			fieldPath = pathPrefix + "." + field.Name
		}
		fields[fieldPath] = field
	}

	return fields
}

// areTypesCompatible checks if two field types are compatible
func areTypesCompatible(oldField, newField glint.PrinterSchemaField) bool {
	// Compare the base wire types and flags
	return oldField.TypeID == newField.TypeID
}

// getFieldTypeName returns a human-readable type name for a field
func getFieldTypeName(field glint.PrinterSchemaField) string {
	return field.TypeID.String()
}
package main

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/kungfusheep/glint"
)

// GenerateStruct generates Go struct code from a glint document
func GenerateStruct(doc []byte, packageName, structName string) (result string, err error) {
	// Parse the glint document with error handling
	if len(doc) < 5 {
		return "", fmt.Errorf("invalid glint document: too short")
	}
	
	// Use defer to catch panics from invalid documents
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("invalid glint document: %v", r)
		}
	}()
	
	reader := glint.NewReader(doc)
	printerDoc := glint.NewPrinterDocument(&reader)
	schema := glint.NewPrinterSchema(&printerDoc.Schema)

	// Generate the struct
	generator := &structGenerator{
		packageName: packageName,
		structs:     make(map[string]*structInfo),
		imports:     make(map[string]bool),
	}

	// Generate main struct
	structDef, err := generator.generateStructFromSchema(structName, &schema)
	if err != nil {
		return "", fmt.Errorf("failed to generate struct: %v", err)
	}

	// Build the complete Go file
	return generator.buildGoFile(structName, structDef), nil
}

// structGenerator handles the generation of Go structs from glint schemas
type structGenerator struct {
	packageName string
	structs     map[string]*structInfo
	imports     map[string]bool
}

// structInfo represents a generated struct
type structInfo struct {
	name   string
	fields []fieldInfo
}

// fieldInfo represents a field in a generated struct
type fieldInfo struct {
	name     string
	goType   string
	tag      string
	comment  string
}

// generateStructFromSchema converts a PrinterSchema to a Go struct
func (g *structGenerator) generateStructFromSchema(structName string, schema *glint.PrinterSchema) (*structInfo, error) {
	// Check if we've already generated this struct
	if existing, exists := g.structs[structName]; exists {
		return existing, nil
	}

	structDef := &structInfo{
		name:   structName,
		fields: make([]fieldInfo, 0, len(schema.Fields)),
	}

	// Process each field in the schema
	for _, field := range schema.Fields {
		fieldDef, err := g.generateField(field)
		if err != nil {
			return nil, fmt.Errorf("failed to generate field %s: %v", field.Name, err)
		}
		structDef.fields = append(structDef.fields, fieldDef)
	}

	// Store the struct definition
	g.structs[structName] = structDef
	return structDef, nil
}

// generateField converts a PrinterSchemaField to a Go struct field
func (g *structGenerator) generateField(field glint.PrinterSchemaField) (fieldInfo, error) {
	// Convert field name to Go naming convention
	goFieldName := toGoFieldName(field.Name)
	
	// Generate the Go type
	goType, err := g.wireTypeToGoType(field)
	if err != nil {
		return fieldInfo{}, fmt.Errorf("failed to convert type for field %s: %v", field.Name, err)
	}

	// Generate glint tag
	tag := fmt.Sprintf(`glint:"%s"`, field.Name)

	return fieldInfo{
		name:   goFieldName,
		goType: goType,
		tag:    tag,
	}, nil
}

// wireTypeToGoType converts a glint wire type to a Go type string
func (g *structGenerator) wireTypeToGoType(field glint.PrinterSchemaField) (string, error) {
	baseType := field.TypeID & glint.WireTypeMask
	
	// Handle pointer types
	var prefix string
	if field.IsPointer {
		prefix = "*"
	}

	// Handle slice types
	if field.IsSlice {
		elementType, err := g.getElementType(field)
		if err != nil {
			return "", err
		}
		return prefix + "[]" + elementType, nil
	}

	// Handle base types
	goType, err := g.getBaseGoType(baseType, field)
	if err != nil {
		return "", err
	}

	return prefix + goType, nil
}

// getElementType gets the Go type for slice elements
func (g *structGenerator) getElementType(field glint.PrinterSchemaField) (string, error) {
	baseType := field.TypeID & glint.WireTypeMask
	
	if baseType == glint.WireStruct {
		// Array of structs - generate nested struct type
		if field.NestedSchema == nil {
			return "", fmt.Errorf("struct slice missing nested schema")
		}
		
		// Generate a nested struct name
		nestedName := toGoFieldName(field.Name) + "Item"
		_, err := g.generateStructFromSchema(nestedName, field.NestedSchema)
		if err != nil {
			return "", err
		}
		return nestedName, nil
	}
	
	// Primitive slice element
	return g.getBaseGoType(baseType, field)
}

// getBaseGoType converts a base wire type to a Go type
func (g *structGenerator) getBaseGoType(wireType glint.WireType, field glint.PrinterSchemaField) (string, error) {
	switch wireType {
	case glint.WireBool:
		return "bool", nil
	case glint.WireInt:
		return "int", nil
	case glint.WireInt8:
		return "int8", nil
	case glint.WireInt16:
		return "int16", nil
	case glint.WireInt32:
		return "int32", nil
	case glint.WireInt64:
		return "int64", nil
	case glint.WireUint:
		return "uint", nil
	case glint.WireUint8:
		return "uint8", nil
	case glint.WireUint16:
		return "uint16", nil
	case glint.WireUint32:
		return "uint32", nil
	case glint.WireUint64:
		return "uint64", nil
	case glint.WireFloat32:
		return "float32", nil
	case glint.WireFloat64:
		return "float64", nil
	case glint.WireString:
		return "string", nil
	case glint.WireBytes:
		return "[]byte", nil
	case glint.WireTime:
		g.imports["time"] = true
		return "time.Time", nil
	case glint.WireStruct:
		// Nested struct
		if field.NestedSchema == nil {
			return "", fmt.Errorf("struct field missing nested schema")
		}
		
		// Generate nested struct name
		nestedName := toGoFieldName(field.Name)
		_, err := g.generateStructFromSchema(nestedName, field.NestedSchema)
		if err != nil {
			return "", err
		}
		return nestedName, nil
	case glint.WireMap:
		// Map type
		if len(field.MapType) != 2 {
			return "", fmt.Errorf("invalid map type")
		}
		
		keyType, err := g.wireTypeToGoTypeSimple(field.MapType[0])
		if err != nil {
			return "", fmt.Errorf("invalid map key type: %v", err)
		}
		
		valueType, err := g.wireTypeToGoTypeSimple(field.MapType[1])
		if err != nil {
			return "", fmt.Errorf("invalid map value type: %v", err)
		}
		
		return fmt.Sprintf("map[%s]%s", keyType, valueType), nil
	default:
		return "", fmt.Errorf("unsupported wire type: %v", wireType)
	}
}

// wireTypeToGoTypeSimple converts simple wire types to Go types (for map keys/values)
func (g *structGenerator) wireTypeToGoTypeSimple(wireType glint.WireType) (string, error) {
	switch wireType {
	case glint.WireBool:
		return "bool", nil
	case glint.WireInt:
		return "int", nil
	case glint.WireInt8:
		return "int8", nil
	case glint.WireInt16:
		return "int16", nil
	case glint.WireInt32:
		return "int32", nil
	case glint.WireInt64:
		return "int64", nil
	case glint.WireUint:
		return "uint", nil
	case glint.WireUint8:
		return "uint8", nil
	case glint.WireUint16:
		return "uint16", nil
	case glint.WireUint32:
		return "uint32", nil
	case glint.WireUint64:
		return "uint64", nil
	case glint.WireFloat32:
		return "float32", nil
	case glint.WireFloat64:
		return "float64", nil
	case glint.WireString:
		return "string", nil
	case glint.WireBytes:
		return "[]byte", nil
	case glint.WireTime:
		g.imports["time"] = true
		return "time.Time", nil
	default:
		return "", fmt.Errorf("unsupported simple wire type: %v", wireType)
	}
}

// buildGoFile builds the complete Go file with package, imports, and structs
func (g *structGenerator) buildGoFile(mainStructName string, mainStruct *structInfo) string {
	var b strings.Builder
	
	// Package declaration
	b.WriteString(fmt.Sprintf("package %s\n\n", g.packageName))
	
	// Imports
	if len(g.imports) > 0 {
		b.WriteString("import (\n")
		
		// Sort imports for consistent output
		imports := make([]string, 0, len(g.imports))
		for imp := range g.imports {
			imports = append(imports, imp)
		}
		sort.Strings(imports)
		
		for _, imp := range imports {
			b.WriteString(fmt.Sprintf("\t\"%s\"\n", imp))
		}
		
		// Always include glint for tags
		b.WriteString("\t\"github.com/kungfusheep/glint\"\n")
		b.WriteString(")\n\n")
	} else {
		b.WriteString("import \"github.com/kungfusheep/glint\"\n\n")
	}
	
	// Generate structs in dependency order (main struct last)
	generated := make(map[string]bool)
	
	// Generate nested structs first
	for name, structDef := range g.structs {
		if name != mainStructName && !generated[name] {
			g.writeStruct(&b, structDef)
			generated[name] = true
		}
	}
	
	// Generate main struct last
	if !generated[mainStructName] {
		g.writeStruct(&b, mainStruct)
	}
	
	return b.String()
}

// writeStruct writes a single struct definition
func (g *structGenerator) writeStruct(b *strings.Builder, structDef *structInfo) {
	b.WriteString(fmt.Sprintf("type %s struct {\n", structDef.name))
	
	// Find the maximum field name length for alignment
	maxNameLen := 0
	maxTypeLen := 0
	for _, field := range structDef.fields {
		if len(field.name) > maxNameLen {
			maxNameLen = len(field.name)
		}
		if len(field.goType) > maxTypeLen {
			maxTypeLen = len(field.goType)
		}
	}
	
	// Write fields with aligned formatting
	for _, field := range structDef.fields {
		b.WriteString(fmt.Sprintf("\t%-*s %-*s `%s`", 
			maxNameLen, field.name,
			maxTypeLen, field.goType,
			field.tag))
		
		if field.comment != "" {
			b.WriteString(" // " + field.comment)
		}
		b.WriteString("\n")
	}
	
	b.WriteString("}\n\n")
}

// toGoFieldName converts a glint field name to Go field naming convention
func toGoFieldName(fieldName string) string {
	if fieldName == "" {
		return ""
	}
	
	// Split on underscores and convert to PascalCase
	parts := strings.Split(fieldName, "_")
	var result strings.Builder
	
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		
		// Capitalize first letter and add the rest
		runes := []rune(part)
		result.WriteRune(unicode.ToUpper(runes[0]))
		if len(runes) > 1 {
			result.WriteString(strings.ToLower(string(runes[1:])))
		}
	}
	
	return result.String()
}
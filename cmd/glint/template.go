package main

import (
	"fmt"
	"os"
	"text/template"

	"github.com/kungfusheep/glint"
)

// Template handles glint document templating with full map support
type Template struct {
	data     map[string]interface{}
	document []byte
	schema   *glint.PrinterSchema // Store schema for complex navigation
}

// NewTemplate creates a template processor for a glint document
func NewTemplate(document []byte) (*Template, error) {
	t := &Template{
		document: document,
		data:     make(map[string]interface{}),
	}

	err := t.documentToMap()
	if err != nil {
		return nil, fmt.Errorf("failed to parse document: %v", err)
	}

	return t, nil
}

// Execute runs the template with the document data
func (t *Template) Execute(templateStr string) error {
	tmpl, err := template.New("glint").Parse(templateStr)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	err = tmpl.Execute(os.Stdout, t.data)
	if err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	return nil
}

// ExecuteFile runs a template from file
func (t *Template) ExecuteFile(path string) error {
	templateBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read template file: %v", err)
	}

	return t.Execute(string(templateBytes))
}

// documentToMap converts glint document to template-friendly map
func (t *Template) documentToMap() error {
	// Create reader to parse the document
	r := glint.NewReader(t.document)

	// Parse document structure
	doc := glint.NewPrinterDocument(&r)

	// Parse schema
	schema := glint.NewPrinterSchema(&doc.Schema)
	t.schema = &schema // Store schema reference for complex navigation

	// Convert each field to the map
	for _, field := range schema.Fields {
		value, err := t.fieldToInterface(&doc.Body, &field)
		if err != nil {
			return fmt.Errorf("failed to convert field '%s': %v", field.Name, err)
		}
		t.data[field.Name] = value
	}

	return nil
}

// fieldToInterface converts a field value to appropriate Go type
func (t *Template) fieldToInterface(reader *glint.Reader, field *glint.PrinterSchemaField) (interface{}, error) {
	return t.fieldValueByType(reader, field.TypeID, field)
}

// fieldValueByType unified field reading method with full type support
func (t *Template) fieldValueByType(reader *glint.Reader, wireType glint.WireType, field *glint.PrinterSchemaField) (interface{}, error) {
	// Handle pointer wrapper
	if wireType&glint.WirePtrFlag > 0 {
		if reader.ReadByte() == 0 {
			return nil, nil
		}
		wireType ^= glint.WirePtrFlag
	}

	// Handle slice wrapper
	if wireType&glint.WireSliceFlag > 0 {
		return t.sliceValueByType(reader, wireType, field)
	}

	switch wireType & glint.WireTypeMask {
	case glint.WireStruct:
		return t.structToInterface(reader, field)
	case glint.WireMap:
		return t.mapToInterface(reader, field)
	default:
		return t.simpleFieldToInterface(reader, wireType)
	}
}

// simpleFieldToInterface converts a simple field to Go interface{}
func (t *Template) simpleFieldToInterface(reader *glint.Reader, typeID glint.WireType) (interface{}, error) {
	switch typeID & glint.WireTypeMask {
	case glint.WireBool:
		return reader.ReadBool(), nil
	case glint.WireInt:
		return reader.ReadInt(), nil
	case glint.WireInt8:
		return int(reader.ReadInt8()), nil
	case glint.WireInt16:
		return int(reader.ReadInt16()), nil
	case glint.WireInt32:
		return int(reader.ReadInt32()), nil
	case glint.WireInt64:
		return reader.ReadInt64(), nil
	case glint.WireUint:
		return reader.ReadUint(), nil
	case glint.WireUint8:
		return uint(reader.ReadUint8()), nil
	case glint.WireUint16:
		return uint(reader.ReadUint16()), nil
	case glint.WireUint32:
		return uint(reader.ReadUint32()), nil
	case glint.WireUint64:
		return reader.ReadUint64(), nil
	case glint.WireFloat32:
		return reader.ReadFloat32(), nil
	case glint.WireFloat64:
		return reader.ReadFloat64(), nil
	case glint.WireString:
		return reader.ReadString(), nil
	case glint.WireBytes:
		return reader.Read(reader.ReadVarint()), nil
	case glint.WireTime:
		return reader.ReadTime(), nil
	default:
		return nil, fmt.Errorf("unsupported primitive field type: %v", typeID)
	}
}

// mapToInterface converts a map field to map[string]interface{} with full nested support
func (t *Template) mapToInterface(reader *glint.Reader, field *glint.PrinterSchemaField) (interface{}, error) {
	length := reader.ReadVarint()
	result := make(map[string]interface{})

	if length == 0 {
		return result, nil
	}

	keyType := field.MapType[0]
	valueType := field.MapType[1]

	for i := uint(0); i < length; i++ {
		// Read key and convert to string for map access
		keyVal, err := t.fieldValueByType(reader, keyType, &glint.PrinterSchemaField{TypeID: keyType})
		if err != nil {
			return nil, fmt.Errorf("failed to read map key %d: %v", i, err)
		}
		keyStr := fmt.Sprintf("%v", keyVal)

		// Read value with full type support
		value, err := t.fieldValueByType(reader, valueType, &glint.PrinterSchemaField{
			TypeID:  valueType,
			MapType: field.MapType, // Preserve map type info for nested maps
		})
		if err != nil {
			return nil, fmt.Errorf("failed to read map value %d: %v", i, err)
		}

		result[keyStr] = value
	}

	return result, nil
}

// structToInterface converts nested struct to map[string]interface{}
func (t *Template) structToInterface(reader *glint.Reader, field *glint.PrinterSchemaField) (interface{}, error) {
	// For struct fields, we need to access the nested schema
	if field.NestedSchema == nil {
		return nil, fmt.Errorf("struct field missing nested schema")
	}

	result := make(map[string]interface{})

	// Process each field in the nested struct
	for _, nestedField := range field.NestedSchema.Fields {
		value, err := t.fieldValueByType(reader, nestedField.TypeID, &nestedField)
		if err != nil {
			return nil, fmt.Errorf("failed to convert nested field '%s': %v", nestedField.Name, err)
		}
		result[nestedField.Name] = value
	}

	return result, nil
}

// sliceValueByType handles slices with any element type
func (t *Template) sliceValueByType(reader *glint.Reader, wireType glint.WireType, field *glint.PrinterSchemaField) (interface{}, error) {
	length := reader.ReadVarint()
	result := make([]interface{}, length)

	// Handle delta-encoded slices
	if wireType&glint.WireDeltaFlag != 0 {
		return t.deltaSliceToInterface(reader, field, int(length))
	}

	elementType := wireType & glint.WireTypeMask

	// Handle slices of complex types
	for i := uint(0); i < length; i++ {
		var value interface{}
		var err error

		switch elementType {
		case glint.WireStruct:
			if field.NestedSchema != nil {
				value, err = t.structToInterface(reader, field)
			} else {
				err = fmt.Errorf("slice of structs missing nested schema")
			}
		case glint.WireMap:
			value, err = t.mapToInterface(reader, field)
		default:
			value, err = t.simpleFieldToInterface(reader, glint.WireType(elementType))
		}

		if err != nil {
			return nil, fmt.Errorf("failed to read slice element %d: %v", i, err)
		}
		result[i] = value
	}

	return result, nil
}

// sliceToInterface converts a slice field to []interface{} (legacy wrapper)
func (t *Template) sliceToInterface(reader *glint.Reader, field *glint.PrinterSchemaField) (interface{}, error) {
	return t.sliceValueByType(reader, field.TypeID, field)
}

// deltaSliceToInterface converts a delta-encoded slice to []interface{}
func (t *Template) deltaSliceToInterface(reader *glint.Reader, field *glint.PrinterSchemaField, length int) (interface{}, error) {
	if length == 0 {
		return []interface{}{}, nil
	}

	result := make([]interface{}, length)
	baseType := field.TypeID & glint.WireTypeMask

	switch baseType {
	case glint.WireInt:
		// First value
		prev := reader.ReadInt()
		result[0] = prev
		// Subsequent values are deltas
		for i := 1; i < length; i++ {
			delta := reader.ReadZigzagVarint()
			prev += delta
			result[i] = prev
		}
	case glint.WireInt16:
		prev := reader.ReadInt16()
		result[0] = int(prev)
		for i := 1; i < length; i++ {
			delta := int16(reader.ReadZigzagVarint())
			prev += delta
			result[i] = int(prev)
		}
	case glint.WireInt32:
		prev := reader.ReadInt32()
		result[0] = int(prev)
		for i := 1; i < length; i++ {
			delta := int32(reader.ReadZigzagVarint())
			prev += delta
			result[i] = int(prev)
		}
	case glint.WireInt64:
		prev := reader.ReadInt64()
		result[0] = prev
		for i := 1; i < length; i++ {
			delta := int64(reader.ReadZigzagVarint())
			prev += delta
			result[i] = prev
		}
	case glint.WireUint:
		prev := reader.ReadUint()
		result[0] = prev
		for i := 1; i < length; i++ {
			delta := uint(reader.ReadZigzagVarint())
			prev += delta
			result[i] = prev
		}
	case glint.WireUint16:
		prev := reader.ReadUint16()
		result[0] = uint(prev)
		for i := 1; i < length; i++ {
			delta := uint16(reader.ReadZigzagVarint())
			prev += delta
			result[i] = uint(prev)
		}
	case glint.WireUint32:
		prev := reader.ReadUint32()
		result[0] = uint(prev)
		for i := 1; i < length; i++ {
			delta := uint32(reader.ReadZigzagVarint())
			prev += delta
			result[i] = uint(prev)
		}
	case glint.WireUint64:
		prev := reader.ReadUint64()
		result[0] = prev
		for i := 1; i < length; i++ {
			delta := uint64(reader.ReadZigzagVarint())
			prev += delta
			result[i] = prev
		}
	default:
		return nil, fmt.Errorf("delta encoding not supported for type %v", baseType)
	}

	return result, nil
}
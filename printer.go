package glint

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// The code for Print and its supporting parts are not written with the same strict performance
// concerns as the rest of the code in this file. Instead they're written with the intent of providing
// easy-to-use data structures for tooling such as commandline utilities.

// PrinterDocument represents the top level parts of a glint document, intended for tooling purposes.
type PrinterDocument struct {
	Flags  byte
	CRC32  []byte
	Schema Reader
	Body   Reader
}

// NewPrinterDocument reads a document from a Reader and returns a PrinterDocument
func NewPrinterDocument(r *Reader) PrinterDocument {
	return PrinterDocument{
		Flags:  r.ReadByte(),
		CRC32:  r.Read(4),
		Schema: NewReader(r.Read(r.ReadVarint())),
		Body:   NewReader(r.Remaining()),
	}
}

// PrinterSchema represents the schema of a glint document, intended for tooling purposes.
type PrinterSchema struct {
	Fields []PrinterSchemaField

	NestedSchema *PrinterSchema
}

// NewPrinterSchema reads a schema from a Reader and returns a PrinterSchema
func NewPrinterSchema(r *Reader) PrinterSchema {
	s := PrinterSchema{}

	for {
		if len(r.Remaining()) == 0 {
			break
		}

		s.Fields = append(s.Fields, NewPrinterSchemaField(r))
	}

	return s
}

// PrinterSchemaField represents a single field in a schema, intended for tooling purposes.
type PrinterSchemaField struct {
	TypeID WireType
	Name   string

	IsSlice   bool
	IsPointer bool

	NestedSchema *PrinterSchema

	NestedSlice *PrinterSchemaField

	MapType [2]WireType
}

// NewRawPrinterSchemaField is almost identical to NewPrinterSchemaField except it doesn't read a name. Used for when
// parsing nested struct schemas like arrays or maps.
func NewRawPrinterSchemaField(r *Reader) PrinterSchemaField {
	f := PrinterSchemaField{
		TypeID: WireType(r.ReadVarint()),
	}
	f.ReadSubSchema(r)

	return f
}

// NewPrinterSchemaField reads a single field from a Reader and returns a PrinterSchemaField
func NewPrinterSchemaField(r *Reader) PrinterSchemaField {
	f := PrinterSchemaField{
		TypeID: WireType(r.ReadVarint()),
		Name:   string(r.Read(r.ReadVarint())),
	}

	f.IsSlice = f.TypeID&WireSliceFlag > 0
	f.IsPointer = f.TypeID&WirePtrFlag > 0

	f.ReadSubSchema(r)

	return f
}

// ReadSubSchema reads a nested schema from a Reader and sets it on the PrinterSchemaField
func (f *PrinterSchemaField) ReadSubSchema(r *Reader) {

	switch {
	case f.TypeID&WireTypeMask == WireStruct:
		nr := NewReader(r.Read(r.ReadVarint()))
		ns := NewPrinterSchema(&nr)
		f.NestedSchema = &ns

	case f.TypeID == WireMap:
		f.MapType = [2]WireType{WireType(r.ReadVarint()), WireType(r.ReadVarint())}

		switch {
		case f.MapType[1]&WireSliceFlag > 0:
			r.Unread(1)

			ns := NewRawPrinterSchemaField(r)
			f.NestedSlice = &ns

		case f.MapType[1]&WireTypeMask == WireStruct:

			nr := NewReader(r.Read(r.ReadVarint()))
			ns := NewPrinterSchema(&nr)

			f.NestedSchema = &ns // drop the schema on the last level of the slice we parsed

		case f.MapType[1] == WireMap:

			r.Unread(1)
			f.NestedSchema = &PrinterSchema{Fields: []PrinterSchemaField{NewRawPrinterSchemaField(r)}}
		}

	case f.TypeID&WireSliceFlag > 0:
		switch {
		case WireType(f.TypeID&WireTypeMask) == WireStruct:

			nr := NewReader(r.Read(r.ReadVarint()))
			ns := NewPrinterSchema(&nr)
			f.NestedSchema = &ns

		case f.TypeID == WireSliceFlag:

			ct := f.TypeID

			for {
				if ct == WireSliceFlag {
					f.NestedSlice = &PrinterSchemaField{TypeID: ct}
					f = f.NestedSlice

					ct = WireType(r.ReadVarint())
					continue
				}
				break
			}

			f.TypeID = ct

			if ct&WireTypeMask == WireStruct {
				nr := NewReader(r.Read(r.ReadVarint()))
				ns := NewPrinterSchema(&nr)

				f.NestedSchema = &ns // drop the schema on the last level of the slice we parsed
			}

		default: // just simple slice types
		}
	}
}

// typeIDString returns a string representation of the typeID of a PrinterSchemaField
func typeIDString(field PrinterSchemaField) string {

	id := field.TypeID

	var t string

	if id&WireSliceFlag > 0 {
		t += "[]"
		if id&WireDeltaFlag > 0 {
			t += "(delta)"
		}
	}

	if id&WirePtrFlag > 0 {
		t += "*"
	}

	switch WireType(id) & WireTypeMask {
	case WireBool:
		t += "Bool"
	case WireInt:
		t += "Int"
	case WireInt8:
		t += "Int8"
	case WireInt16:
		t += "Int16"
	case WireInt32:
		t += "Int32"
	case WireInt64:
		t += "Int64"
	case WireUint:
		t += "Uint"
	case WireUint8:
		t += "Uint8"
	case WireUint16:
		t += "Uint16"
	case WireUint32:
		t += "Uint32"
	case WireUint64:
		t += "Uint64"
	case WireFloat32:
		t += "Float32"
	case WireFloat64:
		t += "Float64"
	case WireString:
		t += "String"
	case WireBytes:
		t += "Bytes"
	case WireStruct:
		t += "Struct"
	case WireMap:

		keyType := typeIDString(PrinterSchemaField{TypeID: WireType(field.MapType[0])})
		valueType := typeIDString(PrinterSchemaField{TypeID: WireType(field.MapType[1])})

		t += "Map[" + keyType + "]" + valueType
	case WireTime:
		t += "Time"
	case 0:
		if field.NestedSlice != nil {
			t += typeIDString(*field.NestedSlice)
		}

	default:
		t += "unknown type"
	}

	return t
}

// fieldValueString returns a string representation of a document value
func fieldValueString(r *Reader, schemaField *PrinterSchemaField) string {

	typeID := WireType(schemaField.TypeID)

	if schemaField.TypeID&WirePtrFlag > 0 {
		nullCheck := r.ReadByte()
		if nullCheck == 0 {
			return "nil"
		}

		typeID ^= WirePtrFlag
	}

	switch typeID {
	case WireInt:
		return strconv.Itoa(r.ReadInt())
	case WireInt8:
		return strconv.Itoa(int(r.ReadInt8()))
	case WireInt16:
		return strconv.Itoa(int(r.ReadInt16()))
	case WireInt32:
		return strconv.Itoa(int(r.ReadInt32()))
	case WireInt64:
		return strconv.Itoa(int(r.ReadInt64()))
	case WireUint:
		return strconv.FormatUint(uint64(r.ReadUint()), 10)
	case WireUint8:
		return strconv.FormatUint(uint64(r.ReadUint8()), 10)
	case WireUint16:
		return strconv.FormatUint(uint64(r.ReadUint16()), 10)
	case WireUint32:
		return strconv.FormatUint(uint64(r.ReadUint32()), 10)
	case WireUint64:
		return strconv.FormatUint(r.ReadUint64(), 10)
	case WireFloat32:
		return strconv.FormatFloat(float64(r.ReadFloat32()), 'f', -1, 32)
	case WireFloat64:
		return strconv.FormatFloat(r.ReadFloat64(), 'f', -1, 64)
	case WireString:
		return r.ReadString()
	case WireBytes:
		return fmt.Sprintf("%v", r.Read(r.ReadVarint()))
	case WireTime:
		return fmt.Sprintf("%v", r.ReadTime())
	case WireBool:
		if r.ReadBool() {
			return "true"
		} else {
			return "false"
		}
	}
	panic(fmt.Sprintf("unknown type %v", schemaField.TypeID))
}

// SPrintSlice returns a string representation of a slice
func SPrintSlice(r *Reader, field *PrinterSchemaField, nestLevel int) string {

	var buf strings.Builder

	if field.NestedSlice != nil {
		for i, l := 0, r.ReadVarint(); i < int(l); i++ {
			fmt.Fprintf(&buf, "   %v└─  [%v]:\n", strings.Repeat("  ", nestLevel), i)
			fmt.Fprintf(&buf, "%v ", SPrintSlice(r, field.NestedSlice, nestLevel+1))
		}
		return buf.String()
	}

	if field.NestedSchema != nil {
		for i, l := 0, r.ReadVarint(); i < int(l); i++ {
			fmt.Fprintf(&buf, "   %v└─┐ [%v]:\n", strings.Repeat("  ", nestLevel), i)
			fmt.Fprintf(&buf, "%v", SPrintStruct(r, field.NestedSchema, nestLevel+1))
		}
		return buf.String()
	}

	if field.TypeID == WireBytes {
		fmt.Fprintf(&buf, "   %v├  %v \n", strings.Repeat("  ", nestLevel), fieldValueString(r, field))
		return buf.String()
	}

	// Check if this is a delta-encoded slice
	if field.TypeID&WireDeltaFlag != 0 {
		length := r.ReadVarint()
		if length == 0 {
			return buf.String()
		}
		
		// Read values based on type
		baseType := field.TypeID & WireTypeMask
		switch baseType {
		case WireInt:
			// First value
			prev := r.ReadInt()
			fmt.Fprintf(&buf, "   %v├  [0]: %v \n", strings.Repeat("  ", nestLevel), prev)
			// Subsequent values are deltas
			for i := uint(1); i < length; i++ {
				delta := r.ReadZigzagVarint()
				prev += delta
				fmt.Fprintf(&buf, "   %v├  [%v]: %v \n", strings.Repeat("  ", nestLevel), i, prev)
			}
		case WireInt16:
			// First value
			prev := r.ReadInt16()
			fmt.Fprintf(&buf, "   %v├  [0]: %v \n", strings.Repeat("  ", nestLevel), prev)
			// Subsequent values are deltas
			for i := uint(1); i < length; i++ {
				delta := int16(r.ReadZigzagVarint())
				prev += delta
				fmt.Fprintf(&buf, "   %v├  [%v]: %v \n", strings.Repeat("  ", nestLevel), i, prev)
			}
		case WireInt32:
			// First value
			prev := r.ReadInt32()
			fmt.Fprintf(&buf, "   %v├  [0]: %v \n", strings.Repeat("  ", nestLevel), prev)
			// Subsequent values are deltas
			for i := uint(1); i < length; i++ {
				delta := int32(r.ReadZigzagVarint())
				prev += delta
				fmt.Fprintf(&buf, "   %v├  [%v]: %v \n", strings.Repeat("  ", nestLevel), i, prev)
			}
		case WireInt64:
			// First value
			prev := r.ReadInt64()
			fmt.Fprintf(&buf, "   %v├  [0]: %v \n", strings.Repeat("  ", nestLevel), prev)
			// Subsequent values are deltas
			for i := uint(1); i < length; i++ {
				delta := int64(r.ReadZigzagVarint())
				prev += delta
				fmt.Fprintf(&buf, "   %v├  [%v]: %v \n", strings.Repeat("  ", nestLevel), i, prev)
			}
		case WireUint:
			// First value
			prev := r.ReadUint()
			fmt.Fprintf(&buf, "   %v├  [0]: %v \n", strings.Repeat("  ", nestLevel), prev)
			// Subsequent values are deltas
			for i := uint(1); i < length; i++ {
				delta := uint(r.ReadZigzagVarint())
				prev += delta
				fmt.Fprintf(&buf, "   %v├  [%v]: %v \n", strings.Repeat("  ", nestLevel), i, prev)
			}
		case WireUint16:
			// First value
			prev := r.ReadUint16()
			fmt.Fprintf(&buf, "   %v├  [0]: %v \n", strings.Repeat("  ", nestLevel), prev)
			// Subsequent values are deltas
			for i := uint(1); i < length; i++ {
				delta := uint16(r.ReadZigzagVarint())
				prev += delta
				fmt.Fprintf(&buf, "   %v├  [%v]: %v \n", strings.Repeat("  ", nestLevel), i, prev)
			}
		case WireUint32:
			// First value
			prev := r.ReadUint32()
			fmt.Fprintf(&buf, "   %v├  [0]: %v \n", strings.Repeat("  ", nestLevel), prev)
			// Subsequent values are deltas
			for i := uint(1); i < length; i++ {
				delta := uint32(r.ReadZigzagVarint())
				prev += delta
				fmt.Fprintf(&buf, "   %v├  [%v]: %v \n", strings.Repeat("  ", nestLevel), i, prev)
			}
		case WireUint64:
			// First value
			prev := r.ReadUint64()
			fmt.Fprintf(&buf, "   %v├  [0]: %v \n", strings.Repeat("  ", nestLevel), prev)
			// Subsequent values are deltas
			for i := uint(1); i < length; i++ {
				delta := uint64(r.ReadZigzagVarint())
				prev += delta
				fmt.Fprintf(&buf, "   %v├  [%v]: %v \n", strings.Repeat("  ", nestLevel), i, prev)
			}
		default:
			// For unsupported types, just show a message
			fmt.Fprintf(&buf, "   %v├  <delta encoding not supported for this type> \n", strings.Repeat("  ", nestLevel))
		}
		return buf.String()
	}

	field.TypeID &= WireTypeMask
	for i, l := 0, r.ReadVarint(); i < int(l); i++ {
		fmt.Fprintf(&buf, "   %v├  [%v]: %v \n", strings.Repeat("  ", nestLevel), i, fieldValueString(r, field))
	}

	return buf.String()
}

// PrintSlice prints a slice to stdout
func PrintSlice(r *Reader, field *PrinterSchemaField, nestLevel int) {
	fmt.Println(SPrintSlice(r, field, nestLevel))
}

// SPrintMap returns a string representation of a map
func SPrintMap(r *Reader, schema *PrinterSchemaField, nestLevel int) string {

	var buf strings.Builder

	for i, l := 0, r.ReadVarint(); i < int(l); i++ {
		key := fieldValueString(r, &PrinterSchemaField{TypeID: WireType(schema.MapType[0])})

		rem := r.position // this allows us to print the byte values next to the textual representation of the field

		var value string

		if schema.MapType[1]&WirePtrFlag > 0 && r.ReadByte() == 0 { // cater for pointer nils
			value = "nil"

			goto the_print
		}

		switch typ := schema.MapType[1]; {
		case typ&WireSliceFlag > 0:
			value = "\n" + SPrintSlice(r, schema.NestedSlice, nestLevel)

		case typ&WireTypeMask == WireStruct:
			value = "\n" + SPrintStruct(r, schema.NestedSchema, nestLevel)

		case schema.MapType[1] == WireMap:
			value = "\n" + SPrintMap(r, &schema.NestedSchema.Fields[0], nestLevel+1)

		default:
			value = fmt.Sprintf("%v %v", fieldValueString(r, &PrinterSchemaField{TypeID: WireType(schema.MapType[1])}), r.bytes[rem:r.position])
		}

	the_print:
		fmt.Fprintf(&buf, "   %v├  {%v}: %v \n", strings.Repeat("  ", nestLevel), key, value)
	}

	return buf.String()
}

// PrintMap prints a map within a document
func PrintMap(r *Reader, schema *PrinterSchemaField, nestLevel int) {
	fmt.Println(SPrintMap(r, schema, nestLevel))
}

// SPrintStruct returns a string representation of a struct within a document
func SPrintStruct(r *Reader, schema *PrinterSchema, nestLevel int) string {
	return SPrintStructVerbose(r, schema, nestLevel, false)
}

// SPrintStructWithColors returns a string representation of a struct with optional color support
func SPrintStructWithColors(r *Reader, schema *PrinterSchema, nestLevel int, useColors bool) string {
	return SPrintStructVerboseWithColors(r, schema, nestLevel, false, useColors)
}

// SPrintStructVerbose returns a string representation of a struct within a document with the option to print the bytes for values
func SPrintStructVerbose(r *Reader, schema *PrinterSchema, nestLevel int, printBytes bool) string {
	return SPrintStructVerboseWithColors(r, schema, nestLevel, printBytes, terminaloutput)
}

// SPrintStructVerboseWithColors returns a string representation of a struct with optional color support
func SPrintStructVerboseWithColors(r *Reader, schema *PrinterSchema, nestLevel int, printBytes bool, useColors bool) string {
	var buf strings.Builder

t:
	for fieldIndex, f := range schema.Fields {

		var char string
		if fieldIndex == len(schema.Fields)-1 {
			char = "└─"
		} else {
			char = "├─"
		}

		if f.TypeID&WirePtrFlag > 0 {
			if r.ReadByte() == 0 { // check for nil and continue to the next field if we find it
				fmt.Fprintf(&buf, "   %v%v %v: %v \n", strings.Repeat("  ", nestLevel), char, f.Name, "nil")
				continue
			}
			f.TypeID ^= WirePtrFlag // allow the parsing to carry on as normal by stripping the pointer flag off
		}

		switch {
		case f.TypeID&WireSliceFlag > 0:
			fmt.Fprintf(&buf, "   %v%v %v:\n", strings.Repeat("  ", nestLevel), char, f.Name)
			fmt.Fprintf(&buf, "%v", SPrintSlice(r, &f, nestLevel)) // Note: SPrintSlice doesn't use colors

			continue t

		case f.TypeID == WireStruct:
			fmt.Fprintf(&buf, "   %v%v %v\n", strings.Repeat("  ", nestLevel), char, f.Name)
			fmt.Fprintf(&buf, "%v", SPrintStructWithColors(r, f.NestedSchema, nestLevel+1, useColors))
			continue

		case f.TypeID == WireMap:
			fmt.Fprintf(&buf, "   %v%v %v:\n", strings.Repeat("  ", nestLevel), char, f.Name)
			fmt.Fprintf(&buf, "%v", SPrintMap(r, &f, nestLevel+1)) // Note: SPrintMap doesn't use colors
			continue
		}

		rem := r.position

		valueString := fieldValueString(r, &f)

		var valueBytes any
		if printBytes {
			valueBytes = r.bytes[rem:r.position]
		} else {
			valueBytes = ""
		}

		fmt.Fprintf(&buf, "   %v%v %v: %v %v \n", strings.Repeat("  ", nestLevel), char, f.Name, colorTextWithFlag(valueString, Purple, useColors), valueBytes)
	}
	return buf.String()
}

// PrintStruct prints a struct within a document, or the top level of the document itself
func PrintStruct(r *Reader, schema *PrinterSchema, nestLevel int) {
	fmt.Println(SPrintStruct(r, schema, nestLevel))
}

// PrintSchema prints the schema of a document
func PrintSchema(schema *PrinterSchema, nestLevel int) {

	for i, v := range schema.Fields {

		var char string
		if i == len(schema.Fields)-1 || v.NestedSchema != nil {
			char = "└─"
		} else {
			char = "├─"
		}

		fmt.Printf("│  %v%v %v: %v\n", strings.Repeat("  ", nestLevel), char, typeIDString(v), v.Name)

		var nested *PrinterSchema
		if v.NestedSlice != nil && v.NestedSlice.NestedSchema != nil {
			nested = v.NestedSlice.NestedSchema
		} else {
			nested = v.NestedSchema
		}

		if nested != nil {
			PrintSchema(nested, nestLevel+1)
			continue
		}
	}

	if schema.NestedSchema != nil {
		PrintSchema(schema.NestedSchema, nestLevel+1)
	}
}

// Print prints a document
func Print(bytes []byte) []byte {

	r := NewReader(bytes) // used to traverse the doc easily

	doc := NewPrinterDocument(&r)

	defer func() {
		if rc := recover(); rc != nil {
			fmt.Println("remaining schema:", doc.Schema.Remaining())
			fmt.Println("remaining body:", doc.Body)

			panic(rc)
		}
	}()

	schema := NewPrinterSchema(&doc.Schema)

	fmt.Println("Glint Document")

	fmt.Println("├─ Schema")
	PrintSchema(&schema, 0)

	fmt.Println("└─ Values")
	PrintStruct(&doc.Body, &schema, 0)

	return doc.Body.Remaining()
}

// SPrint returns a string representation of a glint document, similar to Print but returns the string instead of printing to stdout
func SPrint(bytes []byte) string {
	if len(bytes) == 0 {
		return ""
	}
	var buf strings.Builder

	r := NewReader(bytes) // used to traverse the doc easily

	doc := NewPrinterDocument(&r)

	schema := NewPrinterSchema(&doc.Schema)

	buf.WriteString("Glint Document\n")
	buf.WriteString("├─ Schema\n")

	// Build schema string using the same logic as PrintSchema
	var printSchema func(*PrinterSchema, int)
	printSchema = func(schema *PrinterSchema, nestLevel int) {
		for i, v := range schema.Fields {
			var char string
			if i == len(schema.Fields)-1 || v.NestedSchema != nil {
				char = "└─"
			} else {
				char = "├─"
			}

			buf.WriteString(fmt.Sprintf("│  %v%v %v: %v\n", strings.Repeat("  ", nestLevel), char, typeIDString(v), v.Name))

			var nested *PrinterSchema
			if v.NestedSlice != nil && v.NestedSlice.NestedSchema != nil {
				nested = v.NestedSlice.NestedSchema
			} else {
				nested = v.NestedSchema
			}

			if nested != nil {
				printSchema(nested, nestLevel+1)
				continue
			}
		}

		if schema.NestedSchema != nil {
			printSchema(schema.NestedSchema, nestLevel+1)
		}
	}

	printSchema(&schema, 0)

	buf.WriteString("└─ Values\n")

	// Build values string without colors
	structStr := SPrintStructWithColors(&doc.Body, &schema, 0, false)
	buf.WriteString(structStr)

	return buf.String()
}

var terminaloutput = func() bool {
	o, _ := os.Stdout.Stat()
	return (o.Mode() & os.ModeCharDevice) == os.ModeCharDevice
}()

const (
	Red    = "\033[91m"
	Orange = "\033[38;5;208m"
	Yellow = "\033[93m"
	Green  = "\033[92m"
	Blue   = "\033[94m"
	Purple = "\033[95m"
	Cyan   = "\033[96m"
	White  = "\033[97m"
	Reset  = "\033[0m"
)

// colorTextWithFlag takes a string, color, and useColors flag and returns the colored string if colors are enabled
func colorTextWithFlag(text, color string, useColors bool) string {
	if !useColors {
		return text
	}
	return color + text + Reset
}

// Document is a type alias for []byte that implements fmt.Formatter for pretty printing glint documents
type Document []byte

// String implements fmt.Stringer interface
func (d Document) String() string {
	if len(d) == 0 {
		return ""
	}
	return SPrint([]byte(d))
}

// Format implements fmt.Formatter interface for custom formatting verbs
func (d Document) Format(f fmt.State, verb rune) {
	switch verb {
	case 's':
		// %s - pretty printed document
		f.Write([]byte(d.String()))
	case 'v':
		if f.Flag('+') {
			// %+v - verbose format showing both tree and hex
			f.Write([]byte(fmt.Sprintf("Glint Document (hex: %x)\n%s", []byte(d), d.String())))
		} else {
			// %v - same as %s
			f.Write([]byte(d.String()))
		}
	case 'x':
		// %x - lowercase hex representation
		fmt.Fprintf(f, "%x", []byte(d))
	case 'X':
		// %X - uppercase hex representation
		fmt.Fprintf(f, "%X", []byte(d))
	case 'q':
		// %q - quoted hex representation
		fmt.Fprintf(f, "%q", []byte(d))
	default:
		// fallback to default []byte formatting
		fmt.Fprintf(f, "%%!%c(glint.Document=%x)", verb, []byte(d))
	}
}

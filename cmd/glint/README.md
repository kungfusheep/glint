# glint - Glint CLI Tool

A command-line utility for inspecting and manipulating glint binary documents.

## Installation

```bash
go install github.com/kungfusheep/glint/cmd/glint@latest
```

Or build from source:
```bash
go build ./cmd/glint
```

## Usage

### Document Inspection

Inspect glint documents with a human-readable tree view:

```bash
# From file
cat data.glint | glint

# From hex bytes  
echo "0 236 149 102 93 11 14 4 110 97 109 101 2 3 97 103 101 5 65 108 105 99 101 60" | glint debug ascii
```

Output:
```
Glint Document
├─ Schema
│  ├─ String: name
│  └─ Int: age
└─ Values
   ├─ name: Alice  
   └─ age: 30
```

### Schema Extraction

Display only the document schema without values:

```bash
glint schema < data.glint
```

Output:
```
Glint Schema
│  ├─ String: name
│  └─ Int: age
```

### Field Extraction

Extract specific field values using dot notation:

```bash
glint get name < data.glint              # Simple field
glint get tags[0] < data.glint           # Array access  
glint get user.name < data.glint         # Nested field
glint get scores[level1] < data.glint    # Map access
```

### Template Output

Format document data using Go templates:

```bash
# Simple template string
glint printf "Hello {{.name}}, you are {{.age}} years old!" < data.glint
# Output: Hello Alice, you are 30 years old!

# JSON-like output
glint printf '{"user": "{{.name}}", "age": {{.age}}}' < data.glint
# Output: {"user": "Alice", "age": 30}

# Template with array iteration
glint printf "Tags: {{range .tags}}{{.}} {{end}}" < data.glint
# Output: Tags: developer golang glint

# Template with map access
glint printf "Score: {{.scores.level1}}, Name: {{.metadata.name}}" < data.glint
# Output: Score: 100, Name: Alice

# Template from file
glint printf -f report.tmpl < data.glint
```

### JSON Input

Convert JSON data to glint format for easy testing and data creation:

```bash
# Convert JSON to glint and inspect
echo '{"name":"Alice","age":30,"tags":["dev","go"]}' | glint convert --from json | glint

# Extract fields and use templates  
echo '{"name":"Alice","age":30}' | glint convert --from json | glint get name
echo '{"name":"Alice","age":30}' | glint convert --from json | glint printf "Hello {{.name}}"
```

### CSV Export

Convert glint documents to CSV format with intelligent flattening for spreadsheet and bash processing:

```bash
# Convert objects to CSV
echo '{"name":"Alice","age":30}' | glint convert --from json | glint convert --to csv

# Array of objects becomes table rows
echo '[{"name":"Alice","age":30},{"name":"Bob","age":25}]' | glint convert --from json | glint convert --to csv
```

CSV output flattens nested objects with dot notation and handles arrays intelligently.

### Go Struct Generation

Generate type-safe Go structs from glint documents for development workflows:

```bash
# Generate Go struct from JSON data
echo '{"name":"Alice","age":30,"active":true}' | glint convert --from json | glint generate go models.User
```

Generates complete Go structs with proper types, tags, and imports.

### Schema Compatibility Checking

Check if schema changes between glint documents are backward/forward compatible:

```bash
# Check compatibility between two schemas
echo '{"name":"Alice","age":30}' | glint convert --from json > old.glint
echo '{"name":"Alice","age":"thirty"}' | glint convert --from json | glint compat old.glint
```

Reports safe changes (add/remove fields) vs breaking changes (type changes).

### Document Statistics

Analyze glint document structure and size breakdown:

```bash
echo '{"name":"Alice","age":30,"tags":["dev","go"]}' | glint convert --from json | glint stats
```

Shows size breakdown, field count, nesting deglinth, and wire type distribution.

### Wire Format Debugging

Decode variable-length integers (varints) used in glint's wire format:

```bash
# Decode unsigned varint
glint debug varint 172 2                  # Output: 300

# Decode zigzag-encoded signed varint
glint debug zigzag 3                      # Output: -2
```


## Features

**Core**: Inspect documents, extract fields, template output  
**Conversion**: JSON ↔ glint, CSV export, Go struct generation  
**Analysis**: Schema inspection, compatibility checking, document statistics  
**Debug**: Wire format tools, varint/zigzag decoding

## Current Limitations

### Type Support
- **JSON Struct field skipping**: Cannot skip struct fields during field extraction, limiting navigation through complex documents
- **Time formatting**: Time fields are displayed in default format with no customization oglintions

### JSON Input
- **Type inference**: JSON numbers are auto-detected as int vs float64, but no control over specific numeric types
- **Null values**: JSON null values are converted to emglinty strings rather than pointer types
- **Mixed arrays**: Arrays with mixed types are converted to string arrays (glint doesn't support mixed-type arrays)

package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// fieldPathToTemplate converts a field path like "user.name" or "nums1[level1]" to Go template syntax
func fieldPathToTemplate(fieldPath string) (string, error) {
	if fieldPath == "" {
		return "", fmt.Errorf("empty field path")
	}

	// Handle simple field access (no dots or brackets)
	if !strings.Contains(fieldPath, ".") && !strings.Contains(fieldPath, "[") {
		return "{{." + fieldPath + "}}", nil
	}

	// Parse the field path and convert to template syntax
	return parseFieldPath(fieldPath)
}

// parseFieldPath parses a field path and converts it to Go template syntax
func parseFieldPath(path string) (string, error) {
	// Handle complex cases with multiple bracket accesses
	// Examples: items[1].name, data[key][0], data[config][server][port]
	
	result, err := parseComplexPath(path)
	if err != nil {
		return "", err
	}
	
	return "{{" + result + "}}", nil
}

// parseComplexPath handles complex field paths with multiple bracket accesses
func parseComplexPath(path string) (string, error) {
	// Split by dots first
	parts := strings.Split(path, ".")
	var resultParts []string
	
	for i, part := range parts {
		if part == "" {
			continue
		}
		
		// Handle bracket access within this part
		processedPart, needsParens, err := parsePartWithBrackets(part, i == 0)
		if err != nil {
			return "", err
		}
		
		if needsParens && i < len(parts)-1 {
			// If this part has array access and there are more parts, wrap in parentheses
			processedPart = "(" + processedPart + ")"
		}
		
		resultParts = append(resultParts, processedPart)
	}
	
	if len(resultParts) == 0 {
		return "", fmt.Errorf("no valid field parts found")
	}
	
	// Join parts appropriately
	hasIndexAccess := false
	indexPart := -1
	for i, part := range resultParts {
		if strings.Contains(part, "index ") {
			hasIndexAccess = true
			indexPart = i
			break
		}
	}
	
	if hasIndexAccess {
		// Handle array access - need to combine field access before index
		if indexPart == 0 {
			// First part is an index access (e.g., skills[0])
			result := resultParts[0]
			if len(resultParts) > 1 {
				result += "." + strings.Join(resultParts[1:], ".")
			}
			return result, nil
		} else {
			// Array access is later in the chain (e.g., user.skills[0])
			// Combine the field parts before the index
			fieldParts := resultParts[:indexPart]
			indexPartString := resultParts[indexPart]
			
			// Extract the field name and index from the index part
			// indexPartString looks like "index skills 0"
			parts := strings.Fields(indexPartString)
			if len(parts) >= 3 && parts[0] == "index" {
				fieldName := parts[1]
				indexValue := parts[2]
				
				// Build the complete field path
				fieldPath := strings.Join(fieldParts, ".") + "." + fieldName
				result := fmt.Sprintf("(index %s %s)", fieldPath, indexValue)
				
				// Add any remaining parts after the index
				if indexPart+1 < len(resultParts) {
					remainingParts := resultParts[indexPart+1:]
					result += "." + strings.Join(remainingParts, ".")
				}
				
				return result, nil
			}
		}
	}
	
	return strings.Join(resultParts, "."), nil
}

// parsePartWithBrackets processes a single part that may contain bracket notation
func parsePartWithBrackets(part string, isFirst bool) (string, bool, error) {
	if !strings.Contains(part, "[") {
		// Simple field access
		if isFirst {
			return "." + part, false, nil
		}
		return part, false, nil
	}
	
	// Handle multiple bracket accesses: field[key1][key2][index]
	brackets := findBracketRanges(part)
	if len(brackets) == 0 {
		return "", false, fmt.Errorf("invalid bracket notation in: %s", part)
	}
	
	// Extract the base field name
	fieldName := part[:brackets[0].start]
	if fieldName == "" {
		return "", false, fmt.Errorf("empty field name in: %s", part)
	}
	
	// Process each bracket access
	result := ""
	if isFirst {
		result = "." + fieldName
	} else {
		result = fieldName
	}
	
	hasArrayAccess := false
	
	for _, bracket := range brackets {
		key := part[bracket.start+1 : bracket.end]
		if key == "" {
			return "", false, fmt.Errorf("empty bracket content in: %s", part)
		}
		
		// Check if it's numeric (array access)
		if index, err := strconv.Atoi(key); err == nil {
			// Array access - use index function
			if hasArrayAccess {
				return "", false, fmt.Errorf("multiple array accesses not supported in: %s", part)
			}
			result = fmt.Sprintf("index %s %d", result, index)
			hasArrayAccess = true
		} else {
			// Map access - use dot notation
			result += "." + key
		}
	}
	
	return result, hasArrayAccess, nil
}

// bracketRange represents start and end positions of bracket notation
type bracketRange struct {
	start, end int
}

// findBracketRanges finds all [key] patterns in a string
func findBracketRanges(s string) []bracketRange {
	var ranges []bracketRange
	start := -1
	
	for i, r := range s {
		if r == '[' {
			if start != -1 {
				// Nested brackets - invalid
				return nil
			}
			start = i
		} else if r == ']' {
			if start == -1 {
				// Unmatched closing bracket
				return nil
			}
			ranges = append(ranges, bracketRange{start: start, end: i})
			start = -1
		}
	}
	
	if start != -1 {
		// Unmatched opening bracket
		return nil
	}
	
	return ranges
}

// isValidIdentifier checks if a string is a valid Go identifier
func isValidIdentifier(s string) bool {
	if s == "" {
		return false
	}
	
	// Simple check - starts with letter or underscore, contains only letters, digits, underscore
	validIdentifier := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	return validIdentifier.MatchString(s)
}
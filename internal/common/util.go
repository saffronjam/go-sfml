package common

import (
	"github.com/golang-cz/textcase"
	"strings"
)

// CleanCType returns the raw C typedef name with no "const ", "struct ", or "*".
// e.g. "const sfVector2i*" → "sfVector2i"
func CleanCType(cType string) string {
	t := strings.ReplaceAll(cType, "const ", "")
	t = strings.ReplaceAll(t, "struct ", "")
	t = strings.ReplaceAll(t, "*", "")
	return strings.TrimSpace(t)
}

// StripPointer removes any pointer symbols from a type name.
func StripPointer(typeName string) string {
	typeName = strings.TrimSpace(typeName)
	if strings.HasPrefix(typeName, "*") {
		return strings.TrimPrefix(typeName, "*")
	}
	return typeName
}

func IsVoidReturnType(returnType string) bool {
	returnType = strings.TrimSpace(returnType)
	return returnType == "void" || returnType == "void*" || returnType == ""
}

// IsPointerType checks if a type name is a pointer type.
// Works for both C and Go pointer types.
func IsPointerType(typeName string) bool {
	// Check if prefix or suffix is a pointer symbol after trimming spaces.
	typeName = strings.TrimSpace(typeName)
	if strings.HasPrefix(typeName, "*") || strings.HasSuffix(typeName, "*") {
		return true
	}

	return false
}

// SanitizeFieldName fixes a field name if it is not valid in Go.
// For instance, it should not start with a digit or be called "type" or "func".
func SanitizeFieldName(field Field) Field {
	isBad := func(name string) bool {
		if name == "" {
			return true
		}

		if len(name) == 0 || (name[0] >= '0' && name[0] <= '9') {
			return true // Starts with a digit
		}

		reservedKeywords := []string{"type", "func", "interface", "struct", "map", "chan"}
		for _, keyword := range reservedKeywords {
			if name == keyword {
				return true // Matches a reserved keyword
			}
		}

		return false // Valid field name
	}

	if isBad(field.Name) {
		return Field{
			Name: textcase.CamelCase(field.Type),
			Type: field.Type,
		}

	}

	return Field{
		Name: field.Name,
		Type: field.Type,
	}
}

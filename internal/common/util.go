package common

import (
	"fmt"
	"github.com/golang-cz/textcase"
	"strings"
	"unicode"
)

var nativeTypes = map[string]struct{}{
	"int":     {},
	"int8":    {},
	"int16":   {},
	"int32":   {},
	"int64":   {},
	"uint":    {},
	"uint8":   {},
	"uint16":  {},
	"uint32":  {},
	"uint64":  {},
	"float32": {},
	"float64": {},
	"bool":    {},
	"string":  {},
	"byte":    {},
	"uintptr": {},
}

var goKeywords = map[string]bool{
	"break":       true,
	"case":        true,
	"chan":        true,
	"const":       true,
	"continue":    true,
	"default":     true,
	"defer":       true,
	"else":        true,
	"fallthrough": true,
	"for":         true,
	"func":        true,
	"go":          true,
	"goto":        true,
	"if":          true,
	"import":      true,
	"interface":   true,
	"map":         true,
	"package":     true,
	"range":       true,
	"return":      true,
	"select":      true,
	"struct":      true,
	"switch":      true,
	"type":        true,
	"var":         true,
}

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

func MakePointerType(typeName string) string {
	// No type, like void or empty string, should return empty.
	if typeName == "" {
		return ""
	}

	typeName = strings.TrimSpace(typeName)
	if !strings.HasPrefix(typeName, "*") {
		return "*" + typeName
	}
	return typeName
}

func IsVoidReturnType(returnType string) bool {
	returnType = strings.TrimSpace(returnType)
	return returnType == "void" || returnType == "void*" || returnType == ""
}

func IsNativeGoType(typeName string) bool {
	typeName = strings.TrimSpace(typeName)
	if _, ok := nativeTypes[typeName]; ok {
		return true
	}

	return false
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

// SanitizeFieldNameStr sanitizes a string to be a valid Go identifier.
// It checks for Go keywords and prepends/appends underscores if necessary.
func SanitizeFieldNameStr(name string) string {
	if name == "" {
		return "" // Let caller decide default for empty
	}
	if goKeywords[name] {
		return name + "_"
	}
	if r := rune(name[0]); unicode.IsDigit(r) {
		return "_" + name
	}
	return name
}

// SanitizeFieldName fixes a field name if it is not valid in Go.
// For instance, it should not start with a digit or be called "type" or "func".
func SanitizeFieldName(field Field) Field {
	isBad := func(name string) bool {
		if name == "" {
			return true
		}

		if unicode.IsDigit(rune(name[0])) {
			return true // Starts with a digit
		}

		if goKeywords[name] {
			return true // Matches a reserved keyword
		}
		return false // Valid field name
	}

	if isBad(field.Name) {
		// Fallback to a name derived from the type if the original name is bad.
		// Ensure this fallback is also sanitized.
		newName := textcase.CamelCase(StripPointer(field.Type))
		if newName == "" || isBad(newName) { // if type is also problematic or empty
			if field.Name != "" { // try to salvage original name by prefixing
				return Field{Name: SanitizeFieldNameStr("p_" + field.Name), Type: field.Type}
			}
			return Field{Name: "arg", Type: field.Type} // absolute fallback
		}
		return Field{
			Name: SanitizeFieldNameStr(newName), // Sanitize the generated name too
			Type: field.Type,
		}
	}

	return Field{
		Name: field.Name,
		Type: field.Type,
	}
}

func TypeConverterToC(cRawType string) string {
	cleanCType := CleanCType(cRawType)
	if IsPointerType(cRawType) {
		if cleanCType == "void" {
			return "unsafe.Pointer"
		}

		return fmt.Sprintf("(*C.%s)", cleanCType)
	}

	if cleanCType == "sfBool" {
		return "boolToSfBool"
	}

	if cleanCType == "string" || cleanCType == "sfString" {
		return "C.CString"
	}

	// If prefixed with "unsigned ", replace with it with u
	if strings.HasPrefix(cleanCType, "unsigned ") {
		return fmt.Sprintf("C.%s", "u"+strings.TrimPrefix(cleanCType, "unsigned "))
	}

	return fmt.Sprintf("C.%s", cleanCType)
}

func TypeConverterToGo(goType string) string {
	if IsPointerType(goType) {
		return fmt.Sprintf("(%s)", goType)
	}

	if goType == "bool" {
		return "sfBoolToBool"
	}

	if goType == "string" {
		return "C.GoString"
	}

	return fmt.Sprintf("%s", goType)
}

func PrependReturnType(returnType string, prepend string) string {
	if returnType == "" {
		return prepend
	}

	if strings.HasPrefix(returnType, "(") {
		// If the return type is already a function signature, prepend to the first part
		return fmt.Sprintf("(%s, %s", prepend, returnType[1:])
	}

	return fmt.Sprintf("(%s, %s)", prepend, returnType)
}

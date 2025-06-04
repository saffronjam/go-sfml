package common

// TypeDecl represents an entry in types.json.
type TypeDecl struct {
	ID          int          `json:"id"`             // Unique identifier for the type
	Name        string       `json:"name"`           // e.g. "sfVector2i", "sfColor", "sfEvent"
	Type        string       `json:"type,omitempty"` // e.g. "sfVector2i", "sfColor"
	Enumerators []Enumerator `json:"enumerators,omitempty"`
}

type Enumerator struct {
	Name string `json:"name"`
}

type Field struct {
	Name string
	Type string
}

// FunctionDecl represents a C function entry from functions.json.
type FunctionDecl struct {
	Name       string  `json:"name"`
	Parameters []Field `json:"parameters"`
	ReturnType string  `json:"return_type"`
}

// StructInfo holds the Go‐side name of a vector typedef and its field names.
type StructInfo struct {
	GoName  string
	Fields  []Field // Go‐side field names, e.g. "X", "Y", "Z", "W"
	CFields []Field // C‐side field names, e.g. "x", "y", "z", "w"
}

type Struct struct {
	Name   string
	Fields []Field
}

type Enum struct {
	Name        string
	Enumerators []Enumerator
}

type FunctionHeader struct {
	MethodName string  // e.g. "GetPosition"
	Parameters []Field // Function parameter
	ReturnType string  // e.g. "Vector2i", "int32" or omit for empty (void)
}
type ReceiverFunctionHeader struct {
	ReceiverName string  // e.g. "r"
	ReceiverType string  // e.g. "*RenderWindow"
	MethodName   string  // e.g. "GetPosition"
	Parameters   []Field // Function parameter
	ReturnType   string  // e.g. "Vector2i", "int32" or omit for empty (void)
}

type FunctionBody struct {
	Rows []string // Each row is a line of code, will be tab‐indented and joined with newlines
}

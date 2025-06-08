package common

// TypeDecl represents an entry in types.json.
type TypeDecl struct {
	ID          int          `json:"id"`             // Unique identifier for the type
	Name        string       `json:"name"`           // e.g. "sfVector2i", "sfColor", "sfEvent"
	Type        string       `json:"type,omitempty"` // e.g. "struct", "enum"
	Enumerators []Enumerator `json:"enumerators,omitempty"`
}

type Enumerator struct {
	Name  string `json:"name"`
	Value string
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
	Signature  string  `json:"signature"` // e.g. "sfVector2i sfRenderWindow_getPosition(sfRenderWindow*)"
}

type ArrayParamOverride struct {
	CFunc       string
	CParam      string
	CCountParam string
}

// StructOverride holds the Go‐side name of a vector typedef and its field names.
type StructOverride struct {
	GoName              string
	BaseType            string  // Go‐side base type name, e.g. "EventBase"
	Fields              []Field // Go‐side field names, e.g. "X", "Y", "Z", "W"
	CFields             []Field // C‐side field names, e.g. "x", "y", "z", "w"
	ArrayParamOverrides []ArrayParamOverride
}

type UnionMapper struct {
	CTypeField  Field    // C‐side type field name, e.g. "sfKeyEvent"
	CEnumValues []string // C‐side enum type name, e.g. "sfEvtClosed"
	GoName      string   // Go‐side name of the union typedef, e.g. "KeyEvent"
	EnumName    string   // Go‐side name of the enum type, e.g. "EvtClosed"
}

// UnionOverride holds the Go‐side name of a union typedef and its field names.
type UnionOverride struct {
	GoName     string
	GoBaseName string // Go‐side base type name, e.g. "BaseEvent"
	TypeField  Field
	CTypeField Field
	Mappers    []UnionMapper
}

type Struct struct {
	Name     string
	Fields   []Field
	BaseType string
}

type Interface struct {
	Name    string
	Methods []FunctionHeader
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

type Metadata struct {
	HeaderFiles []string `json:"header_files"` // List of header files used in the C code generation
}

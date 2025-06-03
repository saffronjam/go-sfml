package common

import (
	"encoding/json"
	"fmt"
	"github.com/golang-cz/textcase"
	"os"
	"regexp"
	"strings"
)

type Converter struct {
	RawTypes       []TypeDecl
	RawFunctions   []FunctionDecl
	RawEnumerators []Enumerator

	PrefixMap map[string]string

	RawTypesMap map[string]TypeDecl
	GoTypesMap  map[string]struct{}
	GoEnumsMap  map[string]struct{} // Map Go‐side enum names to struct{} for quick lookup

	StructOverrides  map[string]VectorInfo
	PointerOverrides map[string]struct{} // Map cTypes (as translated to GoTypes) that should remain pointers in Go

	SkippedTypes     map[string]struct{}
	SkippedFunctions map[string]struct{}
	SkipNameRegex    []string // Regex patterns to skip certain function names
}

// NewConverter initializes a Converter with the types from types.json.
func NewConverter(typesFile string, functionsFile string) (*Converter, error) {
	c := &Converter{
		StructOverrides: map[string]VectorInfo{
			"sfVector2i": {
				"Vector2i",
				[]Field{{Name: "X", Type: "int32"}, {Name: "Y", Type: "int32"}},
				[]Field{{Name: "x", Type: "int"}, {Name: "y", Type: "int"}},
			},
			"sfVector2f": {
				"Vector2f",
				[]Field{{Name: "X", Type: "float32"}, {Name: "Y", Type: "float32"}},
				[]Field{{Name: "x", Type: "float"}, {Name: "y", Type: "float"}}},
			"sfVector2u": {
				"Vector2u",
				[]Field{{Name: "X", Type: "uint32"}, {Name: "Y", Type: "uint32"}},
				[]Field{{Name: "x", Type: "sfUint32"}, {Name: "y", Type: "sfUint32"}}},
			"sfVector3f": {
				"Vector3f",
				[]Field{{Name: "X", Type: "float32"}, {Name: "Y", Type: "float32"}, {Name: "Z", Type: "float32"}},
				[]Field{{Name: "x", Type: "float"}, {Name: "y", Type: "float"}, {Name: "z", Type: "float"}}},
			"sfGlslIvec2": {
				"Vector2i",
				[]Field{{Name: "X", Type: "int32"}, {Name: "Y", Type: "int32"}},
				[]Field{{Name: "x", Type: "int"}, {Name: "y", Type: "int"}}},
			"sfGlslIvec3": {
				"Vector3i",
				[]Field{{Name: "X", Type: "int32"}, {Name: "Y", Type: "int32"}, {Name: "Z", Type: "int32"}},
				[]Field{{Name: "x", Type: "int"}, {Name: "y", Type: "int"}, {Name: "z", Type: "int"}}},
			"sfGlslIvec4": {
				"Vector4i",
				[]Field{{Name: "X", Type: "int32"}, {Name: "Y", Type: "int32"}, {Name: "Z", Type: "int32"}, {Name: "W", Type: "int32"}},
				[]Field{{Name: "x", Type: "int"}, {Name: "y", Type: "int"}, {Name: "z", Type: "int"}, {Name: "w", Type: "int"}}},
			"sfGlslBvec2": {
				"Vector2b",
				[]Field{{Name: "X", Type: "bool"}, {Name: "Y", Type: "bool"}},
				[]Field{{Name: "x", Type: "bool"}, {Name: "y", Type: "bool"}}},
			"sfGlslBvec3": {
				"Vector3b",
				[]Field{{Name: "X", Type: "bool"}, {Name: "Y", Type: "bool"}, {Name: "Z", Type: "bool"}},
				[]Field{{Name: "x", Type: "bool"}, {Name: "y", Type: "bool"}, {Name: "z", Type: "bool"}}},
			"sfGlslBvec4": {
				"Vector4b",
				[]Field{{Name: "X", Type: "bool"}, {Name: "Y", Type: "bool"}, {Name: "Z", Type: "bool"}, {Name: "W", Type: "bool"}},
				[]Field{{Name: "x", Type: "bool"}, {Name: "y", Type: "bool"}, {Name: "z", Type: "bool"}, {Name: "w", Type: "bool"}}},
			"sfGlslVec2": {
				"Vector2f",
				[]Field{{Name: "X", Type: "float32"}, {Name: "Y", Type: "float32"}},
				[]Field{{Name: "x", Type: "float"}, {Name: "y", Type: "float"}}},
			"sfGlslVec3": {
				"Vector3f",
				[]Field{{Name: "X", Type: "float32"}, {Name: "Y", Type: "float32"}, {Name: "Z", Type: "float32"}},
				[]Field{{Name: "x", Type: "float"}, {Name: "y", Type: "float"}, {Name: "z", Type: "float"}}},
			"sfGlslVec4": {
				"Vector4f",
				[]Field{{Name: "X", Type: "float32"}, {Name: "Y", Type: "float32"}, {Name: "Z", Type: "float32"}, {Name: "W", Type: "float32"}},
				[]Field{{Name: "x", Type: "float"}, {Name: "y", Type: "float"}, {Name: "z", Type: "float"}, {Name: "w", Type: "float"}}},
			"sfVideoMode": {
				"VideoMode",
				[]Field{{Name: "Width", Type: "uint32"}, {Name: "Height", Type: "uint32"}, {Name: "BitsPerPixel", Type: "uint32"}},
				[]Field{{Name: "width", Type: "unsigned int"}, {Name: "height", Type: "unsigned int"}, {Name: "bitsPerPixel", Type: "unsigned int"}},
			},
			"sfContextSettings": {
				"ContextSettings",
				[]Field{{Name: "DepthBits", Type: "uint32"}, {Name: "StencilBits", Type: "uint32"}, {Name: "AntialiasingLevel", Type: "uint32"}, {Name: "MajorVersion", Type: "uint32"}, {Name: "MinorVersion", Type: "uint32"}, {Name: "AttributeFlags", Type: "uint32"}, {Name: "SRgbCapable", Type: "bool"}},
				[]Field{{Name: "depthBits", Type: "unsigned int"}, {Name: "stencilBits", Type: "unsigned int"}, {Name: "antialiasingLevel", Type: "unsigned int"}, {Name: "majorVersion", Type: "unsigned int"}, {Name: "minorVersion", Type: "unsigned int"}, {Name: "attributeFlags", Type: "sfUint32"}, {Name: "sRgbCapable", Type: "sfBool"}},
			},
			"sfTime": {
				"Time",
				[]Field{{Name: "Microseconds", Type: "int64"}},
				[]Field{{Name: "microseconds", Type: "sfInt64"}},
			},
			"sfColor": {
				"Color",
				[]Field{{Name: "R", Type: "uint8"}, {Name: "G", Type: "uint8"}, {Name: "B", Type: "uint8"}, {Name: "A", Type: "uint8"}},
				[]Field{{Name: "r", Type: "sfUint8"}, {Name: "g", Type: "sfUint8"}, {Name: "b", Type: "sfUint8"}, {Name: "a", Type: "sfUint8"}},
			},
			"sfIntRect": {
				"IntRect",
				[]Field{{Name: "Left", Type: "int32"}, {Name: "Top", Type: "int32"}, {Name: "Width", Type: "int32"}, {Name: "Height", Type: "int32"}},
				[]Field{{Name: "left", Type: "sfInt32"}, {Name: "top", Type: "sfInt32"}, {Name: "width", Type: "sfInt32"}, {Name: "height", Type: "sfInt32"}},
			},
			"sfFloatRect": {
				"FloatRect",
				[]Field{{Name: "Left", Type: "float32"}, {Name: "Top", Type: "float32"}, {Name: "Width", Type: "float32"}, {Name: "Height", Type: "float32"}},
				[]Field{{Name: "left", Type: "float"}, {Name: "top", Type: "float"}, {Name: "width", Type: "float"}, {Name: "height", Type: "float"}},
			},

			"sfRenderStates": {
				"RenderStates",
				[]Field{{Name: "BlendMode", Type: "BlendMode"}, {Name: "Transform", Type: "Transform"}, {Name: "Texture", Type: "Texture"}, {Name: "Shader", Type: "Shader"}},
				[]Field{{Name: "blendMode", Type: "sfBlendMode"}, {Name: "transform", Type: "sfTransform"}, {Name: "texture", Type: "sfTexture"}, {Name: "shader", Type: "sfShader"}},
			},
			"sfBlendMode": {
				"BlendMode",
				[]Field{{Name: "ColorSrcFactor", Type: "BlendFactor"}, {Name: "ColorDstFactor", Type: "BlendFactor"}, {Name: "ColorEquation", Type: "BlendEquation"}, {Name: "AlphaSrcFactor", Type: "BlendFactor"}, {Name: "AlphaDstFactor", Type: "BlendFactor"}, {Name: "AlphaEquation", Type: "BlendEquation"}},
				[]Field{{Name: "colorSrcFactor", Type: "sfBlendFactor"}, {Name: "colorDstFactor", Type: "sfBlendFactor"}, {Name: "colorEquation", Type: "sfBlendEquation"}, {Name: "alphaSrcFactor", Type: "sfBlendFactor"}, {Name: "alphaDstFactor", Type: "sfBlendFactor"}, {Name: "alphaEquation", Type: "sfBlendEquation"}},
			},
			"sfGlyph": {
				"Glyph",
				[]Field{{Name: "Advance", Type: "float32"}, {Name: "Bounds", Type: "FloatRect"}, {Name: "TextureRect", Type: "IntRect"}},
				[]Field{{Name: "advance", Type: "float"}, {Name: "bounds", Type: "sfFloatRect"}, {Name: "textureRect", Type: "sfIntRect"}},
			},
			"sfFontInfo": {
				"FontInfo",
				[]Field{{Name: "Family", Type: "string"}},
				[]Field{{Name: "family", Type: "sfString"}},
			},
		},
		PointerOverrides: map[string]struct{}{
			"Transform": struct{}{}, // Keep Transform as a pointer in Go
		},
		PrefixMap: map[string]string{
			"sf": "",
		},
		RawTypesMap: make(map[string]TypeDecl),
		GoTypesMap:  make(map[string]struct{}),
		GoEnumsMap:  make(map[string]struct{}), // Map Go‐side enum names to struct{} for quick lookup
		// Skip native types that are not needed in Go.
		SkippedTypes:     map[string]struct{}{"sfWindowHandle": {}, "sfBool": {}, "sfChar32": {}, "sfUint8": {}, "sfUint16": {}, "sfUint32": {}, "sfUint64": {}, "sfInt8": {}, "sfInt16": {}, "sfInt32": {}, "sfInt64": {}},
		SkippedFunctions: map[string]struct{}{"sfMusic_setEffectProcessor": {}, "sfShape_create": {}, "sfSound_setEffectProcessor": {}},
		SkipNameRegex: []string{
			// Skip Sound Recorder completely
			"sfSoundRecorder*",
			"sfSoundStream*",
			"sfJoystick*",
			"sfVulkan*",
		},
	}

	var err error

	c.RawTypes, err = c.readTypes(typesFile)
	if err != nil {
		return nil, err
	}

	c.RawFunctions, err = c.readFunctions(functionsFile)
	if err != nil {
		return nil, err
	}

	for _, t := range c.RawTypes {
		c.RawTypesMap[t.Name] = t
	}

	for name, raw := range c.RawTypesMap {
		goName := textcase.PascalCase(c.StripPrefix(name))
		c.GoTypesMap[goName] = struct{}{}
		if raw.Type == "enum" {
			c.GoEnumsMap[goName] = struct{}{} // Store Go‐side enum names for quick lookup
		}
	}

	for _, vi := range c.StructOverrides {
		c.GoTypesMap[vi.GoName] = struct{}{}
	}

	return c, nil
}

func (c *Converter) readTypes(typesFile string) ([]TypeDecl, error) {
	typeData, err := os.ReadFile(typesFile)
	if err != nil {
		return nil, fmt.Errorf("Error reading %s: %v\n", typesFile, err)
	}

	var typeDecls []TypeDecl
	if err := json.Unmarshal(typeData, &typeDecls); err != nil {
		return nil, fmt.Errorf("Error parsing types.json: %v\n", err)
	}

	// Filter out skipped types
	for i := len(typeDecls) - 1; i >= 0; i-- {
		if _, ok := c.SkippedTypes[typeDecls[i].Name]; ok {
			typeDecls = append(typeDecls[:i], typeDecls[i+1:]...)
		}

		// Skip types with names matching any regex pattern
		for _, pattern := range c.SkipNameRegex {
			match, err := regexp.Match(pattern, []byte(typeDecls[i].Name))
			if err != nil {
				return nil, fmt.Errorf("Error matching regex %s: %v\n", pattern, err)
			}

			if match {
				typeDecls = append(typeDecls[:i], typeDecls[i+1:]...)
				break // No need to check other patterns for this type
			}
		}
	}

	return typeDecls, nil
}

func (c *Converter) readFunctions(functionsFile string) ([]FunctionDecl, error) {
	functionData, err := os.ReadFile(functionsFile)
	if err != nil {
		return nil, fmt.Errorf("Error reading functions.json: %v\n", err)
	}

	var functionDecls []FunctionDecl
	if err := json.Unmarshal(functionData, &functionDecls); err != nil {
		return nil, fmt.Errorf("Error parsing functions.json: %v\n", err)
	}

	// Filter out skipped functions
	for i := len(functionDecls) - 1; i >= 0; i-- {
		if _, ok := c.SkippedFunctions[functionDecls[i].Name]; ok {
			functionDecls = append(functionDecls[:i], functionDecls[i+1:]...)
		}

		// Skip functions with names matching any regex pattern
		for _, pattern := range c.SkipNameRegex {
			match, err := regexp.MatchString(pattern, functionDecls[i].Name)
			if err != nil {
				return nil, fmt.Errorf("Error matching regex %s: %v\n", pattern, err)
			}

			if match {
				functionDecls = append(functionDecls[:i], functionDecls[i+1:]...)
				break // No need to check other patterns for this function
			}
		}
	}

	return functionDecls, nil
}

// MapCParamToGoType maps a C parameter type (e.g. "const sfSprite*", "int", "sfVector2i")
// to a Go type string (e.g. "*Sprite", "int32", "Vector2i") using global knownTypes.
func (c *Converter) MapCParamToGoType(cType string) string {
	if cType == "const char *" {
		return "string" // Special case for C strings
	}

	// Strip "const ", "struct ", "*" from cType to get the base.
	base := strings.ReplaceAll(cType, "const ", "")
	base = strings.ReplaceAll(base, "struct ", "")
	ptr := ""
	if strings.Contains(cType, "*") {
		ptr = "*"
	}
	base = strings.ReplaceAll(base, "*", "")
	base = strings.TrimSpace(base)

	if structOverride, ok := c.StructOverrides[CleanCType(cType)]; ok {
		return ptr + structOverride.GoName
	}

	// If base is one of our known raw C types, convert to PascalCase and strip prefix.
	if _, ok := c.RawTypesMap[base]; ok {
		goName := textcase.PascalCase(c.StripPrefix(base))
		return ptr + goName
	}

	switch base {
	// SFML primitive types
	case "sfBool":
		return "bool"
	case "sfChar32":
		return "uint32"
	case "sfUint8":
		return "uint8"
	case "sfUint16":
		return "uint16"
	case "sfUint32":
		return "uint32"
	case "sfUint64":
		return "uint64"
	case "sfInt8":
		return "int8"
	case "sfInt16":
		return "int16"
	case "sfInt32":
		return "int32"
	case "sfInt64":
		return "int64"
	case "sfWindowHandle":
		return "uintptr" // Go's equivalent for window handles
	// C primitive types
	case "int":
		return "int32"
	case "float":
		return "float32"
	case "double":
		return "float64"
	case "unsigned int":
		return "uint32"
	case "char":
		return "byte"
	case "const char *":
		return "string"
	default:
		return "int32"
	}
}

// MapReturnType maps a C return type (e.g. "void", "sfVector2i", "int", "sfRenderWindow*")
// to the Go return type (e.g. "", "Vector2i", "int32", "*RenderWindow") using global knownTypes.
func (c *Converter) MapReturnType(cReturnType string) string {
	base := strings.ReplaceAll(cReturnType, "const ", "")
	base = strings.ReplaceAll(base, "struct ", "")
	ptr := ""
	if strings.Contains(cReturnType, "*") {
		ptr = "*"
	}
	base = strings.ReplaceAll(base, "*", "")
	base = strings.TrimSpace(base)
	goName := textcase.PascalCase(c.StripPrefix(base))
	if _, ok := c.PointerOverrides[goName]; ok {
		ptr = "*" // Ensure we return a pointer type
	}

	// If this is a known vector typedef, return its Go‐side name.
	if vi, ok := c.StructOverrides[base]; ok {
		return vi.GoName
	}

	// If base in knownTypes, map to PascalCase + pointer if needed.
	if _, ok := c.RawTypesMap[base]; ok {
		return ptr + goName
	}

	// Fallback for primitives.
	switch base {
	case "void":
		return "" // Go doesn't have a void return type
	case "float":
		return "float32"
	case "double":
		return "float64"
	case "int":
		return "int32"
	case "sfBool":
		return "bool"
	case "unsigned int":
		return "uint32"
	default:
		return "int32"
	}
}

// StripPrefix removes any known C‐style prefix (like "sf") from the given name.
func (c *Converter) StripPrefix(name string) string {
	for prefix, replacement := range c.PrefixMap {
		if strings.HasPrefix(name, prefix) {
			return replacement + name[len(prefix):]
		}
	}
	return name
}

// ParamCallExpr returns the expression to pass a Go parameter into the C call.
// If it’s a pointer to a known opaque struct, call .CPtr(), else pass directly.
func (c *Converter) ParamCallExpr(cParam Field, goParam Field) string {
	cleanCType := CleanCType(cParam.Type)

	if cParam.Type == "const char *" {
		return fmt.Sprintf("C.CString(%s)", goParam.Name)
	}

	if strings.HasPrefix(goParam.Type, "*") {
		if _, ok := c.GoTypesMap[cleanCType]; ok {
			return fmt.Sprintf("%s.CPtr()", goParam.Name)
		}
	} else {
		if _, ok := c.StructOverrides[cParam.Type]; ok {
			return fmt.Sprintf("%s.ToC()", goParam.Name)
		}
	}

	return goParam.Name
}

// IsKnownGoType checks if the provided type name is in knownGoTypes.
func (c *Converter) IsKnownGoType(name string) bool {
	_, ok := c.GoTypesMap[name]
	return ok
}

// IsEnum checks if the provided type name is a Go‐side enum.
func (c *Converter) IsEnum(name string) bool {
	_, ok := c.GoEnumsMap[name]
	return ok
}

// GetReceiverType determines if a function should be a method on a Go struct.
// If the C function name is "sfType_method" and the first parameter is "sfType*" or
// "const sfType*", it returns the Go‐side name of that type (e.g. "RenderWindow").
// Otherwise, it returns an empty string.
func (c *Converter) GetReceiverType(fnName, firstParamType string) string {
	parts := strings.SplitN(fnName, "_", 2)
	if len(parts) < 1 {
		return ""
	}
	base := parts[0]                // e.g. "sfRenderWindow"
	expected := c.StripPrefix(base) // e.g. "RenderWindow"
	expectedGo := textcase.PascalCase(expected)

	normalized := strings.ReplaceAll(firstParamType, "const ", "")
	normalized = strings.ReplaceAll(normalized, "struct ", "")
	normalized = strings.ReplaceAll(normalized, "*", "")
	normalized = strings.TrimSpace(normalized)

	if normalized == "sf"+expected {
		if _, ok := c.RawTypesMap["sf"+expected]; ok {
			return expectedGo
		}
	}
	return ""
}

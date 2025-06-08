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

	StructOverrides        map[string]StructOverride
	PhantomStructOverrides []StructOverride // Map C struct names that should be overridden with a Go struct, but does not exist in SFML.
	UnionOverrides         map[string]UnionOverride
	ReturnParamOverrides   map[string]Field // Map C param names that should be moved to a Go return value (possibly creating a multi-return function).

	StoreAsValue map[string]struct{} // Map cTypes (as translated to GoTypes) that should be stored as values, not pointers, like sfTransform.

	SkippedTypes     map[string]struct{}
	SkippedFunctions map[string]struct{}
	SkipNameRegex    []string // Regex patterns to skip certain function names
}

// NewConverter initializes a Converter with the types from types.json.
func NewConverter(typesFile string, functionsFile string) (*Converter, error) {
	c := &Converter{
		StructOverrides: map[string]StructOverride{
			"sfVector2i": {

				GoName:  "Vector2i",
				Fields:  []Field{{Name: "X", Type: "int32"}, {Name: "Y", Type: "int32"}},
				CFields: []Field{{Name: "x", Type: "int"}, {Name: "y", Type: "int"}},
			},
			"sfVector2f": {
				GoName:  "Vector2f",
				Fields:  []Field{{Name: "X", Type: "float32"}, {Name: "Y", Type: "float32"}},
				CFields: []Field{{Name: "x", Type: "float"}, {Name: "y", Type: "float"}},
			},
			"sfVector2u": {
				GoName:  "Vector2u",
				Fields:  []Field{{Name: "X", Type: "uint32"}, {Name: "Y", Type: "uint32"}},
				CFields: []Field{{Name: "x", Type: "sfUint32"}, {Name: "y", Type: "sfUint32"}},
			},
			"sfVector3f": {
				GoName:  "Vector3f",
				Fields:  []Field{{Name: "X", Type: "float32"}, {Name: "Y", Type: "float32"}, {Name: "Z", Type: "float32"}},
				CFields: []Field{{Name: "x", Type: "float"}, {Name: "y", Type: "float"}, {Name: "z", Type: "float"}},
			},
			"sfGlslIvec2": {
				GoName:  "Vector2i",
				Fields:  []Field{{Name: "X", Type: "int32"}, {Name: "Y", Type: "int32"}},
				CFields: []Field{{Name: "x", Type: "int"}, {Name: "y", Type: "int"}},
			},
			"sfGlslIvec3": {
				GoName:  "Vector3i",
				Fields:  []Field{{Name: "X", Type: "int32"}, {Name: "Y", Type: "int32"}, {Name: "Z", Type: "int32"}},
				CFields: []Field{{Name: "x", Type: "int"}, {Name: "y", Type: "int"}, {Name: "z", Type: "int"}},
			},
			"sfGlslIvec4": {
				GoName:  "Vector4i",
				Fields:  []Field{{Name: "X", Type: "int32"}, {Name: "Y", Type: "int32"}, {Name: "Z", Type: "int32"}, {Name: "W", Type: "int32"}},
				CFields: []Field{{Name: "x", Type: "int"}, {Name: "y", Type: "int"}, {Name: "z", Type: "int"}, {Name: "w", Type: "int"}},
			},
			"sfGlslBvec2": {
				GoName:  "Vector2b",
				Fields:  []Field{{Name: "X", Type: "bool"}, {Name: "Y", Type: "bool"}},
				CFields: []Field{{Name: "x", Type: "sfBool"}, {Name: "y", Type: "sfBool"}},
			},
			"sfGlslBvec3": {
				GoName:  "Vector3b",
				Fields:  []Field{{Name: "X", Type: "bool"}, {Name: "Y", Type: "bool"}, {Name: "Z", Type: "bool"}},
				CFields: []Field{{Name: "x", Type: "sfBool"}, {Name: "y", Type: "sfBool"}, {Name: "z", Type: "sfBool"}},
			},
			"sfGlslBvec4": {
				GoName:  "Vector4b",
				Fields:  []Field{{Name: "X", Type: "bool"}, {Name: "Y", Type: "bool"}, {Name: "Z", Type: "bool"}, {Name: "W", Type: "bool"}},
				CFields: []Field{{Name: "x", Type: "sfBool"}, {Name: "y", Type: "sfBool"}, {Name: "z", Type: "sfBool"}, {Name: "w", Type: "sfBool"}},
			},
			"sfGlslVec2": {
				GoName:  "Vector2f",
				Fields:  []Field{{Name: "X", Type: "float32"}, {Name: "Y", Type: "float32"}},
				CFields: []Field{{Name: "x", Type: "float"}, {Name: "y", Type: "float"}},
			},
			"sfGlslVec3": {
				GoName:  "Vector3f",
				Fields:  []Field{{Name: "X", Type: "float32"}, {Name: "Y", Type: "float32"}, {Name: "Z", Type: "float32"}},
				CFields: []Field{{Name: "x", Type: "float"}, {Name: "y", Type: "float"}, {Name: "z", Type: "float"}},
			},
			"sfGlslVec4": {
				GoName:  "Vector4f",
				Fields:  []Field{{Name: "X", Type: "float32"}, {Name: "Y", Type: "float32"}, {Name: "Z", Type: "float32"}, {Name: "W", Type: "float32"}},
				CFields: []Field{{Name: "x", Type: "float"}, {Name: "y", Type: "float"}, {Name: "z", Type: "float"}, {Name: "w", Type: "float"}},
			},
			"sfVideoMode": {
				GoName:  "VideoMode",
				Fields:  []Field{{Name: "Width", Type: "uint32"}, {Name: "Height", Type: "uint32"}, {Name: "BitsPerPixel", Type: "uint32"}},
				CFields: []Field{{Name: "width", Type: "sfUint32"}, {Name: "height", Type: "sfUint32"}, {Name: "bitsPerPixel", Type: "sfUint32"}},
			},
			"sfContextSettings": {
				GoName:  "ContextSettings",
				Fields:  []Field{{Name: "DepthBits", Type: "uint32"}, {Name: "StencilBits", Type: "uint32"}, {Name: "AntialiasingLevel", Type: "uint32"}, {Name: "MajorVersion", Type: "uint32"}, {Name: "MinorVersion", Type: "uint32"}, {Name: "AttributeFlags", Type: "uint32"}, {Name: "SRgbCapable", Type: "bool"}},
				CFields: []Field{{Name: "depthBits", Type: "sfUint32"}, {Name: "stencilBits", Type: "sfUint32"}, {Name: "antialiasingLevel", Type: "sfUint32"}, {Name: "majorVersion", Type: "sfUint32"}, {Name: "minorVersion", Type: "sfUint32"}, {Name: "attributeFlags", Type: "sfUint32"}, {Name: "sRgbCapable", Type: "sfBool"}},
			},
			"sfTime": {
				GoName:  "Time",
				Fields:  []Field{{Name: "Microseconds", Type: "int64"}},
				CFields: []Field{{Name: "microseconds", Type: "sfInt64"}},
			},
			"sfColor": {
				GoName:  "Color",
				Fields:  []Field{{Name: "R", Type: "uint8"}, {Name: "G", Type: "uint8"}, {Name: "B", Type: "uint8"}, {Name: "A", Type: "uint8"}},
				CFields: []Field{{Name: "r", Type: "sfUint8"}, {Name: "g", Type: "sfUint8"}, {Name: "b", Type: "sfUint8"}, {Name: "a", Type: "sfUint8"}},
			},
			"sfIntRect": {
				GoName:  "IntRect",
				Fields:  []Field{{Name: "Left", Type: "int32"}, {Name: "Top", Type: "int32"}, {Name: "Width", Type: "int32"}, {Name: "Height", Type: "int32"}},
				CFields: []Field{{Name: "left", Type: "sfInt32"}, {Name: "top", Type: "sfInt32"}, {Name: "width", Type: "sfInt32"}, {Name: "height", Type: "sfInt32"}},
			},
			"sfFloatRect": {
				GoName:  "FloatRect",
				Fields:  []Field{{Name: "Left", Type: "float32"}, {Name: "Top", Type: "float32"}, {Name: "Width", Type: "float32"}, {Name: "Height", Type: "float32"}},
				CFields: []Field{{Name: "left", Type: "float"}, {Name: "top", Type: "float"}, {Name: "width", Type: "float"}, {Name: "height", Type: "float"}},
			},

			"sfRenderStates": {
				GoName:  "RenderStates",
				Fields:  []Field{{Name: "BlendMode", Type: "BlendMode"}, {Name: "Transform", Type: "Transform"}, {Name: "Texture", Type: "Texture"}, {Name: "Shader", Type: "Shader"}},
				CFields: []Field{{Name: "blendMode", Type: "sfBlendMode"}, {Name: "transform", Type: "sfTransform"}, {Name: "texture", Type: "sfTexture"}, {Name: "shader", Type: "sfShader"}},
			},
			"sfBlendMode": {
				GoName:  "BlendMode",
				Fields:  []Field{{Name: "ColorSrcFactor", Type: "BlendFactor"}, {Name: "ColorDstFactor", Type: "BlendFactor"}, {Name: "ColorEquation", Type: "BlendEquation"}, {Name: "AlphaSrcFactor", Type: "BlendFactor"}, {Name: "AlphaDstFactor", Type: "BlendFactor"}, {Name: "AlphaEquation", Type: "BlendEquation"}},
				CFields: []Field{{Name: "colorSrcFactor", Type: "sfBlendFactor"}, {Name: "colorDstFactor", Type: "sfBlendFactor"}, {Name: "colorEquation", Type: "sfBlendEquation"}, {Name: "alphaSrcFactor", Type: "sfBlendFactor"}, {Name: "alphaDstFactor", Type: "sfBlendFactor"}, {Name: "alphaEquation", Type: "sfBlendEquation"}},
			},
			"sfGlyph": {
				GoName:  "Glyph",
				Fields:  []Field{{Name: "Advance", Type: "float32"}, {Name: "Bounds", Type: "FloatRect"}, {Name: "TextureRect", Type: "IntRect"}},
				CFields: []Field{{Name: "advance", Type: "float"}, {Name: "bounds", Type: "sfFloatRect"}, {Name: "textureRect", Type: "sfIntRect"}},
			},
			"sfFontInfo": {
				GoName:  "FontInfo",
				Fields:  []Field{{Name: "Family", Type: "string"}},
				CFields: []Field{{Name: "family", Type: "sfString"}},
			},
			"sfVertex": {
				GoName:  "Vertex",
				Fields:  []Field{{Name: "Position", Type: "Vector2f"}, {Name: "Color", Type: "Color"}, {Name: "TexCoords", Type: "Vector2f"}},
				CFields: []Field{{Name: "position", Type: "sfVector2f"}, {Name: "color", Type: "sfColor"}, {Name: "texCoords", Type: "sfVector2f"}},
				ArrayParamOverrides: []ArrayParamOverride{
					{
						CFunc:       "sfVertexBuffer_update",
						CParam:      "vertices",
						CCountParam: "vertexCount",
					},
				},
			},
			// data events
			"sfKeyEvent": {
				GoName:   "KeyEvent",
				BaseType: "BaseEvent",
				Fields:   []Field{{Name: "Type", Type: "EventType"}, {Name: "Code", Type: "KeyCode"}, {Name: "Scancode", Type: "Scancode"}, {Name: "Alt", Type: "bool"}, {Name: "Control", Type: "bool"}, {Name: "Shift", Type: "bool"}, {Name: "System", Type: "bool"}},
				CFields:  []Field{{Name: "type", Type: "sfEventType"}, {Name: "code", Type: "sfKeyCode"}, {Name: "scancode", Type: "sfScancode"}, {Name: "alt", Type: "sfBool"}, {Name: "control", Type: "sfBool"}, {Name: "shift", Type: "sfBool"}, {Name: "system", Type: "sfBool"}},
			},
			"sfTextEvent": {
				GoName:   "TextEvent",
				BaseType: "BaseEvent",
				Fields:   []Field{{Name: "Type", Type: "EventType"}, {Name: "Unicode", Type: "uint32"}},
				CFields:  []Field{{Name: "type", Type: "sfEventType"}, {Name: "unicode", Type: "sfUint32"}},
			},
			"sfMouseMoveEvent": {
				GoName:   "MouseMoveEvent",
				BaseType: "BaseEvent",
				Fields:   []Field{{Name: "Type", Type: "EventType"}, {Name: "X", Type: "int32"}, {Name: "Y", Type: "int32"}},
				CFields:  []Field{{Name: "type", Type: "sfEventType"}, {Name: "x", Type: "int"}, {Name: "y", Type: "int"}},
			},
			"sfMouseButtonEvent": {
				GoName:   "MouseButtonEvent",
				BaseType: "BaseEvent",
				Fields:   []Field{{Name: "Type", Type: "EventType"}, {Name: "Button", Type: "MouseButton"}, {Name: "X", Type: "int32"}, {Name: "Y", Type: "int32"}},
				CFields:  []Field{{Name: "type", Type: "sfEventType"}, {Name: "button", Type: "sfMouseButton"}, {Name: "x", Type: "int"}, {Name: "y", Type: "int"}},
			},
			"sfMouseWheelEvent": {
				GoName:   "MouseWheelEvent",
				BaseType: "BaseEvent",
				Fields:   []Field{{Name: "Type", Type: "EventType"}, {Name: "Delta", Type: "int32"}, {Name: "X", Type: "int32"}, {Name: "Y", Type: "int32"}},
				CFields:  []Field{{Name: "type", Type: "sfEventType"}, {Name: "delta", Type: "int"}, {Name: "x", Type: "int"}, {Name: "y", Type: "int"}},
			},
			"sfMouseWheelScrollEvent": {
				GoName:   "MouseWheelScrollEvent",
				BaseType: "BaseEvent",
				Fields:   []Field{{Name: "Type", Type: "EventType"}, {Name: "Wheel", Type: "MouseWheel"}, {Name: "Delta", Type: "float32"}, {Name: "X", Type: "int32"}, {Name: "Y", Type: "int32"}},
				CFields:  []Field{{Name: "type", Type: "sfEventType"}, {Name: "wheel", Type: "sfMouseWheel"}, {Name: "delta", Type: "float"}, {Name: "x", Type: "int"}, {Name: "y", Type: "int"}},
			},
			"sfSizeEvent": {
				GoName:   "SizeEvent",
				BaseType: "BaseEvent",
				Fields:   []Field{{Name: "Type", Type: "EventType"}, {Name: "Width", Type: "uint32"}, {Name: "Height", Type: "uint32"}},
				CFields:  []Field{{Name: "type", Type: "sfEventType"}, {Name: "width", Type: "unsigned int"}, {Name: "height", Type: "unsigned int"}},
			},
			"sfTouchEvent": {
				GoName:   "TouchEvent",
				BaseType: "BaseEvent",
				Fields:   []Field{{Name: "Type", Type: "EventType"}, {Name: "Finger", Type: "uint32"}, {Name: "X", Type: "int32"}, {Name: "Y", Type: "int32"}},
				CFields:  []Field{{Name: "type", Type: "sfEventType"}, {Name: "finger", Type: "unsigned int"}, {Name: "x", Type: "int"}, {Name: "y", Type: "int"}},
			},
			"sfSensorEvent": {
				GoName:   "SensorEvent",
				BaseType: "BaseEvent",
				Fields:   []Field{{Name: "Type", Type: "EventType"}, {Name: "SensorType", Type: "SensorType"}, {Name: "X", Type: "float32"}, {Name: "Y", Type: "float32"}, {Name: "Z", Type: "float32"}},
				CFields:  []Field{{Name: "type", Type: "sfEventType"}, {Name: "sensorType", Type: "sfSensorType"}, {Name: "x", Type: "float"}, {Name: "y", Type: "float"}, {Name: "z", Type: "float"}},
			},
			// Parent type for all events
			"sfEvent": {
				GoName: "Event",
				Fields: []Field{},
			},
		},
		PhantomStructOverrides: []StructOverride{
			// No data events
			{
				GoName:   "ClosedEvent",
				BaseType: "BaseEvent",
				Fields:   []Field{{Name: "Type", Type: "EventType"}},
				CFields:  []Field{{Name: "type", Type: "sfEventType"}},
			}, {
				GoName:   "LostFocusEvent",
				BaseType: "BaseEvent",
				Fields:   []Field{{Name: "Type", Type: "EventType"}},
				CFields:  []Field{{Name: "type", Type: "sfEventType"}},
			}, {
				GoName:   "GainedFocusEvent",
				BaseType: "BaseEvent",
				Fields:   []Field{{Name: "Type", Type: "EventType"}},
				CFields:  []Field{{Name: "type", Type: "sfEventType"}},
			}, {
				GoName:   "MouseEnteredEvent",
				BaseType: "BaseEvent",
				Fields:   []Field{{Name: "Type", Type: "EventType"}},
				CFields:  []Field{{Name: "type", Type: "sfEventType"}},
			}, {
				GoName:   "MouseLeftEvent",
				BaseType: "BaseEvent",
				Fields:   []Field{{Name: "Type", Type: "EventType"}},
				CFields:  []Field{{Name: "type", Type: "sfEventType"}},
			},
		},
		UnionOverrides: map[string]UnionOverride{
			"sfEvent": {
				GoName:     "Event",
				GoBaseName: "BaseEvent",
				TypeField:  Field{Name: "Type", Type: "EventType"},
				CTypeField: Field{Name: "type", Type: "sfEventType"},
				// typedef enum
				//{
				//    sfEvtClosed,                 ///< The window requested to be closed (no data)
				//    sfEvtResized,                ///< The window was resized (data in event.size)
				//    sfEvtLostFocus,              ///< The window lost the focus (no data)
				//    sfEvtGainedFocus,            ///< The window gained the focus (no data)
				//    sfEvtTextEntered,            ///< A character was entered (data in event.text)
				//    sfEvtKeyPressed,             ///< A key was pressed (data in event.key)
				//    sfEvtKeyReleased,            ///< A key was released (data in event.key)
				//    sfEvtMouseWheelMoved,        ///< The mouse wheel was scrolled (data in event.mouseWheel) (deprecated)
				//    sfEvtMouseWheelScrolled,     ///< The mouse wheel was scrolled (data in event.mouseWheelScroll)
				//    sfEvtMouseButtonPressed,     ///< A mouse button was pressed (data in event.mouseButton)
				//    sfEvtMouseButtonReleased,    ///< A mouse button was released (data in event.mouseButton)
				//    sfEvtMouseMoved,             ///< The mouse cursor moved (data in event.mouseMove)
				//    sfEvtMouseEntered,           ///< The mouse cursor entered the area of the window (no data)
				//    sfEvtMouseLeft,              ///< The mouse cursor left the area of the window (no data)
				//    sfEvtJoystickButtonPressed,  ///< A joystick button was pressed (data in event.joystickButton)
				//    sfEvtJoystickButtonReleased, ///< A joystick button was released (data in event.joystickButton)
				//    sfEvtJoystickMoved,          ///< The joystick moved along an axis (data in event.joystickMove)
				//    sfEvtJoystickConnected,      ///< A joystick was connected (data in event.joystickConnect)
				//    sfEvtJoystickDisconnected,   ///< A joystick was disconnected (data in event.joystickConnect)
				//    sfEvtTouchBegan,             ///< A touch event began (data in event.touch)
				//    sfEvtTouchMoved,             ///< A touch moved (data in event.touch)
				//    sfEvtTouchEnded,             ///< A touch event ended (data in event.touch)
				//    sfEvtSensorChanged,          ///< A sensor value changed (data in event.sensor)
				//
				//    sfEvtCount                   ///< Keep last -- the total number of event types
				//} sfEventType;
				Mappers: []UnionMapper{
					// No data events
					{GoName: "ClosedEvent", CEnumValues: []string{"sfEvtClosed"}},
					{GoName: "LostFocusEvent", CEnumValues: []string{"sfEvtLostFocus"}},
					{GoName: "GainedFocusEvent", CEnumValues: []string{"sfEvtGainedFocus"}},
					{GoName: "MouseEnteredEvent", CEnumValues: []string{"sfEvtMouseEntered"}},
					{GoName: "MouseLeftEvent", CEnumValues: []string{"sfEvtMouseLeft"}},

					// Data events
					{CTypeField: Field{Name: "size", Type: "sfSizeEvent"}, GoName: "SizeEvent", CEnumValues: []string{"sfEvtResized"}},
					{CTypeField: Field{Name: "key", Type: "sfKeyEvent"}, GoName: "KeyEvent", CEnumValues: []string{"sfEvtKeyPressed", "sfEvtKeyReleased"}},
					{CTypeField: Field{Name: "text", Type: "sfTextEvent"}, GoName: "TextEvent", CEnumValues: []string{"sfEvtTextEntered"}},
					{CTypeField: Field{Name: "mouseMove", Type: "sfMouseMoveEvent"}, GoName: "MouseMoveEvent", CEnumValues: []string{"sfEvtMouseMoved"}},
					{CTypeField: Field{Name: "mouseButton", Type: "sfMouseButtonEvent"}, GoName: "MouseButtonEvent", CEnumValues: []string{"sfEvtMouseButtonPressed", "sfEvtMouseButtonReleased"}},
					{CTypeField: Field{Name: "mouseWheelScroll", Type: "sfMouseWheelScrollEvent"}, GoName: "MouseWheelScrollEvent", CEnumValues: []string{"sfEvtMouseWheelScrolled"}},
					{CTypeField: Field{Name: "touch", Type: "sfTouchEvent"}, GoName: "TouchEvent", CEnumValues: []string{"sfEvtTouchBegan", "sfEvtTouchMoved", "sfEvtTouchEnded"}},
					{CTypeField: Field{Name: "sensor", Type: "sfSensorEvent"}, GoName: "SensorEvent", CEnumValues: []string{"sfEvtSensorChanged"}},
				},
			},
		},
		ReturnParamOverrides: map[string]Field{
			"sfRenderWindow_pollEvent": {
				Name: "event",
				Type: "sfEvent",
			},
			"sfRenderWindow_waitEvent": {
				Name: "event",
				Type: "sfEvent",
			},
			"sfWindowBase_pollEvent": {
				Name: "event",
				Type: "sfEvent",
			},
			"sfWindowBase_waitEvent": {
				Name: "event",
				Type: "sfEvent",
			},
			"sfWindow_pollEvent": {
				Name: "event",
				Type: "sfEvent",
			},
			"sfWindow_waitEvent": {
				Name: "event",
				Type: "sfEvent",
			},
			"sfIntRect_intersects": {
				Name: "intersection",
				Type: "sfIntRect",
			},
			"sfFloatRect_intersects": {
				Name: "intersection",
				Type: "sfFloatRect",
			},
		},
		StoreAsValue: map[string]struct{}{
			"sfTransform": {}, // Keep Transform as a pointer in Go
		},
		PrefixMap: map[string]string{
			"sf": "",
		},
		RawTypesMap: make(map[string]TypeDecl),
		GoTypesMap:  make(map[string]struct{}),
		GoEnumsMap:  make(map[string]struct{}), // Map Go‐side enum names to struct{} for quick lookup
		// Skip native types that are not needed in Go.
		SkippedTypes:     map[string]struct{}{"sfWindowHandle": {}, "sfBool": {}, "sfChar32": {}, "sfUint8": {}, "sfUint16": {}, "sfUint32": {}, "sfUint64": {}, "sfInt8": {}, "sfInt16": {}, "sfInt32": {}, "sfInt64": {}},
		SkippedFunctions: map[string]struct{}{"sfShape_create": {}, "sfContext_getFunction": {}, "sfVideoMode_getFullscreenModes": {}, "sfVertexArray_getVertex": {}},
		SkipNameRegex: []string{
			"sfJoystick*",
			"sfVulkan*",
			"sfThread*",
			".*_createVulkanSurface",
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

// MapCToGoType maps a C parameter type (e.g. "const sfSprite*", "int", "sfVector2i")
// to a Go type string (e.g. "*Sprite", "int32", "Vector2i") using global knownTypes.
func (c *Converter) MapCToGoType(cType string) string {
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

	if base == "void" {
		if ptr != "" {
			return "uintptr"
		} else {
			return ""
		}
	}

	fallbackType := func() string {

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
		case "size_t":
			return "uint64"
		case "int":
			return "int32"
		case "float":
			return "float32"
		case "double":
			return "float64"
		case "sfUint":
			return "uint32"
		case "char":
			return "byte"
		case "const char *":
			return "string"
		default:
			return "int32"
		}
	}()

	return ptr + fallbackType // Return the mapped type with pointer if applicable
}

func (c *Converter) TranslateMethodName(cMethodName string) string {
	pascalCase := textcase.PascalCase(cMethodName)

	// If contains "Create" and has something before it, prepend "New" and remove "Create".
	if strings.Contains(pascalCase, "Create") {
		withoutCreate := strings.ReplaceAll(pascalCase, "Create", "")
		return "New" + withoutCreate // Prepend "New" and remove "Create"
	}

	if strings.HasSuffix(pascalCase, "Destroy") {
		return "Free" + pascalCase[:len(pascalCase)-7] // Remove "Destroy" and prepend "Free"
	}

	// If starts with "Get" (and is followed by uppercase), remove "Get" prefix. With some exceptions.
	exceptions := []string{"GetScale"}
	if strings.HasPrefix(pascalCase, "Get") && len(pascalCase) > 3 && (pascalCase[3] >= 'A' && pascalCase[3] <= 'Z') {
		for _, ex := range exceptions {
			if pascalCase == ex {
				return pascalCase // Keep the exception as is
			}
		}
		return pascalCase[3:] // Remove "Get" prefix
	}

	return pascalCase // Return as is if no special cases matched
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
	_, ok := c.GoTypesMap[StripPointer(name)]
	return ok
}

// IsEnum checks if the provided type name is a Go‐side enum.
func (c *Converter) IsEnum(name string) bool {
	_, ok := c.GoEnumsMap[StripPointer(name)]
	return ok
}

// GetOverriddenType checks if a Go type has a struct override and returns it.
func (c *Converter) GetOverriddenType(goType string) (string, *StructOverride) {
	for cName, vi := range c.StructOverrides {
		if vi.GoName == goType {
			return cName, &vi
		}
	}

	return "", nil
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

// IsSliceParam checks if a parameter is a slice parameter
// and returns the corresponding ArrayParamOverride if it exists.
func (c *Converter) IsSliceParam(cFunc string, cParamName string) *ArrayParamOverride {
	for _, so := range c.StructOverrides {
		for _, override := range so.ArrayParamOverrides {
			if override.CFunc == cFunc && override.CParam == cParamName {
				return &override
			}
		}
	}
	return nil
}

// IsSliceCountParam checks if a parameter is a slice count parameter
// and returns the corresponding ArrayParamOverride if it exists.
func (c *Converter) IsSliceCountParam(cFunc string, cParamName string) *ArrayParamOverride {
	for _, so := range c.StructOverrides {
		for _, override := range so.ArrayParamOverrides {
			if override.CFunc == cFunc && override.CCountParam == cParamName {
				return &override
			}
		}
	}
	return nil
}

// IsReturnParam checks if a parameter is a return type
// that should be returned as a Go value instead of a pointer.
func (c *Converter) IsReturnParam(cFunc string, cParamName string) *Field {
	if field, ok := c.ReturnParamOverrides[cFunc]; ok && field.Name == cParamName {
		return &field
	}
	return nil
}

// GetUnionType checks if a Go type is a union type and returns the corresponding UnionOverride.
func (c *Converter) GetUnionType(goType string) (string, *UnionOverride) {
	for cName, uo := range c.UnionOverrides {
		if uo.GoName == goType {
			return cName, &uo
		}
	}
	return "", nil
}

// IsPhantomStruct checks if a Go type is a phantom struct.
func (c *Converter) IsPhantomStruct(goType string) *StructOverride {
	for _, pso := range c.PhantomStructOverrides {
		if pso.GoName == goType {
			return &pso
		}
	}
	return nil
}

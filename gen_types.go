package main

import (
	"fmt"
	"github.com/golang-cz/textcase"
	"github.com/saffronjam/go-sfml/internal/common"
	"log"
	"path"
	"strings"
)

func main() {
	config, err := common.LoadConfig()
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	converter, err := common.NewConverter(common.TypesFile, common.FunctionsFile)
	if err != nil {
		panic(err)
	}

	writer, err := common.NewWriter(config.GithubRepo, converter)
	if err != nil {
		panic(err)
	}

	writer.HeaderTypes()

	for _, t := range converter.RawTypes {
		if t.Type == "struct" {

			// Name of the type, e.g. "sfMouseButton", "sfCursorType"
			rawName := t.Name

			// Only process names starting with "sf"
			if rawName == "" || !strings.HasPrefix(rawName, "sf") {
				continue
			}

			baseName := converter.StripPrefix(rawName)
			goName := textcase.PascalCase(baseName)

			// Vector override case: value‐type vectors
			if structOverride, ok := converter.StructOverrides[rawName]; ok {
				writer.Struct(common.Struct{
					Name:   structOverride.GoName,
					Fields: structOverride.Fields,
				})

				receiverName := strings.ToLower(structOverride.GoName[:1]) // e.g. "v" for "Vector2i"
				writer.ReceiverFunctionHeader(common.ReceiverFunctionHeader{
					ReceiverName: receiverName, // e.g. "v" for "Vector2i"
					ReceiverType: structOverride.GoName,
					MethodName:   "ToC",
					Parameters:   []common.Field{},
					ReturnType:   fmt.Sprintf("C.%s", rawName),
				})
				returnValue := strings.Builder{}
				returnValue.WriteString(fmt.Sprintf("C.%s{ ", rawName))
				for i, field := range structOverride.Fields {
					if i > 0 {
						returnValue.WriteString(", ")
					}
					returnValue.WriteString(fmt.Sprintf("%s: %s.%s", structOverride.CFields[i].Name, receiverName, field.Name))
				}
				returnValue.WriteString(" }")

				writer.ReturnValue(returnValue.String())

				//if len(structOverride.Fields) == 2 {
				//	writer.ReturnValue(fmt.Sprintf("C.%s{ x: C.%s(v.X), y: C.%s(v.Y) }", rawName, structOverride.Fields[0].Type, structOverride.Fields[1].Type))
				//} else if len(structOverride.Fields) == 3 {
				//	writer.ReturnValue(fmt.Sprintf("C.%s{ x: C.%s(v.X), y: C.%s(v.Y), z: C.%s(v.Z) }", rawName, structOverride.Fields[0].Type, structOverride.Fields[1].Type, structOverride.Fields[2].Type))
				//} else {
				//	writer.ReturnValue(fmt.Sprintf("C.%s{ x: C.%s(v.X), y: C.%s(v.Y), z: C.%s(v.Z), w: C.%s(v.W) }", rawName, structOverride.Fields[0].Type, structOverride.Fields[1].Type, structOverride.Fields[2].Type, structOverride.Fields[3].Type))
				//}

				continue
			}

			// Non‐vector case: treat as opaque struct with structOverride
			// Generate struct:
			writer.Struct(common.Struct{
				Name: goName,
				Fields: append([]common.Field{
					{Name: "ptr", Type: "unsafe.Pointer"},
				}),
			})

			// Generate CPtr() method:
			// func (s *GoName) CPtr() unsafe.Pointer { return s.ptr }
			receiverName := strings.ToLower(goName[:1]) // e.g. "s" for "Sprite"
			writer.ReceiverFunctionHeader(common.ReceiverFunctionHeader{
				ReceiverName: receiverName,
				ReceiverType: fmt.Sprintf("*%s", goName),
				MethodName:   "CPtr",
				Parameters:   []common.Field{},
				ReturnType:   "unsafe.Pointer",
			})
			writer.ReturnValue(fmt.Sprintf("(*C.%s)(%s.ptr)", rawName, receiverName))
		} else if t.Type == "enum" {
			// Name of the type, e.g. "sfMouseButton", "sfCursorType"
			rawName := t.Name

			// Only process names starting with "sf"
			if rawName == "" || !strings.HasPrefix(rawName, "sf") {
				continue
			}

			baseName := converter.StripPrefix(rawName)
			goName := textcase.PascalCase(baseName)

			enumerators := make([]common.Enumerator, len(t.Enumerators))
			for i, enumerator := range t.Enumerators {
				// Convert enumerator name to Go style
				enumerators[i] = common.Enumerator{
					Name: textcase.PascalCase(converter.StripPrefix(enumerator.Name)),
				}
			}

			// Generate enum type:
			writer.Enum(common.Enum{
				Name:        goName,
				Enumerators: enumerators,
			})
		}
	}

	err = writer.WriteToFile(path.Join(common.OutputDir, "/go_types.go"))
	if err != nil {
		log.Fatalf("Failed to write to file: %v", err)
	}
}

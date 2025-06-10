package main

import "C"
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

	writer, err := common.NewWriter(config.GithubRepo, converter, common.MetadataFile)
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
			isPointer := true
			if _, ok := converter.StoreAsValue[rawName]; ok {
				isPointer = false
			}

			if structOverride, ok := converter.StructOverrides[rawName]; ok {
				// Override behavior if union
				if unionOverride, isUnion := converter.UnionOverrides[rawName]; isUnion {
					writer.Interface(common.Interface{
						Name: unionOverride.GoName,
						Methods: []common.FunctionHeader{
							{
								MethodName: "EventType",
								Parameters: []common.Field{},
								ReturnType: unionOverride.TypeField.Type, // e.g. "EvtType"
							},
							{
								MethodName: "BaseToC",
								Parameters: []common.Field{},
								ReturnType: fmt.Sprintf("C.%s", rawName), // e.g. "C.sfEvent" (not "C.sfKeyEvent")
							},
						},
					})

					// Create NewFromC function for the union
					writer.FunctionHeader(common.FunctionHeader{
						MethodName: "New" + unionOverride.GoName + "FromC",
						Parameters: []common.Field{
							{
								Name: "cObj",
								Type: fmt.Sprintf("C.%s", rawName),
							},
						},
						ReturnType: unionOverride.GoName, // Go interface
					})
					rows := []string{
						fmt.Sprintf("eventType := C.get_%s_type(&cObj)", rawName),
						fmt.Sprintf("switch eventType {"),
					}
					for _, mapper := range unionOverride.Mappers {
						caseValues := make([]string, len(mapper.CEnumValues))
						for i, value := range mapper.CEnumValues {
							caseValues[i] = fmt.Sprintf("C.%s", value)
						}

						rows = append(rows, fmt.Sprintf("case %s:", strings.Join(caseValues, ", ")))

						if phantomOverride := converter.IsPhantomStruct(mapper.GoName); phantomOverride != nil {
							var typeField common.Field
							for _, field := range phantomOverride.Fields {
								if field.Name == unionOverride.TypeField.Name {
									typeField = field
									break
								}
							}
							rows = append(rows, fmt.Sprintf("\treturn &%s{%s: %s{cObj: cObj}, %s: %s(eventType)}", mapper.GoName, unionOverride.GoBaseName, unionOverride.GoBaseName, unionOverride.TypeField.Name, typeField.Type))
						} else {
							rows = append(rows, fmt.Sprintf("\treturn New%sFromC(%s{cObj: cObj}, C.get_%s_from_%s_union(&cObj))", mapper.GoName, unionOverride.GoBaseName, mapper.CTypeField.Type, rawName))
						}
					}
					rows = append(rows, "default:")
					rows = append(rows, "\treturn nil // or a fallback type like "+unionOverride.GoName+"Unknown{}")
					rows = append(rows, "}")

					writer.FunctionBody(common.FunctionBody{
						Rows: rows,
					})

					writer.VoidReturn()

					writer.Struct(common.Struct{
						Name: unionOverride.GoBaseName,
						Fields: []common.Field{
							{
								Name: "cObj",
								Type: fmt.Sprintf("C.%s", rawName),
							},
						},
					})

					for _, mapper := range unionOverride.Mappers {
						receiverName := strings.ToLower(mapper.GoName[:1]) // e.g. "k" for "KeyEvent"
						writer.ReceiverFunctionHeader(common.ReceiverFunctionHeader{
							ReceiverName: receiverName,
							ReceiverType: common.MakePointerType(mapper.GoName),
							MethodName:   "EventType",
							Parameters:   []common.Field{},
							ReturnType:   unionOverride.TypeField.Type,
						})
						writer.ReturnValue(fmt.Sprintf("%s.%s", receiverName, unionOverride.TypeField.Name))

						writer.ReceiverFunctionHeader(common.ReceiverFunctionHeader{
							ReceiverName: receiverName,
							ReceiverType: common.MakePointerType(mapper.GoName),
							MethodName:   "BaseToC",
							Parameters:   []common.Field{},
							ReturnType:   fmt.Sprintf("C.%s", rawName),
						})
						writer.ReturnValue(fmt.Sprintf("%s.%s.cObj", receiverName, unionOverride.GoBaseName))
					}

					continue
				}

				writer.Struct(common.Struct{
					Name:     structOverride.GoName,
					Fields:   structOverride.Fields,
					BaseType: structOverride.BaseType, // e.g. "EventBase"
				})

				// ToC
				receiverName := strings.ToLower(structOverride.GoName[:1]) // e.g. "v" for "Vector2i"
				writer.ReceiverFunctionHeader(common.ReceiverFunctionHeader{
					ReceiverName: receiverName, // e.g. "v" for "Vector2i"
					ReceiverType: common.MakePointerType(structOverride.GoName),
					MethodName:   "ToC",
					Parameters:   []common.Field{},
					ReturnType:   fmt.Sprintf("C.%s", rawName),
				})
				body := make([]string, 0, len(structOverride.CFields))
				funcRes := strings.Builder{}
				funcRes.WriteString(fmt.Sprintf("C.%s{ ", rawName))
				hasWrittenField := false
				for i, field := range structOverride.Fields {
					cField := structOverride.CFields[i]
					if cField.Name == "type" {
						body = append(body, fmt.Sprintf("C.set_%s_type(&funcRes, C.%s(%s.%s))", rawName, cField.Type, receiverName, field.Name))
						continue
					}

					if hasWrittenField {
						funcRes.WriteString(", ")
					}
					if _, subOverrideField := converter.GetOverriddenType(field.Type); subOverrideField != nil {
						funcRes.WriteString(fmt.Sprintf("%s: %s.%s.ToC()", cField.Name, receiverName, field.Name))
					} else if converter.IsKnownGoType(field.Type) && !converter.IsEnum(field.Type) {
						_, storeAsValue := converter.StoreAsValue[cField.Type]
						dereference := ""
						if storeAsValue {
							dereference = "*"
						}

						funcRes.WriteString(fmt.Sprintf("%s: %s%s.%s.ToC()", cField.Name, dereference, receiverName, common.TypeConverterToGo(field.Type)))
					} else {
						funcRes.WriteString(fmt.Sprintf("%s: %s(%s.%s)", structOverride.CFields[i].Name, common.TypeConverterToC(structOverride.CFields[i].Type), receiverName, field.Name))
					}
					hasWrittenField = true
				}
				funcRes.WriteString(" }")

				body = append([]string{fmt.Sprintf("funcRes := %s", funcRes.String())}, body...)

				writer.FunctionBody(common.FunctionBody{
					Rows: body,
				})
				writer.ReturnValue("funcRes")

				// NewFromC
				var params []common.Field
				if structOverride.BaseType != "" {
					params = append(params, common.Field{
						Name: "base",
						Type: structOverride.BaseType,
					})
				}
				writer.FunctionHeader(common.FunctionHeader{
					MethodName: "New" + structOverride.GoName + "FromC",
					Parameters: append(params, common.Field{
						Name: "cObj",
						Type: fmt.Sprintf("C.%s", rawName),
					}),
					ReturnType: fmt.Sprintf("*%s", structOverride.GoName),
				})
				funcRes = strings.Builder{}
				funcRes.WriteString(fmt.Sprintf("&%s{ ", structOverride.GoName))
				if structOverride.BaseType != "" {
					funcRes.WriteString(fmt.Sprintf("%s: base, ", structOverride.BaseType))
				}

				for i, field := range structOverride.Fields {
					if i > 0 {
						funcRes.WriteString(", ")
					}
					cField := structOverride.CFields[i]
					dereference := ""
					if !common.IsPointerType(field.Type) {
						dereference = "*"
					}

					if _, subOverrideField := converter.GetOverriddenType(field.Type); subOverrideField != nil {
						funcRes.WriteString(fmt.Sprintf("%s: %sNew%sFromC(cObj.%s)", field.Name, dereference, subOverrideField.GoName, cField.Name))
					} else if converter.IsKnownGoType(field.Type) && !converter.IsEnum(field.Type) {
						funcRes.WriteString(fmt.Sprintf("%s: %sNew%sFromC(cObj.%s)", field.Name, dereference, common.TypeConverterToGo(field.Type), cField.Name))
					} else {
						typeAccessor := fmt.Sprintf("cObj.%s", structOverride.CFields[i].Name)
						if structOverride.CFields[i].Name == "type" {
							typeAccessor = fmt.Sprintf("C.get_%s_type(&cObj)", rawName)
						}

						funcRes.WriteString(fmt.Sprintf("%s: %s(%s)", field.Name, common.TypeConverterToGo(field.Type), typeAccessor))
					}
				}
				funcRes.WriteString(" }")
				writer.ReturnValue(funcRes.String())

				// NewArrayFromC
				writer.FunctionHeader(common.FunctionHeader{
					MethodName: "New" + structOverride.GoName + "SliceFromCArray",
					Parameters: []common.Field{
						{
							Name: "ptr",
							Type: fmt.Sprintf("*C.%s", rawName),
						},
						{
							Name: "count",
							Type: "C.size_t",
						},
					},
					ReturnType: fmt.Sprintf("[]%s", structOverride.GoName),
				})
				writer.FunctionBody(common.FunctionBody{
					Rows: []string{
						// Assert sizeof matches between Go and C
						fmt.Sprintf("if unsafe.Sizeof(%s{}) != unsafe.Sizeof(C.%s{}) {", structOverride.GoName, rawName),
						"\tpanic(\"Size mismatch between Go and C types\")",
						"}",
						"",
						fmt.Sprintf("goSlice := make([]%s, int(count))", structOverride.GoName),
						"src := unsafe.Pointer(ptr)",
						"dst := unsafe.Pointer(&goSlice[0])",
						"size := int(count) * int(unsafe.Sizeof(" + structOverride.GoName + "{}))",
						"copy((*[1 << 30]byte)(dst)[:size:size], (*[1 << 30]byte)(src)[:size:size])",
					},
				})
				writer.ReturnValue("goSlice")

				// NewCArrayFromGo
				writer.FunctionHeader(common.FunctionHeader{
					MethodName: "New" + structOverride.GoName + "CArrayFromGoSlice",
					Parameters: []common.Field{
						{
							Name: "slice",
							Type: fmt.Sprintf("[]%s", structOverride.GoName),
						},
					},
					ReturnType: fmt.Sprintf("*C.%s", rawName),
				})
				writer.FunctionBody(common.FunctionBody{
					Rows: []string{
						// Assert sizeof matches between Go and C
						fmt.Sprintf("if unsafe.Sizeof(%s{}) != unsafe.Sizeof(C.%s{}) {", structOverride.GoName, rawName),
						"\tpanic(\"Size mismatch between Go and C types\")",
						"}",
						"",
						"if len(slice) == 0 {",
						"\treturn nil",
						"}",
						fmt.Sprintf("size := uintptr(len(slice)) * unsafe.Sizeof(%s{})", structOverride.GoName),
						"ptr := C.malloc(C.size_t(size))",
						"if ptr == nil {",
						"\tpanic(\"C.malloc failed\")",
						"}",
						"src := unsafe.Pointer(&slice[0])",
						"C.memcpy(ptr, src, C.size_t(size))",
					},
				})
				writer.ReturnValue(fmt.Sprintf("(*C.%s)(ptr)", rawName))

				continue
			}

			receiverName := strings.ToLower(goName[:1]) // e.g. "s" for "Sprite"
			if isPointer {
				writer.Struct(common.Struct{
					Name: goName,
					Fields: append([]common.Field{
						{Name: "ptr", Type: fmt.Sprintf("*C.%s", rawName)},
					}),
				})

				writer.ReceiverFunctionHeader(common.ReceiverFunctionHeader{
					ReceiverName: receiverName,
					ReceiverType: fmt.Sprintf("*%s", goName),
					MethodName:   "ToC",
					Parameters:   []common.Field{},
					ReturnType:   fmt.Sprintf("*C.%s", rawName),
				})
				writer.ReturnValue(fmt.Sprintf("%s.ptr", receiverName))

				writer.FunctionHeader(common.FunctionHeader{
					MethodName: "New" + goName + "FromC",
					Parameters: []common.Field{
						{
							Name: "cPtr",
							Type: fmt.Sprintf("*C.%s", rawName),
						},
					},
					ReturnType: common.MakePointerType(goName),
				})
				writer.ReturnValue(fmt.Sprintf("&%s{ptr: cPtr}", goName))
			} else {
				writer.Struct(common.Struct{
					Name: goName,
					Fields: append([]common.Field{
						{Name: "obj", Type: fmt.Sprintf("C.%s", rawName)},
					}),
				})

				writer.ReceiverFunctionHeader(common.ReceiverFunctionHeader{
					ReceiverName: receiverName,
					ReceiverType: common.MakePointerType(goName),
					MethodName:   "ToC",
					Parameters:   []common.Field{},
					ReturnType:   fmt.Sprintf("*C.%s", rawName),
				})
				writer.ReturnValue(fmt.Sprintf("&%s.obj", receiverName))

				writer.FunctionHeader(common.FunctionHeader{
					MethodName: "New" + goName + "FromC",
					Parameters: []common.Field{
						{
							Name: "cObj",
							Type: fmt.Sprintf("C.%s", rawName),
						},
					},
					ReturnType: common.MakePointerType(goName),
				})
				writer.ReturnValue(fmt.Sprintf("&%s{obj: cObj}", goName))
			}
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
					Name:  textcase.PascalCase(converter.StripPrefix(enumerator.Name)),
					Value: enumerator.Name,
				}
			}

			// Generate enum type:
			writer.Enum(common.Enum{
				Name:        goName,
				Enumerators: enumerators,
			})
		}
	}

	for _, f := range converter.PhantomStructOverrides {
		// Phantom structs are not real C types, but we still need to define them in Go
		writer.Struct(common.Struct{
			Name:     f.GoName,
			Fields:   f.Fields,
			BaseType: f.BaseType, // e.g. "EventBase"
		})
	}

	err = writer.WriteToFile(path.Join(common.OutputDir, "/go_types.go"))
	if err != nil {
		log.Fatalf("Failed to write to file: %v", err)
	}
}

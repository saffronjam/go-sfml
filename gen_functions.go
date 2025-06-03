package main

import (
	"fmt"
	"github.com/golang-cz/textcase"
	"github.com/saffronjam/go-sfml/internal/common"
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

	writer.HeaderFunctions()

	for _, fn := range converter.RawFunctions {
		originalName := fn.Name                         // e.g. "sfMouse_getPosition"
		stripped := converter.StripPrefix(originalName) // e.g. "Mouse_getPosition"
		parts := strings.SplitN(stripped, "_", 2)       // e.g. ["Mouse","getPosition"]
		if len(parts) != 2 {                            // skip if not "Type_Method"
			continue
		}

		_, methodPart := parts[0], parts[1]
		goMethod := textcase.PascalCase(methodPart) // e.g. "GetPosition"

		paramsC := fn.Parameters
		returnTypeC := fn.ReturnType                         // e.g. "sfVector2i" or "int"
		goReturnType := converter.MapReturnType(returnTypeC) // e.g. "Vector2i" or "int32"

		// Determine if this is a method with a receiver or a top-level function.
		var receiverType string
		usePtrReceiver := false
		if len(paramsC) > 0 {
			recv := converter.GetReceiverType(stripped, paramsC[0].Type)
			if recv != "" {
				receiverType = recv
				if strings.Contains(paramsC[0].Type, "*") {
					usePtrReceiver = true
				}
			}
		}

		if receiverType != "" {
			// --- METHOD ON A TYPE ---
			receiverVar := strings.ToLower(string(receiverType[0])) // e.g. "r" for "RenderWindow"
			receiverDecl := receiverType
			if usePtrReceiver {
				receiverDecl = "*" + receiverType
			}

			// Build parameter list excluding the first (receiver) param.
			otherParamsC := paramsC[1:]
			var goParams []common.Field
			var functionBodyRows []string
			var callArgs []string
			if usePtrReceiver {
				// Edge-case is when pointer receiver is actually a struct in Go, then we need to create a new
				// C object and pass its pointer.
				// e.g. for sfFloatRect.getPosition, we need to create a new C object:
				// cval := C.sfFloatRect{ ... }
				// and then pass its pointer:
				// callArgs = append(callArgs, fmt.Sprintf("&%s", receiverVar))
				if overrideInfo, ok := converter.StructOverrides[common.CleanCType(paramsC[0].Type)]; ok {
					var fieldAssignmentExpressions []string
					for idx, field := range overrideInfo.Fields {
						cField := overrideInfo.CFields[idx]
						fieldAssignmentExpressions = append(fieldAssignmentExpressions,
							fmt.Sprintf("%s: C.%s(%s.%s)", cField.Name, cField.Type, receiverVar, field.Name),
						)
					}

					functionBodyRows = append(functionBodyRows, fmt.Sprintf("var0 := C.%s{%s}", common.CleanCType(paramsC[0].Type), strings.Join(fieldAssignmentExpressions, ", ")))
					callArgs = append(callArgs, fmt.Sprintf("&var0"))

				} else {
					callArgs = append(callArgs, fmt.Sprintf("%s.CPtr()", receiverVar))
				}

			} else {
				callArgs = append(callArgs, receiverVar)
			}

			for i, cParam := range otherParamsC {
				pname := cParam.Name
				if pname == "" {
					pname = fmt.Sprintf("var%d", i)
				}
				// Map the C type to Go type
				pty := converter.MapCParamToGoType(cParam.Type)
				goParam := common.SanitizeFieldName(common.Field{Name: pname, Type: pty})

				argVarName := fmt.Sprintf("var%d", len(functionBodyRows))

				if overrideInfo, hasOverride := converter.StructOverrides[common.CleanCType(cParam.Type)]; hasOverride {
					// If the parameter is a struct with an override, we need to create a new C object
					var fieldAssignmentExpressions []string
					for idx, field := range overrideInfo.Fields {
						cField := overrideInfo.CFields[idx]
						fieldAssignmentExpressions = append(fieldAssignmentExpressions,
							fmt.Sprintf("%s: C.%s(%s.%s)", cField.Name, cField.Type, goParam.Name, field.Name),
						)
					}

					functionBodyRows = append(functionBodyRows, fmt.Sprintf("%s := C.%s{%s}", argVarName, common.CleanCType(cParam.Type), strings.Join(fieldAssignmentExpressions, ", ")))
					callArgs = append(callArgs, fmt.Sprintf("&%s", argVarName))
				} else if converter.IsKnownGoType(goParam.Type) && !converter.IsEnum(goParam.Type) {
					// If it is a known Go type, we need to pass it as a pointer using var1.CPtr()
					functionBodyRows = append(functionBodyRows, fmt.Sprintf("%s := %s.CPtr()", argVarName, goParam.Name))
					callArgs = append(callArgs, argVarName)
				} else {
					// Otherwise, we can use the type directly
					functionBodyRows = append(functionBodyRows, fmt.Sprintf("%s := %s", argVarName, goParam.Name))
					callArgs = append(callArgs, argVarName)
				}
				goParams = append(goParams, goParam)
			}

			// Signature line: func (r *RenderWindow) GetPosition(...)
			writer.ReceiverFunctionHeader(common.ReceiverFunctionHeader{
				ReceiverName: receiverVar,
				ReceiverType: receiverDecl,
				MethodName:   goMethod,
				Parameters:   goParams,
				ReturnType:   goReturnType,
			})

			if overrideInfo, hasOverride := converter.StructOverrides[common.CleanCType(returnTypeC)]; hasOverride {
				writer.FunctionBody(common.FunctionBody{
					Rows: append(functionBodyRows, fmt.Sprintf("funcRes0 := C.%s(%s)", originalName, strings.Join(callArgs, ", "))),
				})

				// Determine which fields to extract
				var fieldAssignmentExpressions []string
				for idx, field := range overrideInfo.Fields {
					cField := overrideInfo.CFields[idx]
					fieldAssignmentExpressions = append(fieldAssignmentExpressions,
						fmt.Sprintf("%s: %s(funcRes0.%s)", field.Name, field.Type, cField.Name),
					)
				}

				if !common.IsVoidReturnType(goReturnType) {
					writer.ReturnValue(fmt.Sprintf("%s{%s}", goReturnType, strings.Join(fieldAssignmentExpressions, ", ")))
				} else {
					writer.VoidReturn()
				}
			} else {
				callExpr := fmt.Sprintf("C.%s(%s)", originalName, strings.Join(callArgs, ", "))
				if common.IsVoidReturnType(goReturnType) {
					writer.FunctionBody(common.FunctionBody{Rows: append(functionBodyRows, callExpr)})
					writer.VoidReturn()
				} else {
					if strings.HasPrefix(goReturnType, "*") || converter.IsKnownGoType(goReturnType) {
						// Opaque pointer return
						pureGoType := common.StripPointer(goReturnType)
						if converter.IsKnownGoType(pureGoType) && !converter.IsEnum(pureGoType) {
							// If type is a known Go type we can't cast it directly, and we need to create it from the pointer
							// e.g. for *Transform we need to:
							// cval := C.sfTransform_create()
							// return &Transform{ ptr: cval }

							writer.FunctionBody(common.FunctionBody{Rows: append(functionBodyRows, fmt.Sprintf("funcRes0 := unsafe.Pointer(%s)", callExpr))})
							writer.ReturnValue(fmt.Sprintf("&%s{ptr: funcRes0}", common.StripPointer(goReturnType)))
						} else {
							// For other types, we can directly cast the pointer
							writer.ReturnValue(fmt.Sprintf("%s(%s)", goReturnType, callExpr))
						}

					} else {
						if len(functionBodyRows) > 0 {
							// If we have a function body, we need to ensure we return the pointer correctly
							writer.FunctionBody(common.FunctionBody{Rows: functionBodyRows})
						}
						// Primitive return, int32, float32, etc. (any other type should have been caught earlier)
						writer.ReturnValue(callExpr)
					}
				}
			}

		} else {
			// --- TOP‐LEVEL (GLOBAL) FUNCTION ---
			goFuncName := textcase.PascalCase(stripped)
			var goParams []common.Field
			var callArgs []string

			for i, cParam := range paramsC {
				pname := cParam.Name
				if pname == "" {
					pname = fmt.Sprintf("arg%d", i)
				}
				pty := converter.MapCParamToGoType(cParam.Type)
				goParam := common.SanitizeFieldName(common.Field{Name: pname, Type: pty})

				goParams = append(goParams, goParam)
				callArgs = append(callArgs, converter.ParamCallExpr(cParam, goParam))
			}

			// Determine return type for the function signature
			writer.FunctionHeader(common.FunctionHeader{
				MethodName: goFuncName,
				Parameters: goParams,
				ReturnType: goReturnType,
			})

			if overrideInfo, hasOverride := converter.StructOverrides[common.CleanCType(returnTypeC)]; hasOverride {
				writer.FunctionBody(common.FunctionBody{
					Rows: []string{
						fmt.Sprintf("cval := C.%s(%s)", originalName, strings.Join(callArgs, ", ")),
					},
				})

				var fieldAssignmentExpressions []string
				for idx, field := range overrideInfo.Fields {
					cField := overrideInfo.CFields[idx]
					fieldAssignmentExpressions = append(fieldAssignmentExpressions,
						fmt.Sprintf("%s: %s(cval.%s)", field.Name, field.Type, cField.Name),
					)
				}

				if !common.IsVoidReturnType(goReturnType) {
					writer.ReturnValue(fmt.Sprintf("%s{%s}", goReturnType, strings.Join(fieldAssignmentExpressions, ", ")))
				} else {
					writer.VoidReturn()
				}
			} else {

				// Non-vector return
				callExpr := fmt.Sprintf("C.%s(%s)", originalName, strings.Join(callArgs, ", "))
				if common.IsVoidReturnType(goReturnType) {
					writer.FunctionBody(common.FunctionBody{Rows: []string{callExpr}})
					writer.VoidReturn()
				} else {
					if strings.HasPrefix(goReturnType, "*") || converter.IsKnownGoType(goReturnType) {
						// Opaque pointer return
						pureGoType := common.StripPointer(goReturnType)
						if converter.IsKnownGoType(pureGoType) && !converter.IsEnum(pureGoType) {
							// If type is a known Go type we can't cast it directly, and we need to create it from the pointer
							// e.g. for *Transform we need to:
							// cval := C.sfTransform_create()
							// return &Transform{ ptr: cval }

							writer.FunctionBody(common.FunctionBody{Rows: []string{fmt.Sprintf("cval := unsafe.Pointer(%s)", callExpr)}})
							writer.ReturnValue(fmt.Sprintf("&%s{ptr: cval}", common.StripPointer(goReturnType)))

						} else {
							// For other types, we can directly cast the pointer
							writer.ReturnValue(fmt.Sprintf("%s(%s)", goReturnType, callExpr))
						}

					} else {
						// Primitive return, int32, float32, etc. (any other type should have been caught earlier)
						writer.ReturnValue(callExpr)
					}
				}
			}
		}
	}

	err = writer.WriteToFile(path.Join(common.OutputDir, "go_functions.go"))
	if err != nil {
		panic(fmt.Sprintf("Failed to write to file: %v", err))
	}

	fmt.Println("✅ Generated go_functions.go with correct Vector2*/Vector3f return handling.")
}

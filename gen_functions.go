package main

import (
	"fmt"
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

	writer, err := common.NewWriter(config.GithubRepo, converter, common.MetadataFile)
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
		goStaticMethod := converter.TranslateMethodName(stripped)     // e.g. false for "sfMouse_getPosition"
		goReceiverMethod := converter.TranslateMethodName(methodPart) // e.g. "GetPosition"

		paramsC := fn.Parameters
		returnTypeC := fn.ReturnType                        // e.g. "sfVector2i" or "int"
		goReturnType := converter.MapCToGoType(returnTypeC) // e.g. "Vector2i" or "int32"

		// Determine if this is a method with a receiver or a top-level function.
		var receiverType string
		if len(paramsC) > 0 {
			recv := converter.GetReceiverType(stripped, paramsC[0].Type)
			if recv != "" {
				receiverType = recv
			}
		}

		if receiverType != "" {
			// --- METHOD ON A TYPE ---
			receiverVar := strings.ToLower(string(receiverType[0])) // e.g. "r" for "RenderWindow"
			receiverDecl := common.MakePointerType(receiverType)

			// Build parameter list excluding the first (receiver) param.
			otherParamsC := paramsC[1:]
			var goParams []common.Field
			var functionBodyRows []string
			var callArgs []string

			functionBodyRows = append(functionBodyRows, fmt.Sprintf("var0 := %s.ToC()", receiverVar))
			ampersand := ""
			_, isStructOverride := converter.StructOverrides[common.CleanCType(paramsC[0].Type)]
			if common.IsPointerType(paramsC[0].Type) && isStructOverride {
				ampersand = "&"
			}
			callArgs = append(callArgs, fmt.Sprintf("%svar0", ampersand))

			returnType := goReturnType
			if !converter.IsEnum(returnType) && !common.IsNativeGoType(goReturnType) {
				// If the return type is not an enum, we need to prepend a pointer
				returnType = common.MakePointerType(goReturnType)
			}

			for i, cParam := range otherParamsC {
				pname := cParam.Name
				if pname == "" {
					pname = fmt.Sprintf("var%d", i)
				}
				// Map the C type to Go type
				pty := converter.MapCToGoType(cParam.Type)
				goParam := common.SanitizeFieldName(common.Field{Name: pname, Type: pty})

				argVarName := fmt.Sprintf("var%d", len(functionBodyRows))

				if structOverride, hasOverride := converter.StructOverrides[common.CleanCType(cParam.Type)]; hasOverride {
					sliceParam := converter.IsSliceParam(originalName, cParam.Name)
					returnParam := converter.IsReturnParam(originalName, cParam.Name)

					if sliceParam != nil {
						var countParamType string
						for _, p := range paramsC {
							if p.Name == sliceParam.CCountParam {
								countParamType = p.Type
								break
							}
						}
						if countParamType == "" {
							panic(fmt.Sprintf("Slice count parameter '%s' not found for function '%s'", sliceParam.CCountParam, originalName))
						}

						functionBodyRows = append(functionBodyRows, fmt.Sprintf("%sCount := %s(len(%s))", argVarName, common.TypeConverterToC(countParamType), goParam.Name))
						functionBodyRows = append(functionBodyRows, fmt.Sprintf("%sArray := New%sCArrayFromGoSlice(%s)", argVarName, common.StripPointer(goParam.Type), goParam.Name))
						callArgs = append(callArgs, fmt.Sprintf("%sArray, %sCount", argVarName, argVarName))
						// Overwrite the goParam to be the array type
						goParam.Type = fmt.Sprintf("[]%s", common.StripPointer(goParam.Type))
					} else if returnParam != nil {
						// Overwrite the return type to move the param to the return value
						if _, isUnion := converter.UnionOverrides[common.CleanCType(returnParam.Type)]; isUnion {
							returnType = common.PrependReturnType(returnType, structOverride.GoName)
						} else {
							returnType = common.PrependReturnType(returnType, common.MakePointerType(structOverride.GoName))
						}

						// Since this should only be done for params that are not expected to have a value going
						// into the function, we can create its C type empty directly, and then pass it as a pointer
						// to the function.
						functionBodyRows = append(functionBodyRows, fmt.Sprintf("returnParam := C.%s{}", common.StripPointer(returnParam.Type)))
						callArgs = append(callArgs, fmt.Sprintf("&returnParam"))
						continue
					} else {
						functionBodyRows = append(functionBodyRows, fmt.Sprintf("%s := %s.ToC()", argVarName, goParam.Name))
						ampersand = ""
						if common.IsPointerType(cParam.Type) {
							ampersand = "&"
						}
						callArgs = append(callArgs, fmt.Sprintf("%s%s", ampersand, argVarName))
					}
				} else if converter.IsKnownGoType(common.StripPointer(goParam.Type)) && !converter.IsEnum(goParam.Type) {
					// If it is a known Go type, we need to pass it as a pointer using var1.ToC()
					functionBodyRows = append(functionBodyRows, fmt.Sprintf("%s := %s.ToC()", argVarName, goParam.Name))
					dereference := ""
					_, storeAsValue := converter.StoreAsValueOverrides[common.CleanCType(cParam.Type)]
					if !common.IsPointerType(cParam.Type) && storeAsValue {
						dereference = "*"
					}

					callArgs = append(callArgs, fmt.Sprintf("%s%s", dereference, argVarName))
				} else if goParam.Type == "string" {
					// If the parameter is a string, we need to convert it to a C string
					functionBodyRows = append(functionBodyRows, fmt.Sprintf("%s := C.CString(%s)", argVarName, goParam.Name))
					callArgs = append(callArgs, argVarName)
				} else {
					sliceCountParam := converter.IsSliceCountParam(originalName, cParam.Name)
					if sliceCountParam != nil {
						// Assume all slice count params are preceded by the slice param, so its
						// param is already in the function body. But since length is built into
						// the slice, we need to delete the CParam
						continue
					}

					functionBodyRows = append(functionBodyRows, fmt.Sprintf("%s := %s(%s)", argVarName, common.TypeConverterToC(cParam.Type), goParam.Name))
					callArgs = append(callArgs, argVarName)
				}
				goParams = append(goParams, goParam)
			}

			// Signature line: func (r *RenderWindow) GetPosition(...)
			writer.ReceiverFunctionHeader(common.ReceiverFunctionHeader{
				ReceiverName: receiverVar,
				ReceiverType: receiverDecl,
				MethodName:   goReceiverMethod,
				Parameters:   goParams,
				ReturnType:   returnType,
			})

			callExpr := fmt.Sprintf("C.%s(%s)", originalName, strings.Join(callArgs, ", "))
			if common.IsVoidReturnType(returnType) {
				functionBodyRows = append(functionBodyRows, callExpr)
				writer.FunctionBody(common.FunctionBody{Rows: functionBodyRows})
				writer.VoidReturn()
			} else {
				functionBodyRows = append(functionBodyRows, fmt.Sprintf("funcRes0 := %s", callExpr))

				// Check if we have multi-return values, meaning: check if we have returnParams
				returnParam, hasMultiReturn := converter.ReturnParamOverrides[originalName]
				if hasMultiReturn {
					// Every return param override is a struct override
					structOverride, _ := converter.StructOverrides[common.CleanCType(returnParam.Type)]
					functionBodyRows = append(functionBodyRows, fmt.Sprintf("returnParamRes := New%sFromC(returnParam)", common.StripPointer(structOverride.GoName)))
				}

				if converter.IsKnownGoType(returnType) && !converter.IsEnum(returnType) {
					functionBodyRows = append(functionBodyRows, fmt.Sprintf("res := New%sFromC(funcRes0)", common.StripPointer(goReturnType)))
				} else {
					functionBodyRows = append(functionBodyRows, fmt.Sprintf("res := %s(funcRes0)", common.TypeConverterToGo(goReturnType)))
				}
				writer.FunctionBody(common.FunctionBody{Rows: functionBodyRows})
				if hasMultiReturn {
					writer.ReturnValue(fmt.Sprintf("returnParamRes, res"))
				} else {
					writer.ReturnValue("res")
				}
			}
		} else {
			// --- TOP‐LEVEL (GLOBAL) FUNCTION ---
			var goParams []common.Field
			var functionBodyRows []string
			var callArgs []string

			for i, cParam := range paramsC {
				pname := cParam.Name
				if pname == "" {
					pname = fmt.Sprintf("arg%d", i)
				}
				pty := converter.MapCToGoType(cParam.Type)
				goParam := common.SanitizeFieldName(common.Field{Name: pname, Type: pty})

				argVarName := fmt.Sprintf("var%d", len(functionBodyRows))

				if _, hasOverride := converter.StructOverrides[common.CleanCType(cParam.Type)]; hasOverride {
					functionBodyRows = append(functionBodyRows, fmt.Sprintf("%s := %s.ToC()", argVarName, goParam.Name))
					ampersand := ""
					if common.IsPointerType(goParam.Type) {
						ampersand = "&"
					}
					callArgs = append(callArgs, fmt.Sprintf("%s%s", ampersand, argVarName))
				} else if converter.IsKnownGoType(common.StripPointer(goParam.Type)) && !converter.IsEnum(goParam.Type) {
					functionBodyRows = append(functionBodyRows, fmt.Sprintf("%s := %s.ToC()", argVarName, goParam.Name))
					callArgs = append(callArgs, argVarName)
				} else if goParam.Type == "string" {
					nilParamOverride := converter.IsNilParamOverride(originalName, cParam.Name)
					if nilParamOverride != nil {
						goParam.Type = common.MakePointerType(goParam.Type)
						functionBodyRows = append(functionBodyRows, fmt.Sprintf("var %s *C.char = nil", argVarName))
						functionBodyRows = append(functionBodyRows, fmt.Sprintf("if %s != nil {", goParam.Name))
						functionBodyRows = append(functionBodyRows, fmt.Sprintf("  %s = C.CString(*%s)", argVarName, goParam.Name))
						functionBodyRows = append(functionBodyRows, fmt.Sprintf("}"))
						callArgs = append(callArgs, fmt.Sprintf("%s", argVarName))
					} else {
						functionBodyRows = append(functionBodyRows, fmt.Sprintf("%s := C.CString(%s)", argVarName, goParam.Name))
						callArgs = append(callArgs, argVarName)
					}
				} else {
					functionBodyRows = append(functionBodyRows, fmt.Sprintf("%s := %s(%s)", argVarName, common.TypeConverterToC(cParam.Type), goParam.Name))
					callArgs = append(callArgs, argVarName)
				}
				goParams = append(goParams, goParam)
			}

			returnType := goReturnType
			if !converter.IsEnum(returnType) && !common.IsNativeGoType(goReturnType) {
				// If the return type is not an enum, we need to prepend a pointer
				returnType = common.MakePointerType(goReturnType)
			}

			// Determine return type for the function signature
			writer.FunctionHeader(common.FunctionHeader{
				MethodName: goStaticMethod,
				Parameters: goParams,
				ReturnType: returnType,
			})

			if _, hasOverride := converter.StructOverrides[common.CleanCType(returnTypeC)]; hasOverride {

				functionBodyRows = append(functionBodyRows, fmt.Sprintf("funcRes0 := C.%s(%s)", originalName, strings.Join(callArgs, ", ")))

				writer.FunctionBody(common.FunctionBody{Rows: functionBodyRows})
				if !common.IsVoidReturnType(goReturnType) {
					writer.ReturnValue(fmt.Sprintf("New%sFromC(%s)", common.StripPointer(goReturnType), "funcRes0"))
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
						pureGoType := common.StripPointer(goReturnType)
						if converter.IsKnownGoType(pureGoType) && !converter.IsEnum(pureGoType) {
							functionBodyRows = append(functionBodyRows, fmt.Sprintf("funcRes0 := %s", callExpr))
							writer.FunctionBody(common.FunctionBody{Rows: functionBodyRows})
							writer.ReturnValue(fmt.Sprintf("New%sFromC(funcRes0)", common.StripPointer(goReturnType)))
						} else {
							writer.FunctionBody(common.FunctionBody{Rows: functionBodyRows})
							writer.ReturnValue(fmt.Sprintf("%s(%s)", common.TypeConverterToGo(goReturnType), callExpr))
						}
					} else {
						writer.FunctionBody(common.FunctionBody{Rows: functionBodyRows})
						writer.ReturnValue(fmt.Sprintf("%s(%s)", common.TypeConverterToGo(goReturnType), callExpr))
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

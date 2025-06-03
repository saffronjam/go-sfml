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
			var callArgs []string
			if usePtrReceiver {
				callArgs = append(callArgs, fmt.Sprintf("%s.CPtr()", receiverVar))
			} else {
				callArgs = append(callArgs, receiverVar)
			}

			for i, cParam := range otherParamsC {
				pname := cParam.Name
				if pname == "" {
					pname = fmt.Sprintf("arg%d", i)
				}
				// Map the C type to Go type
				pty := converter.MapCParamToGoType(cParam.Type)
				goParam := common.SanitizeFieldName(common.Field{Name: pname, Type: pty})

				goParams = append(goParams, goParam)
				callArgs = append(callArgs, converter.ParamCallExpr(cParam, goParam))
			}

			// Signature line: func (r *RenderWindow) GetPosition(...)
			writer.ReceiverFunctionHeader(common.ReceiverFunctionHeader{
				ReceiverName: receiverVar,
				ReceiverType: receiverDecl,
				MethodName:   goMethod,
				Parameters:   goParams,
				ReturnType:   goReturnType,
			})

			// Handle vector returns specially
			if structOverrideInfo, hasOverride := converter.StructOverrides[common.CleanCType(returnTypeC)]; hasOverride {
				// cval := C.sfMouse_getPosition(r.CPtr(), ...)

				writer.FunctionBody(common.FunctionBody{
					Rows: []string{
						fmt.Sprintf("cval := C.%s(%s)", originalName, strings.Join(callArgs, ", ")),
					},
				})

				// Determine which fields to extract
				var fieldAssignmentExpressions []string
				for _, field := range structOverrideInfo.Fields {
					cFieldName := strings.ToLower(field.Name) // C fields are lowercase
					goCast := common.PrimitiveCast(structOverrideInfo.GoName)
					fieldAssignmentExpressions = append(fieldAssignmentExpressions,
						fmt.Sprintf("%s: %s(cval.%s)", field, goCast, cFieldName),
					)
				}

				writer.ReturnValue(fmt.Sprintf("%s{%s}", goReturnType, strings.Join(fieldAssignmentExpressions, ", ")))
			} else {
				// Non-vector return
				callExpr := fmt.Sprintf("C.%s(%s)", originalName, strings.Join(callArgs, ", "))
				if common.IsVoidReturnType(goReturnType) {
					writer.FunctionBody(common.FunctionBody{Rows: []string{callExpr}})
				} else {
					if strings.HasPrefix(goReturnType, "*") || converter.IsKnownGoType(goReturnType) {
						// Opaque pointer return
						writer.ReturnValue(fmt.Sprintf("%s(unsafe.Pointer(%s))", goReturnType, callExpr))
					} else {
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

			if structOverrideInfo, hasOverride := converter.StructOverrides[common.CleanCType(returnTypeC)]; hasOverride {
				writer.FunctionBody(common.FunctionBody{
					Rows: []string{
						fmt.Sprintf("cval := C.%s(%s)", originalName, strings.Join(callArgs, ", ")),
					},
				})

				var fieldAssignmentExpressions []string
				for _, field := range structOverrideInfo.Fields {
					cFieldName := strings.ToLower(field.Name) // C fields are lowercase
					goCast := common.PrimitiveCast(structOverrideInfo.GoName)
					fieldAssignmentExpressions = append(fieldAssignmentExpressions,
						fmt.Sprintf("%s: %s(cval.%s)", field, goCast, cFieldName),
					)
				}

				writer.ReturnValue(fmt.Sprintf("%s{%s}", goReturnType, strings.Join(fieldAssignmentExpressions, ", ")))
			} else {

				// Non-vector return
				callExpr := fmt.Sprintf("C.%s(%s)", originalName, strings.Join(callArgs, ", "))
				if common.IsVoidReturnType(goReturnType) {
					writer.FunctionBody(common.FunctionBody{Rows: []string{callExpr}})
				} else {
					if strings.HasPrefix(goReturnType, "*") || converter.IsKnownGoType(goReturnType) {
						// Opaque pointer return
						writer.ReturnValue(fmt.Sprintf("%s(unsafe.Pointer(%s))", goReturnType, callExpr))
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

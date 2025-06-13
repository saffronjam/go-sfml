package main

import (
	"log"
	"os"
	"strings"
	"text/template"
)

// Defines the data structure for each vector type we want to generate.
var vectorTypes = []struct {
	Name       string   // e.g., "Vector2f"
	TypeName   string   // e.g., "float32"
	Components []string // e.g., []string{"X", "Y"}
	HasFloat   bool     // True if the type is a float, for float-specific functions
}{
	{Name: "Vector2f", TypeName: "float32", Components: []string{"X", "Y"}, HasFloat: true},
	{Name: "Vector2d", TypeName: "float64", Components: []string{"X", "Y"}, HasFloat: true},
	{Name: "Vector2i", TypeName: "int32", Components: []string{"X", "Y"}, HasFloat: false},
	{Name: "Vector2u", TypeName: "uint32", Components: []string{"X", "Y"}, HasFloat: false},
	{Name: "Vector3f", TypeName: "float32", Components: []string{"X", "Y", "Z"}, HasFloat: true},
	{Name: "Vector3d", TypeName: "float64", Components: []string{"X", "Y", "Z"}, HasFloat: true},
	{Name: "Vector3i", TypeName: "int32", Components: []string{"X", "Y", "Z"}, HasFloat: false},
	{Name: "Vector3u", TypeName: "uint32", Components: []string{"X", "Y", "Z"}, HasFloat: false},
	{Name: "Vector4f", TypeName: "float32", Components: []string{"X", "Y", "Z", "W"}, HasFloat: true},
	{Name: "Vector4d", TypeName: "float64", Components: []string{"X", "Y", "Z", "W"}, HasFloat: true},
	{Name: "Vector4i", TypeName: "int32", Components: []string{"X", "Y", "Z", "W"}, HasFloat: false},
	{Name: "Vector4u", TypeName: "uint32", Components: []string{"X", "Y", "Z", "W"}, HasFloat: false},
}

func main() {
	// --- 1. Setup: Create output directory and file ---
	log.Println("Starting code generation...")
	outputDir := "./generated"
	outputFile := outputDir + "/go_addon_vector.go"

	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	f, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer f.Close()

	// --- 2. Parsing: Load all templates from the ./templates directory ---
	// We also add a custom "ToLower" function to use in the templates.
	tmpl, err := template.New("main.tpl").Funcs(template.FuncMap{
		"ToLower": strings.ToLower,
	}).ParseGlob("templates/*.tpl")

	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}

	// --- 3. Execution: Run the main template and write to the file ---
	// We execute "main.tpl" and pass our vectorTypes slice as the data.
	if err := tmpl.Execute(f, vectorTypes); err != nil {
		log.Fatalf("Failed to execute template: %v", err)
	}

	log.Printf("Successfully generated %s", outputFile)
}

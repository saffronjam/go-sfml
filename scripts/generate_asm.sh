#!/usr/bin/env bash

set -euo pipefail

if [[ -z "${CSFML_DIR:-}" ]]; then
    echo "Error: CSFML_DIR is not set. Please set it to the path of the CSFML repository."
    exit 1
fi

if [[ -z "${AST_DIR:-}" ]]; then
    echo "Error: AST_DIR is not set. Please set it to the path where you want to store the AST JSON files."
    exit 1
fi


INCLUDE_DIR="$CSFML_DIR/include"
if [[ ! -d "$INCLUDE_DIR" ]]; then
    echo "Error: Include directory $INCLUDE_DIR does not exist. Please check your CSFML_DIR."
    exit 1
fi

mkdir -p "$AST_DIR"

echo "🔍 Searching for headers in $INCLUDE_DIR..."

find "$INCLUDE_DIR" -name '*.h' | while read -r header; do
    rel_path="${header#$INCLUDE_DIR/}"                # remove include prefix
    json_file="${rel_path//\//_}"                     # flatten path (Window/Window.h -> Window_Window.h)
    json_file="${json_file%.h}.json"                  # change .h to .json
    out_path="$AST_DIR/$json_file"

    if [[ -f "$out_path" ]]; then
        echo "⏩ Skipping $rel_path (already exists)"
        continue
    fi

    echo "📦 Processing $rel_path → $out_path"
    clang -Xclang -ast-dump=json -fsyntax-only "$header" -I"$INCLUDE_DIR" > "$out_path"
done

echo "✅ All headers processed into $AST_DIR/"

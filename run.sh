#!/usr/bin/env bash

set -euo pipefail

git_repo_root="$(git rev-parse --show-toplevel)"
if [[ -z "$git_repo_root" ]]; then
    echo "Error: Not in a git repository. Please run this script from the root of the CSFML repository."
    exit 1
fi

cd "$git_repo_root"

export CSFML_DIR="./CSFML"
export GEN_DIR="./generated"
export JSON_DIR="$GEN_DIR/json"
export AST_DIR="$JSON_DIR/ast_json"
export SCRIPT_DIR="./scripts"
export PUBLIC_DIR="./public"

mkdir -p "$AST_DIR"
mkdir -p "$JSON_DIR"

# clone CSFML repository if not already done
if [ ! -d "CSFML" ]; then
    echo "🔄 Cloning CSFML repository..."
    git clone --depth 1 https://github.com/SFML/CSFML -b "2.6.x" CSFML
else
    echo "🔄 CSFML repository already exists, pulling latest changes..."
    cd CSFML
    git pull origin 2.6.x
    cd ..
fi

# If --clean is passed, remove existing generated files
if [[ "${1:-}" == "--clean" ]]; then
    echo "🧹 Cleaning up existing generated files..."
    rm -rf "./generated"
fi

echo "🔍 Generating AST from headers..."
"$SCRIPT_DIR/generate_asm.sh"

echo "📦 Running extract_all.py..."
python3 scripts/extract_all.py

echo "📦 Running Go code generators..."
go run gen_types.go
go run gen_functions.go

# Move to public directory
echo "📁 Moving generated files to public directory..."
rm -rf "$PUBLIC_DIR"
mkdir -p "$PUBLIC_DIR/sfml"

mv "$GEN_DIR/go_types.go" "$PUBLIC_DIR/sfml/go_types.go"
mv "$GEN_DIR/go_functions.go" "$PUBLIC_DIR/sfml/go_functions.go"

echo "✅ Done. Output in $PUBLIC_DIR/sfml/"


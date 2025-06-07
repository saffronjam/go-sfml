#!/usr/bin/env bash

# This script builds the libraries from the local CSFML repository.

set -e

REPO_ROOT=$(dirname "$(realpath "$0")")/..
cd "$REPO_ROOT"

CSFML_DIR="./CSFML"
if [ ! -d "$CSFML_DIR" ]; then
    echo "CSFML directory not found. Please clone the CSFML repository into the root directory."
    exit 1
fi


# Build CSFML with CMake
cd "$CSFML_DIR"
mkdir -p build
cd build
cmake .. -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX="$REPO_ROOT/CSFML/install"
make -j$(nproc)


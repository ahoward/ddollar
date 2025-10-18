#!/bin/bash

# Cross-platform build script for ddollar

set -e

VERSION="0.1.0"
OUTPUT_DIR="./dist"

echo "Building ddollar v${VERSION} for multiple platforms..."

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Build for macOS Intel
echo "Building for macOS (Intel x86_64)..."
GOOS=darwin GOARCH=amd64 go build -o "${OUTPUT_DIR}/ddollar-macos-x86_64" ./src

# Build for macOS Apple Silicon
echo "Building for macOS (Apple Silicon ARM64)..."
GOOS=darwin GOARCH=arm64 go build -o "${OUTPUT_DIR}/ddollar-macos-arm64" ./src

# Build for Linux x86_64
echo "Building for Linux (x86_64)..."
GOOS=linux GOARCH=amd64 go build -o "${OUTPUT_DIR}/ddollar-linux-x86_64" ./src

# Build for Linux ARM64
echo "Building for Linux (ARM64)..."
GOOS=linux GOARCH=arm64 go build -o "${OUTPUT_DIR}/ddollar-linux-arm64" ./src

# Build for Windows x86_64
echo "Building for Windows (x86_64)..."
GOOS=windows GOARCH=amd64 go build -o "${OUTPUT_DIR}/ddollar-windows-x86_64.exe" ./src

echo ""
echo "✓ Build complete! Binaries available in ${OUTPUT_DIR}:"
ls -lh "${OUTPUT_DIR}"

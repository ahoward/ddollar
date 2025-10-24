#!/bin/bash
# Build ddollar binaries for all platforms

set -e

VERSION=$(grep 'const version' src/main.go | cut -d'"' -f2)
OUTDIR="releases"

echo "Building ddollar v$VERSION for all platforms..."

mkdir -p "$OUTDIR"

# Build for each platform
build() {
    local os=$1
    local arch=$2
    local output="$OUTDIR/ddollar-${os}-${arch}"

    if [ "$os" = "windows" ]; then
        output="${output}.exe"
    fi

    echo "Building for $os/$arch..."
    GOOS=$os GOARCH=$arch go build -ldflags="-s -w" -o "$output" ./src

    if [ -f "$output" ]; then
        echo "  ✓ $output ($(du -h "$output" | cut -f1))"
    fi
}

# macOS
build darwin amd64
build darwin arm64

# Linux
build linux amd64
build linux arm64

# Windows
build windows amd64

echo ""
echo "✓ Release builds complete!"
echo ""
ls -lh "$OUTDIR"

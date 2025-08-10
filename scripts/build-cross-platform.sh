#!/bin/bash

# Build script for Frugal Language Server
# For local development, builds for current platform only (CGO dependency limits cross-compilation)

set -e

# Colors for output
GREEN='\033[32m'
YELLOW='\033[33m'
RED='\033[31m'
BLUE='\033[34m'
RESET='\033[0m'

# Configuration
BINARY_NAME="frugal-ls"
BUILD_DIR="build"
CMD_DIR="./cmd/frugal-ls"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS="-s -w -X main.version=${VERSION}"

# Platform matrix - for local development, only build for current platform
# Full cross-compilation requires cross-compilers for CGO
CURRENT_OS=$(go env GOOS)
CURRENT_ARCH=$(go env GOARCH)

# Only build for current platform locally due to CGO requirements
PLATFORMS=(
    "${CURRENT_OS}/${CURRENT_ARCH}"
)

echo -e "${BLUE}Cross-platform build for Frugal Language Server${RESET}"
echo -e "${BLUE}Version: ${VERSION}${RESET}"
echo ""

# Create build directory
mkdir -p "${BUILD_DIR}"

# Build for each platform
for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r GOOS GOARCH <<< "$platform"
    
    echo -e "${YELLOW}Building for ${GOOS}/${GOARCH}...${RESET}"
    
    # Set binary name with .exe extension for Windows
    OUTPUT_NAME="${BINARY_NAME}"
    if [ "$GOOS" = "windows" ]; then
        OUTPUT_NAME="${BINARY_NAME}.exe"
    fi
    
    OUTPUT_PATH="${BUILD_DIR}/${BINARY_NAME}-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        OUTPUT_PATH="${OUTPUT_PATH}.exe"
    fi
    
    # Build with CGO enabled (required for tree-sitter)
    env GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED=1 \
        go build -v -ldflags="$LDFLAGS" -o "$OUTPUT_PATH" "$CMD_DIR"
    
    if [ $? -eq 0 ]; then
        # Get file size
        if command -v stat >/dev/null 2>&1; then
            if [[ "$OSTYPE" == "darwin"* ]]; then
                SIZE=$(stat -f%z "$OUTPUT_PATH" 2>/dev/null || echo "unknown")
            else
                SIZE=$(stat -c%s "$OUTPUT_PATH" 2>/dev/null || echo "unknown")
            fi
            if [ "$SIZE" != "unknown" ]; then
                SIZE_MB=$(echo "scale=1; $SIZE/1024/1024" | bc 2>/dev/null || echo "$(($SIZE/1024/1024))")
                echo -e "${GREEN}✓ Built ${OUTPUT_PATH} (${SIZE_MB}MB)${RESET}"
            else
                echo -e "${GREEN}✓ Built ${OUTPUT_PATH}${RESET}"
            fi
        else
            echo -e "${GREEN}✓ Built ${OUTPUT_PATH}${RESET}"
        fi
    else
        echo -e "${RED}✗ Failed to build for ${GOOS}/${GOARCH}${RESET}"
        exit 1
    fi
done

echo ""
echo -e "${GREEN}Cross-platform build completed successfully!${RESET}"
echo -e "${BLUE}Binaries created in ${BUILD_DIR}/:${RESET}"
ls -la "${BUILD_DIR}/${BINARY_NAME}"-* 2>/dev/null || echo "No binaries found"
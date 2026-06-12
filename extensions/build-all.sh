#!/bin/bash

# ============================================================================
# Master Build Script - Build All Browser Extensions
# ============================================================================
# Builds Chrome, Firefox, Edge, and Safari extensions in one command
# ============================================================================

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

echo "========================================="
echo "DB Backup Manager - Build All Extensions"
echo "========================================="
echo ""

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Build counters
TOTAL=0
SUCCESS=0
FAILED=0

# Function to build an extension
build_extension() {
    local name=$1
    local dir=$2

    ((TOTAL++))

    echo "Building $name Extension..."
    echo "-----------------------------------"

    if [ ! -d "$dir" ]; then
        echo -e "${RED}✗${NC} Directory not found: $dir"
        ((FAILED++))
        echo ""
        return 1
    fi

    if [ ! -f "$dir/build.sh" ]; then
        echo -e "${YELLOW}⚠${NC} No build script found, skipping"
        echo ""
        return 0
    fi

    cd "$dir"

    if ./build.sh > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} $name extension built successfully"
        ((SUCCESS++))
    else
        echo -e "${RED}✗${NC} $name extension build failed"
        ((FAILED++))
    fi

    cd "$SCRIPT_DIR"
    echo ""
}

# Build each extension
build_extension "Chrome" "chrome"
build_extension "Firefox" "firefox"
build_extension "Edge" "edge"
build_extension "Safari" "safari"

# Generate icons if needed
echo "Checking for icons..."
echo "-----------------------------------"

if [ ! -f "chrome/icons/icon-48.png" ]; then
    echo -e "${YELLOW}⚠${NC} Icons not found. Generating..."

    if [ -f "chrome/icons/generate-icons.sh" ]; then
        cd chrome/icons
        ./generate-icons.sh > /dev/null 2>&1
        cd "$SCRIPT_DIR"

        if [ -f "chrome/icons/icon-48.png" ]; then
            echo -e "${GREEN}✓${NC} Icons generated successfully"

            # Copy icons to other browsers
            for browser in firefox edge safari; do
                if [ -d "$browser/icons" ]; then
                    cp chrome/icons/*.png "$browser/icons/" 2>/dev/null || true
                fi
            done
        else
            echo -e "${RED}✗${NC} Icon generation failed"
        fi
    else
        echo -e "${YELLOW}⚠${NC} Icon generation script not found"
        echo "  Please generate icons manually:"
        echo "  1. Open chrome/icons/generate-icons.html in a browser"
        echo "  2. Download the generated PNG files"
        echo "  3. Place them in chrome/icons/"
    fi
else
    echo -e "${GREEN}✓${NC} Icons already exist"
fi

echo ""

# Summary
echo "========================================="
echo "Build Summary"
echo "========================================="
echo "Total extensions: $TOTAL"
echo -e "${GREEN}Successful: $SUCCESS${NC}"

if [ $FAILED -gt 0 ]; then
    echo -e "${RED}Failed: $FAILED${NC}"
fi

echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All extensions built successfully!${NC}"
    echo ""
    echo "Next steps:"
    echo "  - Test each extension in its respective browser"
    echo "  - Package extensions: ./package-all.sh"
    echo "  - Submit to extension stores"
else
    echo -e "${RED}✗ Some builds failed. Check the output above.${NC}"
    exit 1
fi

echo ""
echo "Extension locations:"
echo "  Chrome:  ./chrome/"
echo "  Firefox: ./firefox/"
echo "  Edge:    ./edge/"
echo "  Safari:  ./safari/"

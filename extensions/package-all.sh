#!/bin/bash

# ============================================================================
# Master Package Script - Package All Browser Extensions
# ============================================================================
# Creates distributable packages for all browser extensions
# ============================================================================

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

echo "==========================================="
echo "DB Backup Manager - Package All Extensions"
echo "==========================================="
echo ""

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Package counters
TOTAL=0
SUCCESS=0
FAILED=0
SKIPPED=0

# Create dist directory
DIST_DIR="$SCRIPT_DIR/dist"
mkdir -p "$DIST_DIR"

echo "Output directory: $DIST_DIR"
echo ""

# Function to package an extension
package_extension() {
    local name=$1
    local dir=$2

    ((TOTAL++))

    echo "Packaging $name Extension..."
    echo "-----------------------------------"

    if [ ! -d "$dir" ]; then
        echo -e "${RED}✗${NC} Directory not found: $dir"
        ((FAILED++))
        echo ""
        return 1
    fi

    if [ ! -f "$dir/package.sh" ]; then
        echo -e "${YELLOW}⚠${NC} No package script found, skipping"
        ((SKIPPED++))
        echo ""
        return 0
    fi

    cd "$dir"

    if ./package.sh > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} $name extension packaged successfully"
        ((SUCCESS++))

        # Move package to dist directory
        for file in *.zip *.xpi; do
            if [ -f "$file" ]; then
                mv "$file" "$DIST_DIR/"
                echo "  → $DIST_DIR/$file"
            fi
        done
    else
        echo -e "${RED}✗${NC} $name extension packaging failed"
        ((FAILED++))
    fi

    cd "$SCRIPT_DIR"
    echo ""
}

# Build first
echo "Running build-all.sh first..."
if [ -f "build-all.sh" ]; then
    ./build-all.sh > /dev/null 2>&1
    echo -e "${GREEN}✓${NC} Build complete"
    echo ""
else
    echo -e "${YELLOW}⚠${NC} build-all.sh not found, assuming extensions are already built"
    echo ""
fi

# Package each extension
package_extension "Chrome" "chrome"
package_extension "Firefox" "firefox"
package_extension "Edge" "edge"

# Safari requires special handling (macOS only)
if [[ "$OSTYPE" == "darwin"* ]]; then
    package_extension "Safari" "safari"
else
    echo "Skipping Safari (requires macOS)"
    echo "-----------------------------------"
    echo -e "${YELLOW}⚠${NC} Safari extension can only be packaged on macOS"
    ((SKIPPED++))
    echo ""
fi

# Generate checksums
echo "Generating checksums..."
echo "-----------------------------------"

cd "$DIST_DIR"

if ls *.zip *.xpi 1> /dev/null 2>&1; then
    shasum -a 256 * > checksums.txt 2>/dev/null || true

    if [ -f "checksums.txt" ]; then
        echo -e "${GREEN}✓${NC} Checksums generated: checksums.txt"
    fi
fi

cd "$SCRIPT_DIR"
echo ""

# Summary
echo "==========================================="
echo "Package Summary"
echo "==========================================="
echo "Total extensions: $TOTAL"
echo -e "${GREEN}Packaged: $SUCCESS${NC}"

if [ $SKIPPED -gt 0 ]; then
    echo -e "${YELLOW}Skipped: $SKIPPED${NC}"
fi

if [ $FAILED -gt 0 ]; then
    echo -e "${RED}Failed: $FAILED${NC}"
fi

echo ""

# List packages
if ls "$DIST_DIR"/*.zip "$DIST_DIR"/*.xpi 1> /dev/null 2>&1; then
    echo "Packaged files:"
    echo ""
    for file in "$DIST_DIR"/*; do
        if [ -f "$file" ] && [ "$(basename "$file")" != "checksums.txt" ]; then
            filename=$(basename "$file")
            filesize=$(ls -lh "$file" | awk '{print $5}')
            echo "  $filename ($filesize)"
        fi
    done
else
    echo -e "${YELLOW}⚠${NC} No packages found"
fi

echo ""

# Show checksums
if [ -f "$DIST_DIR/checksums.txt" ]; then
    echo "Checksums:"
    echo ""
    cat "$DIST_DIR/checksums.txt" | sed 's/^/  /'
    echo ""
fi

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All extensions packaged successfully!${NC}"
    echo ""
    echo "Next steps:"
    echo ""
    echo "Chrome Web Store:"
    echo "  1. Go to: https://chrome.google.com/webstore/devconsole"
    echo "  2. Upload: $DIST_DIR/db-backup-manager-chrome-*.zip"
    echo ""
    echo "Firefox Add-ons:"
    echo "  1. Go to: https://addons.mozilla.org/developers/"
    echo "  2. Upload: $DIST_DIR/db-backup-manager-firefox-*.xpi"
    echo ""
    echo "Microsoft Edge Add-ons:"
    echo "  1. Go to: https://partner.microsoft.com/dashboard/microsoftedge/"
    echo "  2. Upload: $DIST_DIR/db-backup-manager-edge-*.zip"
    echo ""

    if [[ "$OSTYPE" == "darwin"* ]]; then
        echo "Safari (App Store):"
        echo "  1. Open Xcode project in ./safari/"
        echo "  2. Archive: Product > Archive"
        echo "  3. Submit to App Store Connect"
        echo ""
    fi
else
    echo -e "${RED}✗ Some packages failed. Check the output above.${NC}"
    exit 1
fi

echo "All packages are in: $DIST_DIR"

#!/bin/bash

# ============================================================================
# Chrome Extension Build Script
# ============================================================================
# Chrome is the source of truth for background/popup/options/content code,
# but the shared libraries live in ../shared (one canonical copy that every
# browser build consumes). Chrome loads them from chrome/shared via
# "../shared/*.js" relative imports, so this script materializes that folder
# by copying the canonical shared libraries into chrome/shared/.
#
# Run this before loading the unpacked extension (or via ../build-all.sh).
# ============================================================================

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
CHROME_DIR="$SCRIPT_DIR"
SHARED_DIR="$SCRIPT_DIR/../shared"

echo "Building Chrome Extension"
echo "========================="
echo ""

# Copy shared libraries into chrome/shared
echo "Copying shared libraries..."
mkdir -p "$CHROME_DIR/shared"
rm -f "$CHROME_DIR/shared"/*.js
if [ -d "$SHARED_DIR" ]; then
    cp "$SHARED_DIR"/*.js "$CHROME_DIR/shared/"
    echo "  ✓ Copied shared libraries into chrome/shared/"
else
    echo "  ✗ Warning: Shared directory not found: $SHARED_DIR"
    exit 1
fi

# Verify build
echo ""
echo "Verifying build..."

required_files=(
    "manifest.json"
    "shared/api.js"
    "shared/utils.js"
    "background/background.js"
    "popup/popup.html"
    "popup/popup.css"
    "popup/popup.js"
    "options/options.html"
    "options/options.css"
    "options/options.js"
    "content/content.js"
)

missing_files=()

for file in "${required_files[@]}"; do
    if [ ! -f "$CHROME_DIR/$file" ]; then
        missing_files+=("$file")
    fi
done

if [ ${#missing_files[@]} -eq 0 ]; then
    echo "  ✓ All required files present"
else
    echo "  ✗ Missing files:"
    for file in "${missing_files[@]}"; do
        echo "    - $file"
    done
    exit 1
fi

echo ""
echo "========================="
echo "Build complete!"
echo ""
echo "Next steps:"
echo "  1. Test in Chrome: chrome://extensions/ (enable Developer mode, Load unpacked -> this directory)"
echo "  2. Generate icons if needed: cd icons && ./generate-icons.sh"
echo "  3. Package for distribution: ./package.sh"

#!/bin/bash

# ============================================================================
# Firefox Extension Packaging Script
# ============================================================================
# Creates a distributable XPI file for Firefox Add-ons
# ============================================================================

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

VERSION=$(grep '"version"' manifest.json | head -1 | sed 's/.*"version": "\(.*\)".*/\1/')
OUTPUT_FILE="db-backup-manager-firefox-v${VERSION}.xpi"

echo "Firefox Extension Packager"
echo "=========================="
echo "Version: $VERSION"
echo ""

# Run build first
if [ -f "build.sh" ]; then
    echo "Running build script first..."
    ./build.sh
    echo ""
fi

# Check if required files exist
echo "Checking required files..."

required_files=(
    "manifest.json"
    "shared/api.js"
    "shared/utils.js"
    "background/background.js"
    "popup/popup.html"
    "options/options.html"
    "content/content.js"
)

missing_files=()

for file in "${required_files[@]}"; do
    if [ ! -f "$file" ]; then
        missing_files+=("$file")
    fi
done

if [ ${#missing_files[@]} -ne 0 ]; then
    echo "✗ Missing required files:"
    for file in "${missing_files[@]}"; do
        echo "  - $file"
    done
    echo ""
    echo "Please run ./build.sh first"
    exit 1
fi

echo "✓ All required files present"
echo ""

# Check for icons
if [ ! -f "icons/icon-48.png" ]; then
    echo "⚠ Warning: PNG icons not found"
    echo "The extension may not display correctly without icons"
    echo "Run: cd icons && ./generate-icons.sh"
    echo ""
fi

# Remove old package
if [ -f "$OUTPUT_FILE" ]; then
    echo "Removing old package..."
    rm "$OUTPUT_FILE"
fi

# Create XPI file (which is just a ZIP file with .xpi extension)
echo "Creating XPI package..."

zip -r -q "$OUTPUT_FILE" \
    manifest.json \
    shared/ \
    background/ \
    popup/ \
    options/ \
    content/ \
    icons/ \
    -x "*.DS_Store" \
    -x "*/.git/*" \
    -x "*/node_modules/*" \
    -x "*.sh" \
    -x "*/generate-icons.html" \
    -x "*/README.md" \
    -x "*.svg"

if [ -f "$OUTPUT_FILE" ]; then
    file_size=$(ls -lh "$OUTPUT_FILE" | awk '{print $5}')
    echo "✓ Package created: $OUTPUT_FILE ($file_size)"
else
    echo "✗ Failed to create package"
    exit 1
fi

echo ""
echo "Package contents:"
unzip -l "$OUTPUT_FILE" | tail -n +4 | head -n -2

echo ""
echo "=========================="
echo "Packaging complete!"
echo ""
echo "File: $OUTPUT_FILE"
echo "Size: $file_size"
echo ""
echo "Next steps:"
echo "  1. Test the XPI: Drag and drop into Firefox"
echo "  2. Submit to Firefox Add-ons: https://addons.mozilla.org/developers/"
echo "  3. Include source code if using minified libraries"
echo ""
echo "Validation:"
echo "  You can validate the extension at:"
echo "  https://addons.mozilla.org/developers/addon/submit/upload-listed"

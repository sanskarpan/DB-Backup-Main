#!/bin/bash

# ============================================================================
# Microsoft Edge Extension Build Script
# ============================================================================
# This script copies shared files from Chrome extension and prepares
# the Edge extension for packaging
# ============================================================================

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
EDGE_DIR="$SCRIPT_DIR"
CHROME_DIR="$SCRIPT_DIR/../chrome"
SHARED_DIR="$SCRIPT_DIR/../shared"

echo "Building Microsoft Edge Extension"
echo "=================================="
echo ""

# Clean previous build
echo "Cleaning previous build..."
rm -rf "$EDGE_DIR/shared"
rm -rf "$EDGE_DIR/background"
rm -rf "$EDGE_DIR/popup"
rm -rf "$EDGE_DIR/options"
rm -rf "$EDGE_DIR/content"
rm -rf "$EDGE_DIR/icons"

# Create directories
echo "Creating directories..."
mkdir -p "$EDGE_DIR/shared"
mkdir -p "$EDGE_DIR/background"
mkdir -p "$EDGE_DIR/popup"
mkdir -p "$EDGE_DIR/options"
mkdir -p "$EDGE_DIR/content"
mkdir -p "$EDGE_DIR/icons"

# Copy shared libraries
echo "Copying shared libraries..."
if [ -d "$SHARED_DIR" ]; then
    cp "$SHARED_DIR"/*.js "$EDGE_DIR/shared/"
    echo "  ✓ Copied shared libraries"
else
    echo "  ✗ Warning: Shared directory not found"
fi

# Copy background script
echo "Copying background script..."
if [ -f "$CHROME_DIR/background/background.js" ]; then
    cp "$CHROME_DIR/background/background.js" "$EDGE_DIR/background/"
    echo "  ✓ Copied background script"
else
    echo "  ✗ Warning: Background script not found"
fi

# Copy popup files
echo "Copying popup files..."
if [ -d "$CHROME_DIR/popup" ]; then
    cp "$CHROME_DIR/popup"/*.html "$EDGE_DIR/popup/" 2>/dev/null || true
    cp "$CHROME_DIR/popup"/*.css "$EDGE_DIR/popup/" 2>/dev/null || true
    cp "$CHROME_DIR/popup"/*.js "$EDGE_DIR/popup/" 2>/dev/null || true
    echo "  ✓ Copied popup files"
else
    echo "  ✗ Warning: Popup directory not found"
fi

# Copy options files
echo "Copying options files..."
if [ -d "$CHROME_DIR/options" ]; then
    cp "$CHROME_DIR/options"/*.html "$EDGE_DIR/options/" 2>/dev/null || true
    cp "$CHROME_DIR/options"/*.css "$EDGE_DIR/options/" 2>/dev/null || true
    cp "$CHROME_DIR/options"/*.js "$EDGE_DIR/options/" 2>/dev/null || true
    echo "  ✓ Copied options files"
else
    echo "  ✗ Warning: Options directory not found"
fi

# Copy content script
echo "Copying content script..."
if [ -f "$CHROME_DIR/content/content.js" ]; then
    cp "$CHROME_DIR/content/content.js" "$EDGE_DIR/content/"
    echo "  ✓ Copied content script"
else
    echo "  ✗ Warning: Content script not found"
fi

# Copy or generate icons
echo "Copying icons..."
if [ -d "$CHROME_DIR/icons" ]; then
    # Check if PNG icons exist
    if ls "$CHROME_DIR/icons"/*.png 1> /dev/null 2>&1; then
        cp "$CHROME_DIR/icons"/*.png "$EDGE_DIR/icons/"
        echo "  ✓ Copied PNG icons"
    else
        # Copy SVG and generation tools
        cp "$CHROME_DIR/icons/icon.svg" "$EDGE_DIR/icons/" 2>/dev/null || true
        cp "$CHROME_DIR/icons/generate-icons.sh" "$EDGE_DIR/icons/" 2>/dev/null || true
        cp "$CHROME_DIR/icons/generate-icons.html" "$EDGE_DIR/icons/" 2>/dev/null || true
        echo "  ⚠ PNG icons not found. Copied SVG and generation tools."
        echo "  Please run ./icons/generate-icons.sh to create PNG icons"
    fi
else
    echo "  ✗ Warning: Icons directory not found"
fi

# Create README for Edge
cat > "$EDGE_DIR/README.md" << 'EOF'
# DB Backup Manager - Microsoft Edge Extension

This is the Microsoft Edge version of the DB Backup Manager browser extension.

## Building

Run the build script to copy files from the Chrome extension:

```bash
./build.sh
```

## Testing

1. Open Microsoft Edge
2. Navigate to `edge://extensions/`
3. Enable "Developer mode" toggle
4. Click "Load unpacked"
5. Select this directory
6. The extension will be loaded

## Packaging

To create a distributable ZIP file:

```bash
./package.sh
```

This will create a `db-backup-manager-edge.zip` file that can be submitted to Microsoft Edge Add-ons.

## Edge-Specific Notes

- Uses Manifest V3 (same as Chrome)
- Fully compatible with Chrome extension code
- Edge supports the same APIs as Chrome (Chromium-based)
- No code changes needed from Chrome version

## Differences from Chrome Extension

- **Store**: Submitted to Microsoft Edge Add-ons instead of Chrome Web Store
- **Publisher**: Requires Microsoft Partner Center account
- **Store Listing**: Different store requirements and screenshots

The extension code is identical to Chrome version since Edge is Chromium-based.

## Submission to Microsoft Edge Add-ons

1. Create a Microsoft Partner Center account at https://partner.microsoft.com/
2. Enroll in the Microsoft Edge program ($9 one-time fee)
3. Package the extension using `./package.sh`
4. Submit the ZIP file for review
5. Provide store listing information (description, screenshots, etc.)
6. Wait for review (usually 1-3 days)

## Store Listing Requirements

- Detailed description (minimum 100 characters)
- At least 1 screenshot (1280x800 or 640x400)
- Icon (128x128)
- Privacy policy URL (if collecting data)
- Support URL

## Development

The Edge extension is 100% compatible with Chrome.
Files are copied from `../chrome/` during build.
No Edge-specific code changes are required.
EOF

echo "  ✓ Created README.md"

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
    if [ ! -f "$EDGE_DIR/$file" ]; then
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
fi

# Count files
echo ""
echo "Build Summary"
echo "============="
echo "Files created:"
find "$EDGE_DIR" -type f ! -path "*/.*" ! -name "build.sh" ! -name "package.sh" ! -name "README.md" | wc -l | xargs echo "  Total files:"

# Show directory structure
echo ""
echo "Directory structure:"
tree -L 2 -I 'node_modules|.git' "$EDGE_DIR" 2>/dev/null || find "$EDGE_DIR" -maxdepth 2 -not -path '*/\.*' | sed 's|[^/]*/|  |g'

echo ""
echo "=================================="
echo "Build complete!"
echo ""
echo "Next steps:"
echo "  1. Test in Edge: edge://extensions/"
echo "  2. Generate icons if needed: cd icons && ./generate-icons.sh"
echo "  3. Package for distribution: ./package.sh"

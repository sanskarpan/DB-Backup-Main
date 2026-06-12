#!/bin/bash

# ============================================================================
# Safari Web Extension Build Script
# ============================================================================
# This script copies shared files from Chrome extension and prepares
# the Safari extension for use in Xcode project
# ============================================================================

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
SAFARI_DIR="$SCRIPT_DIR"
CHROME_DIR="$SCRIPT_DIR/../chrome"
SHARED_DIR="$SCRIPT_DIR/../shared"

# Check if we have an Xcode project
XCODE_EXTENSION_DIR=""
if [ -d "$SAFARI_DIR/DBBackupManager/Extension" ]; then
    XCODE_EXTENSION_DIR="$SAFARI_DIR/DBBackupManager/Extension"
    echo "Found Xcode project Extension directory"
elif [ -d "$SAFARI_DIR/DB Backup Manager Extension/Resources" ]; then
    XCODE_EXTENSION_DIR="$SAFARI_DIR/DB Backup Manager Extension/Resources"
    echo "Found Xcode project Resources directory"
fi

echo "Building Safari Web Extension"
echo "============================="
echo ""

if [ -n "$XCODE_EXTENSION_DIR" ]; then
    echo "Building into Xcode project at: $XCODE_EXTENSION_DIR"
    BUILD_DIR="$XCODE_EXTENSION_DIR"
else
    echo "No Xcode project found. Building standalone files."
    echo "You'll need to create an Xcode project and copy these files."
    BUILD_DIR="$SAFARI_DIR"
fi

echo ""

# Clean previous build (only our managed directories)
echo "Cleaning previous build..."
rm -rf "$BUILD_DIR/shared"
rm -rf "$BUILD_DIR/background"
rm -rf "$BUILD_DIR/popup"
rm -rf "$BUILD_DIR/options"
rm -rf "$BUILD_DIR/content"
rm -rf "$BUILD_DIR/icons"

# Create directories
echo "Creating directories..."
mkdir -p "$BUILD_DIR/shared"
mkdir -p "$BUILD_DIR/background"
mkdir -p "$BUILD_DIR/popup"
mkdir -p "$BUILD_DIR/options"
mkdir -p "$BUILD_DIR/content"
mkdir -p "$BUILD_DIR/icons"

# Copy shared libraries
echo "Copying shared libraries..."
if [ -d "$SHARED_DIR" ]; then
    cp "$SHARED_DIR"/*.js "$BUILD_DIR/shared/"
    echo "  ✓ Copied shared libraries"
else
    echo "  ✗ Warning: Shared directory not found"
fi

# Copy background script
echo "Copying background script..."
if [ -f "$CHROME_DIR/background/background.js" ]; then
    cp "$CHROME_DIR/background/background.js" "$BUILD_DIR/background/"
    echo "  ✓ Copied background script"
else
    echo "  ✗ Warning: Background script not found"
fi

# Copy popup files
echo "Copying popup files..."
if [ -d "$CHROME_DIR/popup" ]; then
    cp "$CHROME_DIR/popup"/*.html "$BUILD_DIR/popup/" 2>/dev/null || true
    cp "$CHROME_DIR/popup"/*.css "$BUILD_DIR/popup/" 2>/dev/null || true
    cp "$CHROME_DIR/popup"/*.js "$BUILD_DIR/popup/" 2>/dev/null || true
    echo "  ✓ Copied popup files"
else
    echo "  ✗ Warning: Popup directory not found"
fi

# Copy options files
echo "Copying options files..."
if [ -d "$CHROME_DIR/options" ]; then
    cp "$CHROME_DIR/options"/*.html "$BUILD_DIR/options/" 2>/dev/null || true
    cp "$CHROME_DIR/options"/*.css "$BUILD_DIR/options/" 2>/dev/null || true
    cp "$CHROME_DIR/options"/*.js "$BUILD_DIR/options/" 2>/dev/null || true
    echo "  ✓ Copied options files"
else
    echo "  ✗ Warning: Options directory not found"
fi

# Copy content script
echo "Copying content script..."
if [ -f "$CHROME_DIR/content/content.js" ]; then
    cp "$CHROME_DIR/content/content.js" "$BUILD_DIR/content/"
    echo "  ✓ Copied content script"
else
    echo "  ✗ Warning: Content script not found"
fi

# Copy or generate icons
echo "Copying icons..."
if [ -d "$CHROME_DIR/icons" ]; then
    # Check if PNG icons exist
    if ls "$CHROME_DIR/icons"/*.png 1> /dev/null 2>&1; then
        cp "$CHROME_DIR/icons"/*.png "$BUILD_DIR/icons/"
        echo "  ✓ Copied PNG icons"
    else
        # Copy SVG and generation tools
        cp "$CHROME_DIR/icons/icon.svg" "$BUILD_DIR/icons/" 2>/dev/null || true
        cp "$CHROME_DIR/icons/generate-icons.sh" "$BUILD_DIR/icons/" 2>/dev/null || true
        cp "$CHROME_DIR/icons/generate-icons.html" "$BUILD_DIR/icons/" 2>/dev/null || true
        echo "  ⚠ PNG icons not found. Copied SVG and generation tools."
        echo "  Please run ./icons/generate-icons.sh to create PNG icons"
    fi
else
    echo "  ✗ Warning: Icons directory not found"
fi

# Copy manifest
echo "Copying manifest..."
if [ -f "$SAFARI_DIR/manifest.json" ]; then
    cp "$SAFARI_DIR/manifest.json" "$BUILD_DIR/"
    echo "  ✓ Copied Safari-specific manifest"
fi

# Create Safari-specific polyfill
cat > "$BUILD_DIR/shared/safari-polyfill.js" << 'EOF'
/**
 * Safari Polyfill for Web Extensions
 * Safari natively supports the 'browser' namespace with promises
 */

// Safari already provides 'browser' API with promises
// Our shared utils.js handles browser detection
// No additional polyfill needed for Safari

console.log('[Safari Polyfill] Loaded');
EOF

echo "  ✓ Created Safari polyfill"

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
    if [ ! -f "$BUILD_DIR/$file" ]; then
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
echo "Build directory: $BUILD_DIR"
echo "Files created:"
find "$BUILD_DIR" -type f ! -path "*/.*" ! -name "build.sh" ! -name "package.sh" ! -name "README.md" ! -name "manifest.json" | wc -l | xargs echo "  Extension files:"

echo ""
echo "============================="
echo "Build complete!"
echo ""

if [ -n "$XCODE_EXTENSION_DIR" ]; then
    echo "Files copied to Xcode project."
    echo ""
    echo "Next steps:"
    echo "  1. Open the Xcode project"
    echo "  2. Build and run (⌘R)"
    echo "  3. Safari will launch with the extension"
    echo "  4. Enable in Safari Preferences > Extensions"
else
    echo "No Xcode project found."
    echo ""
    echo "Next steps:"
    echo "  1. Run ./convert-to-safari.sh to create Xcode project"
    echo "  2. Or manually create Safari Extension project in Xcode"
    echo "  3. Copy the extension files to the Xcode project"
    echo "  4. Build and run in Xcode"
fi

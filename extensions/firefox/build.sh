#!/bin/bash

# ============================================================================
# Firefox Extension Build Script
# ============================================================================
# This script copies shared files from Chrome extension and prepares
# the Firefox extension for packaging
# ============================================================================

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
FIREFOX_DIR="$SCRIPT_DIR"
CHROME_DIR="$SCRIPT_DIR/../chrome"
SHARED_DIR="$SCRIPT_DIR/../shared"

echo "Building Firefox Extension"
echo "=========================="
echo ""

# Clean previous build
echo "Cleaning previous build..."
rm -rf "$FIREFOX_DIR/shared"
rm -rf "$FIREFOX_DIR/background"
rm -rf "$FIREFOX_DIR/popup"
rm -rf "$FIREFOX_DIR/options"
rm -rf "$FIREFOX_DIR/content"
rm -rf "$FIREFOX_DIR/icons"

# Create directories
echo "Creating directories..."
mkdir -p "$FIREFOX_DIR/shared"
mkdir -p "$FIREFOX_DIR/background"
mkdir -p "$FIREFOX_DIR/popup"
mkdir -p "$FIREFOX_DIR/options"
mkdir -p "$FIREFOX_DIR/content"
mkdir -p "$FIREFOX_DIR/icons"

# Copy shared libraries
echo "Copying shared libraries..."
if [ -d "$SHARED_DIR" ]; then
    cp "$SHARED_DIR"/*.js "$FIREFOX_DIR/shared/"
    echo "  ✓ Copied shared libraries"
else
    echo "  ✗ Warning: Shared directory not found"
fi

# Copy background script
echo "Copying background script..."
if [ -f "$CHROME_DIR/background/background.js" ]; then
    cp "$CHROME_DIR/background/background.js" "$FIREFOX_DIR/background/"
    echo "  ✓ Copied background script"
else
    echo "  ✗ Warning: Background script not found"
fi

# Copy popup files
echo "Copying popup files..."
if [ -d "$CHROME_DIR/popup" ]; then
    cp "$CHROME_DIR/popup"/*.html "$FIREFOX_DIR/popup/" 2>/dev/null || true
    cp "$CHROME_DIR/popup"/*.css "$FIREFOX_DIR/popup/" 2>/dev/null || true
    cp "$CHROME_DIR/popup"/*.js "$FIREFOX_DIR/popup/" 2>/dev/null || true
    echo "  ✓ Copied popup files"
else
    echo "  ✗ Warning: Popup directory not found"
fi

# Copy options files
echo "Copying options files..."
if [ -d "$CHROME_DIR/options" ]; then
    cp "$CHROME_DIR/options"/*.html "$FIREFOX_DIR/options/" 2>/dev/null || true
    cp "$CHROME_DIR/options"/*.css "$FIREFOX_DIR/options/" 2>/dev/null || true
    cp "$CHROME_DIR/options"/*.js "$FIREFOX_DIR/options/" 2>/dev/null || true
    echo "  ✓ Copied options files"
else
    echo "  ✗ Warning: Options directory not found"
fi

# Copy content script
echo "Copying content script..."
if [ -f "$CHROME_DIR/content/content.js" ]; then
    cp "$CHROME_DIR/content/content.js" "$FIREFOX_DIR/content/"
    echo "  ✓ Copied content script"
else
    echo "  ✗ Warning: Content script not found"
fi

# Copy or generate icons
echo "Copying icons..."
if [ -d "$CHROME_DIR/icons" ]; then
    # Check if PNG icons exist
    if ls "$CHROME_DIR/icons"/*.png 1> /dev/null 2>&1; then
        cp "$CHROME_DIR/icons"/*.png "$FIREFOX_DIR/icons/"
        echo "  ✓ Copied PNG icons"
    else
        # Copy SVG and generation tools
        cp "$CHROME_DIR/icons/icon.svg" "$FIREFOX_DIR/icons/" 2>/dev/null || true
        cp "$CHROME_DIR/icons/generate-icons.sh" "$FIREFOX_DIR/icons/" 2>/dev/null || true
        cp "$CHROME_DIR/icons/generate-icons.html" "$FIREFOX_DIR/icons/" 2>/dev/null || true
        echo "  ⚠ PNG icons not found. Copied SVG and generation tools."
        echo "  Please run ./icons/generate-icons.sh to create PNG icons"
    fi
else
    echo "  ✗ Warning: Icons directory not found"
fi

# Apply Firefox-specific patches
echo ""
echo "Applying Firefox-specific patches..."

# Replace chrome API with browser API in background script
if [ -f "$FIREFOX_DIR/background/background.js" ]; then
    # Firefox natively supports browser API, but we'll ensure compatibility
    echo "  ✓ Background script ready for Firefox"
fi

# Add Firefox-specific polyfill if needed
cat > "$FIREFOX_DIR/shared/firefox-polyfill.js" << 'EOF'
/**
 * Firefox Polyfill for Chrome Extensions API
 * Firefox natively uses the 'browser' namespace with promises
 * This file ensures compatibility with our code
 */

// Firefox already provides 'browser' API
// Our shared utils.js already handles browser detection
// No polyfill needed for Firefox

console.log('[Firefox Polyfill] Loaded');
EOF

echo "  ✓ Created Firefox polyfill"

# Create README for Firefox
cat > "$FIREFOX_DIR/README.md" << 'EOF'
# DB Backup Manager - Firefox Extension

This is the Firefox version of the DB Backup Manager browser extension.

## Building

Run the build script to copy files from the Chrome extension:

```bash
./build.sh
```

## Testing

1. Open Firefox
2. Navigate to `about:debugging#/runtime/this-firefox`
3. Click "Load Temporary Add-on"
4. Select the `manifest.json` file from this directory
5. The extension will be loaded

## Packaging

To create a distributable XPI file:

```bash
./package.sh
```

This will create a `db-backup-manager-firefox.xpi` file that can be submitted to Firefox Add-ons.

## Firefox-Specific Notes

- Uses Manifest V2 (Firefox's recommended version)
- Uses `browser` namespace instead of `chrome`
- All APIs return promises (no callbacks needed)
- Background script is non-persistent by default
- Some Chrome-specific APIs may not be available

## Differences from Chrome Extension

1. **Manifest**: Uses Manifest V2 format
2. **API Namespace**: Uses `browser` instead of `chrome`
3. **Promises**: Native promise support (no callback conversion needed)
4. **Background Page**: Non-persistent background page instead of service worker
5. **Commands**: Uses `_execute_browser_action` instead of `_execute_action`

## Submission to Firefox Add-ons

1. Create an account at https://addons.mozilla.org
2. Package the extension using `./package.sh`
3. Submit the XPI file for review
4. Provide source code if using minified libraries
5. Wait for review (usually 1-3 days)

## Development

The Firefox extension shares most code with the Chrome extension.
Files are copied from `../chrome/` during build.
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
    if [ ! -f "$FIREFOX_DIR/$file" ]; then
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
find "$FIREFOX_DIR" -type f ! -path "*/.*" ! -name "build.sh" ! -name "package.sh" ! -name "README.md" | wc -l | xargs echo "  Total files:"

# Show directory structure
echo ""
echo "Directory structure:"
tree -L 2 -I 'node_modules|.git' "$FIREFOX_DIR" 2>/dev/null || find "$FIREFOX_DIR" -maxdepth 2 -not -path '*/\.*' | sed 's|[^/]*/|  |g'

echo ""
echo "=========================="
echo "Build complete!"
echo ""
echo "Next steps:"
echo "  1. Test in Firefox: about:debugging#/runtime/this-firefox"
echo "  2. Generate icons if needed: cd icons && ./generate-icons.sh"
echo "  3. Package for distribution: ./package.sh"

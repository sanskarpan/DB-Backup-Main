#!/bin/bash

# ============================================================================
# Safari Web Extension Converter
# ============================================================================
# Uses Apple's xcrun safari-web-extension-converter to convert Chrome extension
# ============================================================================

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
CHROME_DIR="$SCRIPT_DIR/../chrome"
SAFARI_DIR="$SCRIPT_DIR"

echo "Safari Web Extension Converter"
echo "==============================="
echo ""

# Check if on macOS
if [[ "$OSTYPE" != "darwin"* ]]; then
    echo "✗ Error: This script requires macOS"
    echo "  Safari Web Extensions can only be developed on macOS with Xcode"
    exit 1
fi

# Check if Xcode is installed
if ! command -v xcrun &> /dev/null; then
    echo "✗ Error: Xcode command line tools not found"
    echo ""
    echo "Please install Xcode:"
    echo "  1. Install Xcode from the Mac App Store"
    echo "  2. Run: xcode-select --install"
    echo "  3. Run this script again"
    exit 1
fi

# Check if Chrome extension exists
if [ ! -d "$CHROME_DIR" ]; then
    echo "✗ Error: Chrome extension directory not found at $CHROME_DIR"
    exit 1
fi

if [ ! -f "$CHROME_DIR/manifest.json" ]; then
    echo "✗ Error: Chrome extension manifest.json not found"
    exit 1
fi

echo "✓ Found Chrome extension at $CHROME_DIR"
echo ""

# Create temporary directory for conversion
TEMP_DIR=$(mktemp -d)
echo "Creating temporary directory: $TEMP_DIR"

# Copy Chrome extension to temp directory
cp -r "$CHROME_DIR" "$TEMP_DIR/chrome-extension"

# Run the converter
echo ""
echo "Running Apple's safari-web-extension-converter..."
echo "This may take a few moments..."
echo ""

xcrun safari-web-extension-converter \
    "$TEMP_DIR/chrome-extension" \
    --app-name "DB Backup Manager" \
    --bundle-identifier "com.dbbackup.manager" \
    --macos-only \
    --project-location "$SAFARI_DIR"

if [ $? -eq 0 ]; then
    echo ""
    echo "✓ Conversion successful!"
    echo ""
    echo "Xcode project created at: $SAFARI_DIR"
else
    echo ""
    echo "✗ Conversion failed"
    echo "  Check the error messages above"
    exit 1
fi

# Clean up temp directory
rm -rf "$TEMP_DIR"

# Show next steps
echo ""
echo "==============================="
echo "Next Steps:"
echo ""
echo "1. Open the Xcode project:"
echo "   open '$SAFARI_DIR/DB Backup Manager.xcodeproj'"
echo ""
echo "2. In Xcode:"
echo "   - Select your development team in Signing & Capabilities"
echo "   - Build and run the project (⌘R)"
echo "   - Safari will launch with the extension"
echo ""
echo "3. In Safari:"
echo "   - Open Preferences > Extensions"
echo "   - Enable 'DB Backup Manager Extension'"
echo "   - Grant any required permissions"
echo ""
echo "4. For distribution:"
echo "   - Archive the project in Xcode"
echo "   - Submit to App Store Connect"
echo ""
echo "Useful commands:"
echo "  - Rebuild extension files: ./build.sh"
echo "  - Open Xcode project: open *.xcodeproj"
echo "  - Clean Xcode build: rm -rf ~/Library/Developer/Xcode/DerivedData"
echo ""
echo "Documentation:"
echo "  - Read README.md for detailed instructions"
echo "  - Safari Web Extensions: https://developer.apple.com/safari/extensions/"

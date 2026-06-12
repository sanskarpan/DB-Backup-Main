#!/bin/bash

# ============================================================================
# Safari Web Extension Packaging Script
# ============================================================================
# Archives the Xcode project for App Store submission
# ============================================================================

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

echo "Safari Web Extension Packager"
echo "=============================="
echo ""

# Check if on macOS
if [[ "$OSTYPE" != "darwin"* ]]; then
    echo "✗ Error: This script requires macOS"
    exit 1
fi

# Check if Xcode is installed
if ! command -v xcodebuild &> /dev/null; then
    echo "✗ Error: xcodebuild not found"
    echo "  Please install Xcode from the Mac App Store"
    exit 1
fi

# Find Xcode project
XCODE_PROJECT=$(find . -name "*.xcodeproj" -maxdepth 2 | head -1)

if [ -z "$XCODE_PROJECT" ]; then
    echo "✗ Error: No Xcode project found"
    echo ""
    echo "Please run one of these first:"
    echo "  - ./convert-to-safari.sh (to create project from Chrome extension)"
    echo "  - Create a Safari Extension project manually in Xcode"
    exit 1
fi

echo "✓ Found Xcode project: $XCODE_PROJECT"
echo ""

# Get project name
PROJECT_NAME=$(basename "$XCODE_PROJECT" .xcodeproj)

# Clean build
echo "Cleaning previous builds..."
xcodebuild clean \
    -project "$XCODE_PROJECT" \
    -scheme "$PROJECT_NAME" \
    > /dev/null 2>&1

echo "✓ Clean complete"
echo ""

# Build for release
echo "Building for release..."
echo "This may take a few minutes..."
echo ""

xcodebuild build \
    -project "$XCODE_PROJECT" \
    -scheme "$PROJECT_NAME" \
    -configuration Release \
    -derivedDataPath "./build"

if [ $? -eq 0 ]; then
    echo ""
    echo "✓ Build successful!"
else
    echo ""
    echo "✗ Build failed"
    echo "  Check the error messages above"
    exit 1
fi

# Find the built app
APP_PATH=$(find ./build -name "*.app" -type d | head -1)

if [ -z "$APP_PATH" ]; then
    echo "✗ Error: Built app not found"
    exit 1
fi

echo "✓ Built app: $APP_PATH"
echo ""

# Create archive for distribution
echo "Creating distribution archive..."

ARCHIVE_NAME="${PROJECT_NAME}.xcarchive"
EXPORT_PATH="./build/archive"

mkdir -p "$EXPORT_PATH"

xcodebuild archive \
    -project "$XCODE_PROJECT" \
    -scheme "$PROJECT_NAME" \
    -archivePath "$EXPORT_PATH/$ARCHIVE_NAME" \
    > /dev/null 2>&1

if [ $? -eq 0 ]; then
    echo "✓ Archive created: $EXPORT_PATH/$ARCHIVE_NAME"
else
    echo "⚠ Archive creation failed (this is normal for development builds)"
    echo "  You can still test the app from: $APP_PATH"
fi

# Get app size
if [ -d "$APP_PATH" ]; then
    APP_SIZE=$(du -sh "$APP_PATH" | cut -f1)
    echo ""
    echo "=============================="
    echo "Build Complete!"
    echo ""
    echo "App location: $APP_PATH"
    echo "App size: $APP_SIZE"
    echo ""
fi

echo "Next steps:"
echo ""
echo "For Testing:"
echo "  1. Run the app: open '$APP_PATH'"
echo "  2. Enable extension in Safari Preferences > Extensions"
echo "  3. Test all functionality"
echo ""
echo "For App Store Distribution:"
echo "  1. Open Xcode"
echo "  2. Window > Organizer"
echo "  3. Select your archive"
echo "  4. Click 'Distribute App'"
echo "  5. Choose 'App Store Connect'"
echo "  6. Follow the wizard to upload"
echo ""
echo "Or use Xcode to archive:"
echo "  1. Open: open '$XCODE_PROJECT'"
echo "  2. Product > Archive"
echo "  3. Distribute to App Store"
echo ""
echo "Requirements for App Store:"
echo "  - Valid Apple Developer account (\$99/year)"
echo "  - App Store Connect access"
echo "  - Privacy policy URL (if collecting data)"
echo "  - macOS screenshots (1280x800 or higher)"
echo "  - App icon (512x512 and 1024x1024)"

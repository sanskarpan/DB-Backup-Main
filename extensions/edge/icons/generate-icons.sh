#!/bin/bash

# ============================================================================
# Icon Generation Script for DB Backup Manager Extension
# ============================================================================

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

echo "DB Backup Manager - Icon Generator"
echo "===================================="
echo ""

# Check if icon.svg exists
if [ ! -f "icon.svg" ]; then
    echo "Error: icon.svg not found in current directory"
    exit 1
fi

# Detect available tools
HAS_IMAGEMAGICK=false
HAS_INKSCAPE=false
HAS_RSVG=false

if command -v magick &> /dev/null || command -v convert &> /dev/null; then
    HAS_IMAGEMAGICK=true
    echo "✓ ImageMagick detected"
fi

if command -v inkscape &> /dev/null; then
    HAS_INKSCAPE=true
    echo "✓ Inkscape detected"
fi

if command -v rsvg-convert &> /dev/null; then
    HAS_RSVG=true
    echo "✓ rsvg-convert detected"
fi

if [ "$HAS_IMAGEMAGICK" = false ] && [ "$HAS_INKSCAPE" = false ] && [ "$HAS_RSVG" = false ]; then
    echo ""
    echo "Error: No suitable tool found for converting SVG to PNG"
    echo ""
    echo "Please install one of the following:"
    echo "  - ImageMagick: brew install imagemagick (macOS) or apt-get install imagemagick (Linux)"
    echo "  - Inkscape: brew install --cask inkscape (macOS) or apt-get install inkscape (Linux)"
    echo "  - librsvg: brew install librsvg (macOS) or apt-get install librsvg2-bin (Linux)"
    echo ""
    echo "Alternatively, use the generate-icons.html file in a web browser"
    exit 1
fi

echo ""

# Define sizes
SIZES=(16 32 48 128)

# Generate icons
echo "Generating icons..."
echo ""

for size in "${SIZES[@]}"; do
    output_file="icon-${size}.png"

    if [ "$HAS_IMAGEMAGICK" = true ]; then
        # Try modern 'magick' command first, fall back to 'convert'
        if command -v magick &> /dev/null; then
            magick icon.svg -resize ${size}x${size} -background none "$output_file"
        else
            convert icon.svg -resize ${size}x${size} -background none "$output_file"
        fi
        echo "  ✓ Generated $output_file (using ImageMagick)"
    elif [ "$HAS_INKSCAPE" = true ]; then
        inkscape icon.svg -w ${size} -h ${size} -o "$output_file" &> /dev/null
        echo "  ✓ Generated $output_file (using Inkscape)"
    elif [ "$HAS_RSVG" = true ]; then
        rsvg-convert -w ${size} -h ${size} icon.svg -o "$output_file"
        echo "  ✓ Generated $output_file (using rsvg-convert)"
    fi
done

echo ""
echo "===================================="
echo "All icons generated successfully!"
echo ""
echo "Generated files:"
for size in "${SIZES[@]}"; do
    output_file="icon-${size}.png"
    if [ -f "$output_file" ]; then
        file_size=$(ls -lh "$output_file" | awk '{print $5}')
        echo "  - $output_file ($file_size)"
    fi
done

echo ""
echo "Next steps:"
echo "  1. Verify the icons look correct"
echo "  2. Optionally optimize them with: optipng icon-*.png"
echo "  3. The extension is ready to use!"

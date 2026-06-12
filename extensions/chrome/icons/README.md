# Extension Icons

This directory contains the icon files for the DB Backup Manager Chrome extension.

## Icon Sizes Required

The extension requires the following icon sizes:

- **16x16** - Favicon on extension pages
- **32x32** - Windows computers often require this size
- **48x48** - Used in the extensions management page
- **128x128** - Used during installation and in the Chrome Web Store

## Generating Icons

### Method 1: Using the HTML Generator (Recommended)

1. Open `generate-icons.html` in a web browser
2. The page will automatically generate all icon sizes from `icon.svg`
3. Click "Download All Icons" to download all sizes
4. Move the downloaded files to this directory

### Method 2: Using Command Line Tools

#### Using ImageMagick

```bash
# Install ImageMagick (if not already installed)
# macOS: brew install imagemagick
# Ubuntu: sudo apt-get install imagemagick
# Windows: Download from https://imagemagick.org/

# Convert SVG to PNG at different sizes
magick icon.svg -resize 16x16 icon-16.png
magick icon.svg -resize 32x32 icon-32.png
magick icon.svg -resize 48x48 icon-48.png
magick icon.svg -resize 128x128 icon-128.png
```

#### Using Inkscape

```bash
# Install Inkscape (if not already installed)
# macOS: brew install --cask inkscape
# Ubuntu: sudo apt-get install inkscape
# Windows: Download from https://inkscape.org/

# Convert SVG to PNG at different sizes
inkscape icon.svg -w 16 -h 16 -o icon-16.png
inkscape icon.svg -w 32 -h 32 -o icon-32.png
inkscape icon.svg -w 48 -h 48 -o icon-48.png
inkscape icon.svg -w 128 -h 128 -o icon-128.png
```

#### Using Node.js with Sharp

```bash
# Install dependencies
npm install sharp

# Create convert.js
cat > convert.js << 'EOF'
const sharp = require('sharp');
const fs = require('fs');

const sizes = [16, 32, 48, 128];
const svgBuffer = fs.readFileSync('icon.svg');

Promise.all(
  sizes.map(size =>
    sharp(svgBuffer)
      .resize(size, size)
      .png()
      .toFile(`icon-${size}.png`)
  )
).then(() => {
  console.log('All icons generated successfully!');
}).catch(err => {
  console.error('Error generating icons:', err);
});
EOF

# Run the script
node convert.js
```

### Method 3: Online Tools

You can also use online SVG to PNG converters:

1. Go to https://cloudconvert.com/svg-to-png
2. Upload `icon.svg`
3. Set the output size (16, 32, 48, or 128)
4. Download the PNG file
5. Repeat for all sizes

## Icon Design

The icon design represents:

- **Database Cylinder**: The core database backup functionality
- **Download Arrow**: The backup/download action
- **Blue Gradient**: Professional, trustworthy appearance
- **Green Circle**: Success and reliability

### Color Palette

- Primary Blue: `#3b82f6` to `#2563eb` (gradient)
- Success Green: `#10b981`
- White/Light: `#ffffff` to `#e0e7ff`

## File Structure

```
icons/
├── icon.svg              # Master SVG icon (source file)
├── generate-icons.html   # HTML tool to generate PNGs
├── README.md            # This file
├── icon-16.png          # Generated 16x16 icon
├── icon-32.png          # Generated 32x32 icon
├── icon-48.png          # Generated 48x48 icon
└── icon-128.png         # Generated 128x128 icon
```

## Notes

- The SVG file (`icon.svg`) is the source of truth - edit this file to change the icon design
- After editing the SVG, regenerate all PNG files
- PNG files should be optimized for web (use tools like TinyPNG or ImageOptim)
- Icons should have transparent backgrounds
- Use consistent padding to ensure the icon looks good at all sizes

## Customization

To customize the icon:

1. Open `icon.svg` in a text editor or vector graphics editor (Inkscape, Adobe Illustrator, Figma)
2. Modify colors, shapes, or add new elements
3. Save the file
4. Regenerate all PNG files using one of the methods above

## Troubleshooting

**Problem**: Icons look blurry at small sizes
- Solution: Ensure the SVG has clean, simple shapes without too much detail

**Problem**: Colors don't match the design
- Solution: Check the gradient definitions in the SVG file

**Problem**: Transparent background not working
- Solution: Make sure the SVG doesn't have a background rectangle

**Problem**: Icons not showing in the extension
- Solution: Check that the file names match exactly: `icon-16.png`, `icon-32.png`, `icon-48.png`, `icon-128.png`

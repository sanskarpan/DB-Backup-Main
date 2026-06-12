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

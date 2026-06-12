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

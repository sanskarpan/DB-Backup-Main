# Privacy Policy for DB Backup Manager Browser Extension

**Last Updated:** January 13, 2026
**Effective Date:** January 13, 2026

## Introduction

DB Backup Manager ("we", "our", or "the extension") is committed to protecting your privacy. This Privacy Policy explains how our browser extension collects, uses, and protects your information.

## Information We Collect

### Information You Provide

**API Configuration:**
- API server URL
- API authentication key

This information is stored locally in your browser's encrypted storage and is never transmitted to us.

**Settings and Preferences:**
- Sync interval
- Notification preferences
- Display options
- Keyboard shortcuts

These settings are stored locally on your device.

### Automatically Collected Information

**Usage Analytics (Optional):**
If you choose to enable analytics, we collect:
- Feature usage (which features you use)
- Error reports (to improve stability)
- Performance metrics (to optimize speed)
- Extension version
- Browser type
- Operating system type

**What We DON'T Collect:**
- Personal information (name, email, address)
- Database names or content
- Database credentials
- URLs you visit
- Files you create or download
- Any personally identifiable information (PII)

## How We Use Information

### API Configuration
- To connect to your DB Backup Manager server
- Stored locally, encrypted by your browser
- Never leaves your device except to call your API server

### Usage Analytics
- To understand which features are used
- To identify and fix bugs
- To improve performance
- To prioritize new features

All analytics data is:
- Anonymous (no user identification)
- Aggregated (no individual tracking)
- Optional (you can opt-out anytime)

## Data Storage

### Local Storage
All extension data is stored locally using your browser's storage API:
- Chrome: `chrome.storage.local` (encrypted)
- Firefox: `browser.storage.local` (encrypted)
- Edge: `chrome.storage.local` (encrypted)
- Safari: Browser encrypted storage

### Remote Storage
The extension does NOT store any data on our servers. All data remains on:
1. Your device (browser local storage)
2. Your DB Backup Manager API server (that you control)

## Data Sharing

We do NOT:
- Sell your data
- Share your data with third parties
- Use your data for advertising
- Track you across websites
- Collect personal information

The extension only communicates with:
- Your configured DB Backup Manager API server
- (Optional) Our anonymous analytics endpoint (if you opt-in)

## Third-Party Services

### Your API Server
The extension connects to your DB Backup Manager API server at the URL you configure. We have no control over or access to this server.

### Browser Storage
We use the browser's built-in storage API, which is managed by:
- Google (Chrome)
- Mozilla (Firefox)
- Microsoft (Edge)
- Apple (Safari)

These companies have their own privacy policies.

## Analytics Opt-Out

Analytics are OPTIONAL and OFF by default.

**To disable analytics:**
1. Open extension settings
2. Navigate to "Privacy" section
3. Uncheck "Enable analytics"
4. Save settings

**To view collected data:**
1. Open extension settings
2. Navigate to "Privacy" section
3. Click "View Analytics Data"
4. Export or delete as desired

**To delete all analytics data:**
1. Open extension settings
2. Navigate to "Privacy" section
3. Click "Clear All Data"

## Data Retention

### Local Data
- Stored indefinitely on your device
- Cleared when you uninstall the extension
- Can be manually cleared in settings

### Analytics Data
- Kept for 90 days
- Automatically purged after 90 days
- Can be manually cleared anytime

## Your Rights

You have the right to:

### Access
- View all data stored by the extension
- Export your data in JSON format
- See exactly what analytics are collected

### Deletion
- Clear all extension data
- Delete analytics data
- Uninstall the extension (removes all data)

### Opt-Out
- Disable analytics collection
- Disable specific features
- Control all permissions

### Portability
- Export your settings
- Transfer to another device
- Backup your configuration

## Children's Privacy

This extension is not directed to children under 13. We do not knowingly collect information from children.

## Security

We implement security measures:

### In-Transit
- All API calls use HTTPS
- No unencrypted communication
- Certificate validation enforced

### At-Rest
- Browser's encrypted storage
- No plaintext credentials
- Secure key management

### Permissions
- Request minimal permissions
- Explain why each permission is needed
- Only access what's necessary

## Changes to Privacy Policy

We may update this Privacy Policy:
- When adding new features
- To comply with legal requirements
- To improve clarity

**Notification of Changes:**
- Updated "Last Updated" date
- Changelog in extension update notes
- Notification on first run after update

**Your Continued Use:**
Using the extension after changes constitutes acceptance of the new policy.

## Compliance

### GDPR (European Union)
- No personal data collection
- Anonymous user IDs only
- Opt-out capability
- Data export/deletion
- Transparent data collection

### CCPA (California)
- No sale of personal information
- Right to know what's collected
- Right to delete
- Right to opt-out

### Other Regulations
We strive to comply with all applicable privacy laws.

## Contact Us

For privacy concerns or questions:

**Email:** privacy@dbbackup.example.com
**GitHub:** https://github.com/yourusername/db-backup/issues
**Website:** https://dbbackup.example.com/privacy

**Response Time:** We aim to respond within 48 hours.

## Open Source

DB Backup Manager is open source. You can:
- Review the source code
- Verify privacy claims
- Contribute improvements
- Report security issues

**Repository:** https://github.com/yourusername/db-backup

## Data Breach Notification

In the unlikely event of a data breach:
- We will notify affected users within 72 hours
- We will explain what data was affected
- We will provide remediation steps
- We will report to relevant authorities

**Note:** Since all data is stored locally and we don't collect personal information, the risk of a data breach affecting users is minimal.

## Cookies

This extension does NOT use:
- Cookies
- Tracking pixels
- Fingerprinting
- Cross-site tracking

## Permission Justification

The extension requests these permissions:

### storage
**Why:** Store settings and configuration
**Data:** API URL, preferences, analytics (opt-in)
**Scope:** Local device only

### alarms
**Why:** Schedule periodic sync and monitoring
**Data:** None
**Scope:** Browser alarms API

### notifications
**Why:** Alert you of backup status
**Data:** Backup status, alerts
**Scope:** Local notifications only

### contextMenus
**Why:** Right-click menu integration
**Data:** None
**Scope:** Extension menu items only

### tabs
**Why:** Detect database management tools
**Data:** Current tab URL (not stored)
**Scope:** Active tab only

### activeTab
**Why:** Inject content script on database tools
**Data:** None
**Scope:** Current tab only, when activated

### host_permissions (<all_urls>)
**Why:** Connect to your API server (any domain)
**Data:** API requests/responses
**Scope:** Your configured API server only

## Analytics Details

If you opt-in to analytics, we collect:

### Events
- Extension installed/updated
- Features used (e.g., "Quick Backup clicked")
- Errors encountered
- Settings changed

### Metadata
- Timestamp
- Extension version
- Browser type (Chrome, Firefox, etc.)
- OS type (Windows, macOS, Linux)

### Anonymous ID
- Randomly generated UUID
- Not linked to any personal information
- Cannot be used to identify you
- Can be reset anytime

### What's Sanitized
- URLs (domain only, path removed)
- Database names (removed)
- API keys (removed)
- Email addresses (removed)
- Any PII (removed)

## Your API Server

The extension connects to YOUR API server:
- We don't control your server
- We don't have access to your server
- We don't know what your server logs
- Your server's privacy policy applies

**Your Responsibility:**
Review the privacy policy of your DB Backup Manager API server provider.

## Uninstall

When you uninstall the extension:
- All local data is removed
- Settings are deleted
- Analytics data is purged
- No data remains on your device

**To completely remove all data:**
1. Open extension settings
2. Click "Clear All Data"
3. Uninstall the extension

## International Users

The extension works worldwide. Your data:
- Stays on your device (encrypted)
- Goes to your API server (your location)
- (Optional) Analytics sent to our servers (US)

We comply with international data protection laws.

## Disclaimer

This extension is provided "as is" without warranty. We are not responsible for:
- Data loss
- Security of your API server
- Third-party services
- Misuse of the extension

Always maintain backups of important data.

## License

The extension is open source under [LICENSE]. This Privacy Policy does not affect the license terms.

---

**Questions?** Contact us at privacy@dbbackup.example.com

**Last Updated:** January 13, 2026

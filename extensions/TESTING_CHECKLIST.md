# Browser Extensions - Testing Checklist

Comprehensive testing checklist for DB Backup Manager browser extensions before store submission.

## Pre-Testing Setup

- [ ] API server running at `http://localhost:8080`
- [ ] Valid API key configured
- [ ] Test database with sample data
- [ ] Browser DevTools open
- [ ] Console monitoring for errors
- [ ] Network tab monitoring

---

## Chrome Extension Testing

### Installation & Setup

- [ ] Load unpacked extension
- [ ] Extension appears in toolbar
- [ ] Extension icon displays correctly
- [ ] No console errors on load
- [ ] Permissions requested are appropriate

### Popup Interface

- [ ] Popup opens when clicking extension icon
- [ ] Dashboard statistics load correctly
- [ ] Recent backups list populates
- [ ] Active alerts section displays
- [ ] All buttons are clickable
- [ ] Quick Backup button works
- [ ] View All button works
- [ ] Dashboard button opens web interface
- [ ] Settings icon opens options page
- [ ] No layout issues at different sizes

### Options Page

- [ ] Options page opens
- [ ] All sections visible
- [ ] API URL field pre-populated
- [ ] API key field masked
- [ ] Show/hide API key toggle works
- [ ] Test Connection button works
- [ ] Connection status displays
- [ ] Sync interval slider works
- [ ] Value updates as slider moves
- [ ] All checkboxes toggle correctly
- [ ] Theme selector works
- [ ] Save button works
- [ ] Success message displays
- [ ] Reset to Defaults works (with confirmation)
- [ ] Clear All Data works (with double confirmation)
- [ ] Statistics display correctly
- [ ] No console errors

### Background Service Worker

- [ ] Service worker starts on extension load
- [ ] Periodic sync runs (check after 5 minutes)
- [ ] Monitoring checks run (check after 1 minute)
- [ ] Badge updates with failed backup count
- [ ] Alarms are set correctly
- [ ] Service worker doesn't crash
- [ ] State persists across restarts

### Context Menu

- [ ] Right-click on extension icon shows menu
- [ ] Quick Backup menu item works
- [ ] View Backups opens web interface
- [ ] Open Dashboard opens web interface
- [ ] Refresh Data updates extension
- [ ] Open Settings opens options
- [ ] About shows version info

### Content Script

- [ ] Visit phpMyAdmin (or similar tool)
- [ ] Extension detects database tool
- [ ] Page indicator appears
- [ ] Floating action button appears
- [ ] Floating button displays menu on click
- [ ] Quick Backup from floating button works
- [ ] Schedule Backup opens interface
- [ ] View Backups works
- [ ] Open Dashboard works
- [ ] Menu closes on outside click
- [ ] No interference with page functionality

### Keyboard Shortcuts

- [ ] Ctrl+Shift+B opens extension
- [ ] Ctrl+Shift+Q triggers quick backup
- [ ] Ctrl+Shift+V opens view backups
- [ ] Shortcuts work from any page
- [ ] Shortcuts don't conflict with browser

### Notifications

- [ ] Backup success notification appears
- [ ] Backup failure notification appears
- [ ] Alert notification appears
- [ ] Notification icon displays correctly
- [ ] Notification text is readable
- [ ] Clicking notification opens relevant page
- [ ] Notifications can be dismissed

### Analytics (if enabled)

- [ ] Extension start event tracked
- [ ] Feature usage events tracked
- [ ] Error events tracked
- [ ] Events stored locally
- [ ] Analytics can be viewed in settings
- [ ] Analytics can be exported
- [ ] Analytics can be cleared
- [ ] Opt-out works correctly

### Performance

- [ ] Popup loads in < 1 second
- [ ] Options page loads in < 1 second
- [ ] Background CPU usage < 1%
- [ ] Memory usage 10-20 MB
- [ ] No memory leaks over 1 hour
- [ ] Network requests are efficient
- [ ] No unnecessary API calls

### Error Handling

- [ ] API server offline: Shows error message
- [ ] Invalid API key: Shows error message
- [ ] Network timeout: Handles gracefully
- [ ] Invalid data: Doesn't crash
- [ ] Empty states display correctly
- [ ] Error messages are user-friendly

### Security

- [ ] No plaintext credentials in console
- [ ] API key masked in options
- [ ] HTTPS enforced for API calls
- [ ] No mixed content warnings
- [ ] CSP compliant (no console warnings)
- [ ] No XSS vulnerabilities

### Cross-Browser Compatibility

- [ ] Works in Chrome 88+
- [ ] Works in Chromium-based browsers
- [ ] No Chrome-specific features used

---

## Firefox Extension Testing

### Installation & Setup

- [ ] Load temporary add-on
- [ ] Extension appears in toolbar
- [ ] Extension icon displays correctly
- [ ] No console errors on load
- [ ] Permissions requested are appropriate

### Core Functionality

- [ ] All Chrome tests apply
- [ ] `browser` namespace works correctly
- [ ] Promises work (no callbacks)
- [ ] Storage API works
- [ ] Notifications work
- [ ] Context menus work
- [ ] Keyboard shortcuts work (Cmd on Mac)

### Firefox-Specific

- [ ] Manifest V2 valid
- [ ] Background script (non-persistent) works
- [ ] WebExtensions API compatibility
- [ ] No Firefox-specific errors
- [ ] about:debugging shows no warnings

---

## Edge Extension Testing

### Installation & Setup

- [ ] Load unpacked extension
- [ ] Extension appears in toolbar
- [ ] Extension icon displays correctly
- [ ] No console errors on load
- [ ] Permissions requested are appropriate

### Core Functionality

- [ ] All Chrome tests apply (Edge is Chromium-based)
- [ ] Works identically to Chrome
- [ ] No Edge-specific issues

---

## Safari Extension Testing (macOS only)

### Xcode Project

- [ ] Xcode project builds successfully
- [ ] No build warnings
- [ ] No code signing errors
- [ ] Archive builds successfully

### Installation & Setup

- [ ] Extension loads in Safari
- [ ] Extension appears in toolbar
- [ ] Extension icon displays correctly
- [ ] Extension enabled in preferences
- [ ] No console errors on load

### Core Functionality

- [ ] All Chrome tests apply
- [ ] `browser` namespace works
- [ ] Native promises work
- [ ] Storage API works
- [ ] Content script works
- [ ] Background page works (non-persistent)

### Safari-Specific

- [ ] macOS app wrapper works
- [ ] Extension preferences in macOS app
- [ ] App icon displays correctly
- [ ] Minimum macOS version check
- [ ] Safari 14+ compatibility

---

## Cross-Browser Testing Matrix

| Feature | Chrome | Firefox | Edge | Safari |
|---------|--------|---------|------|--------|
| Popup | ✓ | ✓ | ✓ | ✓ |
| Options | ✓ | ✓ | ✓ | ✓ |
| Background | ✓ | ✓ | ✓ | ✓ |
| Content Script | ✓ | ✓ | ✓ | ✓ |
| Context Menu | ✓ | ✓ | ✓ | ✓ |
| Keyboard Shortcuts | ✓ | ✓ | ✓ | ✓ |
| Notifications | ✓ | ✓ | ✓ | ✓ |
| Storage | ✓ | ✓ | ✓ | ✓ |
| Analytics | ✓ | ✓ | ✓ | ✓ |

---

## Accessibility Testing

- [ ] Keyboard navigation works
- [ ] Screen reader compatibility
- [ ] Sufficient color contrast
- [ ] Focus indicators visible
- [ ] Alt text for images
- [ ] ARIA labels where appropriate
- [ ] No keyboard traps

---

## Responsive Design Testing

### Popup Sizes

- [ ] Default size (400x600)
- [ ] Minimum size (400x400)
- [ ] Maximum size (800x800)
- [ ] Content doesn't overflow
- [ ] Scrolling works if needed

### Options Page

- [ ] Desktop (1920x1080)
- [ ] Laptop (1366x768)
- [ ] Tablet (768x1024)
- [ ] Mobile (375x667 - not primary but good to check)

---

## Data Integrity Testing

### Sync

- [ ] Data syncs correctly from API
- [ ] Local storage updates
- [ ] No data corruption
- [ ] Handles large datasets
- [ ] Handles empty datasets

### Offline

- [ ] Works when API is offline
- [ ] Queues operations
- [ ] Syncs when back online
- [ ] Shows offline indicator
- [ ] No data loss

---

## Security Testing

### OWASP Top 10

- [ ] No SQL injection (backend)
- [ ] No XSS vulnerabilities
- [ ] No CSRF vulnerabilities
- [ ] Secure authentication
- [ ] No sensitive data exposure
- [ ] Secure dependencies
- [ ] No insecure deserialization
- [ ] Access control enforced
- [ ] Logging and monitoring
- [ ] Known vulnerabilities patched

### Extension-Specific

- [ ] No remote code execution
- [ ] CSP properly configured
- [ ] Minimal permissions
- [ ] No eval() usage
- [ ] No inline scripts
- [ ] HTTPS only
- [ ] API key storage secure

---

## Localization Testing (if supported)

- [ ] English (US)
- [ ] Spanish (es)
- [ ] French (fr)
- [ ] German (de)
- [ ] Date formats correct
- [ ] Number formats correct
- [ ] RTL languages (if supported)

---

## Performance Testing

### Load Times

- [ ] Popup: < 1 second
- [ ] Options: < 1 second
- [ ] Content script injection: < 100ms
- [ ] Background start: < 500ms

### Resource Usage

- [ ] Memory: < 50 MB after 1 hour
- [ ] CPU: < 5% average
- [ ] Network: < 1 MB/hour
- [ ] Storage: < 10 MB
- [ ] Battery impact: Minimal (< 1%)

### Stress Testing

- [ ] 1000+ backups in list
- [ ] 100+ alerts
- [ ] Rapid button clicking
- [ ] Multiple tabs with content script
- [ ] Long-running (24 hours)

---

## Browser Console Checklist

### No Errors

- [ ] No JavaScript errors
- [ ] No network errors
- [ ] No CSP violations
- [ ] No deprecation warnings
- [ ] No failed API calls

### Clean Logs

- [ ] Only intentional logs
- [ ] No sensitive data logged
- [ ] Log levels appropriate
- [ ] Debug logs removable for production

---

## Documentation Testing

### README

- [ ] Accurate information
- [ ] All links work
- [ ] Installation steps correct
- [ ] Screenshots up-to-date

### Code Comments

- [ ] All functions documented
- [ ] Complex logic explained
- [ ] TODOs resolved

### User Guides

- [ ] Easy to understand
- [ ] Screenshots included
- [ ] Step-by-step instructions
- [ ] Troubleshooting section

---

## Store Submission Testing

### Pre-Submission

- [ ] Version number updated
- [ ] Changelog written
- [ ] Screenshots prepared
- [ ] Description accurate
- [ ] Privacy policy updated
- [ ] All links work
- [ ] Icons correct size
- [ ] Package created successfully

### Test Package

- [ ] Install from package (.zip/.xpi)
- [ ] All features work
- [ ] No packaging errors
- [ ] File size reasonable (< 5 MB)

---

## Regression Testing

After any code change:

- [ ] Re-run critical tests
- [ ] Test changed feature
- [ ] Test related features
- [ ] No new console errors
- [ ] Performance unchanged

---

## User Acceptance Testing

- [ ] Internal team testing
- [ ] Beta testers feedback
- [ ] Real-world scenarios
- [ ] Edge cases covered
- [ ] User feedback incorporated

---

## Final Checklist

Before submitting to stores:

- [ ] All tests passed
- [ ] No known bugs
- [ ] Code reviewed
- [ ] Documentation complete
- [ ] Privacy policy compliant
- [ ] Security audit done
- [ ] Performance acceptable
- [ ] Analytics opt-in clear
- [ ] Version number correct
- [ ] Changelog written
- [ ] Screenshots prepared
- [ ] Store listings ready
- [ ] Team approval obtained

---

## Bug Severity Classification

### Critical (Must Fix Before Release)
- Extension doesn't load
- Data loss
- Security vulnerability
- Crashes browser
- API authentication fails

### High (Should Fix Before Release)
- Feature completely broken
- Major UI issues
- Performance problems
- Frequent errors

### Medium (Can Fix in Update)
- Minor feature issues
- UI inconsistencies
- Edge case bugs
- Usability problems

### Low (Nice to Fix)
- Cosmetic issues
- Minor typos
- Rare edge cases
- Enhancement requests

---

## Test Environment

### Browsers

- Chrome 88+ (latest stable)
- Firefox 109+ (latest stable)
- Edge 88+ (latest stable)
- Safari 14+ (latest stable - macOS)

### Operating Systems

- Windows 10/11
- macOS 11+ (Big Sur or later)
- Linux (Ubuntu 20.04+)

### API Server

- Local development server
- Staging environment
- Production-like setup

---

## Automated Testing (Future)

Potential automated tests:

- [ ] Unit tests (Jest)
- [ ] Integration tests (Playwright)
- [ ] E2E tests (Selenium/Puppeteer)
- [ ] Visual regression tests
- [ ] Performance tests
- [ ] Security scans

---

## Sign-Off

- [ ] QA Team: __________________ Date: __________
- [ ] Developer: _________________ Date: __________
- [ ] Product Owner: _____________ Date: __________

---

**Notes:**
- Document any issues found during testing
- Track resolution of all bugs
- Retest after fixes applied
- Final sign-off required before submission

/**
 * Firefox Polyfill for Chrome Extensions API
 * Firefox natively uses the 'browser' namespace with promises
 * This file ensures compatibility with our code
 */

// Firefox already provides 'browser' API
// Our shared utils.js already handles browser detection
// No polyfill needed for Firefox

console.log('[Firefox Polyfill] Loaded');

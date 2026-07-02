import {createApiClient, DbBackupApiClient} from '@db-backup/api-client';
import AsyncStorage from '@react-native-async-storage/async-storage';

// In React Native, REACT_APP_ env vars are not available.
// The default is used for local dev but can be overridden at runtime from the
// Settings screen (persisted in AsyncStorage under `app_settings`).
export const DEFAULT_API_BASE_URL = 'http://localhost:8080/api/v1';

const SETTINGS_STORAGE_KEY = 'app_settings';
const AUTH_TOKEN_KEY = 'auth_token';

// The shared api-client reads the auth token *synchronously* inside its request
// interceptor. We therefore keep a cached, synchronously-readable token here and
// resolve the async AsyncStorage value ahead of time (on init / login / logout).
let cachedAuthToken: string | null = null;
let currentBaseURL = DEFAULT_API_BASE_URL;

function buildClient(baseURL: string): DbBackupApiClient {
  return createApiClient({
    baseURL,
    timeout: 30000,
    // Must be synchronous — returning a Promise here would serialize to
    // "Bearer [object Promise]".
    getAuthToken: () => cachedAuthToken,
  });
}

let client: DbBackupApiClient = buildClient(currentBaseURL);

/**
 * Update the bearer token used by the api client. Call on login/logout and
 * whenever the persisted token changes.
 */
export function setApiAuthToken(token: string | null): void {
  cachedAuthToken = token;
}

/**
 * Point the api client at a different backend. Rebuilds the underlying axios
 * instance so the new base URL takes effect immediately.
 */
export function setApiBaseURL(baseURL: string | undefined | null): void {
  const next = (baseURL || '').trim();
  if (!next || next === currentBaseURL) {
    return;
  }
  currentBaseURL = next;
  client = buildClient(currentBaseURL);
}

export function getApiBaseURL(): string {
  return currentBaseURL;
}

/**
 * Hydrate the api client from persisted storage (base URL + auth token) before
 * any request is made. Safe to call multiple times.
 */
export async function initApiService(): Promise<void> {
  try {
    const [token, settingsRaw] = await Promise.all([
      AsyncStorage.getItem(AUTH_TOKEN_KEY),
      AsyncStorage.getItem(SETTINGS_STORAGE_KEY),
    ]);
    cachedAuthToken = token;
    if (settingsRaw) {
      const parsed = JSON.parse(settingsRaw) as {apiUrl?: string};
      if (parsed?.apiUrl) {
        setApiBaseURL(parsed.apiUrl);
      }
    }
  } catch (error) {
    console.warn('[initApiService] Failed to hydrate api client:', error);
  }
}

/**
 * Stable api service reference. Method calls are always delegated to the current
 * underlying client, so reconstructing the client (e.g. on a base-URL change)
 * does not invalidate imported references.
 */
export const apiService: DbBackupApiClient = new Proxy(
  {} as DbBackupApiClient,
  {
    get(_target, prop: string | symbol) {
      const value = (client as unknown as Record<string | symbol, unknown>)[
        prop
      ];
      if (typeof value === 'function') {
        return (value as (...args: unknown[]) => unknown).bind(client);
      }
      return value;
    },
  },
);

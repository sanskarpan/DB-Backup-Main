/**
 * Derives the notifications WebSocket URL from the configured API base URL.
 *
 * The backend serves the notifications WebSocket at `/notifications/ws` on the
 * same origin/path prefix as the REST API (e.g. `.../api/v1/notifications/ws`)
 * and authenticates the connection via a `?token=<jwt>` query parameter.
 *
 * We convert the HTTP(S) API URL to its WebSocket equivalent:
 *   - http://  -> ws://
 *   - https:// -> wss://
 * and append the JWT read from localStorage ('auth_token'). No host is
 * hardcoded; everything derives from NEXT_PUBLIC_API_URL.
 */
export function getNotificationsWsUrl(): string | null {
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1'

  // http -> ws, https -> wss (only touch the leading scheme).
  const wsBase = apiUrl.replace(/^http(s?):\/\//, 'ws$1://')
  const wsUrl = `${wsBase.replace(/\/$/, '')}/notifications/ws`

  const token =
    typeof window !== 'undefined' ? localStorage.getItem('auth_token') : null

  if (!token) {
    return wsUrl
  }

  return `${wsUrl}?token=${encodeURIComponent(token)}`
}

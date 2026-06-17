const DEFAULT_BASE_URL = 'http://localhost:8080/api/v1'

function getBaseUrl(): string {
  return (typeof window !== 'undefined' && (window as any).__DB_BACKUP_API_URL__) || DEFAULT_BASE_URL
}

function getAuthToken(): string | null {
  try {
    return localStorage.getItem('auth_token')
  } catch {
    return null
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = getAuthToken()
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  }

  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const response = await fetch(`${getBaseUrl()}${path}`, {
    ...options,
    headers,
  })

  if (response.status === 401) {
    // Clear stale token and let the caller handle the redirect/re-auth
    localStorage.removeItem('auth_token')
    throw new Error('Unauthorized: session expired. Please log in again.')
  }

  if (!response.ok) {
    const text = await response.text().catch(() => response.statusText)
    throw new Error(`API error ${response.status}: ${text}`)
  }

  // Return null for 204 No Content
  if (response.status === 204) {
    return null as T
  }

  return response.json() as Promise<T>
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body: unknown) =>
    request<T>(path, { method: 'POST', body: JSON.stringify(body) }),
  put: <T>(path: string, body: unknown) =>
    request<T>(path, { method: 'PUT', body: JSON.stringify(body) }),
  delete: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
}

export default api

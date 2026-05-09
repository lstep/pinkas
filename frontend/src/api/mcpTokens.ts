import { useAuthStore } from '../store/auth'

const API_BASE = '/api'

function getHeaders(): Record<string, string> {
  const token = useAuthStore.getState().accessToken
  return {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  }
}

// Token refresh — deduplicates concurrent refreshes
let isRefreshing = false
let refreshPromise: Promise<void> | null = null

async function refreshToken(): Promise<void> {
  if (isRefreshing && refreshPromise) {
    return refreshPromise
  }
  isRefreshing = true
  refreshPromise = fetch(`${API_BASE}/auth/refresh`, {
    method: 'POST',
    credentials: 'include',
  })
    .then(async (res) => {
      if (!res.ok) {
        useAuthStore.getState().logout()
        window.location.href = '/login'
        throw new Error('Session expired')
      }
      const data = await res.json()
      if (data.token?.accessToken) {
        useAuthStore.setState({ accessToken: data.token.accessToken })
      }
      if (data.user) {
        useAuthStore.setState({ user: data.user })
      }
    })
    .finally(() => {
      isRefreshing = false
      refreshPromise = null
    })
  return refreshPromise
}

async function handleResponse(res: Response, retry?: () => Promise<Response>): Promise<Response> {
  if (res.status === 401) {
    try {
      await refreshToken()
      if (retry) {
        const retried = await retry()
        if (!retried.ok) {
          const body = await retried.text()
          let msg: string
          try {
            const parsed = JSON.parse(body)
            if (parsed.error && typeof parsed.error === 'object') {
              msg = parsed.error.message || parsed.error.code || body
            } else {
              msg = parsed.message || parsed.error || body
            }
          } catch {
            msg = body || retried.statusText
          }
          throw new Error(msg)
        }
        return retried
      }
    } catch (err) {
      throw err instanceof Error ? err : new Error('Failed to refresh session')
    }
    return res // shouldn't reach here if retry succeeds
  }

  if (!res.ok) {
    const body = await res.text()
    let msg: string
    try {
      const parsed = JSON.parse(body)
      // Handle {error: {message: "..."}} from httputil.WriteError
      if (parsed.error && typeof parsed.error === 'object') {
        msg = parsed.error.message || parsed.error.code || body
      } else {
        msg = parsed.message || parsed.error || body
      }
    } catch {
      msg = body || res.statusText
    }
    throw new Error(msg)
  }
  return res
}

export interface MCPToken {
  id: string
  userId: string
  name: string
  tokenPrefix: string
  scopes: string
  spaceId?: string
  lastUsedAt?: number
  createdAt: number
  expiresAt?: number
}

export interface CreateTokenRequest {
  name: string
  scopes: string[]
  spaceId?: string
  expiresInDays?: number
}

export interface CreateTokenResponse {
  token: MCPToken
  secret: string
}

export interface ListTokensResponse {
  tokens: MCPToken[]
}

export async function createMCPToken(req: CreateTokenRequest): Promise<CreateTokenResponse> {
  const doFetch = () =>
    fetch('/api/mcp-tokens', {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify(req),
    })
  const res = await doFetch()
  const finalRes = await handleResponse(res, doFetch)
  return finalRes.json()
}

export async function listMCPTokens(): Promise<MCPToken[]> {
  const doFetch = () =>
    fetch('/api/mcp-tokens', {
      headers: getHeaders(),
    })
  const res = await doFetch()
  const finalRes = await handleResponse(res, doFetch)
  const data: ListTokensResponse = await finalRes.json()
  return data.tokens
}

export async function deleteMCPToken(id: string): Promise<void> {
  const doFetch = () =>
    fetch(`/api/mcp-tokens/${id}`, {
      method: 'DELETE',
      headers: getHeaders(),
    })
  const res = await doFetch()
  await handleResponse(res, doFetch)
}

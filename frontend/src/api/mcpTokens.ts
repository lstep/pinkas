import { useAuthStore } from '../store/auth'

function getHeaders(): Record<string, string> {
  const token = useAuthStore.getState().accessToken
  return {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  }
}

async function handleResponse(res: Response): Promise<Response> {
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
  const res = await fetch('/api/mcp-tokens', {
    method: 'POST',
    headers: getHeaders(),
    body: JSON.stringify(req),
  })
  await handleResponse(res)
  return res.json()
}

export async function listMCPTokens(): Promise<MCPToken[]> {
  const res = await fetch('/api/mcp-tokens', {
    headers: getHeaders(),
  })
  await handleResponse(res)
  const data: ListTokensResponse = await res.json()
  return data.tokens
}

export async function deleteMCPToken(id: string): Promise<void> {
  const res = await fetch(`/api/mcp-tokens/${id}`, {
    method: 'DELETE',
    headers: getHeaders(),
  })
  await handleResponse(res)
}

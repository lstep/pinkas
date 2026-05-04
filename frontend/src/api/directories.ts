import { useAuthStore } from '../store/auth'

const API_BASE = '/api'

function getHeaders(): Record<string, string> {
  const token = useAuthStore.getState().accessToken
  return {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  }
}

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
          const err = await retried.json().catch(() => ({}))
          throw new Error(err.error?.message || `Request failed (${retried.status})`)
        }
        return retried
      }
    } catch (e) {
      throw new Error('Unauthorized (401)')
    }
  }
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.error?.message || `Request failed (${res.status})`)
  }
  return res
}

// Backend returns camelCase; we normalize to snake_case for consistency
function normalizeDirectory(raw: any): Directory {
  return {
    id: raw.id,
    space_id: raw.spaceId ?? raw.space_id ?? '',
    parent_id: raw.parentId ?? raw.parent_id ?? null,
    name: raw.name ?? '',
    slug: raw.slug ?? '',
    position: raw.position ?? 0,
    icon: raw.icon ?? undefined,
    created_by: raw.createdBy ?? raw.created_by ?? '',
    created_at: raw.createdAt ?? raw.created_at ?? 0,
    updated_at: raw.updatedAt ?? raw.updated_at ?? 0,
  }
}

export interface Directory {
  id: string
  space_id: string
  parent_id: string | null
  name: string
  slug: string
  position: number
  icon?: string
  created_by: string
  created_at: number
  updated_at: number
}

export async function listRootDirectories(spaceId: string): Promise<Directory[]> {
  const doFetch = () => fetch(`${API_BASE}/spaces/${spaceId}/directories`, { headers: getHeaders() })
  const res = await doFetch()
  await handleResponse(res, doFetch)
  const data = await res.json()
  return (data.directories || []).map(normalizeDirectory)
}

export async function getDirectory(id: string): Promise<Directory> {
  const doFetch = () => fetch(`${API_BASE}/directories/${id}`, { headers: getHeaders() })
  const res = await doFetch()
  await handleResponse(res, doFetch)
  return normalizeDirectory(await res.json())
}

export async function getDirectoryBySlug(spaceId: string, slug: string): Promise<Directory> {
  const doFetch = () => fetch(`${API_BASE}/spaces/${spaceId}/directories/${slug}`, { headers: getHeaders() })
  const res = await doFetch()
  await handleResponse(res, doFetch)
  return normalizeDirectory(await res.json())
}

export async function createDirectory(params: {
  space_id: string
  parent_id?: string | null
  name: string
  icon?: string
}): Promise<Directory> {
  const body = {
    spaceId: params.space_id,
    parentId: params.parent_id,
    name: params.name,
    icon: params.icon,
  }
  const doFetch = () => fetch(`${API_BASE}/directories`, {
    method: 'POST',
    headers: getHeaders(),
    body: JSON.stringify(body),
  })
  const res = await doFetch()
  await handleResponse(res, doFetch)
  return normalizeDirectory(await res.json())
}

export async function updateDirectory(id: string, params: { name?: string; icon?: string }): Promise<Directory> {
  const body: any = {}
  if (params.name !== undefined) body.name = params.name
  if (params.icon !== undefined) body.icon = params.icon
  const doFetch = () => fetch(`${API_BASE}/directories/${id}`, {
    method: 'PATCH',
    headers: getHeaders(),
    body: JSON.stringify(body),
  })
  const res = await doFetch()
  await handleResponse(res, doFetch)
  return normalizeDirectory(await res.json())
}

export async function deleteDirectory(id: string): Promise<void> {
  const doFetch = () => fetch(`${API_BASE}/directories/${id}`, {
    method: 'DELETE',
    headers: getHeaders(),
  })
  const res = await doFetch()
  await handleResponse(res, doFetch)
}

export async function moveDirectory(id: string, parent_id: string | null, position: number): Promise<void> {
  const doFetch = () => fetch(`${API_BASE}/directories/${id}/move`, {
    method: 'POST',
    headers: getHeaders(),
    body: JSON.stringify({ parentId: parent_id, position }),
  })
  const res = await doFetch()
  await handleResponse(res, doFetch)
}

export async function listDirectoryChildren(parentId: string): Promise<Directory[]> {
  const doFetch = () => fetch(`${API_BASE}/directories/${parentId}/children`, { headers: getHeaders() })
  const res = await doFetch()
  await handleResponse(res, doFetch)
  const data = await res.json()
  return (data.children || []).map(normalizeDirectory)
}

export async function getDirectoryBreadcrumb(id: string): Promise<Directory[]> {
  const doFetch = () => fetch(`${API_BASE}/directories/${id}/breadcrumb`, { headers: getHeaders() })
  const res = await doFetch()
  await handleResponse(res, doFetch)
  const data = await res.json()
  return (data.breadcrumb || []).map(normalizeDirectory)
}

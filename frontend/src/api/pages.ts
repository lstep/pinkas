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
function normalizePage(raw: any): Page {
  return {
    id: raw.id,
    title: raw.title ?? '',
    slug: raw.slug ?? '',
    icon: raw.icon ?? null,
    position: raw.position ?? 0,
    directory_id: raw.directoryId ?? raw.directory_id ?? null,
    space_id: raw.spaceId ?? raw.space_id ?? '',
    content: raw.content,
    created_by: raw.createdBy ?? raw.created_by ?? '',
    created_at: raw.createdAt ?? raw.created_at ?? 0,
    updated_at: raw.updatedAt ?? raw.updated_at ?? 0,
  }
}

export interface Page {
  id: string
  title: string
  slug: string
  icon: string | null
  position: number
  directory_id: string | null
  space_id: string
  content?: string
  created_by: string
  created_at: number
  updated_at: number
}

export async function listRootPages(spaceId: string): Promise<Page[]> {
  const doFetch = () => fetch(`${API_BASE}/spaces/${spaceId}/pages`, { headers: getHeaders() })
  const res = await doFetch()
  await handleResponse(res, doFetch)
  const data = await res.json()
  return (data.pages || []).map(normalizePage)
}

export async function listPagesByDirectory(directoryId: string): Promise<Page[]> {
  const doFetch = () => fetch(`${API_BASE}/directories/${directoryId}/pages`, { headers: getHeaders() })
  const res = await doFetch()
  await handleResponse(res, doFetch)
  const data = await res.json()
  return (data.pages || []).map(normalizePage)
}

export async function getPage(id: string): Promise<Page> {
  const doFetch = () => fetch(`${API_BASE}/pages/${id}`, { headers: getHeaders() })
  const res = await doFetch()
  await handleResponse(res, doFetch)
  return normalizePage(await res.json())
}

export async function getPageBySlug(spaceId: string, slug: string): Promise<Page> {
  const doFetch = () => fetch(`${API_BASE}/spaces/${spaceId}/pages/${slug}`, { headers: getHeaders() })
  const res = await doFetch()
  await handleResponse(res, doFetch)
  return normalizePage(await res.json())
}

export async function createPage(params: {
  space_id: string
  title: string
  directory_id?: string | null
  icon?: string | null
}): Promise<Page> {
  const body = {
    spaceId: params.space_id,
    title: params.title,
    directoryId: params.directory_id,
    icon: params.icon,
  }
  const doFetch = () => fetch(`${API_BASE}/pages`, {
    method: 'POST',
    headers: getHeaders(),
    body: JSON.stringify(body),
  })
  const res = await doFetch()
  await handleResponse(res, doFetch)
  return normalizePage(await res.json())
}

export async function updatePage(id: string, updates: Partial<Page> & { position?: number }): Promise<Page> {
  const body: any = {}
  if (updates.title !== undefined) body.title = updates.title
  if (updates.icon !== undefined) body.icon = updates.icon
  if (updates.position !== undefined) body.position = updates.position
  const doFetch = () => fetch(`${API_BASE}/pages/${id}`, {
    method: 'PATCH',
    headers: getHeaders(),
    body: JSON.stringify(body),
  })
  const res = await doFetch()
  await handleResponse(res, doFetch)
  return normalizePage(await res.json())
}

export async function deletePage(id: string): Promise<void> {
  const doFetch = () => fetch(`${API_BASE}/pages/${id}`, {
    method: 'DELETE',
    headers: getHeaders(),
  })
  const res = await doFetch()
  await handleResponse(res, doFetch)
}

export async function movePage(id: string, directory_id: string | null, position: number): Promise<void> {
  const doFetch = () => fetch(`${API_BASE}/pages/${id}/move`, {
    method: 'POST',
    headers: getHeaders(),
    body: JSON.stringify({ directoryId: directory_id, position }),
  })
  const res = await doFetch()
  await handleResponse(res, doFetch)
}

export async function getBreadcrumb(id: string): Promise<Page[]> {
  const doFetch = () => fetch(`${API_BASE}/pages/${id}/breadcrumb`, { headers: getHeaders() })
  const res = await doFetch()
  await handleResponse(res, doFetch)
  const data = await res.json()
  return (data.breadcrumb || []).map(normalizePage)
}

function normalizeSpace(raw: any): Space {
  return {
    id: raw.id,
    name: raw.name,
    slug: raw.slug,
    default_permission: raw.defaultPermission ?? raw.default_permission ?? 'none',
  }
}

export async function listSpaces(): Promise<Space[]> {
  const doFetch = () => fetch(`${API_BASE}/spaces`, { headers: getHeaders() })
  const res = await doFetch()
  await handleResponse(res, doFetch)
  const data = await res.json()
  return (data.spaces || []).map(normalizeSpace)
}

export interface Space {
  id: string
  name: string
  slug: string
  default_permission: string
}

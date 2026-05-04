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
  if (isRefreshing && refreshPromise) return refreshPromise
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
    } catch {
      throw new Error('Unauthorized (401)')
    }
  }
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.error?.message || `Request failed (${res.status})`)
  }
  return res
}

// ─── Users ────────────────────────────────────────────

export interface User {
  id: string
  email: string
  name: string
  role: string
}

export async function listUsers(): Promise<User[]> {
  const doFetch = () => fetch(`${API_BASE}/users`, { headers: getHeaders() })
  const res = await doFetch()
  await handleResponse(res, doFetch)
  const data = await res.json()
  return (data.users || []).map((u: any) => ({
    id: u.id,
    email: u.email,
    name: u.name ?? '',
    role: u.role ?? '',
  }))
}

export async function getUser(id: string): Promise<User> {
  const doFetch = () => fetch(`${API_BASE}/users/${id}`, { headers: getHeaders() })
  const res = await doFetch()
  await handleResponse(res, doFetch)
  const data = await res.json()
  return {
    id: data.id,
    email: data.email,
    name: data.name ?? '',
    role: data.role ?? '',
  }
}

export async function updateUser(id: string, updates: { name?: string; role?: string }): Promise<User> {
  const body: any = {}
  if (updates.name !== undefined) body.name = updates.name
  if (updates.role !== undefined) body.role = updates.role
  const doFetch = () =>
    fetch(`${API_BASE}/users/${id}`, {
      method: 'PATCH',
      headers: getHeaders(),
      body: JSON.stringify(body),
    })
  const res = await doFetch()
  await handleResponse(res, doFetch)
  const data = await res.json()
  return {
    id: data.id,
    email: data.email,
    name: data.name ?? '',
    role: data.role ?? '',
  }
}

export async function deleteUser(id: string): Promise<void> {
  const doFetch = () =>
    fetch(`${API_BASE}/users/${id}`, {
      method: 'DELETE',
      headers: getHeaders(),
    })
  const res = await doFetch()
  await handleResponse(res, doFetch)
}

// ─── Groups ────────────────────────────────────────────

export interface Group {
  id: string
  name: string
}

export interface GroupMember {
  id: string
  email: string
  name: string
}

export async function listGroups(): Promise<Group[]> {
  const doFetch = () => fetch(`${API_BASE}/groups`, { headers: getHeaders() })
  const res = await doFetch()
  await handleResponse(res, doFetch)
  const data = await res.json()
  return data.groups || []
}

export async function createGroup(name: string): Promise<Group> {
  const doFetch = () =>
    fetch(`${API_BASE}/groups`, {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify({ name }),
    })
  const res = await doFetch()
  await handleResponse(res, doFetch)
  return await res.json()
}

export async function updateGroup(id: string, name: string): Promise<Group> {
  const doFetch = () =>
    fetch(`${API_BASE}/groups/${id}`, {
      method: 'PATCH',
      headers: getHeaders(),
      body: JSON.stringify({ name }),
    })
  const res = await doFetch()
  await handleResponse(res, doFetch)
  return await res.json()
}

export async function deleteGroup(id: string): Promise<void> {
  const doFetch = () =>
    fetch(`${API_BASE}/groups/${id}`, {
      method: 'DELETE',
      headers: getHeaders(),
    })
  const res = await doFetch()
  await handleResponse(res, doFetch)
}

export async function listGroupMembers(groupId: string): Promise<GroupMember[]> {
  const doFetch = () => fetch(`${API_BASE}/groups/${groupId}/members`, { headers: getHeaders() })
  const res = await doFetch()
  await handleResponse(res, doFetch)
  const data = await res.json()
  return data.members || []
}

export async function addGroupMember(groupId: string, userId: string): Promise<void> {
  const doFetch = () =>
    fetch(`${API_BASE}/groups/${groupId}/members`, {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify({ userId }),
    })
  const res = await doFetch()
  await handleResponse(res, doFetch)
}

export async function removeGroupMember(groupId: string, userId: string): Promise<void> {
  const doFetch = () =>
    fetch(`${API_BASE}/groups/${groupId}/members/${userId}`, {
      method: 'DELETE',
      headers: getHeaders(),
    })
  const res = await doFetch()
  await handleResponse(res, doFetch)
}

// ─── Permissions ───────────────────────────────────────

export interface Permission {
  targetType: string
  targetId: string
  granteeType: string
  granteeId: string
  level: number
}

export async function listPermissionsForTarget(targetType: string, targetId: string): Promise<Permission[]> {
  const doFetch = () =>
    fetch(`${API_BASE}/permissions?targetType=${encodeURIComponent(targetType)}&targetId=${encodeURIComponent(targetId)}`, {
      headers: getHeaders(),
    })
  const res = await doFetch()
  await handleResponse(res, doFetch)
  const data = await res.json()
  return data.permissions || []
}

export async function setPermission(
  targetType: string,
  targetId: string,
  granteeType: string,
  granteeId: string,
  level: number
): Promise<void> {
  const doFetch = () =>
    fetch(`${API_BASE}/permissions`, {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify({ targetType, targetId, granteeType, granteeId, level }),
    })
  const res = await doFetch()
  await handleResponse(res, doFetch)
}

export async function removePermission(
  targetType: string,
  targetId: string,
  granteeType: string,
  granteeId: string
): Promise<void> {
  const doFetch = () =>
    fetch(`${API_BASE}/permissions`, {
      method: 'DELETE',
      headers: getHeaders(),
      body: JSON.stringify({ targetType, targetId, granteeType, granteeId }),
    })
  const res = await doFetch()
  await handleResponse(res, doFetch)
}

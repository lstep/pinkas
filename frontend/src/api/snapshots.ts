import { useAuthStore } from '../store/auth'

const API_BASE = '/api'

function getHeaders(): Record<string, string> {
  const token = useAuthStore.getState().accessToken
  return {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  }
}

export interface Snapshot {
  id: string
  pageId: string
  label?: string
  markdown?: string
  authorId?: string
  createdAt: number
}

function normalizeSnapshot(raw: any): Snapshot {
  return {
    id: raw.id,
    pageId: raw.pageId ?? raw.page_id ?? '',
    label: raw.label ?? undefined,
    markdown: raw.markdown ?? undefined,
    authorId: raw.authorId ?? raw.author_id ?? undefined,
    createdAt: raw.createdAt ?? raw.created_at ?? 0,
  }
}

export async function listSnapshots(pageId: string): Promise<Snapshot[]> {
  const res = await fetch(`${API_BASE}/pages/${pageId}/snapshots`, {
    headers: getHeaders(),
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.error?.message || `Failed to list snapshots (${res.status})`)
  }
  const data = await res.json()
  return (data.snapshots || []).map(normalizeSnapshot)
}

export async function getSnapshot(pageId: string, snapshotId: string): Promise<Snapshot> {
  const res = await fetch(`${API_BASE}/pages/${pageId}/snapshots/${snapshotId}`, {
    headers: getHeaders(),
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.error?.message || `Failed to get snapshot (${res.status})`)
  }
  return normalizeSnapshot(await res.json())
}

export async function restoreSnapshot(pageId: string, snapshotId: string): Promise<void> {
  const res = await fetch(`${API_BASE}/pages/${pageId}/restore`, {
    method: 'POST',
    headers: getHeaders(),
    body: JSON.stringify({ snapshotId }),
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.error?.message || `Failed to restore snapshot (${res.status})`)
  }
}

function timeAgo(ts: number): string {
  const seconds = Math.floor((Date.now() - ts) / 1000)
  if (seconds < 60) return 'just now'
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days}d ago`
  const months = Math.floor(days / 30)
  return `${months}mo ago`
}

function formatTimestamp(ts: number): string {
  return new Date(ts).toLocaleString()
}

export { timeAgo, formatTimestamp }

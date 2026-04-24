import { useTreeStore } from '../store/tree'

let eventSource: EventSource | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let lastEventId: string | null = null

export function connectSSE(token: string) {
  if (eventSource) {
    eventSource.close()
  }

  const url = new URL('/api/events', window.location.origin)
  url.searchParams.set('token', token)
  if (lastEventId) {
    url.searchParams.set('last_event_id', lastEventId)
  }

  eventSource = new EventSource(url.toString())

  eventSource.onopen = () => {
    console.log('[sse] connected')
  }

  eventSource.onmessage = (event) => {
    try {
      if (event.lastEventId) {
        lastEventId = event.lastEventId
      }
      const msg = JSON.parse(event.data)
      console.log('[sse] event:', msg.type, msg.payload)
      handleSSEEvent(msg)
    } catch (e) {
      console.error('[sse] parse error:', e)
    }
  }

  eventSource.onerror = (err) => {
    console.error('[sse] error:', err)
    eventSource?.close()
    // Reconnect after 3s with lastEventId
    if (reconnectTimer) clearTimeout(reconnectTimer)
    reconnectTimer = setTimeout(() => {
      console.log('[sse] reconnecting...')
      connectSSE(token)
    }, 3000)
  }
}

export function disconnectSSE() {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
  if (eventSource) {
    eventSource.close()
    eventSource = null
  }
}

function handleSSEEvent(msg: { type: string; payload: any }) {
  const store = useTreeStore.getState()

  switch (msg.type) {
    case 'page_created': {
      const p = msg.payload
      store.addNode({
        id: p.id,
        type: 'page',
        title: p.title,
        slug: p.slug,
        icon: p.icon ?? undefined,
        position: p.position ?? 0,
        directory_id: p.directoryId ?? p.directory_id ?? null,
        space_id: p.spaceId ?? p.space_id ?? '',
        parent_id: null, // pages don't use parent_id
        created_by: p.createdBy ?? p.created_by ?? '',
        created_at: p.createdAt ?? p.created_at ?? Date.now(),
        updated_at: p.updatedAt ?? p.updated_at ?? Date.now(),
      })
      break
    }
    case 'page_updated': {
      const p = msg.payload
      store.updateNode(p.id, {
        title: p.title,
        slug: p.slug,
        icon: p.icon ?? undefined,
        position: p.position,
      })
      break
    }
    case 'page_moved': {
      const p = msg.payload
      // For pages, the new parent is the directory_id
      store.moveNode(p.id, (p.directoryId ?? p.directory_id) || null, p.position ?? 0)
      break
    }
    case 'page_deleted': {
      store.removeNode(msg.payload.id)
      break
    }
    case 'directory_created': {
      const d = msg.payload
      store.addNode({
        id: d.id,
        type: 'directory',
        name: d.name,
        title: d.name, // alias for UI
        slug: d.slug,
        icon: d.icon ?? undefined,
        position: d.position ?? 0,
        parent_id: d.parentId ?? d.parent_id ?? null,
        directory_id: null, // directories don't use directory_id
        space_id: d.spaceId ?? d.space_id ?? '',
        created_by: d.createdBy ?? d.created_by ?? '',
        created_at: d.createdAt ?? d.created_at ?? Date.now(),
        updated_at: d.updatedAt ?? d.updated_at ?? Date.now(),
      })
      break
    }
    case 'directory_updated': {
      const d = msg.payload
      store.updateNode(d.id, {
        name: d.name,
        title: d.name, // alias for UI
        slug: d.slug,
        icon: d.icon ?? undefined,
        position: d.position,
      })
      break
    }
    case 'directory_moved': {
      const d = msg.payload
      // For directories, the new parent is the parent_id
      store.moveNode(d.id, (d.parentId ?? d.parent_id) || null, d.position ?? 0)
      break
    }
    case 'directory_deleted': {
      store.removeNode(msg.payload.id)
      break
    }
    default:
      console.log('[sse] unhandled event type:', msg.type)
  }
}

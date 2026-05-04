import React, { useEffect, useRef, useState } from 'react'
import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Collaboration from '@tiptap/extension-collaboration'
import CollaborationCaret from '@tiptap/extension-collaboration-caret'
import { FileUpload } from '@tiptap-codeless/extension-file-upload'
import { HocuspocusProvider } from '@hocuspocus/provider'
import { useEditorStore } from '../store/editor'
import { useAuthStore } from '../store/auth'
import { Card, Badge } from '../components/ui'
import './editor.css'

const colors = [
  '#958DF1', '#F98181', '#FBBC88', '#FAF594', '#70CFF8', '#94FADB', '#B9F18D',
]

function TipTapEditor({ provider, userName, permission, pageId }: { provider: HocuspocusProvider; userName: string; permission: string; pageId: string }) {
  const isReadOnly = permission === 'viewer' || permission === 'none'
  const editor = useEditor({
    editable: !isReadOnly,
    extensions: [
      StarterKit.configure({
        undoRedo: false,
      }),
      Collaboration.configure({
        document: provider.document,
      }),
      CollaborationCaret.configure({
        provider,
        user: {
          name: userName,
          color: colors[Math.floor(Math.random() * colors.length)],
        },
      }),
      FileUpload.configure({
        locale: 'en',
        storage: {
          mode: 'custom',
          upload: async (files: File[], _ctx: any) => {
            const assets = await Promise.all(
              files.map(async (file: File) => {
                const formData = new FormData()
                formData.append('file', file)
                formData.append('pageId', pageId)

                const { useAuthStore } = await import('../store/auth')
                const token = useAuthStore.getState().accessToken

                const res = await fetch('/api/attachments', {
                  method: 'POST',
                  headers: token ? { Authorization: `Bearer ${token}` } : {},
                  body: formData,
                })

                if (!res.ok) {
                  const err = await res.json().catch(() => ({}))
                  throw new Error(err.error?.message || 'Upload failed')
                }

                const data = await res.json()
                const url = token
                  ? `${data.url}${data.url.includes('?') ? '&' : '?'}token=${encodeURIComponent(token)}`
                  : data.url

                return {
                  kind: file.type.startsWith('image/') ? 'image' as const : 'file' as const,
                  url,
                  name: file.name,
                  mimeType: file.type,
                  size: file.size,
                }
              })
            )
            return { assets }
          },
        },
        ingest: {
          paste: true,
          drop: true,
          allowedMimeTypes: ['image/jpeg', 'image/png', 'image/gif', 'image/webp'],
        },
        ui: {
          bubbleMenu: { enabled: true, zIndex: 1000 },
          uploadPlaceholder: { enabled: true },
        },
        onError: (error: unknown) => {
          console.error('[file-upload] error:', error)
        },
      }),
    ],
  })

  // Debug: log Yjs document updates
  useEffect(() => {
    const ydoc = provider.document
    console.log('[debug] Y.Doc initialized, name:', provider.configuration.name)
    console.log('[debug] Y.Doc guid:', (ydoc as any).guid)

    const updateHandler = (update: Uint8Array, origin: any) => {
      const isRemote = origin !== null && origin !== ydoc
      console.log('[debug] Yjs update:', update.length, 'bytes, remote:', isRemote, 'origin:', origin?.constructor?.name)
    }

    ydoc.on('update', updateHandler)

    // Debug awareness
    const awareness = provider.awareness
    if (awareness) {
      const awarenessHandler = () => {
        const states = Array.from(awareness.getStates().values())
        console.log('[debug] Awareness states count:', states.length, 'users:', states.map((s: any) => s.user?.name || 'anon'))
      }
      awareness.on('change', awarenessHandler)
      return () => {
        ydoc.off('update', updateHandler)
        awareness.off('change', awarenessHandler)
      }
    }

    return () => {
      ydoc.off('update', updateHandler)
    }
  }, [provider])

  return (
    <div className="editor-wrapper">
      <EditorContent editor={editor} className="tiptap" />
    </div>
  )
}

export const CollaborativeEditor: React.FC<{ onToggleHistory?: () => void }> = ({ onToggleHistory }) => {
  const { providerUrl, docId, permission } = useEditorStore()
  const accessToken = useAuthStore((s) => s.accessToken)
  const user = useAuthStore((s) => s.user)
  const providerRef = useRef<HocuspocusProvider | null>(null)
  const [provider, setProvider] = useState<HocuspocusProvider | null>(null)
  const [status, setStatus] = useState<'connecting' | 'connected' | 'disconnected'>('connecting')
  const pageId = docId

  const userName = user?.name || user?.email || 'Anonymous'
  const token = accessToken || 'iteration1-stub'

  useEffect(() => {
    const p = new HocuspocusProvider({
      url: providerUrl,
      name: docId,
      token: token,
      onStatus: ({ status }) => {
        console.log('[provider] status:', status)
        setStatus(status as 'connecting' | 'connected' | 'disconnected')
      },
      onConnect: () => console.log('[provider] WebSocket connected'),
      onAuthenticated: () => console.log('[provider] authenticated'),
      onSynced: () => console.log('[provider] document synced'),
      onClose: () => console.log('[provider] closed'),
    })

    providerRef.current = p
    setProvider(p)

    return () => {
      p.destroy()
    }
  }, [providerUrl, docId, token])

  if (!provider) {
    return (
      <div className="editor-container">
        <Card padding="md" className="editor-loading">
          <div className="skeleton" style={{ width: 200, height: 20, marginBottom: 12 }} />
          <div className="skeleton" style={{ width: '100%', height: 16, marginBottom: 8 }} />
          <div className="skeleton" style={{ width: '90%', height: 16, marginBottom: 8 }} />
          <div className="skeleton" style={{ width: '60%', height: 16 }} />
        </Card>
      </div>
    )
  }

  return (
    <div className="editor-container">
      <Card padding="none" className="editor-card" hover>
        <div className="editor-status-bar">
          <span className={`status-dot ${status}`} title={status} />
          <span className="status-text">
            {status === 'connected' ? 'Live' : status}
          </span>
          {(permission === 'viewer' || permission === 'none') && (
            <Badge variant="viewer" size="sm">Read-only</Badge>
          )}
          <div className="editor-status-actions">
            <button
              className="toolbar-btn"
              onClick={onToggleHistory}
              title="Page history"
              type="button"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <circle cx="12" cy="12" r="10" />
                <polyline points="12 6 12 12 16 14" />
              </svg>
              History
            </button>
          </div>
        </div>
        <div className="editor-content">
          <TipTapEditor provider={provider} userName={userName} permission={permission} pageId={pageId} />
        </div>
      </Card>
    </div>
  )
}

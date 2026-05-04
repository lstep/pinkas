import React, { useEffect, useRef, useState } from 'react'
import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Collaboration from '@tiptap/extension-collaboration'
import CollaborationCursor from '@tiptap/extension-collaboration-cursor'
import { HocuspocusProvider } from '@hocuspocus/provider'
import { useEditorStore } from '../store/editor'
import { useAuthStore } from '../store/auth'
import './editor.css'

const colors = [
  '#958DF1', '#F98181', '#FBBC88', '#FAF594', '#70CFF8', '#94FADB', '#B9F18D',
]

function TipTapEditor({ provider, userName, permission }: { provider: HocuspocusProvider; userName: string; permission: string }) {
  const isReadOnly = permission === 'viewer' || permission === 'none'
  const editor = useEditor({
    editable: !isReadOnly,
    extensions: [
      StarterKit.configure({
        history: false,
      }),
      Collaboration.configure({
        document: provider.document,
      }),
      CollaborationCursor.configure({
        provider,
        user: {
          name: userName,
          color: colors[Math.floor(Math.random() * colors.length)],
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

  return <EditorContent editor={editor} className="tiptap" />
}

export const CollaborativeEditor: React.FC = () => {
  const { providerUrl, docId, permission } = useEditorStore()
  const accessToken = useAuthStore((s) => s.accessToken)
  const user = useAuthStore((s) => s.user)
  const providerRef = useRef<HocuspocusProvider | null>(null)
  const [provider, setProvider] = useState<HocuspocusProvider | null>(null)
  const [status, setStatus] = useState<'connecting' | 'connected' | 'disconnected'>('connecting')

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
    return <div className="editor-container"><p>Initializing editor...</p></div>
  }

  return (
    <div className="editor-container">
      <div className="editor-status-bar">
        <span className={`status-dot ${status}`} title={status} />
        <span className="status-text">{status === 'connected' ? 'Live' : status}</span>
        {(permission === 'viewer' || permission === 'none') && <span className="status-ro">Read-only</span>}
      </div>
      <TipTapEditor provider={provider} userName={userName} permission={permission} />
    </div>
  )
}

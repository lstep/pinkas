import React, { useEffect, useRef, useState } from 'react'
import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Collaboration from '@tiptap/extension-collaboration'
import CollaborationCursor from '@tiptap/extension-collaboration-cursor'
import { HocuspocusProvider } from '@hocuspocus/provider'
import { useEditorStore } from '../store/editor'
import './editor.css'

const colors = [
  '#958DF1', '#F98181', '#FBBC88', '#FAF594', '#70CFF8', '#94FADB', '#B9F18D',
]

function TipTapEditor({ provider, userName }: { provider: HocuspocusProvider; userName: string }) {
  const editor = useEditor({
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

  return <EditorContent editor={editor} className="tiptap" />
}

export const CollaborativeEditor: React.FC = () => {
  const { providerUrl, docId, userName } = useEditorStore()
  const providerRef = useRef<HocuspocusProvider | null>(null)
  const [provider, setProvider] = useState<HocuspocusProvider | null>(null)
  const [status, setStatus] = useState<'connecting' | 'connected' | 'disconnected'>('connecting')

  useEffect(() => {
    const p = new HocuspocusProvider({
      url: providerUrl,
      name: docId,
      token: 'iteration1-stub',
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
  }, [providerUrl, docId])

  if (!provider) {
    return <div className="editor-container"><p>Initializing editor...</p></div>
  }

  return (
    <div className="editor-container">
      <div className="editor-header">
        <h1>Mostdoc — Iteration 1</h1>
        <div className="status">
          <span className={`status-dot ${status}`} />
          {status}
        </div>
      </div>
      <TipTapEditor provider={provider} userName={userName} />
    </div>
  )
}

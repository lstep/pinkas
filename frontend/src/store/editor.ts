import { create } from 'zustand'

function getProviderUrl(): string {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}/collab`
}

interface EditorState {
  providerUrl: string
  docId: string
  userName: string
  setProviderUrl: (url: string) => void
  setDocId: (id: string) => void
  setUserName: (name: string) => void
}

export const useEditorStore = create<EditorState>((set) => ({
  providerUrl: getProviderUrl(),
  docId: 'seed-page-001',
  userName: 'Anonymous',
  setProviderUrl: (providerUrl) => set({ providerUrl }),
  setDocId: (docId) => set({ docId }),
  setUserName: (userName) => set({ userName }),
}))

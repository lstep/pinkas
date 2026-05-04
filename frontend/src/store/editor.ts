import { create } from 'zustand'

function getProviderUrl(): string {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}/collab`
}

interface EditorState {
  providerUrl: string
  docId: string
  userName: string
  permission: string
  setProviderUrl: (url: string) => void
  setDocId: (id: string) => void
  setPermission: (perm: string) => void
  setUserName: (name: string) => void
}

export const useEditorStore = create<EditorState>((set) => ({
  providerUrl: getProviderUrl(),
  docId: 'seed-page-001',
  userName: 'Anonymous',
  permission: 'editor',
  setProviderUrl: (providerUrl) => set({ providerUrl }),
  setDocId: (docId) => set({ docId }),
  setPermission: (permission) => set({ permission }),
  setUserName: (userName) => set({ userName }),
}))

import { create } from 'zustand'

export interface TreeNode {
  id: string
  type: 'page' | 'directory'
  title: string       // pages have title, directories have name (alias in UI)
  name?: string       // directories use name
  slug: string
  space_id: string
  parent_id: string | null   // for directories: parent directory; for pages: always null (pages use directory_id)
  directory_id: string | null // for pages: containing directory; for directories: null
  position: number
  icon?: string
  created_by: string
  created_at: number
  updated_at: number
  children?: TreeNode[]
  expanded?: boolean
  loading?: boolean
}

interface TreeState {
  nodes: Record<string, TreeNode>
  rootIds: string[]
  spaceId: string | null
  setSpaceId: (id: string) => void
  setNodes: (nodes: TreeNode[]) => void
  addNode: (node: TreeNode) => void
  updateNode: (id: string, updates: Partial<TreeNode>) => void
  removeNode: (id: string) => void
  setChildren: (parentId: string, children: TreeNode[]) => void
  toggleExpanded: (id: string) => void
  setLoading: (id: string, loading: boolean) => void
  moveNode: (id: string, newParentId: string | null, newPosition: number) => void
}

export const useTreeStore = create<TreeState>((set) => ({
  nodes: {},
  rootIds: [],
  spaceId: null,

  setSpaceId: (spaceId) => set({ spaceId, nodes: {}, rootIds: [] }),

  setNodes: (nodes) => {
    const map: Record<string, TreeNode> = {}
    const rootIds: string[] = []

    // First pass: add all directories (they form the tree structure)
    for (const n of nodes) {
      if (n.type === 'directory') {
        map[n.id] = { ...n, expanded: false, children: [] }
        if (!n.parent_id) {
          rootIds.push(n.id)
        }
      }
    }

    // Second pass: add all pages into their directory's children or root
    for (const n of nodes) {
      if (n.type === 'page') {
        map[n.id] = { ...n, expanded: false }
        if (n.directory_id && map[n.directory_id]) {
          // Add to directory's children
          const parent = map[n.directory_id]
          parent.children = parent.children || []
          parent.children.push(map[n.id])
        } else if (!n.directory_id) {
          // Add to root (orphan pages)
          rootIds.push(n.id)
        }
      }
    }

    // Sort rootIds by position
    rootIds.sort((a, b) => (map[a].position || 0) - (map[b].position || 0))

    // Sort children within each directory
    for (const id in map) {
      const node = map[id]
      if (node.type === 'directory' && node.children) {
        node.children.sort((a, b) => (a.position || 0) - (b.position || 0))
      }
    }

    set({ nodes: map, rootIds })
  },

  addNode: (node) => {
    set((state) => {
      const newNode = { ...node, expanded: false }
      const nodes = { ...state.nodes, [node.id]: newNode }
      let rootIds = state.rootIds

      if (node.type === 'directory') {
        // Directories use parent_id for tree structure
        if (!node.parent_id) {
          rootIds = [...rootIds, node.id]
          rootIds.sort((a, b) => (nodes[a].position || 0) - (nodes[b].position || 0))
        } else {
          const parent = state.nodes[node.parent_id]
          if (parent && parent.expanded) {
            const newChildren = [...(parent.children || []), newNode]
              .sort((a, b) => (a.position || 0) - (b.position || 0))
            nodes[node.parent_id] = { ...parent, children: newChildren }
          }
        }
      } else {
        // Pages use directory_id for tree structure
        if (node.directory_id) {
          const directory = state.nodes[node.directory_id]
          if (directory && directory.expanded) {
            const newChildren = [...(directory.children || []), newNode]
              .sort((a, b) => (a.position || 0) - (b.position || 0))
            nodes[node.directory_id] = { ...directory, children: newChildren }
          }
        } else {
          // Orphan page - add to root
          rootIds = [...rootIds, node.id]
          rootIds.sort((a, b) => (nodes[a].position || 0) - (nodes[b].position || 0))
        }
      }

      return { nodes, rootIds }
    })
  },

  updateNode: (id, updates) => {
    set((state) => {
      const node = state.nodes[id]
      if (!node) return state
      const nodes = { ...state.nodes, [id]: { ...node, ...updates } }
      return { nodes }
    })
  },

  removeNode: (id) => {
    set((state) => {
      const nodes = { ...state.nodes }
      const nodeToRemove = nodes[id]
      delete nodes[id]

      // Remove from rootIds
      let rootIds = state.rootIds.filter((rid) => rid !== id)

      // Remove from parent's children
      if (nodeToRemove) {
        if (nodeToRemove.type === 'directory' && nodeToRemove.parent_id) {
          const parent = nodes[nodeToRemove.parent_id]
          if (parent?.children) {
            parent.children = parent.children.filter((c) => c.id !== id)
          }
        } else if (nodeToRemove.type === 'page' && nodeToRemove.directory_id) {
          const directory = nodes[nodeToRemove.directory_id]
          if (directory?.children) {
            directory.children = directory.children.filter((c) => c.id !== id)
          }
        }
      }

      // Cascade: if removing a directory, also remove all its children (pages and subdirectories)
      const removeCascade = (nodeId: string) => {
        const node = state.nodes[nodeId]
        if (!node) return

        if (node.type === 'directory' && node.children) {
          for (const child of node.children) {
            delete nodes[child.id]
            removeCascade(child.id)
          }
        }
      }
      removeCascade(id)

      return { nodes, rootIds }
    })
  },

  setChildren: (parentId, children) => {
    set((state) => {
      const nodes = { ...state.nodes }
      const parent = nodes[parentId]
      if (!parent) return state

      // Add/update all children in the nodes map
      for (const c of children) {
        nodes[c.id] = { ...c, expanded: false }
      }

      // Sort children by position
      parent.children = children.sort((a, b) => (a.position || 0) - (b.position || 0))

      return { nodes }
    })
  },

  toggleExpanded: (id) => {
    set((state) => {
      const node = state.nodes[id]
      if (!node) return state
      const nodes = { ...state.nodes, [id]: { ...node, expanded: !node.expanded } }
      return { nodes }
    })
  },

  setLoading: (id, loading) => {
    set((state) => {
      const node = state.nodes[id]
      if (!node) return state
      const nodes = { ...state.nodes, [id]: { ...node, loading } }
      return { nodes }
    })
  },

  moveNode: (id, newParentId, newPosition) => {
    set((state) => {
      const node = state.nodes[id]
      if (!node) return state

      const nodes = { ...state.nodes }
      let rootIds = [...state.rootIds]

      if (node.type === 'directory') {
        // Directories: update parent_id
        const oldParentId = node.parent_id

        // Remove from old parent
        if (oldParentId) {
          const oldParent = nodes[oldParentId]
          if (oldParent?.children) {
            oldParent.children = oldParent.children.filter((c) => c.id !== id)
          }
        } else {
          rootIds = rootIds.filter((rid) => rid !== id)
        }

        // Add to new parent
        if (newParentId) {
          const newParent = nodes[newParentId]
          if (newParent) {
            newParent.children = [...(newParent.children || []), { ...node, parent_id: newParentId, position: newPosition }]
              .sort((a, b) => (a.position || 0) - (b.position || 0))
          }
        } else {
          rootIds = [...rootIds, id]
          rootIds.sort((a, b) => (nodes[a].position || 0) - (nodes[b].position || 0))
        }

        nodes[id] = { ...node, parent_id: newParentId, position: newPosition }
      } else {
        // Pages: update directory_id
        const oldDirectoryId = node.directory_id

        // Remove from old directory
        if (oldDirectoryId) {
          const oldDirectory = nodes[oldDirectoryId]
          if (oldDirectory?.children) {
            oldDirectory.children = oldDirectory.children.filter((c) => c.id !== id)
          }
        } else {
          rootIds = rootIds.filter((rid) => rid !== id)
        }

        // Add to new directory
        if (newParentId) {
          const newDirectory = nodes[newParentId]
          if (newDirectory) {
            newDirectory.children = [...(newDirectory.children || []), { ...node, directory_id: newParentId, position: newPosition }]
              .sort((a, b) => (a.position || 0) - (b.position || 0))
          }
        } else {
          rootIds = [...rootIds, id]
          rootIds.sort((a, b) => (nodes[a].position || 0) - (nodes[b].position || 0))
        }

        nodes[id] = { ...node, directory_id: newParentId, position: newPosition }
      }

      return { nodes, rootIds }
    })
  },
}))

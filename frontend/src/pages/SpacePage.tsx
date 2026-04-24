import { useEffect, useState, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { CollaborativeEditor } from '../editor/CollaborativeEditor'
import { useAuthStore } from '../store/auth'
import { useTreeStore, TreeNode } from '../store/tree'
import { useEditorStore } from '../store/editor'
import { Sidebar } from '../components/Sidebar/Sidebar'
import { Breadcrumb } from '../components/Breadcrumb/Breadcrumb'
import { listSpaces, listRootPages, createPage, getPageBySlug, Space } from '../api/pages'
import { listRootDirectories, createDirectory } from '../api/directories'
import { connectSSE, disconnectSSE } from '../api/sse'
import './SpacePage.css'

export function SpacePage() {
  const { spaceSlug, '*': pagePath } = useParams()
  const navigate = useNavigate()
  const user = useAuthStore((s) => s.user)
  const logout = useAuthStore((s) => s.logout)
  const accessToken = useAuthStore((s) => s.accessToken)
  const setSpaceId = useTreeStore((s) => s.setSpaceId)
  const setNodes = useTreeStore((s) => s.setNodes)
  const nodes = useTreeStore((s) => s.nodes)
  const setDocId = useEditorStore((s) => s.setDocId)

  const [space, setSpace] = useState<Space | null>(null)
  const [activePageId, setActivePageId] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [newPageParentId, setNewPageParentId] = useState<string | null>(null)
  const [showNewPageInput, setShowNewPageInput] = useState(false)
  const [newPageTitle, setNewPageTitle] = useState('')

  // Load space, root directories, and root pages
  useEffect(() => {
    async function init() {
      try {
        setLoading(true)
        const spaces = await listSpaces()
        const currentSpace = spaces.find((s) => s.slug === spaceSlug) || spaces[0]
        if (!currentSpace) {
          setLoading(false)
          return
        }
        setSpace(currentSpace)
        setSpaceId(currentSpace.id)

        // Fetch both root directories and root pages
        const [directories, pages] = await Promise.all([
          listRootDirectories(currentSpace.id),
          listRootPages(currentSpace.id),
        ])

        // Convert directories to TreeNodes
        const directoryNodes: TreeNode[] = directories.map(d => ({
          ...d,
          type: 'directory' as const,
          title: d.name,
          name: d.name,
          directory_id: null,
          expanded: false,
          loading: false,
        }))

        // Convert pages to TreeNodes
        const pageNodes: TreeNode[] = pages.map(p => ({
          ...p,
          type: 'page' as const,
          parent_id: null,
          expanded: false,
          loading: false,
          icon: p.icon ?? undefined,
        }))

        // Merge and set in tree store
        const allNodes = [...directoryNodes, ...pageNodes]
        setNodes(allNodes)

        // If no page selected, redirect to first page (if any)
        if (!pagePath && pages.length > 0) {
          navigate(`/s/${currentSpace.slug}/${pages[0].slug}`, { replace: true })
        }
      } catch (err: any) {
        console.error('Failed to load space:', err)
        // Redirect to login on auth failure
        if (err.message?.includes('Unauthorized') || err.message?.includes('401')) {
          logout()
          navigate('/login', { replace: true })
        }
      } finally {
        setLoading(false)
      }
    }
    init()
  }, [spaceSlug])

  // Resolve current page from URL
  useEffect(() => {
    if (!space || !pagePath) return
    const slug = pagePath.split('/')[0]
    const currentSpace = space
    async function loadPage() {
      // Try to find in loaded nodes first
      const node = Object.values(nodes).find((n) => n.slug === slug && n.space_id === currentSpace.id && n.type === 'page')
      if (node) {
        setActivePageId(node.id)
        setDocId(node.id)
        return
      }
      // Otherwise fetch by slug from backend
      try {
        const page = await getPageBySlug(currentSpace.id, slug)
        setActivePageId(page.id)
        setDocId(page.id)
      } catch (err) {
        console.error('Failed to load page by slug:', err)
      }
    }
    loadPage()
  }, [pagePath, space, nodes])

  // Connect SSE
  useEffect(() => {
    if (accessToken) {
      connectSSE(accessToken)
    }
    return () => disconnectSSE()
  }, [accessToken])

  const addNode = useTreeStore((s) => s.addNode)

  const handleCreateDirectory = useCallback(async (name: string, parentId: string | null = null) => {
    if (!space || !name.trim()) return
    try {
      const directory = await createDirectory({
        space_id: space.id,
        parent_id: parentId,
        name: name.trim(),
      })
      console.log('Directory created:', directory)

      // Add to tree store immediately (optimistic update)
      addNode({
        id: directory.id,
        type: 'directory',
        title: directory.name,
        name: directory.name,
        slug: directory.slug,
        icon: directory.icon,
        position: directory.position,
        parent_id: parentId,
        directory_id: null,
        space_id: space.id,
        created_by: directory.created_by,
        created_at: directory.created_at,
        updated_at: directory.updated_at,
        expanded: false,
        loading: false,
        children: [],
      })
    } catch (err: any) {
      console.error('Failed to create directory:', err)
      alert(err.message || 'Failed to create directory')
    }
  }, [space, addNode])

  const handleCreatePage = useCallback(async (title: string, directoryId: string | null = null) => {
    if (!space || !title.trim()) return
    try {
      const page = await createPage({
        space_id: space.id,
        title: title.trim(),
        directory_id: directoryId,
      })
      console.log('Page created:', page)
      setNewPageTitle('')
      setShowNewPageInput(false)
      setNewPageParentId(null)

      // Add to tree store immediately (optimistic update)
      addNode({
        id: page.id,
        type: 'page',
        title: page.title,
        slug: page.slug,
        icon: page.icon || undefined,
        position: page.position,
        parent_id: null,
        directory_id: directoryId,
        space_id: space.id,
        created_by: page.created_by,
        created_at: page.created_at,
        updated_at: page.updated_at,
        expanded: false,
        loading: false,
      })

      // Navigate to the new page
      navigate(`/s/${space.slug}/${page.slug}`)
    } catch (err: any) {
      console.error('Failed to create page:', err)
      alert(err.message || 'Failed to create page')
    }
  }, [space, navigate, addNode])

  if (loading) {
    return <div className="space-page"><p>Loading...</p></div>
  }

  if (!space) {
    return <div className="space-page"><p>No spaces found</p></div>
  }

  return (
    <div className="space-page">
      <header className="space-header">
        <div className="space-header-left">
          <h1>{space.name}</h1>
        </div>
        <div className="space-header-right">
          <span className="user-name">{user?.name || user?.email}</span>
          <button onClick={logout} className="btn-logout">Logout</button>
        </div>
      </header>
      <div className="space-body">
        <Sidebar
          activePageId={activePageId}
          spaceId={space?.id || null}
          onCreatePage={handleCreatePage}
          onCreateDirectory={handleCreateDirectory}
        />
        <div className="space-content">
          <Breadcrumb pageId={activePageId} spaceSlug={space.slug} />
          {activePageId ? (
            <CollaborativeEditor key={activePageId} />
          ) : (
            <div className="empty-page">
              <p>Select a page from the sidebar</p>
              <button
                className="btn-primary"
                onClick={() => {
                  setNewPageParentId(null)
                  setShowNewPageInput(true)
                }}
              >
                Create first page
              </button>
            </div>
          )}
        </div>
      </div>
      {showNewPageInput && (
        <div className="modal-overlay" onClick={() => setShowNewPageInput(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h3>New Page</h3>
            <input
              value={newPageTitle}
              onChange={(e) => setNewPageTitle(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleCreatePage(newPageTitle, newPageParentId)
                if (e.key === 'Escape') setShowNewPageInput(false)
              }}
              placeholder="Page title..."
              autoFocus
            />
            <div className="modal-actions">
              <button onClick={() => setShowNewPageInput(false)}>Cancel</button>
              <button className="btn-primary" onClick={() => handleCreatePage(newPageTitle, newPageParentId)}>Create</button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

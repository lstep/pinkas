import React, { useState, useRef } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
  DragOverlay,
  DragStartEvent,
} from '@dnd-kit/core'
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
  useSortable,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { useTreeStore, TreeNode as TreeNodeType } from '../../store/tree'
import { listPagesByDirectory, deletePage, updatePage, movePage } from '../../api/pages'
import { listDirectoryChildren, deleteDirectory, updateDirectory, moveDirectory } from '../../api/directories'
import './Sidebar.css'

interface SortableTreeNodeProps {
  node: TreeNodeType
  depth: number
  activePageId: string | null
  onCreatePage: (title: string, directoryId: string | null) => void
  onCreateDirectory: (name: string, parentId: string | null) => void
}

function SortableTreeNode({ node, depth, activePageId, onCreatePage, onCreateDirectory }: SortableTreeNodeProps) {
  const navigate = useNavigate()
  const { spaceSlug } = useParams()
  const toggleExpanded = useTreeStore((s) => s.toggleExpanded)
  const setChildren = useTreeStore((s) => s.setChildren)
  const setLoading = useTreeStore((s) => s.setLoading)
  const removeNode = useTreeStore((s) => s.removeNode)
  const [menuOpen, setMenuOpen] = useState(false)
  const [isEditing, setIsEditing] = useState(false)
  const [editTitle, setEditTitle] = useState(node.type === 'directory' ? node.name || node.title : node.title)
  const [showNewInput, setShowNewInput] = useState(false)
  const [newTitle, setNewTitle] = useState('')
  const [newInputIsDir, setNewInputIsDir] = useState(false)
  const [showIconPicker, setShowIconPicker] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)
  const iconPickerRef = useRef<HTMLDivElement>(null)

  const isDirectory = node.type === 'directory'
  const displayTitle = isDirectory ? (node.name || node.title) : node.title

  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: node.id })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  }

  const isActive = node.id === activePageId

  const handleToggle = async (e: React.MouseEvent) => {
    e.stopPropagation()
    if (!isDirectory) return
    if (!node.expanded) {
      setLoading(node.id, true)
      try {
        // Load both child directories and child pages
        const [childDirectories, childPages] = await Promise.all([
          listDirectoryChildren(node.id),
          listPagesByDirectory(node.id),
        ])

        // Convert to TreeNodes and merge
        const directoryNodes: TreeNodeType[] = childDirectories.map(d => ({
          ...d,
          type: 'directory' as const,
          title: d.name,
          name: d.name,
          directory_id: null,
          expanded: false,
          loading: false,
        }))

        const pageNodes: TreeNodeType[] = childPages.map(p => ({
          ...p,
          type: 'page' as const,
          parent_id: null,
          expanded: false,
          loading: false,
          icon: p.icon ?? undefined,
        }))

        // Merge and sort by position
        const allChildren = [...directoryNodes, ...pageNodes].sort((a, b) => (a.position || 0) - (b.position || 0))
        setChildren(node.id, allChildren)
      } catch (err) {
        console.error('Failed to load children:', err)
      } finally {
        setLoading(node.id, false)
      }
    }
    toggleExpanded(node.id)
  }

  const handleClick = () => {
    if (isDirectory) {
      handleToggle({ stopPropagation: () => {} } as any)
    } else {
      navigate(`/s/${spaceSlug}/${node.slug}`)
    }
  }

  const handleDelete = async () => {
    const itemType = isDirectory ? 'directory' : 'page'
    if (!window.confirm(`Delete "${displayTitle}" ${itemType}?`)) return
    try {
      if (isDirectory) {
        await deleteDirectory(node.id)
      } else {
        await deletePage(node.id)
      }
      removeNode(node.id)
    } catch (err) {
      alert(`Failed to delete ${itemType}`)
    }
    setMenuOpen(false)
  }

  const handleRename = async () => {
    if (editTitle.trim() && editTitle !== displayTitle) {
      try {
        if (isDirectory) {
          await updateDirectory(node.id, { name: editTitle.trim() })
          useTreeStore.getState().updateNode(node.id, { name: editTitle.trim(), title: editTitle.trim() })
        } else {
          await updatePage(node.id, { title: editTitle.trim() })
          useTreeStore.getState().updateNode(node.id, { title: editTitle.trim() })
        }
      } catch (err) {
        alert('Failed to rename')
        setEditTitle(displayTitle)
      }
    }
    setIsEditing(false)
    setMenuOpen(false)
  }

  const handleChangeIcon = async (icon: string) => {
    try {
      if (isDirectory) {
        await updateDirectory(node.id, { icon })
        useTreeStore.getState().updateNode(node.id, { icon })
      } else {
        await updatePage(node.id, { icon })
        useTreeStore.getState().updateNode(node.id, { icon })
      }
    } catch (err) {
      alert('Failed to change icon')
    }
    setShowIconPicker(false)
    setMenuOpen(false)
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') handleRename()
    if (e.key === 'Escape') {
      setEditTitle(displayTitle)
      setIsEditing(false)
    }
  }

  const handleNewItemKeyDown = (e: React.KeyboardEvent, isDir: boolean) => {
    if (e.key === 'Enter' && newTitle.trim()) {
      if (isDir) {
        onCreateDirectory(newTitle.trim(), node.id)
      } else {
        onCreatePage(newTitle.trim(), node.id)
      }
      setNewTitle('')
      setShowNewInput(false)
    }
    if (e.key === 'Escape') {
      setNewTitle('')
      setShowNewInput(false)
    }
  }

  return (
    <div ref={setNodeRef} style={style} {...attributes}>
      <div
        className={`tree-row ${isActive ? 'active' : ''}`}
        style={{ paddingLeft: `${depth * 16 + 8}px` }}
        onClick={handleClick}
      >
        <span className="drag-handle" {...listeners} title="Drag to reorder">
          ⋮⋮
        </span>
        <span className="tree-toggle" onClick={handleToggle}>
          {isDirectory ? (node.expanded ? '▼' : '▶') : ' '}
        </span>
        <span className="tree-icon">{node.icon || (isDirectory ? '📁' : '📄')}</span>
        {isEditing ? (
          <input
            className="tree-rename-input"
            value={editTitle}
            onChange={(e) => setEditTitle(e.target.value)}
            onKeyDown={handleKeyDown}
            onBlur={handleRename}
            autoFocus
          />
        ) : (
          <span className="tree-title">{displayTitle}</span>
        )}
        <span
          className="tree-menu-btn"
          onClick={(e) => {
            e.stopPropagation()
            setMenuOpen(!menuOpen)
          }}
        >
          ⋮
        </span>
        {menuOpen && (
          <div className="tree-menu" ref={menuRef}>
            <button onClick={() => { setIsEditing(true); setMenuOpen(false) }}>Rename</button>
            <button onClick={() => { setShowIconPicker(true) }}>Change Icon</button>
            {/* Only directories can have children */}
            {isDirectory && (
              <>
                <button onClick={() => { setShowNewInput(true); setNewInputIsDir(false); setMenuOpen(false) }}>New Page</button>
                <button onClick={() => { setShowNewInput(true); setNewInputIsDir(true); setMenuOpen(false) }}>New Directory</button>
              </>
            )}
            <button onClick={handleDelete} className="danger">Delete</button>
          </div>
        )}
        {showIconPicker && (
          <div className="icon-picker" ref={iconPickerRef}>
            {['📄','📁','🚀','⭐','🔥','💡','📝','📊','🎯','🔧','💻','🐛','✅','❌','⚠️','❓','💬','🔒','🌐','📅','📎','🏷️','📌','🔖'].map(icon => (
              <button key={icon} onClick={() => handleChangeIcon(icon)}>{icon}</button>
            ))}
            <button onClick={() => handleChangeIcon('')}>❌</button>
          </div>
        )}
      </div>
      {showNewInput && (
        <div className="tree-new-input" style={{ paddingLeft: `${(depth + 1) * 16 + 8}px` }}>
          <input
            value={newTitle}
            onChange={(e) => setNewTitle(e.target.value)}
            onKeyDown={(e) => handleNewItemKeyDown(e, newInputIsDir)}
            onBlur={() => { setShowNewInput(false); setNewTitle('') }}
            placeholder={newInputIsDir ? 'New directory name...' : 'New page title...'}
            autoFocus
          />
        </div>
      )}
      {node.expanded && node.children && node.children.length > 0 && (
        <SortableContext
          items={node.children.map((c) => c.id)}
          strategy={verticalListSortingStrategy}
        >
          <div className="tree-children">
            {node.children.map((child) => (
              <SortableTreeNode
                key={child.id}
                node={child}
                depth={depth + 1}
                activePageId={activePageId}
                onCreatePage={onCreatePage}
                onCreateDirectory={onCreateDirectory}
              />
            ))}
          </div>
        </SortableContext>
      )}
      {node.expanded && node.loading && (
        <div className="tree-loading" style={{ paddingLeft: `${(depth + 1) * 16 + 8}px` }}>
          Loading...
        </div>
      )}
    </div>
  )
}

interface SidebarProps {
  activePageId: string | null
  spaceId: string | null
  onCreatePage: (title: string, directoryId: string | null) => void
  onCreateDirectory: (name: string, parentId: string | null) => void
}

export const Sidebar: React.FC<SidebarProps> = ({ activePageId, spaceId, onCreatePage, onCreateDirectory }) => {
  const rootIds = useTreeStore((s) => s.rootIds)
  const nodes = useTreeStore((s) => s.nodes)
  const moveNode = useTreeStore((s) => s.moveNode)
  const [newTitle, setNewTitle] = useState('')
  const [showNewInput, setShowNewInput] = useState(false)
  const [newInputIsDir, setNewInputIsDir] = useState(false)
  const [activeId, setActiveId] = useState<string | null>(null)

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 8,
      },
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  )

  const handleDragStart = (event: DragStartEvent) => {
    setActiveId(event.active.id as string)
  }

  const handleDragEnd = async (event: DragEndEvent) => {
    const { active, over } = event
    setActiveId(null)

    if (!over || active.id === over.id) return

    const activeNode = nodes[active.id as string]
    const overNode = nodes[over.id as string]
    if (!activeNode || !overNode) return

    // Only allow reordering within the same parent/container for now
    // For directories: compare parent_id
    // For pages: compare directory_id
    let activeContainer: string | null
    let overContainer: string | null

    if (activeNode.type === 'directory') {
      activeContainer = activeNode.parent_id
    } else {
      activeContainer = activeNode.directory_id
    }

    if (overNode.type === 'directory') {
      overContainer = overNode.parent_id
    } else {
      overContainer = overNode.directory_id
    }

    if (activeContainer !== overContainer) {
      // Cross-container moves not yet implemented
      return
    }

    // Get sibling IDs based on container
    const siblingIds = activeContainer
      ? (nodes[activeContainer]?.children?.map((c) => c.id) || [])
      : rootIds

    const oldIndex = siblingIds.indexOf(active.id as string)
    const newIndex = siblingIds.indexOf(over.id as string)

    if (oldIndex === -1 || newIndex === -1) return

    const newSiblingIds = arrayMove(siblingIds, oldIndex, newIndex)

    // Update positions optimistically
    for (let i = 0; i < newSiblingIds.length; i++) {
      const id = newSiblingIds[i]
      const node = nodes[id]
      moveNode(id, activeContainer, i)
      try {
        if (node.type === 'directory') {
          await moveDirectory(id, activeContainer, i)
        } else {
          await movePage(id, activeContainer, i)
        }
      } catch (err) {
        console.error('Failed to update position:', err)
      }
    }
  }

  const handleCreate = async () => {
    if (!newTitle.trim() || !spaceId) {
      console.log('Create blocked: empty title or no spaceId', { newTitle, spaceId })
      return
    }
    try {
      if (newInputIsDir) {
        await onCreateDirectory(newTitle.trim(), null)
      } else {
        await onCreatePage(newTitle.trim(), null)
      }
      setShowNewInput(false)
      setNewTitle('')
      setNewInputIsDir(false)
    } catch (err: any) {
      console.error('Create error:', err)
      alert(err.message || 'Failed to create')
    }
  }

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCenter}
      onDragStart={handleDragStart}
      onDragEnd={handleDragEnd}
    >
      <aside className="sidebar">
        <div className="sidebar-header">
          <h2>Pages</h2>
          <button className="btn-new" onClick={() => setShowNewInput(true)}>+</button>
        </div>
        {showNewInput && (
          <div className="new-page-input">
            <input
              value={newTitle}
              onChange={(e) => setNewTitle(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleCreate()
                if (e.key === 'Escape') { setShowNewInput(false); setNewTitle(''); setNewInputIsDir(false) }
              }}
              placeholder={newInputIsDir ? 'Directory name...' : 'Page title...'}
              autoFocus
            />
            <div className="new-input-type-toggle">
              <label>
                <input
                  type="radio"
                  name="newType"
                  checked={!newInputIsDir}
                  onChange={() => setNewInputIsDir(false)}
                />
                Page
              </label>
              <label>
                <input
                  type="radio"
                  name="newType"
                  checked={newInputIsDir}
                  onChange={() => setNewInputIsDir(true)}
                />
                Directory
              </label>
            </div>
          </div>
        )}
        <div className="tree">
          <SortableContext items={rootIds} strategy={verticalListSortingStrategy}>
            {rootIds.map((id) => (
              <SortableTreeNode
                key={id}
                node={nodes[id]}
                depth={0}
                activePageId={activePageId}
                onCreatePage={onCreatePage}
                onCreateDirectory={onCreateDirectory}
              />
            ))}
          </SortableContext>
          {rootIds.length === 0 && <p className="empty">No pages yet</p>}
        </div>
      </aside>
      <DragOverlay>
        {activeId ? (
          <div className="tree-row drag-overlay">
            <span className="tree-icon">{nodes[activeId]?.icon || (nodes[activeId]?.type === 'directory' ? '📁' : '📄')}</span>
            <span className="tree-title">
              {nodes[activeId]?.type === 'directory'
                ? (nodes[activeId]?.name || nodes[activeId]?.title)
                : nodes[activeId]?.title}
            </span>
          </div>
        ) : null}
      </DragOverlay>
    </DndContext>
  )
}

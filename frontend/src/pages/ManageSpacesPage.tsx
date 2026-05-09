import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Space,
  listSpaces,
  createSpace,
  updateSpace,
  deleteSpace,
} from '../api/pages'
import { listPermissionsForTarget, listUsers, User, Permission } from '../api/admin'
import { useAuthStore } from '../store/auth'
import { Button, Input, Card, Modal } from '../components/ui'
import './ManageSpacesPage.css'

interface SpaceSettings {
  name: string
  slug: string
  icon: string
}

function levelLabel(level: number): string {
  switch (level) {
    case 1: return 'Viewer'
    case 2: return 'Editor'
    case 3: return 'Admin'
    default: return `Level ${level}`
  }
}

export function ManageSpacesPage() {
  const navigate = useNavigate()
  const user = useAuthStore((s) => s.user)
  const isAdmin = user?.role === 'admin'

  const [spaces, setSpaces] = useState<Space[]>([])
  const [loading, setLoading] = useState(true)

  // Create form
  const [showCreate, setShowCreate] = useState(false)
  const [newName, setNewName] = useState('')
  const [newSlug, setNewSlug] = useState('')
  const [newIcon, setNewIcon] = useState('')
  const [creating, setCreating] = useState(false)

  // Settings modal
  const [settingsSpace, setSettingsSpace] = useState<Space | null>(null)
  const [settingsForm, setSettingsForm] = useState<SpaceSettings>({ name: '', slug: '', icon: '' })
  const [saving, setSaving] = useState(false)

  // Members in settings modal
  const [members, setMembers] = useState<Permission[]>([])
  const [users, setUsers] = useState<User[]>([])
  const [membersLoading, setMembersLoading] = useState(false)

  const loadSpaces = async () => {
    try {
      const data = await listSpaces()
      setSpaces(data)
    } catch {
      // silent
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadSpaces()
    listUsers().then(setUsers).catch(() => {})
  }, [])

  if (!isAdmin) {
    return (
      <div className="manage-spaces-page">
        <Card className="manage-unauthorized" padding="lg">
          <h1>Access Denied</h1>
          <p>You need admin privileges to manage spaces.</p>
          <Button variant="primary" onClick={() => navigate('/')}>Back to Home</Button>
        </Card>
      </div>
    )
  }

  const handleCreate = async () => {
    if (!newName.trim()) return
    setCreating(true)
    try {
      await createSpace(newName.trim(), newSlug.trim() || undefined, newIcon.trim() || undefined)
      setNewName('')
      setNewSlug('')
      setNewIcon('')
      setShowCreate(false)
      await loadSpaces()
    } catch (err: any) {
      alert(err.message || 'Failed to create space')
    } finally {
      setCreating(false)
    }
  }

  const handleDelete = async (space: Space) => {
    if (!confirm(`Delete space "${space.name}"? This will remove all pages, directories, and permissions.`)) return
    try {
      await deleteSpace(space.id)
      await loadSpaces()
    } catch (err: any) {
      alert(err.message || 'Failed to delete space')
    }
  }

  const openSettings = async (space: Space) => {
    setSettingsSpace(space)
    setSettingsForm({ name: space.name, slug: space.slug, icon: space.icon || '' })
    setMembersLoading(true)
    try {
      const [perms] = await Promise.all([
        listPermissionsForTarget('space', space.id),
      ])
      setMembers(perms)
    } catch {
      setMembers([])
    } finally {
      setMembersLoading(false)
    }
  }

  const handleSaveSettings = async () => {
    if (!settingsSpace) return
    setSaving(true)
    try {
      const updated = await updateSpace(settingsSpace.id, {
        name: settingsForm.name.trim() || undefined,
        slug: settingsForm.slug.trim() || undefined,
        icon: settingsForm.icon.trim() || undefined,
      })
      setSpaces((prev) => prev.map((s) => s.id === updated.id ? { ...s, ...updated } : s))
      setSettingsSpace(null)
    } catch (err: any) {
      alert(err.message || 'Failed to update space')
    } finally {
      setSaving(false)
    }
  }

  const getUserName = (userId: string): string => {
    const u = users.find((u) => u.id === userId)
    return u?.name || u?.email || userId
  }

  return (
    <div className="manage-spaces-page">
      <div className="manage-header">
        <button className="manage-back" onClick={() => navigate('/')}>← Back</button>
        <h1>Manage Spaces</h1>
        <Button variant="primary" onClick={() => setShowCreate(true)}>
          Create Space
        </Button>
      </div>

      {loading ? (
        <div className="manage-loading">
          {[1, 2, 3].map((i) => <div key={i} className="skeleton" style={{ height: 48, marginBottom: 8 }} />)}
        </div>
      ) : (
        <Card padding="none" className="manage-table-card">
          <table className="manage-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Slug</th>
                <th>Default Permission</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {spaces.length === 0 && (
                <tr><td colSpan={4} className="empty">No spaces yet</td></tr>
              )}
              {spaces.map((space) => (
                <tr key={space.id}>
                  <td>
                    <span className="manage-space-icon">{space.icon}</span>
                    <span className="manage-space-name">{space.name}</span>
                  </td>
                  <td><code>{space.slug}</code></td>
                  <td>{space.default_permission}</td>
                  <td className="manage-actions">
                    <Button variant="secondary" size="sm" onClick={() => openSettings(space)}>
                      Settings
                    </Button>
                    <Button variant="danger" size="sm" onClick={() => handleDelete(space)}>
                      Delete
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </Card>
      )}

      {/* Create Modal */}
      <Modal
        isOpen={showCreate}
        onClose={() => setShowCreate(false)}
        title="Create Space"
        footer={
          <div className="modal-footer-buttons">
            <Button variant="secondary" onClick={() => setShowCreate(false)}>Cancel</Button>
            <Button variant="primary" onClick={handleCreate} loading={creating} disabled={!newName.trim()}>
              Create
            </Button>
          </div>
        }
      >
        <div className="create-form">
          <div className="form-field">
            <label htmlFor="space-name">Name *</label>
            <Input
              id="space-name"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              placeholder="e.g. Engineering Wiki"
              onKeyDown={(e) => e.key === 'Enter' && handleCreate()}
            />
          </div>
          <div className="form-field">
            <label htmlFor="space-slug">Slug (optional)</label>
            <Input
              id="space-slug"
              value={newSlug}
              onChange={(e) => setNewSlug(e.target.value)}
              placeholder="e.g. engineering"
            />
          </div>
          <div className="form-field">
            <label htmlFor="space-icon">Icon (optional)</label>
            <Input
              id="space-icon"
              value={newIcon}
              onChange={(e) => setNewIcon(e.target.value)}
              placeholder="e.g. 🚀 or emoji name"
            />
          </div>
        </div>
      </Modal>

      {/* Settings Modal */}
      <Modal
        isOpen={settingsSpace !== null}
        onClose={() => setSettingsSpace(null)}
        title={settingsSpace ? `Settings: ${settingsSpace.name}` : 'Space Settings'}
        footer={
          <div className="modal-footer-buttons">
            <Button variant="secondary" onClick={() => setSettingsSpace(null)}>Cancel</Button>
            <Button variant="primary" onClick={handleSaveSettings} loading={saving}>
              Save
            </Button>
          </div>
        }
      >
        <div className="settings-form">
          <h3 className="settings-section-title">General</h3>
          <div className="form-field">
            <label htmlFor="settings-name">Name</label>
            <Input
              id="settings-name"
              value={settingsForm.name}
              onChange={(e) => setSettingsForm((p) => ({ ...p, name: e.target.value }))}
            />
          </div>
          <div className="form-field">
            <label htmlFor="settings-slug">Slug</label>
            <Input
              id="settings-slug"
              value={settingsForm.slug}
              onChange={(e) => setSettingsForm((p) => ({ ...p, slug: e.target.value }))}
            />
          </div>
          <div className="form-field">
            <label htmlFor="settings-icon">Icon</label>
            <Input
              id="settings-icon"
              value={settingsForm.icon}
              onChange={(e) => setSettingsForm((p) => ({ ...p, icon: e.target.value }))}
            />
          </div>

          <h3 className="settings-section-title">Members</h3>
          {membersLoading ? (
            <p className="settings-loading-text">Loading members...</p>
          ) : members.length === 0 ? (
            <p className="settings-empty-text">No members found</p>
          ) : (
            <table className="members-table">
              <thead>
                <tr>
                  <th>User</th>
                  <th>Level</th>
                </tr>
              </thead>
              <tbody>
                {members.map((perm, idx) => (
                  <tr key={`${perm.granteeId}-${idx}`}>
                    <td>{getUserName(perm.granteeId)}</td>
                    <td>{levelLabel(perm.level)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </Modal>
    </div>
  )
}

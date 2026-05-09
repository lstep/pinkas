import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '../store/auth'
import {
  listUsers,
  listGroups,
  createGroup,
  deleteGroup,
  listGroupMembers,
  addGroupMember,
  removeGroupMember,
  updateUser,
  deleteUser,
  listPermissionsForTarget,
  setPermission,
  removePermission,
  inviteUser,
  InviteResponse,
  Permission,
  User,
  Group,
  GroupMember,
} from '../api/admin'
import {
  createMCPToken,
  listMCPTokens,
  deleteMCPToken,
  MCPToken,
  CreateTokenRequest,
} from '../api/mcpTokens'
import { Button, Input, Card, Badge, Modal } from '../components/ui'
import './SettingsPage.css'

export function SettingsPage() {
  const navigate = useNavigate()
  const currentUser = useAuthStore((s) => s.user)
  const logout = useAuthStore((s) => s.logout)

  const [users, setUsers] = useState<User[]>([])
  const [groups, setGroups] = useState<Group[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<'users' | 'groups' | 'permissions' | 'tokens'>('users')

  // Group management state
  const [newGroupName, setNewGroupName] = useState('')
  const [selectedGroup, setSelectedGroup] = useState<string | null>(null)
  const [groupMembers, setGroupMembers] = useState<GroupMember[]>([])
  const [newMemberId, setNewMemberId] = useState('')

  // Permissions state
  const [permTargetType, setPermTargetType] = useState('space')
  const [permTargetId, setPermTargetId] = useState('')
  const [permissions, setPermissions] = useState<Permission[]>([])
  const [newPermGranteeType, setNewPermGranteeType] = useState<'user' | 'group'>('user')
  const [newPermGranteeId, setNewPermGranteeId] = useState('')
  const [newPermLevel, setNewPermLevel] = useState(1)

  // Invite user state
  const [inviteEmail, setInviteEmail] = useState('')
  const [inviteName, setInviteName] = useState('')
  const [inviteRole, setInviteRole] = useState('viewer')
  const [isInviting, setIsInviting] = useState(false)
  const [inviteResponse, setInviteResponse] = useState<InviteResponse | null>(null)
  const [showInviteModal, setShowInviteModal] = useState(false)

  // MCP token management state
  const [tokens, setTokens] = useState<MCPToken[]>([])
  const [newTokenName, setNewTokenName] = useState('')
  const [newTokenScopes, setNewTokenScopes] = useState<string[]>(['read'])
  const [newTokenSpaceId, setNewTokenSpaceId] = useState('')
  const [newTokenExpiry, setNewTokenExpiry] = useState('')
  const [isCreatingToken, setIsCreatingToken] = useState(false)
  const [showTokenSecretModal, setShowTokenSecretModal] = useState(false)
  const [createdTokenSecret, setCreatedTokenSecret] = useState('')
  const [createdTokenName, setCreatedTokenName] = useState('')

  const toggleScope = (scope: string) => {
    setNewTokenScopes((prev) =>
      prev.includes(scope) ? prev.filter((s) => s !== scope) : [...prev, scope]
    )
  }

  const handleCreateToken = async () => {
    if (!newTokenName.trim()) return
    setIsCreatingToken(true)
    try {
      const req: CreateTokenRequest = {
        name: newTokenName.trim(),
        scopes: newTokenScopes,
        spaceId: newTokenSpaceId.trim() || undefined,
        expiresInDays: newTokenExpiry ? parseInt(newTokenExpiry, 10) : undefined,
      }
      const resp = await createMCPToken(req)
      setCreatedTokenName(resp.token.name)
      setCreatedTokenSecret(resp.secret)
      setShowTokenSecretModal(true)
      setNewTokenName('')
      setNewTokenScopes(['read'])
      setNewTokenSpaceId('')
      setNewTokenExpiry('')
      await loadTokens()
    } catch (err: any) {
      alert(err.message || 'Failed to create token')
    } finally {
      setIsCreatingToken(false)
    }
  }

  const handleDeleteToken = async (token: MCPToken) => {
    if (!confirm(`Are you sure you want to revoke the token "${token.name}"? This action cannot be undone.`)) return
    try {
      await deleteMCPToken(token.id)
      setTokens((prev) => prev.filter((t) => t.id !== token.id))
    } catch (err: any) {
      alert(err.message || 'Failed to delete token')
    }
  }

  const loadTokens = async () => {
    try {
      const data = await listMCPTokens()
      setTokens(data)
    } catch (err: any) {
      console.error('Failed to load tokens:', err)
    }
  }

  const formatDate = (ts: number) => {
    if (!ts) return 'Never'
    return new Date(ts * 1000).toLocaleDateString(undefined, {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    })
  }

  const formatScopes = (scopesJson: string) => {
    try {
      const scopes: string[] = JSON.parse(scopesJson)
      return scopes.join(', ')
    } catch {
      return scopesJson
    }
  }

  // Redirect non-admins
  if (currentUser?.role !== 'admin') {
    return (
      <div className="settings-page">
        <Card className="settings-unauthorized" padding="lg">
          <h1>Settings</h1>
          <p>You do not have permission to access this page.</p>
          <Button variant="primary" onClick={() => navigate('/')}>
            Back to Space
          </Button>
        </Card>
      </div>
    )
  }

  const loadData = async () => {
    setLoading(true)
    setError(null)
    try {
      const [usersData, groupsData, tokensData] = await Promise.all([listUsers(), listGroups(), listMCPTokens()])
      setUsers(usersData)
      setGroups(groupsData)
      setTokens(tokensData)
    } catch (err: any) {
      setError(err.message || 'Failed to load data')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadData()
  }, [])

  const handleRoleChange = async (userId: string, role: string) => {
    try {
      const updated = await updateUser(userId, { role })
      setUsers((prev) => prev.map((u) => (u.id === userId ? updated : u)))
    } catch (err: any) {
      alert(err.message || 'Failed to update role')
    }
  }

  const handleDeleteUser = async (userId: string) => {
    if (!confirm('Are you sure you want to delete this user? This cannot be undone.')) return
    try {
      await deleteUser(userId)
      setUsers((prev) => prev.filter((u) => u.id !== userId))
    } catch (err: any) {
      alert(err.message || 'Failed to delete user')
    }
  }

  const handleCreateGroup = async () => {
    if (!newGroupName.trim()) return
    try {
      await createGroup(newGroupName.trim())
      setNewGroupName('')
      await loadData()
    } catch (err: any) {
      alert(err.message || 'Failed to create group')
    }
  }

  const handleDeleteGroup = async (groupId: string) => {
    if (!confirm('Are you sure you want to delete this group?')) return
    try {
      await deleteGroup(groupId)
      setGroups((prev) => prev.filter((g) => g.id !== groupId))
      if (selectedGroup === groupId) {
        setSelectedGroup(null)
        setGroupMembers([])
      }
    } catch (err: any) {
      alert(err.message || 'Failed to delete group')
    }
  }

  const loadGroupMembers = async (groupId: string) => {
    try {
      const members = await listGroupMembers(groupId)
      setGroupMembers(members)
    } catch (err: any) {
      alert(err.message || 'Failed to load group members')
    }
  }

  const handleAddMember = async () => {
    if (!newMemberId || !selectedGroup) return
    try {
      await addGroupMember(selectedGroup, newMemberId)
      setNewMemberId('')
      await loadGroupMembers(selectedGroup)
    } catch (err: any) {
      alert(err.message || 'Failed to add member')
    }
  }

  const handleRemoveMember = async (userId: string) => {
    if (!selectedGroup) return
    try {
      await removeGroupMember(selectedGroup, userId)
      setGroupMembers((prev) => prev.filter((m) => m.id !== userId))
    } catch (err: any) {
      alert(err.message || 'Failed to remove member')
    }
  }

  const loadPermissions = async () => {
    if (!permTargetId.trim()) return
    try {
      const perms = await listPermissionsForTarget(permTargetType, permTargetId.trim())
      setPermissions(perms)
    } catch (err: any) {
      alert(err.message || 'Failed to load permissions')
    }
  }

  const handleAddPermission = async () => {
    if (!permTargetId.trim() || !newPermGranteeId.trim()) return
    try {
      await setPermission(permTargetType, permTargetId.trim(), newPermGranteeType, newPermGranteeId.trim(), newPermLevel)
      setNewPermGranteeId('')
      await loadPermissions()
    } catch (err: any) {
      alert(err.message || 'Failed to set permission')
    }
  }

  const handleRemovePermission = async (granteeType: string, granteeId: string) => {
    try {
      await removePermission(permTargetType, permTargetId.trim(), granteeType, granteeId)
      await loadPermissions()
    } catch (err: any) {
      alert(err.message || 'Failed to remove permission')
    }
  }

  const handleInviteUser = async () => {
    if (!inviteEmail.trim()) {
      alert('Email is required')
      return
    }
    setIsInviting(true)
    try {
      const response = await inviteUser(inviteEmail.trim(), inviteName.trim() || undefined, inviteRole)
      setInviteResponse(response)
      setShowInviteModal(true)
      // Reset form
      setInviteEmail('')
      setInviteName('')
      setInviteRole('viewer')
    } catch (err: any) {
      alert(err.message || 'Failed to invite user')
    } finally {
      setIsInviting(false)
    }
  }

  const handleCopyPassword = () => {
    if (inviteResponse?.tempPassword) {
      navigator.clipboard.writeText(inviteResponse.tempPassword)
      alert('Password copied to clipboard!')
    }
  }

  const handleCloseInviteModal = () => {
    setShowInviteModal(false)
    setInviteResponse(null)
    // Refresh user list
    loadData()
  }

  const levelLabel = (level: number) => {
    switch (level) {
      case 1: return 'Viewer'
      case 2: return 'Editor'
      case 3: return 'Admin'
      default: return 'None'
    }
  }

  const levelBadgeVariant = (level: number): 'viewer' | 'editor' | 'admin' | 'default' => {
    switch (level) {
      case 1: return 'viewer'
      case 2: return 'editor'
      case 3: return 'admin'
      default: return 'default'
    }
  }

  return (
    <div className="settings-page">
      <header className="settings-header">
        <h1>Settings</h1>
        <nav className="settings-nav">
          <Button variant="ghost" size="sm" onClick={() => navigate('/')}>
            ← Back to Space
          </Button>
          <Button variant="ghost" size="sm" onClick={logout}>
            Logout
          </Button>
        </nav>
      </header>

      <div className="settings-tabs">
        <button
          className={`settings-tab ${activeTab === 'users' ? 'active' : ''}`}
          onClick={() => setActiveTab('users')}
        >
          Users ({users.length})
        </button>
        <button
          className={`settings-tab ${activeTab === 'groups' ? 'active' : ''}`}
          onClick={() => setActiveTab('groups')}
        >
          Groups ({groups.length})
        </button>
        <button
          className={`settings-tab ${activeTab === 'permissions' ? 'active' : ''}`}
          onClick={() => setActiveTab('permissions')}
        >
          Permissions
        </button>
        <button
          className={`settings-tab ${activeTab === 'tokens' ? 'active' : ''}`}
          onClick={() => setActiveTab('tokens')}
        >
          API Tokens ({tokens.length})
        </button>
      </div>

      {error && (
        <div className="settings-error" role="alert">
          {error}
        </div>
      )}

      {loading ? (
        <div className="settings-loading">
          <div className="skeleton" style={{ width: 200, height: 24, marginBottom: 16 }} />
          <div className="skeleton" style={{ width: '100%', height: 40, marginBottom: 8 }} />
          <div className="skeleton" style={{ width: '100%', height: 40, marginBottom: 8 }} />
          <div className="skeleton" style={{ width: '100%', height: 40 }} />
        </div>
      ) : activeTab === 'users' ? (
        <div className="settings-section">
          {/* Invite User Card */}
          <Card className="invite-user-card" padding="md">
            <h3 className="invite-user-title">Invite User</h3>
            <div className="invite-user-form">
              <div className="invite-field">
                <label htmlFor="invite-email">Email *</label>
                <Input
                  id="invite-email"
                  type="email"
                  value={inviteEmail}
                  onChange={(e) => setInviteEmail(e.target.value)}
                  placeholder="user@example.com"
                  onKeyDown={(e) => e.key === 'Enter' && handleInviteUser()}
                />
              </div>
              <div className="invite-field">
                <label htmlFor="invite-name">Name</label>
                <Input
                  id="invite-name"
                  type="text"
                  value={inviteName}
                  onChange={(e) => setInviteName(e.target.value)}
                  placeholder="Optional display name"
                />
              </div>
              <div className="invite-field">
                <label htmlFor="invite-role">Role</label>
                <select
                  id="invite-role"
                  value={inviteRole}
                  onChange={(e) => setInviteRole(e.target.value)}
                  className="invite-role-select"
                >
                  <option value="viewer">Viewer</option>
                  <option value="editor">Editor</option>
                  <option value="admin">Admin</option>
                </select>
              </div>
              <Button
                variant="primary"
                onClick={handleInviteUser}
                loading={isInviting}
                disabled={!inviteEmail.trim()}
              >
                Invite
              </Button>
            </div>
          </Card>

          {/* Users Table */}
          <Card className="users-table-card" padding="none">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Email</th>
                  <th>Name</th>
                  <th>Role</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {users.map((u) => (
                  <tr key={u.id}>
                    <td>{u.email}</td>
                    <td>{u.name}</td>
                    <td>
                      <Badge variant={u.role as 'admin' | 'editor' | 'viewer' || 'default'}>
                        {u.role}
                      </Badge>
                    </td>
                    <td>
                      <div className="table-actions">
                        <select
                          value={u.role}
                          onChange={(e) => handleRoleChange(u.id, e.target.value)}
                          disabled={u.id === currentUser?.id}
                        >
                          <option value="viewer">Viewer</option>
                          <option value="editor">Editor</option>
                          <option value="admin">Admin</option>
                        </select>
                        <Button
                          variant="danger"
                          size="sm"
                          onClick={() => handleDeleteUser(u.id)}
                          disabled={u.id === currentUser?.id}
                        >
                          Delete
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </Card>

          {/* Invite Success Modal */}
          <Modal
            isOpen={showInviteModal}
            onClose={handleCloseInviteModal}
            title="User Invited"
            footer={
              <Button variant="primary" onClick={handleCloseInviteModal}>
                Done
              </Button>
            }
          >
            {inviteResponse && (
              <div className="invite-success-content">
                <div className="invite-success-row">
                  <span className="invite-success-label">Email:</span>
                  <span className="invite-success-value">{inviteResponse.email}</span>
                </div>
                {inviteResponse.name && (
                  <div className="invite-success-row">
                    <span className="invite-success-label">Name:</span>
                    <span className="invite-success-value">{inviteResponse.name}</span>
                  </div>
                )}
                <div className="invite-success-row">
                  <span className="invite-success-label">Role:</span>
                  <span className="invite-success-value">
                    <Badge variant={inviteResponse.role as 'admin' | 'editor' | 'viewer' || 'default'}>
                      {inviteResponse.role}
                    </Badge>
                  </span>
                </div>
                <div className="invite-password-section">
                  <span className="invite-success-label">Temporary Password:</span>
                  <div className="invite-password-box">
                    <code className="invite-password-code">{inviteResponse.tempPassword}</code>
                    <Button variant="secondary" size="sm" onClick={handleCopyPassword}>
                      Copy Password
                    </Button>
                  </div>
                  <p className="invite-password-hint">
                    Share this password with the user. They should change it after their first login.
                  </p>
                </div>
              </div>
            )}
          </Modal>
        </div>
      ) : activeTab === 'groups' ? (
        <div className="settings-section">
          <Card className="create-group-card" padding="md">
            <div className="create-group">
              <Input
                type="text"
                value={newGroupName}
                onChange={(e) => setNewGroupName(e.target.value)}
                placeholder="New group name..."
                onKeyDown={(e) => e.key === 'Enter' && handleCreateGroup()}
              />
              <Button variant="primary" onClick={handleCreateGroup}>
                Create Group
              </Button>
            </div>
          </Card>

          <div className="groups-list">
            {groups.map((g) => (
              <Card
                key={g.id}
                className={`group-item ${selectedGroup === g.id ? 'selected' : ''}`}
                padding="none"
              >
                <div
                  className="group-header"
                  onClick={() => {
                    setSelectedGroup(g.id)
                    loadGroupMembers(g.id)
                  }}
                >
                  <span className="group-name">{g.name}</span>
                  <Button
                    variant="danger"
                    size="sm"
                    onClick={(e) => {
                      e.stopPropagation()
                      handleDeleteGroup(g.id)
                    }}
                  >
                    Delete
                  </Button>
                </div>

                {selectedGroup === g.id && (
                  <div className="group-members">
                    <div className="add-member">
                      <select
                        value={newMemberId}
                        onChange={(e) => setNewMemberId(e.target.value)}
                      >
                        <option value="">Select user...</option>
                        {users
                          .filter((u) => !groupMembers.find((m) => m.id === u.id))
                          .map((u) => (
                            <option key={u.id} value={u.id}>
                              {u.email} ({u.name || 'no name'})
                            </option>
                          ))}
                      </select>
                      <Button variant="primary" size="sm" onClick={handleAddMember}>
                        Add
                      </Button>
                    </div>
                    <ul className="members-list">
                      {groupMembers.map((m) => (
                        <li key={m.id} className="member-item">
                          <span>
                            {m.email} ({m.name || 'no name'})
                          </span>
                          <Button
                            variant="danger"
                            size="sm"
                            onClick={() => handleRemoveMember(m.id)}
                          >
                            Remove
                          </Button>
                        </li>
                      ))}
                      {groupMembers.length === 0 && (
                        <li className="empty">No members yet</li>
                      )}
                    </ul>
                  </div>
                )}
              </Card>
            ))}
            {groups.length === 0 && <p className="empty">No groups created yet</p>}
          </div>
        </div>
      ) : activeTab === 'tokens' ? (
        <div className="settings-section">
          {/* Create Token Card */}
          <Card className="invite-user-card" padding="md">
            <h3 className="invite-user-title">Create API Token</h3>
            <div className="invite-user-form">
              <div className="invite-field">
                <label htmlFor="token-name">Name *</label>
                <Input
                  id="token-name"
                  type="text"
                  value={newTokenName}
                  onChange={(e) => setNewTokenName(e.target.value)}
                  placeholder="e.g. CLI integration"
                  onKeyDown={(e) => e.key === 'Enter' && handleCreateToken()}
                />
              </div>
              <div className="invite-field">
                <label>Scopes</label>
                <div className="token-scopes">
                  <label className="token-scope-check">
                    <input
                      type="checkbox"
                      checked={newTokenScopes.includes('read')}
                      onChange={() => toggleScope('read')}
                    />
                    read
                  </label>
                  <label className="token-scope-check">
                    <input
                      type="checkbox"
                      checked={newTokenScopes.includes('write')}
                      onChange={() => toggleScope('write')}
                    />
                    write
                  </label>
                  <label className="token-scope-check">
                    <input
                      type="checkbox"
                      checked={newTokenScopes.includes('admin')}
                      onChange={() => toggleScope('admin')}
                    />
                    admin
                  </label>
                </div>
              </div>
              <div className="invite-field">
                <label htmlFor="token-space">Space ID (optional)</label>
                <Input
                  id="token-space"
                  type="text"
                  value={newTokenSpaceId}
                  onChange={(e) => setNewTokenSpaceId(e.target.value)}
                  placeholder="Restrict to a space UUID"
                />
              </div>
              <div className="invite-field">
                <label htmlFor="token-expiry">Expires in (days, optional)</label>
                <Input
                  id="token-expiry"
                  type="number"
                  value={newTokenExpiry}
                  onChange={(e) => setNewTokenExpiry(e.target.value)}
                  placeholder="Leave empty for no expiry"
                  min="1"
                />
              </div>
              <Button
                variant="primary"
                onClick={handleCreateToken}
                loading={isCreatingToken}
                disabled={!newTokenName.trim()}
              >
                Create Token
              </Button>
            </div>
          </Card>

          {/* Tokens Table */}
          <Card className="users-table-card" padding="none">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Prefix</th>
                  <th>Scopes</th>
                  <th>Space</th>
                  <th>Created</th>
                  <th>Last Used</th>
                  <th>Expires</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {tokens.map((t) => (
                  <tr key={t.id}>
                    <td style={{ fontWeight: 500 }}>{t.name}</td>
                    <td><code className="token-prefix">{t.tokenPrefix}...</code></td>
                    <td><span className="token-scopes-label">{formatScopes(t.scopes)}</span></td>
                    <td>{t.spaceId || <span className="empty-inline">All spaces</span>}</td>
                    <td>{formatDate(t.createdAt)}</td>
                    <td>{t.lastUsedAt ? formatDate(t.lastUsedAt) : <span className="empty-inline">Never</span>}</td>
                    <td>{t.expiresAt ? formatDate(t.expiresAt) : <span className="empty-inline">Never</span>}</td>
                    <td>
                      <Button variant="danger" size="sm" onClick={() => handleDeleteToken(t)}>
                        Revoke
                      </Button>
                    </td>
                  </tr>
                ))}
                {tokens.length === 0 && (
                  <tr>
                    <td colSpan={8} className="empty">No API tokens yet</td>
                  </tr>
                )}
              </tbody>
            </table>
          </Card>

          {/* Token Secret Modal */}
          <Modal
            isOpen={showTokenSecretModal}
            onClose={() => {
              setShowTokenSecretModal(false)
              setCreatedTokenSecret('')
            }}
            title="API Token Created"
            footer={
              <Button
                variant="primary"
                onClick={() => {
                  setShowTokenSecretModal(false)
                  setCreatedTokenSecret('')
                }}
              >
                Done
              </Button>
            }
          >
            <div className="token-secret-content">
              <p className="token-secret-intro">
                Token <strong>{createdTokenName}</strong> has been created.
                Copy the secret below — it will <strong>never be shown again</strong>.
              </p>
              <div className="token-secret-box">
                <code className="token-secret-code">{createdTokenSecret}</code>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => {
                    navigator.clipboard.writeText(createdTokenSecret)
                    alert('Token secret copied to clipboard!')
                  }}
                >
                  Copy
                </Button>
              </div>
            </div>
          </Modal>
        </div>
      ) : (
        <div className="settings-section">
          <Card className="perm-target-card" padding="md">
            <div className="perm-target-form">
              <label>
                Target Type:
                <select value={permTargetType} onChange={(e) => setPermTargetType(e.target.value)}>
                  <option value="space">Space</option>
                  <option value="directory">Directory</option>
                  <option value="page">Page</option>
                </select>
              </label>
              <label>
                Target ID:
                <Input
                  type="text"
                  value={permTargetId}
                  onChange={(e) => setPermTargetId(e.target.value)}
                  placeholder="UUID or slug..."
                />
              </label>
              <Button variant="primary" onClick={loadPermissions}>
                Load Permissions
              </Button>
            </div>
          </Card>

          <Card className="perm-list-card" padding="md">
            <h3>Current Permissions</h3>
            {permissions.length === 0 ? (
              <p className="empty">No explicit permissions set. Default permission from space applies.</p>
            ) : (
              <table className="data-table perms-table">
                <thead>
                  <tr>
                    <th>Grantee Type</th>
                    <th>Grantee ID</th>
                    <th>Level</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {permissions.map((p, i) => (
                    <tr key={i}>
                      <td>{p.granteeType}</td>
                      <td>
                        {p.granteeType === 'user'
                          ? (users.find((u) => u.id === p.granteeId)?.email || p.granteeId)
                          : (groups.find((g) => g.id === p.granteeId)?.name || p.granteeId)
                        }
                      </td>
                      <td>
                        <Badge variant={levelBadgeVariant(p.level)}>
                          {levelLabel(p.level)}
                        </Badge>
                      </td>
                      <td>
                        <Button
                          variant="danger"
                          size="sm"
                          onClick={() => handleRemovePermission(p.granteeType, p.granteeId)}
                        >
                          Remove
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </Card>

          <Card className="perm-add-card" padding="md">
            <h3>Add / Edit Permission</h3>
            <div className="perm-add-row">
              <select value={newPermGranteeType} onChange={(e) => setNewPermGranteeType(e.target.value as 'user' | 'group')}>
                <option value="user">User</option>
                <option value="group">Group</option>
              </select>
              <select value={newPermGranteeId} onChange={(e) => setNewPermGranteeId(e.target.value)}>
                <option value="">Select {newPermGranteeType}...</option>
                {newPermGranteeType === 'user'
                  ? users.map((u) => (
                      <option key={u.id} value={u.id}>{u.email}</option>
                    ))
                  : groups.map((g) => (
                      <option key={g.id} value={g.id}>{g.name}</option>
                    ))
                }
              </select>
              <select value={newPermLevel} onChange={(e) => setNewPermLevel(Number(e.target.value))}>
                <option value={1}>Viewer</option>
                <option value={2}>Editor</option>
                <option value={3}>Admin</option>
              </select>
              <Button variant="primary" size="sm" onClick={handleAddPermission}>
                Apply
              </Button>
            </div>
          </Card>
        </div>
      )}
    </div>
  )
}

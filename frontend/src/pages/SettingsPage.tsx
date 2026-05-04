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
  Permission,
  User,
  Group,
  GroupMember,
} from '../api/admin'
import './SettingsPage.css'

export function SettingsPage() {
  const navigate = useNavigate()
  const currentUser = useAuthStore((s) => s.user)
  const logout = useAuthStore((s) => s.logout)

  const [users, setUsers] = useState<User[]>([])
  const [groups, setGroups] = useState<Group[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<'users' | 'groups' | 'permissions'>('users')

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

  // Redirect non-admins
  if (currentUser?.role !== 'admin') {
    return (
      <div className="settings-page">
        <div className="settings-container">
          <h1>Settings</h1>
          <p>You do not have permission to access this page.</p>
          <button onClick={() => navigate('/')}>Back to Space</button>
        </div>
      </div>
    )
  }

  const loadData = async () => {
    setLoading(true)
    setError(null)
    try {
      const [usersData, groupsData] = await Promise.all([listUsers(), listGroups()])
      setUsers(usersData)
      setGroups(groupsData)
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

  const levelLabel = (level: number) => {
    switch (level) {
      case 1: return 'Viewer'
      case 2: return 'Editor'
      case 3: return 'Admin'
      default: return 'None'
    }
  }

  return (
    <div className="settings-page">
      <header className="settings-header">
        <h1>Settings</h1>
        <nav>
          <button onClick={() => navigate('/')} className="btn-back">
            &larr; Back to Space
          </button>
          <button onClick={logout} className="btn-logout">
            Logout
          </button>
        </nav>
      </header>

      <div className="settings-tabs">
        <button
          className={`tab ${activeTab === 'users' ? 'active' : ''}`}
          onClick={() => setActiveTab('users')}
        >
          Users ({users.length})
        </button>
        <button
          className={`tab ${activeTab === 'groups' ? 'active' : ''}`}
          onClick={() => setActiveTab('groups')}
        >
          Groups ({groups.length})
        </button>
        <button
          className={`tab ${activeTab === 'permissions' ? 'active' : ''}`}
          onClick={() => setActiveTab('permissions')}
        >
          Permissions
        </button>
      </div>

      {error && <div className="settings-error">{error}</div>}

      {loading ? (
        <p className="settings-loading">Loading...</p>
      ) : activeTab === 'users' ? (
        <div className="settings-section">
          <table className="users-table">
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
                    <select
                      value={u.role}
                      onChange={(e) => handleRoleChange(u.id, e.target.value)}
                      disabled={u.id === currentUser?.id}
                    >
                      <option value="viewer">Viewer</option>
                      <option value="editor">Editor</option>
                      <option value="admin">Admin</option>
                    </select>
                  </td>
                  <td>
                    <button
                      className="btn-danger"
                      onClick={() => handleDeleteUser(u.id)}
                      disabled={u.id === currentUser?.id}
                    >
                      Delete
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : activeTab === 'groups' ? (
        <div className="settings-section">
          <div className="create-group">
            <input
              type="text"
              value={newGroupName}
              onChange={(e) => setNewGroupName(e.target.value)}
              placeholder="New group name..."
              onKeyDown={(e) => e.key === 'Enter' && handleCreateGroup()}
            />
            <button className="btn-primary" onClick={handleCreateGroup}>
              Create Group
            </button>
          </div>

          <div className="groups-list">
            {groups.map((g) => (
              <div
                key={g.id}
                className={`group-item ${selectedGroup === g.id ? 'selected' : ''}`}
              >
                <div
                  className="group-header"
                  onClick={() => {
                    setSelectedGroup(g.id)
                    loadGroupMembers(g.id)
                  }}
                >
                  <span className="group-name">{g.name}</span>
                  <button
                    className="btn-danger btn-sm"
                    onClick={(e) => {
                      e.stopPropagation()
                      handleDeleteGroup(g.id)
                    }}
                  >
                    Delete
                  </button>
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
                      <button className="btn-primary btn-sm" onClick={handleAddMember}>
                        Add
                      </button>
                    </div>
                    <ul>
                      {groupMembers.map((m) => (
                        <li key={m.id} className="member-item">
                          <span>
                            {m.email} ({m.name || 'no name'})
                          </span>
                          <button
                            className="btn-danger btn-sm"
                            onClick={() => handleRemoveMember(m.id)}
                          >
                            Remove
                          </button>
                        </li>
                      ))}
                      {groupMembers.length === 0 && (
                        <li className="empty">No members yet</li>
                      )}
                    </ul>
                  </div>
                )}
              </div>
            ))}
            {groups.length === 0 && <p className="empty">No groups created yet</p>}
          </div>
        </div>
      ) : (
        <div className="settings-section">
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
              <input
                type="text"
                value={permTargetId}
                onChange={(e) => setPermTargetId(e.target.value)}
                placeholder="UUID or slug..."
              />
            </label>
            <button className="btn-primary" onClick={loadPermissions}>
              Load Permissions
            </button>
          </div>

          <div className="perm-list">
            <h3>Current Permissions</h3>
            {permissions.length === 0 ? (
              <p className="empty">No explicit permissions set. Default permission from space applies.</p>
            ) : (
              <table className="perms-table">
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
                      <td>{levelLabel(p.level)}</td>
                      <td>
                        <button
                          className="btn-danger btn-sm"
                          onClick={() => handleRemovePermission(p.granteeType, p.granteeId)}
                        >
                          Remove
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>

          <div className="perm-add-form">
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
              <button className="btn-primary btn-sm" onClick={handleAddPermission}>
                Apply
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

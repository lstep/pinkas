import { useEffect, useState, useMemo } from 'react'
import { Modal, Button, Badge, Card, Input } from '../ui'
import {
  listPermissionsForTarget,
  setPermission,
  removePermission,
  listUsers,
  listGroups,
  Permission,
  User,
  Group,
} from '../../api/admin'
import './ShareModal.css'

export interface ShareModalProps {
  isOpen: boolean
  onClose: () => void
  targetType: 'page' | 'directory' | 'space'
  targetId: string
  targetTitle: string
}

export function ShareModal({ isOpen, onClose, targetType, targetId, targetTitle }: ShareModalProps) {
  const [permissions, setPermissions] = useState<Permission[]>([])
  const [users, setUsers] = useState<User[]>([])
  const [groups, setGroups] = useState<Group[]>([])
  const [loading, setLoading] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedGrantee, setSelectedGrantee] = useState<{ type: 'user' | 'group'; id: string; name: string } | null>(null)
  const [selectedLevel, setSelectedLevel] = useState(1)
  const [isAdding, setIsAdding] = useState(false)

  // Load data when modal opens
  useEffect(() => {
    if (!isOpen) return
    loadData()
  }, [isOpen, targetType, targetId])

  const loadData = async () => {
    setLoading(true)
    try {
      const [permsData, usersData, groupsData] = await Promise.all([
        listPermissionsForTarget(targetType, targetId),
        listUsers(),
        listGroups(),
      ])
      setPermissions(permsData)
      setUsers(usersData)
      setGroups(groupsData)
    } catch (err: any) {
      console.error('Failed to load share data:', err)
    } finally {
      setLoading(false)
    }
  }

  // Filter users and groups by search query
  const filteredUsers = useMemo(() => {
    if (!searchQuery.trim()) return users.slice(0, 5)
    const query = searchQuery.toLowerCase()
    return users.filter(
      (u) =>
        u.email.toLowerCase().includes(query) ||
        u.name?.toLowerCase().includes(query)
    )
  }, [users, searchQuery])

  const filteredGroups = useMemo(() => {
    if (!searchQuery.trim()) return groups.slice(0, 3)
    const query = searchQuery.toLowerCase()
    return groups.filter((g) => g.name.toLowerCase().includes(query))
  }, [groups, searchQuery])

  // Get grantee name from ID
  const getGranteeName = (granteeType: string, granteeId: string) => {
    if (granteeType === 'user') {
      const user = users.find((u) => u.id === granteeId)
      return user ? `${user.name || user.email}` : `User (${granteeId.slice(0, 8)}...)`
    } else {
      const group = groups.find((g) => g.id === granteeId)
      return group ? group.name : `Group (${granteeId.slice(0, 8)}...)`
    }
  }

  const getGranteeSubtitle = (granteeType: string, granteeId: string) => {
    if (granteeType === 'user') {
      const user = users.find((u) => u.id === granteeId)
      return user?.email || ''
    }
    return ''
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

  const handleGrant = async () => {
    if (!selectedGrantee) return
    setIsAdding(true)
    try {
      await setPermission(targetType, targetId, selectedGrantee.type, selectedGrantee.id, selectedLevel)
      setSelectedGrantee(null)
      setSearchQuery('')
      setSelectedLevel(1)
      await loadData()
    } catch (err: any) {
      alert(err.message || 'Failed to grant permission')
    } finally {
      setIsAdding(false)
    }
  }

  const handleRemove = async (granteeType: string, granteeId: string) => {
    try {
      await removePermission(targetType, targetId, granteeType, granteeId)
      await loadData()
    } catch (err: any) {
      alert(err.message || 'Failed to remove permission')
    }
  }

  const handleSelectGrantee = (type: 'user' | 'group', id: string, name: string) => {
    setSelectedGrantee({ type, id, name })
    setSearchQuery('')
  }

  const handleClearSelection = () => {
    setSelectedGrantee(null)
    setSearchQuery('')
  }

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={`Share — ${targetTitle}`}
      footer={
        <Button variant="primary" onClick={onClose}>
          Done
        </Button>
      }
    >
      <div className="share-modal-content">
        {/* Add people section */}
        <Card className="share-add-section" padding="md">
          <h4 className="share-section-title">Add people or groups</h4>
          
          {selectedGrantee ? (
            <div className="share-selected-grantee">
              <div className="share-grantee-chip">
                <span className="share-grantee-name">{selectedGrantee.name}</span>
                <button className="share-grantee-remove" onClick={handleClearSelection}>
                  ×
                </button>
              </div>
              <div className="share-grant-actions">
                <select
                  value={selectedLevel}
                  onChange={(e) => setSelectedLevel(Number(e.target.value))}
                  className="share-level-select"
                >
                  <option value={1}>Viewer</option>
                  <option value={2}>Editor</option>
                  <option value={3}>Admin</option>
                </select>
                <Button
                  variant="primary"
                  size="sm"
                  onClick={handleGrant}
                  loading={isAdding}
                >
                  Grant
                </Button>
              </div>
            </div>
          ) : (
            <div className="share-search-container">
              <Input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="Search users or groups..."
                className="share-search-input"
              />
              
              {searchQuery && (
                <div className="share-search-results">
                  {filteredUsers.length > 0 && (
                    <div className="share-result-section">
                      <span className="share-result-label">Users</span>
                      {filteredUsers.map((user) => (
                        <button
                          key={user.id}
                          className="share-result-item"
                          onClick={() => handleSelectGrantee('user', user.id, user.name || user.email)}
                        >
                          <span className="share-result-name">{user.name || user.email}</span>
                          {user.name && <span className="share-result-email">{user.email}</span>}
                        </button>
                      ))}
                    </div>
                  )}
                  
                  {filteredGroups.length > 0 && (
                    <div className="share-result-section">
                      <span className="share-result-label">Groups</span>
                      {filteredGroups.map((group) => (
                        <button
                          key={group.id}
                          className="share-result-item"
                          onClick={() => handleSelectGrantee('group', group.id, group.name)}
                        >
                          <span className="share-result-name">{group.name}</span>
                          <span className="share-result-type">Group</span>
                        </button>
                      ))}
                    </div>
                  )}
                  
                  {filteredUsers.length === 0 && filteredGroups.length === 0 && (
                    <div className="share-no-results">No users or groups found</div>
                  )}
                </div>
              )}
            </div>
          )}
        </Card>

        {/* Current permissions list */}
        <div className="share-permissions-section">
          <h4 className="share-section-title">
            Current permissions
            {loading && <span className="share-loading-indicator">Loading...</span>}
          </h4>
          
          {permissions.length === 0 ? (
            <p className="share-empty">No explicit permissions set. Default permissions apply.</p>
          ) : (
            <div className="share-permissions-list">
              {permissions.map((perm, index) => (
                <div key={index} className="share-permission-row">
                  <div className="share-permission-info">
                    <span className="share-permission-name">
                      {getGranteeName(perm.granteeType, perm.granteeId)}
                    </span>
                    <span className="share-permission-subtitle">
                      {getGranteeSubtitle(perm.granteeType, perm.granteeId)}
                    </span>
                  </div>
                  <div className="share-permission-actions">
                    <Badge variant={levelBadgeVariant(perm.level)}>
                      {levelLabel(perm.level)}
                    </Badge>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleRemove(perm.granteeType, perm.granteeId)}
                    >
                      Remove
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </Modal>
  )
}
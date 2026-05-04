import React, { useEffect, useState, useCallback } from 'react'
import { listSnapshots, restoreSnapshot, timeAgo, formatTimestamp, Snapshot } from '../../api/snapshots'
import { Button, Modal } from '../ui'
import { DiffView } from '../DiffView'
import './HistoryPanel.css'

interface HistoryPanelProps {
  pageId: string
  isOpen: boolean
  onClose: () => void
  onRestore?: () => void
}

export const HistoryPanel: React.FC<HistoryPanelProps> = ({ pageId, isOpen, onClose, onRestore }) => {
  const [snapshots, setSnapshots] = useState<Snapshot[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [selectedSnapshot, setSelectedSnapshot] = useState<Snapshot | null>(null)
  const [showDiff, setShowDiff] = useState(false)
  const [showRestoreConfirm, setShowRestoreConfirm] = useState<string | null>(null)
  const [restoring, setRestoring] = useState(false)

  const fetchSnapshots = useCallback(async () => {
    if (!pageId || !isOpen) return
    setLoading(true)
    setError(null)
    try {
      const list = await listSnapshots(pageId)
      setSnapshots(list)
    } catch (err: any) {
      setError(err.message || 'Failed to load history')
    } finally {
      setLoading(false)
    }
  }, [pageId, isOpen])

  useEffect(() => {
    fetchSnapshots()
  }, [fetchSnapshots])

  const handleViewDiff = useCallback(async (snapshot: Snapshot) => {
    setSelectedSnapshot(snapshot)
    setShowDiff(true)
  }, [])

  const handleRestore = useCallback(async (snapshotId: string) => {
    setRestoring(true)
    try {
      await restoreSnapshot(pageId, snapshotId)
      setShowRestoreConfirm(null)
      onRestore?.()
      // Refresh snapshot list and close panel after a brief delay
      setTimeout(() => {
        onClose()
      }, 600)
    } catch (err: any) {
      setError(err.message || 'Failed to restore snapshot')
    } finally {
      setRestoring(false)
    }
  }, [pageId, onRestore, onClose])

  if (!isOpen) return null

  const hasPreRestore = snapshots.some(s => s.label === 'pre-restore')

  return (
    <div className="history-panel">
      <div className="history-panel-header">
        <h3>Page History</h3>
        <button className="history-panel-close" onClick={onClose} aria-label="Close history panel" type="button">
          ×
        </button>
      </div>

      <div className="history-panel-list">
        {loading && (
          <div className="history-loading">Loading history...</div>
        )}

        {error && (
          <div className="history-error">{error}</div>
        )}

        {!loading && !error && snapshots.length === 0 && (
          <div className="history-panel-empty">No version history yet</div>
        )}

        {!loading && snapshots.map((snapshot) => (
          <div
            key={snapshot.id}
            className={`history-snapshot-item ${selectedSnapshot?.id === snapshot.id ? 'selected' : ''}`}
          >
            <div className="history-snapshot-meta">
              <span className="history-snapshot-time" title={formatTimestamp(snapshot.createdAt)}>
                {timeAgo(snapshot.createdAt)}
              </span>
              <div className="history-snapshot-actions">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleViewDiff(snapshot)}
                >
                  Diff
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={snapshot.label === 'pre-restore' || restoring}
                  onClick={() => setShowRestoreConfirm(snapshot.id)}
                >
                  Restore
                </Button>
              </div>
            </div>
            {snapshot.label && (
              <span className="history-snapshot-label">{snapshot.label}</span>
            )}
          </div>
        ))}
      </div>

      {/* Diff Modal */}
      {selectedSnapshot && (
        <DiffView
          snapshot={selectedSnapshot}
          isOpen={showDiff}
          onClose={() => {
            setShowDiff(false)
            setSelectedSnapshot(null)
          }}
        />
      )}

      {/* Restore Confirmation */}
      <Modal
        isOpen={showRestoreConfirm !== null}
        onClose={() => setShowRestoreConfirm(null)}
        title="Restore Version"
        footer={
          <>
            <Button variant="secondary" onClick={() => setShowRestoreConfirm(null)}>
              Cancel
            </Button>
            <Button
              variant="primary"
              loading={restoring}
              onClick={() => showRestoreConfirm && handleRestore(showRestoreConfirm)}
            >
              Restore
            </Button>
          </>
        }
      >
        <p style={{ color: 'var(--color-text-secondary)', lineHeight: 1.6 }}>
          This will create a backup of the current version and restore the page
          to the selected snapshot. Other collaborators will see the restored
          content immediately.
        </p>
        {!hasPreRestore && (
          <p style={{ color: 'var(--color-text-muted)', fontSize: 'var(--text-sm)', marginTop: 'var(--sp-2)' }}>
            A pre-restore backup snapshot will be created automatically.
          </p>
        )}
      </Modal>
    </div>
  )
}

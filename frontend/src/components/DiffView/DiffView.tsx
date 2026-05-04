import React, { useEffect, useState } from 'react'
import { Modal } from '../ui'
import { getSnapshot, Snapshot } from '../../api/snapshots'
import './DiffView.css'

interface DiffViewProps {
  snapshot: Snapshot
  isOpen: boolean
  onClose: () => void
}

export const DiffView: React.FC<DiffViewProps> = ({ snapshot, isOpen, onClose }) => {
  const [snapshotMarkdown, setSnapshotMarkdown] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!isOpen || !snapshot) return

    async function load() {
      setLoading(true)
      setError(null)
      try {
        const detail = await getSnapshot(snapshot.pageId, snapshot.id)
        setSnapshotMarkdown(detail.markdown ?? '')
      } catch (err: any) {
        setError(err.message || 'Failed to load snapshot content')
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [isOpen, snapshot])

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={`Snapshot: ${new Date(snapshot.createdAt).toLocaleString()}`}
      footer={
        <div className="diff-legend" style={{ display: 'flex', gap: 'var(--sp-3)', fontSize: 'var(--text-xs)', color: 'var(--color-text-muted)' }}>
          <span style={{ color: '#2d6a4f' }}>● Added</span>
          <span style={{ color: '#a83226' }}>● Removed</span>
        </div>
      }
    >
      <div className="diff-view-content">
        {loading && <div className="diff-loading">Loading snapshot content...</div>}
        {error && <div className="diff-error">{error}</div>}
        {!loading && !error && snapshotMarkdown === '' && (
          <div className="diff-empty">Empty page content</div>
        )}
        {!loading && !error && snapshotMarkdown && snapshotMarkdown.split('\n').map((line, i) => (
          <div key={i} className="diff-line diff-line-unchanged">
            <span className="diff-line-number">{i + 1}</span>
            {line || ' '}
          </div>
        ))}
      </div>
    </Modal>
  )
}

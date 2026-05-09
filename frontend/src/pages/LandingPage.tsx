import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { listSpaces, listRecentPages, listMyPages, PageSummary, Space } from '../api/pages'
import { useAuthStore } from '../store/auth'
import './LandingPage.css'

type Tab = 'recent' | 'mine'

const SPACE_EMOJIS = ['📝', '📄', '📘', '📗', '📙', '📕', '📓', '📔', '📒', '📑', '🗂️', '📁', '💡', '🎯', '🧠', '🚀', '⚡', '🎨', '🔬', '📊']

function getSpaceEmoji(icon: string | null | undefined, name: string): string {
  if (icon) return icon
  const index = name.split('').reduce((acc, c) => acc + c.charCodeAt(0), 0)
  return SPACE_EMOJIS[index % SPACE_EMOJIS.length]
}

function formatDate(ts: number): string {
  const d = new Date(ts * 1000)
  const now = new Date()
  const diff = now.getTime() - d.getTime()
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) return 'Just now'
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  if (days < 7) return `${days}d ago`
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
}

export function LandingPage() {
  const navigate = useNavigate()
  const user = useAuthStore((s) => s.user)

  const [spaces, setSpaces] = useState<Space[]>([])
  const [recentPages, setRecentPages] = useState<PageSummary[]>([])
  const [myPages, setMyPages] = useState<PageSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState<Tab>('recent')

  const spacesLoading = spaces.length === 0 && loading

  useEffect(() => {
    async function load() {
      try {
        const [spacesData, recentData, myData] = await Promise.all([
          listSpaces(),
          listRecentPages(12),
          listMyPages(12),
        ])
        setSpaces(spacesData)
        setRecentPages(recentData)
        setMyPages(myData)
      } catch {
        // silent
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [])

  const handleSpaceClick = (slug: string) => {
    navigate(`/s/${slug}`)
  }

  const handlePageClick = (page: PageSummary) => {
    navigate(`/s/${page.spaceId}/${page.slug}`)
  }

  return (
    <div className="landing-page">
      <div className="landing-header">
        <h1>
          {user?.name ? `${user.name}'s` : 'My'} Wiki
        </h1>
        <p className="landing-subtitle">Select a space or browse recent pages</p>
      </div>

      {spacesLoading ? (
        <div className="landing-loading">
          {[1, 2, 3].map((i) => (
            <div key={i} className="skeleton space-card-skeleton" />
          ))}
        </div>
      ) : (
        <div className="space-grid">
          {spaces.map((space) => (
            <button
              key={space.id}
              className="space-card"
              onClick={() => handleSpaceClick(space.slug)}
            >
              <span className="space-card-icon">{getSpaceEmoji(space.icon, space.name)}</span>
              <span className="space-card-name">{space.name}</span>
            </button>
          ))}
        </div>
      )}

      <div className="pages-section">
        <div className="pages-tabs">
          <button
            className={`pages-tab ${activeTab === 'recent' ? 'active' : ''}`}
            onClick={() => setActiveTab('recent')}
          >
            Recently Updated
          </button>
          <button
            className={`pages-tab ${activeTab === 'mine' ? 'active' : ''}`}
            onClick={() => setActiveTab('mine')}
          >
            My Pages
          </button>
        </div>

        {activeTab === 'recent' && (
          <div className="pages-table-wrap">
            <table className="pages-table">
              <thead>
                <tr>
                  <th>Page</th>
                  <th>Space</th>
                  <th>Updated</th>
                </tr>
              </thead>
              <tbody>
                {recentPages.length === 0 && !loading && (
                  <tr><td colSpan={3} className="empty">No pages yet</td></tr>
                )}
                {recentPages.map((page) => (
                  <tr
                    key={page.id}
                    className="pages-row"
                    onClick={() => handlePageClick(page)}
                  >
                    <td>
                      <span className="page-icon">{page.icon || '📄'}</span>
                      {page.title}
                    </td>
                    <td className="pages-space-cell">{page.spaceName}</td>
                    <td className="pages-date-cell">{formatDate(page.updatedAt)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {activeTab === 'mine' && (
          <div className="pages-table-wrap">
            <table className="pages-table">
              <thead>
                <tr>
                  <th>Page</th>
                  <th>Space</th>
                  <th>Updated</th>
                </tr>
              </thead>
              <tbody>
                {myPages.length === 0 && !loading && (
                  <tr><td colSpan={3} className="empty">You haven't created any pages yet</td></tr>
                )}
                {myPages.map((page) => (
                  <tr
                    key={page.id}
                    className="pages-row"
                    onClick={() => handlePageClick(page)}
                  >
                    <td>
                      <span className="page-icon">{page.icon || '📄'}</span>
                      {page.title}
                    </td>
                    <td className="pages-space-cell">{page.spaceName}</td>
                    <td className="pages-date-cell">{formatDate(page.updatedAt)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}

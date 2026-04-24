import React, { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useTreeStore } from '../../store/tree'
import { getBreadcrumb } from '../../api/pages'
import './Breadcrumb.css'

interface BreadcrumbProps {
  pageId: string | null
  spaceSlug: string
}

interface BreadcrumbItem {
  id: string
  title: string
  slug: string
  type: 'page' | 'directory'
}

export const Breadcrumb: React.FC<BreadcrumbProps> = ({ pageId, spaceSlug }) => {
  const nodes = useTreeStore((s) => s.nodes)
  const [breadcrumb, setBreadcrumb] = useState<BreadcrumbItem[]>([])
  const [loading, setLoading] = useState(false)

  // Build breadcrumb from tree data and/or API
  useEffect(() => {
    if (!pageId) {
      setBreadcrumb([])
      return
    }

    async function buildBreadcrumb(currentPageId: string) {
      // First try to build from local tree state
      const page = nodes[currentPageId]
      if (!page || page.type !== 'page') {
        setBreadcrumb([])
        return
      }

      const chain: BreadcrumbItem[] = [
        { id: page.id, title: page.title, slug: page.slug, type: 'page' }
      ]

      // Walk up directory ancestors from tree state
      let currentDirId: string | null = page.directory_id
      const directoryChain: BreadcrumbItem[] = []

      while (currentDirId && nodes[currentDirId]) {
        const dir = nodes[currentDirId]
        if (dir.type === 'directory') {
          directoryChain.unshift({
            id: dir.id,
            title: dir.name || dir.title,
            slug: dir.slug,
            type: 'directory'
          })
          currentDirId = dir.parent_id
        } else {
          break
        }
      }

      // If we couldn't complete the chain from local state, fetch from API
      if (currentDirId) {
        setLoading(true)
        try {
          // Fetch page breadcrumb from API (includes directory ancestors)
          const apiBreadcrumb = await getBreadcrumb(currentPageId)
          const apiChain: BreadcrumbItem[] = apiBreadcrumb.map(item => ({
            id: item.id,
            title: item.title,
            slug: item.slug,
            type: 'page' // API returns pages, but may include directory info
          }))
          setBreadcrumb(apiChain)
          return
        } catch (err) {
          console.error('Failed to fetch breadcrumb:', err)
        } finally {
          setLoading(false)
        }
      }

      // Combine directory chain with page
      setBreadcrumb([...directoryChain, ...chain])
    }

    buildBreadcrumb(pageId)
  }, [pageId, nodes])

  if (!pageId) {
    return (
      <nav className="breadcrumb">
        <Link to={`/s/${spaceSlug}`}>Home</Link>
      </nav>
    )
  }

  return (
    <nav className="breadcrumb">
      <Link to={`/s/${spaceSlug}`}>Home</Link>
      {breadcrumb.map((item, idx) => (
        <React.Fragment key={item.id}>
          <span className="breadcrumb-sep">/</span>
          {idx === breadcrumb.length - 1 ? (
            <span className="breadcrumb-current">{item.title}</span>
          ) : (
            <Link to={`/s/${spaceSlug}/${item.slug}`}>{item.title}</Link>
          )}
        </React.Fragment>
      ))}
      {loading && <span className="breadcrumb-loading">...</span>}
    </nav>
  )
}

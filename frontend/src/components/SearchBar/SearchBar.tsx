import { useState, useEffect, useRef, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { searchPages, SearchResult } from '../../api/pages'
import { listSpaces } from '../../api/pages'
import './SearchBar.css'

interface SpaceCache {
  [id: string]: { slug: string; name: string }
}

export function SearchBar() {
  const navigate = useNavigate()
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<SearchResult[]>([])
  const [loading, setLoading] = useState(false)
  const [isOpen, setIsOpen] = useState(false)
  const [spaces, setSpaces] = useState<SpaceCache>({})
  const inputRef = useRef<HTMLInputElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const debounceRef = useRef<NodeJS.Timeout | null>(null)

  // Load spaces for slug lookup
  useEffect(() => {
    listSpaces().then((spaceList) => {
      const cache: SpaceCache = {}
      spaceList.forEach((s) => {
        cache[s.id] = { slug: s.slug, name: s.name }
      })
      setSpaces(cache)
    })
  }, [])

  // Debounced search
  useEffect(() => {
    if (debounceRef.current) {
      clearTimeout(debounceRef.current)
    }

    if (!query.trim()) {
      setResults([])
      setLoading(false)
      return
    }

    setLoading(true)
    debounceRef.current = setTimeout(async () => {
      try {
        const searchResults = await searchPages(query.trim(), 8)
        setResults(searchResults)
      } catch (err) {
        console.error('Search failed:', err)
        setResults([])
      } finally {
        setLoading(false)
      }
    }, 300)

    return () => {
      if (debounceRef.current) {
        clearTimeout(debounceRef.current)
      }
    }
  }, [query])

  // Close on click outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside)
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [isOpen])

  // Close on Escape key
  useEffect(() => {
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        setIsOpen(false)
        inputRef.current?.blur()
      }
    }

    if (isOpen) {
      document.addEventListener('keydown', handleKeyDown)
    }

    return () => {
      document.removeEventListener('keydown', handleKeyDown)
    }
  }, [isOpen])

  const handleInputFocus = useCallback(() => {
    setIsOpen(true)
  }, [])

  const handleResultClick = useCallback(
    (result: SearchResult) => {
      const spaceSlug = spaces[result.spaceId]?.slug || result.spaceId
      setIsOpen(false)
      setQuery('')
      setResults([])
      navigate(`/s/${spaceSlug}/${result.slug}`)
    },
    [navigate, spaces]
  )

  // Highlight matching text in excerpt
  const renderHighlightedText = (text: string, searchQuery: string) => {
    if (!searchQuery.trim()) return text

    // The backend returns matches wrapped in ###...###
    // Replace with highlighted spans
    const parts = text.split(/(###[^#]+###)/g)

    return parts.map((part, index) => {
      if (part.startsWith('###') && part.endsWith('###')) {
        const matchText = part.slice(3, -3)
        return (
          <mark key={index} className="search-highlight">
            {matchText}
          </mark>
        )
      }
      return part
    })
  }

  // Get excerpt (first 200 chars, trimmed)
  const getExcerpt = (markdown: string): string => {
    return markdown.slice(0, 200).trim()
  }

  const hasResults = results.length > 0
  const showEmpty = query.trim() && !loading && !hasResults
  const showDropdown = isOpen && (hasResults || showEmpty || loading)

  return (
    <div className="search-bar" ref={containerRef}>
      <div className="search-input-wrapper">
        <svg
          className="search-icon"
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <circle cx="11" cy="11" r="8" />
          <path d="m21 21-4.3-4.3" />
        </svg>
        <input
          ref={inputRef}
          type="search"
          className="search-input"
          placeholder="Search pages..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onFocus={handleInputFocus}
          aria-label="Search pages"
        />
        {loading && (
          <div className="search-spinner">
            <div className="spinner-ring" />
          </div>
        )}
      </div>

      {showDropdown && (
        <div className="search-dropdown">
          {loading && results.length === 0 && (
            <div className="search-loading">Searching...</div>
          )}

          {showEmpty && <div className="search-empty">No results found</div>}

          {hasResults && (
            <ul className="search-results">
              {results.map((result) => (
                <li
                  key={result.id}
                  className="search-result-item"
                  onClick={() => handleResultClick(result)}
                >
                  <div className="search-result-header">
                    <span className="search-result-icon">{result.icon || '📄'}</span>
                    <span className="search-result-title">{result.title}</span>
                  </div>
                  <div className="search-result-excerpt">
                    {renderHighlightedText(getExcerpt(result.markdown), query)}
                  </div>
                  <div className="search-result-meta">
                    {spaces[result.spaceId]?.name || 'Unknown space'}
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  )
}

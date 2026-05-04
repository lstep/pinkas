import { useEffect } from 'react'
import { Navigate } from 'react-router-dom'
import { useAuthStore } from '../store/auth'

export function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const setUser = useAuthStore((s) => s.setUser)

  // Refresh user data from server on mount to pick up role changes
  useEffect(() => {
    if (!isAuthenticated) return
    const token = useAuthStore.getState().accessToken
    if (!token) return

    fetch('/api/auth/me', {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    })
      .then((res) => {
        if (!res.ok) throw new Error('unauthorized')
        return res.json()
      })
      .then((data) => {
        if (data.user) setUser(data.user)
      })
      .catch(() => {
        // silently fail — user will see stale data until next login
      })
  }, [isAuthenticated, setUser])

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  return <>{children}</>
}

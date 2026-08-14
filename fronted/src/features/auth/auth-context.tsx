import { useQueryClient } from "@tanstack/react-query"
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type PropsWithChildren,
} from "react"

import { api, UNAUTHORIZED_EVENT } from "@/lib/api"
import type { AdminUser } from "@/lib/types"

interface AuthContextValue {
  user: AdminUser | null
  loading: boolean
  login: (username: string, password: string) => Promise<AdminUser>
  bootstrap: (body: { username: string; password: string; displayName?: string; avatar?: string }) => Promise<AdminUser>
  logout: () => Promise<void>
  setUser: (user: AdminUser | null) => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: PropsWithChildren) {
  const queryClient = useQueryClient()
  const [user, setUserState] = useState<AdminUser | null>(null)
  const [loading, setLoading] = useState(true)

  const clearSession = useCallback(() => {
    setUserState(null)
    queryClient.clear()
  }, [queryClient])

  const setUser = useCallback((next: AdminUser | null) => {
    setUserState(next)
  }, [])

  useEffect(() => {
    let active = true
    api
      .get<{ user: AdminUser }>("/auth/me")
      .then(({ user: current }) => {
        if (active) setUserState(current)
      })
      .catch(() => {
        if (active) setUserState(null)
      })
      .finally(() => {
        if (active) setLoading(false)
      })

    window.addEventListener(UNAUTHORIZED_EVENT, clearSession)
    return () => {
      active = false
      window.removeEventListener(UNAUTHORIZED_EVENT, clearSession)
    }
  }, [clearSession])

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      loading,
      setUser,
      login: async (username, password) => {
        const result = await api.post<{ user: AdminUser }>("/auth/login", { username, password })
        setUserState(result.user)
        return result.user
      },
      bootstrap: async (body) => {
        const result = await api.post<{ user: AdminUser }>("/auth/bootstrap", body)
        setUserState(result.user)
        return result.user
      },
      logout: async () => {
        try {
          await api.post<{ success: boolean }>("/auth/logout")
        } finally {
          clearSession()
        }
      },
    }),
    [clearSession, loading, setUser, user],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const value = useContext(AuthContext)
  if (!value) throw new Error("useAuth must be used within AuthProvider")
  return value
}

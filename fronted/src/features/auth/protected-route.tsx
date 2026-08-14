import { LoaderCircle } from "lucide-react"
import { Navigate, Outlet, useLocation } from "react-router-dom"

import { useAuth } from "@/features/auth/auth-context"

export function ProtectedRoute() {
  const { loading, user } = useAuth()
  const location = useLocation()

  if (loading) {
    return (
      <div className="grid min-h-svh place-items-center bg-background">
        <LoaderCircle className="size-6 animate-spin text-primary" aria-label="正在验证登录状态" />
      </div>
    )
  }
  if (!user) {
    return <Navigate replace to="/auth/login" state={{ from: location.pathname }} />
  }
  return <Outlet />
}

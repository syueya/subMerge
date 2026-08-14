import { Suspense, lazy } from "react"
import { Navigate, RouterProvider, createBrowserRouter } from "react-router-dom"

import { AppShell } from "@/components/app-shell"
import { ProtectedRoute } from "@/features/auth/protected-route"
import { Spinner } from "@/components/ui/spinner"

const page = <T extends Record<string, unknown>>(loader: () => Promise<T>, name: keyof T) =>
  lazy(async () => ({ default: (await loader())[name] as React.ComponentType }))

const LoginPage = page(() => import("@/pages/login-page"), "LoginPage")
const DashboardPage = page(() => import("@/pages/dashboard-page"), "DashboardPage")
const SourcesPage = page(() => import("@/pages/sources-page"), "SourcesPage")
const GroupsPage = page(() => import("@/pages/groups-page"), "GroupsPage")
const RulesPage = page(() => import("@/pages/rules-page"), "RulesPage")
const ReleasesPage = page(() => import("@/pages/releases-page"), "ReleasesPage")
const TokensPage = page(() => import("@/pages/tokens-page"), "TokensPage")
const GeoPage = page(() => import("@/pages/geo-page"), "GeoPage")
const NetCheckPage = page(() => import("@/pages/net-check-page"), "NetCheckPage")
const AccountPage = page(() => import("@/pages/account-page"), "AccountPage")
const SettingsPage = page(() => import("@/pages/settings-page"), "SettingsPage")
const LogsPage = page(() => import("@/pages/logs-page"), "LogsPage")
const ErrorPage = page(() => import("@/pages/error-page"), "ErrorPage")

function PendingPage() {
  return <div className="grid min-h-56 place-items-center"><Spinner className="size-5" /></div>
}

function LazyPage({ children }: { children: React.ReactNode }) {
  return <Suspense fallback={<PendingPage />}>{children}</Suspense>
}

const router = createBrowserRouter([
  { path: "/", element: <Navigate replace to="/main/dashboard" /> },
  { path: "/auth", element: <Navigate replace to="/auth/login" /> },
  { path: "/auth/login", element: <LazyPage><LoginPage /></LazyPage> },
  { path: "/auth/system-init-config", element: <Navigate replace to="/auth/login" /> },
  {
    element: <ProtectedRoute />,
    children: [
      {
        path: "/main",
        element: <AppShell />,
        children: [
          { index: true, element: <Navigate replace to="dashboard" /> },
          { path: "dashboard", element: <LazyPage><DashboardPage /></LazyPage> },
          { path: "sources", element: <LazyPage><SourcesPage /></LazyPage> },
          { path: "groups", element: <LazyPage><GroupsPage /></LazyPage> },
          { path: "rules", element: <LazyPage><RulesPage /></LazyPage> },
          { path: "releases", element: <LazyPage><ReleasesPage /></LazyPage> },
          { path: "tokens", element: <LazyPage><TokensPage /></LazyPage> },
          { path: "geo", element: <LazyPage><GeoPage /></LazyPage> },
          { path: "net-check", element: <LazyPage><NetCheckPage /></LazyPage> },
          { path: "account-setting", element: <LazyPage><AccountPage /></LazyPage> },
          { path: "settings", element: <Navigate replace to="setting-settings" /> },
          { path: "settings/setting-settings", element: <LazyPage><SettingsPage /></LazyPage> },
          { path: "settings/logs", element: <LazyPage><LogsPage /></LazyPage> },
          { path: "logs", element: <LazyPage><LogsPage /></LazyPage> },
        ],
      },
    ],
  },
  { path: "/error", element: <LazyPage><ErrorPage /></LazyPage> },
  { path: "*", element: <Navigate replace to="/error" /> },
])

export default function App() {
  return <RouterProvider router={router} />
}

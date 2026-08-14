import {
  Boxes,
  ChevronLeft,
  ChevronRight,
  ChevronUp,
  CloudDownload,
  Database,
  FileClock,
  Gauge,
  KeyRound,
  ListChecks,
  LogOut,
  Network,
  Rocket,
  Settings,
  UserRound,
} from "lucide-react"
import { NavLink, Outlet, useLocation } from "react-router-dom"

import submergeLogo from "@/assets/submerge-logo.svg"
import { ThemeToggle } from "@/components/theme-toggle"
import { Button } from "@/components/ui/button"
import { UpdateIndicator } from "@/features/app-update/update-indicator"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarRail,
  SidebarTrigger,
  useSidebar,
} from "@/components/ui/sidebar"
import { useAuth } from "@/features/auth/auth-context"

const navItems = [
  { label: "工作概览", path: "/main/dashboard", icon: Gauge },
  { label: "订阅管理", path: "/main/sources", icon: CloudDownload },
  { label: "策略分组", path: "/main/groups", icon: Boxes },
  { label: "分流规则", path: "/main/rules", icon: ListChecks },
  { label: "版本发布", path: "/main/releases", icon: Rocket },
  { label: "订阅链接", path: "/main/tokens", icon: KeyRound },
  { label: "Geo 数据", path: "/main/geo", icon: Database },
  { label: "网络检测", path: "/main/net-check", icon: Network },
  { label: "系统设置", path: "/main/settings/setting-settings", icon: Settings },
  { label: "系统日志", path: "/main/logs", icon: FileClock },
]

export function AppShell() {
  return <SidebarProvider><AppShellContent /></SidebarProvider>
}

function AppShellContent() {
  const { user, logout } = useAuth()
  const { pathname } = useLocation()
  const { isMobile, setOpenMobile, state, toggleSidebar } = useSidebar()
  const closeMobile = () => { if (isMobile) setOpenMobile(false) }
  const initials = (user?.displayName || user?.username || "S").slice(0, 2).toUpperCase()

  return (
    <>
      <Sidebar collapsible="icon">
        <SidebarHeader className="h-[3.75rem] shrink-0 border-b p-3">
          <div className="flex h-9 items-center gap-2">
            <NavLink to="/main/dashboard" onClick={closeMobile} className="flex min-w-0 flex-1 items-center gap-2 overflow-hidden group-data-[collapsible=icon]:hidden">
              <img src={submergeLogo} alt="SubMerge" className="size-8 shrink-0" />
              <div className="min-w-0">
                <div className="truncate text-sm font-semibold">SubMerge</div>
                <div className="mt-0.5 flex w-fit items-center gap-1.5 rounded-full bg-primary/10 px-1.5 py-0.5 text-[10px] font-medium leading-none text-primary"><span className="size-1.5 rounded-full bg-primary" />v{__APP_VERSION__}</div>
              </div>
            </NavLink>
            <Button variant="secondary" size="icon-sm" className="ml-auto shrink-0 rounded-full group-data-[collapsible=icon]:mx-auto" aria-label={state === "expanded" || isMobile ? "收起导航栏" : "展开导航栏"} onClick={toggleSidebar}>
              {state === "expanded" || isMobile ? <ChevronLeft /> : <ChevronRight />}
            </Button>
          </div>
        </SidebarHeader>
        <SidebarContent>
          <SidebarGroup className="px-3 py-3">
            <SidebarGroupContent>
              <SidebarMenu className="gap-1.5">
                {navItems.map((item) => {
                  const active = pathname === item.path || pathname.startsWith(`${item.path}/`)
                  return (
                    <SidebarMenuItem key={item.path}>
                      <SidebarMenuButton asChild tooltip={item.label} isActive={active} className="h-9">
                        <NavLink to={item.path} onClick={closeMobile}><item.icon /><span>{item.label}</span></NavLink>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  )
                })}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        </SidebarContent>
        <SidebarFooter className="border-t p-2">
          <SidebarMenu>
            <SidebarMenuItem>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <SidebarMenuButton size="lg" className="data-open:bg-sidebar-accent data-open:text-sidebar-accent-foreground">
                    <span className="grid size-8 shrink-0 place-items-center overflow-hidden rounded-full border bg-muted text-xs font-semibold">{user?.avatar ? <img src={user.avatar} alt="" className="size-full object-cover" /> : initials}</span>
                    <span className="min-w-0 flex-1 truncate text-left text-sm font-medium">{user?.displayName || user?.username}</span>
                    <ChevronUp className="ml-auto size-4" />
                  </SidebarMenuButton>
                </DropdownMenuTrigger>
                <DropdownMenuContent className="min-w-52 rounded-md border border-sidebar-border bg-sidebar p-1.5 text-sidebar-foreground shadow-lg" align="start" side="top" sideOffset={8}>
                  <DropdownMenuItem className="h-9 gap-2 px-2 focus:bg-sidebar-accent focus:text-sidebar-accent-foreground" asChild><NavLink to="/main/account-setting" onClick={closeMobile}><UserRound />账户设置</NavLink></DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem className="h-9 gap-2 px-2" variant="destructive" onClick={() => void logout()}><LogOut />退出登录</DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarFooter>
        <SidebarRail />
      </Sidebar>
      <SidebarInset>
        <header className="sticky top-0 z-20 flex h-[3.75rem] shrink-0 items-center justify-between border-b bg-background/95 px-4 backdrop-blur lg:px-6">
          <SidebarTrigger className="-ml-1 md:hidden" />
          <div className="ml-auto flex items-center gap-1"><UpdateIndicator /><ThemeToggle /></div>
        </header>
        <main className="w-full flex-1 px-4 py-5 lg:px-6 lg:py-6"><div className="mx-auto w-full max-w-[100rem]"><Outlet /></div></main>
      </SidebarInset>
    </>
  )
}

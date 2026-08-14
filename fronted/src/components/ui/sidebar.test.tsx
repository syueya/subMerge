import { render, screen } from "@testing-library/react"
import { beforeAll, describe, expect, it, vi } from "vitest"

import { Sidebar, SidebarMenu, SidebarMenuButton, SidebarMenuItem, SidebarProvider } from "@/components/ui/sidebar"

describe("SidebarMenuButton", () => {
  beforeAll(() => {
    window.matchMedia = vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }))
  })

  it("only applies the active surface to an explicitly active item", () => {
    render(
      <SidebarProvider>
        <Sidebar>
          <SidebarMenu>
            <SidebarMenuItem><SidebarMenuButton isActive>当前页面</SidebarMenuButton></SidebarMenuItem>
            <SidebarMenuItem><SidebarMenuButton isActive={false}>其它页面</SidebarMenuButton></SidebarMenuItem>
          </SidebarMenu>
        </Sidebar>
      </SidebarProvider>,
    )

    const active = screen.getByRole("button", { name: "当前页面" })
    const inactive = screen.getByRole("button", { name: "其它页面" })

    expect(active).toHaveAttribute("data-active", "true")
    expect(inactive).toHaveAttribute("data-active", "false")
    expect(active.className).toContain("data-[active=true]:bg-sidebar-accent")
    expect(active.className).not.toContain("data-[active=true]:shadow-[inset_2px_0_0_var(--sidebar-primary)]")
    expect(active.className).not.toContain("data-active:bg-sidebar-accent")
  })
})

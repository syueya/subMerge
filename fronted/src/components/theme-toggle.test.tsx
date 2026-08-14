import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { ThemeToggle } from "@/components/theme-toggle"

const themeState = vi.hoisted(() => ({
  theme: "light",
  setTheme: vi.fn(),
}))

vi.mock("next-themes", () => ({
  useTheme: () => themeState,
}))

describe("ThemeToggle", () => {
  beforeEach(() => {
    themeState.setTheme.mockReset()
  })

  it.each([
    ["light", "深色", "dark"],
    ["dark", "跟随系统", "system"],
    ["system", "浅色", "light"],
  ])("switches %s to the next theme on click", async (current, nextLabel, nextTheme) => {
    themeState.theme = current
    const user = userEvent.setup()
    render(<ThemeToggle />)

    await user.click(screen.getByRole("button", { name: new RegExp(`点击切换为${nextLabel}`) }))

    expect(themeState.setTheme).toHaveBeenCalledWith(nextTheme)
  })
})

import { Laptop, Moon, Sun } from "lucide-react"
import { useTheme } from "next-themes"

import { Button } from "@/components/ui/button"

const themeOrder = ["light", "dark", "system"] as const
const themeLabels = { light: "浅色", dark: "深色", system: "跟随系统" } as const

export function ThemeToggle() {
  const { theme, setTheme } = useTheme()
  const currentTheme = themeOrder.includes(theme as (typeof themeOrder)[number])
    ? theme as (typeof themeOrder)[number]
    : "system"
  const nextTheme = themeOrder[(themeOrder.indexOf(currentTheme) + 1) % themeOrder.length]

  return (
    <Button
      variant="ghost"
      size="icon"
      aria-label={`当前${themeLabels[currentTheme]}，点击切换为${themeLabels[nextTheme]}`}
      title={`当前：${themeLabels[currentTheme]}；点击切换为${themeLabels[nextTheme]}`}
      onClick={() => setTheme(nextTheme)}
    >
      {currentTheme === "light" ? <Sun className="size-4" /> : currentTheme === "dark" ? <Moon className="size-4" /> : <Laptop className="size-4" />}
    </Button>
  )
}

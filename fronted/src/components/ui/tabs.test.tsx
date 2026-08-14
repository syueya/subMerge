import { render, screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

describe("Tabs", () => {
  it("adds the primary selected style to line tabs", () => {
    render(
      <Tabs defaultValue="general">
        <TabsList variant="line">
          <TabsTrigger value="general">常规</TabsTrigger>
          <TabsTrigger value="update">更新</TabsTrigger>
        </TabsList>
        <TabsContent value="general">常规内容</TabsContent>
      </Tabs>,
    )

    const selected = screen.getByRole("tab", { name: "常规" })
    const inactive = screen.getByRole("tab", { name: "更新" })

    expect(selected).toHaveAttribute("data-state", "active")
    expect(inactive).toHaveAttribute("data-state", "inactive")
    expect(selected.className).toContain("group-data-[variant=line]/tabs-list:data-[state=active]:bg-primary/10")
    expect(selected.className).toContain("group-data-[variant=line]/tabs-list:data-[state=active]:text-primary")
  })
})

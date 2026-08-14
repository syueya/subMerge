import { render, screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import { PageHeader } from "@/components/page-header"

describe("PageHeader", () => {
  it("renders the page title, description, and actions", () => {
    render(<PageHeader title="订阅源" description="管理上游" actions={<button type="button">刷新</button>} />)
    expect(screen.getByRole("heading", { name: "订阅源" })).toBeInTheDocument()
    expect(screen.getByText("管理上游")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "刷新" })).toBeInTheDocument()
  })
})

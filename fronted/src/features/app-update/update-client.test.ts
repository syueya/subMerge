import { describe, expect, it } from "vitest"

import { updateBusy, updateProgress } from "@/features/app-update/update-client"

describe("app update status helpers", () => {
  it("clamps download progress", () => {
    expect(updateProgress({ downloaded: 75, total: 100 } as never)).toBe(75)
    expect(updateProgress({ downloaded: 150, total: 100 } as never)).toBe(100)
    expect(updateProgress()).toBe(0)
  })

  it("recognizes lifecycle phases that block another action", () => {
    expect(updateBusy({ phase: "downloading" } as never)).toBe(true)
    expect(updateBusy({ phase: "available" } as never)).toBe(false)
  })
})

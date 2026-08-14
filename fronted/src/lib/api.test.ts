import { afterEach, describe, expect, it, vi } from "vitest"

import { api, ApiError, UNAUTHORIZED_EVENT } from "@/lib/api"

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe("api client", () => {
  it("unwraps a successful SubMerge envelope", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ ok: true, data: { value: 3 } }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    })))

    await expect(api.get<{ value: number }>("/example")).resolves.toEqual({ value: 3 })
    expect(fetch).toHaveBeenCalledWith("/api/example", expect.objectContaining({ credentials: "include" }))
  })

  it("surfaces a business error", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({
      ok: false,
      error: { code: "bad_request", message: "字段错误" },
    }), { status: 400, headers: { "Content-Type": "application/json" } })))

    await expect(api.get("/example")).rejects.toMatchObject({
      message: "字段错误",
      status: 400,
      code: "bad_request",
    })
  })

  it("emits the unauthorized event for an expired cookie session", async () => {
    const listener = vi.fn()
    window.addEventListener(UNAUTHORIZED_EVENT, listener)
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({
      ok: false,
      error: { message: "未登录" },
    }), { status: 401, headers: { "Content-Type": "application/json" } })))

    await expect(api.get("/auth/me")).rejects.toBeInstanceOf(ApiError)
    expect(listener).toHaveBeenCalledOnce()
    window.removeEventListener(UNAUTHORIZED_EVENT, listener)
  })
})

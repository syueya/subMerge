import type { ApiEnvelope } from "@/lib/types"

export const UNAUTHORIZED_EVENT = "submerge:unauthorized"

export class ApiError extends Error {
  readonly status: number
  readonly code?: string

  constructor(
    message: string,
    status: number,
    code?: string,
  ) {
    super(message)
    this.name = "ApiError"
    this.status = status
    this.code = code
  }
}

export interface ApiRequestOptions extends Omit<RequestInit, "body"> {
  body?: unknown
  params?: Record<string, string | number | boolean | null | undefined>
  timeoutMs?: number
}

export async function apiRequest<T>(path: string, options: ApiRequestOptions = {}): Promise<T> {
  const { body, params, timeoutMs = 30_000, headers, signal, ...init } = options
  const query = new URLSearchParams()
  for (const [key, value] of Object.entries(params ?? {})) {
    if (value !== null && value !== undefined) query.set(key, String(value))
  }
  const suffix = query.size ? `?${query}` : ""
  const controller = new AbortController()
  const onAbort = () => controller.abort(signal?.reason)
  signal?.addEventListener("abort", onAbort, { once: true })
  const timer = window.setTimeout(() => controller.abort(new Error("请求超时")), timeoutMs)

  try {
    const response = await fetch(`/api${path}${suffix}`, {
      ...init,
      credentials: "include",
      headers: {
        Accept: "application/json",
        ...(body === undefined ? {} : { "Content-Type": "application/json" }),
        ...headers,
      },
      body: body === undefined ? undefined : JSON.stringify(body),
      signal: controller.signal,
    })

    let envelope: ApiEnvelope<T> | undefined
    try {
      envelope = (await response.json()) as ApiEnvelope<T>
    } catch {
      if (response.ok) throw new ApiError("服务器返回了无法解析的响应", response.status)
    }

    if (response.status === 401) {
      window.dispatchEvent(new CustomEvent(UNAUTHORIZED_EVENT))
    }
    if (!response.ok || !envelope?.ok || envelope.data === undefined) {
      throw new ApiError(
        envelope?.error?.message || response.statusText || "请求失败",
        response.status,
        envelope?.error?.code,
      )
    }
    return envelope.data
  } catch (error) {
    if (controller.signal.aborted && !signal?.aborted) {
      throw new ApiError("请求超时，请稍后重试", 0, "timeout")
    }
    throw error
  } finally {
    window.clearTimeout(timer)
    signal?.removeEventListener("abort", onAbort)
  }
}

export const api = {
  get: <T>(path: string, options?: ApiRequestOptions) => apiRequest<T>(path, options),
  post: <T>(path: string, body?: unknown, options?: ApiRequestOptions) =>
    apiRequest<T>(path, { ...options, method: "POST", body: body ?? {} }),
  put: <T>(path: string, body?: unknown, options?: ApiRequestOptions) =>
    apiRequest<T>(path, { ...options, method: "PUT", body: body ?? {} }),
  delete: <T>(path: string, options?: ApiRequestOptions) =>
    apiRequest<T>(path, { ...options, method: "DELETE" }),
}

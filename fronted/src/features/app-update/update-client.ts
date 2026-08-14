import type { HealthResponse, UpdateStatus } from "@/lib/types"

export function updateProgress(status?: UpdateStatus) {
  if (!status?.total || !status.downloaded) return 0
  return Math.min(100, Math.max(0, Math.round((status.downloaded / status.total) * 100)))
}

export function updateBusy(status?: UpdateStatus) {
  return ["checking", "downloading", "installing", "restarting"].includes(status?.phase ?? "")
}

export async function waitForVersion(expectedVersion: string, options: { attempts?: number; intervalMs?: number } = {}) {
  const attempts = options.attempts ?? 90
  const intervalMs = options.intervalMs ?? 2_000
  for (let attempt = 0; attempt < attempts; attempt += 1) {
    try {
      const response = await fetch(`/api/health?t=${Date.now()}`, { cache: "no-store", credentials: "include" })
      if (response.ok) {
        const envelope = await response.json() as { ok?: boolean; data?: HealthResponse }
        if (envelope.ok && envelope.data?.version === expectedVersion) return envelope.data
      }
    } catch {
      // A short connection failure is expected while the supervisor restarts the process.
    }
    await new Promise((resolve) => window.setTimeout(resolve, intervalMs))
  }
  throw new Error(`新版本 ${expectedVersion} 未在预期时间内恢复服务`)
}

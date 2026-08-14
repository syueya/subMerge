export interface ApiErrorBody {
  code?: string
  message?: string
}

export interface ApiEnvelope<T> {
  ok: boolean
  data?: T
  error?: ApiErrorBody
}

export interface ListResponse<T> {
  items: T[]
}

export interface AdminUser {
  id: number
  username: string
  displayName: string
  avatar?: string
  createdAt: string
  lastLoginAt?: string
}

export type SourceKind = "remote" | "manual"
export type RegionMode = "auto" | "fixed"

export interface SubscriptionSource {
  id: number
  name: string
  region: string
  kind: SourceKind
  urlMasked: string
  url?: string
  manualContent?: string
  excludeNameRegex?: string
  excludeServers?: string
  includeNameRegex?: string
  enabled: boolean
  regionMode: RegionMode
  refreshStatus: "idle" | "success" | "failed" | "running"
  lastRefreshAt?: string
  lastError?: string
  proxyCount: number
  trafficUpload: number
  trafficDownload: number
  trafficTotal: number
  trafficExpire: number
  createdAt: string
  updatedAt: string
}

export interface ProxyNode {
  id: number
  sourceId: number
  name: string
  region: string
  type: string
  server: string
  port: number
  enabled: boolean
  ok: boolean
  issue?: string
  udp?: boolean
}

export interface ProxyGroup {
  id: number
  name: string
  type: string
  proxies: string[]
  url?: string
  interval?: number
  enabled: boolean
  sortOrder: number
}

export interface Rule {
  id: number
  type: string
  payload: string
  target: string
  enabled: boolean
  sortOrder: number
  note?: string
  category?: string
}

export interface ShareToken {
  id: number
  name: string
  token?: string
  tokenMasked: string
  status: "active" | "disabled" | "revoked"
  sourceIds: number[]
  sourceNames?: string[]
  groupMode: "auto" | "all" | "custom"
  groupNames?: string[]
  accessCount: number
  lastAccessAt?: string
  createdAt: string
  updatedAt: string
  subscribeUrl?: string
}

export interface Release {
  id: number
  version: number
  status: "draft" | "published" | "rolled_back"
  note?: string
  proxyCount: number
  ruleCount: number
  configHash: string
  publishedAt?: string
  createdAt: string
  createdBy: string
}

export interface DraftChange {
  kind: "proxy" | "group" | "rule"
  action: "added" | "removed" | "modified"
  name: string
  detail?: string
}

export interface DraftStatus {
  hasPublished: boolean
  dirty: boolean
  publishedHash?: string
  draftHash?: string
  publishedVersion?: number
  buildError?: string
  changes?: DraftChange[]
}

export interface HealthResponse {
  status: string
  version: string
  time: string
}

export interface RegionCatalogEntry { code: string; name: string }
export interface RegionCatalogResponse { items: RegionCatalogEntry[]; fallbackRegion: string }

export interface ManualSourceImportResult {
  source: SubscriptionSource
  inputTotal: number
  parsed: number
  previous: number
  kept: number
  added: number
  removed: number
  modified: number
  parseDropped?: Record<string, number>
  regionCounts?: Record<string, number>
  regionConflictTotal?: number
  regionConflicts?: Array<{ name: string; flagRegion: string; keywordRegion: string; resolvedRegion: string }>
  regionConflictOmitted?: number
}

export interface RefreshSourceResult extends ManualSourceImportResult {
  upstreamTotal: number
  skipped: number
  filterDropped?: Record<string, number>
  filteredNames?: string[]
  filteredNamesOmitted?: number
  parseDroppedNames?: string[]
}

export interface RefreshAllResult {
  total: number
  ok: number
  failed: number
  results: Array<{ sourceId: number; name: string; ok: boolean; error?: string }>
}

export interface ReleaseRuleLine { type: string; payload?: string; target: string; raw: string }
export interface ReleaseDetail extends Release { configYaml: string; rules: ReleaseRuleLine[]; groups: string[] }
export interface ReleasePreview { proxyCount: number; ruleCount: number; groups: string[]; yamlPreview: string; warnings: string[] }

export interface ApiKey {
  id: number
  name: string
  key?: string
  keyMasked: string
  scopes: string[]
  status: "active" | "disabled" | "revoked"
  note?: string
  expiresAt?: string
  lastUsedAt?: string
  createdBy: string
  createdAt: string
  updatedAt: string
}

export interface GeoResourceStatus {
  name: string
  available: boolean
  size: number
  modifiedAt?: string
  sha256?: string
  version?: string
  databaseType?: string
  error?: string
}
export interface GeoStatus { items?: GeoResourceStatus[]; resources?: GeoResourceStatus[] }
export interface GeoCategory { name: string; count: number }
export interface GeoCategories { geosite: GeoCategory[]; geoip: GeoCategory[]; metadb: { file: string; supportsReverse: boolean }; asn: { file: string; supportsReverse: boolean } }
export interface GeoQueryResult {
  domain: string
  inputType?: string
  ips: string[]
  resolveSkipped: boolean
  resolveError?: string
  geosite: Array<{ category: string; type: string; value: string }>
  geoip: Array<{ ip: string; category: string; cidr: string }>
  metadb: Array<{ ip: string; cidr: string; record: string }>
  asn: Array<{ ip: string; cidr: string; asn: number; organization: string }>
}
export interface IPGeoResult { ip: string; country?: string; countryCode?: string; region?: string; city?: string; isp?: string; organization?: string; asn?: string }

export interface NetCheckTarget { name: string; url: string; enabled: boolean }
export interface NetCheckConfig { timeout: number; autoRefresh: number; targets: NetCheckTarget[] }
export interface NetCheckResult {
  summary: { total: number; ok: number; fail: number; durationMs: number; checkedAt: string }
  results: Array<{ name: string; url: string; status: string; checkedAt: string; http: { ok: boolean; status: string; code: number; timeMs: number; remoteIp?: string; effectiveUrl?: string; error?: string; timing: { connectMs: number; tlsMs: number; firstByteMs: number; totalMs: number } } }>
  proxyInfo?: string
  proxyMode?: string
}

export interface LogFileInfo { name: string; size: number; updatedAt: number }
export interface LogEntry { timestamp: number; caller: string; content: string; level: string }

export type UpdatePhase = "disabled" | "idle" | "checking" | "available" | "downloading" | "ready" | "installing" | "restarting" | "failed"
export interface UpdateStatus {
  enabled: boolean
  phase: UpdatePhase
  currentVersion: string
  latestVersion?: string
  available: boolean
  releaseUrl?: string
  notes?: string
  publishedAt?: string
  downloaded?: number
  total?: number
  error?: string
  checkedAt?: string
  rollbackAvailable?: boolean
  rollbackVersion?: string
}

export function formatDateTime(value?: string | number) {
  if (!value) return "-"
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? "-" : date.toLocaleString("zh-CN", { hour12: false })
}

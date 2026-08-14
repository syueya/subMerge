export const queryKeys = {
  dashboard: ["dashboard"] as const,
  sources: ["sources"] as const,
  regions: ["regions"] as const,
  proxies: (sourceId?: number) => ["proxies", sourceId ?? "all"] as const,
  groups: ["groups"] as const,
  rules: ["rules"] as const,
  releases: ["releases"] as const,
  draft: ["release-draft"] as const,
  preview: ["release-preview"] as const,
  tokens: ["tokens"] as const,
  geo: ["geo"] as const,
  netCheck: ["net-check"] as const,
  apiKeys: ["api-keys"] as const,
  settings: ["system-settings"] as const,
  logs: ["logs"] as const,
  update: ["app-update"] as const,
}

export const draftDependencies = [
  queryKeys.dashboard,
  queryKeys.draft,
  queryKeys.preview,
] as const

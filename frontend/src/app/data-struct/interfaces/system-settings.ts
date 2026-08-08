export interface SystemSettingsView {
  settings: {
    sourceFetchUA: string;
    sourceFetchTimeout: string;
    sourceMaxBytes: number;
    refreshInterval: number;
    geoipUrl: string;
    geositeUrl: string;
    geodbUrl: string;
    geoasnUrl: string;
    ipGeoUrl: string;
    ipGeoTimeout: string;
    logOutput: 'console' | 'file' | 'both' | 'none';
    debugLogging: boolean;
    logRetentionDays: number;
    proxyEnabled: boolean;
    proxyConfigured: boolean;
    proxyMaskedUrl?: string;
    publicBaseUrl: string;
    trustedProxies: string;
    cookieSecure: boolean;
  };
  source: Record<string, 'web' | 'default'>;
  override: Record<string, boolean>;
  restartRequired: boolean;
}

export interface SystemSettingsUpdate {
  sourceFetchUA?: string;
  sourceFetchTimeout?: string;
  sourceMaxBytes?: number;
  refreshInterval?: number;
  geoipUrl?: string;
  geositeUrl?: string;
  geodbUrl?: string;
  geoasnUrl?: string;
  ipGeoUrl?: string;
  ipGeoTimeout?: string;
  logOutput?: 'console' | 'file' | 'both' | 'none';
  debugLogging?: boolean;
  logRetentionDays?: number;
  proxyEnabled?: boolean;
  proxyUrl?: string;
  publicBaseUrl?: string;
  trustedProxies?: string;
  cookieSecure?: boolean;
}

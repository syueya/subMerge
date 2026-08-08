export interface OutboundProxyView {
  enabled: boolean;
  configured: boolean;
  source: 'environment' | 'web';
  maskedUrl?: string;
  hasOverride: boolean;
}

export interface OutboundProxyUpdate {
  enabled?: boolean;
  url?: string;
}

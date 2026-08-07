export interface NetCheckTarget {
  name: string;
  url: string;
  enabled: boolean;
}

export interface NetCheckConfig {
  timeout: number;
  autoRefresh: number;
  targets: NetCheckTarget[];
}

export interface NetCheckProxy {
  enabled: boolean;
  url: string;
}

export interface NetCheckRequest {
  proxy?: NetCheckProxy;
  timeout?: number;
  autoRefresh?: number;
  targets?: NetCheckTarget[];
}

export interface NetCheckTiming {
  connectMs: number;
  tlsMs: number;
  firstByteMs: number;
  totalMs: number;
}

export interface NetCheckHttpResult {
  ok: boolean;
  status: 'OK' | 'FAIL';
  code: number;
  timeMs: number;
  timing: NetCheckTiming;
  remoteIp?: string;
  effectiveUrl?: string;
  error?: string;
}

export interface NetCheckResult {
  name: string;
  url: string;
  status: 'OK' | 'FAIL';
  checkedAt: string;
  http: NetCheckHttpResult;
}

export interface NetCheckSummary {
  total: number;
  ok: number;
  fail: number;
  durationMs: number;
  checkedAt: string;
}

export interface NetCheckResponse {
  summary: NetCheckSummary;
  results: NetCheckResult[];
}

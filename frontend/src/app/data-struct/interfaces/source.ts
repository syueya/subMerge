import type { Region } from '../enums/region';
import type { RefreshStatus } from '../enums/source';

/** 订阅源地区模式：auto=按节点名识别，fixed=全部用源默认地区 */
export type RegionMode = 'auto' | 'fixed';

/** 新建源默认过滤（与后端 DefaultExclude* 保持一致） */
export const DEFAULT_EXCLUDE_NAME_REGEX = '剩余流量|套餐到期|流量|到期|过期|官网|电报|重置|距离下次|消耗|续费|客服|公告|测试|过滤掉|过滤了|已过滤';
export const DEFAULT_EXCLUDE_SERVERS = '127.0.0.1,0.0.0.0,localhost';

export interface SubscriptionSource {
  id: number;
  name: string;
  region: Region;
  urlMasked: string;
  url: string;
  enabled: boolean;
  regionMode: RegionMode;
  excludeNameRegex: string;
  excludeServers: string;
  includeNameRegex: string;
  refreshStatus: RefreshStatus;
  lastRefreshAt?: string | null;
  lastError?: string | null;
  proxyCount: number;
  /** 上游 Subscription-Userinfo，字节；0 = 未知 */
  trafficUpload?: number;
  trafficDownload?: number;
  trafficTotal?: number;
  /** Unix 秒；0 = 未知/不限 */
  trafficExpire?: number;
  createdAt: string;
  updatedAt: string;
}

export interface ProxyNode {
  id: number;
  sourceId: number;
  name: string;
  region: Region;
  type: string;
  server: string;
  port: number;
  enabled: boolean;
  /** 上游订阅中的 UDP 配置；未提供时为 undefined */
  udp?: boolean;
  /** 是否正常线路 */
  ok?: boolean;
  /** 非正常原因 */
  issue?: string;
}

export interface SourceUpsertBody {
  name?: string;
  region?: Region;
  url?: string;
  enabled?: boolean;
  regionMode?: RegionMode;
  excludeNameRegex?: string;
  excludeServers?: string;
  includeNameRegex?: string;
}

export interface RefreshSourceResult {
  source: SubscriptionSource;
  upstreamTotal?: number;
  parsed?: number;
  previous?: number;
  kept?: number;
  added: number;
  removed: number;
  modified?: number;
  skipped: number;
  parseDropped?: Record<string, number>;
  filterDropped?: Record<string, number>;
  /** 被过滤掉的上游节点名样本，最多返回 1000 条 */
  filteredNames?: string[];
  /** 因响应上限未返回的过滤节点名数量 */
  filteredNamesOmitted?: number;
  parseDroppedNames?: string[];
  regionCounts?: Record<string, number>;
  regionConflictTotal?: number;
  regionConflicts?: RegionConflict[];
  regionConflictOmitted?: number;
}

export interface RegionConflict {
  name: string;
  flagRegion: string;
  flagMatched?: string;
  keywordRegion: string;
  keywordMatched?: string;
  resolvedRegion: string;
}

export interface RefreshAllItem {
  sourceId: number;
  name: string;
  ok: boolean;
  previous?: number;
  kept?: number;
  added?: number;
  removed?: number;
  modified?: number;
  skipped?: number;
  regionConflictTotal?: number;
  error?: string;
}

export interface RefreshAllResult {
  total: number;
  ok: number;
  failed: number;
  results: RefreshAllItem[];
}

export interface RegionCatalogEntry {
  code: string;
  name: string;
}

export interface RegionCatalogResponse {
  items: RegionCatalogEntry[];
  fallbackRegion: string;
}

/** 拉取成功结果的展示侧字段（list / form 摘要文案共用） */
export interface RefreshResultLike {
  source: SubscriptionSource;
  upstreamTotal?: number;
  parsed?: number;
  previous?: number;
  kept?: number;
  added: number;
  removed?: number;
  modified?: number;
  skipped: number;
  parseDropped?: Record<string, number>;
  filterDropped?: Record<string, number>;
  filteredNames?: string[];
  filteredNamesOmitted?: number;
  regionCounts?: Record<string, number>;
  regionConflictTotal?: number;
  regionConflicts?: RegionConflict[];
  regionConflictOmitted?: number;
}

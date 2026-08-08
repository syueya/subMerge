import type { APIKey } from './api-key';
import type { GeoEntryRow } from './geo';
import type { NetCheckConfig } from './net-check';
import type { ProxyGroup } from './proxy-group';
import type { DraftChange, ReleaseDetail } from './release';
import type { Rule } from './rule';
import type { RegionCatalogEntry, SubscriptionSource } from './source';
import type { ShareToken } from './token';

export interface NetCheckTargetManageDialogData {
  config: NetCheckConfig;
}

export interface NetCheckTargetManageDialogResult {
  config: NetCheckConfig;
}

/** 创建 / 编辑令牌共用弹窗数据；token 有值时为编辑模式 */
export interface TokenFormDialogData {
  token?: ShareToken | null;
  sourceList: SubscriptionSource[];
  groupList: ProxyGroup[];
}

/** @deprecated 使用 TokenFormDialogData（传 token 即为编辑） */
export type TokenEditDialogData = TokenFormDialogData;

/** 创建 / 编辑 API 密钥弹窗；key 有值时为编辑模式 */
export interface APIKeyFormDialogData {
  key?: APIKey | null;
}

/** 展示完整 API 密钥弹窗 */
export interface APIKeySecretDialogData {
  name: string;
  key: string;
  /** 轮换/创建后的提示文案 */
  hint?: string;
}

export interface SourceFormDialogData {
  source: SubscriptionSource | null;
  regionCatalog: RegionCatalogEntry[];
  fallbackRegion: string;
}

export interface SourceProxiesDialogData {
  source: SubscriptionSource;
}

export interface RuleFormDialogData {
  rule: Rule | null;
  groups: ProxyGroup[];
  rules: Rule[];
  extraCategories: string[];
  defaultCategory?: string;
  defaultTarget?: string;
  defaultType?: Rule['type'];
  defaultPayload?: string;
}

export interface BatchImportDialogData {
  groups: ProxyGroup[];
  rules: Rule[];
  extraCategories: string[];
  defaultTarget?: string;
}

export interface GroupFormDialogData {
  group: ProxyGroup | null;
  groups: ProxyGroup[];
  regionCatalog: { code: string; name: string }[];
  extraRegionCodes: string[];
  knownSources: { id: number; name: string }[];
}

export interface ReleaseDetailDialogData {
  detail: ReleaseDetail;
}

/** 草稿相对已发布版本的差异弹窗 */
export interface DraftChangesDialogData {
  changes: DraftChange[];
  summary?: string;
  publishedVersion?: number;
}

export interface GeoEntriesAddRuleContext {
  /** 仅 GeoSite / GeoIP 可直接转换为 Clash 规则 */
  type: 'GEOSITE' | 'GEOIP';
  payload: string;
}

export interface GeoEntriesAddRuleResult {
  action: 'add';
  context: GeoEntriesAddRuleContext;
}

export interface GeoEntriesDialogData {
  title: string;
  subtitle: string;
  items: GeoEntryRow[];
  total: number;
  offset: number;
  limit: number;
  loading: boolean;
  paginated: boolean;
  /** GeoSite / GeoIP 条目可直接带入添加规则表单 */
  addRule?: GeoEntriesAddRuleContext;
  /** 分页变化（不关闭弹窗，由父组件重新拉数后 applyData） */
  onPage?: (pageIndex: number, pageSize: number) => void;
}

export type GeoEntriesDialogResult = null | GeoEntriesAddRuleResult;

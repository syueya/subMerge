import type {
  GeoEntryRow,
  NetCheckConfig,
  ProxyGroup,
  RegionCatalogEntry,
  ReleaseDetail,
  Rule,
  ShareToken,
  SubscriptionSource,
} from './types';

export interface NetCheckTargetManageDialogData {
  config: NetCheckConfig;
}

export interface NetCheckTargetManageDialogResult {
  config: NetCheckConfig;
}


export interface TokenFormDialogData {
	sourceList: SubscriptionSource[];
	groupList: ProxyGroup[];
}

export interface TokenEditDialogData {
	token: ShareToken;
	sourceList: SubscriptionSource[];
	groupList: ProxyGroup[];
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

export interface GeoEntriesDialogData {
	title: string;
	subtitle: string;
	items: GeoEntryRow[];
	total: number;
	offset: number;
	limit: number;
	loading: boolean;
	paginated: boolean;
	/** 分页变化（不关闭弹窗，由父组件重新拉数后 applyData） */
	onPage?: (pageIndex: number, pageSize: number) => void;
}

export type GeoEntriesDialogResult = null;

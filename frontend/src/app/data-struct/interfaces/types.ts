// Re-export shared contracts used by the panel.

export {
				 FALLBACK_REGION,
				 RuleType,
				 TokenStatus,
				 TokenGroupMode,
				 ReleaseStatus,
				 RefreshStatus,
				 ProxyGroupType,
				 regionLabel,
				 regionOptionText,
				 RULE_TYPE_OPTIONS,
				 TOKEN_STATUS_OPTIONS,
				 TOKEN_STATUS_BADGE,
				 TOKEN_GROUP_MODE_OPTIONS,
				 RELEASE_STATUS_OPTIONS,
				 RELEASE_STATUS_BADGE,
				 REFRESH_STATUS_OPTIONS,
				 REFRESH_STATUS_BADGE,
				 PROXY_GROUP_TYPE_OPTIONS,
				 BADGE_OK,
				 BADGE_WARN,
				 BADGE_ERR,
				 BADGE_MUTED,
				 enumText,
				 enumBadgeClass,
			} from '../enums/enums';
		
		export type {
		 Region,
		 Region as RegionValue,
		 RuleType as RuleTypeValue,
		 TokenStatus as TokenStatusValue,
		 TokenGroupMode as TokenGroupModeValue,
		 ReleaseStatus as ReleaseStatusValue,
		 RefreshStatus as RefreshStatusValue,
		 ProxyGroupType as ProxyGroupTypeValue,
		 EnumOption,
		} from '../enums/enums';
	
	import type {
	 Region,
	 RuleType,
	 TokenStatus,
	 TokenGroupMode,
	 ReleaseStatus,
	 RefreshStatus,
	 ProxyGroupType,
	} from '../enums/enums';

export interface ApiError {
 code: string;
 message: string;
 details?: string;
}

export interface ApiResponse<T> {
 ok: boolean;
 data?: T;
 error?: ApiError;
}

export interface ListResponse<T> {
 items: T[];
}

export interface AdminUser {
	 id: number;
	 username: string;
	 displayName: string;
	 avatar?: string;
	 createdAt: string;
	 lastLoginAt?: string | null;
}

export interface LoginRequest {
 username: string;
 password: string;
}

export interface LoginResponse {
 // 会话令牌改由 HttpOnly cookie 承载，不再出现在响应体中
 user: AdminUser;
}

/** 订阅源地区模式：auto=按节点名识别，fixed=全部用源默认地区 */
export type RegionMode = 'auto' | 'fixed';

/** 新建源默认过滤（与后端 DefaultExclude* 保持一致） */
export const DEFAULT_EXCLUDE_NAME_REGEX =
	'剩余流量|套餐到期|流量|到期|过期|官网|电报|重置|距离下次|消耗|续费|客服|公告|测试|过滤掉|过滤了|已过滤';
export const DEFAULT_EXCLUDE_SERVERS = '127.0.0.1,0.0.0.0,localhost';

export interface SubscriptionSource {
		 id: number;
		 name: string;
		 region: Region;
		 urlMasked: string;
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
		added: number;
		removed: number;
		skipped: number;
		parseDropped?: Record<string, number>;
		filterDropped?: Record<string, number>;
		/** 被过滤掉的上游节点名样本，最多返回 1000 条 */
		filteredNames?: string[];
		/** 因响应上限未返回的过滤节点名数量 */
		filteredNamesOmitted?: number;
		parseDroppedNames?: string[];
		regionCounts?: Record<string, number>;
	}

	export interface RefreshAllItem {
		sourceId: number;
		name: string;
		ok: boolean;
		added?: number;
		skipped?: number;
		error?: string;
	}

	export interface RefreshAllResult {
		total: number;
		ok: number;
		failed: number;
		results: RefreshAllItem[];
	}

	export interface DraftStatus {
		hasPublished: boolean;
		dirty: boolean;
		publishedHash?: string;
		draftHash?: string;
		publishedVersion?: number;
		buildError?: string;
		/** 草稿相对已发布配置的实体级变更列表（节点/策略组/规则） */
		changes?: DraftChange[];
	}

	/** 单条草稿变更（相对当前已发布版本） */
	export interface DraftChange {
		/** proxy（节点）| group（策略组）| rule（分流规则） */
		kind: 'proxy' | 'group' | 'rule' | string;
		/** added（新增）| removed（删除）| modified（修改） */
		action: 'added' | 'removed' | 'modified' | string;
		/** 变更对象名称（规则用规则内容） */
		name: string;
		/** 变更细节，可空 */
		detail?: string;
	}

export interface RegionCatalogEntry {
	 code: string;
	 name: string;
}

export interface RegionCatalogResponse {
	 items: RegionCatalogEntry[];
	 fallbackRegion: string;
}

export interface Rule {
	 id: number;
	 type: RuleType | string;
	 payload: string;
	 target: string;
	 enabled: boolean;
	 sortOrder: number;
	 note?: string;
	 /** 业务分类（仅面板分组，不进 Clash） */
	 category?: string;
	}

/** 预设业务分类顺序（面板展示用） */
		export const RULE_CATEGORY_ORDER = [
			'系统分类', // 广告 / 国内 GEOIP / MATCH 兜底
			'海外AI',
			'国内AI',
			'流媒体',
			'影视元数据',
			'电报',
			'开发',
			'Google',
			'社交',
			'其它',
			'PT站',
		] as const;

	export const RULE_CATEGORY_OPTIONS = [
		...RULE_CATEGORY_ORDER.map((c) => ({ value: c, text: c })),
		{ value: '', text: '未分类' },
	];

export interface ProxyGroup {
 id: number;
 name: string;
 type: ProxyGroupType | string;
 proxies: string[];
 url?: string;
 interval?: number;
 enabled: boolean;
 sortOrder: number;
}

export interface ShareToken {
		 id: number;
		 name: string;
		 token?: string;
		 tokenMasked: string;
		 status: TokenStatus;
		 /** 允许的订阅源；空数组 = 全部源 */
		 sourceIds: number[];
		 /** 与 sourceIds 对应的源名（已失效源会带标记） */
		 sourceNames?: string[];
		 /** 策略组投影：auto / all / custom */
		 groupMode: TokenGroupMode;
		 /** custom 时的策略组白名单 */
		 groupNames?: string[];
		 accessCount: number;
		 lastAccessAt?: string | null;
		 createdAt: string;
		 updatedAt: string;
		 subscribeUrl?: string;
		}

export interface Release {
	 id: number;
	 version: number;
	 status: ReleaseStatus;
	 note?: string;
	 proxyCount: number;
	 ruleCount: number;
	 configHash: string;
	 publishedAt?: string | null;
	 createdAt: string;
	 createdBy: string;
	}

	/** 发布配置中的单条规则（历史查看 / 匹配测试） */
	export interface ReleaseRuleLine {
		type: string;
		payload?: string;
		target: string;
		raw: string;
	}

	/** 发布版本详情 */
	export interface ReleaseDetail extends Release {
		configYaml: string;
		rules: ReleaseRuleLine[];
		groups: string[];
	}

	export interface ReleasePreview {
	 proxyCount: number;
	 ruleCount: number;
	 groups: string[];
	 yamlPreview: string;
	 warnings: string[];
	}

export interface PublishResponse {
 release: Release;
 preview: ReleasePreview;
}

/** 健康检查 / 应用版本 */
export interface HealthResponse {
	status: string;
	version: string;
	time: string;
}

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

export interface GeoStatus {
	name: string;
	available: boolean;
	size: number;
	modifiedAt?: string;
	sha256?: string;
	version?: string;
	databaseType?: string;
	buildEpoch?: number;
	error?: string;
}

export interface GeoCategory {
	name: string;
	count: number;
}

export interface GeoCategoryInfo {
	file: string;
	supportsReverse: boolean;
}

export interface GeoCategoriesResponse {
	geosite: GeoCategory[];
	geoip: GeoCategory[];
	metadb: GeoCategoryInfo;
	asn: GeoCategoryInfo;
}

export interface GeoSiteHit {
	category: string;
	type: string;
	value: string;
}

export interface GeoIPHit {
	ip: string;
	category: string;
	cidr: string;
}

export interface GeoDBHit {
	ip: string;
	cidr: string;
	record: string;
}

export interface GeoASNHit {
	ip: string;
	cidr: string;
	asn: number;
	organization: string;
}

export interface GeoQueryResponse {
	domain: string;
	/** "domain" | "ip" — present after geo query enhancement */
	inputType?: 'domain' | 'ip' | string;
	ips: string[];
	resolveSkipped: boolean;
	resolveError?: string;
	geosite: GeoSiteHit[];
	geoip: GeoIPHit[];
	metadb: GeoDBHit[];
	asn: GeoASNHit[];
}

export interface GeoReverseItem {
	value: string;
	type: string;
}

export interface GeoReverseResponse {
	file: string;
	category: string;
	items?: GeoReverseItem[];
	total: number;
	limit: number;
	offset: number;
	message?: string;
}

export interface GeoSearchResponse {
	file: string;
	field: string;
	keyword: string;
	items?: GeoEntryRow[];
	total: number;
	limit: number;
	offset: number;
	message?: string;
}

export interface GeoEntryRow {
	type: string;
	value: string;
	detail?: string;
}

export interface GeoUpdateItem {
	name: string;
	updated: boolean;
	error?: string;
}

export interface GeoUpdateResponse {
	items: GeoUpdateItem[];
}

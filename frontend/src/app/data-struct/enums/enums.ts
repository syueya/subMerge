/** 带中文文案的枚举项 */
export interface EnumOption<V extends string = string> {
 value: V;
 text: string;
}

export function enumText(
 options: readonly EnumOption[],
 value: string | null | undefined,
 fallback = value ?? '-',
): string {
 if (value == null || value === '') return String(fallback);
 return options.find((o) => o.value === value)?.text ?? String(fallback);
}

/** 状态徽标：复用 assets/scss 工具类，不引入自定义 badge-* */
export const BADGE_OK = 'bg-light-success text-success rounded f-s-12 f-w-600 p-x-8 p-y-2';
export const BADGE_WARN = 'bg-light-warning text-warning rounded f-s-12 f-w-600 p-x-8 p-y-2';
export const BADGE_ERR = 'bg-light-error text-error rounded f-s-12 f-w-600 p-x-8 p-y-2';
export const BADGE_MUTED = 'bg-light-primary text-primary rounded f-s-12 f-w-600 p-x-8 p-y-2';

export function enumBadgeClass(
 map: Record<string, string>,
 value: string | null | undefined,
 fallback = BADGE_MUTED,
): string {
 if (!value) return fallback;
 return map[value] ?? fallback;
}

// --- Region ---
// 地区目录只维护 backend/defaults/regions.yaml（经 GET /regions）。
// 前端不再内置地区列表；展示名由 API 的 name 提供。

export type Region = string;

/** 自动识别失败时的回退地区（与后端 FallbackRegion 一致） */
export const FALLBACK_REGION = 'UNKNOWN';

/** 地区展示名：优先用目录 labels；无则回退地区码本身 */
export function regionLabel(code: string, labels?: Record<string, string>): string {
	const c = String(code || '').toUpperCase();
	if (!c) return '';
	const name = labels?.[c]?.trim();
	return name || c;
}

/** 下拉/列表文案：美国 (US)；无中文名时仅码 */
export function regionOptionText(code: string, name?: string): string {
	const c = String(code || '').toUpperCase();
	const n = (name || '').trim();
	if (!c) return n;
	if (!n || n === c) return c;
	return `${n} (${c})`;
}

// --- RuleType ---

export type RuleType =
		| 'DOMAIN'
		| 'DOMAIN-SUFFIX'
		| 'DOMAIN-KEYWORD'
		| 'GEOSITE'
		| 'GEOIP'
		| 'IP-CIDR'
		| 'IP-CIDR6'
		| 'MATCH';

	export const RuleType = {
		DOMAIN: 'DOMAIN',
		DOMAIN_SUFFIX: 'DOMAIN-SUFFIX',
		DOMAIN_KEYWORD: 'DOMAIN-KEYWORD',
		GEOSITE: 'GEOSITE',
		GEOIP: 'GEOIP',
		IP_CIDR: 'IP-CIDR',
		IP_CIDR6: 'IP-CIDR6',
		MATCH: 'MATCH',
	} as const;

	export const RULE_TYPE_OPTIONS: readonly EnumOption<RuleType>[] = [
		{ value: 'DOMAIN', text: '域名' },
		{ value: 'DOMAIN-SUFFIX', text: '域名后缀' },
		{ value: 'DOMAIN-KEYWORD', text: '域名关键词' },
		{ value: 'GEOSITE', text: 'GeoSite' },
		{ value: 'GEOIP', text: 'GeoIP' },
		{ value: 'IP-CIDR', text: 'IP-CIDR' },
		{ value: 'IP-CIDR6', text: 'IP-CIDR6' },
		{ value: 'MATCH', text: '最终匹配' },
	];

// --- TokenStatus ---

export type TokenStatus = 'active' | 'disabled' | 'revoked';

export const TokenStatus = {
 Active: 'active',
 Disabled: 'disabled',
 Revoked: 'revoked',
} as const;

export const TOKEN_STATUS_OPTIONS: readonly EnumOption<TokenStatus>[] = [
 { value: 'active', text: '启用' },
 { value: 'disabled', text: '禁用' },
 { value: 'revoked', text: '已撤销' },
];

export const TOKEN_STATUS_BADGE: Record<TokenStatus, string> = {
 active: BADGE_OK,
 disabled: BADGE_WARN,
 revoked: BADGE_ERR,
};

// --- TokenGroupMode（令牌策略组投影）---

export type TokenGroupMode = 'auto' | 'all' | 'custom';

export const TokenGroupMode = {
 Auto: 'auto',
 All: 'all',
 Custom: 'custom',
} as const;

export const TOKEN_GROUP_MODE_OPTIONS: readonly EnumOption<TokenGroupMode>[] = [
 { value: 'auto', text: '自动' },
 { value: 'all', text: '全部' },
 { value: 'custom', text: '自定义' },
];

// --- ReleaseStatus ---

export type ReleaseStatus = 'draft' | 'published' | 'rolled_back';

export const ReleaseStatus = {
 Draft: 'draft',
 Published: 'published',
 RolledBack: 'rolled_back',
} as const;

export const RELEASE_STATUS_OPTIONS: readonly EnumOption<ReleaseStatus>[] = [
	 { value: 'draft', text: '草稿' },
	 { value: 'published', text: '当前生效' },
	 // 被更新发布/回滚顶替后的旧版，并非「曾执行过回滚操作」
	 { value: 'rolled_back', text: '历史版本' },
	];

export const RELEASE_STATUS_BADGE: Record<ReleaseStatus, string> = {
 draft: BADGE_WARN,
 published: BADGE_OK,
 rolled_back: BADGE_MUTED,
};

// --- RefreshStatus ---

export type RefreshStatus = 'idle' | 'success' | 'failed' | 'running';

export const RefreshStatus = {
 Idle: 'idle',
 Success: 'success',
 Failed: 'failed',
 Running: 'running',
} as const;

export const REFRESH_STATUS_OPTIONS: readonly EnumOption<RefreshStatus>[] = [
 { value: 'idle', text: '未刷新' },
 { value: 'success', text: '成功' },
 { value: 'failed', text: '失败' },
 { value: 'running', text: '拉取中' },
];

export const REFRESH_STATUS_BADGE: Record<RefreshStatus, string> = {
 idle: BADGE_MUTED,
 success: BADGE_OK,
 failed: BADGE_ERR,
 running: BADGE_WARN,
};

// --- ProxyGroupType ---

export type ProxyGroupType = 'select' | 'url-test' | 'fallback' | 'load-balance';

export const ProxyGroupType = {
 Select: 'select',
 UrlTest: 'url-test',
 Fallback: 'fallback',
 LoadBalance: 'load-balance',
} as const;

export const PROXY_GROUP_TYPE_OPTIONS: readonly EnumOption<ProxyGroupType>[] = [
 { value: 'select', text: '手动选择' },
 { value: 'url-test', text: '自动测速' },
 { value: 'fallback', text: '故障转移' },
 { value: 'load-balance', text: '负载均衡' },
];

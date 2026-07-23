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

export function enumBadgeClass(
 map: Record<string, string>,
 value: string | null | undefined,
 fallback = 'badge-muted',
): string {
 if (!value) return fallback;
 return map[value] ?? fallback;
}

// --- Region ---
// 地区码以 defaults/regions.yaml 为准；前端优先用 GET /regions 目录。
// 下列常量仅作离线/首屏兜底，与后端目录大致对齐。

export type Region = string;

/** 自动识别失败时的回退地区（与后端 regioncatalog.FallbackCode 一致） */
export const FALLBACK_REGION = 'UNKNOWN';

export const Region = {
	 UNKNOWN: 'UNKNOWN',
	 US: 'US',
	 PH: 'PH',
	 JP: 'JP',
	 HK: 'HK',
	 SG: 'SG',
	 TW: 'TW',
} as const;

/** 离线兜底选项；正式 UI 用 API 目录覆盖 */
export const REGION_OPTIONS: readonly EnumOption[] = [
	 { value: 'UNKNOWN', text: '未知 (UNKNOWN)' },
	 { value: 'US', text: '美国 (US)' },
	 { value: 'JP', text: '日本 (JP)' },
	 { value: 'HK', text: '香港 (HK)' },
	 { value: 'TW', text: '台湾 (TW)' },
	 { value: 'SG', text: '新加坡 (SG)' },
	 { value: 'KR', text: '韩国 (KR)' },
	 { value: 'GB', text: '英国 (GB)' },
	 { value: 'PH', text: '菲律宾 (PH)' },
	 { value: 'TR', text: '土耳其 (TR)' },
	 { value: 'DE', text: '德国 (DE)' },
];

/** 离线兜底中文名 */
export const REGION_LABELS: Record<string, string> = {
	UNKNOWN: '未知',
	US: '美国',
	PH: '菲律宾',
	JP: '日本',
	HK: '香港',
	SG: '新加坡',
	TW: '台湾',
	KR: '韩国',
	DE: '德国',
	GB: '英国',
	FR: '法国',
	CA: '加拿大',
	AU: '澳洲',
	NL: '荷兰',
	IN: '印度',
	MY: '马来西亚',
	TH: '泰国',
	VN: '越南',
	TR: '土耳其',
	RU: '俄罗斯',
	IT: '意大利',
	BR: '巴西',
MO: '澳门',
		CN: '中国',
		NG: '尼日利亚',
		UA: '乌克兰',
	};

export function regionLabel(code: string, labels?: Record<string, string>): string {
	const c = String(code || '').toUpperCase();
	const map = labels || REGION_LABELS;
	return map[c] || c;
}

export function regionOptionText(code: string, name?: string): string {
	const c = String(code || '').toUpperCase();
	const n = (name || REGION_LABELS[c] || '').trim();
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
 active: 'badge-ok',
 disabled: 'badge-warn',
 revoked: 'badge-err',
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
 draft: 'badge-warn',
 published: 'badge-ok',
 rolled_back: 'badge-muted',
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
 idle: 'badge-muted',
 success: 'badge-ok',
 failed: 'badge-err',
 running: 'badge-warn',
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

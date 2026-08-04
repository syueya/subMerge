/** 规则匹配用的可匹配规则（前端快照 / 后端 API 共用） */
export interface MatchableRule {
	type: string;
	payload?: string;
	target: string;
	enabled?: boolean;
	/** 展示用原文，可选 */
	raw?: string;
}

/** GEOSITE/GEOIP 命中时后端附带的 dat 条目 */
export interface RuleMatchGeoHit {
	category: string;
	type?: string;
	value?: string;
	cidr?: string;
	ip?: string;
}

export interface RuleMatchResult {
	input: string;
	host: string;
	kind: 'domain' | 'ipv4' | 'ipv6' | 'empty' | 'invalid';
	/**
	 * 是否命中「具体」规则（DOMAIN / GEOSITE / IP-CIDR 等）。
	 * 落入最终 MATCH 兜底时为 false（算未命中业务规则，仅兜底出口）。
	 */
	matched: boolean;
	/**
	 * 是否落入 MATCH 兜底（matched 为 false 时仍可能有 rule）。
	 * 始终给出布尔值，避免模板侧 optional 属性推断异常。
	 */
	fallbackMatch: boolean;
	rule: MatchableRule | null;
	/** 扫描过但未命中的规则条数 */
	skipped: number;
	note: string;
	/** GEOSITE/GEOIP 命中条目（服务端匹配时可能返回） */
	geoHit?: RuleMatchGeoHit | null;
	ips?: string[];
	resolveSkipped?: boolean;
	resolveError?: string;
}

export interface RuleMatchDialogData {
	title?: string;
	subtitle?: string;
	rules: MatchableRule[];
	loading?: boolean;
	typeText?: (type: string) => string;
	targetText?: (target: string) => string;
	showEditAction?: boolean;
	showLocateAction?: boolean;
	canLocate?: (rule: MatchableRule) => boolean;
}

export type RuleMatchDialogResult =
	| { action: 'edit'; rule: MatchableRule }
	| { action: 'locate'; rule: MatchableRule }
	| null;

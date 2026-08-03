/** 浏览器侧规则匹配用的可匹配规则 */
export interface MatchableRule {
	type: string;
	payload?: string;
	target: string;
	enabled?: boolean;
	/** 展示用原文，可选 */
	raw?: string;
}

export interface RuleMatchResult {
	input: string;
	host: string;
	kind: 'domain' | 'ipv4' | 'ipv6' | 'empty' | 'invalid';
	/**
	 * 是否命中「具体」规则（DOMAIN / IP-CIDR 等）。
	 * 落入最终 MATCH 兜底时为 false（算未命中业务规则，仅兜底出口）。
	 */
	matched: boolean;
	/**
	 * 是否落入 MATCH 兜底（matched 为 false 时仍可能有 rule）。
	 * 始终给出布尔值，避免模板侧 optional 属性推断异常。
	 */
	fallbackMatch: boolean;
	rule: MatchableRule | null;
	/** 扫描过但未命中的规则条数（含跳过的 GEOSITE 等） */
	skipped: number;
	note: string;
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

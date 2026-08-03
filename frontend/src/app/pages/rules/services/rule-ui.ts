import {
	CategoryOption,
	CategorySection,
	RULE_CATEGORY_ORDER,
	Rule,
	RuleType,
} from '@data-struct';

/** 下拉专用：新建分类哨兵值 */
export const CATEGORY_NEW_VALUE = '__new__';

export type { CategoryOption, CategorySection };

/** 系统托管规则：广告 / 国内 GEOIP / MATCH 兜底（顺序固定，不可删） */
export function isSystemRule(rule: { type?: string; payload?: string } | null | undefined): boolean {
	if (!rule) return false;
	const typ = String(rule.type || '').trim();
	const payload = String(rule.payload || '').trim();
	if (typ === RuleType.MATCH || typ === 'MATCH') return true;
	if ((typ === RuleType.GEOIP || typ === 'GEOIP') && payload.toUpperCase() === 'CN') return true;
	if ((typ === RuleType.GEOSITE || typ === 'GEOSITE') && payload === 'category-ads-all') return true;
	return false;
}

export function isMatchType(type: string): boolean {
	return type === RuleType.MATCH || type === 'MATCH';
}

export function payloadLabel(type: string): string {
	switch (type) {
		case RuleType.DOMAIN:
			return '完整域名';
		case RuleType.DOMAIN_SUFFIX:
			return '域名后缀';
		case RuleType.DOMAIN_KEYWORD:
			return '域名关键词';
		case RuleType.GEOSITE:
			return 'GeoSite 分类';
		case RuleType.GEOIP:
			return '国家/地区代码';
		case RuleType.IP_CIDR:
			return 'IPv4 网段';
		case RuleType.IP_CIDR6:
			return 'IPv6 网段';
		case RuleType.MATCH:
			return '匹配内容';
		default:
			return '匹配内容';
	}
}

export function payloadTip(type: string): string {
	switch (type) {
		case RuleType.DOMAIN:
			return '填写完整域名（不含协议）。\n例如：www.google.com';
		case RuleType.DOMAIN_SUFFIX:
			return '填写域名后缀。\n例如：google.com、openai.com';
		case RuleType.DOMAIN_KEYWORD:
			return '填写域名中包含的关键词。\n例如：google、openai';
		case RuleType.GEOSITE:
			return 'Clash Meta 域名分类（客户端 geosite.dat）。\n广告拦截常用：category-ads-all\n其它如：youtube、google、cn';
		case RuleType.GEOIP:
			return '填写 GeoIP 国家代码。\n例如：CN（国内）、US、JP。\n国内直连常用 CN。';
		case RuleType.IP_CIDR:
			return '填写 IPv4 CIDR。\n例如：10.0.0.0/8、1.1.1.1/32';
		case RuleType.IP_CIDR6:
			return '填写 IPv6 CIDR。\n例如：2001:db8::/32';
		case RuleType.MATCH:
			return '最终匹配无需填写匹配内容，会兜底所有剩余流量。';
		default:
			return '按规则类型填写对应匹配值。';
	}
}

export function payloadPlaceholder(type: string): string {
	switch (type) {
		case RuleType.DOMAIN:
			return 'www.example.com';
		case RuleType.DOMAIN_SUFFIX:
			return 'openai.com';
		case RuleType.DOMAIN_KEYWORD:
			return 'google';
		case RuleType.GEOSITE:
			return 'category-ads-all';
		case RuleType.GEOIP:
			return 'CN';
		case RuleType.IP_CIDR:
			return '10.0.0.0/8';
		case RuleType.IP_CIDR6:
			return '2001:db8::/32';
		default:
			return '';
	}
}

/** 全局匹配顺序（与 Clash 一致） */
export function sortRules(rules: Rule[]): Rule[] {
	return [...rules].sort((a, b) => a.sortOrder - b.sortOrder || a.id - b.id);
}

/** 面板分组：按业务分类聚合，组内仍保持全局顺序；空分类（刚新建）也会显示 */
export function buildCategorySections(
	rules: Rule[],
	extraCategories: string[] = [],
): CategorySection[] {
	const sorted = sortRules(rules);
	const buckets = new Map<string, Rule[]>();
	for (const r of sorted) {
		const key = (r.category || '').trim() || '';
		const list = buckets.get(key) || [];
		list.push(r);
		buckets.set(key, list);
	}
	const sections: CategorySection[] = [];
	const seen = new Set<string>();
	for (const cat of RULE_CATEGORY_ORDER) {
		const list = buckets.get(cat) || [];
		const isExtra = extraCategories.includes(cat);
		if (list.length || isExtra) {
			sections.push({ key: cat, label: cat, rules: list });
			buckets.delete(cat);
			seen.add(cat);
		}
	}
	// 未分类
	const uncategorized = buckets.get('');
	if (uncategorized?.length) {
		sections.push({ key: '', label: '未分类', rules: uncategorized });
		buckets.delete('');
	}
	// 自定义分类（规则里已有 + 用户新建的空分类）
	const customKeys = new Set<string>([
		...[...buckets.keys()].filter((k) => k !== ''),
		...extraCategories.filter((c) => !seen.has(c)),
	]);
	for (const key of [...customKeys].sort((a, b) => a.localeCompare(b, 'zh'))) {
		const list = buckets.get(key) || [];
		sections.push({ key, label: key, rules: list });
		buckets.delete(key);
	}
	return sections;
}

/**
 * 面板分组：按出口（策略组）聚合，组内保持全局匹配顺序。
 * groupNames：已有策略组名（用于固定顺序与空组不展示——仅展示有规则的出口；
 * 失效出口也会单独成组排在最后）。
 */
export function buildTargetSections(rules: Rule[], groupNames: string[] = []): CategorySection[] {
	const sorted = sortRules(rules);
	const buckets = new Map<string, Rule[]>();
	for (const r of sorted) {
		const key = (r.target || '').trim() || '';
		const list = buckets.get(key) || [];
		list.push(r);
		buckets.set(key, list);
	}
	const sections: CategorySection[] = [];
	const seen = new Set<string>();
	for (const name of groupNames) {
		const n = (name || '').trim();
		if (!n) continue;
		const list = buckets.get(n);
		if (!list?.length) continue;
		sections.push({ key: n, label: n, rules: list });
		buckets.delete(n);
		seen.add(n);
	}
	// 未指定出口（理论上不应出现）
	const empty = buckets.get('');
	if (empty?.length) {
		sections.push({ key: '', label: '未指定出口', rules: empty });
		buckets.delete('');
	}
	// 失效出口 / 不在策略组列表中的
	for (const key of [...buckets.keys()].sort((a, b) => a.localeCompare(b, 'zh'))) {
		const list = buckets.get(key) || [];
		if (!list.length) continue;
		sections.push({ key, label: key || '未指定出口', rules: list });
	}
	return sections;
}

/**
	 * 分类下拉：预设 + 规则中出现过的 + extra 空分类 + 可选「新建」。
	 * 不填 / 空串 = 未分类，故不下发「未分类」选项。
	 * current：编辑时若当前值不在列表，仍保留。
	 */
	export function buildCategoryOptions(
		rules: Rule[],
		extraCategories: string[] = [],
		opts?: { current?: string; allowNew?: boolean; newValue?: string },
	): CategoryOption[] {
		const allowNew = opts?.allowNew !== false;
		const newValue = opts?.newValue ?? CATEGORY_NEW_VALUE;
		const seen = new Set<string>();
		const out: CategoryOption[] = [];
		const push = (value: string, text: string) => {
			if (!value || seen.has(value)) return;
			seen.add(value);
			out.push({ value, text });
		};
		for (const c of RULE_CATEGORY_ORDER) push(c, c);
		for (const r of rules) {
			const c = (r.category || '').trim();
			if (c) push(c, c);
		}
		for (const c of extraCategories) {
			if (c) push(c, c);
		}
		const cur = (opts?.current || '').trim();
		if (cur && cur !== newValue && !seen.has(cur)) {
			push(cur, cur);
		}
		if (allowNew) {
			out.push({ value: newValue, text: '＋ 新建分类…' });
		}
		return out;
	}

export function resolveSelectedCategory(
	selected: string,
	custom: string,
	newValue: string = CATEGORY_NEW_VALUE,
): string {
	if (selected === newValue) return custom.trim();
	return (selected || '').trim();
}

/** 若分类是新名字且不在预设/已有规则中，返回应记入 extra 的名字；否则 null */
export function categoryToRemember(
	name: string,
	rules: Rule[],
	extraCategories: string[],
): string | null {
	const n = name.trim();
	if (!n) return null;
	if (RULE_CATEGORY_ORDER.includes(n as (typeof RULE_CATEGORY_ORDER)[number])) return null;
	if (extraCategories.includes(n)) return null;
	if (rules.some((r) => (r.category || '').trim() === n)) return null;
	return n;
}

export function defaultRuleTarget(groupNames: string[]): string {
	if (groupNames.includes('直连')) return '直连';
	if (groupNames.includes('PROXY')) return 'PROXY';
	if (groupNames.length > 0) return groupNames[0];
	return '';
}

export function orphanTargetValue(target: string, groupNames: string[]): string | null {
	const v = (target || '').trim();
	if (!v) return null;
	if (groupNames.includes(v)) return null;
	return v;
}

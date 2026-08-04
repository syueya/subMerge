import type { RuleType } from '../enums/rule-type';

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

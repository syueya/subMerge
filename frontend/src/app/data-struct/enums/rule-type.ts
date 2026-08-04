import type { EnumOption } from './common';

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

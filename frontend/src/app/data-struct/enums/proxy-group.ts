import type { EnumOption } from './common';

export type ProxyGroupType = 'select' | 'url-test' | 'fallback' | 'load-balance';

export const ProxyGroupType = {
	Select: 'select',
	UrlTest: 'url-test',
	Fallback: 'fallback',
	LoadBalance: 'load-balance',
} as const;

export const PROXY_GROUP_TYPE_OPTIONS: ReadonlyArray<EnumOption<ProxyGroupType>> = [
	{ value: 'select', text: '手动选择' },
	{ value: 'url-test', text: '自动测速' },
	{ value: 'fallback', text: '故障转移' },
	{ value: 'load-balance', text: '负载均衡' },
];

import type { EnumOption } from './common';
import { BADGE_ERR, BADGE_OK, BADGE_WARN } from './common';

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

/** 令牌策略组投影 */
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

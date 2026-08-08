import type { EnumOption } from './common';
import {
	TOKEN_STATUS_BADGE,
	TOKEN_STATUS_OPTIONS,
	TokenStatus,
	type TokenStatus as TokenStatusType,
} from './token';

export type APIKeyStatus = TokenStatusType;

export const APIKeyStatus = TokenStatus;

export const API_KEY_STATUS_OPTIONS = TOKEN_STATUS_OPTIONS;

export const API_KEY_STATUS_BADGE = TOKEN_STATUS_BADGE;

/** 粗粒度权限 */
export type APIKeyScope = 'read' | 'write' | 'publish' | '*';

export const APIKeyScope = {
	Read: 'read',
	Write: 'write',
	Publish: 'publish',
	All: '*',
} as const;

/** 权限 chips：全部优先；读取 / 写入 / 发布可组合（* 覆盖三者） */
export const API_KEY_SCOPE_OPTIONS: ReadonlyArray<EnumOption<APIKeyScope>> = [
	{ value: '*', text: '全部' },
	{ value: 'read', text: '读取' },
	{ value: 'write', text: '写入' },
	{ value: 'publish', text: '发布' },
];

/** 各 scope 的悬浮说明（创建弹窗 chips tooltip） */
export const API_KEY_SCOPE_HINTS: Record<APIKeyScope, string> = {
	'*': '业务 API 全部权限（读取 + 写入 + 发布）',
	read: '查询源、规则、发布记录、Geo、日志等',
	write: '改配置（源/规则/组/令牌等），不含发布与回滚',
	publish: '仅发布与回滚',
};

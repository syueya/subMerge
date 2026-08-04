import type { EnumOption } from './common';
import { BADGE_ERR, BADGE_MUTED, BADGE_OK, BADGE_WARN } from './common';

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

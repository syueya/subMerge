import type { EnumOption } from './common';
import { BADGE_MUTED, BADGE_OK, BADGE_WARN } from './common';

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
	draft: BADGE_WARN,
	published: BADGE_OK,
	rolled_back: BADGE_MUTED,
};

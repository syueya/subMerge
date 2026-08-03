import type { SubscriptionSource } from './types';

/** 拉取成功结果的展示侧字段（list / form 摘要文案共用） */
export interface RefreshResultLike {
	source: SubscriptionSource;
	upstreamTotal?: number;
	parsed?: number;
	added: number;
	skipped: number;
	parseDropped?: Record<string, number>;
	filterDropped?: Record<string, number>;
	filteredNames?: string[];
	filteredNamesOmitted?: number;
	regionCounts?: Record<string, number>;
}

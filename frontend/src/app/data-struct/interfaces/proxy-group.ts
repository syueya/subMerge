import type { ProxyGroupType } from '../enums/proxy-group';

export interface ProxyGroup {
	id: number;
	name: string;
	type: ProxyGroupType | string;
	proxies: string[];
	url?: string;
	interval?: number;
	enabled: boolean;
	sortOrder: number;
}

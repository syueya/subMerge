import { SubscriptionSource } from '../../common/types';

export type RefreshResultLike = {
	source: SubscriptionSource;
	upstreamTotal?: number;
	parsed?: number;
	added: number;
	skipped: number;
	parseDropped?: Record<string, number>;
	filterDropped?: Record<string, number>;
	filteredNames?: string[];
	regionCounts?: Record<string, number>;
};

function formatDropMap(
	m: Record<string, number> | undefined,
	labels: Record<string, string>,
): string {
	if (!m) return '';
	return Object.entries(m)
		.filter(([, n]) => n > 0)
		.sort((a, b) => b[1] - a[1])
		.map(([k, n]) => {
			const label = labels[k] || (k.startsWith('server blocked') ? 'server黑名单' : k);
			return `${label}:${n}`;
		})
		.join(' ');
}

/** 拉取成功后的摘要文案（与原先 source-list 一致） */
export function formatRefreshMsg(res: RefreshResultLike, title: string): string {
	const regions = res.regionCounts
		? Object.entries(res.regionCounts)
				.sort((a, b) => a[0].localeCompare(b[0]))
				.map(([k, v]) => `${k}:${v}`)
				.join(' ')
		: '';
	const up = res.upstreamTotal ?? res.added + (res.skipped ?? 0);
	const parsed = res.parsed ?? up;
	const lines = [
		`${res.source.name} ${title}`,
		`上游 ${up} → 解析 ${parsed} → 入库 ${res.added}（过滤 ${res.skipped ?? 0}）`,
	];
	const parseDrop = formatDropMap(res.parseDropped, {
		missing_name: '缺名称',
		missing_type: '缺type',
		missing_server: '缺server',
		invalid_port: '端口无效',
		duplicate_name: '重名',
		not_object: '格式异常',
	});
	if (parseDrop) lines.push(`解析丢弃 ${parseDrop}`);
	const filterDrop = formatDropMap(res.filterDropped, {
		'name excluded': '名称排除',
		'name not in include list': '不在白名单',
		info_node: '信息节点',
	});
	if (filterDrop) lines.push(`过滤 ${filterDrop}`);
	const names = res.filteredNames || [];
	if (names.length) {
		// 完整列出（与后端日志一致）；名称本身含「过滤掉N条」时是机场信息节点，不是本系统又滤了 N 条
		lines.push(`过滤明细（共 ${names.length} 条，完整）：`);
		for (const n of names) {
			lines.push(`· ${n}`);
		}
	}
	if (regions) lines.push(`地区 ${regions}`);
	return lines.join('\n');
}

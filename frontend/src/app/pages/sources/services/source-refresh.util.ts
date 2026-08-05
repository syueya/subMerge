import type { RefreshResultLike } from '@data-struct';

export type { RefreshResultLike };

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
	const hasDiffStats = res.previous !== undefined || res.kept !== undefined || res.modified !== undefined;
	const lines = [`${res.source.name} ${title}`];
	if (hasDiffStats) {
		const previous = res.previous ?? 0;
		const kept = res.kept ?? 0;
		const modified = res.modified ?? 0;
		const current = res.added + kept + modified;
		lines.push(
			`上游 ${up} → 解析 ${parsed} → 当前 ${current}（保留 ${kept}，新增 ${res.added}，移除 ${res.removed ?? 0}，修改 ${modified}）`,
		);
		lines.push(`刷新前 ${previous}，过滤 ${res.skipped ?? 0}`);
	} else {
		lines.push(`上游 ${up} → 解析 ${parsed} → 入库 ${res.added}（过滤 ${res.skipped ?? 0}）`);
	}
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
	const omitted = res.filteredNamesOmitted ?? 0;
	if (names.length) {
		const suffix = omitted > 0 ? `，另有 ${omitted} 条未显示` : '';
		lines.push(`过滤明细（显示 ${names.length} 条${suffix}）：`);
		for (const n of names) {
			lines.push(`· ${n}`);
		}
	}
	if (regions) lines.push(`地区 ${regions}`);
	const conflicts = res.regionConflictTotal ?? 0;
	if (conflicts > 0) {
		lines.push(`地区标记冲突 ${conflicts} 条（已按名称关键词归类）`);
		for (const item of (res.regionConflicts || []).slice(0, 5)) {
			lines.push(`· ${item.name}：国旗 ${item.flagRegion}，名称 ${item.keywordRegion} → ${item.resolvedRegion}`);
		}
		if ((res.regionConflictOmitted ?? 0) > 0) {
			lines.push(`· 另有 ${res.regionConflictOmitted} 条冲突未显示`);
		}
	}
	return lines.join('\n');
}

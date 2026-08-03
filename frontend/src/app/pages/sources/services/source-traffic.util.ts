import { SubscriptionSource } from '@data-struct';

export function formatBytes(n: number): string {
	if (!Number.isFinite(n) || n < 0) n = 0;
	const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
	let v = n;
	let i = 0;
	while (v >= 1024 && i < units.length - 1) {
		v /= 1024;
		i++;
	}
	const digits = i === 0 ? 0 : v >= 100 ? 0 : v >= 10 ? 1 : 2;
	return `${v.toFixed(digits)} ${units[i]}`;
}

/** 是否有上游 Subscription-Userinfo */
export function hasTraffic(item: SubscriptionSource): boolean {
	return (
		(item.trafficUpload ?? 0) > 0 ||
		(item.trafficDownload ?? 0) > 0 ||
		(item.trafficTotal ?? 0) > 0 ||
		(item.trafficExpire ?? 0) > 0
	);
}

export function trafficUsed(item: SubscriptionSource): number {
	return Math.max(0, item.trafficUpload ?? 0) + Math.max(0, item.trafficDownload ?? 0);
}

/** 已用/总量，如 12.3 GB / 100 GB；无总量时只显示已用 */
export function trafficText(item: SubscriptionSource): string {
	if (!hasTraffic(item)) return '';
	const used = trafficUsed(item);
	const total = item.trafficTotal ?? 0;
	if (total > 0) {
		return `${formatBytes(used)} / ${formatBytes(total)}`;
	}
	if (used > 0) {
		return `已用 ${formatBytes(used)}`;
	}
	return '';
}

export function trafficPercent(item: SubscriptionSource): number | null {
	const total = item.trafficTotal ?? 0;
	if (total <= 0) return null;
	return Math.min(100, Math.round((trafficUsed(item) / total) * 100));
}

export function trafficExpireText(item: SubscriptionSource): string {
	const exp = item.trafficExpire ?? 0;
	if (exp <= 0) return '';
	const d = new Date(exp * 1000);
	if (Number.isNaN(d.getTime())) return '';
	const y = d.getFullYear();
	const m = String(d.getMonth() + 1).padStart(2, '0');
	const day = String(d.getDate()).padStart(2, '0');
	const now = Date.now();
	if (d.getTime() < now) {
		return `已过期 ${y}-${m}-${day}`;
	}
	return `到期 ${y}-${m}-${day}`;
}

export function trafficTitle(item: SubscriptionSource): string {
	if (!hasTraffic(item)) return '上游未返回 Subscription-Userinfo';
	const parts = [
		`上传 ${formatBytes(item.trafficUpload ?? 0)}`,
		`下载 ${formatBytes(item.trafficDownload ?? 0)}`,
		`总量 ${item.trafficTotal ? formatBytes(item.trafficTotal) : '未知'}`,
	];
	const exp = trafficExpireText(item);
	if (exp) parts.push(exp);
	parts.push('（来自上游响应头，拉取时更新）');
	return parts.join('\n');
}

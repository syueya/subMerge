// 地区目录只维护 backend/defaults/regions.yaml（经 GET /regions）。
// 前端不再内置地区列表；展示名由 API 的 name 提供。

export type Region = string;

/** 自动识别失败时的回退地区（与后端 FallbackRegion 一致；节点前缀 UNK-xxx） */
export const FALLBACK_REGION = 'UNK';

/** 地区展示名：优先用目录 labels；无则回退地区码本身 */
export function regionLabel(code: string, labels?: Record<string, string>): string {
	const c = String(code || '').toUpperCase();
	if (!c) return '';
	const name = labels?.[c]?.trim();
	return name || c;
}

/** 下拉/列表文案：美国 (US)；无中文名时仅码 */
export function regionOptionText(code: string, name?: string): string {
	const c = String(code || '').toUpperCase();
	const n = (name || '').trim();
	if (!c) return n;
	if (!n || n === c) return c;
	return `${n} (${c})`;
}

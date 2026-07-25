import type { DraftStatus } from './types';

/**
 * 统一的草稿/发布状态提示文案。规则页、策略组页、发布页共用，避免文案漂移。
 * buildError 优先（草稿无法生成）；其次未发布过、有未发布更改、已一致。
 */
export function draftStatusNote(s: DraftStatus): string {
	if (s.buildError) {
		return `草稿暂无法生成：${localizeBuildError(s.buildError)}`;
	}
	if (!s.hasPublished) {
		return '尚未发布过配置，订阅链接在发布后才会有内容';
	}
	if (s.dirty) {
		return `有未发布更改（当前生效 v${s.publishedVersion || '?'}）`;
	}
	return `已与 v${s.publishedVersion || '?'} 一致`;
}

/** 共享时间格式化：ISO 字符串 → "YYYY-MM-DD HH:mm"；空值返回占位符 */
export function formatDateTime(iso?: string | null): string {
	if (!iso) return '—';
	const d = new Date(iso);
	if (Number.isNaN(d.getTime())) return String(iso);
	const pad = (n: number) => String(n).padStart(2, '0');
	return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

/**
 * 将后端草稿/发布错误映射为中文（兼容旧英文错误串）。
 * 已是中文的原文尽量原样返回。
 */
export function localizeBuildError(err?: string | null): string {
	const raw = String(err || '').trim();
	if (!raw) return '';
	const e = raw.toLowerCase();
	if (
		e.includes('no proxies') ||
		e.includes('proxies available') ||
		e.includes('refresh sources') ||
		raw.includes('暂无可用节点')
	) {
		return '暂无可用节点，请先添加并刷新订阅源';
	}
	if (e.includes('no usable proxy groups') || raw.includes('没有可用策略组')) {
		return '没有可用策略组，请添加节点或检查 REGION/SOURCE 成员引用';
	}
	if (e.includes('no enabled rules') || raw.includes('没有启用的分流规则')) {
		return '没有启用的分流规则';
	}
	if (
		e.includes('match rule is required') ||
		e.includes('match rule must be the last') ||
		raw.includes('MATCH 兜底') ||
		raw.includes('必须有 MATCH')
	) {
		return '必须有 MATCH 兜底规则，且放在最后';
	}
	if (e.includes('rule target') || raw.includes('规则出口')) {
		const m =
			raw.match(/rule target "([^"]+)"/i) ||
			raw.match(/规则出口「([^」]+)」/);
		if (m?.[1]) return `规则出口「${m[1]}」不存在`;
		return '规则出口不存在';
	}
	// 已是中文或未知：原样展示
	return raw;
}

/** 概览等处的短标签 */
export function shortBuildError(err?: string | null): string {
	const full = localizeBuildError(err);
	if (!full) return '草稿异常';
	if (full.includes('暂无可用节点') || full.includes('无可用节点')) return '无可用节点';
	if (full.includes('策略组')) return '策略组不可用';
	if (full.includes('MATCH')) return '缺少 MATCH 规则';
	if (full.includes('规则')) return '规则不完整';
	if (full.length <= 12) return full;
	return '草稿异常';
}

import { Injectable, computed, inject, signal } from '@angular/core';
import { DraftChange, DraftStatus } from '@data-struct';
import { ReleaseService } from './release.service';

/**
 * 草稿/发布状态共享 store。
 *
 * 发布页、规则页、策略组页、概览页都要展示「有未发布更改 / 改了什么」。
 * 以前每个组件各自调用 draft-status 并维护 draftDirty/draftNote 信号，
 * 逻辑重复且发布后无法互相同步。这里集中一份状态：
 * - 组件读派生信号 dirty()/note()/changes()/summary()
 * - 任何写操作（增删改规则/组、发布、回滚）后调用 refresh() 让所有页面同步
 */
@Injectable({ providedIn: 'root' })
export class DraftStatusStore {
	private readonly releaseSvc = inject(ReleaseService);

	private readonly state = signal<DraftStatus | null>(null);
	private readonly loading = signal(false);

	/** 是否有未发布更改 */
	readonly dirty = computed(() => !!this.state()?.dirty);
	/** 统一的中文状态文案（错误由后端直接返回中文） */
	readonly note = computed(() => {
		const s = this.state();
		return s ? draftStatusNote(s) : '';
	});
	/** 实体级变更列表 */
	readonly changes = computed<DraftChange[]>(() => this.state()?.changes || []);
	/** 原始状态（需要 buildError / publishedVersion 等细节时用） */
	readonly status = computed<DraftStatus | null>(() => this.state());

	/** 按类别汇总，如「节点 +5/-2、策略组 改1、规则 +3」 */
	readonly summary = computed<string>(() => summarizeChanges(this.changes()));

	/** 拉取最新草稿状态；失败时清空（视为无变更，避免误报） */
	refresh(): void {
		if (this.loading()) return;
		this.loading.set(true);
		this.releaseSvc.draftStatus().subscribe({
			next: (s) => {
				this.state.set(s);
				this.loading.set(false);
			},
			error: () => {
				this.state.set(null);
				this.loading.set(false);
			},
		});
	}
}

/**
 * 统一的草稿/发布状态提示文案。规则页、策略组页、发布页共用，避免文案漂移。
 * buildError 优先（草稿无法生成）；其次未发布过、有未发布更改、已一致。
 */
function draftStatusNote(s: DraftStatus): string {
	if (s.buildError) {
		return `草稿暂无法生成：${s.buildError}`;
	}
	if (!s.hasPublished) {
		return '尚未发布过配置，订阅链接在发布后才会有内容';
	}
	if (s.dirty) {
		return `有未发布更改（当前生效 v${s.publishedVersion || '?'}）`;
	}
	return `已与 v${s.publishedVersion || '?'} 一致`;
}

/** 把变更列表压成一行中文汇总；无变更返回空串。 */
export function summarizeChanges(changes: DraftChange[]): string {
	const stat: Record<string, { added: number; removed: number; modified: number }> = {
		proxy: { added: 0, removed: 0, modified: 0 },
		group: { added: 0, removed: 0, modified: 0 },
		rule: { added: 0, removed: 0, modified: 0 },
	};
	for (const c of changes) {
		const bucket = stat[c.kind];
		if (!bucket) continue;
		if (c.action === 'added') bucket.added++;
		else if (c.action === 'removed') bucket.removed++;
		else if (c.action === 'modified') bucket.modified++;
	}
	const labels: Record<string, string> = { proxy: '节点', group: '策略组', rule: '规则' };
	const parts: string[] = [];
	for (const kind of ['proxy', 'group', 'rule']) {
		const b = stat[kind];
		const seg: string[] = [];
		if (b.added) seg.push(`+${b.added}`);
		if (b.removed) seg.push(`-${b.removed}`);
		if (b.modified) seg.push(`改${b.modified}`);
		if (seg.length) parts.push(`${labels[kind]} ${seg.join('/')}`);
	}
	return parts.join('、');
}

import { Component, OnInit, WritableSignal, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import {
	TOKEN_GROUP_MODE_OPTIONS,
	TOKEN_STATUS_BADGE,
	TOKEN_STATUS_OPTIONS,
	ProxyGroup,
	ShareToken,
	SubscriptionSource,
	TokenGroupMode,
	TokenStatus,
	enumBadgeClass,
	enumText,
} from '../../common/types';
import { DialogService } from '../../common/dialog/dialog.service';
import { formatDateTime } from '../../common/format';
import { RuleService } from '../rules/rule.service';
import { SourceService } from '../sources/source.service';
import { TokenService } from './token.service';

@Component({
	selector: 'app-token-list',
	standalone: true,
	imports: [FormsModule],
	templateUrl: './token-list.component.html',
})
export class TokenListComponent implements OnInit {
	private readonly svc = inject(TokenService);
	private readonly sources = inject(SourceService);
	private readonly rules = inject(RuleService);
	private readonly dialog = inject(DialogService);

	readonly groupModeOptions = TOKEN_GROUP_MODE_OPTIONS;

	items = signal<ShareToken[]>([]);
	sourceList = signal<SubscriptionSource[]>([]);
	groupList = signal<ProxyGroup[]>([]);

	name = '';
	/** 创建时勾选的源；空且 createAllSources=true 表示全部源 */
	createSourceIds = signal<Set<number>>(new Set());
	/** 创建时是否「全部源」模式（动态包含之后新增的源） */
	createAllSources = signal(true);
	createGroupMode = signal<TokenGroupMode>('auto');
	createGroupNames = signal<Set<string>>(new Set());

	/** 编辑某条令牌的允许源 / 策略组投影 */
	editingId = signal<number | null>(null);
	editAllSources = signal(true);
	editSourceIds = signal<Set<number>>(new Set());
	editGroupMode = signal<TokenGroupMode>('auto');
	editGroupNames = signal<Set<string>>(new Set());

	ngOnInit(): void {
		this.reload();
		this.reloadSources();
		this.reloadGroups();
	}

	statusText(v: string): string {
		return enumText(TOKEN_STATUS_OPTIONS, v);
	}

	statusClass(v: string): string {
		return enumBadgeClass(TOKEN_STATUS_BADGE, v);
	}

	groupModeText(v?: string): string {
		return enumText(TOKEN_GROUP_MODE_OPTIONS, v || 'auto');
	}

	formatTime = formatDateTime;

	/** 列表展示：路径保留，token 中间用 **，仅保留后 4 位 */
	maskSubscribeUrl(url: string): string {
		const i = url.lastIndexOf('/');
		if (i < 0) {
			if (url.length <= 4) return url;
			return '**' + url.slice(-4);
		}
		const base = url.slice(0, i + 1);
		const token = url.slice(i + 1);
		if (token.length <= 4) return url;
		return base + '**' + token.slice(-4);
	}

	/** 列表展示：全部源 / 源名列表（多行） */
	sourceScopeLines(item: ShareToken): string[] {
		const ids = item.sourceIds || [];
		if (ids.length === 0) return ['全部源'];
		return item.sourceNames?.length ? item.sourceNames : ids.map((id) => `#${id}`);
	}

	sourceScopeTitle(item: ShareToken): string {
		const ids = item.sourceIds || [];
		if (ids.length === 0) return '包含全部启用订阅源';
		const names = item.sourceNames?.length ? item.sourceNames : ids.map((id) => `源#${id}`);
		return names.join('\n');
	}

	/** 列表：策略组模式摘要 */
	groupScopeText(item: ShareToken): string {
		const mode = item.groupMode || 'auto';
		if (mode === 'custom') {
			const names = item.groupNames || [];
			if (names.length === 0) return '自定义';
			if (names.length <= 2) return `自定义 · ${names.join('、')}`;
			return `自定义 · ${names.length} 个`;
		}
		return this.groupModeText(mode);
	}

	groupScopeTitle(item: ShareToken): string {
		const mode = item.groupMode || 'auto';
		if (mode === 'auto') return '自动：按该链接节点去掉空策略组；规则目标缺失优先回退「节点选择」';
		if (mode === 'all') return '全部：保留模板中的策略组（空组占位 DIRECT）';
		const names = item.groupNames || [];
		return names.length ? names.join('\n') : '自定义（未选组）';
	}

	reload(): void {
		this.svc.list().subscribe({
			next: (r) => this.items.set(r.items || []),
			error: (e: Error) => void this.dialog.error(e.message),
		});
	}

	reloadSources(): void {
		this.sources.list().subscribe({
			next: (r) => {
				const list = r.items || [];
				this.sourceList.set(list);
				// 创建区默认「全部源」：列表加载后勾选全部，便于一眼看到范围
				if (this.createAllSources()) {
					this.createSourceIds.set(new Set(list.map((s) => s.id)));
				}
			},
			error: () => {
				/* 源列表失败不阻断令牌页 */
			},
		});
	}

	reloadGroups(): void {
		this.rules.listGroups().subscribe({
			next: (r) => this.groupList.set(r.items || []),
			error: () => {
				/* 策略组失败不阻断 */
			},
		});
	}

	private allSourceIdSet(): Set<number> {
		return new Set(this.sourceList().map((s) => s.id));
	}

	// --- 源勾选：create/edit 共用的底层逻辑（参数化 signal，避免两套复制） ---

	/** 切换「全部源」：勾全部；取消时若当前空则回退全选，便于微调 */
	private setAllSources(allFlag: WritableSignal<boolean>, ids: WritableSignal<Set<number>>, all: boolean): void {
		allFlag.set(all);
		if (all || ids().size === 0) {
			ids.set(this.allSourceIdSet());
		}
	}

	/** 勾选单个源；勾满全部时自动回到「全部源」状态 */
	private toggleSource(
		allFlag: WritableSignal<boolean>,
		ids: WritableSignal<Set<number>>,
		id: number,
		checked: boolean,
	): void {
		const next = new Set(ids());
		if (checked) next.add(id);
		else next.delete(id);
		ids.set(next);

		const allIds = this.sourceList().map((s) => s.id);
		allFlag.set(allIds.length > 0 && next.size === allIds.length && allIds.every((sid) => next.has(sid)));
	}

	/** 全部源模式下一律显示为勾选 */
	private isSourceChecked(allFlag: WritableSignal<boolean>, ids: WritableSignal<Set<number>>, id: number): boolean {
		return allFlag() || ids().has(id);
	}

	// --- 策略组勾选：create/edit 共用 ---

	/** 切到 custom 且尚未选组时，默认勾选启用中的策略组 */
	private applyGroupMode(names: WritableSignal<Set<string>>, mode: TokenGroupMode): void {
		if (mode === 'custom' && names().size === 0) {
			names.set(new Set(this.groupList().filter((g) => g.enabled).map((g) => g.name)));
		}
	}

	private toggleGroup(names: WritableSignal<Set<string>>, name: string, checked: boolean): void {
		const next = new Set(names());
		if (checked) next.add(name);
		else next.delete(name);
		names.set(next);
	}

	// --- create 视图：薄封装转发到共用逻辑 ---

	setCreateAll(all: boolean): void {
		this.setAllSources(this.createAllSources, this.createSourceIds, all);
	}

	toggleCreateSource(id: number, checked: boolean): void {
		this.toggleSource(this.createAllSources, this.createSourceIds, id, checked);
	}

	isCreateSourceChecked(id: number): boolean {
		return this.isSourceChecked(this.createAllSources, this.createSourceIds, id);
	}

	setCreateGroupMode(mode: TokenGroupMode): void {
		this.createGroupMode.set(mode);
		this.applyGroupMode(this.createGroupNames, mode);
	}

	toggleCreateGroup(name: string, checked: boolean): void {
		this.toggleGroup(this.createGroupNames, name, checked);
	}

	isCreateGroupChecked(name: string): boolean {
		return this.createGroupNames().has(name);
	}

	create(): void {
		if (!this.name.trim()) {
			void this.dialog.error('请填写朋友备注名');
			return;
		}
		const all = this.createAllSources();
		const ids = all ? [] : Array.from(this.createSourceIds());
		if (!all && ids.length === 0) {
			void this.dialog.error('请至少选择一个订阅源，或勾选「全部源」');
			return;
		}
		const groupMode = this.createGroupMode();
		const groupNames =
			groupMode === 'custom' ? Array.from(this.createGroupNames()) : [];
		if (groupMode === 'custom' && groupNames.length === 0) {
			void this.dialog.error('自定义模式请至少勾选一个策略组');
			return;
		}
		this.svc.create(this.name.trim(), ids, groupMode, groupNames).subscribe({
			next: (t) => {
				this.name = '';
				this.createAllSources.set(true);
				this.createSourceIds.set(this.allSourceIdSet());
				this.createGroupMode.set('auto');
				this.createGroupNames.set(new Set());
				const scope =
					(t.sourceIds || []).length === 0
						? '全部源'
						: `指定 ${(t.sourceIds || []).length} 个源`;
				const gscope = this.groupModeText(t.groupMode);
				void this.dialog.success(
					t.subscribeUrl
						? `令牌「${t.name}」已创建（${scope} · ${gscope}），可直接复制订阅链接`
						: `令牌「${t.name}」已创建（${scope} · ${gscope}）`,
				);
				this.reload();
			},
			error: (e: Error) => void this.dialog.error(e.message),
		});
	}

	openEditSources(item: ShareToken): void {
		const ids = item.sourceIds || [];
		this.editingId.set(item.id);
		if (ids.length === 0) {
			this.editAllSources.set(true);
			this.editSourceIds.set(this.allSourceIdSet());
		} else {
			this.editAllSources.set(false);
			this.editSourceIds.set(new Set(ids));
		}
		const mode = (item.groupMode || 'auto') as TokenGroupMode;
		this.editGroupMode.set(mode);
		this.editGroupNames.set(new Set(item.groupNames || []));
		if (mode === 'custom' && this.editGroupNames().size === 0) {
			this.editGroupNames.set(
				new Set(this.groupList().filter((g) => g.enabled).map((g) => g.name)),
			);
		}
	}

	closeEditSources(): void {
		this.editingId.set(null);
		this.editAllSources.set(true);
		this.editSourceIds.set(new Set());
		this.editGroupMode.set('auto');
		this.editGroupNames.set(new Set());
	}

	setEditAll(all: boolean): void {
		this.setAllSources(this.editAllSources, this.editSourceIds, all);
	}

	toggleEditSource(id: number, checked: boolean): void {
		this.toggleSource(this.editAllSources, this.editSourceIds, id, checked);
	}

	isEditSourceChecked(id: number): boolean {
		return this.isSourceChecked(this.editAllSources, this.editSourceIds, id);
	}

	setEditGroupMode(mode: TokenGroupMode): void {
		this.editGroupMode.set(mode);
		this.applyGroupMode(this.editGroupNames, mode);
	}

	toggleEditGroup(name: string, checked: boolean): void {
		this.toggleGroup(this.editGroupNames, name, checked);
	}

	isEditGroupChecked(name: string): boolean {
		return this.editGroupNames().has(name);
	}

	saveEditSources(): void {
		const id = this.editingId();
		if (id == null) return;
		const all = this.editAllSources();
		const ids = all ? [] : Array.from(this.editSourceIds());
		if (!all && ids.length === 0) {
			void this.dialog.error('请至少选择一个订阅源，或勾选「全部源」');
			return;
		}
		const groupMode = this.editGroupMode();
		const groupNames =
			groupMode === 'custom' ? Array.from(this.editGroupNames()) : [];
		if (groupMode === 'custom' && groupNames.length === 0) {
			void this.dialog.error('自定义模式请至少勾选一个策略组');
			return;
		}
		this.svc
			.update(id, {
				sourceIds: ids,
				groupMode,
				groupNames,
			})
			.subscribe({
				next: (t) => {
					const scope =
						(t.sourceIds || []).length === 0
							? '全部源'
							: `指定 ${(t.sourceIds || []).length} 个源`;
					void this.dialog.success(
						`已更新「${t.name}」（${scope} · ${this.groupModeText(t.groupMode)}）`,
					);
					this.closeEditSources();
					this.reload();
				},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}

	disable(item: ShareToken): void {
		this.svc.update(item.id, { status: TokenStatus.Disabled }).subscribe({
			next: () => {
				void this.dialog.success(`已禁用「${item.name}」`);
				this.reload();
			},
			error: (e: Error) => void this.dialog.error(e.message),
		});
	}

	enable(item: ShareToken): void {
		this.svc.update(item.id, { status: TokenStatus.Active }).subscribe({
			next: () => {
				void this.dialog.success(`已启用「${item.name}」`);
				this.reload();
			},
			error: (e: Error) => void this.dialog.error(e.message),
		});
	}

	async revoke(item: ShareToken): Promise<void> {
		const ok = await this.dialog.confirm(
			`作废「${item.name}」？\n旧链接立即失效，记录与访问次数保留，之后可重新生成。`,
			'作废确认',
			'作废',
		);
		if (!ok) return;
		this.svc.revoke(item.id).subscribe({
			next: () => {
				void this.dialog.success(`已作废「${item.name}」`);
				this.reload();
			},
			error: (e: Error) => void this.dialog.error(e.message),
		});
	}

	async regenerate(item: ShareToken): Promise<void> {
		const ok = await this.dialog.confirm(
			`重新生成「${item.name}」的 token？\n旧链接将立即失效，请把新链接发给对方。`,
			'重新生成',
			'重新生成',
		);
		if (!ok) return;
		this.svc.regenerate(item.id).subscribe({
			next: (t) => {
				void this.dialog.success(
					t.subscribeUrl ? `已重新生成，请复制新链接` : `已重新生成「${item.name}」`,
				);
				this.reload();
			},
			error: (e: Error) => void this.dialog.error(e.message),
		});
	}

	async remove(item: ShareToken): Promise<void> {
		const ok = await this.dialog.confirm(
			`永久删除「${item.name}」？\n记录不可恢复（与「作废」不同）。`,
			'删除确认',
			'删除',
		);
		if (!ok) return;
		this.svc.delete(item.id).subscribe({
			next: () => {
				void this.dialog.success(`已删除「${item.name}」`);
				this.reload();
			},
			error: (e: Error) => void this.dialog.error(e.message),
		});
	}

	async copy(text: string): Promise<void> {
		try {
			await this.writeClipboard(text);
			void this.dialog.success('已复制到剪贴板');
		} catch {
			void this.dialog.error('复制失败，请手动选择文本');
		}
	}

	/** Clipboard API 仅在 HTTPS / localhost 可用；HTTP 局域网访问时回退到 execCommand。 */
	private async writeClipboard(text: string): Promise<void> {
		if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText && window.isSecureContext) {
			await navigator.clipboard.writeText(text);
			return;
		}
		const ta = document.createElement('textarea');
		ta.value = text;
		ta.setAttribute('readonly', '');
		ta.style.position = 'fixed';
		ta.style.left = '-9999px';
		ta.style.top = '0';
		document.body.appendChild(ta);
		ta.select();
		ta.setSelectionRange(0, text.length);
		try {
			if (!document.execCommand('copy')) {
				throw new Error('execCommand copy failed');
			}
		} finally {
			document.body.removeChild(ta);
		}
	}
}

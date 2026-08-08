import { AfterViewInit, Component, inject, signal } from '@angular/core';
import { MatTableDataSource } from '@angular/material/table';
		import { CM_DIALOG_WIDTH, CmDialogOpenService } from '@common/modules/dialog';
		import { CmParentTableComponent } from '@common/parents/parent-table/parent-table.component';
		import { DialogService } from '@common/services/dialog.service';
		import { copyToClipboard, formatDateTime, MSG_COPIED, MSG_COPY_FAILED, msgDisabled, msgEnabled, msgRevoked, msgDeleted, TITLE_CONFIRM_DELETE, TITLE_CONFIRM_REVOKE } from '@common/util';
import {
		BADGE_MUTED,
		BADGE_WARN,
		ProxyGroup,
		ShareToken,
		SubscriptionSource,
		TOKEN_STATUS_BADGE,
		TOKEN_STATUS_OPTIONS,
TokenFormDialogData,
			TokenStatus,
			enumBadgeClass,
			enumText,
		} from '@data-struct';
		import { finalize, takeUntil } from 'rxjs';

		import { RuleService } from '../../rules/services/rule.service';
		import { SourceService } from '../../sources/services/source.service';
		import { TokenService } from '../services/token.service';
		import { TokenFormComponent } from '../token-form/token-form.component';

@Component({
	selector: 'app-token-list',
	templateUrl: './token-list.component.html',
	standalone: false,
})
export class TokenListComponent extends CmParentTableComponent implements AfterViewInit {
	private svc = inject(TokenService);
	private sources = inject(SourceService);
	private rules = inject(RuleService);
	private dialog = inject(DialogService);
	private dialogOpen = inject(CmDialogOpenService);

	dataSource = new MatTableDataSource<ShareToken>([]);
override displayedColumns: string[] = [
			'name',
			'sources',
			'groups',
			'url',
			'access',
			'action',
		];

	busy = signal(false);
	sourceList = signal<SubscriptionSource[]>([]);
	groupList = signal<ProxyGroup[]>([]);

	formatTime = formatDateTime;
	badgeMuted = BADGE_MUTED;
	badgeWarn = BADGE_WARN;

	constructor() {
		super();
		this.rememberPageSize(this.constructor.name);
	}

	override handlerAfterViewInit(): void {
		super.handlerAfterViewInit();
		this.reloadSources();
		this.reloadGroups();
		this.reloadTableDataByFirstPage();
	}

	override reloadTableData(): void {
		this.svc
			.list(true)
			.pipe(
				takeUntil(this.$destroy),
				this.trackLoading(),
			)
			.subscribe({
				next: (r) => {
					const list = r.items || [];
					this.dataSource.data = list;
					this.paginatorProps.length = list.length;
				},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}

	refreshAll(): void {
		this.reloadTableDataByFirstPage();
		this.reloadSources(true);
		this.reloadGroups(true);
	}

	reloadSources(force = false): void {
		this.sources
			.list(force)
			.pipe(takeUntil(this.$destroy))
			.subscribe({
				next: (r) => this.sourceList.set(r.items || []),
				error: () => {},
			});
	}

	reloadGroups(force = false): void {
		this.rules
			.listGroups(force)
			.pipe(takeUntil(this.$destroy))
			.subscribe({
				next: (r) => this.groupList.set(r.items || []),
				error: () => {},
			});
	}

	statusText(v: string): string {
		return enumText(TOKEN_STATUS_OPTIONS, v);
	}

	statusClass(v: string): string {
		return enumBadgeClass(TOKEN_STATUS_BADGE, v);
	}

	groupModeText(v?: string): string {
		if (v === 'all') return '全部';
		if (v === 'custom') return '自定义';
		return '自动';
	}

	/** 列表展示：路径保留，token 中间用 **，仅保留后 4 位 */
	maskSubscribeUrl(url: string): string {
		const i = url.lastIndexOf('/');
		if (i < 0) {
			if (url.length <= 4) return url;
			return `**${  url.slice(-4)}`;
		}
		const base = url.slice(0, i + 1);
		const token = url.slice(i + 1);
		if (token.length <= 4) return url;
		return `${base  }**${  token.slice(-4)}`;
	}

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

openCreate(): void {
			const data: TokenFormDialogData = {
				sourceList: this.sourceList(),
				groupList: this.groupList(),
			};
			const ref = this.dialogOpen.openForm(TokenFormComponent, data, {
				width: CM_DIALOG_WIDTH.form,
			});
			ref.afterClosed().subscribe((ok) => {
				if (ok) this.reloadTableDataByFirstPage();
			});
		}

		openEdit(item: ShareToken): void {
			const data: TokenFormDialogData = {
				token: item,
				sourceList: this.sourceList(),
				groupList: this.groupList(),
			};
			const ref = this.dialogOpen.openForm(TokenFormComponent, data, {
				width: CM_DIALOG_WIDTH.form,
			});
			ref.afterClosed().subscribe((ok) => {
				if (ok) this.reloadTableDataByFirstPage();
			});
		}

	disable(item: ShareToken): void {
		this.busy.set(true);
		this.svc
			.update(item.id, { status: TokenStatus.Disabled })
			.pipe(
				takeUntil(this.$destroy),
				finalize(() => this.busy.set(false)),
			)
			.subscribe({
next: () => {
						void this.dialog.success(msgDisabled(item.name));
						this.reloadTableDataByFirstPage();
					},
					error: (e: Error) => void this.dialog.error(e.message),
				});
		}

		enable(item: ShareToken): void {
			this.busy.set(true);
			this.svc
				.update(item.id, { status: TokenStatus.Active })
				.pipe(
					takeUntil(this.$destroy),
					finalize(() => this.busy.set(false)),
				)
				.subscribe({
					next: () => {
						void this.dialog.success(msgEnabled(item.name));
					this.reloadTableDataByFirstPage();
				},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}

	async revoke(item: ShareToken): Promise<void> {
		const ok = await this.dialog.confirm(
			`作废「${item.name}」？\n旧链接立即失效，记录与访问次数保留，之后可重新生成。`,
			TITLE_CONFIRM_REVOKE,
			'作废',
		);
		if (!ok) return;
		this.busy.set(true);
		this.svc
			.revoke(item.id)
			.pipe(
				takeUntil(this.$destroy),
				finalize(() => this.busy.set(false)),
			)
			.subscribe({
				next: () => {
					void this.dialog.success(msgRevoked(item.name));
					this.reloadTableDataByFirstPage();
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
		this.busy.set(true);
		this.svc
			.regenerate(item.id)
			.pipe(
				takeUntil(this.$destroy),
				finalize(() => this.busy.set(false)),
			)
			.subscribe({
				next: (t) => {
					void this.dialog.success(
						t.subscribeUrl ? `已重新生成，请复制新链接` : `已重新生成「${item.name}」`,
					);
					this.reloadTableDataByFirstPage();
				},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}

	async remove(item: ShareToken): Promise<void> {
		const ok = await this.dialog.confirm(
			`永久删除「${item.name}」？\n记录不可恢复（与「作废」不同）。`,
			TITLE_CONFIRM_DELETE,
			'删除',
		);
		if (!ok) return;
		this.busy.set(true);
		this.svc
			.delete(item.id)
			.pipe(
				takeUntil(this.$destroy),
				finalize(() => this.busy.set(false)),
			)
			.subscribe({
				next: () => {
					void this.dialog.success(msgDeleted(item.name));
					this.reloadTableDataByFirstPage();
				},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}

	async copy(text: string): Promise<void> {
		try {
			await copyToClipboard(text);
			void this.dialog.success(MSG_COPIED);
		} catch {
			void this.dialog.error(MSG_COPY_FAILED);
		}
	}
}

import { AfterViewInit, Component, inject, signal } from '@angular/core';
import { MatTableDataSource } from '@angular/material/table';
import {
	BADGE_WARN,
	DraftChangesDialogData,
	MatchableRule,
	RELEASE_STATUS_BADGE,
	RELEASE_STATUS_OPTIONS,
	Release,
	ReleaseDetailDialogData,
	ReleaseRuleLine,
	RuleMatchDialogData,
	enumBadgeClass,
	enumText,
} from '@data-struct';
import { DialogService } from '@common/services/dialog.service';
import { DraftStatusStore } from '../services/draft-status.store';
import { ReleaseService } from '../services/release.service';
import { CM_DIALOG_WIDTH, CmDialogOpenService } from '@common/modules/dialog';
import { CmParentTableComponent } from '@common/parents/parent-table/parent-table.component';
import { finalize, takeUntil } from 'rxjs';
import { RuleMatchDialogComponent } from '../../_shared/rule-match-dialog/rule-match-dialog.component';
import { DraftChangesComponent } from '../draft-changes/draft-changes.component';
import { ReleaseDetailComponent } from '../release-detail/release-detail.component';

@Component({
	selector: 'app-release-list',
	templateUrl: './release-list.component.html',
	standalone: false,
})
export class ReleaseListComponent extends CmParentTableComponent implements AfterViewInit {
	private svc = inject(ReleaseService);
	private draftStore = inject(DraftStatusStore);
	private dialog = inject(DialogService);
	private dialogOpen = inject(CmDialogOpenService);

	dataSource = new MatTableDataSource<Release>([]);
	override displayedColumns = ['version', 'status', 'counts', 'note', 'hash', 'createdBy', 'time', 'action'];

	publishing = signal(false);
	draftDirty = this.draftStore.dirty;
	draftChanges = this.draftStore.changes;
	draftSummary = this.draftStore.summary;
	readonly badgeWarn = BADGE_WARN;
	readonly changesPreviewLimit = 20;

	constructor() {
		super();
		this.rememberPageSize(this.constructor.name);
	}

	override handlerAfterViewInit(): void {
		super.handlerAfterViewInit();
		this.refreshDraft();
		this.reloadTableDataByFirstPage();
	}

	refreshDraft(): void {
		this.draftStore.refresh();
	}

	openDraftChanges(): void {
		const data: DraftChangesDialogData = {
			changes: this.draftChanges(),
			summary: this.draftSummary(),
			publishedVersion: this.draftStore.status()?.publishedVersion,
		};
		this.dialogOpen.openContent(DraftChangesComponent, data, {
			width: CM_DIALOG_WIDTH.medium,
		});
	}

	private changeActionText(a: string): string {
		switch (a) {
			case 'added':
				return '新增';
			case 'removed':
				return '删除';
			case 'modified':
				return '修改';
			default:
				return a;
		}
	}

	private changeKindText(k: string): string {
		switch (k) {
			case 'proxy':
				return '节点';
			case 'group':
				return '策略组';
			case 'rule':
				return '规则';
			default:
				return k;
		}
	}

	private buildPublishConfirmMessage(): string {
		const head = '确认发布当前草稿配置？\n发布后「全部源 + 自动」的订阅链接将使用新配置。';
		const changes = this.draftChanges();
		if (!changes.length) return head;
		const summary = this.draftSummary();
		const lines = changes.slice(0, this.changesPreviewLimit).map((c) => {
			const detail = c.detail ? `（${c.detail}）` : '';
			return `· ${this.changeActionText(c.action)} ${this.changeKindText(c.kind)} ${c.name}${detail}`;
		});
		const more =
			changes.length > this.changesPreviewLimit ? `\n… 等共 ${changes.length} 项` : '';
		const summaryLine = summary ? `\n本次更改：${summary}` : '';
		return `${head}\n${summaryLine}\n\n${lines.join('\n')}${more}`;
	}

	async publishNow(): Promise<void> {
		const ok = await this.dialog.confirm(this.buildPublishConfirmMessage(), '发布确认', '发布');
		if (!ok) return;
		this.publishing.set(true);
		this.svc
			.publish('')
			.pipe(
				takeUntil(this.$destroy),
				finalize(() => this.publishing.set(false)),
			)
			.subscribe({
				next: (res) => {
					void this.dialog.success(`已发布 v${res.release.version}`);
					this.reloadTableDataByFirstPage();
					this.refreshDraft();
				},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}

	statusText(v: string): string {
		return enumText(RELEASE_STATUS_OPTIONS, v);
	}

	statusClass(v: string): string {
		return enumBadgeClass(RELEASE_STATUS_BADGE, v);
	}

	override reloadTableData(): void {
		this.isLoading = true;
		this.svc
			.list(true)
			.pipe(
				takeUntil(this.$destroy),
				finalize(() => (this.isLoading = false)),
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

	async rollback(item: Release): Promise<void> {
		const ok = await this.dialog.confirm(
			`回滚到 v${item.version}？\n将基于该版本生成新的发布记录。`,
			'回滚确认',
			'回滚',
		);
		if (!ok) return;
		this.svc
			.rollback(item.id)
			.pipe(takeUntil(this.$destroy))
			.subscribe({
				next: (r) => {
					void this.dialog.success(`已回滚，新版本 v${r.version}`);
					this.reloadTableDataByFirstPage();
					this.refreshDraft();
				},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}

	openView(item: Release): void {
		this.svc
			.get(item.id)
			.pipe(takeUntil(this.$destroy))
			.subscribe({
				next: (d) => {
					const data: ReleaseDetailDialogData = { detail: d };
					this.dialogOpen.openContent(ReleaseDetailComponent, data, {
						width: CM_DIALOG_WIDTH.large,
					});
				},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}

	openTest(item: Release): void {
		this.svc
			.get(item.id)
			.pipe(takeUntil(this.$destroy))
			.subscribe({
				next: (d) => {
					const data: RuleMatchDialogData = {
						title: `测试 v${d.version} 规则`,
						subtitle: `v${d.version} · 按该版本已发布规则从上到下由服务端模拟匹配（含 GEOSITE/GEOIP）。`,
						rules: this.toMatchRules(d.rules || []),
						typeText: (t) => t,
						targetText: (t) => t,
					};
					this.dialogOpen.openContent(RuleMatchDialogComponent, data, {
						width: CM_DIALOG_WIDTH.large,
					});
				},
				error: (e: Error) => void this.dialog.error(e.message || '加载版本规则失败'),
			});
	}

	private toMatchRules(rules: ReleaseRuleLine[]): MatchableRule[] {
		return rules.map((r) => ({
			type: r.type,
			payload: r.payload || '',
			target: r.target,
			raw: r.raw,
			enabled: true,
		}));
	}
}

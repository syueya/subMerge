import { AfterViewInit, Component, inject, signal } from '@angular/core';
import { MatTableDataSource } from '@angular/material/table';
import {
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
import { DraftStatusStore, summarizeChanges } from '../services/draft-status.store';
	import { ReleaseService } from '../services/release.service';
import { CM_DIALOG_WIDTH, CmDialogOpenService } from '@common/modules/dialog';
import { CmParentTableComponent } from '@common/parents/parent-table/parent-table.component';
import { finalize, firstValueFrom, takeUntil } from 'rxjs';
import { RuleMatchDialogComponent } from '../../_shared/rule-match-dialog/rule-match-dialog.component';
import {
	PublishNoteFormComponent,
	PublishNoteFormResult,
} from '../publish-note-form/publish-note-form.component';
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
	override displayedColumns = ['version', 'counts', 'note', 'hash', 'createdBy', 'time', 'action'];

	publishing = signal(false);
  draftDirty = this.draftStore.dirty;
	draftChanges = this.draftStore.changes;
	draftSummary = this.draftStore.summary;

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

	async publishNow(): Promise<void> {
		const changes = await this.draftStore.ensureChanges();
		const result = (await firstValueFrom(
			this.dialogOpen
				.openSmallForm(PublishNoteFormComponent, {
					changes,
					summary: summarizeChanges(changes) || this.draftSummary(),
				})
				.afterClosed(),
		)) as PublishNoteFormResult | undefined;
		if (!result?.confirmed) return;
		this.publishing.set(true);
		this.svc
			.publish(result.note)
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

	async remove(item: Release): Promise<void> {
		const ok = await this.dialog.confirm(
			`确认删除 v${item.version}？\n删除后不可恢复。`,
			'删除确认',
			'删除',
		);
		if (!ok) return;
		this.svc
			.delete(item.id)
			.pipe(takeUntil(this.$destroy))
			.subscribe({
				next: () => {
					void this.dialog.success(`已删除 v${item.version}`);
					this.reloadTableDataByFirstPage();
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

import { AfterViewInit, ChangeDetectorRef, Component, computed, effect, inject, signal } from '@angular/core';
import { MatTableDataSource } from '@angular/material/table';
import {
	BADGE_MUTED,
	BADGE_WARN,
	FALLBACK_REGION,
	REFRESH_STATUS_BADGE,
	REFRESH_STATUS_OPTIONS,
	RegionCatalogEntry,
	SourceFormDialogData,
	SourceProxiesDialogData,
	SubscriptionSource,
	enumBadgeClass,
	enumText,
	regionOptionText,
} from '@data-struct';
import { DialogService } from '@common/services/dialog.service';
import { formatDateTime } from '@common/util/format';
import { SourceService } from '../services/source.service';
import { formatRefreshMsg } from '../services/source-refresh.util';
import {
	hasTraffic,
	trafficExpireText,
	trafficPercent,
	trafficText,
	trafficTitle,
} from '../services/source-traffic.util';
import { CM_DIALOG_WIDTH, CmDialogOpenService } from '@common/modules/dialog';
import { CmParentTableComponent } from '@common/parents/parent-table/parent-table.component';
import { finalize, firstValueFrom, takeUntil } from 'rxjs';
import { SourceFormComponent } from '../source-form/source-form.component';
import { SourceProxiesComponent } from '../source-proxies/source-proxies.component';

@Component({
	selector: 'app-source-list',
	templateUrl: './source-list.component.html',
	standalone: false,
})
export class SourceListComponent extends CmParentTableComponent implements AfterViewInit {
	private svc = inject(SourceService);
	private dialog = inject(DialogService);
	private dialogOpen = inject(CmDialogOpenService);
	private cdr = inject(ChangeDetectorRef);

	dataSource = new MatTableDataSource<SubscriptionSource>([]);
	override displayedColumns: string[] = [
		'select',
		'name',
		'nodes',
		'traffic',
		'status',
		'action',
	];

	busy = signal(false);
	/** 后台拉取全部状态来自 SourceService，跨路由共享 */
	readonly refreshingAll = this.svc.refreshingAll;
	selectedIds = signal<Set<number>>(new Set());
	regionCatalog = signal<RegionCatalogEntry[]>([]);
	fallbackRegion = signal(FALLBACK_REGION);

	formatTime = formatDateTime;
	hasTraffic = hasTraffic;
	trafficText = trafficText;
	trafficPercent = trafficPercent;
	trafficExpireText = trafficExpireText;
	trafficTitle = trafficTitle;
	badgeMuted = BADGE_MUTED;
	badgeWarn = BADGE_WARN;

	selectedCount = computed(() => this.selectedIds().size);

	allSelected = computed(() => {
		const list = this.dataSource.data;
		if (list.length === 0) return false;
		const sel = this.selectedIds();
		return list.every((i) => sel.has(i.id));
	});

	someSelected = computed(() => {
		const n = this.selectedIds().size;
		return n > 0 && !this.allSelected();
	});

	private wasRefreshingAll = false;

	constructor() {
		super();
		this.rememberPageSize(this.constructor.name);
		// 后台拉取全部结束后若仍在本页，自动刷新列表状态/节点数
		effect(() => {
			const busy = this.refreshingAll();
			if (this.wasRefreshingAll && !busy) {
				this.reloadTableDataByFirstPage();
			}
			this.wasRefreshingAll = busy;
		});
	}

	override handlerAfterViewInit(): void {
		super.handlerAfterViewInit();
		this.loadRegionCatalog();
		this.reloadTableDataByFirstPage();
	}

	loadRegionCatalog(): void {
		this.svc
			.listRegions()
			.pipe(takeUntil(this.$destroy))
			.subscribe({
				next: (res) => {
					const items = res.items || [];
					if (items.length) this.regionCatalog.set(items);
					if (res.fallbackRegion) this.fallbackRegion.set(String(res.fallbackRegion).toUpperCase());
				},
				error: () => {},
			});
	}

	override reloadTableData(): void {
		void this.reloadTableDataAsync();
	}

	/** 列表刷新；在弹窗打开期间完成时也强制刷新视图，避免 isLoading 遮罩卡住 */
	private reloadTableDataAsync(): Promise<void> {
		this.isLoading = true;
		this.safeDetectChanges();
		return new Promise((resolve) => {
			this.svc
				.list(true)
				.pipe(
					takeUntil(this.$destroy),
					finalize(() => {
						this.isLoading = false;
						this.safeDetectChanges();
						resolve();
					}),
				)
				.subscribe({
					next: (res) => {
						const list = res.items || [];
						this.dataSource.data = list;
						this.paginatorProps.length = list.length;
						const valid = new Set(list.map((i) => i.id));
						this.selectedIds.update((prev) => {
							const next = new Set<number>();
							for (const id of prev) {
								if (valid.has(id)) next.add(id);
							}
							return next;
						});
					},
					error: (err: Error) => void this.dialog.error(err.message),
				});
		});
	}

	private safeDetectChanges(): void {
		try {
			this.cdr.detectChanges();
		} catch {
			// 组件已销毁时忽略
		}
	}

	regionText(v: string): string {
		const code = String(v || '').toUpperCase();
		const hit = this.regionCatalog().find((r) => String(r.code).toUpperCase() === code);
		return regionOptionText(code, hit?.name);
	}

	regionCode(v: string): string {
		return String(v || '').toUpperCase() || '—';
	}

	statusText(v: string): string {
		return enumText(REFRESH_STATUS_OPTIONS, v);
	}

	statusClass(v: string): string {
		return enumBadgeClass(REFRESH_STATUS_BADGE, v);
	}

	regionModeText(v: string): string {
		return v === 'fixed' ? '固定地区' : '自动识别';
	}

	isSelected(id: number): boolean {
		return this.selectedIds().has(id);
	}

	toggleSelect(id: number, checked: boolean): void {
		this.selectedIds.update((prev) => {
			const next = new Set(prev);
			if (checked) next.add(id);
			else next.delete(id);
			return next;
		});
	}

	toggleSelectAll(checked: boolean): void {
		if (!checked) {
			this.selectedIds.set(new Set());
			return;
		}
		this.selectedIds.set(new Set(this.dataSource.data.map((i) => i.id)));
	}

	clearSelection(): void {
		this.selectedIds.set(new Set());
	}

	openCreate(): void {
		this.openForm(null);
	}

	openEdit(item: SubscriptionSource): void {
		this.openForm(item);
	}

	private openForm(source: SubscriptionSource | null): void {
		const data: SourceFormDialogData = {
			source,
			regionCatalog: this.regionCatalog(),
			fallbackRegion: this.fallbackRegion(),
		};
		const ref = this.dialogOpen.openForm(SourceFormComponent, data, {
			width: CM_DIALOG_WIDTH.form,
		});
		ref.afterClosed().subscribe((ok) => {
			if (ok) this.reloadTableDataByFirstPage();
		});
	}

	openProxies(item: SubscriptionSource): void {
		const data: SourceProxiesDialogData = { source: item };
		const ref = this.dialogOpen.openContent(SourceProxiesComponent, data, {
			width: CM_DIALOG_WIDTH.large,
		});
		ref.afterClosed().subscribe((changed) => {
			if (changed) this.reloadTableDataByFirstPage();
		});
	}

	async refresh(item: SubscriptionSource): Promise<void> {
		const confirmed = await this.dialog.confirm(
			`确认拉取订阅源「${item.name}」？\n将重新从上游获取节点并覆盖本地数据。`,
			'拉取确认',
			'拉取',
		);
		if (!confirmed) return;

		this.busy.set(true);
		try {
			const res = await firstValueFrom(this.svc.refresh(item.id).pipe(takeUntil(this.$destroy)));
			// 先刷新列表再弹结果，避免成功弹窗与 isLoading 叠在一起导致遮罩不消失
			await this.reloadTableDataAsync();
			await this.dialog.success(formatRefreshMsg(res, '拉取成功'));
		} catch (err) {
			void this.dialog.error((err as Error).message);
		} finally {
			this.busy.set(false);
		}
	}

	async refreshAll(): Promise<void> {
		if (this.refreshingAll()) return;
		const enabled = this.dataSource.data.filter((i) => i.enabled);
		if (enabled.length === 0) {
			void this.dialog.error('没有已启用的订阅源');
			return;
		}
		const confirmed = await this.dialog.confirm(
			`确认拉取全部已启用的订阅源？\n共 ${enabled.length} 个，将重新从上游获取节点并覆盖本地数据。\n拉取在后台进行，切换页面不会中断；完成后以通知提示结果。`,
			'拉取确认',
			'拉取全部',
		);
		if (!confirmed) return;

		if (!this.svc.startBackgroundRefreshAll()) return;
		void this.dialog.success('已开始后台拉取全部订阅源');
	}

	toggle(item: SubscriptionSource): void {
		this.svc
			.update(item.id, { enabled: !item.enabled })
			.pipe(takeUntil(this.$destroy))
			.subscribe({
				next: () => {
					void this.dialog.success(item.enabled ? `已禁用「${item.name}」` : `已启用「${item.name}」`);
					this.reloadTableDataByFirstPage();
				},
				error: (err: Error) => void this.dialog.error(err.message),
			});
	}

	async remove(item: SubscriptionSource): Promise<void> {
		const ok = await this.dialog.confirm(
			`确认删除订阅源「${item.name}」？\n其节点也会一并删除（订阅 URL 已加密保存在库中，删除后不可恢复）。`,
			'删除确认',
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
					void this.dialog.success(`已删除「${item.name}」`);
					this.reloadTableDataByFirstPage();
				},
				error: (err: Error) => void this.dialog.error(err.message),
			});
	}

	async batchRemove(): Promise<void> {
		const ids = [...this.selectedIds()];
		if (ids.length === 0) {
			void this.dialog.error('请先勾选要删除的订阅源');
			return;
		}
		const ok = await this.dialog.confirm(
			`确认删除选中的 ${ids.length} 个订阅源？\n对应节点会一并删除，且不可恢复。`,
			'批量删除',
			'删除',
		);
		if (!ok) return;
		this.busy.set(true);
		this.svc
			.batchDelete(ids)
			.pipe(
				takeUntil(this.$destroy),
				finalize(() => this.busy.set(false)),
			)
			.subscribe({
				next: (res) => {
					this.selectedIds.set(new Set());
					void this.dialog.success(`已删除 ${res.deleted} 个订阅源`);
					this.reloadTableDataByFirstPage();
				},
				error: (err: Error) => void this.dialog.error(err.message),
			});
	}
}

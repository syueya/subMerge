import { Component, HostListener, OnInit, computed, inject, signal } from '@angular/core';
import {
	FALLBACK_REGION,
	REFRESH_STATUS_BADGE,
	REFRESH_STATUS_OPTIONS,
	RegionCatalogEntry,
	SubscriptionSource,
	enumBadgeClass,
	enumText,
	regionOptionText,
} from '../../common/types';
import { DialogService } from '../../common/dialog/dialog.service';
import { formatDateTime } from '../../common/format';
import { SourceService } from './source.service';
import { SourceFormModalComponent } from './source-form-modal.component';
import { SourceProxiesModalComponent } from './source-proxies-modal.component';
import { formatRefreshMsg } from './source-refresh.util';
import {
	hasTraffic,
	trafficExpireText,
	trafficPercent,
	trafficText,
	trafficTitle,
} from './source-traffic.util';

@Component({
	selector: 'app-source-list',
	standalone: true,
	imports: [SourceFormModalComponent, SourceProxiesModalComponent],
	templateUrl: './source-list.component.html',
})
export class SourceListComponent implements OnInit {
	private readonly svc = inject(SourceService);
	private readonly dialog = inject(DialogService);

	items = signal<SubscriptionSource[]>([]);
	loading = signal(false);
	busy = signal(false);
	/** 多选删除 */
	selectedIds = signal<Set<number>>(new Set());

	// create / edit form
	showForm = signal(false);
	editingSource: SubscriptionSource | null = null;

	/** 来自 defaults/regions.yaml（GET /regions）；唯一地区目录 */
	regionCatalog = signal<RegionCatalogEntry[]>([]);
	fallbackRegion = signal(FALLBACK_REGION);

	// proxy panel
	showProxies = signal(false);
	proxySource = signal<SubscriptionSource | null>(null);

	/** 列表时间：本地简写 */
	formatTime = formatDateTime;

	// traffic helpers for template
	hasTraffic = hasTraffic;
	trafficText = trafficText;
	trafficPercent = trafficPercent;
	trafficExpireText = trafficExpireText;
	trafficTitle = trafficTitle;

	selectedCount = computed(() => this.selectedIds().size);

	allSelected = computed(() => {
		const list = this.items();
		if (list.length === 0) return false;
		const sel = this.selectedIds();
		return list.every((i) => sel.has(i.id));
	});

	someSelected = computed(() => {
		const n = this.selectedIds().size;
		return n > 0 && !this.allSelected();
	});

	ngOnInit(): void {
		this.loadRegionCatalog();
		this.reload();
	}

	loadRegionCatalog(): void {
		this.svc.listRegions().subscribe({
			next: (res) => {
				const items = res.items || [];
				if (items.length) this.regionCatalog.set(items);
				if (res.fallbackRegion) this.fallbackRegion.set(String(res.fallbackRegion).toUpperCase());
			},
			error: () => {
				/* 无本地地区表；目录为空时下拉暂不可选 */
			},
		});
	}

	@HostListener('document:keydown.escape')
	onEsc(): void {
		if (this.busy()) return;
		if (this.showProxies()) {
			this.closeProxies();
			return;
		}
		if (this.showForm()) {
			this.closeForm();
		}
	}

	regionText(v: string): string {
		const code = String(v || '').toUpperCase();
		const hit = this.regionCatalog().find((r) => String(r.code).toUpperCase() === code);
		return regionOptionText(code, hit?.name);
	}

	/** 表格里用短码，完整名放 title */
	regionCode(v: string): string {
		const code = String(v || '').toUpperCase() || '—';
		return code;
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

	reload(): void {
		this.loading.set(true);
		this.svc.list().subscribe({
			next: (res) => {
				const list = res.items || [];
				this.items.set(list);
				// 清理已不存在的选中项
				const valid = new Set(list.map((i) => i.id));
				this.selectedIds.update((prev) => {
					const next = new Set<number>();
					for (const id of prev) {
						if (valid.has(id)) next.add(id);
					}
					return next;
				});
				this.loading.set(false);
			},
			error: (err: Error) => {
				this.loading.set(false);
				void this.dialog.error(err.message);
			},
		});
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
		this.selectedIds.set(new Set(this.items().map((i) => i.id)));
	}

	clearSelection(): void {
		this.selectedIds.set(new Set());
	}

	openCreate(): void {
		this.editingSource = null;
		this.showForm.set(true);
	}

	openEdit(item: SubscriptionSource): void {
		this.editingSource = item;
		this.showForm.set(true);
	}

	closeForm(): void {
		this.showForm.set(false);
		this.editingSource = null;
	}

	onFormSaved(): void {
		this.closeForm();
		this.reload();
	}

	refresh(item: SubscriptionSource): void {
		this.busy.set(true);
		this.svc.refresh(item.id).subscribe({
			next: (res) => {
				this.busy.set(false);
				void this.dialog.success(formatRefreshMsg(res, '刷新成功'));
				this.reload();
			},
			error: (err: Error) => {
				this.busy.set(false);
				void this.dialog.error(err.message);
			},
		});
	}

	refreshAll(): void {
		const enabled = this.items().filter((i) => i.enabled);
		if (enabled.length === 0) {
			void this.dialog.error('没有已启用的订阅源');
			return;
		}
		this.busy.set(true);
		this.svc.refreshAll().subscribe({
			next: (res) => {
				this.busy.set(false);
				const lines = [`全部拉取完成：成功 ${res.ok} / 共 ${res.total}`];
				if (res.failed > 0) {
					const fails = (res.results || [])
						.filter((r) => !r.ok)
						.map((r) => `· ${r.name}: ${r.error || '失败'}`)
						.slice(0, 5);
					lines.push(`失败 ${res.failed}`, ...fails);
				}
				void this.dialog.success(lines.join('\n'));
				this.reload();
			},
			error: (err: Error) => {
				this.busy.set(false);
				void this.dialog.error(err.message);
			},
		});
	}

	toggle(item: SubscriptionSource): void {
		this.svc.update(item.id, { enabled: !item.enabled }).subscribe({
			next: () => {
				void this.dialog.success(item.enabled ? `已禁用「${item.name}」` : `已启用「${item.name}」`);
				this.reload();
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
		this.svc.delete(item.id).subscribe({
			next: () => {
				this.busy.set(false);
				void this.dialog.success(`已删除「${item.name}」`);
				this.reload();
			},
			error: (err: Error) => {
				this.busy.set(false);
				void this.dialog.error(err.message);
			},
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
		this.svc.batchDelete(ids).subscribe({
			next: (res) => {
				this.busy.set(false);
				this.selectedIds.set(new Set());
				void this.dialog.success(`已删除 ${res.deleted} 个订阅源`);
				this.reload();
			},
			error: (err: Error) => {
				this.busy.set(false);
				void this.dialog.error(err.message);
			},
		});
	}

	openProxies(item: SubscriptionSource): void {
		this.proxySource.set(item);
		this.showProxies.set(true);
	}

	closeProxies(): void {
		this.showProxies.set(false);
		this.proxySource.set(null);
	}

	onProxiesRefreshed(): void {
		this.reload();
	}
}

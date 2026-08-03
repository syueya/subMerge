import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { MatTableDataSource } from '@angular/material/table';
import {
	BADGE_MUTED,
	BADGE_OK,
	BADGE_WARN,
	ProxyNode,
	SourceProxiesDialogData,
	SubscriptionSource,
} from '@data-struct';
import { DialogService } from '@common/services/dialog.service';
import { SourceService } from '../services/source.service';
import { formatRefreshMsg } from '../services/source-refresh.util';
import { takeUntil } from 'rxjs';
import { CmParentComponent } from '@common/parents/parent/parent.component';

@Component({
	selector: 'app-source-proxies',
	templateUrl: './source-proxies.component.html',
	standalone: false,
})
export class SourceProxiesComponent extends CmParentComponent implements OnInit {
	dialogRef = inject<MatDialogRef<SourceProxiesComponent, boolean>>(MatDialogRef);
	data = inject<SourceProxiesDialogData>(MAT_DIALOG_DATA);
	private svc = inject(SourceService);
	private dialog = inject(DialogService);

	dataSource = new MatTableDataSource<ProxyNode>([]);
	displayedColumns = ['enabled', 'name', 'region', 'type', 'server', 'quality'];

	proxies = signal<ProxyNode[]>([]);
	isLoading = false;
	proxySearch = signal('');
	proxyRegionFilter = signal('');
	proxyQualityFilter = signal('');
	lastRefreshSummary = signal('');
	busy = signal(false);
	refreshed = false;

	filteredProxies = computed(() => {
		const q = this.proxySearch().trim().toLowerCase();
		const region = this.proxyRegionFilter().trim().toUpperCase();
		const quality = this.proxyQualityFilter();
		return this.proxies().filter((p) => {
			if (region && String(p.region).toUpperCase() !== region) return false;
			if (quality === 'ok' && p.ok === false) return false;
			if (quality === 'bad' && p.ok !== false) return false;
			if (!q) return true;
			const hay = `${p.name} ${p.server} ${p.type} ${p.region} ${p.issue || ''}`.toLowerCase();
			return hay.includes(q);
		});
	});

	proxyRegions = computed(() => {
		const set = new Set<string>();
		for (const p of this.proxies()) {
			if (p.region) set.add(String(p.region).toUpperCase());
		}
		return [...set].sort();
	});

	readonly badgeOk = BADGE_OK;
	readonly badgeWarn = BADGE_WARN;
	readonly badgeMuted = BADGE_MUTED;

	get source(): SubscriptionSource {
		return this.data.source;
	}

	override ngOnInit(): void {
		super.ngOnInit();
		this.loadProxies();
	}

	applyFilter(): void {
		this.dataSource.data = this.filteredProxies();
	}

	onSearchChange(v: string): void {
		this.proxySearch.set(v);
		this.applyFilter();
	}

	onRegionChange(v: string): void {
		this.proxyRegionFilter.set(v);
		this.applyFilter();
	}

	onQualityChange(v: string): void {
		this.proxyQualityFilter.set(v);
		this.applyFilter();
	}

	loadProxies(): void {
		this.isLoading = true;
		this.svc
			.listProxies(this.source.id)
			.pipe(takeUntil(this.$destroy))
			.subscribe({
				next: (res) => {
					this.proxies.set(res.items || []);
					this.applyFilter();
					this.isLoading = false;
					const counts: Record<string, number> = {};
					for (const p of res.items || []) {
						const r = String(p.region || '?').toUpperCase();
						counts[r] = (counts[r] || 0) + 1;
					}
					const summary = Object.entries(counts)
						.sort((a, b) => a[0].localeCompare(b[0]))
						.map(([k, v]) => `${k}:${v}`)
						.join(' ');
					this.lastRefreshSummary.set(summary || '无节点');
				},
				error: (err: Error) => {
					this.isLoading = false;
					void this.dialog.error(err.message);
				},
			});
	}

	toggleProxy(p: ProxyNode): void {
		this.svc
			.updateProxy(p.id, !p.enabled)
			.pipe(takeUntil(this.$destroy))
			.subscribe({
				next: (updated) => {
					this.proxies.update((list) =>
						list.map((x) => (x.id === updated.id ? { ...x, enabled: updated.enabled } : x)),
					);
					this.applyFilter();
					this.refreshed = true;
				},
				error: (err: Error) => void this.dialog.error(err.message),
			});
	}

	batchSetFiltered(enabled: boolean): void {
		const ids = this.filteredProxies().map((p) => p.id);
		if (ids.length === 0) {
			void this.dialog.error('当前筛选结果为空');
			return;
		}
		this.svc
			.batchUpdateProxies(ids, enabled)
			.pipe(takeUntil(this.$destroy))
			.subscribe({
				next: (res) => {
					const idSet = new Set(ids);
					this.proxies.update((list) => list.map((x) => (idSet.has(x.id) ? { ...x, enabled } : x)));
					this.applyFilter();
					this.refreshed = true;
					void this.dialog.success(`已${enabled ? '启用' : '禁用'} ${res.updated} 个节点`);
				},
				error: (err: Error) => void this.dialog.error(err.message),
			});
	}

	refresh(): void {
		this.busy.set(true);
		this.svc
			.refresh(this.source.id)
			.pipe(takeUntil(this.$destroy))
			.subscribe({
				next: (res) => {
					this.busy.set(false);
					void this.dialog.success(formatRefreshMsg(res, '刷新成功'));
					this.refreshed = true;
					this.loadProxies();
				},
				error: (err: Error) => {
					this.busy.set(false);
					void this.dialog.error(err.message);
				},
			});
	}

	close(): void {
		this.dialogRef.close(this.refreshed);
	}

	tableTrackBy = (_: number, row: ProxyNode) => row.id;
}

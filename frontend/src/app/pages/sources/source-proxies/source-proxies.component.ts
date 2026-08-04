import { ChangeDetectorRef, Component, OnInit, computed, inject, signal } from '@angular/core';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
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
import { Subject, debounceTime, distinctUntilChanged, finalize, takeUntil } from 'rxjs';
import { CmParentComponent } from '@common/parents/parent/parent.component';

/** 搜索归一化：去零宽字符 + NFKC（全角→半角）+ 小写 */
function normalizeSearch(s: string): string {
	return String(s || '')
		.replace(/[\u200B-\u200D\uFEFF]/g, '')
		.normalize('NFKC')
		.trim()
		.toLowerCase();
}

/** 节点是否命中关键词（分字段 includes，单字母也可） */
function proxyMatchesQuery(p: ProxyNode, q: string): boolean {
	if (!q) return true;
	const fields = [
		p.name,
		p.server,
		p.type,
		p.region,
		p.issue || '',
		p.port != null ? String(p.port) : '',
	];
	return fields.some((f) => normalizeSearch(String(f ?? '')).includes(q));
}

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
	private cdr = inject(ChangeDetectorRef);

	displayedColumns = ['enabled', 'name', 'region', 'type', 'server', 'quality'];

	proxies = signal<ProxyNode[]>([]);
	isLoading = false;
	/** 输入框展示值 */
	proxySearchInput = signal('');
	/** 防抖后的筛选词（已 normalize） */
	proxySearch = signal('');
	proxyRegionFilter = signal('');
	proxyQualityFilter = signal('');
	lastRefreshSummary = signal('');
	batchBusy = signal(false);
	refreshed = false;

	private search$ = new Subject<string>();

	/**
	 * 表格直接绑定此数组（不要再用 MatTableDataSource 中转）。
	 * 中转时 signal 更新与 dataSource.data 赋值容易不同步，表现为单字符筛完列表不刷新。
	 */
	filteredProxies = computed(() => {
		const q = this.proxySearch();
		const region = this.proxyRegionFilter().trim().toUpperCase();
		const quality = this.proxyQualityFilter();
		return this.proxies().filter((p) => {
			if (region && String(p.region || '').toUpperCase() !== region) return false;
			if (quality === 'ok' && p.ok === false) return false;
			if (quality === 'bad' && p.ok !== false) return false;
			return proxyMatchesQuery(p, q);
		});
	});

	proxyRegions = computed(() => {
		const set = new Set<string>();
		for (const p of this.proxies()) {
			if (p.region) set.add(String(p.region).toUpperCase());
		}
		return [...set].sort();
	});

	allFilteredEnabled = computed(() => {
		const list = this.filteredProxies();
		return list.length > 0 && list.every((p) => p.enabled);
	});

	someFilteredEnabled = computed(() => {
		const list = this.filteredProxies();
		if (list.length === 0) return false;
		const n = list.filter((p) => p.enabled).length;
		return n > 0 && n < list.length;
	});

	readonly badgeOk = BADGE_OK;
	readonly badgeWarn = BADGE_WARN;
	readonly badgeMuted = BADGE_MUTED;

	get source(): SubscriptionSource {
		return this.data.source;
	}

	override ngOnInit(): void {
		super.ngOnInit();
		this.search$
			.pipe(debounceTime(200), distinctUntilChanged(), takeUntil(this.$destroy))
			.subscribe((raw) => {
				this.proxySearch.set(normalizeSearch(raw));
			});
		this.loadProxies();
	}

	onSearchChange(v: string): void {
		const raw = v ?? '';
		this.proxySearchInput.set(raw);
		this.search$.next(raw);
	}

	onRegionChange(v: string): void {
		this.proxyRegionFilter.set(v ?? '');
	}

	onQualityChange(v: string): void {
		this.proxyQualityFilter.set(v ?? '');
	}

	loadProxies(): void {
		this.isLoading = true;
		this.safeDetectChanges();
		this.svc
			.listProxies(this.source.id)
			.pipe(
				takeUntil(this.$destroy),
				finalize(() => {
					this.isLoading = false;
					this.safeDetectChanges();
				}),
			)
			.subscribe({
				next: (res) => {
					this.proxies.set(res.items || []);
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
				error: (err: Error) => void this.dialog.error(err.message),
			});
	}

	private safeDetectChanges(): void {
		try {
			this.cdr.detectChanges();
		} catch {
			// 组件已销毁时忽略
		}
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
					this.refreshed = true;
				},
				error: (err: Error) => void this.dialog.error(err.message),
			});
	}

	/** 表头启用全选：对当前筛选结果批量启用/禁用 */
	toggleAllFiltered(enabled: boolean): void {
		const ids = this.filteredProxies().map((p) => p.id);
		if (ids.length === 0) {
			void this.dialog.error('当前筛选结果为空');
			return;
		}
		this.batchBusy.set(true);
		this.svc
			.batchUpdateProxies(ids, enabled)
			.pipe(
				takeUntil(this.$destroy),
				finalize(() => this.batchBusy.set(false)),
			)
			.subscribe({
				next: (res) => {
					const idSet = new Set(ids);
					this.proxies.update((list) => list.map((x) => (idSet.has(x.id) ? { ...x, enabled } : x)));
					this.refreshed = true;
					void this.dialog.success(`已${enabled ? '启用' : '禁用'} ${res.updated} 个节点`);
				},
				error: (err: Error) => void this.dialog.error(err.message),
			});
	}

	close(): void {
		this.dialogRef.close(this.refreshed);
	}

	tableTrackBy = (_: number, row: ProxyNode) => row.id;
}

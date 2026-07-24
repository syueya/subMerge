import {
	Component,
	EventEmitter,
	Input,
	OnChanges,
	Output,
	SimpleChanges,
	computed,
	inject,
	signal,
} from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ProxyNode, SubscriptionSource } from '../../common/types';
import { DialogService } from '../../common/dialog/dialog.service';
import { SourceService } from './source.service';
import { formatRefreshMsg } from './source-refresh.util';

@Component({
	selector: 'app-source-proxies-modal',
	standalone: true,
	imports: [FormsModule],
	templateUrl: './source-proxies-modal.component.html',
})
export class SourceProxiesModalComponent implements OnChanges {
	private readonly svc = inject(SourceService);
	private readonly dialog = inject(DialogService);

	@Input({ required: true }) open = false;
	@Input() source: SubscriptionSource | null = null;

	@Output() closed = new EventEmitter<void>();
	/** 重新拉取后通知父组件刷新列表计数等 */
	@Output() refreshed = new EventEmitter<void>();

	proxies = signal<ProxyNode[]>([]);
	proxyLoading = signal(false);
	proxySearch = signal('');
	proxyRegionFilter = signal('');
	/** 识别筛选：'' | 'ok' | 'bad' */
	proxyQualityFilter = signal('');
	lastRefreshSummary = signal('');
	busy = signal(false);

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

	ngOnChanges(changes: SimpleChanges): void {
		if (changes['open'] || changes['source']) {
			if (this.open && this.source) {
				this.proxySearch.set('');
				this.proxyRegionFilter.set('');
				this.proxyQualityFilter.set('');
				this.lastRefreshSummary.set('');
				this.loadProxies(this.source);
			}
			if (!this.open) {
				this.proxies.set([]);
			}
		}
	}

	close(): void {
		this.closed.emit();
	}

	loadProxies(item: SubscriptionSource): void {
		this.proxyLoading.set(true);
		this.svc.listProxies(item.id).subscribe({
			next: (res) => {
				this.proxies.set(res.items || []);
				this.proxyLoading.set(false);
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
				this.proxyLoading.set(false);
				void this.dialog.error(err.message);
			},
		});
	}

	toggleProxy(p: ProxyNode): void {
		this.svc.updateProxy(p.id, !p.enabled).subscribe({
			next: (updated) => {
				this.proxies.update((list) =>
					list.map((x) => (x.id === updated.id ? { ...x, enabled: updated.enabled } : x)),
				);
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
		this.svc.batchUpdateProxies(ids, enabled).subscribe({
			next: (res) => {
				const idSet = new Set(ids);
				this.proxies.update((list) => list.map((x) => (idSet.has(x.id) ? { ...x, enabled } : x)));
				void this.dialog.success(`已${enabled ? '启用' : '禁用'} ${res.updated} 个节点`);
			},
			error: (err: Error) => void this.dialog.error(err.message),
		});
	}

	refresh(): void {
		const item = this.source;
		if (!item) return;
		this.busy.set(true);
		this.svc.refresh(item.id).subscribe({
			next: (res) => {
				this.busy.set(false);
				void this.dialog.success(formatRefreshMsg(res, '刷新成功'));
				this.loadProxies(item);
				this.refreshed.emit();
			},
			error: (err: Error) => {
				this.busy.set(false);
				void this.dialog.error(err.message);
			},
		});
	}
}

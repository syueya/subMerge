import { Component, HostListener, OnInit, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { concatMap, finalize } from 'rxjs';
import {
	DEFAULT_EXCLUDE_NAME_REGEX,
	DEFAULT_EXCLUDE_SERVERS,
	FALLBACK_REGION,
	ProxyNode,
	REGION_OPTIONS,
	REFRESH_STATUS_BADGE,
	REFRESH_STATUS_OPTIONS,
	Region,
	RegionCatalogEntry,
	RegionMode,
	SubscriptionSource,
	enumBadgeClass,
	enumText,
	regionOptionText,
} from '../../common/types';
import { DialogService } from '../../common/dialog/dialog.service';
import { FieldTipComponent } from '../../common/field-tip/field-tip.component';
import { formatDateTime } from '../../common/format';
import { SourceService } from './source.service';

@Component({
	selector: 'app-source-list',
	standalone: true,
	imports: [FormsModule, FieldTipComponent],
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
	editingId: number | null = null;
	formName = '';
	formRegion: Region = FALLBACK_REGION;
	formUrl = '';
	formEnabled = true;
	formRegionMode: RegionMode = 'auto';
	formExcludeName = DEFAULT_EXCLUDE_NAME_REGEX;
	formExcludeServers = DEFAULT_EXCLUDE_SERVERS;
	formIncludeName = '';
	showAdvanced = signal(false);
	/** 来自 defaults/regions.yaml（API）；失败时用离线兜底 */
	regionCatalog = signal<RegionCatalogEntry[]>(
		REGION_OPTIONS.map((o) => ({
			code: o.value,
			name: String(o.text).replace(/\s*\([^)]*\)\s*$/, ''),
		})),
	);
	fallbackRegion = signal(FALLBACK_REGION);

	// proxy panel
	showProxies = signal(false);
	proxySource = signal<SubscriptionSource | null>(null);
	proxies = signal<ProxyNode[]>([]);
	proxyLoading = signal(false);
	proxySearch = signal('');
	proxyRegionFilter = signal('');
	lastRefreshSummary = signal<string>('');

readonly tip = {
			name: '备注名会拼到节点名末尾（如「良心云」→ …-良心云），请起简短可辨认的名字。',
		region:
			'固定模式：从目录选择地区，该源全部节点加此前缀。\n自动识别：回退建议选「未知 (UNKNOWN)」，仅当节点名识别失败时使用。',
		regionMode:
			'自动识别（机场推荐）：从节点名 emoji/关键词推断 JP/HK/US…，混合地区订阅一条即可；回退默认 UNKNOWN。\n固定地区：整源强制同一前缀（单地区订阅更合适）。',
		url: '上游 Clash 订阅地址。加密存储，界面只显示脱敏结果。编辑时留空表示不修改。',
		excludeName:
				'名称匹配此正则的节点会被丢弃（不区分大小写）。\n默认含：剩余流量、套餐到期、过滤掉… 等机场信息节点。\n拉取成功后弹窗会列出被过滤的名称。',
excludeServers: 'server 黑名单，逗号/换行分隔。如 127.0.0.1,localhost',
			includeName: '可选。非空时只保留名称匹配的节点（白名单）。',
		};

	/** 固定模式下拉：不含 UNKNOWN */
	fixedRegionOptions = computed(() =>
		this.regionCatalog().filter((r) => String(r.code).toUpperCase() !== FALLBACK_REGION),
	);

	/** 自动模式回退下拉：含 UNKNOWN，并置顶 */
	autoRegionOptions = computed(() => {
		const list = [...this.regionCatalog()];
		list.sort((a, b) => {
			const au = String(a.code).toUpperCase() === FALLBACK_REGION ? 0 : 1;
			const bu = String(b.code).toUpperCase() === FALLBACK_REGION ? 0 : 1;
			if (au !== bu) return au - bu;
			return String(a.code).localeCompare(String(b.code));
		});
		return list;
	});

/** 识别筛选：'' | 'ok' | 'bad' */
	proxyQualityFilter = signal('');

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
					/* 使用离线兜底 REGION_OPTIONS */
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

	/** 列表时间：本地简写 */
	formatTime = formatDateTime;

	regionOptionLabel(r: RegionCatalogEntry): string {
		return regionOptionText(r.code, r.name);
	}

		onRegionModeChange(mode: RegionMode): void {
			this.formRegionMode = mode;
			if (mode === 'auto') {
				// 自动模式默认回退 UNKNOWN
				if (!this.formRegion || this.formRegion === 'US') {
					this.formRegion = this.fallbackRegion();
				}
			} else if (mode === 'fixed') {
				// 固定模式不要默认 UNKNOWN
				if (!this.formRegion || String(this.formRegion).toUpperCase() === FALLBACK_REGION) {
					const first = this.fixedRegionOptions()[0];
					this.formRegion = first?.code || 'US';
				}
			}
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
			this.editingId = null;
			this.formName = '';
			this.formRegionMode = 'auto';
			this.formRegion = this.fallbackRegion(); // 自动模式默认 UNKNOWN
			this.formUrl = '';
			this.formEnabled = true;
			this.formExcludeName = DEFAULT_EXCLUDE_NAME_REGEX;
			this.formExcludeServers = DEFAULT_EXCLUDE_SERVERS;
			this.formIncludeName = '';
			this.showAdvanced.set(false);
			this.showForm.set(true);
		}

		openEdit(item: SubscriptionSource): void {
			this.editingId = item.id;
			this.formName = item.name;
			this.formRegionMode = item.regionMode || 'auto';
			this.formRegion =
				item.region ||
				(this.formRegionMode === 'auto' ? this.fallbackRegion() : this.fixedRegionOptions()[0]?.code || 'US');
			this.formUrl = '';
			this.formEnabled = item.enabled;
this.formExcludeName = item.excludeNameRegex ?? DEFAULT_EXCLUDE_NAME_REGEX;
				this.formExcludeServers = item.excludeServers ?? DEFAULT_EXCLUDE_SERVERS;
				this.formIncludeName = item.includeNameRegex ?? '';
				// 过滤规则面板默认收起，需要时再点开
				this.showAdvanced.set(false);
				this.showForm.set(true);
		}

	closeForm(): void {
		if (this.busy()) return;
		this.showForm.set(false);
		this.editingId = null;
	}

	save(): void {
		const region = String(this.formRegion || '')
			.trim()
			.toUpperCase();
		if (!this.formName.trim()) {
			void this.dialog.error('请填写名称');
			return;
		}
		if (!this.editingId && !this.formUrl.trim()) {
			void this.dialog.error('请填写订阅 URL');
			return;
		}
if (!/^[A-Z0-9]{1,16}$/.test(region)) {
				void this.dialog.error('请选择有效地区（如 UNKNOWN、US、JP、HK）');
				return;
			}
			if (this.formRegionMode === 'fixed' && region === FALLBACK_REGION) {
				void this.dialog.error('固定地区模式请选择具体国家/地区，不要用 UNKNOWN');
				return;
			}

		const body = {
			name: this.formName.trim(),
			region,
			enabled: this.formEnabled,
			regionMode: this.formRegionMode,
			excludeNameRegex: this.formExcludeName,
			excludeServers: this.formExcludeServers,
			includeNameRegex: this.formIncludeName,
			...(this.formUrl.trim() ? { url: this.formUrl.trim() } : {}),
		};

		this.busy.set(true);
if (this.editingId == null) {
				this.svc
					.create({ ...body, url: this.formUrl.trim() })
					.pipe(
						concatMap((source) => this.svc.refresh(source.id)),
						finalize(() => this.busy.set(false)),
					)
					.subscribe({
						next: (res) => {
							this.showForm.set(false);
							void this.dialog.success(this.formatRefreshMsg(res, '已创建并拉取完成'));
							this.reload();
						},
						error: (err: Error) => {
							// 源可能已入库但拉取失败
							this.showForm.set(false);
							void this.dialog.error(
								`自动拉取失败：${err.message}\n若列表中已出现该源，可稍后点「拉取」重试。`,
							);
							this.reload();
						},
					});
				return;
			}

		this.svc
			.update(this.editingId, body)
			.pipe(finalize(() => this.busy.set(false)))
			.subscribe({
				next: (src) => {
					this.showForm.set(false);
					void this.dialog.success(`已更新「${src.name}」`);
					this.reload();
				},
				error: (err: Error) => void this.dialog.error(err.message),
			});
	}

refresh(item: SubscriptionSource): void {
			this.busy.set(true);
			this.svc.refresh(item.id).subscribe({
				next: (res) => {
					this.busy.set(false);
					void this.dialog.success(this.formatRefreshMsg(res, '刷新成功'));
					this.reload();
					if (this.showProxies() && this.proxySource()?.id === item.id) {
						this.loadProxies(item);
					}
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

private formatRefreshMsg(
			res: {
				source: SubscriptionSource;
				upstreamTotal?: number;
				parsed?: number;
				added: number;
				skipped: number;
				parseDropped?: Record<string, number>;
				filterDropped?: Record<string, number>;
				filteredNames?: string[];
				regionCounts?: Record<string, number>;
			},
			title: string,
		): string {
			const regions = res.regionCounts
				? Object.entries(res.regionCounts)
						.sort((a, b) => a[0].localeCompare(b[0]))
						.map(([k, v]) => `${k}:${v}`)
						.join(' ')
				: '';
			const up = res.upstreamTotal ?? res.added + (res.skipped ?? 0);
			const parsed = res.parsed ?? up;
			const lines = [
				`${res.source.name} ${title}`,
				`上游 ${up} → 解析 ${parsed} → 入库 ${res.added}（过滤 ${res.skipped ?? 0}）`,
			];
			const parseDrop = this.formatDropMap(res.parseDropped, {
				missing_name: '缺名称',
				missing_type: '缺type',
				missing_server: '缺server',
				invalid_port: '端口无效',
				duplicate_name: '重名',
				not_object: '格式异常',
			});
			if (parseDrop) lines.push(`解析丢弃 ${parseDrop}`);
			const filterDrop = this.formatDropMap(res.filterDropped, {
				'name excluded': '名称排除',
				'name not in include list': '不在白名单',
				info_node: '信息节点',
			});
			if (filterDrop) lines.push(`过滤 ${filterDrop}`);
const names = res.filteredNames || [];
				if (names.length) {
					// 完整列出（与后端日志一致）；名称本身含「过滤掉N条」时是机场信息节点，不是本系统又滤了 N 条
					lines.push(`过滤明细（共 ${names.length} 条，完整）：`);
					for (const n of names) {
						lines.push(`· ${n}`);
					}
				}
			if (regions) lines.push(`地区 ${regions}`);
			return lines.join('\n');
		}

		private formatDropMap(
			m: Record<string, number> | undefined,
			labels: Record<string, string>,
		): string {
			if (!m) return '';
			return Object.entries(m)
				.filter(([, n]) => n > 0)
				.sort((a, b) => b[1] - a[1])
				.map(([k, n]) => {
					const label =
						labels[k] ||
						(k.startsWith('server blocked') ? 'server黑名单' : k);
					return `${label}:${n}`;
				})
				.join(' ');
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
			this.proxySearch.set('');
			this.proxyRegionFilter.set('');
			this.proxyQualityFilter.set('');
			this.lastRefreshSummary.set('');
			this.showProxies.set(true);
			this.loadProxies(item);
		}

	closeProxies(): void {
		this.showProxies.set(false);
		this.proxySource.set(null);
		this.proxies.set([]);
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
				this.proxies.update((list) => list.map((x) => (x.id === updated.id ? { ...x, enabled: updated.enabled } : x)));
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

	/** 是否有上游 Subscription-Userinfo */
	hasTraffic(item: SubscriptionSource): boolean {
		return (
			(item.trafficUpload ?? 0) > 0 ||
			(item.trafficDownload ?? 0) > 0 ||
			(item.trafficTotal ?? 0) > 0 ||
			(item.trafficExpire ?? 0) > 0
		);
	}

	trafficUsed(item: SubscriptionSource): number {
		return Math.max(0, item.trafficUpload ?? 0) + Math.max(0, item.trafficDownload ?? 0);
	}

	/** 已用/总量，如 12.3 GB / 100 GB；无总量时只显示已用 */
	trafficText(item: SubscriptionSource): string {
		if (!this.hasTraffic(item)) return '';
		const used = this.trafficUsed(item);
		const total = item.trafficTotal ?? 0;
		if (total > 0) {
			return `${this.formatBytes(used)} / ${this.formatBytes(total)}`;
		}
		if (used > 0) {
			return `已用 ${this.formatBytes(used)}`;
		}
		return '';
	}

	trafficPercent(item: SubscriptionSource): number | null {
		const total = item.trafficTotal ?? 0;
		if (total <= 0) return null;
		return Math.min(100, Math.round((this.trafficUsed(item) / total) * 100));
	}

	trafficExpireText(item: SubscriptionSource): string {
		const exp = item.trafficExpire ?? 0;
		if (exp <= 0) return '';
		const d = new Date(exp * 1000);
		if (Number.isNaN(d.getTime())) return '';
		const y = d.getFullYear();
		const m = String(d.getMonth() + 1).padStart(2, '0');
		const day = String(d.getDate()).padStart(2, '0');
		const now = Date.now();
		if (d.getTime() < now) {
			return `已过期 ${y}-${m}-${day}`;
		}
		return `到期 ${y}-${m}-${day}`;
	}

	trafficTitle(item: SubscriptionSource): string {
		if (!this.hasTraffic(item)) return '上游未返回 Subscription-Userinfo';
		const parts = [
			`上传 ${this.formatBytes(item.trafficUpload ?? 0)}`,
			`下载 ${this.formatBytes(item.trafficDownload ?? 0)}`,
			`总量 ${item.trafficTotal ? this.formatBytes(item.trafficTotal) : '未知'}`,
		];
		const exp = this.trafficExpireText(item);
		if (exp) parts.push(exp);
		parts.push('（来自上游响应头，拉取时更新）');
		return parts.join('\n');
	}

	private formatBytes(n: number): string {
		if (!Number.isFinite(n) || n < 0) n = 0;
		const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
		let v = n;
		let i = 0;
		while (v >= 1024 && i < units.length - 1) {
			v /= 1024;
			i++;
		}
		const digits = i === 0 ? 0 : v >= 100 ? 0 : v >= 10 ? 1 : 2;
		return `${v.toFixed(digits)} ${units[i]}`;
	}
}

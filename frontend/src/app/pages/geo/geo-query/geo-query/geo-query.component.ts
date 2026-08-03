import { Component, OnInit, ViewChild, inject, signal } from '@angular/core';
import { DialogService } from '@common/services/dialog.service';
import {
	BADGE_ERR,
	BADGE_MUTED,
	BADGE_OK,
	BADGE_WARN,
	GeoCategoriesResponse,
	GeoStatus,
	GeoUpdateResponse,
} from '@data-struct';
import { CmParentComponent } from '@common/parents/parent/parent.component';
import { takeUntil } from 'rxjs';
import { GeoService } from '../../services/geo.service';
import { GeoCategorySearchComponent } from '../geo-category-search/geo-category-search.component';
import { GeoDomainQueryComponent } from '../geo-domain-query/geo-domain-query.component';

@Component({
	selector: 'app-geo-query',
	templateUrl: './geo-query.component.html',
	standalone: false,
})
export class GeoQueryComponent extends CmParentComponent implements OnInit {
	private readonly svc = inject(GeoService);
	private readonly dialog = inject(DialogService);

	@ViewChild(GeoDomainQueryComponent) private domainQuery?: GeoDomainQueryComponent;
	@ViewChild(GeoCategorySearchComponent) private categorySearch?: GeoCategorySearchComponent;

	readonly statuses = signal<GeoStatus[]>([]);
	readonly categories = signal<GeoCategoriesResponse>({
		geosite: [],
		geoip: [],
		metadb: { file: 'geoip.metadb', supportsReverse: false },
		asn: { file: 'GeoLite2-ASN.mmdb', supportsReverse: false },
	});
	readonly updating = signal(false);

	readonly badgeOk = BADGE_OK;
	readonly badgeErr = BADGE_ERR;
	readonly badgeMuted = BADGE_MUTED;
	readonly badgeWarn = BADGE_WARN;

	/** 查询区 Tab：域名查询 | 分类搜索 */
	queryTab = signal<'domain' | 'search'>('domain');

	override ngOnInit(): void {
		super.ngOnInit();
		this.reloadMeta();
	}

	setQueryTab(tab: 'domain' | 'search'): void {
		this.queryTab.set(tab);
	}

	reloadMeta(): void {
		this.svc
			.status()
			.pipe(takeUntil(this.$destroy))
			.subscribe({
				next: (items) => this.statuses.set(items || []),
				error: (err: Error) => void this.dialog.error(err.message),
			});
		this.svc
			.categories()
			.pipe(takeUntil(this.$destroy))
			.subscribe({
				next: (items) => this.categories.set(items),
				error: (err: Error) => void this.dialog.error(err.message),
			});
	}

	resourceName(name: string): string {
		return (
			(
				{
					'geoip.dat': 'GeoIP',
					'geosite.dat': 'GeoSite',
					'geoip.metadb': 'MetaDB',
					'GeoLite2-ASN.mmdb': 'ASN',
				} as Record<string, string>
			)[name] || name
		);
	}

	/** 第三行：版本/库类型合并展示，避免 version 与 databaseType+build 重复 */
	statusDetail(item: GeoStatus): string {
		const type = String(item.databaseType || '').trim();
		const build = item.buildEpoch ? String(item.buildEpoch) : '';
		if (type && build) return `${type} · build ${build}`;
		if (type) return type;
		if (build) return `build ${build}`;
		const ver = String(item.version || '').trim();
		return ver ? `版本 ${ver}` : '版本 —';
	}

	statusDetailTitle(item: GeoStatus): string {
		const parts = [this.statusDetail(item), item.sha256 ? `sha256 ${item.sha256}` : ''].filter(Boolean);
		return parts.join('\n');
	}

	formatBytes(size: number): string {
		if (!size) return '—';
		if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
		return `${(size / 1024 / 1024).toFixed(1)} MB`;
	}

	update(): void {
		this.updating.set(true);
		this.svc
			.update()
			.pipe(takeUntil(this.$destroy))
			.subscribe({
				next: (result: GeoUpdateResponse) => {
					this.updating.set(false);
					const failed = (result.items || []).filter((item) => !item.updated);
					const lines = (result.items || []).map(
						(item) => `${this.resourceName(item.name)}：${item.updated ? '已更新' : item.error || '失败'}`,
					);
					void this.dialog[failed.length ? 'error' : 'success'](lines.join('\n'));
					this.domainQuery?.clearResult();
					this.categorySearch?.clearResult();
					this.reloadMeta();
				},
				error: (err: Error) => {
					this.updating.set(false);
					void this.dialog.error(err.message);
				},
			});
	}

	/** 域名查询结果 → 跳到分类搜索并打开该 GeoSite 分类 */
	onOpenCategory(category: string): void {
		this.queryTab.set('search');
		// 等 *ngIf 挂上分类搜索子组件后再调用
		setTimeout(() => this.categorySearch?.openGeoSiteCategory(category), 0);
	}
}

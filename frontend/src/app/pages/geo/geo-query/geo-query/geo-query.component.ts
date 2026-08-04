import { Component, OnInit, ViewChild, effect, inject, signal } from '@angular/core';
import { DialogService } from '@common/services/dialog.service';
import {
	BADGE_ERR,
	BADGE_MUTED,
	BADGE_OK,
	BADGE_WARN,
	GeoCategoriesResponse,
	GeoStatus,
} from '@data-struct';
import { CmParentComponent } from '@common/parents/parent/parent.component';
import { formatDateTime } from '@common/util/format';
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
		/** 后台更新状态来自 GeoService，跨路由共享 */
		readonly updating = this.svc.updating;

		readonly badgeOk = BADGE_OK;
		readonly badgeErr = BADGE_ERR;
		readonly badgeMuted = BADGE_MUTED;
		readonly badgeWarn = BADGE_WARN;

			/** 查询区 Tab：域名/IP 查询 | 分类搜索 */
			queryTab = signal<'domain' | 'search'>('domain');
			formatTime = formatDateTime;
			private wasUpdating = false;

			constructor() {
				super();
				// 后台更新结束后若仍在本页，自动刷新状态卡上的 modifiedAt
				effect(() => {
					const busy = this.updating();
					if (this.wasUpdating && !busy) {
						this.reloadMeta();
					}
					this.wasUpdating = busy;
				});
			}

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
			const parts = [
				item.modifiedAt ? `更新于 ${this.formatTime(item.modifiedAt)}` : '',
				this.statusDetail(item),
				item.sha256 ? `sha256 ${item.sha256}` : '',
			].filter(Boolean);
			return parts.join('\n');
		}

	formatBytes(size: number): string {
		if (!size) return '—';
		if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
		return `${(size / 1024 / 1024).toFixed(1)} MB`;
	}

				async update(): Promise<void> {
					if (this.updating()) return;
					const ok = await this.dialog.confirm(
						'将从上游下载并覆盖 defaults/geo 下的 Geo 数据文件。更新在后台进行，切换页面不会中断；完成后以通知提示结果。',
						'更新 Geo 数据',
						'开始更新',
					);
					if (!ok) return;

					if (!this.svc.startBackgroundUpdate()) return;

					// 本地查询缓存可能基于旧文件，先清掉；状态等你再回来/点刷新即可
					this.domainQuery?.clearResult();
					this.categorySearch?.clearResult();
					void this.dialog.success('已开始后台更新 Geo 数据');
				}

}

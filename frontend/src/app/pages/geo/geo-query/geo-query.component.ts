import { Component, OnDestroy, OnInit, inject, signal } from '@angular/core';
import { MatDialogRef } from '@angular/material/dialog';
import { DialogService } from '@common/services/dialog.service';
import {
	BADGE_ERR,
	BADGE_MUTED,
	BADGE_OK,
	BADGE_WARN,
	GeoCategoriesResponse,
	GeoCategory,
	GeoEntryRow,
	GeoQueryResponse,
	GeoReverseResponse,
	GeoSearchResponse,
	GeoStatus,
	GeoUpdateResponse,
} from '@data-struct';
import { GeoService } from '../services/geo.service';
import { CM_DIALOG_WIDTH, CmDialogOpenService } from '@common/modules/dialog';
import { CmParentComponent } from '@common/parents/parent/parent.component';
import { takeUntil } from 'rxjs';
import {
	GeoEntriesComponent,
	GeoEntriesDialogData,
	GeoEntriesDialogResult,
} from '../geo-entries/geo-entries.component';

@Component({
	selector: 'app-geo-query',
	templateUrl: './geo-query.component.html',
	standalone: false,
})
export class GeoQueryComponent extends CmParentComponent implements OnInit, OnDestroy {
	private readonly svc = inject(GeoService);
	private readonly dialog = inject(DialogService);
	private readonly dialogOpen = inject(CmDialogOpenService);

	readonly statuses = signal<GeoStatus[]>([]);
	readonly categories = signal<GeoCategoriesResponse>({
		geosite: [],
		geoip: [],
		metadb: { file: 'geoip.metadb', supportsReverse: false },
		asn: { file: 'GeoLite2-ASN.mmdb', supportsReverse: false },
	});
	readonly queryResult = signal<GeoQueryResponse | null>(null);
	readonly reverseResult = signal<GeoReverseResponse | null>(null);
	readonly loading = signal(false);
	readonly updating = signal(false);
	readonly reverseLoading = signal(false);
	readonly searchLoading = signal(false);
	readonly entriesLoading = signal(false);

	readonly badgeOk = BADGE_OK;
	readonly badgeErr = BADGE_ERR;
	readonly badgeMuted = BADGE_MUTED;
	readonly badgeWarn = BADGE_WARN;

	domain = '';
	resolve = false;
	reverseFile = 'geosite';
	reverseKeyword = '';
	searchField = 'asn';
	searchKeyword = '';
	private reverseCommittedKeyword = '';
	private reverseKeywordTimer: ReturnType<typeof setTimeout> | null = null;
	reverseCategory = '';
	reverseOffset = 0;
	entriesLimit = 100;

	private entriesMode: 'reverse' | 'search' = 'reverse';
	private searchContextFile = '';
	private searchContextField = '';
	private searchContextKeyword = '';
	private entriesTotal = 0;
	private entriesDialogRef: MatDialogRef<GeoEntriesComponent, GeoEntriesDialogResult> | null = null;

	override ngOnInit(): void {
		super.ngOnInit();
		this.reloadMeta();
	}

	override ngOnDestroy(): void {
		if (this.reverseKeywordTimer) clearTimeout(this.reverseKeywordTimer);
		this.entriesDialogRef?.close(null);
		super.ngOnDestroy();
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
				next: (items) => {
					this.categories.set(items);
					this.syncReverseCategory();
				},
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

	formatBytes(size: number): string {
		if (!size) return '—';
		if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
		return `${(size / 1024 / 1024).toFixed(1)} MB`;
	}

	query(): void {
		if (!this.domain.trim()) {
			void this.dialog.error('请输入域名');
			return;
		}
		this.loading.set(true);
		this.svc
			.query(this.domain, this.resolve)
			.pipe(takeUntil(this.$destroy))
			.subscribe({
				next: (result) => {
					this.queryResult.set(result);
					this.loading.set(false);
				},
				error: (err: Error) => {
					this.loading.set(false);
					void this.dialog.error(err.message);
				},
			});
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
					this.queryResult.set(null);
					this.closeEntries();
					this.reloadMeta();
				},
				error: (err: Error) => {
					this.updating.set(false);
					void this.dialog.error(err.message);
				},
			});
	}

	openGeoSiteCategoryEntries(category: string): void {
		this.reverseFile = 'geosite';
		this.reverseCategory = category;
		this.reverseOffset = 0;
		this.reverseResult.set(null);
		this.entriesLimit = 100;
		this.reverse();
	}

	openDatabaseSearch(): void {
		if (!this.searchKeyword.trim()) {
			void this.dialog.error('请输入搜索关键词');
			return;
		}
		this.searchContextFile = this.reverseFile;
		this.searchContextField = this.searchField.trim().toLowerCase();
		this.searchContextKeyword = this.searchKeyword.trim();
		this.reverseOffset = 0;
		this.entriesLimit = 100;
		this.entriesMode = 'search';
		this.loadDatabaseSearch(true);
	}

	private loadDatabaseSearch(openDialog = false): void {
		this.searchLoading.set(true);
		this.entriesLoading.set(true);
		this.svc
			.search(
				this.searchContextFile,
				this.searchContextField,
				this.searchContextKeyword,
				this.entriesLimit,
				this.reverseOffset,
			)
			.pipe(takeUntil(this.$destroy))
			.subscribe({
				next: (result: GeoSearchResponse) => {
					this.searchLoading.set(false);
					this.entriesLoading.set(false);
					const title = `${this.searchContextFile === 'asn' ? 'ASN' : 'MetaDB'} · ${this.searchContextKeyword}`;
					const subtitle =
						result.message ||
						`共 ${result.total} 条，当前第 ${result.offset + 1}-${result.offset + (result.items?.length || 0)} 条`;
					this.openOrUpdateEntries({
						title,
						subtitle,
						items: result.items || [],
						total: result.total,
						offset: result.offset,
						limit: result.limit,
						loading: false,
						paginated: true,
					}, openDialog);
				},
				error: (err: Error) => {
					this.searchLoading.set(false);
					this.entriesLoading.set(false);
					void this.dialog.error(err.message);
				},
			});
	}

	closeEntries(): void {
		this.entriesDialogRef?.close(null);
		this.entriesDialogRef = null;
	}

	categoriesForFile(): GeoCategory[] {
		return this.reverseFile === 'geoip' ? this.categories().geoip : this.categories().geosite;
	}

	filteredCategories(): GeoCategory[] {
		const keyword = this.reverseCommittedKeyword.trim().toLowerCase();
		if (!keyword) return this.categoriesForFile();
		return this.categoriesForFile().filter((item) => item.name.toLowerCase().includes(keyword));
	}

	commitReverseKeyword(): void {
		if (this.reverseKeywordTimer) clearTimeout(this.reverseKeywordTimer);
		this.reverseKeywordTimer = setTimeout(() => {
			this.reverseCommittedKeyword = this.reverseKeyword;
			this.reverseCategory = '';
			this.reverseOffset = 0;
			this.reverseResult.set(null);
			this.reverseKeywordTimer = null;
		}, 400);
	}

	selectReverseCategory(category: GeoCategory): void {
		this.reverseCategory = category.name;
		this.reverseOffset = 0;
		this.reverseResult.set(null);
	}

	syncReverseCategory(): void {
		const available = this.categoriesForFile();
		if (!available.some((item) => item.name === this.reverseCategory)) this.reverseCategory = '';
	}

	changeReverseFile(file: string): void {
		this.reverseFile = file;
		if (this.reverseKeywordTimer) {
			clearTimeout(this.reverseKeywordTimer);
			this.reverseKeywordTimer = null;
		}
		this.reverseKeyword = '';
		this.reverseCommittedKeyword = '';
		this.searchField = file === 'asn' ? 'asn' : file === 'metadb' ? 'code' : this.searchField;
		this.searchKeyword = '';
		this.reverseOffset = 0;
		this.reverseResult.set(null);
		this.syncReverseCategory();
	}

	reverse(): void {
		if (this.reverseFile !== 'geosite' && this.reverseFile !== 'geoip') {
			this.reverseResult.set({
				file: this.reverseFile,
				category: '',
				total: 0,
				limit: 100,
				offset: 0,
				message: '该文件不保存域名分类，无法反查域名',
			});
			return;
		}
		if (!this.reverseCategory) {
			void this.dialog.error('请选择分类');
			return;
		}
		this.reverseLoading.set(true);
		this.entriesLoading.set(true);
		this.entriesMode = 'reverse';
		this.svc
			.reverse(this.reverseFile, this.reverseCategory, this.entriesLimit, this.reverseOffset)
			.pipe(takeUntil(this.$destroy))
			.subscribe({
				next: (result) => {
					this.reverseResult.set(result);
					this.reverseLoading.set(false);
					this.entriesLoading.set(false);
					this.openReverseEntries(result, true);
				},
				error: (err: Error) => {
					this.reverseLoading.set(false);
					this.entriesLoading.set(false);
					void this.dialog.error(err.message);
				},
			});
	}

	openReverseEntries(result: GeoReverseResponse, openDialog = true): void {
		const fileName = this.reverseFile === 'geosite' ? 'GeoSite' : 'GeoIP';
		const items: GeoEntryRow[] = (result.items || []).map((item) => ({
			type: item.type,
			value: item.value,
		}));
		this.openOrUpdateEntries(
			{
				title: `${fileName} · ${result.category}`,
				subtitle: `共 ${result.total} 条，当前第 ${result.offset + 1}-${result.offset + (result.items?.length || 0)} 条`,
				items,
				total: result.total,
				offset: result.offset,
				limit: result.limit,
				loading: false,
				paginated: true,
			},
			openDialog,
		);
	}

	previousReverse(): void {
		const pageSize = this.entriesLimit;
		this.reverseOffset = Math.max(0, this.reverseOffset - pageSize);
		if (this.entriesMode === 'search') {
			this.loadDatabaseSearch(true);
			return;
		}
		this.reverse();
	}

	nextReverse(): void {
		const pageSize = this.entriesLimit;
		if (this.reverseOffset + pageSize >= this.entriesTotal) return;
		this.reverseOffset += pageSize;
		if (this.entriesMode === 'search') {
			this.loadDatabaseSearch(true);
			return;
		}
		this.reverse();
	}

	private openOrUpdateEntries(data: GeoEntriesDialogData, forceOpen: boolean): void {
		this.entriesTotal = data.total || 0;
		// MatDialog data is fixed at open; re-open with fresh payload for paging.
		const prev = this.entriesDialogRef;
		this.entriesDialogRef = null;
		if (prev) {
			// Close without treating as user dismiss — detach silently then open next page.
			prev.close(null);
		}
		if (!forceOpen && !data.items.length && !data.loading) return;

		const ref = this.dialogOpen.openContent(GeoEntriesComponent, data, {
			width: CM_DIALOG_WIDTH.large,
		});
		this.entriesDialogRef = ref;
		ref.afterClosed().subscribe((result) => {
			// Ignore close from page-refresh replace (entriesDialogRef already nulled/replaced).
			if (this.entriesDialogRef !== ref) return;
			this.entriesDialogRef = null;
			if (result === 'previous') this.previousReverse();
			else if (result === 'next') this.nextReverse();
		});
	}
}

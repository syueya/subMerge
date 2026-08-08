import { Component, Input, OnChanges, OnDestroy, SimpleChanges, inject, signal } from '@angular/core';
import { MatChipListboxChange } from '@angular/material/chips';
import { MatDialogRef } from '@angular/material/dialog';
import { DialogService } from '@common/services/dialog.service';
import { GeoCategoriesResponse, GeoCategory, GeoEntriesDialogData, GeoEntriesDialogResult, GeoEntryRow, GeoReverseResponse, GeoSearchResponse } from '@data-struct';
import { CM_DIALOG_WIDTH, CmDialogOpenService } from '@common/modules/dialog';
import { CmParentComponent } from '@common/parents/parent/parent.component';
import { takeUntil } from 'rxjs';
import { GeoEntriesComponent } from '../../geo-entries/geo-entries.component';
import { GeoService } from '../../services/geo.service';
import { RuleCreateService } from '../../../_shared/rule-create.service';

@Component({
  selector: 'app-geo-category-search',
  templateUrl: './geo-category-search.component.html',
  standalone: false
})
export class GeoCategorySearchComponent extends CmParentComponent implements OnChanges, OnDestroy {
  private readonly svc = inject(GeoService);
  private readonly dialog = inject(DialogService);
  private readonly dialogOpen = inject(CmDialogOpenService);
  private readonly ruleCreate = inject(RuleCreateService);

  /** 由父组件加载后的分类元数据 */
  @Input() categories: GeoCategoriesResponse = {
    geosite: [],
    geoip: [],
    metadb: { file: 'geoip.metadb', supportsReverse: false },
    asn: { file: 'GeoLite2-ASN.mmdb', supportsReverse: false }
  };

  readonly reverseResult = signal<GeoReverseResponse | null>(null);
  readonly reverseLoading = signal(false);
  readonly searchLoading = signal(false);
  readonly entriesLoading = signal(false);

  readonly categoryVisibleLimit = 30;
  /** 条目弹窗默认每页条数 */
  readonly defaultEntriesLimit = 50;

  reverseFile = 'geosite';
  reverseKeyword = '';
  searchField = 'asn';
  searchKeyword = '';
  /** 防抖后的分类关键词；模板里用于判断是否已过滤 */
  protected reverseCommittedKeyword = '';
  private reverseKeywordTimer: ReturnType<typeof setTimeout> | null = null;
  reverseCategory = '';
  reverseOffset = 0;
  entriesLimit = this.defaultEntriesLimit;

  private entriesMode: 'reverse' | 'search' = 'reverse';
  private searchContextFile = '';
  private searchContextField = '';
  private searchContextKeyword = '';
  private entriesDialogRef: MatDialogRef<GeoEntriesComponent, GeoEntriesDialogResult> | null = null;

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['categories']) {
      this.syncReverseCategory();
    }
  }

  override ngOnDestroy(): void {
    if (this.reverseKeywordTimer) clearTimeout(this.reverseKeywordTimer);
    this.entriesDialogRef?.close(null);
    super.ngOnDestroy();
  }

  clearResult(): void {
    this.reverseResult.set(null);
    this.closeEntries();
  }

  categoriesForFile(): GeoCategory[] {
    return this.reverseFile === 'geoip' ? this.categories.geoip : this.categories.geosite;
  }

  filteredCategories(): GeoCategory[] {
    const keyword = this.reverseCommittedKeyword.trim().toLowerCase();
    if (!keyword) return this.categoriesForFile();
    return this.categoriesForFile().filter(item => item.name.toLowerCase().includes(keyword));
  }

  visibleCategories(): GeoCategory[] {
    return this.filteredCategories().slice(0, this.categoryVisibleLimit);
  }

  categoryTotalCount(): number {
    return this.categoriesForFile().length;
  }

  categoryMatchCount(): number {
    return this.filteredCategories().length;
  }

  categoryTruncated(): boolean {
    return this.categoryMatchCount() > this.categoryVisibleLimit;
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

  onCategoryChipChange(ev: MatChipListboxChange): void {
    const name = String(ev.value || '');
    this.reverseCategory = name;
    this.reverseOffset = 0;
    this.reverseResult.set(null);
  }

  syncReverseCategory(): void {
    const available = this.categoriesForFile();
    if (!available.some(item => item.name === this.reverseCategory)) this.reverseCategory = '';
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
        limit: this.entriesLimit,
        offset: 0,
        message: '该文件不保存域名分类，无法反查域名'
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
        next: result => {
          this.reverseResult.set(result);
          this.reverseLoading.set(false);
          this.entriesLoading.set(false);
          this.openReverseEntries(result, true);
        },
        error: (err: Error) => {
          this.reverseLoading.set(false);
          this.entriesLoading.set(false);
          void this.dialog.error(err.message);
        }
      });
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
    this.entriesLimit = this.defaultEntriesLimit;
    this.entriesMode = 'search';
    this.loadDatabaseSearch(true);
  }

  openReverseEntries(result: GeoReverseResponse, openDialog = true): void {
    const fileName = this.reverseFile === 'geosite' ? 'GeoSite' : 'GeoIP';
    const items: GeoEntryRow[] = (result.items || []).map(item => ({
      type: item.type,
      value: item.value
    }));
    this.openOrUpdateEntries(
      {
        title: `${fileName} · ${result.category}`,
        subtitle: `共 ${result.total} 条`,
        items,
        total: result.total,
        offset: result.offset,
        limit: result.limit || this.entriesLimit,
        loading: false,
        paginated: true,
        addRule:
          this.reverseFile === 'geosite' || this.reverseFile === 'geoip' ? { type: this.reverseFile === 'geosite' ? 'GEOSITE' : 'GEOIP', payload: result.category } : undefined
      },
      openDialog
    );
  }

  private loadDatabaseSearch(openDialog = false): void {
    this.searchLoading.set(true);
    this.entriesLoading.set(true);
    this.markEntriesLoading();
    this.svc
      .search(this.searchContextFile, this.searchContextField, this.searchContextKeyword, this.entriesLimit, this.reverseOffset)
      .pipe(takeUntil(this.$destroy))
      .subscribe({
        next: (result: GeoSearchResponse) => {
          this.searchLoading.set(false);
          this.entriesLoading.set(false);
          const title = `${this.searchContextFile === 'asn' ? 'ASN' : 'MetaDB'} · ${this.searchContextKeyword}`;
          const subtitle = result.message || `共 ${result.total} 条`;
          this.openOrUpdateEntries(
            {
              title,
              subtitle,
              items: result.items || [],
              total: result.total,
              offset: result.offset,
              limit: result.limit || this.entriesLimit,
              loading: false,
              paginated: true,
              addRule: undefined
            },
            openDialog
          );
        },
        error: (err: Error) => {
          this.searchLoading.set(false);
          this.entriesLoading.set(false);
          this.markEntriesLoading(false);
          void this.dialog.error(err.message);
        }
      });
  }

  private closeEntries(): void {
    this.entriesDialogRef?.close(null);
    this.entriesDialogRef = null;
  }

  /** mat-paginator 翻页 / 改每页条数 */
  private onEntriesPage(pageIndex: number, pageSize: number): void {
    this.entriesLimit = pageSize || this.defaultEntriesLimit;
    this.reverseOffset = pageIndex * this.entriesLimit;
    if (this.entriesMode === 'search') {
      this.loadDatabaseSearch(false);
      return;
    }
    this.reloadReversePage();
  }

  /** 已打开的条目弹窗显示 loading，不关窗 */
  private markEntriesLoading(loading = true): void {
    const ref = this.entriesDialogRef;
    if (!ref) return;
    const inst = ref.componentInstance;
    inst.applyData({
      title: inst.title,
      subtitle: inst.subtitle,
      items: inst.items,
      total: inst.total,
      offset: this.reverseOffset,
      limit: this.entriesLimit,
      loading,
      paginated: true,
      addRule: this.entriesMode === 'reverse' ? { type: this.reverseFile === 'geosite' ? 'GEOSITE' : 'GEOIP', payload: this.reverseCategory } : undefined
    });
  }

  private reloadReversePage(): void {
    if (!this.reverseCategory) return;
    this.entriesLoading.set(true);
    this.markEntriesLoading(true);
    this.svc
      .reverse(this.reverseFile, this.reverseCategory, this.entriesLimit, this.reverseOffset)
      .pipe(takeUntil(this.$destroy))
      .subscribe({
        next: result => {
          this.reverseResult.set(result);
          this.entriesLoading.set(false);
          this.openReverseEntries(result, false);
        },
        error: (err: Error) => {
          this.entriesLoading.set(false);
          this.markEntriesLoading(false);
          void this.dialog.error(err.message);
        }
      });
  }

  private openOrUpdateEntries(data: GeoEntriesDialogData, forceOpen: boolean): void {
    const payload: GeoEntriesDialogData = {
      ...data,
      onPage: (pageIndex, pageSize) => this.onEntriesPage(pageIndex, pageSize)
    };

    if (this.entriesDialogRef) {
      this.entriesDialogRef.componentInstance.applyData(payload);
      return;
    }
    if (!forceOpen && !payload.items.length && !payload.loading) return;

    const ref = this.dialogOpen.openContent(GeoEntriesComponent, payload, {
      width: CM_DIALOG_WIDTH.large
    });
    this.entriesDialogRef = ref;
    ref.afterClosed().subscribe((result: GeoEntriesDialogResult) => {
      if (this.entriesDialogRef === ref) {
        this.entriesDialogRef = null;
      }
      if (result?.action === 'add') {
        this.ruleCreate.open(result.context);
      }
    });
  }
}

import { Component, Input, OnChanges, OnDestroy, SimpleChanges, inject, signal } from '@angular/core';
import { MatChipListboxChange } from '@angular/material/chips';
import { CmParentComponent } from '@common/parents/parent/parent.component';
import { DialogService } from '@common/services/dialog.service';
import { GeoCategoriesResponse, GeoCategory, GeoEntryRow, GeoReverseResponse, GeoSearchResponse } from '@data-struct';
import { takeUntil } from 'rxjs';

import { RuleCreateService } from '../../../_shared/rule-create.service';
import { GeoService } from '../../services/geo.service';
import { GeoEntriesDialogHelper } from '../geo-entries-dialog.helper';

@Component({
  selector: 'app-geo-category-search',
  templateUrl: './geo-category-search.component.html',
  standalone: false
})
export class GeoCategorySearchComponent extends CmParentComponent implements OnChanges, OnDestroy {
  private readonly svc = inject(GeoService);
  private readonly dialog = inject(DialogService);
  private readonly ruleCreate = inject(RuleCreateService);
  private readonly entriesDialog = inject(GeoEntriesDialogHelper);

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

  /** 当前条目弹窗的 addRule（reverse 模式按分类，search 模式无） */
  private currentAddRule(): { type: 'GEOSITE' | 'GEOIP'; payload: string } | undefined {
    return this.entriesMode === 'reverse'
      ? { type: this.reverseFile === 'geosite' ? 'GEOSITE' : 'GEOIP', payload: this.reverseCategory }
      : undefined;
  }

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['categories']) {
      this.syncReverseCategory();
    }
  }

  override ngOnDestroy(): void {
    if (this.reverseKeywordTimer) clearTimeout(this.reverseKeywordTimer);
    this.entriesDialog.close();
    super.ngOnDestroy();
  }

  clearResult(): void {
    this.reverseResult.set(null);
    this.entriesDialog.close();
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
    this.entriesDialog.openOrUpdate({
      data: {
        title: `${fileName} · ${result.category}`,
        subtitle: `共 ${result.total} 条`,
        items,
        total: result.total,
        offset: result.offset,
        limit: result.limit || this.entriesLimit,
        loading: false,
        paginated: true,
        addRule: this.currentAddRule()
      },
      forceOpen: openDialog,
      onPage: (pageIndex, pageSize) => this.onEntriesPage(pageIndex, pageSize)
    });
  }

  private loadDatabaseSearch(openDialog = false): void {
    this.searchLoading.set(true);
    this.entriesLoading.set(true);
    this.entriesDialog.markLoading(true, () => ({
      title: '',
      subtitle: '',
      items: [],
      total: 0,
      offset: this.reverseOffset,
      limit: this.entriesLimit,
      loading: true,
      paginated: true,
      addRule: undefined
    }));
    this.svc
      .search(this.searchContextFile, this.searchContextField, this.searchContextKeyword, this.entriesLimit, this.reverseOffset)
      .pipe(takeUntil(this.$destroy))
      .subscribe({
        next: (result: GeoSearchResponse) => {
          this.searchLoading.set(false);
          this.entriesLoading.set(false);
          const title = `${this.searchContextFile === 'asn' ? 'ASN' : 'MetaDB'} · ${this.searchContextKeyword}`;
          const subtitle = result.message || `共 ${result.total} 条`;
          this.entriesDialog.openOrUpdate({
            data: {
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
            forceOpen: openDialog,
            onPage: (pageIndex, pageSize) => this.onEntriesPage(pageIndex, pageSize)
          });
        },
        error: (err: Error) => {
          this.searchLoading.set(false);
          this.entriesLoading.set(false);
          this.entriesDialog.markLoading(false, () => ({
            title: '',
            subtitle: '',
            items: [],
            total: 0,
            offset: this.reverseOffset,
            limit: this.entriesLimit,
            loading: false,
            paginated: true,
            addRule: undefined
          }));
          void this.dialog.error(err.message);
        }
      });
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

  private reloadReversePage(): void {
    if (!this.reverseCategory) return;
    this.entriesLoading.set(true);
    this.entriesDialog.markLoading(true, () => ({
      title: '',
      subtitle: '',
      items: [],
      total: 0,
      offset: this.reverseOffset,
      limit: this.entriesLimit,
      loading: true,
      paginated: true,
      addRule: this.currentAddRule()
    }));
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
          this.entriesDialog.markLoading(false, () => ({
            title: '',
            subtitle: '',
            items: [],
            total: 0,
            offset: this.reverseOffset,
            limit: this.entriesLimit,
            loading: false,
            paginated: true,
            addRule: this.currentAddRule()
          }));
          void this.dialog.error(err.message);
        }
      });
  }
}

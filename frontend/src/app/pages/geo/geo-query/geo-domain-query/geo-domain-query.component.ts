import { Component, OnDestroy, inject, signal } from '@angular/core';
import { MatDialogRef } from '@angular/material/dialog';
import { DialogService } from '@common/services/dialog.service';
import { CM_DIALOG_WIDTH, CmDialogOpenService } from '@common/modules/dialog';
import { BADGE_MUTED, GeoEntriesDialogData, GeoEntriesDialogResult, GeoEntryRow, GeoIPGeoResponse, GeoQueryResponse, GeoReverseResponse } from '@data-struct';
import { CmParentComponent } from '@common/parents/parent/parent.component';
import { takeUntil, finalize } from 'rxjs';
import { GeoEntriesComponent } from '../../geo-entries/geo-entries.component';
import { GeoService } from '../../services/geo.service';
import { RuleCreateService } from '../../../_shared/rule-create.service';

@Component({
  selector: 'app-geo-domain-query',
  templateUrl: './geo-domain-query.component.html',
  host: { class: 'page-fill__host' },
  standalone: false
})
export class GeoDomainQueryComponent extends CmParentComponent implements OnDestroy {
  private readonly svc = inject(GeoService);
  private readonly dialog = inject(DialogService);
  private readonly dialogOpen = inject(CmDialogOpenService);
  private readonly ruleCreate = inject(RuleCreateService);

  readonly queryResult = signal<GeoQueryResponse | null>(null);
  readonly loading = signal(false);
  readonly ipGeoLoading = signal<Record<string, boolean>>({});
  readonly ipGeoResults = signal<Record<string, GeoIPGeoResponse>>({});
  readonly ipGeoErrors = signal<Record<string, string>>({});

  readonly badgeMuted = BADGE_MUTED;

  queryText = '';
  resolve = true;

  private readonly defaultEntriesLimit = 50;
  private entriesLimit = this.defaultEntriesLimit;
  private entriesOffset = 0;
  private entriesCategory = '';
  private entriesDialogRef: MatDialogRef<GeoEntriesComponent, GeoEntriesDialogResult> | null = null;

  /** True when the current input looks like an IPv4/IPv6 address. */
  get isIPInput(): boolean {
    return this.looksLikeIP(this.queryText);
  }

  override ngOnDestroy(): void {
    this.entriesDialogRef?.close(null);
    super.ngOnDestroy();
  }

  query(): void {
    const value = this.queryText.trim();
    if (!value) {
      void this.dialog.error('请输入域名或 IP');
      return;
    }
    const resolve = this.looksLikeIP(value) ? false : this.resolve;
    this.loading.set(true);
    this.svc
      .query(value, resolve)
      .pipe(takeUntil(this.$destroy))
      .subscribe({
        next: result => {
          this.queryResult.set(result);
          this.ipGeoLoading.set({});
          this.ipGeoResults.set({});
          this.ipGeoErrors.set({});
          this.loading.set(false);
        },
        error: (err: Error) => {
          this.loading.set(false);
          void this.dialog.error(err.message);
        }
      });
  }

  lookupIPGeo(ip: string): void {
    const value = ip.trim();
    if (!value || this.ipGeoLoading()[value]) return;
    this.ipGeoLoading.update(state => ({ ...state, [value]: true }));
    this.ipGeoErrors.update(state => {
      const next = { ...state };
      delete next[value];
      return next;
    });
    this.svc
      .lookupIPGeo(value)
      .pipe(
        takeUntil(this.$destroy),
        finalize(() => this.ipGeoLoading.update(state => ({ ...state, [value]: false })))
      )
      .subscribe({
        next: result => this.ipGeoResults.update(state => ({ ...state, [value]: result })),
        error: (err: Error) => {
          this.ipGeoErrors.update(state => ({ ...state, [value]: err.message || '查询失败' }));
          void this.dialog.error(`${value}：${err.message}`);
        }
      });
  }
  clearResult(): void {
    this.queryResult.set(null);
    this.closeEntries();
  }

  /** 留在当前结果页，直接打开 GeoSite 分类条目弹窗 */
  openGeoSiteCategoryEntries(category: string): void {
    const name = category.trim();
    if (!name) return;
    this.entriesCategory = name;
    this.entriesOffset = 0;
    this.entriesLimit = this.defaultEntriesLimit;
    this.loadCategoryEntries(true);
  }

  isIPResult(result: GeoQueryResponse): boolean {
    return result.inputType === 'ip';
  }

  private loadCategoryEntries(openDialog: boolean): void {
    if (!this.entriesCategory) return;
    if (openDialog) {
      this.openOrUpdateEntries(
        {
          title: `GeoSite · ${this.entriesCategory}`,
          subtitle: '加载中…',
          items: [],
          total: 0,
          offset: this.entriesOffset,
          limit: this.entriesLimit,
          loading: true,
          paginated: true,
          addRule: { type: 'GEOSITE', payload: this.entriesCategory }
        },
        true
      );
    } else {
      this.markEntriesLoading(true);
    }

    this.svc
      .reverse('geosite', this.entriesCategory, this.entriesLimit, this.entriesOffset)
      .pipe(takeUntil(this.$destroy))
      .subscribe({
        next: (result: GeoReverseResponse) => {
          const items: GeoEntryRow[] = (result.items || []).map(item => ({
            type: item.type,
            value: item.value
          }));
          this.openOrUpdateEntries(
            {
              title: `GeoSite · ${result.category || this.entriesCategory}`,
              subtitle: `共 ${result.total} 条`,
              items,
              total: result.total,
              offset: result.offset,
              limit: result.limit || this.entriesLimit,
              loading: false,
              paginated: true,
              addRule: { type: 'GEOSITE', payload: result.category || this.entriesCategory }
            },
            openDialog
          );
        },
        error: (err: Error) => {
          this.markEntriesLoading(false);
          void this.dialog.error(err.message);
        }
      });
  }

  private onEntriesPage(pageIndex: number, pageSize: number): void {
    this.entriesLimit = pageSize || this.defaultEntriesLimit;
    this.entriesOffset = pageIndex * this.entriesLimit;
    this.loadCategoryEntries(false);
  }

  private markEntriesLoading(loading = true): void {
    const ref = this.entriesDialogRef;
    if (!ref) return;
    const inst = ref.componentInstance;
    inst.applyData({
      title: inst.title,
      subtitle: inst.subtitle,
      items: inst.items,
      total: inst.total,
      offset: this.entriesOffset,
      limit: this.entriesLimit,
      loading,
      paginated: true,
      addRule: { type: 'GEOSITE', payload: this.entriesCategory }
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

  private closeEntries(): void {
    this.entriesDialogRef?.close(null);
    this.entriesDialogRef = null;
  }

  private looksLikeIP(raw: string): boolean {
    const value = raw.trim();
    if (!value || value.includes('/') || /\s/.test(value)) {
      return false;
    }
    // IPv4
    if (/^\d{1,3}(\.\d{1,3}){3}$/.test(value)) {
      return value.split('.').every(part => {
        const n = Number(part);
        return Number.isInteger(n) && n >= 0 && n <= 255;
      });
    }
    // IPv6 (simplified: contains colon, valid hex/colon/zone)
    if (value.includes(':')) {
      return /^[0-9a-fA-F:.%]+$/.test(value);
    }
    return false;
  }
}

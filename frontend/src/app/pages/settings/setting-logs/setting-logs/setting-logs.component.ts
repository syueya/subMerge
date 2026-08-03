import { CdkVirtualScrollViewport } from '@angular/cdk/scrolling';
import { HttpClient } from '@angular/common/http';
import { ChangeDetectorRef, Component, DestroyRef, ViewChild, inject } from '@angular/core';
import { FormBuilder, FormControl } from '@angular/forms';
import { HttpRespone, ServiceQueryParams } from '@common/interfaces';
import { withWtHttpCacheBypass } from '@common/net';
import { CmParentTableComponent } from '@common/parents/parent-table/parent-table.component';
import { formatDate } from '@common/util';
import { catchError, debounceTime, distinctUntilChanged, finalize, map, of, takeUntil, timeout } from 'rxjs';

import { SystemLogsTypeData, SystemLogsType, SystemLogsContent, SystemLogs } from '../interfaces/SystemLogs';

@Component({
  standalone: false,
  templateUrl: './setting-logs.component.html',
  styleUrl: './setting-logs.component.scss'
})
export class SettingLogsComponent extends CmParentTableComponent {
  private fb = inject(FormBuilder);
  private httpClient = inject(HttpClient);
  private cdr = inject(ChangeDetectorRef);
  private destroyRef = inject(DestroyRef);
  private viewDestroyed = false;

  sidePanelOpened = true;
  systemLogsTypeList: SystemLogsType[] = [];
  systemLogsList: SystemLogs[] = [];
  selectedType: SystemLogsType | null = null;
  lineList = [50, 100, 200, 500, 1000];
  lineSelected = 100;
  lineFormControl = new FormControl();

  @ViewChild(CdkVirtualScrollViewport) viewport!: CdkVirtualScrollViewport;

  private logTypeRequestId = 0;
  private logRequestId = 0;

  /** 页面局部 loading，不依赖全局 HTTP spinner */
  isLoadingLogs = false;
  logLoadError = false;

  isOver(): boolean {
    return window.matchMedia(`(max-width: 960px)`).matches;
  }

  constructor() {
    super();
    this.destroyRef.onDestroy(() => {
      this.viewDestroyed = true;
    });

    this.searchForm = this.fb.group({
      name: ''
    });

    this.searchForm
      .get('name')
      ?.valueChanges.pipe(takeUntil(this.$destroy), debounceTime(300), distinctUntilChanged())
      .subscribe(value => this.loadLogTypes(value || ''));

    this.lineSelected = Number(localStorage.getItem('wt_log_system_defaultLine')) || 100;
    this.lineFormControl.setValue(this.lineSelected);
    this.getSelectedLine();
  }

  override handlerAfterViewInit() {
    super.handlerAfterViewInit();
    this.loadLogTypes(this.searchForm.get('name')?.value || '');
  }

  override reloadTableData(_query?: ServiceQueryParams, _sort?: unknown, options?: { bypassCache?: boolean }): void {
    const name = this.selectedType?.name || this.systemLogsTypeList[0]?.name;
    if (!name) {
      this.systemLogsList = [];
      this.finishLoading();
      return;
    }

    this.systemLogsList = [];
    this.logLoadError = false;
    this.isLoadingLogs = true;
    const requestId = ++this.logRequestId;

    this.loadLogDetails(name, this.lineSelected, options?.bypassCache === true)
      .pipe(
        takeUntil(this.$destroy),
        timeout(15000),
        catchError(() => {
          if (requestId === this.logRequestId) {
            this.logLoadError = true;
          }
          return of([] as SystemLogs[]);
        }),
        finalize(() => {
          if (requestId === this.logRequestId) {
            this.finishLoading();
          }
        })
      )
      .subscribe(list => {
        if (requestId !== this.logRequestId) {
          return;
        }
        this.systemLogsList = list;
        queueMicrotask(() => this.viewport?.checkViewportSize());
      });
  }

  private loadLogTypes(value: string, options?: { bypassCache?: boolean }) {
    const requestId = ++this.logTypeRequestId;
    // 作废进行中的详情请求，避免旧 finalize 误关/误开 loading
    this.logRequestId++;
    this.logLoadError = false;
    this.isLoadingLogs = true;
    this.systemLogsList = [];

    this.httpClient
      .get<HttpRespone<SystemLogsTypeData>>('/api/v1/logs/list', {
        params: value ? { name: value } : undefined,
        context: options?.bypassCache ? withWtHttpCacheBypass() : undefined
      })
      .pipe(
        takeUntil(this.$destroy),
        timeout(15000),
        map(res => (res.code === 20000 ? res.data?.files || [] : [])),
        catchError(() => {
          if (requestId === this.logTypeRequestId) {
            this.logLoadError = true;
          }
          return of([] as SystemLogsType[]);
        })
      )
      .subscribe({
        next: files => {
          if (requestId !== this.logTypeRequestId) {
            return;
          }
          this.systemLogsTypeList = files;
          const currentName = this.selectedType?.name;
          this.selectedType = files.find(file => file.name === currentName) || files[0] || null;
          if (this.selectedType) {
            // 保持 isLoadingLogs，由 reloadTableData 的 finalize 关闭
            this.reloadTableData(undefined, undefined, { bypassCache: options?.bypassCache === true });
            return;
          }
          this.systemLogsList = [];
          this.finishLoading();
        },
        error: () => {
          if (requestId === this.logTypeRequestId) {
            this.finishLoading();
          }
        }
      });
  }

  private loadLogDetails(name: string, line: number, bypassCache = false) {
    return this.httpClient
      .get<HttpRespone<SystemLogsContent>>('/api/v1/logs/details', {
        params: { name, line },
        context: bypassCache ? withWtHttpCacheBypass() : undefined
      })
      .pipe(
        map(res => {
          if (res.code !== 20000 || !res.data?.items?.length) {
            return [] as SystemLogs[];
          }
          return res.data.items.map((item: SystemLogs) => {
            let timestampStr = '';
            try {
              timestampStr = item.timestamp ? formatDate(item.timestamp, 'yyyy-MM-dd HH:mm:ss') : '';
            } catch {
              timestampStr = item.timestamp ? String(item.timestamp) : '';
            }
            return {
              ...item,
              timestampStr,
              colorClass: this.getLevelClass(item.level)
            };
          });
        })
      );
  }

  private finishLoading() {
    this.isLoadingLogs = false;
    // 首屏请求完成时偶发不刷新；组件已销毁时不再 detectChanges
    this.safeDetectChanges();
  }

  private safeDetectChanges() {
    if (!this.viewDestroyed) {
      this.cdr.detectChanges();
    }
  }

  /** 虚拟滚动 trackBy：时间戳 + 调用栈 + 内容，避免切换文件时错复用 DOM */
  logTrackBy(index: number, item: SystemLogs) {
    return `${item.timestamp ?? ''}|${item.caller ?? ''}|${item.content ?? ''}|${index}`;
  }

  getSelectedLine() {
    this.lineFormControl.valueChanges.pipe(takeUntil(this.$destroy)).subscribe(value => {
      this.lineSelected = Number(value) || 100;
      localStorage.setItem('wt_log_system_defaultLine', this.lineSelected.toString());
      this.reloadTableData();
    });
  }

  selectType(type: SystemLogsType | null = null): void {
    this.selectedType = type;
    this.viewport?.scrollToIndex(0);
    this.reloadTableData();
  }

  refreshLogs(): void {
    // 手动刷新强制出网并写回会话缓存
    if (this.logLoadError && !this.selectedType) {
      this.loadLogTypes(this.searchForm.get('name')?.value || '', { bypassCache: true });
      return;
    }
    this.reloadTableData(undefined, undefined, { bypassCache: true });
  }

  getLevelClass(level: string) {
    switch (level) {
      case 'error':
        return 'level-error';
      case 'info':
        return 'level-info';
      case 'debug':
        return 'level-debug';
      case 'slow':
        return 'level-slow';
      case 'fatal':
        return 'level-fatal';
      default:
        return '';
    }
  }
}

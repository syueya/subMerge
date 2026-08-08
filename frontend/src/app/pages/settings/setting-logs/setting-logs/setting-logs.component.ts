import { CdkVirtualScrollViewport } from '@angular/cdk/scrolling';
import { Component, ViewChild, inject, signal } from '@angular/core';
import { FormBuilder, FormControl } from '@angular/forms';
import { ServiceQueryParams } from '@common/interfaces';
import { CmParentTableComponent } from '@common/parents/parent-table/parent-table.component';
import { formatDate } from '@common/util';
import { SystemLogs, SystemLogsType } from '@data-struct';
import { catchError, debounceTime, distinctUntilChanged, finalize, map, of, takeUntil } from 'rxjs';

import { LogService } from '../services/log.service';

@Component({
  standalone: false,
  templateUrl: './setting-logs.component.html',
  styleUrl: './setting-logs.component.scss'
})
export class SettingLogsComponent extends CmParentTableComponent {
  private fb = inject(FormBuilder);
  private logService = inject(LogService);

  sidePanelOpened = true;
  systemLogsTypeList: SystemLogsType[] = [];
  /** 接口返回的完整日志（未按等级过滤） */
  private allSystemLogsList: SystemLogs[] = [];
  /** 展示用（已按等级过滤） */
  systemLogsList: SystemLogs[] = [];
  selectedType: SystemLogsType | null = null;
  lineList = [50, 100, 200, 500, 1000];
  lineSelected = 100;
  lineFormControl = new FormControl();
  // subMerge 实际等级：info|warn|error|debug（见 backend applog / LogEntry 注释）
  levelList = [
    { value: '', label: '全部' },
    { value: 'error', label: 'error' },
    { value: 'warn', label: 'warn' },
    { value: 'info', label: 'info' },
    { value: 'debug', label: 'debug' }
  ];
  levelSelected = '';
  levelFormControl = new FormControl('');

  @ViewChild(CdkVirtualScrollViewport) viewport!: CdkVirtualScrollViewport;

  private logTypeRequestId = 0;
  private logRequestId = 0;

  /** 首屏默认 true，避免进页先闪空日志 */
  isLoadingLogs = signal(true);
  logLoadError = false;

  isOver(): boolean {
    return window.matchMedia(`(max-width: 960px)`).matches;
  }

  constructor() {
    super();

    this.searchForm = this.fb.group({
      name: ''
    });

    this.searchForm
      .get('name')
      ?.valueChanges.pipe(takeUntil(this.$destroy), debounceTime(300), distinctUntilChanged())
      .subscribe(value => this.loadLogTypes(value || ''));

    this.lineSelected = Number(localStorage.getItem('wt_log_system_defaultLine')) || 100;
    this.lineFormControl.setValue(this.lineSelected);
    // 等级默认「全部」，不持久化
    this.levelSelected = '';
    this.levelFormControl.setValue('');
    this.getSelectedLine();
    this.getSelectedLevel();
  }

  override handlerAfterViewInit() {
    super.handlerAfterViewInit();
    this.loadLogTypes(this.searchForm.get('name')?.value || '');
  }

  override reloadTableData(_query?: ServiceQueryParams, _sort?: unknown, options?: { bypassCache?: boolean }): void {
    const name = this.selectedType?.name || this.systemLogsTypeList[0]?.name;
    if (!name) {
      this.allSystemLogsList = [];
      this.systemLogsList = [];
      this.finishLoading();
      return;
    }

    this.allSystemLogsList = [];
    this.systemLogsList = [];
    this.logLoadError = false;
    this.isLoadingLogs.set(true);
    const requestId = ++this.logRequestId;

    this.loadLogDetails(name, this.lineSelected, options?.bypassCache === true)
      .pipe(
        takeUntil(this.$destroy),
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
        this.allSystemLogsList = list;
        this.applyLevelFilter();
        queueMicrotask(() => this.viewport?.checkViewportSize());
      });
  }

  private loadLogTypes(value: string, options?: { bypassCache?: boolean }) {
    const requestId = ++this.logTypeRequestId;
    // 作废进行中的详情请求，避免旧 finalize 误关/误开 loading
    this.logRequestId++;
    this.logLoadError = false;
    this.isLoadingLogs.set(true);
    this.allSystemLogsList = [];
    this.systemLogsList = [];

    this.logService
      .list(value || '', options?.bypassCache === true)
      .pipe(
        takeUntil(this.$destroy),
        map(data => data?.files || []),
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
          this.allSystemLogsList = [];
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

  /** 按等级过滤已加载日志（不重新请求） */
  private applyLevelFilter(): void {
    const level = (this.levelSelected || '').toLowerCase();
    if (!level) {
      this.systemLogsList = this.allSystemLogsList;
      return;
    }
    this.systemLogsList = this.allSystemLogsList.filter(item => {
      const itemLevel = (item.level || '').toLowerCase();
      if (level === 'warn') {
        return itemLevel === 'warn' || itemLevel === 'warning';
      }
      return itemLevel === level;
    });
  }

  private loadLogDetails(name: string, line: number, bypassCache = false) {
    return this.logService.details(name, line, bypassCache).pipe(
      map(data => {
        const items = data?.items || [];
        if (!items.length) {
          return [] as SystemLogs[];
        }
        return items.map((item: SystemLogs) => {
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
    this.isLoadingLogs.set(false);
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

  getSelectedLevel() {
    this.levelFormControl.valueChanges.pipe(takeUntil(this.$destroy)).subscribe(value => {
      this.levelSelected = value || '';
      this.applyLevelFilter();
      this.viewport?.scrollToIndex(0);
      queueMicrotask(() => this.viewport?.checkViewportSize());
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
    switch ((level || '').toLowerCase()) {
      case 'error':
        return 'level-error';
      case 'warn':
      case 'warning':
        return 'level-warn';
      case 'info':
        return 'level-info';
      case 'debug':
        return 'level-debug';
      default:
        return '';
    }
  }
}

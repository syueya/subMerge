/* eslint-disable @typescript-eslint/no-explicit-any */
import { Component, ViewChild, OnInit, inject, signal } from '@angular/core';
import { FormGroup } from '@angular/forms';
import { PageEvent } from '@angular/material/paginator';
import { MatSort, Sort } from '@angular/material/sort';
import { MatTable } from '@angular/material/table';
import { BasicQueryParams, HttpRespone, PaginatorPropsType, ServiceQueryParams, TableSortData, initialPaginatorProps } from '@common/interfaces';
import { CM_DIALOG_WIDTH } from '@common/modules/dialog';
import { CmMessageService } from '@common/modules/message';
import { CmSharedDialogDataModel, CmSharedDialogService } from '@common/modules/shared-dialog';
import { ScreenSizeService } from '@common/services/screen-size.service';
import { ScrollService } from '@common/services/scroll.service';
import { Observable, Subject } from 'rxjs';
import { debounceTime, takeUntil } from 'rxjs/operators';

import { CmParentComponent } from '../parent/parent.component';

@Component({
  selector: 'cm-parent-table',
  templateUrl: './parent-table.component.html',
  standalone: false
})
export class CmParentTableComponent extends CmParentComponent implements OnInit {
  @ViewChild(MatTable, { static: false })
  matTable!: MatTable<any>;
  // 分页器初始化;
  paginatorProps: PaginatorPropsType = initialPaginatorProps();

  // 分页器发生改变时
  $pagintorEvent: Subject<PageEvent>;

  // table 表头搜索页
  searchForm!: FormGroup;


  /**
   * table 表格列搜索表头
   *
   * @type {string[]}
   * @memberof ParentTableComponent
   */
  searchFormColumns: string[] = [];
  displayedColumns: string[] = [];

  // 当前操作table data 数据对象
  activeTableDataItem: any;

  defaultPageSizeOptions = [20, 50, 100, 200];

  @ViewChild(MatSort, { static: true })
  sort!: MatSort;

  /**
   * 搜索表单是否需要刷新数据
   */
  searchFormChangeNeedReload = true;

  /**
   * 记住分页大小的存储key
   */
  rememberPageSizeKey!: string;

  /**
   * 记住筛选条件的存储key
   */
  rememberSearchFormKey!: string;

  /**
   * 列表页默认 true：首屏在 AfterViewInit 发请求前避免空数据闪「暂无数据」。
   * 请求结束由 trackLoading()/withLoading 置回 false。
   */
  override isLoading = signal(true);

  isMobileScreen = false;

  private scrollService = inject(ScrollService);
  private screenSizeService = inject(ScreenSizeService);
  /** 避免与子类 private message/dialogService 命名冲突 */
  private parentDialogService = inject(CmSharedDialogService);
  private parentMessageService = inject(CmMessageService);

  public get sortData(): TableSortData | undefined {
    return this.getSortData(this.sort);
  }

  constructor() {
    super();
    this.$pagintorEvent = new Subject();

    this.screenSizeService.screenSize$.pipe(takeUntil(this.$destroy)).subscribe(info => {
      this.isMobileScreen = info.width < 1024;
    });
  }

  override ngOnInit() {
    super.ngOnInit();
    this.pagintorAndSearchFormChange();
    if (this.sort) {
      this.sort.sortChange.pipe(takeUntil(this.$destroy)).subscribe(sort => {
        // 如果存在分页器，且用户修改了分页器的page size，
        this.reloadTableData(
          !!this.paginatorProps && this.paginatorProps.pageSize !== BasicQueryParams.pageSize
            ? {
                pageNum: BasicQueryParams.pageNum,
                pageSize: this.paginatorProps.pageSize
              }
            : BasicQueryParams,
          this.getSortData(this.sort)
        );
      });
    }
  }

  onSearchSubmit() {
    this.$searchFormEvent.next();
  }

  /**
   * 分页器发生变化
   *
   * @param {PageEvent} e
   * @memberof ParentTableComponent
   */
  onPaginatorChange(e: PageEvent) {
    //存储分页大小
    if (this.rememberPageSizeKey && e?.pageSize) {
      localStorage.setItem(this.rememberPageSizeKey, e.pageSize.toString());
    }

    //触发change
    this.$pagintorEvent.next(e);
  }

  // 表单重置
  resetSearchForm() {
    if (this.searchForm?.dirty) {
      this.searchForm.reset();
    }
  }

  /**
   * 重置筛选并回到第一页（列表页筛选区统一调用）
   */
  resetSearch() {
    this.resetSearchForm();
    this.reloadTableDataByFirstPage();
  }

  /**
   * 表格 trackBy：默认按 id，无 id 时回退索引
   * 签名放宽为 any，便于子类按业务字段覆盖
   */
  tableTrackBy(idx: number, item: any): any {
    return item?.id ?? idx;
  }

  /**
   * 选中表格行
   */
  selectTableRow(row: any) {
    this.activeTableDataItem = row;
  }

  /**
   * 通用删除确认：确认后执行 deleteFn，成功则提示并刷新第一页
   * @param ids 待删除 id 列表
   * @param label 业务名称（如“化合物信息”）
   * @param deleteFn 删除请求
   * @param options.onSuccess 删除成功后的额外回调（如清空多选）
   */
  confirmDelete(
    ids: Array<string | number>,
    label: string,
    deleteFn: (ids: Array<string | number>) => Observable<HttpRespone<boolean>>,
    options?: { onSuccess?: () => void; emptyMessage?: string }
  ) {
    if (!ids?.length) {
      this.parentMessageService.error(options?.emptyMessage || `请先选择要删除的${label}`);
      return;
    }

    this.parentDialogService
      .open({
        model: CmSharedDialogDataModel.confirm,
        title: '确认删除',
        content: `确认删除选中的 ${ids.length} 条${label}？`,
        width: CM_DIALOG_WIDTH.small
      })
      .afterClosed()
      .subscribe(result => {
        if (!result) {
          return;
        }
        deleteFn(ids)
          .pipe(takeUntil(this.$destroy))
          .subscribe(res => {
            if (res.code === 20000) {
              this.parentMessageService.success(`${label}删除成功`);
              options?.onSuccess?.();
              this.reloadTableDataByFirstPage();
            } else {
              this.parentMessageService.error(`${label}删除失败`);
            }
          });
      });
  }

  /**
   * 初始化记住分页大小
   *  需要在构造函数请求表格数据前初始化，否则不会记忆
   * @param {string} ComponentName
   * @memberof CmParentTableComponent
   */
  rememberPageSize(ComponentName: string) {
    //生成记忆key
    this.rememberPageSizeKey = `wt_table_remember_page_size_${ComponentName}`;

    //获取已存储的数据
    const pageSize = localStorage.getItem(this.rememberPageSizeKey);

    if (pageSize && this.paginatorProps) {
      this.paginatorProps.pageSize = parseInt(pageSize) || 20;
    }
  }

  /**
   * 初始化记住筛选条件
   *  在构造函数创建 searchForm 之后、请求数据之前调用
   * @param {string} ComponentName
   * @memberof CmParentTableComponent
   */
  rememberSearchForm(ComponentName: string) {
    this.rememberSearchFormKey = `wt_table_remember_search_${ComponentName}`;
    sessionStorage.removeItem(this.rememberSearchFormKey);
  }

  /**
   * 保存当前筛选条件到 sessionStorage
   * @memberof CmParentTableComponent
   */
  saveSearchForm() {
    // 筛选条件不做持久化，进入页面时始终使用空筛选条件
  }

  /**
   * 当页下表，或者搜索表单改变时，联网获取网络数据
   *
   * @memberof TempalteTemplateListComponent
   */
  pagintorAndSearchFormChange() {
    // 收到搜索条件变化，发出网络请求，设置table data
    this.$searchFormEvent.pipe(debounceTime(500), takeUntil(this.$destroy)).subscribe(_ => {
      if (this.searchFormChangeNeedReload) {
        this.reloadTableData();
      }
    });
    // 分页器数值发生变化时
    this.$pagintorEvent.pipe(debounceTime(500), takeUntil(this.$destroy)).subscribe((e: PageEvent) => {
      const { pageIndex, pageSize } = e;
      if (this.paginatorProps.pageIndex !== pageIndex || this.paginatorProps.pageSize !== pageSize) {
        this.paginatorProps.pageIndex = pageIndex;
        this.paginatorProps.pageSize = pageSize;
        this.scrollService.triggerToTop();
        this.reloadTableDataByPage();
      }
    });
  }
  // 按分页，重新获取
  reloadTableDataByPage() {
    const { pageIndex, pageSize } = this.paginatorProps;
    const query = { pageNum: pageIndex + 1, pageSize };
    if (this.sort) {
      this.reloadTableData(query, this.getSortData(this.sort));
    } else {
      this.reloadTableData(query, undefined);
    }
  }

  // 按分页，从第一页重新获取
  reloadTableDataByFirstPage() {
    const { pageSize } = this.paginatorProps;
    const query = { pageNum: 1, pageSize };
    if (this.sort) {
      this.reloadTableData(query, this.getSortData(this.sort));
    } else {
      this.reloadTableData(query, undefined);
    }
  }

  // 按指定分页器页码重新获取，pageIndex 为 MatPaginator 的 0 基页码
  reloadTableDataBySpecificPage(pageIndex: number) {
    const { pageSize } = this.paginatorProps;
    const query = { pageNum: pageIndex + 1, pageSize };
    if (this.sort) {
      this.reloadTableData(query, this.getSortData(this.sort));
    } else {
      this.reloadTableData(query, undefined);
    }
    this.paginatorProps.pageIndex = pageIndex; // 更新分页器的页码
  }

  /**
   * 转换sort 对象
   *
   * @private
   * @param {MatSort} sort
   * @memberof CmParentTableComponent
   */
  protected getSortData(sort: Sort) {
    if (sort && sort.active && sort.direction) {
      const data: TableSortData = {
        orderBy: sort.active,
        orderDirection: sort.direction
      };
      return data;
    }
    return undefined;
  }
  // 由继承组件实现
  reloadTableData(query?: ServiceQueryParams, sortData?: TableSortData) {}
}

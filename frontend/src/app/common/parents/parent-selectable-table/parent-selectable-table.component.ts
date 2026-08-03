/* eslint-disable @typescript-eslint/no-explicit-any */
import { SelectionModel } from '@angular/cdk/collections';
import { Component } from '@angular/core';
import { MatCheckboxChange } from '@angular/material/checkbox';
import { MatTableDataSource } from '@angular/material/table';
import { debounceTime, takeUntil } from 'rxjs/operators';

import { CmParentTableComponent } from '../parent-table/parent-table.component';

/**
 * 带多选勾选能力的列表页父类。
 *
 * Angular 不允许 Directive 继承 Component（NG0903），
 * 因此这里必须是 @Component 基类，仅供继承，不直接挂路由。
 */
@Component({
  selector: 'cm-parent-selectable-table',
  template: '',
  standalone: false
})
export class CmParentSelectableTableComponent<T = any> extends CmParentTableComponent {
  /** 列表数据源（子类可覆盖为更具体的泛型） */
  dataSource = new MatTableDataSource<T>([]);

  /** 勾选集合，true 表示多选 */
  selectionSource = new SelectionModel<T>(true, []);

  /** 是否部分选中 */
  isIndeterminate = false;

  /** 是否全选当前页 */
  isSelectAll = false;

  /** 当前勾选 id 列表（删除等操作可复用） */
  selectedIds: Array<string | number> = [];

  /**
   * 监听勾选变化，刷新表头全选/半选状态
   * 在子类 constructor 的 super() 之后调用
   */
  protected initSelectionWatcher() {
    this.selectionSource.changed.pipe(debounceTime(200), takeUntil(this.$destroy)).subscribe(s => {
      if (s.source.hasValue()) {
        this.refreshTotalCheckboxStatus(s.source.selected as T[]);
      } else {
        this.isSelectAll = false;
        this.isIndeterminate = false;
      }
    });
  }

  /**
   * 根据当前页数据与已选集合，刷新全选/半选
   */
  protected refreshTotalCheckboxStatus(selected: T[]) {
    const pageData = this.dataSource.data || [];
    this.isSelectAll = !!(selected.length && pageData.length && pageData.every(item => selected.includes(item)));
    this.isIndeterminate = !!(selected.length && pageData.length && selected.every(item => !!pageData.find(sub => sub === item)));
    if (this.isSelectAll) {
      this.isIndeterminate = false;
    }
  }

  /** 表头全选切换 */
  allCheckboxChange(change: MatCheckboxChange) {
    if (change.checked) {
      this.selectAll();
    } else {
      this.selectionSource.clear();
    }
  }

  /** 选中当前页全部 */
  protected selectAll() {
    this.dataSource.data.forEach(item => {
      this.selectionSource.deselect(item);
      this.toggleItemCheck(item);
    });
  }

  /** 切换单行勾选 */
  toggleItemCheck(item: T) {
    this.selectionSource.toggle(item);
  }

  /**
   * 解析删除用 id 列表：单行优先，否则取勾选集合
   */
  protected resolveSelectedIds(row: T | null | undefined, idKey: keyof T = 'id' as keyof T): Array<string | number> {
    if (row) {
      return [(row as any)[idKey]];
    }
    return this.selectionSource.selected.map(item => (item as any)[idKey]);
  }
}

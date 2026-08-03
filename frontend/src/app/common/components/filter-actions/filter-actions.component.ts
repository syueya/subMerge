import { Component, EventEmitter, Input, Output } from '@angular/core';

/**
 * 列表筛选区统一「重置 / 过滤」按钮组
 *
 * 示例：
 * <cm-filter-actions
 *   (resetClick)="resetSearch()"
 *   (filterClick)="reloadTableDataByFirstPage()">
 * </cm-filter-actions>
 *
 * 表单 submit 触发过滤时：
 * <cm-filter-actions filterType="submit" (resetClick)="resetSearch()"></cm-filter-actions>
 */
@Component({
  selector: 'cm-filter-actions',
  standalone: false,
  templateUrl: './filter-actions.component.html',
  styleUrl: './filter-actions.component.scss'
})
export class FilterActionsComponent {
  /** 过滤按钮原生 type：button | submit */
  @Input() filterType: 'button' | 'submit' = 'button';

  @Output() readonly resetClick = new EventEmitter<void>();
  @Output() readonly filterClick = new EventEmitter<void>();
}

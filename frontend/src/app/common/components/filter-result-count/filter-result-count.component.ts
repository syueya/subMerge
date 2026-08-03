import { Component, Input } from '@angular/core';

/**
 * 筛选区右侧结果条数文案
 * <cm-filter-result-count [total]="paginatorProps.length || 0"></cm-filter-result-count>
 */
@Component({
  selector: 'cm-filter-result-count',
  standalone: false,
  templateUrl: './filter-result-count.component.html',
  styleUrl: './filter-result-count.component.scss'
})
export class FilterResultCountComponent {
  @Input() total: number | null | undefined = 0;
}

/* eslint-disable @typescript-eslint/no-explicit-any */
import { Component, Input, TemplateRef } from '@angular/core';

@Component({
    selector: 'cm-no-data',
    templateUrl: './no-data.component.html',
    styleUrl: './no-data.component.scss',
    standalone: false
})
export class NoDataComponent {

  /* 缩小 */
  @Input() 
  isMin = false;

  @Input() 
  loading = false;
  @Input() 
  loadingTitle = '正在请求数据，请稍后...';

  @Input() 
  noDataTitle = '暂无数据';

  @Input()
  btnTemplateRef!: TemplateRef<any>

  constructor() { }

}

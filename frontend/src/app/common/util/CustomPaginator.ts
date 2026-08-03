import { Injectable } from '@angular/core';
import { MatPaginatorIntl } from '@angular/material/paginator';
import { Subject } from 'rxjs';

@Injectable()
export class CustomPaginatorIntl implements MatPaginatorIntl {
  changes = new Subject<void>();
  itemsPerPageLabel = '每页行数';
  previousPageLabel = '上一页';
  nextPageLabel = '下一页';
  firstPageLabel = '首页';
  lastPageLabel = '结束页';
  constructor() {}
  getRangeLabel = (page: number, pageSize: number, length: number) => {
    if (length === 0 || pageSize === 0) {
      return `0 / 0`;
    }
    length = Math.max(length, 0);
    const startIndex = page * pageSize;
    const endIndex = startIndex < length ? Math.min(startIndex + pageSize, length) : startIndex + pageSize;
    return `第${startIndex + 1}-${endIndex}条，共${length}条`;
  };
}

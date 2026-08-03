import { AfterContentInit, AfterViewInit, Component, OnDestroy, OnInit } from '@angular/core';
import { Subject } from 'rxjs';

import { CmOnDestoryComponent } from '../on-destory/on-destory.component';

@Component({
  selector: 'cm-parent',
  templateUrl: './parent.component.html',
  standalone: false
})
export class CmParentComponent extends CmOnDestoryComponent implements OnInit, OnDestroy, AfterViewInit, AfterContentInit {
  /**
   * 搜索表单事件
   */
  $searchFormEvent: Subject<void>;

  constructor() {
    super();
    this.$searchFormEvent = new Subject();
  }

  ngOnInit() {}
  override ngOnDestroy() {
    super.ngOnDestroy();
  }
  ngAfterViewInit() {
    setTimeout(() => {
      this.handlerAfterViewInit();
    }, 0);
  }

  ngAfterContentInit() {}

  /**
   * 内容初始化后调用该方法
   *
   * @memberof CmParentComponent
   */
  handlerAfterViewInit() {}
}

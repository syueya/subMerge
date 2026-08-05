import { Component, OnDestroy, signal } from '@angular/core';
import { withLoading } from '@common/util/with-loading';
import { MonoTypeOperatorFunction, Subject } from 'rxjs';

@Component({
  selector: 'cm-on-destory',
  templateUrl: './on-destory.component.html',
  standalone: false
})
export class CmOnDestoryComponent implements OnDestroy {
  $destroy: Subject<void>;

  /** 局部 loading（表格/弹窗）；全屏遮罩由 pending interceptor 负责 */
  isLoading = signal(false);

  constructor() {
    this.$destroy = new Subject();
  }

  /** 绑定请求与 isLoading signal：subscribe 时 true，结束时 false */
  protected trackLoading<T>(): MonoTypeOperatorFunction<T> {
    return withLoading(this.isLoading);
  }

  ngOnDestroy() {
    this.$destroy.next();
    this.$destroy.complete();
  }
}

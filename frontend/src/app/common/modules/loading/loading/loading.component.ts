import { Component, OnInit, inject } from '@angular/core';
import { debounce, distinctUntilChanged, merge, Observable, partition, switchMap, tap, timer } from 'rxjs';

import { PendingRequestsInterceptor } from '../services/pending-requests-interceptor.service';
import { SpinnerVisibilityService } from '../services/spinner-visibility.service';
/**
 * 该组件为 NgHttpLoaderComponent 组件，用于管理 HTTP 请求加载动画的显示与隐藏
 */
@Component({
    selector: 'cm-loading',
    templateUrl: './loading.component.html',
    styleUrl: './loading.component.scss',
    standalone: false
})
export class LoadingComponent implements OnInit {
  private pendingRequestsInterceptor = inject(PendingRequestsInterceptor);
  private spinnerVisibility = inject(SpinnerVisibilityService);

  isVisible$!: Observable<boolean>; // 加载动画显示状态的可观察对象
  visibleUntil = Date.now(); // 加载动画显示时间戳

  debounceDelay = 200; // 防抖延迟时间
  extraDuration = 0; // 额外持续时间
  minDuration = 0;

  ngOnInit(): void {
    // 初始化加载动画显示状态的可观察对象
    this.initIsvisibleObservable();
  }

  /**
   * 初始化加载动画显示状态的可观察对象
   */
  private initIsvisibleObservable(): void {
    const [showSpinner$, hideSpinner$] = partition(this.pendingRequestsInterceptor.pendingRequestsStatus$, h => h);

    this.isVisible$ = merge(
      this.pendingRequestsInterceptor.pendingRequestsStatus$.pipe(switchMap(() => showSpinner$.pipe(debounce(() => timer(this.debounceDelay))))),
      showSpinner$.pipe(switchMap(() => hideSpinner$.pipe(debounce(() => this.getVisibilityTimer$())))),
      this.spinnerVisibility.visibility$
    ).pipe(
      distinctUntilChanged(),
      tap(h => this.updateExpirationDelay(h))
    );
  }

  /**
   * 更新加载动画显示状态的可观察对象
   * @param showSpinner 是否显示加载动画
   */
  private updateExpirationDelay(showSpinner: boolean): void {
    if (showSpinner) {
      this.visibleUntil = Date.now() + this.minDuration;
    }
  }

  /**
   * 获取加载动画显示状态的可观察对象
   */
  private getVisibilityTimer$(): Observable<number> {
    return timer(Math.max(this.extraDuration, this.visibleUntil - Date.now()));
  }
}

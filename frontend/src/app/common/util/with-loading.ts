import { WritableSignal } from '@angular/core';
import { MonoTypeOperatorFunction, defer } from 'rxjs';
import { finalize } from 'rxjs/operators';

/**
 * 将请求与 loading signal 绑定：订阅时 true，结束（含 error/unsubscribe）时 false。
 * 列表/弹窗统一用此算子，避免手写 isLoading + finalize 遗漏或 Zone 外不刷新。
 */
export function withLoading<T>(loading: WritableSignal<boolean>): MonoTypeOperatorFunction<T> {
  return source =>
    defer(() => {
      loading.set(true);
      return source.pipe(finalize(() => loading.set(false)));
    });
}

import { NgZone } from '@angular/core';
import { Observable } from 'rxjs';

/**
 * 保证 Observable 的 next/error/complete 在 Angular Zone 内执行。
 * HTTP 完成（含缓存命中）若落在 Zone 外，会导致 signal/字段已更新但视图不刷新。
 */
export function observeInAngularZone<T>(ngZone: NgZone, source: Observable<T>): Observable<T> {
  return new Observable<T>(subscriber => {
    const subscription = source.subscribe({
      next: value => {
        if (NgZone.isInAngularZone()) {
          subscriber.next(value);
        } else {
          ngZone.run(() => subscriber.next(value));
        }
      },
      error: err => {
        if (NgZone.isInAngularZone()) {
          subscriber.error(err);
        } else {
          ngZone.run(() => subscriber.error(err));
        }
      },
      complete: () => {
        if (NgZone.isInAngularZone()) {
          subscriber.complete();
        } else {
          ngZone.run(() => subscriber.complete());
        }
      }
    });
    return () => subscription.unsubscribe();
  });
}

/* eslint-disable @typescript-eslint/no-explicit-any */
import {
  HttpErrorResponse,
  HttpEvent,
  HttpHandlerFn,
  HttpInterceptorFn,
  HttpRequest,
  HttpResponse,
  HttpResponseBase
} from '@angular/common/http';
import { Injector, NgZone, inject } from '@angular/core';
import { MatDialog } from '@angular/material/dialog';
import { CmMessageService } from '@common/modules/message';
import { CmNotificationService } from '@common/modules/notification';
import { AuthService } from '@common/services';
import { DEFAULT_TIMEOUT } from '@common/util/defines';
import { processUrl } from '@common/util/urlUtils';
import { Observable, of, throwError, mergeMap, catchError, timeout, TimeoutError } from 'rxjs';

import { ReThrowHttpError, checkStatus, processHeaders, toLogin } from './helper';

function runInAngularZone<T>(ngZone: NgZone, source: Observable<T>): Observable<T> {
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

let lastAuthErrorTime = 0;

function isSubMergeEnvelope(body: any): body is { ok: boolean; data?: unknown; error?: { code?: string; message?: string } } {
  return body && typeof body === 'object' && typeof body.ok === 'boolean';
}

function handleData(injector: Injector, ev: HttpResponseBase, req: HttpRequest<any>, next: HttpHandlerFn): Observable<any> {
  if (ev instanceof TimeoutError) {
    injector.get(CmMessageService).error('请求超时，请稍后重试');
    return throwError(() => ev);
  }

  checkStatus(injector, ev);

  switch (ev.status) {
    case 200:
      if (ev instanceof HttpResponse) {
        const body = ev.body;
        if (isSubMergeEnvelope(body) && body.ok === false) {
          const msg = body.error?.message || '请求失败';
          injector.get(CmMessageService).error(msg);
          return of(ev);
        }
        if (body && typeof body.code === 'number' && body.code > 20000) {
          if (body.code === 50001 || body.code === 40002) {
            const now = Date.now();
            if (now - lastAuthErrorTime > 1000) {
              lastAuthErrorTime = now;
              injector.get(MatDialog).closeAll();
              injector.get(AuthService).clear();
              injector.get(CmNotificationService).error(`未登录或登录已过期，请重新登录。`, ``);
            }
          } else if (body.message) {
            injector.get(CmMessageService).error(body.message);
          }
          return of(ev);
        }
        return of(ev);
      }
      break;
    case 401: {
      const now = Date.now();
      if (now - lastAuthErrorTime > 1000) {
        lastAuthErrorTime = now;
        injector.get(MatDialog).closeAll();
        toLogin(injector);
      }
      break;
    }
    default:
      if (ev instanceof HttpErrorResponse) {
        console.warn('HTTP error', ev.status, ev.url);
      }
      break;
  }

  if (ev instanceof HttpErrorResponse) {
    return throwError(() => ev);
  } else if ((ev as unknown as ReThrowHttpError)._throw === true) {
    return throwError(() => (ev as unknown as ReThrowHttpError).body);
  } else {
    return of(ev);
  }
}

export const defaultInterceptor: HttpInterceptorFn = (req: HttpRequest<any>, next: HttpHandlerFn): Observable<HttpEvent<any>> => {
  const injector = inject(Injector);
  const ngZone = inject(NgZone);

  const url = processUrl(req.url);
  const timeoutTime = Number(req.headers.get('timeout')) || DEFAULT_TIMEOUT;

  const withCreds = url.includes('/api') || req.url.startsWith('/api');
  const newReq = req.clone({
    url,
    headers: processHeaders(req.headers),
    withCredentials: withCreds ? true : req.withCredentials
  });

  return runInAngularZone(
    ngZone,
    next(newReq).pipe(
      timeout(timeoutTime),
      mergeMap((response: HttpEvent<any>) => {
        if (response instanceof HttpResponseBase) {
          return handleData(injector, response, newReq, next);
        }
        return of(response);
      }),
      catchError((err: HttpErrorResponse) => handleData(injector, err, newReq, next))
    )
  );
};

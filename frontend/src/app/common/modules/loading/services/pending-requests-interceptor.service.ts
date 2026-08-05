import { HttpEvent, HttpHandlerFn, HttpInterceptorFn, HttpRequest } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { SessionHttpCacheService } from '@common/net/session-http-cache';
import { Observable, ReplaySubject, finalize } from 'rxjs';

/**
 * 全局待处理 HTTP 计数服务（全屏 cm-loading 用）。
 * 通过 functional interceptor 接入 provideHttpClient，不再走 HTTP_INTERCEPTORS DI。
 */
@Injectable({
  providedIn: 'root'
})
export class PendingRequestsInterceptor {
  private _pendingRequests = 0;
  private _pendingRequestsStatus$ = new ReplaySubject<boolean>(1);
  private _filteredUrlPatterns: RegExp[] = [];
  private _filteredMethods: string[] = [];
  private _filteredHeaders: string[] = [];
  private _forceByPass = false;

  get pendingRequestsStatus$(): Observable<boolean> {
    return this._pendingRequestsStatus$.asObservable();
  }

  get pendingRequests(): number {
    return this._pendingRequests;
  }

  get filteredUrlPatterns(): RegExp[] {
    return this._filteredUrlPatterns;
  }

  set filteredMethods(httpMethods: string[]) {
    this._filteredMethods = httpMethods;
  }

  set filteredHeaders(value: string[]) {
    this._filteredHeaders = value;
  }

  set forceByPass(value: boolean) {
    this._forceByPass = value;
  }

  /** 是否应跳过全屏 loading（读列表走局部 spinner；写操作才遮全屏） */
  shouldBypass(req: HttpRequest<unknown>, isReadRequest: boolean): boolean {
    const hasNoLoadingSpinnerHeader = req.headers.has('REQUEST_NO_LOADING_SPINNER');
    return (
      this._forceByPass ||
      isReadRequest ||
      this.shouldBypassUrl(req.urlWithParams) ||
      this.shouldBypassMethod(req) ||
      this.shouldBypassHeader(req) ||
      hasNoLoadingSpinnerHeader
    );
  }

  begin(): void {
    this._pendingRequests++;
    if (this._pendingRequests === 1) {
      this._pendingRequestsStatus$.next(true);
    }
  }

  end(): void {
    this._pendingRequests = Math.max(0, this._pendingRequests - 1);
    if (this._pendingRequests === 0) {
      this._pendingRequestsStatus$.next(false);
    }
  }

  private shouldBypassUrl(url: string): boolean {
    return this._filteredUrlPatterns.some(e => e.test(url));
  }

  private shouldBypassMethod(req: HttpRequest<unknown>): boolean {
    return this._filteredMethods.some(e => e.toUpperCase() === req.method.toUpperCase());
  }

  private shouldBypassHeader(req: HttpRequest<unknown>): boolean {
    return this._filteredHeaders.some(e => req.headers.has(e));
  }
}

/**
 * 全屏 loading 拦截器：最外层注册，缓存命中也会 finalize。
 * 读接口（与会话缓存 read 判定对齐）默认不遮全屏，避免与表格局部 spinner 叠两层。
 */
export const pendingRequestsInterceptorFn: HttpInterceptorFn = (
  req: HttpRequest<unknown>,
  next: HttpHandlerFn
): Observable<HttpEvent<unknown>> => {
  const pending = inject(PendingRequestsInterceptor);
  const cache = inject(SessionHttpCacheService);

  let request = req;
  if (request.headers.has('REQUEST_NO_LOADING_SPINNER')) {
    request = request.clone({
      headers: request.headers.delete('REQUEST_NO_LOADING_SPINNER')
    });
  }

  const shouldBypass = pending.shouldBypass(req, cache.isReadRequest(req));
  if (!shouldBypass) {
    pending.begin();
  }

  return next(request).pipe(
    finalize(() => {
      if (!shouldBypass) {
        pending.end();
      }
    })
  );
};

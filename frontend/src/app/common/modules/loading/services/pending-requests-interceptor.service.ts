import { HttpInterceptor, HttpRequest, HttpHandler, HttpEvent, HTTP_INTERCEPTORS } from '@angular/common/http';
import { Injectable, ExistingProvider } from '@angular/core';
import { ReplaySubject, Observable, finalize } from 'rxjs';

/**
 * HTTP请求拦截器，用于管理待处理的HTTP请求
 */
@Injectable({
  providedIn: 'root'
})
export class PendingRequestsInterceptor implements HttpInterceptor {
  private _pendingRequests = 0; // 待处理的请求数量
  private _pendingRequestsStatus$ = new ReplaySubject<boolean>(1); // 待处理请求状态的可观察对象
  private _filteredUrlPatterns: RegExp[] = []; // 需要绕过的URL模式
  private _filteredMethods: string[] = []; // 需要绕过的HTTP方法
  private _filteredHeaders: string[] = []; // 需要绕过的请求头信息
  private _forceByPass = false; // 是否强制绕过拦截器处理

  // 获取待处理请求状态的可观察对象
  get pendingRequestsStatus$(): Observable<boolean> {
    return this._pendingRequestsStatus$.asObservable();
  }

  // 获取待处理的请求数量
  get pendingRequests(): number {
    return this._pendingRequests;
  }

  // 获取需要绕过的URL模式数组
  get filteredUrlPatterns(): RegExp[] {
    return this._filteredUrlPatterns;
  }

  // 设置需要绕过的HTTP方法数组
  set filteredMethods(httpMethods: string[]) {
    this._filteredMethods = httpMethods;
  }

  // 设置需要绕过的请求头信息数组
  set filteredHeaders(value: string[]) {
    this._filteredHeaders = value;
  }

  // 设置是否强制绕过拦截器处理
  set forceByPass(value: boolean) {
    this._forceByPass = value;
  }

  /**
   * 判断是否应绕过指定的URL
   */
  private shouldBypassUrl(url: string): boolean {
    return this._filteredUrlPatterns.some(e => {
      return e.test(url);
    });
  }

  /**
   * 判断是否应绕过指定的HTTP方法
   */
  private shouldBypassMethod(req: HttpRequest<unknown>): boolean {
    return this._filteredMethods.some(e => {
      return e.toUpperCase() === req.method.toUpperCase();
    });
  }

  /**
   * 判断是否应绕过指定的请求头信息
   */
  private shouldBypassHeader(req: HttpRequest<unknown>): boolean {
    return this._filteredHeaders.some(e => {
      return req.headers.has(e);
    });
  }

  /**
   * 判断是否应绕过拦截器处理
   */
  private shouldBypass(req: HttpRequest<unknown>): boolean {
    const hasNoLoadingSpinnerHeader = req.headers.has('REQUEST_NO_LOADING_SPINNER');
    return this._forceByPass || this.shouldBypassUrl(req.urlWithParams) || this.shouldBypassMethod(req) || this.shouldBypassHeader(req) || hasNoLoadingSpinnerHeader;
  }

  /**
   * 拦截HTTP请求
   */
  intercept(req: HttpRequest<unknown>, next: HttpHandler): Observable<HttpEvent<unknown>> {
    const shouldBypass = this.shouldBypass(req);

    if (shouldBypass && req.headers.has('REQUEST_NO_LOADING_SPINNER')) {
      // 移除 REQUEST_NO_LOADING_SPINNER 头部
      req = req.clone({
        headers: req.headers.delete('REQUEST_NO_LOADING_SPINNER')
      });
    }

    if (!shouldBypass) {
      this._pendingRequests++;

      if (1 === this._pendingRequests) {
        this._pendingRequestsStatus$.next(true);
      }
    }

    return next.handle(req).pipe(
      finalize(() => {
        if (!shouldBypass) {
          this._pendingRequests--;

          if (0 === this._pendingRequests) {
            this._pendingRequestsStatus$.next(false);
          }
        }
      })
    );
  }
}

/**
 * HTTP请求拦截器提供器
 */
export const PendingRequestsInterceptorProvider: ExistingProvider[] = [
  {
    provide: HTTP_INTERCEPTORS,
    useExisting: PendingRequestsInterceptor,
    multi: true
  }
];

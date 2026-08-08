import { HttpClient, HttpContext, HttpErrorResponse, HttpHeaders, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { ApiResponse } from '@data-struct';
import { Observable, catchError, map, throwError } from 'rxjs';

export interface ApiRequestOptions {
	/** 前端 RxJS 超时（ms）；经 timeout 请求头传给 defaultInterceptor，出网前会剥离 */
	timeoutMs?: number;
	/** 是否跳过全屏 loading；局部按钮已有独立 loading 时使用 */
	noLoadingSpinner?: boolean;
	/** 查询参数 */
	params?: HttpParams | Record<string, string | number | boolean | ReadonlyArray<string | number | boolean>>;
	/** 透传 HttpContext（如会话读缓存 bypass） */
	context?: HttpContext;
}

/**
 * 管理端 JSON API 封装。
 * Authorization 与 HTTP 401 清会话由 authInterceptor 统一处理，这里只负责：
 * - Content-Type
 * - 业务体 { ok:false } 解包
 * - 错误消息提取
 */
@Injectable({ providedIn: 'root' })
export class ApiService {
	private readonly http = inject(HttpClient);
	private readonly base = '/api';
	private readonly jsonHeaders = new HttpHeaders({ 'Content-Type': 'application/json' });

	get<T>(path: string, options?: ApiRequestOptions): Observable<T> {
		return this.http
			.get<ApiResponse<T>>(this.base + path, {
				headers: this.headers(options),
				params: options?.params,
				context: options?.context,
			})
			.pipe(
				map((r) => this.unwrap(r)),
				catchError((e) => this.handle(e)),
			);
	}

	post<T>(path: string, body?: unknown, options?: ApiRequestOptions): Observable<T> {
		return this.http
			.post<ApiResponse<T>>(this.base + path, body ?? {}, {
				headers: this.headers(options),
				params: options?.params,
				context: options?.context,
			})
			.pipe(
				map((r) => this.unwrap(r)),
				catchError((e) => this.handle(e)),
			);
	}

	put<T>(path: string, body?: unknown, options?: ApiRequestOptions): Observable<T> {
		return this.http
			.put<ApiResponse<T>>(this.base + path, body ?? {}, {
				headers: this.headers(options),
				params: options?.params,
				context: options?.context,
			})
			.pipe(
				map((r) => this.unwrap(r)),
				catchError((e) => this.handle(e)),
			);
	}

	delete<T>(path: string, options?: ApiRequestOptions): Observable<T> {
		return this.http
			.delete<ApiResponse<T>>(this.base + path, {
				headers: this.headers(options),
				params: options?.params,
				context: options?.context,
			})
			.pipe(
				map((r) => this.unwrap(r)),
				catchError((e) => this.handle(e)),
			);
	}

	private headers(options?: ApiRequestOptions): HttpHeaders {
		let h = this.jsonHeaders;
		if (options?.timeoutMs !== null && options?.timeoutMs !== undefined && options.timeoutMs > 0) {
			h = h.set('timeout', String(options.timeoutMs));
		}
		if (options?.noLoadingSpinner) {
			h = h.set('REQUEST_NO_LOADING_SPINNER', '1');
		}
		return h;
	}

	private unwrap<T>(r: ApiResponse<T>): T {
		if (!r.ok || r.data === undefined) {
			throw new Error(r.error?.message || 'request failed');
		}
		return r.data;
	}

	private handle(err: HttpErrorResponse | Error) {
		if (err instanceof HttpErrorResponse) {
			const msg =
				err.error?.error?.message ||
				err.error?.message ||
				err.message ||
				'network error';
			// 401 清会话由 authInterceptor 负责，这里只把消息往上抛
			return throwError(() => new Error(msg));
		}
		return throwError(() => err);
	}
}

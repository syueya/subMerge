import { HttpClient, HttpErrorResponse, HttpHeaders } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable, catchError, map, throwError } from 'rxjs';
import { ApiResponse } from '@data-struct';

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

	get<T>(path: string): Observable<T> {
		return this.http
			.get<ApiResponse<T>>(this.base + path, { headers: this.jsonHeaders })
			.pipe(
				map((r) => this.unwrap(r)),
				catchError((e) => this.handle(e)),
			);
	}

	post<T>(path: string, body?: unknown): Observable<T> {
		return this.http
			.post<ApiResponse<T>>(this.base + path, body ?? {}, { headers: this.jsonHeaders })
			.pipe(
				map((r) => this.unwrap(r)),
				catchError((e) => this.handle(e)),
			);
	}

	put<T>(path: string, body?: unknown): Observable<T> {
		return this.http
			.put<ApiResponse<T>>(this.base + path, body ?? {}, { headers: this.jsonHeaders })
			.pipe(
				map((r) => this.unwrap(r)),
				catchError((e) => this.handle(e)),
			);
	}

	delete<T>(path: string): Observable<T> {
		return this.http
			.delete<ApiResponse<T>>(this.base + path, { headers: this.jsonHeaders })
			.pipe(
				map((r) => this.unwrap(r)),
				catchError((e) => this.handle(e)),
			);
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

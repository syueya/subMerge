import { HttpContext } from '@angular/common/http';
import { Observable, shareReplay, tap } from 'rxjs';

import { withWtHttpCacheBypass } from './session-http-cache';

/**
 * 会话内单值缓存：把一个「每次都发请求」的工厂包装成「首个订阅发请求、后续复用结果」。
 *
 * 用于多个页面共享的列表接口（策略组 / 规则 / 订阅源 / 节点），避免每进一个页面就重复拉取。
 * - get()：有缓存返回缓存，无则发起并缓存（shareReplay 1）
 * - get(true)：强制绕过本层 + 会话 HTTP 缓存重新拉取（页面「刷新」按钮用）
 * - invalidate()：清缓存，下次 get() 会重新请求（增删改后调用）
 *
 * 请求出错时不缓存，允许下次重试。
 */
export class CachedRequest<T> {
	private cache$: Observable<T> | null = null;

	/**
	 * @param factory 可选 HttpContext；force 时会注入 WT_HTTP_CACHE=bypass，避免拦截器会话缓存挡住刷新
	 */
	constructor(private readonly factory: (context?: HttpContext) => Observable<T>) {}

	get(forceRefresh = false): Observable<T> {
		if (forceRefresh || !this.cache$) {
			const context = forceRefresh ? withWtHttpCacheBypass() : undefined;
			this.cache$ = this.factory(context).pipe(
				tap({
					error: () => {
						this.cache$ = null;
					},
				}),
				shareReplay({ bufferSize: 1, refCount: false }),
			);
		}
		return this.cache$;
	}

	invalidate(): void {
		this.cache$ = null;
	}
}

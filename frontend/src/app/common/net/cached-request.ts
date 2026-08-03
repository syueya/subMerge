import { Observable, shareReplay, tap } from 'rxjs';

/**
 * 会话内单值缓存：把一个「每次都发请求」的工厂包装成「首个订阅发请求、后续复用结果」。
 *
 * 用于多个页面共享的列表接口（策略组 / 规则 / 订阅源 / 节点），避免每进一个页面就重复拉取。
 * - get()：有缓存返回缓存，无则发起并缓存（shareReplay 1）
 * - get(true)：强制绕过缓存重新拉取（页面「刷新」按钮用）
 * - invalidate()：清缓存，下次 get() 会重新请求（增删改后调用）
 *
 * 请求出错时不缓存，允许下次重试。
 */
export class CachedRequest<T> {
	private cache$: Observable<T> | null = null;

	constructor(private readonly factory: () => Observable<T>) {}

	get(forceRefresh = false): Observable<T> {
		if (forceRefresh || !this.cache$) {
			this.cache$ = this.factory().pipe(
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

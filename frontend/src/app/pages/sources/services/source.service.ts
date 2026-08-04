import { HttpParams } from '@angular/common/http';
import { Injectable, inject, signal } from '@angular/core';
import { Observable, Subscription, shareReplay, tap } from 'rxjs';
import { ApiService } from '@common/net/api.service';
import { CachedRequest } from '@common/net/cached-request';
import { DialogService } from '@common/services/dialog.service';
import {
	ListResponse,
	ProxyNode,
	RefreshAllResult,
	RefreshSourceResult,
	RegionCatalogResponse,
	SourceUpsertBody,
	SubscriptionSource,
} from '@data-struct';

/** 拉取全部可能串行很久；默认 30s 不够，给 10 分钟 */
const REFRESH_ALL_TIMEOUT_MS = 10 * 60 * 1000;

@Injectable({ providedIn: 'root' })
export class SourceService {
	private readonly api = inject(ApiService);
	private readonly dialog = inject(DialogService);

	/**
	 * 地区目录来自编译进后端的 regions.yaml，运行期不变。
	 * 会话内只请求一次（订阅源页 / 规则页共用）；失败不缓存，下次可重试。
	 */
	private regions$: Observable<RegionCatalogResponse> | null = null;

	// 订阅源列表与全量节点列表被概览/策略组/订阅源多页共用，做会话内缓存，写操作后失效。
	private readonly sourcesCache = new CachedRequest<ListResponse<SubscriptionSource>>(() =>
		this.api.get('/sources'),
	);
	private readonly proxiesCache = new CachedRequest<ListResponse<ProxyNode>>(() =>
		this.api.get('/proxies'),
	);

	private refreshAllSub: Subscription | null = null;

	/** 跨页面共享：后台拉取全部进行中，不随页面销毁丢失 */
	readonly refreshingAll = signal(false);

	list(forceRefresh = false): Observable<ListResponse<SubscriptionSource>> {
		return this.sourcesCache.get(forceRefresh);
	}

	listRegions(): Observable<RegionCatalogResponse> {
		if (!this.regions$) {
			this.regions$ = this.api.get<RegionCatalogResponse>('/regions').pipe(
				tap({
					error: () => {
						this.regions$ = null;
					},
				}),
				shareReplay({ bufferSize: 1, refCount: false }),
			);
		}
		return this.regions$;
	}

	/** 使订阅源缓存失效（增删改源后调用）。 */
	private invalidateSources(): void {
		this.sourcesCache.invalidate();
	}

	/** 刷新源会更新节点数据与源状态，两者缓存都失效。 */
	private invalidateSourcesAndProxies(): void {
		this.sourcesCache.invalidate();
		this.proxiesCache.invalidate();
	}

	create(body: SourceUpsertBody & { name: string; region: string; url: string }): Observable<SubscriptionSource> {
		return this.api.post<SubscriptionSource>('/sources', body).pipe(tap(() => this.invalidateSources()));
	}

	update(id: number, body: SourceUpsertBody): Observable<SubscriptionSource> {
		return this.api.put<SubscriptionSource>(`/sources/${id}`, body).pipe(tap(() => this.invalidateSources()));
	}

	delete(id: number): Observable<{ success: boolean }> {
		return this.api.delete<{ success: boolean }>(`/sources/${id}`).pipe(tap(() => this.invalidateSourcesAndProxies()));
	}

	batchDelete(ids: number[]): Observable<{ deleted: number }> {
		return this.api.post<{ deleted: number }>('/sources/batch-delete', { ids }).pipe(tap(() => this.invalidateSourcesAndProxies()));
	}

	refresh(id: number): Observable<RefreshSourceResult> {
		return this.api.post<RefreshSourceResult>(`/sources/${id}/refresh`).pipe(tap(() => this.invalidateSourcesAndProxies()));
	}

	refreshAll(): Observable<RefreshAllResult> {
		return this.api
			.post<RefreshAllResult>('/sources/refresh-all', {}, { timeoutMs: REFRESH_ALL_TIMEOUT_MS })
			.pipe(tap(() => this.invalidateSourcesAndProxies()));
	}

	/**
	 * 启动后台拉取全部：请求挂在 root 服务上，不随页面销毁取消。
	 * 完成用短 toast（无换行，避免弹阻塞对话框）；关标签页后前端无法再提示，后端仍会继续跑完。
	 */
	startBackgroundRefreshAll(): boolean {
		if (this.refreshingAll()) return false;

		this.refreshingAll.set(true);
		this.refreshAllSub?.unsubscribe();
		this.refreshAllSub = this.refreshAll().subscribe({
			next: (res) => {
				this.refreshingAll.set(false);
				this.refreshAllSub = null;
				void this.dialog[res.failed > 0 ? 'error' : 'success'](this.formatRefreshAllMsg(res));
			},
			error: (err: Error) => {
				this.refreshingAll.set(false);
				this.refreshAllSub = null;
				void this.dialog.error(err?.message || '拉取全部失败');
			},
		});
		return true;
	}

	/** 单行摘要，走 message toast（与 Geo 后台更新同属 DialogService） */
	private formatRefreshAllMsg(res: RefreshAllResult): string {
		const total = res.total ?? 0;
		const ok = res.ok ?? 0;
		const failed = res.failed ?? 0;
		if (failed <= 0) return `全部拉取完成：成功 ${ok} / 共 ${total}`;
		const fails = (res.results || [])
			.filter((r) => !r.ok)
			.map((r) => `${r.name}`)
			.slice(0, 3)
			.join('、');
		const more = failed > 3 ? '…' : '';
		return `全部拉取完成：成功 ${ok}，失败 ${failed} / 共 ${total}${fails ? `（${fails}${more}）` : ''}`;
	}

	listProxies(sourceId?: number, forceRefresh = false): Observable<ListResponse<ProxyNode>> {
		// 只缓存全量节点列表（跨页共用的那份）；按源过滤的查询直接透传。
		if (sourceId == null) {
			return this.proxiesCache.get(forceRefresh);
		}
		const params = new HttpParams().set('sourceId', sourceId);
		return this.api.get(`/proxies?${params.toString()}`);
	}

	updateProxy(id: number, enabled: boolean): Observable<ProxyNode> {
		return this.api.put<ProxyNode>(`/proxies/${id}`, { enabled }).pipe(tap(() => this.proxiesCache.invalidate()));
	}

	batchUpdateProxies(ids: number[], enabled: boolean): Observable<{ updated: number }> {
		return this.api.put<{ updated: number }>('/proxies/batch', { ids, enabled }).pipe(tap(() => this.proxiesCache.invalidate()));
	}
}

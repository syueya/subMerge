import { HttpParams } from '@angular/common/http';
import { Injectable, inject, signal } from '@angular/core';
import { ApiRequestOptions, ApiService } from '@common/net/api.service';
import { CachedRequest } from '@common/net/cached-request';
import { withWtHttpCacheBypass } from '@common/net/session-http-cache';
import { DialogService } from '@common/services/dialog.service';
import { ListResponse, ProxyNode, RefreshAllResult, RefreshSourceResult, RegionCatalogResponse, SourceUpsertBody, SubscriptionSource } from '@data-struct';
import { Observable, Subscription, finalize, shareReplay, tap } from 'rxjs';

import { formatRefreshMsg } from './source-refresh.util';

/** 后台拉取可能包含网络等待和解析；默认 30s 不够，给 10 分钟 */
const REFRESH_BACKGROUND_TIMEOUT_MS = 10 * 60 * 1000;

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
  private readonly sourcesCache = new CachedRequest<ListResponse<SubscriptionSource>>(context => this.api.get('/sources', { context }));
  private readonly proxiesCache = new CachedRequest<ListResponse<ProxyNode>>(context => this.api.get('/proxies', { context }));

  private refreshAllSub: Subscription | null = null;
  private readonly refreshSubs = new Map<number, Subscription>();

  /** 跨页面共享：后台拉取全部进行中，不随页面销毁丢失 */
  readonly refreshingAll = signal(false);
  /** 跨页面共享：正在后台拉取的单个订阅源 ID */
  readonly refreshingIds = signal<ReadonlySet<number>>(new Set());
  /** 每次单源后台任务结束时递增，供列表页静默刷新 */
  readonly refreshCompletionVersion = signal(0);

  list(forceRefresh = false): Observable<ListResponse<SubscriptionSource>> {
    return this.sourcesCache.get(forceRefresh);
  }

  listRegions(): Observable<RegionCatalogResponse> {
    if (!this.regions$) {
      this.regions$ = this.api.get<RegionCatalogResponse>('/regions').pipe(
        tap({
          error: () => {
            this.regions$ = null;
          }
        }),
        shareReplay({ bufferSize: 1, refCount: false })
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

  refresh(id: number, options?: ApiRequestOptions): Observable<RefreshSourceResult> {
    return this.api.post<RefreshSourceResult>(`/sources/${id}/refresh`, {}, options).pipe(tap(() => this.invalidateSourcesAndProxies()));
  }

  /** 启动单源后台拉取：请求不绑定列表组件生命周期，也不触发全屏 loading。 */
  startBackgroundRefresh(id: number): boolean {
    if (this.refreshSubs.has(id) || this.refreshingAll()) return false;

    this.refreshingIds.update(ids => new Set(ids).add(id));
    const sub = this.refresh(id, {
      timeoutMs: REFRESH_BACKGROUND_TIMEOUT_MS,
      noLoadingSpinner: true,
    }).pipe(
      finalize(() => {
        this.refreshSubs.delete(id);
        this.refreshingIds.update(ids => {
          const next = new Set(ids);
          next.delete(id);
          return next;
        });
        this.refreshCompletionVersion.update(version => version + 1);
      }),
    ).subscribe({
      next: result => void this.dialog.success(formatRefreshMsg(result, '拉取成功')),
      error: (err: Error) => void this.dialog.error(err?.message || '拉取失败'),
    });
    this.refreshSubs.set(id, sub);
    return true;
  }

  refreshAll(): Observable<RefreshAllResult> {
    return this.api.post<RefreshAllResult>('/sources/refresh-all', {}, {
      timeoutMs: REFRESH_BACKGROUND_TIMEOUT_MS,
      noLoadingSpinner: true,
    }).pipe(tap(() => this.invalidateSourcesAndProxies()));
  }

  /**
   * 启动后台拉取全部：请求挂在 root 服务上，不随页面销毁取消。
   * 完成用短 toast（无换行，避免弹阻塞对话框）；关标签页后前端无法再提示，后端仍会继续跑完。
   */
  startBackgroundRefreshAll(): boolean {
    if (this.refreshingAll() || this.refreshSubs.size > 0) return false;

    this.refreshingAll.set(true);
    this.refreshAllSub?.unsubscribe();
    this.refreshAllSub = this.refreshAll().subscribe({
      next: res => {
        this.refreshingAll.set(false);
        this.refreshAllSub = null;
        void this.dialog[res.failed > 0 ? 'error' : 'success'](this.formatRefreshAllMsg(res));
      },
      error: (err: Error) => {
        this.refreshingAll.set(false);
        this.refreshAllSub = null;
        void this.dialog.error(err?.message || '拉取全部失败');
      }
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
      .filter(r => !r.ok)
      .map(r => `${r.name}`)
      .slice(0, 3)
      .join('、');
    const more = failed > 3 ? '…' : '';
    return `全部拉取完成：成功 ${ok}，失败 ${failed} / 共 ${total}${fails ? `（${fails}${more}）` : ''}`;
  }

  listProxies(sourceId?: number, forceRefresh = false): Observable<ListResponse<ProxyNode>> {
    // 只缓存全量节点列表（跨页共用的那份）；按源过滤的查询直接透传。
    if (sourceId === null || sourceId === undefined) {
      return this.proxiesCache.get(forceRefresh);
    }
    const params = new HttpParams().set('sourceId', sourceId);
    return this.api.get('/proxies', {
      params,
      context: forceRefresh ? withWtHttpCacheBypass() : undefined
    });
  }

  updateProxy(id: number, enabled: boolean): Observable<ProxyNode> {
    return this.api.put<ProxyNode>(`/proxies/${id}`, { enabled }).pipe(tap(() => this.proxiesCache.invalidate()));
  }

  batchUpdateProxies(ids: number[], enabled: boolean): Observable<{ updated: number }> {
    return this.api.put<{ updated: number }>('/proxies/batch', { ids, enabled }).pipe(tap(() => this.proxiesCache.invalidate()));
  }
}

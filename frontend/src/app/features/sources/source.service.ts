import { HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable, shareReplay, tap } from 'rxjs';
import { ApiService } from '../../core/api.service';
import {
		ListResponse,
		ProxyNode,
		RefreshAllResult,
		RefreshSourceResult,
		RegionCatalogResponse,
		SourceUpsertBody,
		SubscriptionSource,
	} from '../../common/types';

@Injectable({ providedIn: 'root' })
export class SourceService {
	private readonly api = inject(ApiService);

	/**
	 * 地区目录来自编译进后端的 regions.yaml，运行期不变。
	 * 会话内只请求一次（订阅源页 / 规则页共用）；失败不缓存，下次可重试。
	 */
	private regions$: Observable<RegionCatalogResponse> | null = null;

	list(): Observable<ListResponse<SubscriptionSource>> {
		return this.api.get('/sources');
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

	create(body: SourceUpsertBody & { name: string; region: string; url: string }): Observable<SubscriptionSource> {
		return this.api.post('/sources', body);
	}

	update(id: number, body: SourceUpsertBody): Observable<SubscriptionSource> {
		return this.api.put(`/sources/${id}`, body);
	}

	delete(id: number): Observable<{ success: boolean }> {
		return this.api.delete(`/sources/${id}`);
	}

	batchDelete(ids: number[]): Observable<{ deleted: number }> {
		return this.api.post('/sources/batch-delete', { ids });
	}

refresh(id: number): Observable<RefreshSourceResult> {
			return this.api.post(`/sources/${id}/refresh`);
		}

		refreshAll(): Observable<RefreshAllResult> {
			return this.api.post('/sources/refresh-all');
		}

	listProxies(sourceId?: number): Observable<ListResponse<ProxyNode>> {
		const params = sourceId == null ? undefined : new HttpParams().set('sourceId', sourceId);
		return this.api.get(`/proxies${params ? `?${params.toString()}` : ''}`);
	}

	updateProxy(id: number, enabled: boolean): Observable<ProxyNode> {
		return this.api.put(`/proxies/${id}`, { enabled });
	}

	batchUpdateProxies(ids: number[], enabled: boolean): Observable<{ updated: number }> {
		return this.api.put('/proxies/batch', { ids, enabled });
	}
}

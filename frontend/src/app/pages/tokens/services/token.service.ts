import { Injectable, inject } from '@angular/core';
import { ApiService } from '@common/net/api.service';
import { CachedRequest } from '@common/net/cached-request';
import {
	ListResponse,
	ShareToken,
	TokenGroupMode,
	TokenUpsertBody,
} from '@data-struct';
import { Observable, tap } from 'rxjs';

@Injectable({ providedIn: 'root' })
export class TokenService {
	private readonly api = inject(ApiService);

// 令牌列表被概览页与令牌页共用，做会话内缓存，写操作后失效。
		private readonly listCache = new CachedRequest<ListResponse<ShareToken>>((context) =>
			this.api.get('/tokens', { context }),
		);

	list(forceRefresh = false): Observable<ListResponse<ShareToken>> {
		return this.listCache.get(forceRefresh);
	}

	/** 包裹写请求：成功后失效列表缓存。 */
	private afterWrite<T>(req: Observable<T>): Observable<T> {
		return req.pipe(tap(() => this.listCache.invalidate()));
	}

	create(
		name: string,
		sourceIds: number[] = [],
		groupMode: TokenGroupMode = 'auto',
		groupNames: string[] = [],
	): Observable<ShareToken> {
		return this.afterWrite(this.api.post('/tokens', { name, sourceIds, groupMode, groupNames }));
	}

	update(id: number, body: TokenUpsertBody): Observable<ShareToken> {
		return this.afterWrite(this.api.put(`/tokens/${id}`, body));
	}

	revoke(id: number): Observable<ShareToken> {
		return this.afterWrite(this.api.post(`/tokens/${id}/revoke`));
	}

	regenerate(id: number): Observable<ShareToken> {
		return this.afterWrite(this.api.post(`/tokens/${id}/regenerate`));
	}

	delete(id: number): Observable<{ success: boolean }> {
		return this.afterWrite(this.api.delete(`/tokens/${id}`));
	}
}

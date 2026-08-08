import { Injectable, inject } from '@angular/core';
import { ApiService } from '@common/net/api.service';
import { CachedRequest } from '@common/net/cached-request';
import {
	APIKey,
	APIKeySecret,
	APIKeyScope,
	APIKeyUpsertBody,
	ListResponse,
} from '@data-struct';
import { Observable, tap } from 'rxjs';

@Injectable({ providedIn: 'root' })
export class ApiKeyService {
	private readonly api = inject(ApiService);

private readonly listCache = new CachedRequest<ListResponse<APIKey>>((context) =>
			this.api.get('/apikeys', { context }),
		);

	list(forceRefresh = false): Observable<ListResponse<APIKey>> {
		return this.listCache.get(forceRefresh);
	}

	private afterWrite<T>(req: Observable<T>): Observable<T> {
		return req.pipe(tap(() => this.listCache.invalidate()));
	}

	create(
		name: string,
		scopes: APIKeyScope[],
		note = '',
		expiresAt?: string | null,
	): Observable<APIKey> {
		const body: Record<string, unknown> = { name, scopes, note };
		if (expiresAt) {
			body['expiresAt'] = expiresAt;
		}
		return this.afterWrite(this.api.post('/apikeys', body));
	}

	update(id: number, body: APIKeyUpsertBody): Observable<APIKey> {
		return this.afterWrite(this.api.put(`/apikeys/${id}`, body));
	}

	secret(id: number): Observable<APIKeySecret> {
		return this.api.get(`/apikeys/${id}/secret`);
	}

	revoke(id: number): Observable<APIKey> {
		return this.afterWrite(this.api.post(`/apikeys/${id}/revoke`));
	}

	regenerate(id: number): Observable<APIKey> {
		return this.afterWrite(this.api.post(`/apikeys/${id}/regenerate`));
	}

	delete(id: number): Observable<{ success: boolean }> {
		return this.afterWrite(this.api.delete(`/apikeys/${id}`));
	}
}

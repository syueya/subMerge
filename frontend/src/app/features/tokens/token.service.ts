import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiService } from '../../core/api.service';
import { ListResponse, ShareToken, TokenGroupMode, TokenStatus } from '../../common/types';

export interface TokenUpsertBody {
	name?: string;
	status?: TokenStatus;
	/** 传 [] 表示全部源；非空为指定源 */
	sourceIds?: number[];
	groupMode?: TokenGroupMode;
	/** custom 时的策略组名白名单 */
	groupNames?: string[];
}

@Injectable({ providedIn: 'root' })
export class TokenService {
	private readonly api = inject(ApiService);

	list(): Observable<ListResponse<ShareToken>> {
		return this.api.get('/tokens');
	}

	create(
		name: string,
		sourceIds: number[] = [],
		groupMode: TokenGroupMode = 'auto',
		groupNames: string[] = [],
	): Observable<ShareToken> {
		return this.api.post('/tokens', { name, sourceIds, groupMode, groupNames });
	}

	update(id: number, body: TokenUpsertBody): Observable<ShareToken> {
		return this.api.put(`/tokens/${id}`, body);
	}

	revoke(id: number): Observable<ShareToken> {
		return this.api.post(`/tokens/${id}/revoke`);
	}

	regenerate(id: number): Observable<ShareToken> {
		return this.api.post(`/tokens/${id}/regenerate`);
	}

	delete(id: number): Observable<{ success: boolean }> {
		return this.api.delete(`/tokens/${id}`);
	}
}

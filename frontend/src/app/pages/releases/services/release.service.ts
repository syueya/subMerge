import { Injectable, inject } from '@angular/core';
import { Observable, tap } from 'rxjs';
import { ApiService } from '@common/net/api.service';
import { CachedRequest } from '@common/net/cached-request';
import {
	DraftStatus,
	PublishResponse,
	Release,
	ReleaseDetail,
	ReleasePreview,
	ListResponse,
} from '@data-struct';

@Injectable({ providedIn: 'root' })
export class ReleaseService {
	private readonly api = inject(ApiService);

	// 发布历史列表被概览页与发布页共用，做会话内缓存，发布/回滚后失效。
	private readonly listCache = new CachedRequest<ListResponse<Release>>(() =>
		this.api.get('/releases'),
	);

	list(forceRefresh = false): Observable<ListResponse<Release>> {
		return this.listCache.get(forceRefresh);
	}

	get(id: number): Observable<ReleaseDetail> {
		return this.api.get(`/releases/${id}`);
	}

	current(): Observable<ReleaseDetail> {
		return this.api.get('/releases/current');
	}

	preview(): Observable<ReleasePreview> {
		return this.api.get('/releases/preview');
	}

	draftStatus(): Observable<DraftStatus> {
		return this.api.get('/releases/draft-status');
	}

	publish(note?: string): Observable<PublishResponse> {
		return this.api
			.post<PublishResponse>('/releases/publish', { note: note || '' })
			.pipe(tap(() => this.listCache.invalidate()));
	}

	rollback(id: number): Observable<Release> {
		return this.api
			.post<Release>(`/releases/${id}/rollback`)
			.pipe(tap(() => this.listCache.invalidate()));
	}
}

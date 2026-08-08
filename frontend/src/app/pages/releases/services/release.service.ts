import { Injectable, inject } from '@angular/core';
import { ApiService } from '@common/net/api.service';
import { CachedRequest } from '@common/net/cached-request';
import { withWtHttpCacheBypass } from '@common/net/session-http-cache';
import {
	DraftStatus,
	PublishResponse,
	Release,
	ReleaseDetail,
	ReleasePreview,
	ListResponse,
} from '@data-struct';
import { Observable, tap } from 'rxjs';

@Injectable({ providedIn: 'root' })
export class ReleaseService {
	private readonly api = inject(ApiService);

// 发布历史列表被概览页与发布页共用，做会话内缓存，发布/回滚后失效。
		private readonly listCache = new CachedRequest<ListResponse<Release>>((context) =>
			this.api.get('/releases', { context }),
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

	/**
	 * 草稿状态。
	 * withChanges=false（默认）：只返回 dirty/hash，进页轻量用。
	 * withChanges=true：dirty 时附带实体级变更列表（查看差异 / 发布确认）。
	 * 始终 bypass 会话 HTTP 缓存：dirty 随写操作变化，不能吃 5 分钟旧结果。
	 * 仍标为读接口（session-http-cache isRead），不挡全屏 loading。
	 */
	draftStatus(withChanges = false): Observable<DraftStatus> {
		const q = withChanges ? '?changes=1' : '';
		return this.api.get(`/releases/draft-status${q}`, {
			context: withWtHttpCacheBypass(),
		});
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

	delete(id: number): Observable<void> {
		return this.api
			.delete<void>(`/releases/${id}`)
			.pipe(tap(() => this.listCache.invalidate()));
	}
}

import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiService } from '../../core/api.service';
import {
	DraftStatus,
	PublishResponse,
	Release,
	ReleaseDetail,
	ReleasePreview,
	ListResponse,
} from '../../common/types';

@Injectable({ providedIn: 'root' })
export class ReleaseService {
	private readonly api = inject(ApiService);

	list(): Observable<ListResponse<Release>> {
		return this.api.get('/releases');
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
		return this.api.post('/releases/publish', { note: note || '' });
	}

	rollback(id: number): Observable<Release> {
		return this.api.post(`/releases/${id}/rollback`);
	}
}

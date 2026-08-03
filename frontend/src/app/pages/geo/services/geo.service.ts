import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import {
	GeoCategoriesResponse,
	GeoQueryResponse,
	GeoReverseResponse,
	GeoSearchResponse,
	GeoStatus,
	GeoUpdateResponse,
} from '@data-struct';
import { ApiService } from '@common/net/api.service';

@Injectable({ providedIn: 'root' })
export class GeoService {
	private readonly api = inject(ApiService);

	status(): Observable<GeoStatus[]> {
		return this.api.get<GeoStatus[]>('/geo/status');
	}

	categories(): Observable<GeoCategoriesResponse> {
		return this.api.get<GeoCategoriesResponse>('/geo/categories');
	}

	query(domain: string, resolve: boolean): Observable<GeoQueryResponse> {
		return this.api.post<GeoQueryResponse>('/geo/query', { domain, resolve });
	}

	reverse(file: string, category: string, limit = 100, offset = 0): Observable<GeoReverseResponse> {
		return this.api.post<GeoReverseResponse>('/geo/reverse', { file, category, limit, offset });
	}

	search(file: string, field: string, keyword: string, limit = 100, offset = 0): Observable<GeoSearchResponse> {
		return this.api.post<GeoSearchResponse>('/geo/search', { file, field, keyword, limit, offset });
	}

	update(): Observable<GeoUpdateResponse> {
		return this.api.post<GeoUpdateResponse>('/geo/update', {});
	}
}

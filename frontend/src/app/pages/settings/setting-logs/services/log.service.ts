import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiService } from '@common/net/api.service';
import { withWtHttpCacheBypass } from '@common/net';
import { SystemLogsContent, SystemLogsTypeData } from '@data-struct';

@Injectable({ providedIn: 'root' })
export class LogService {
	private readonly api = inject(ApiService);

	list(name = '', bypassCache = false): Observable<SystemLogsTypeData> {
		const params: Record<string, string> = {};
		if (name.trim()) {
			params['name'] = name.trim();
		}
		return this.api.get<SystemLogsTypeData>('/logs', {
			params,
			timeoutMs: 15000,
			context: bypassCache ? withWtHttpCacheBypass() : undefined,
		});
	}

	details(name: string, line: number, bypassCache = false): Observable<SystemLogsContent> {
		return this.api.get<SystemLogsContent>('/logs/details', {
			params: { name, line },
			timeoutMs: 15000,
			context: bypassCache ? withWtHttpCacheBypass() : undefined,
		});
	}
}

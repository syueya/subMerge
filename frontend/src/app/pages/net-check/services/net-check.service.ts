import { Injectable, inject } from '@angular/core';
import { ApiService } from '@common/net/api.service';
import { NetCheckConfig, NetCheckRequest, NetCheckResponse } from '@data-struct';
import { Observable } from 'rxjs';

@Injectable({ providedIn: 'root' })
export class NetCheckService {
  private readonly api = inject(ApiService);

  config(): Observable<NetCheckConfig> {
    return this.api.get<NetCheckConfig>('/net-check/config');
  }

  saveConfig(config: NetCheckConfig): Observable<NetCheckConfig> {
    return this.api.put<NetCheckConfig>('/net-check/config', config);
  }

  check(request: NetCheckRequest): Observable<NetCheckResponse> {
    return this.api.post<NetCheckResponse>('/net-check/check', request);
  }

  reset(): Observable<NetCheckConfig> {
    return this.api.post<NetCheckConfig>('/net-check/reset', {});
  }
}

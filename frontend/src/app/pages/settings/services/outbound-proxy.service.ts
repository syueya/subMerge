import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiService } from '@common/net/api.service';
import { OutboundProxyUpdate, OutboundProxyView } from '@data-struct';

@Injectable({ providedIn: 'root' })
export class OutboundProxyService {
  private readonly api = inject(ApiService);

  get(): Observable<OutboundProxyView> {
    return this.api.get<OutboundProxyView>('/outbound-proxy');
  }

  save(body: OutboundProxyUpdate): Observable<OutboundProxyView> {
    return this.api.put<OutboundProxyView>('/outbound-proxy', body);
  }

  reset(): Observable<OutboundProxyView> {
    return this.api.post<OutboundProxyView>('/outbound-proxy/reset', {});
  }
}

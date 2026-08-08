import { Injectable, inject, signal } from '@angular/core';
import { Observable, Subscription } from 'rxjs';
import { GeoCategoriesResponse, GeoIPGeoResponse, GeoQueryResponse, GeoReverseResponse, GeoSearchResponse, GeoStatus, GeoUpdateResponse } from '@data-struct';
import { ApiService } from '@common/net/api.service';
import { DialogService } from '@common/services/dialog.service';

const GEO_RESOURCE_NAME: Record<string, string> = {
  'geoip.dat': 'GeoIP',
  'geosite.dat': 'GeoSite',
  'geoip.metadb': 'MetaDB',
  'GeoLite2-ASN.mmdb': 'ASN'
};

@Injectable({ providedIn: 'root' })
export class GeoService {
  private readonly api = inject(ApiService);
  private readonly dialog = inject(DialogService);
  private updateSub: Subscription | null = null;

  /** 跨页面共享：后台更新中状态不随页面销毁丢失 */
  readonly updating = signal(false);

  status(): Observable<GeoStatus[]> {
    return this.api.get<GeoStatus[]>('/geo/status');
  }

  categories(): Observable<GeoCategoriesResponse> {
    return this.api.get<GeoCategoriesResponse>('/geo/categories');
  }

  query(domain: string, resolve: boolean): Observable<GeoQueryResponse> {
    return this.api.post<GeoQueryResponse>('/geo/query', { domain, resolve });
  }

  lookupIPGeo(ip: string): Observable<GeoIPGeoResponse> {
    return this.api.post<GeoIPGeoResponse>('/geo/ip-geo', { ip }, { noLoadingSpinner: true });
  }

  reverse(file: string, category: string, limit = 100, offset = 0): Observable<GeoReverseResponse> {
    return this.api.post<GeoReverseResponse>('/geo/reverse', { file, category, limit, offset });
  }

  search(file: string, field: string, keyword: string, limit = 100, offset = 0): Observable<GeoSearchResponse> {
    return this.api.post<GeoSearchResponse>('/geo/search', { file, field, keyword, limit, offset });
  }

  /**
   * 启动后台更新：请求挂在 root 服务上，不随页面销毁取消。
   * 完成用短 toast（无换行，避免弹阻塞对话框）；关标签页后前端无法再提示。
   */
  startBackgroundUpdate(): boolean {
    if (this.updating()) return false;

    this.updating.set(true);
    this.updateSub?.unsubscribe();
    this.updateSub = this.api.post<GeoUpdateResponse>('/geo/update', {}, { noLoadingSpinner: true }).subscribe({
      next: result => {
        this.updating.set(false);
        this.updateSub = null;
        void this.dialog[this.hasFailedItems(result) ? 'error' : 'success'](this.formatUpdateMsg(result));
      },
      error: (err: Error) => {
        this.updating.set(false);
        this.updateSub = null;
        void this.dialog.error(err?.message || 'Geo 数据更新失败');
      }
    });
    return true;
  }

  /** 单行摘要，走 message toast（与拉取成功短提示同属 DialogService） */
  private formatUpdateMsg(result: GeoUpdateResponse): string {
    const items = result.items || [];
    if (!items.length) return 'Geo 数据更新完成';
    const ok = items.filter(item => item.updated).length;
    const fail = items.length - ok;
    const detail = items.map(item => `${this.resourceLabel(item.name)}${item.updated ? '✓' : '✗'}`).join(' ');
    if (fail === 0) return `Geo 数据更新成功（${ok}/${items.length}） ${detail}`;
    return `Geo 数据更新完成：成功 ${ok}，失败 ${fail} ${detail}`;
  }

  private hasFailedItems(result: GeoUpdateResponse): boolean {
    return (result.items || []).some(item => !item.updated);
  }

  resourceLabel(name: string): string {
    return GEO_RESOURCE_NAME[name] || name;
  }
}

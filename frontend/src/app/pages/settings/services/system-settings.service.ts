import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiService } from '@common/net/api.service';
import { SystemSettingsUpdate, SystemSettingsView } from '@data-struct';

@Injectable({ providedIn: 'root' })
export class SystemSettingsService {
  private readonly api = inject(ApiService);

  get(): Observable<SystemSettingsView> {
    return this.api.get<SystemSettingsView>('/system-settings');
  }

  save(body: SystemSettingsUpdate): Observable<SystemSettingsView> {
    return this.api.put<SystemSettingsView>('/system-settings', body);
  }

  reset(): Observable<SystemSettingsView> {
    return this.api.post<SystemSettingsView>('/system-settings/reset', {});
  }
}

import { registerLocaleData } from '@angular/common';
import { provideHttpClient, withInterceptors, withXhr } from '@angular/common/http';
import localeZh from '@angular/common/locales/zh';
import { ApplicationConfig, provideZoneChangeDetection, importProvidersFrom, isDevMode } from '@angular/core';
import { provideRouter, withComponentInputBinding, withInMemoryScrolling } from '@angular/router';
import { provideServiceWorker } from '@angular/service-worker';
import { MATERIAL_PROVIDERS } from '@common/material.module';
import { CustomTablerIcons } from '@common/modules/icons/import-config';
import { LoadingModule } from '@common/modules/loading/loading.module';
import { defaultInterceptor, mockBackendInterceptor, sessionHttpCacheInterceptor } from '@common/net';
import { CmProviders } from '@common/providers';
import { provideTablerIcons } from '@luoxiao123/angular-tabler-icons';

import { routes } from './app.routes';
registerLocaleData(localeZh);

export const appConfig: ApplicationConfig = {
  providers: [
    CmProviders,
    provideZoneChangeDetection({ eventCoalescing: true }),
    provideRouter(
      routes,
      withInMemoryScrolling({
        scrollPositionRestoration: 'enabled',
        anchorScrolling: 'enabled'
      }),
      withComponentInputBinding()
    ),
    // Angular 22 默认走 FetchBackend，且只引入 zone.js 时不会 patch fetch。
    // HTTP 完成回调可能落在 NgZone 外，导致 isLoading 已 false 但进度条不刷新。
    // 显式切回 XHR，继续走 zone 补丁的 XMLHttpRequest，保证视图能及时更新。
    // sessionHttpCache 在 default 之后：统一鉴权/超时后再做会话读缓存，避免重复打列表接口
    provideHttpClient(
      withInterceptors([mockBackendInterceptor, defaultInterceptor, sessionHttpCacheInterceptor]),
      withXhr()
    ),
    provideTablerIcons(CustomTablerIcons),
    ...MATERIAL_PROVIDERS,

    importProvidersFrom(LoadingModule.forRoot()),
    provideServiceWorker('ngsw-worker.js', {
      enabled: !isDevMode(),
      registrationStrategy: 'registerWhenStable:30000'
    })
  ]
};

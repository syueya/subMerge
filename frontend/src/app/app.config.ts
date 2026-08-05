import { registerLocaleData } from '@angular/common';
import { provideHttpClient, withInterceptors, withXhr } from '@angular/common/http';
import localeZh from '@angular/common/locales/zh';
import { ApplicationConfig, provideZoneChangeDetection, importProvidersFrom, isDevMode } from '@angular/core';
import {
  PreloadAllModules,
  provideRouter,
  withComponentInputBinding,
  withInMemoryScrolling,
  withPreloading,
} from '@angular/router';
import { provideServiceWorker } from '@angular/service-worker';
import { MATERIAL_PROVIDERS } from '@common/material.module';
import { CustomTablerIcons } from '@common/modules/icons/import-config';
import { LoadingModule } from '@common/modules/loading/loading.module';
import { pendingRequestsInterceptorFn } from '@common/modules/loading/services/pending-requests-interceptor.service';
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
      withComponentInputBinding(),
      // 登录后空闲预加载各业务 chunk，减轻首次点菜单等待
      withPreloading(PreloadAllModules)
    ),
    // Angular 22 默认 FetchBackend 未 patch fetch；显式 XHR + default 内唯一 Zone 出口。
    // 顺序：pending(全屏) → mock(可选) → default(鉴权/超时/Zone) → sessionCache(读缓存)
    provideHttpClient(
      withInterceptors([pendingRequestsInterceptorFn, mockBackendInterceptor, defaultInterceptor, sessionHttpCacheInterceptor]),
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

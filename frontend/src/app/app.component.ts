import { Component, inject, OnInit } from '@angular/core';
import { NavigationEnd, NavigationError, RouteConfigLoadStart, Router, RouterOutlet } from '@angular/router';
import { LoadingModule } from '@common/modules/loading/loading.module';
import { CmNotificationService } from '@common/modules/notification';
import { CmParentComponent } from '@common/parents/parent/parent.component';
import { environment } from '@env/environment';
import { takeUntil } from 'rxjs';

import { stepPreloader } from './common/preloader/preloader';

@Component({
    selector: 'app-root',
    imports: [LoadingModule, RouterOutlet],
    templateUrl: './app.component.html'
})
export class AppComponent extends CmParentComponent implements OnInit {

  private readonly router = inject(Router);

  private readonly notification = inject(CmNotificationService);

  title = 'Angular';

  private donePreloader = stepPreloader();

  constructor() {
    super();
  }

  override ngOnInit(): void {
    let configLoad = false;
    this.router.events.pipe(takeUntil(this.$destroy)).subscribe(ev => {
      if (ev instanceof RouteConfigLoadStart) {
        configLoad = true;
      }
      if (configLoad && ev instanceof NavigationError) {
        // 开发环境打印真实 reason，便于定位懒加载失败根因
        console.error('NavigationError', ev.url, ev.error);
        const detail = !environment.production && ev.error
          ? `${ev.url} | ${ev.error?.message || ev.error}`
          : ev.url;
        const error = environment.production
          ? `加载失败，可能Angular系统版本已更新，请先刷新页面重试。`
          : `无法加载路由：${detail}`;
        this.notification.error('页面加载失败', error);
      }
      if (ev instanceof NavigationEnd) {
        this.donePreloader();
      }
    });
  }

}

import { CommonModule } from '@angular/common';
import { Component, ViewChild, ViewEncapsulation, OnDestroy, inject } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatListModule } from '@angular/material/list';
import { MatSidenav, MatSidenavContent, MatSidenavModule } from '@angular/material/sidenav';
import { MatTooltipModule } from '@angular/material/tooltip';
import { NavigationEnd, Router, RouterModule } from '@angular/router';
import { MenuList } from '@common/data/menu-list';
import { MenuItem } from '@common/interfaces';
import { IconsModule } from '@common/modules/icons/icons.module';
import { CmParentComponent } from '@common/parents/parent/parent.component';
import { AuthService } from '@common/services';
import { CoreService } from '@common/services/core.service';
import { ScreenSizeService } from '@common/services/screen-size.service';
import { ScrollService } from '@common/services/scroll.service';
import { UserRoleEnum } from '@data-struct';
import { NgScrollbarModule } from 'ngx-scrollbar';
import { fromEvent, Subscription } from 'rxjs';
import { debounceTime, distinctUntilChanged, filter, map, takeUntil } from 'rxjs/operators';

import { AppSettings } from '../../config';
import { AppBreadcrumbComponent } from './shared/breadcrumb/breadcrumb.component';
import { SidenavAccountInfoComponent } from './shared/sidenav-account-info/sidenav-account-info.component';
import { AppNavItemComponent } from './vertical/sidebar/nav-item/nav-item.component';
import { SidebarComponent } from './vertical/sidebar/sidebar.component';


@Component({
  selector: 'app-full',
  imports: [
    RouterModule,
    AppNavItemComponent,
    CommonModule,
    MatButtonModule,
    MatListModule,
    MatSidenavModule,
    MatTooltipModule,
    SidebarComponent,
    NgScrollbarModule,
    IconsModule,
    AppBreadcrumbComponent,
    SidenavAccountInfoComponent
  ],
  templateUrl: './full.component.html',
  styleUrls: [],
  encapsulation: ViewEncapsulation.None
})
export class FullComponent extends CmParentComponent implements OnDestroy {
  private settings = inject(CoreService);
  private router = inject(Router);
  private authService = inject(AuthService);
  private scrollService = inject(ScrollService);
  private screenSizeService = inject(ScreenSizeService);

  /* 主菜单 */
  mainMenuList!: MenuItem[];

  @ViewChild('leftsidenav')
  public sidenav!: MatSidenav;
  resView = false;
  @ViewChild('content', { static: true }) content!: MatSidenavContent;
  //get options from service
  options: AppSettings;
  private layoutChangesSubscription = Subscription.EMPTY;
  private isMobileScreen = false;
  private isContentWidthFixed = true;
  private isCollapsedWidthFixed = false;
  private htmlElement!: HTMLHtmlElement;
  private screenSizeSubscription = Subscription.EMPTY;


  get isOver(): boolean {
    return this.isMobileScreen;
  }

  get isTablet(): boolean {
    return this.resView;
  }

  /**
   * 显示返回到顶部按钮
   */
  showBackToTop = false;

  constructor() {
    super();
    this.options = this.settings.getOptions();
    this.htmlElement = document.querySelector('html')!;


    // 订阅屏幕尺寸变化
    this.screenSizeSubscription = this.screenSizeService.screenSize$.pipe(takeUntil(this.$destroy)).subscribe(info => {
      // SidenavOpened must be reset true when layout changes
      this.options.sidenavOpened = true;

      if (info.width >= 1024) {
        this.isMobileScreen = false;
        this.isContentWidthFixed = true;
        this.resView = false;
      } else {
        this.isMobileScreen = true;

        this.isContentWidthFixed = false;
        this.resView = false;
      }
    });

    //订阅用户信息 生成权限菜单
    this.authService.$userInfo.pipe(takeUntil(this.$destroy)).subscribe(userInfo => {
      const OrgMenuList = JSON.parse(JSON.stringify(MenuList)) || [];
      // 根据 userInfo 过滤 sidebarMenu
      if (userInfo) {
        if (Number(userInfo.role) === UserRoleEnum.Admin) {
          // 管理员返回全部菜单
          this.mainMenuList = OrgMenuList;
        } else {
          this.mainMenuList = OrgMenuList.filter((item: MenuItem) => {
            if (item.isAdminOnly) {
              return false; // 过滤掉 isAdminOnly 为 true 的对象
            } else if (item.children) {
              // 过滤子对象数组中的 isAdminOnly 为 true 的对象
              item.children = item.children.filter((child: MenuItem) => !child.isAdminOnly);
            }
            return true;
          });
        }
      } else {
        this.mainMenuList = []; // 如果 userInfo 不存在，则清空 sidebarMenu
      }
    });

    // 初始化主题
    this.applyThemeOptions(this.options);

    // This is for scroll to top
    this.router.events
      .pipe(takeUntil(this.$destroy))
      .pipe(filter(event => event instanceof NavigationEnd))
      .subscribe(e => {
        this.content.scrollTo({ top: 0 });
      });
  }

  override handlerAfterViewInit() {
    const element = this.content?.getElementRef()?.nativeElement;

    // 使用 RxJS 监听滚动事件并进行防抖处理
    fromEvent(element, 'scroll')
      .pipe(
        debounceTime(100),
        map(() => ({ scrollTop: element.scrollTop, scrollHeight: element.scrollHeight, clientHeight: element.clientHeight })),
        distinctUntilChanged((prev, curr) => prev.scrollTop === curr.scrollTop && prev.scrollHeight === curr.scrollHeight && prev.clientHeight === curr.clientHeight),
        takeUntil(this.$destroy)
      )
      .subscribe(({ scrollTop, scrollHeight, clientHeight }) => {
        this.showBackToTop = scrollTop > 200; // 显示返回顶部按钮
        this.scrollService.notifyScroll({ scrollTop, scrollHeight, clientHeight }); // 通知滚动事件
      });

    this.scrollService.triggerToTop$.pipe(takeUntil(this.$destroy)).subscribe(() => {
      this.scrollToTop(0); // 触发返回顶部
    });
  }

  override ngOnDestroy() {
    super.ngOnDestroy();
    this.screenSizeSubscription.unsubscribe(); // 取消屏幕尺寸订阅
    this.layoutChangesSubscription.unsubscribe(); // 取消布局变化订阅
  }

  toggleCollapsed() {
    this.isContentWidthFixed = false;
    this.options.sidenavCollapsed = !this.options.sidenavCollapsed;
    this.resetCollapsedState();
  }

  resetCollapsedState(timer = 200) {
    setTimeout(() => this.settings.setOptions(this.options), timer);
  }

  onSidenavClosedStart() {
    this.isContentWidthFixed = false;
  }

  onSidenavOpenedChange(isOpened: boolean) {
    this.isCollapsedWidthFixed = !this.isOver;
    this.options.sidenavOpened = isOpened;
    this.settings.setOptions(this.options);
  }

  /**
   * 应用主题配置（浅/深色 + 主题色）
   */
  applyThemeOptions(options: AppSettings): void {
    this.options = options;
    this.toggleDarkTheme(options);
    this.toggleColorsTheme(options);
  }

  toggleDarkTheme(options: AppSettings) {
    if (options.theme === 'dark') {
      this.htmlElement.classList.add('dark-theme');
      this.htmlElement.classList.remove('light-theme');
    } else {
      this.htmlElement.classList.remove('dark-theme');
      this.htmlElement.classList.add('light-theme');
    }
  }

  toggleColorsTheme(options: AppSettings) {
    // Remove any existing theme class dynamically
    this.htmlElement.classList.forEach(className => {
      if (className.endsWith('_theme')) {
        this.htmlElement.classList.remove(className);
      }
    });

    // Add the selected theme class
    this.htmlElement.classList.add(options.activeTheme);
  }

  /**
   * 滚动到顶部
   * duration 滚动持续时间（毫秒）
   */
  scrollToTop(duration = 0) {
    const element = this.content?.getElementRef()?.nativeElement;
    if (element) {
      if (duration) {
        const start = element.scrollTop;
        const startTime = performance.now();

        const easeInOutQuad = (t: number) => (t < 0.5 ? 2 * t * t : -1 + (4 - 2 * t) * t);

        const scrollStep = (timestamp: number) => {
          const progress = timestamp - startTime;
          const percent = Math.min(progress / duration, 1);
          const easedPercent = easeInOutQuad(percent);
          element.scrollTop = start * (1 - easedPercent);

          if (progress < duration) {
            requestAnimationFrame(scrollStep);
          }
        };

        requestAnimationFrame(scrollStep);

        this.showBackToTop = false;
      } else {
        element.scrollTop = 0;
      }
    }
  }
}

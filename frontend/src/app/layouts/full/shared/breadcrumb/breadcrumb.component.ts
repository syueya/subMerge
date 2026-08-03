import { Component, inject } from '@angular/core';
import { Title } from '@angular/platform-browser';
import { RouterModule, Router, NavigationEnd, ActivatedRoute, Data } from '@angular/router';
import MenuList from '@common/data/menu-list';
import { MenuItem } from '@common/interfaces';
import { IconsModule } from '@common/modules/icons/icons.module';
import { CmParentComponent } from '@common/parents/parent/parent.component';
import { filter, map, mergeMap, takeUntil } from 'rxjs/operators';

@Component({
    selector: 'app-breadcrumb',
    imports: [RouterModule, IconsModule],
    templateUrl: './breadcrumb.component.html',
    styleUrl: './breadcrumb.component.scss'
})
export class AppBreadcrumbComponent extends CmParentComponent {
  private router = inject(Router);
  private activatedRoute = inject(ActivatedRoute);
  private titleService = inject(Title);

  pageTitle!: string;
  // @Input() layout;
  pageInfo: Data = Object.create(null);
  breadcrumbs: Array<{ title: string; url: string }> = [];
  constructor() {
    super();
    this.router.events
      .pipe(takeUntil(this.$destroy))
      .pipe(filter(event => event instanceof NavigationEnd))
      .pipe(map(() => this.activatedRoute))
      .pipe(
        map(route => {
          while (route.firstChild) {
            route = route.firstChild;
          }
          return route;
        })
      )
      .pipe(filter(route => route.outlet === 'primary'))
      .pipe(mergeMap(route => route.data))

      .subscribe(event => {
        this.pageInfo = event;
        // 隐藏面包屑时也要清空旧状态，避免切页后残留上一页标题/路径
        if (event?.['hideBreadcrumb']) {
          this.pageTitle = '';
          this.breadcrumbs = [];
          if (event['title']) {
            this.titleService.setTitle(`${event['title']} - SubMerge`);
          }
          return;
        }
        if (event['title']) {
          this.pageTitle = event['title'];
          this.titleService.setTitle(`${event['title']} - SubMerge`);
        }
        this.breadcrumbs = this.generateBreadcrumbs(this.router.url, event);
      });
  }

  private generateBreadcrumbs(url: string, routeData: Data): Array<{ title: string; url: string }> {
    const breadcrumbs: Array<{ title: string; url: string }> = [];

    if (routeData?.['title'] || routeData?.['urls']) {
      //优先通过路由传值
      if (routeData?.['urls']) {
        for (const url of routeData['urls']) {
          breadcrumbs.push({ title: url.title, url: url.url });
        }
      } else {
        breadcrumbs.push({ title: routeData?.['title'], url: this.router.url });
      }
    } else {
      //通过菜单查找
      const urlSegments = url.split('/').filter(segment => segment);

      let currentPath = '';
      let isFoundInMenu = false;
      for (const segment of urlSegments) {
        currentPath += `/${segment}`;
        const menuItem = this.findMenuItem(currentPath, MenuList);
        if (menuItem) {
          breadcrumbs.push({ title: menuItem.displayName!, url: menuItem.route! });
          isFoundInMenu = true;
          if (menuItem.displayName) {
            this.pageTitle = menuItem.displayName;
            this.titleService.setTitle(`${menuItem.displayName} - SubMerge`);
          }
        }
      }

      if (isFoundInMenu) {
        // 通过菜单生成的路径自动添加首页
        if (breadcrumbs.length > 0 && breadcrumbs[0].title !== '首页') {
          breadcrumbs.unshift({ title: '首页', url: '/main/dashboard' });
        }
      }
    }

    return breadcrumbs;
  }

  private findMenuItem(route: string, menuList: MenuItem[]): MenuItem | null {
    for (const menuItem of menuList) {
      if (menuItem.route === route) {
        return menuItem;
      }
      if (menuItem.children) {
        const childMenuItem = this.findMenuItem(route, menuItem.children);
        if (childMenuItem) {
          return childMenuItem;
        }
      }
    }
    return null;
  }
}

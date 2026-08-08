import { CommonModule } from '@angular/common';
import { Component, DestroyRef, HostBinding, Input, Output, EventEmitter, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { MatListModule } from '@angular/material/list';
import { MatTooltipModule } from '@angular/material/tooltip';
import { NavigationEnd, Router, RouterModule } from '@angular/router';
import { MenuItem } from '@common/interfaces';
import { IconsModule } from '@common/modules/icons/icons.module';
import { CoreService } from '@common/services/core.service';
import { filter } from 'rxjs/operators';

@Component({
  selector: 'app-nav-item',
  imports: [IconsModule, MatListModule, MatTooltipModule, CommonModule, RouterModule],
  templateUrl: './nav-item.component.html',
  styleUrls: []
})
export class AppNavItemComponent {
  router = inject(Router);
  private destroyRef = inject(DestroyRef);
  private settings = inject(CoreService);
  readonly navigationUrl = signal(this.router.url);

  @Output() readonly toggleMobileLink = new EventEmitter<void>();
  @Output() readonly notify: EventEmitter<boolean> = new EventEmitter<boolean>();

  @HostBinding('attr.aria-expanded') ariaExpanded = true;
  @Input() item!: MenuItem;
  /**
   * 菜单深度
   */
  @Input() depth = 0;

  constructor() {
    this.router.events
      .pipe(
        filter((event): event is NavigationEnd => event instanceof NavigationEnd),
        takeUntilDestroyed(this.destroyRef)
      )
      .subscribe(event => this.navigationUrl.set(event.urlAfterRedirects));
  }

  isActiveRoute(route: string | undefined): boolean {
    if (!route) {
      return false;
    }

    const normalizeUrl = (url: string) => url.split(/[?#]/, 1)[0].replace(/\/$/, '') || '/';
    return normalizeUrl(this.navigationUrl()) === normalizeUrl(route);
  }

  /**
   * 是否为一级分组（有子菜单，仅作标题展示）
   */
  isGroupItem(item: MenuItem | null | undefined): boolean {
    return !!(item?.children && item.children.length);
  }

  /**
   * 收起态仅显示图标时，用 tooltip 展示菜单名
   */
  showNameTooltip(): boolean {
    return !!this.settings.getOptions().sidenavCollapsed;
  }

  onItemSelected(item: MenuItem) {
    if (item.route) {
      this.router.navigate([item.route]);
    }

    window.scroll({
      top: 0,
      left: 0,
      behavior: 'smooth'
    });

    if (window.innerWidth < 1024) {
      this.notify.emit();
    }
  }
}

/* eslint-disable @typescript-eslint/no-explicit-any */

import { Overlay, OverlayRef, PositionStrategy, ConnectionPositionPair } from '@angular/cdk/overlay';
import { TemplatePortal } from '@angular/cdk/portal';
import { Directive, Input, Output, EventEmitter, ViewContainerRef, ElementRef, HostListener, inject } from '@angular/core';
import { MatMenu } from '@angular/material/menu';
import { Subscription, fromEvent } from 'rxjs';
import { filter, take } from 'rxjs/operators';

@Directive({
  selector: '[cmContextmenu]',
  standalone: false
})
export class CmContextMenuDirective {
  private overlay = inject(Overlay);
  private viewContainerRef = inject(ViewContainerRef);
  private eletemRef = inject(ElementRef);

  @Input('cmContextmenu')
  contextMenu!: MatMenu;

  @Input('cmContextmenuData')
  menuData: any;
  // 打开事件
  @Output()
  readonly opendMenuEvent = new EventEmitter();
  /**
   * 是否隐藏上下文
   *
   * @type {boolean}
   * @memberof ContextMenuDirective
   */
  @Input('hideMenu')
  hideMenu = false;

  static overlayRef: OverlayRef | null;

  // 遮罩层点击事件
  static sub: Subscription;

  @HostListener('contextmenu', ['$event'])
  open(event: MouseEvent) {
    if (this.hideMenu) return;

    event.preventDefault();
    // 阻止冒泡
    event.stopPropagation();
    this._close();
    // 发出事件
    this.opendMenuEvent.emit();

    const { x, y } = event;

    CmContextMenuDirective.overlayRef = this.overlay.create({
      scrollStrategy: this.overlay.scrollStrategies.reposition(),
      hasBackdrop: true,
      backdropClass: 'popover-backdrop',
      panelClass: 'mat-menu-panel'
    });
    CmContextMenuDirective.overlayRef.attach(
      new TemplatePortal(this.contextMenu.lazyContent['_template'], this.viewContainerRef, {
        $implicit: this.menuData
      })
    );
    //
    // 更新位置信
    CmContextMenuDirective.overlayRef.updatePositionStrategy(this.getOverlayPosition(x, y));

    // const menuTrigger =   new MatMenuTrigger(this.overlay,this.eletemRef,this.viewContainerRef,this.overlay.scrollStrategies.close(),this.contextMenu,null,new Directionality())
    // menuTrigger.openMenu();
    // 捕获阶段
    CmContextMenuDirective.sub = fromEvent<MouseEvent>(window, 'click')
      .pipe(
        filter(event => {
          const clickTarget = event.target as HTMLElement;
          // if (!!ContextMenuDirective.overlayRef && ContextMenuDirective.overlayRef.overlayElement.contains(clickTarget)) {
          // 外部点击
          return true;
          // } else {
          //   return false;
          // }
        }),
        take(1)
      )
      .subscribe(can => can && this._close());
    // 捕获阶段
    CmContextMenuDirective.sub = CmContextMenuDirective.overlayRef.backdropClick().subscribe(() => {
      this._close();
    });
  }
  private _close() {
    if (CmContextMenuDirective.sub) {
      CmContextMenuDirective.sub.unsubscribe();
    }
    if (CmContextMenuDirective.overlayRef) {
      CmContextMenuDirective.overlayRef.dispose();
      CmContextMenuDirective.overlayRef = null;
    }
  }

  private getOverlayPosition(x: number, y: number): PositionStrategy {
    const positionStractegy = this.overlay.position().flexibleConnectedTo({ x, y }).withPositions(this.getPosition()).withFlexibleDimensions(false).withPush(false);
    return positionStractegy;
  }
  private getPosition(): ConnectionPositionPair[] {
    return [
      {
        originX: 'end',
        originY: 'bottom',
        overlayX: 'start',
        overlayY: 'top'
      },
      {
        originX: 'end',
        originY: 'top',
        overlayX: 'start',
        overlayY: 'bottom'
      }
    ];
  }
}

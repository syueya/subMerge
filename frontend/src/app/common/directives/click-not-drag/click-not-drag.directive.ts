/* eslint-disable @angular-eslint/directive-selector */
import { Directive, EventEmitter, HostListener, Output } from '@angular/core';

/**
 * 避免点击事件和拖拽冲突
 * 只有快速点击才触发点击事件，拖拽不触发点击事件
 */
@Directive({
    selector: '[click.notDrag]',
    standalone: false
})
export class ClickNotDragDirective {
  @Output('click.notDrag') readonly clickEvent = new EventEmitter<Event>();

  private pointerDownTime!: number;
  private clickThreshold = 200; // 快速点击的时间阈值（毫秒）

  @HostListener('pointerdown', ['$event'])
  onPointerDown(event: PointerEvent) {
    this.pointerDownTime = event.timeStamp;
  }

  @HostListener('pointerup', ['$event'])
  onPointerUp(event: PointerEvent) {
    const pointerUpTime = event.timeStamp;
    const timeDifference = pointerUpTime - this.pointerDownTime;

    if (timeDifference < this.clickThreshold) {
      this.clickEvent.emit(event);
    }
  }

  @HostListener('pointercancel', ['$event'])
  onPointerCancel(event: PointerEvent) {
    // 取消操作时不做任何处理
  }
}

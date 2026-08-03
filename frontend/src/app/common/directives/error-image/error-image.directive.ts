import { Directive, ElementRef, HostListener, Input, inject } from '@angular/core';

/**
 * 图片加载失败时显示默认图片
 */
@Directive({
    selector: 'img[cmErrorImage]',
    standalone: false
})
export class ErrorImageDirective {
  private el = inject(ElementRef);


  @Input() appErrorImage!: string;

  @HostListener('error')
  onError() {
    this.el.nativeElement.src = this.appErrorImage || '/assets/images/default-empty.png';
  }

}

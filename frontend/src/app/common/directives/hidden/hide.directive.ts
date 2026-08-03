/* eslint-disable @typescript-eslint/no-explicit-any */
import { Directive, Input, ElementRef, inject } from '@angular/core';

@Directive({
    selector: '[cmHide]',
    standalone: false
})
export class CmHideDirective {
  private elementRef = inject<ElementRef<any>>(ElementRef);

  private originDisplay = '';

  constructor() {
    const elementRef = this.elementRef;

    this.originDisplay = elementRef.nativeElement.style.display;
  }

  @Input() set cmHide(condition: boolean) {
    {
      if (this.elementRef.nativeElement.style) {
        if (condition) {
          this.elementRef.nativeElement.style.display = 'none';
        } else {
          this.elementRef.nativeElement.style.display = this.originDisplay;
        }
      }
    }
  }
}

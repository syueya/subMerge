/* eslint-disable @typescript-eslint/no-explicit-any */
/* eslint-disable @angular-eslint/directive-selector */
/* eslint-disable @angular-eslint/no-output-rename */
 
import { Directive, EventEmitter, Output, HostListener } from '@angular/core';

@Directive({
    selector: '[click.dbl]',
    standalone: false
})
export class DoubleClickDirective {
  @Output('click.dbl')
  readonly dblEvent = new EventEmitter<Event>();
  @Output('click.single')
  readonly signleEvent = new EventEmitter<Event>();
  // $clickSubjector: Subject<Event>;
  // 是否属于双击事件
  isDblClick = false;

  timeouId: any;

  constructor() {}

  @HostListener('click', ['$event']) onClick(e: Event) {
    this.isDblClick = false;
    this.timeouId = setTimeout(() => {
      if (!this.isDblClick) {
        this.signleEvent.emit(e);
      }
      clearTimeout(this.timeouId);
    }, 200);
  }

  @HostListener('dblclick', ['$event']) onDblClick(e: Event) {
    this.isDblClick = true;
    clearTimeout(this.timeouId);
    this.dblEvent.emit(e);
  }
}

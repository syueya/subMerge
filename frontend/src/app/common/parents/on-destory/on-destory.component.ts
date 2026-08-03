import { Component, OnDestroy } from '@angular/core';
import { Subject } from 'rxjs';

@Component({
    selector: 'cm-on-destory',
    templateUrl: './on-destory.component.html',
    standalone: false
})
export class CmOnDestoryComponent implements OnDestroy {

  $destroy: Subject<void>;

  constructor() {
    this.$destroy = new Subject();
  }

  ngOnDestroy() {
    // console.warn('销毁')
    this.$destroy.next();
    this.$destroy.complete();
  }
}


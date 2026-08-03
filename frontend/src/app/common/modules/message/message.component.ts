/**
 * Use of this source code is governed by an MIT-style license that can be
 * found in the LICENSE file at https://github.com/NG-ZORRO/ng-zorro-antd/blob/master/LICENSE
 */

import { CommonModule } from '@angular/common';
import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, OnDestroy, OnInit, Output, ViewEncapsulation, inject } from '@angular/core';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { CmStringTemplateOutletDirective } from '@common/directives/string_template_outlet.directive';

import { CmMNComponent } from './base';
import { CmMessageData } from './typings';
import { IconsModule } from '../icons/icons.module';

@Component({
  changeDetection: ChangeDetectionStrategy.OnPush,
  encapsulation: ViewEncapsulation.None,
  selector: 'cm-message',
  exportAs: 'cmMessage',
  preserveWhitespaces: false,
  template: `
    <div class="ms-message-notice" [class.ms-message-animated]="instance.options.cmAnimate" animate.enter="ms-message-enter" (mouseenter)="onEnter()" (mouseleave)="onLeave()">
      <div class="ms-message-notice-content">
        <div [class]="'ms-message-custom-content ms-message-' + instance.type">
          @switch (instance.type) {
            @case ('success') {
              <i-tabler name="circle-check" class="icon-20"></i-tabler>
            }
            @case ('info') {
              <i-tabler name="info-circle" class="icon-20"></i-tabler>
            }
            @case ('warning') {
              <i-tabler name="exclamation-circle" class="icon-20"></i-tabler>
            }
            @case ('error') {
              <i-tabler name="circle-x" class="icon-20"></i-tabler>
            }
            @case ('loading') {
              <mat-spinner diameter="20"></mat-spinner>
            }
          }

          <ng-container *cmStringTemplateOutlet="instance.content">
            <span>{{ instance.content }}</span>
          </ng-container>
        </div>
      </div>
    </div>
  `,
  imports: [IconsModule, MatProgressSpinnerModule, CmStringTemplateOutletDirective]
})
export class CmMessageComponent extends CmMNComponent implements OnInit, OnDestroy {
  @Input() instance!: Required<CmMessageData>;
  @Output() override readonly destroyed = new EventEmitter<{ id: string; userAction: boolean }>();

  constructor() {
    super();
  }
}

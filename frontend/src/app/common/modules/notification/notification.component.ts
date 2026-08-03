/**
 * Use of this source code is governed by an MIT-style license that can be
 * found in the LICENSE file at https://github.com/NG-ZORRO/ng-zorro-antd/blob/master/LICENSE
 */

import { NgTemplateOutlet } from '@angular/common';
import { ChangeDetectorRef, Component, EventEmitter, Input, OnDestroy, Output, ViewEncapsulation, inject } from '@angular/core';
import { CmStringTemplateOutletDirective } from '@common/directives/string_template_outlet.directive';

import { CmNotificationData } from './typings';
import { IconsModule } from '../icons/icons.module';
import { CmMNComponent } from '../message';

@Component({
  encapsulation: ViewEncapsulation.None,
  selector: 'cm-notification',
  exportAs: 'cmNotification',
  preserveWhitespaces: false,
  template: `
    <div
      class="ms-notification-notice ms-notification-notice-closable"
      [style]="instance.options.cmStyle || {}"
      [class]="(instance.options.cmClass || '') + ' ms-notification-enter-' + enterDirection"
      animate.enter="ms-notification-enter"
      (click)="onClick($event)"
      (mouseenter)="onEnter()"
      (mouseleave)="onLeave()"
    >
      @if (instance.template) {
        <ng-template [ngTemplateOutlet]="instance.template!" [ngTemplateOutletContext]="{ $implicit: this, data: instance.options.cmData }" />
      } @else {
        <div class="ms-notification-notice-content">
          <div class="ms-notification-notice-content">
            <div [class.ms-notification-notice-with-icon]="instance.type !== 'blank'">
              @switch (instance.type) {
                @case ('success') {
                  <i-tabler name="circle-check" class="icon-32 ms-notification-notice-icon ms-notification-notice-icon-success"></i-tabler>
                }
                @case ('info') {
                  <i-tabler name="info-circle" class="icon-32 ms-notification-notice-icon ms-notification-notice-icon-info"></i-tabler>
                }
                @case ('warning') {
                  <i-tabler name="exclamation-circle" class="icon-32 ms-notification-notice-icon ms-notification-notice-icon-warning"></i-tabler>
                }
                @case ('error') {
                  <i-tabler name="circle-x" class="icon-32 ms-notification-notice-icon ms-notification-notice-icon-error"></i-tabler>
                }
              }
              <div class="ms-notification-notice-message">
                <ng-container *cmStringTemplateOutlet="instance.title">
                  <div>{{ instance.title }}</div>
                </ng-container>
              </div>
              <div class="ms-notification-notice-description">
                <ng-container *cmStringTemplateOutlet="instance.content">
                  <div>{{ instance.content }}</div>
                </ng-container>
              </div>
              @if (instance.options.cmButton; as btn) {
                <span class="ms-notification-notice-btn">
                  <ng-template [ngTemplateOutlet]="btn" [ngTemplateOutletContext]="{ $implicit: this }" />
                </span>
              }
            </div>
          </div>
        </div>
      }
      <a tabindex="0" class="ms-notification-notice-close" (click)="close()">
        <span class="ms-notification-notice-close-x">
          @if (instance.options.cmCloseIcon) {
            <ng-container *cmStringTemplateOutlet="instance.options.cmCloseIcon; let closeIcon">
              <i-tabler name="x" class="icon-24"></i-tabler>
            </ng-container>
          } @else {
            <i-tabler name="x" class="icon-24 ms-notification-close-icon"></i-tabler>
          }
        </span>
      </a>
    </div>
  `,
  imports: [NgTemplateOutlet, IconsModule, CmStringTemplateOutletDirective]
})
export class CmNotificationComponent extends CmMNComponent implements OnDestroy {
  @Input() instance!: Required<CmNotificationData>;
  @Input() index!: number; // todo: does this used?
  @Input() placement?: string;

  @Output() override readonly destroyed = new EventEmitter<{ id: string; userAction: boolean }>();

  constructor() {
    super();
  }

  override ngOnDestroy(): void {
    super.ngOnDestroy();
    this.instance.onClick.complete();
  }

  onClick(event: MouseEvent): void {
    this.instance.onClick.next(event);
  }

  close(): void {
    this.destroy(true);
  }

  get enterDirection(): string {
    if (this.instance.state === 'enter') {
      switch (this.placement) {
        case 'topLeft':
        case 'bottomLeft':
          return 'left';
        case 'top':
          return 'top';
        case 'bottom':
          return 'bottom';
        default:
          return 'right';
      }
    }
    return 'right';
  }
}

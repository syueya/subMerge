/**
 * Use of this source code is governed by an MIT-style license that can be
 * found in the LICENSE file at https://github.com/NG-ZORRO/ng-zorro-antd/blob/master/LICENSE
 */

import { ChangeDetectionStrategy, ChangeDetectorRef, Component, ViewEncapsulation, inject } from '@angular/core';
import { NotificationConfig } from '@common/interfaces/NotificationConfig';
import { toCssPixel } from '@common/util/convertUtil';
import { CM_NOTIFICATION_DEFAULT_CONFIG } from '@common/util/defines';
import { Subject } from 'rxjs';


import { CmNotificationComponent } from './notification.component';
import { CmNotificationData, CmNotificationDataOptions, CmNotificationPlacement } from './typings';
import { CmMNContainerComponent } from '../message';

const CM_CONFIG_MODULE_NAME = 'notification';


@Component({
    changeDetection: ChangeDetectionStrategy.OnPush,
    encapsulation: ViewEncapsulation.None,
    selector: 'cm-notification-container',
    exportAs: 'cmNotificationContainer',
    preserveWhitespaces: false,
    template: `
    <div
      class="ms-notification ms-notification-topLeft"
      [style.top]="top"
      [style.left]="'0px'"
    >
      @for (instance of topLeftInstances; track instance) {
        <cm-notification
          [instance]="instance"
          [placement]="'topLeft'"
          (destroyed)="remove($event.id, $event.userAction)"
        />
      }
    </div>
    <div
      class="ms-notification ms-notification-topRight"
      [style.top]="top"
      [style.right]="'0px'"
    >
      @for (instance of topRightInstances; track instance) {
        <cm-notification
          [instance]="instance"
          [placement]="'topRight'"
          (destroyed)="remove($event.id, $event.userAction)"
        />
      }
    </div>
    <div
      class="ms-notification ms-notification-bottomLeft"
      [style.bottom]="bottom"
      [style.left]="'0px'"
    >
      @for (instance of bottomLeftInstances; track instance) {
        <cm-notification
          [instance]="instance"
          [placement]="'bottomLeft'"
          (destroyed)="remove($event.id, $event.userAction)"
        />
      }
    </div>
    <div
      class="ms-notification ms-notification-bottomRight"
      [style.bottom]="bottom"
      [style.right]="'0px'"
    >
      @for (instance of bottomRightInstances; track instance) {
        <cm-notification
          [instance]="instance"
          [placement]="'bottomRight'"
          (destroyed)="remove($event.id, $event.userAction)"
        />
      }
    </div>
    <div
      class="ms-notification ms-notification-top"
      [style.top]="top"
      [style.left]="'50%'"
      [style.transform]="'translateX(-50%)'"
    >
      @for (instance of topInstances; track instance) {
        <cm-notification [instance]="instance" [placement]="'top'" (destroyed)="remove($event.id, $event.userAction)" />
      }
    </div>
    <div
      class="ms-notification ms-notification-bottom"
      [style.bottom]="bottom"
      [style.left]="'50%'"
      [style.transform]="'translateX(-50%)'"
    >
      @for (instance of bottomInstances; track instance) {
        <cm-notification
          [instance]="instance"
          [placement]="'bottom'"
          (destroyed)="remove($event.id, $event.userAction)"
        />
      }
    </div>
  `,
    imports: [CmNotificationComponent]
})
export class CmNotificationContainerComponent extends CmMNContainerComponent<NotificationConfig, CmNotificationData> {
  bottom?: string | null;
  top?: string | null;

  topLeftInstances: Array<Required<CmNotificationData>> = [];
  topRightInstances: Array<Required<CmNotificationData>> = [];
  bottomLeftInstances: Array<Required<CmNotificationData>> = [];
  bottomRightInstances: Array<Required<CmNotificationData>> = [];
  topInstances: Array<Required<CmNotificationData>> = [];
  bottomInstances: Array<Required<CmNotificationData>> = [];

  constructor() {
    super();
  }

  override create(notification: CmNotificationData): Required<CmNotificationData> {
    const noti = this.onCreate(notification);
    const key = noti.options.cmKey;
    const notificationWithSameKey = this.instances.find(
      msg => msg.options.cmKey === (notification.options as Required<CmNotificationDataOptions>).cmKey
    );
    if (key && notificationWithSameKey) {
      this.replaceNotification(notificationWithSameKey, noti);
    } else {
      if (this.instances.length >= this.config!.cmMaxStack) {
        this.instances = this.instances.slice(1);
      }
      this.instances = [...this.instances, noti];
    }

    this.readyInstances();

    return noti;
  }

  protected override onCreate(instance: CmNotificationData): Required<CmNotificationData> {
    instance.options = this.mergeOptions(instance.options);
    instance.onClose = new Subject<boolean>();
    instance.onClick = new Subject<MouseEvent>();
    return instance as Required<CmNotificationData>;
  }


  protected updateConfig(): void {
    this.config = CM_NOTIFICATION_DEFAULT_CONFIG;

    this.top = toCssPixel(this.config.cmTop!);
    this.bottom = toCssPixel(this.config.cmBottom!);

    this.cdr.markForCheck();
  }

  private replaceNotification(old: CmNotificationData, _new: CmNotificationData): void {
    old.title = _new.title;
    old.content = _new.content;
    old.template = _new.template;
    old.type = _new.type;
    old.options = _new.options;
  }

  protected override readyInstances(): void {
    const instancesMap: Record<CmNotificationPlacement, Array<Required<CmNotificationData>>> = {
      topLeft: [],
      topRight: [],
      bottomLeft: [],
      bottomRight: [],
      top: [],
      bottom: []
    };
    this.instances.forEach(m => {
      const placement = m.options.cmPlacement;
      switch (placement) {
        case 'topLeft':
          instancesMap.topLeft.unshift(m);
          break;
        case 'topRight':
          instancesMap.topRight.unshift(m);
          break;
        case 'bottomLeft':
          instancesMap.bottomLeft.unshift(m);
          break;
        case 'bottomRight':
          instancesMap.bottomRight.unshift(m);
          break;
        case 'top':
          instancesMap.top.unshift(m);
          break;
        case 'bottom':
          instancesMap.bottom.unshift(m);
          break;
        default:
          instancesMap.topRight.unshift(m);
      }
    });
    this.topLeftInstances = instancesMap.topLeft;
    this.topRightInstances = instancesMap.topRight;
    this.bottomLeftInstances = instancesMap.bottomLeft;
    this.bottomRightInstances = instancesMap.bottomRight;
    this.topInstances = instancesMap.top;
    this.bottomInstances = instancesMap.bottom;

    this.cdr.detectChanges();
  }

  protected override mergeOptions(options?: CmNotificationDataOptions): CmNotificationDataOptions {
    const { cmDuration, cmAnimate, cmPauseOnHover, cmPlacement } = this.config ?? {};
    return { cmDuration, cmAnimate, cmPauseOnHover, cmPlacement, ...options };
  }
}

/* eslint-disable @typescript-eslint/no-empty-object-type */
/**
 * Use of this source code is governed by an MIT-style license that can be
 * found in the LICENSE file at https://github.com/NG-ZORRO/ng-zorro-antd/blob/master/LICENSE
 */

import { Overlay } from '@angular/cdk/overlay';
import { Injectable, Injector, TemplateRef, inject } from '@angular/core';
import { SingletonService } from '@common/services/singleton.service';

import { CmNotificationContainerComponent } from './notification-container.component';
import { CmNotificationData, CmNotificationDataOptions, CmNotificationRef } from './typings';
import { CmMNService } from '../message';

let notificationId = 0;

@Injectable({
  providedIn: 'root'
})
export class CmNotificationService extends CmMNService<CmNotificationContainerComponent> {
  protected componentPrefix = 'notification-';

  constructor() {
    const singletonService = inject(SingletonService);
    const overlay = inject(Overlay);
    const injector = inject(Injector);

    super(singletonService, overlay, injector);
  }

  success(
    title: string | TemplateRef<void>,
    content: string | TemplateRef<void>,
    options?: CmNotificationDataOptions
  ): CmNotificationRef {
    return this.create('success', title, content, options);
  }

  error(
    title: string | TemplateRef<void>,
    content: string | TemplateRef<void>,
    options?: CmNotificationDataOptions
  ): CmNotificationRef {
    return this.create('error', title, content, options);
  }

  info(
    title: string | TemplateRef<void>,
    content: string | TemplateRef<void>,
    options?: CmNotificationDataOptions
  ): CmNotificationRef {
    return this.create('info', title, content, options);
  }

  warning(
    title: string | TemplateRef<void>,
    content: string | TemplateRef<void>,
    options?: CmNotificationDataOptions
  ): CmNotificationRef {
    return this.create('warning', title, content, options);
  }

  blank(
    title: string | TemplateRef<void>,
    content: string | TemplateRef<void>,
    options?: CmNotificationDataOptions
  ): CmNotificationRef {
    return this.create('blank', title, content, options);
  }

  create(
    type: 'success' | 'info' | 'warning' | 'error' | 'blank' | string,
    title: string | TemplateRef<void>,
    content: string | TemplateRef<void>,
    options?: CmNotificationDataOptions
  ): CmNotificationRef {
    return this.createInstance({ type, title, content }, options);
  }

  template(template: TemplateRef<{}>, options?: CmNotificationDataOptions): CmNotificationRef {
    return this.createInstance({ template }, options);
  }

  protected generateMessageId(): string {
    return `${this.componentPrefix}-${notificationId++}`;
  }

  private createInstance(message: CmNotificationData, options?: CmNotificationDataOptions): CmNotificationRef {
    this.container = this.withContainer(CmNotificationContainerComponent);

    return this.container.create({
      ...message,
      ...{
        createdAt: new Date(),
        messageId: options?.cmKey || this.generateMessageId(),
        options
      }
    });
  }
}

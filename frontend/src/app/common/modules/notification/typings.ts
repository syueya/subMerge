/* eslint-disable @typescript-eslint/no-empty-object-type */
/**
 * Use of this source code is governed by an MIT-style license that can be
 * found in the LICENSE file at https://github.com/NG-ZORRO/ng-zorro-antd/blob/master/LICENSE
 */

import { TemplateRef } from '@angular/core';
import { Subject } from 'rxjs';


import type { CmNotificationComponent } from './notification.component';

export type CmNotificationPlacement = 'topLeft' | 'topRight' | 'bottomLeft' | 'bottomRight' | 'top' | 'bottom';

export interface CmNotificationDataOptions<T = {}> {
  cmKey?: string;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  cmStyle?: Record<string, any>;
  cmClass?: string;
  cmCloseIcon?: TemplateRef<void> | string;
  cmButton?: TemplateRef<{ $implicit: CmNotificationComponent }>;
  cmPlacement?: CmNotificationPlacement;
  cmData?: T;
  cmDuration?: number;
  cmAnimate?: boolean;
  cmPauseOnHover?: boolean;
}

export interface CmNotificationData {
  title?: string | TemplateRef<void>;
  content?: string | TemplateRef<void>;
  createdAt?: Date;
  messageId?: string;
  options?: CmNotificationDataOptions;
  state?: 'enter' | 'leave';
  template?: TemplateRef<{}>;
  type?: 'success' | 'info' | 'warning' | 'error' | 'blank' | string;

  // observables exposed to users
  onClose?: Subject<boolean>;
  onClick?: Subject<MouseEvent>;
}

export type CmNotificationRef = Pick<Required<CmNotificationData>, 'onClose' | 'onClick' | 'messageId'>;

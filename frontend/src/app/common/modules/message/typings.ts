/**
 * Use of this source code is governed by an MIT-style license that can be
 * found in the LICENSE file at https://github.com/NG-ZORRO/ng-zorro-antd/blob/master/LICENSE
 */

import { TemplateRef } from '@angular/core';
import { Subject } from 'rxjs';

export type CmMessageType = 'success' | 'info' | 'warning' | 'error' | 'loading';

export interface CmMessageDataOptions {
  cmDuration?: number;
  cmAnimate?: boolean;
  cmPauseOnHover?: boolean;
}

export interface CmMessageData {
  type?: CmMessageType | string;
  content?: string | TemplateRef<void>;
  messageId?: string;
  createdAt?: Date;
  options?: CmMessageDataOptions;
  state?: 'enter' | 'leave';

  onClose?: Subject<boolean>;
}

export type CmMessageRef = Pick<Required<CmMessageData>, 'onClose' | 'messageId'>;

/**
 * Use of this source code is governed by an MIT-style license that can be
 * found in the LICENSE file at https://github.com/NG-ZORRO/ng-zorro-antd/blob/master/LICENSE
 */

import { Overlay } from '@angular/cdk/overlay';
import { Injectable, Injector, TemplateRef, inject } from '@angular/core';
import { SingletonService } from '@common/services/singleton.service';

import { CmMNService } from './base';
import { CmMessageContainerComponent } from './message-container.component';
import { CmMessageData, CmMessageDataOptions, CmMessageRef } from './typings';

@Injectable({
  providedIn: 'root'
})
export class CmMessageService extends CmMNService<CmMessageContainerComponent> {
  protected componentPrefix = 'message-';

  constructor() {
    const singletonService = inject(SingletonService);
    const overlay = inject(Overlay);
    const injector = inject(Injector);

    super(singletonService, overlay, injector);
  }

  success(content: string | TemplateRef<void>, options?: CmMessageDataOptions): CmMessageRef {
    return this.createInstance({ type: 'success', content }, options);
  }

  error(content: string | TemplateRef<void>, options?: CmMessageDataOptions): CmMessageRef {
    return this.createInstance({ type: 'error', content }, options);
  }

  info(content: string | TemplateRef<void>, options?: CmMessageDataOptions): CmMessageRef {
    return this.createInstance({ type: 'info', content }, options);
  }

  warning(content: string | TemplateRef<void>, options?: CmMessageDataOptions): CmMessageRef {
    return this.createInstance({ type: 'warning', content }, options);
  }

  loading(content: string | TemplateRef<void>, options?: CmMessageDataOptions): CmMessageRef {
    return this.createInstance({ type: 'loading', content }, options);
  }

  create(
    type: 'success' | 'info' | 'warning' | 'error' | 'loading' | string,
    content: string | TemplateRef<void>,
    options?: CmMessageDataOptions
  ): CmMessageRef {
    return this.createInstance({ type, content }, options);
  }

  private createInstance(message: CmMessageData, options?: CmMessageDataOptions): CmMessageRef {
    this.container = this.withContainer(CmMessageContainerComponent);

    return this.container.create({
      ...message,
      ...{
        createdAt: new Date(),
        messageId: this.getInstanceId(),
        options
      }
    });
  }
}

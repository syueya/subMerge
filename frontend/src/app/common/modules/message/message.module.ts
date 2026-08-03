/**
 * Use of this source code is governed by an MIT-style license that can be
 * found in the LICENSE file at https://github.com/NG-ZORRO/ng-zorro-antd/blob/master/LICENSE
 */

import { NgModule } from '@angular/core';

import { CmMessageContainerComponent } from './message-container.component';
import { CmMessageComponent } from './message.component';

@NgModule({
  imports: [CmMessageContainerComponent, CmMessageComponent]
})
export class CmMessageModule {}

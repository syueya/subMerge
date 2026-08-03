/**
 * Use of this source code is governed by an MIT-style license that can be
 * found in the LICENSE file at https://github.com/NG-ZORRO/ng-zorro-antd/blob/master/LICENSE
 */

import { NgModule } from '@angular/core';

import { CmNotificationContainerComponent } from './notification-container.component';
import { CmNotificationComponent } from './notification.component';

@NgModule({
  imports: [CmNotificationComponent, CmNotificationContainerComponent]
})
export class CmNotificationModule {}

import { NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { AppCommonModule } from '@common/common.module';

import { NetCheckPageComponent } from './net-check-page/net-check-page.component';
import { NetCheckRoutingModule } from './net-check-routing.module';
import { NetCheckTargetManageComponent } from './net-check-target-manage/net-check-target-manage.component';

@NgModule({
	declarations: [NetCheckPageComponent, NetCheckTargetManageComponent],
	imports: [AppCommonModule, RouterModule, NetCheckRoutingModule],
})
export class NetCheckModule {}

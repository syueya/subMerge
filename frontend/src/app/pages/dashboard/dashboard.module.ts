import { NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { AppCommonModule } from '@common/common.module';
import { SharedBusinessModule } from '../_shared/shared-business.module';

import { DashboardHomeComponent } from './dashboard-home/dashboard-home.component';
import { DashboardRoutingModule } from './dashboard-routing.module';

@NgModule({
  declarations: [DashboardHomeComponent],
  imports: [
    AppCommonModule,
    SharedBusinessModule,
    RouterModule,
    DashboardRoutingModule
  ]
})
export class DashboardModule { }

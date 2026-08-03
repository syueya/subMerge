import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { AppCommonModule } from '@common/common.module';

import { SettingUserComponent } from './setting-user/setting-user.component';
import { SettingUserLoginLogsComponent } from './setting-user-login-logs/setting-user-login-logs.component';
import { SettingUserRoutingModule } from './setting-user-routing.module';
import { SettingUserUpdateComponent } from './setting-user-update/setting-user-update.component';


@NgModule({
  declarations: [
    SettingUserComponent,
    SettingUserUpdateComponent,
    SettingUserLoginLogsComponent
  ],
  imports: [
    CommonModule,
    AppCommonModule,
    FormsModule,
    SettingUserRoutingModule
  ]
})
export class SettingUserModule { }

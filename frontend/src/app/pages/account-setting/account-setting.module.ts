import { NgModule } from '@angular/core';
import { MatTabsModule } from '@angular/material/tabs';
import { AppCommonModule } from '@common/common.module';

import { AccountSettingIndexComponent } from './account-setting-index/account-setting-index.component';
import { AccountSettingInfoComponent } from './account-setting-info/account-setting-info.component';
import { AccountSettingInfoAvatarSettingComponent } from './account-setting-info-avatar-setting/account-setting-info-avatar-setting.component';
import { AccountSettingInfoBaseInfoComponent } from './account-setting-info-base-info/account-setting-info-base-info.component';
import { AccountSettingInfoPasswordChangeComponent } from './account-setting-info-password-change/account-setting-info-password-change.component';
import { AccountSettingRoutingModule } from './account-setting-routing.module';
import { AccountSettingSecurityComponent } from './account-setting-security/account-setting-security.component';
import { AccountSettingSecurityDeviceManageComponent } from './account-setting-security-device-manage/account-setting-security-device-manage.component';

@NgModule({
  declarations: [
    AccountSettingIndexComponent,
    AccountSettingInfoComponent,
    AccountSettingInfoBaseInfoComponent,
    AccountSettingInfoAvatarSettingComponent,
    AccountSettingInfoPasswordChangeComponent,
    AccountSettingSecurityComponent,
    AccountSettingSecurityDeviceManageComponent
  ],
  imports: [
    AppCommonModule,
    MatTabsModule,
    AccountSettingRoutingModule,
  ]
})
export class AccountSettingModule {}

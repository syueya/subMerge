import { NgModule } from '@angular/core';
import { MatTabsModule } from '@angular/material/tabs';
import { AppCommonModule } from '@common/common.module';

import { AccountSettingApikeyFormComponent } from './account-setting-apikey-form/account-setting-apikey-form.component';
import { AccountSettingApikeySecretComponent } from './account-setting-apikey-secret/account-setting-apikey-secret.component';
import { AccountSettingApikeysComponent } from './account-setting-apikeys/account-setting-apikeys.component';
import { AccountSettingIndexComponent } from './account-setting-index/account-setting-index.component';
import { AccountSettingInfoComponent } from './account-setting-info/account-setting-info.component';
import { AccountSettingInfoAvatarSettingComponent } from './account-setting-info-avatar-setting/account-setting-info-avatar-setting.component';
import { AccountSettingInfoBaseInfoComponent } from './account-setting-info-base-info/account-setting-info-base-info.component';
import { AccountSettingInfoPasswordChangeComponent } from './account-setting-info-password-change/account-setting-info-password-change.component';
import { AccountSettingRoutingModule } from './account-setting-routing.module';

@NgModule({
  declarations: [
    AccountSettingIndexComponent,
    AccountSettingInfoComponent,
    AccountSettingInfoBaseInfoComponent,
    AccountSettingInfoAvatarSettingComponent,
    AccountSettingInfoPasswordChangeComponent,
    AccountSettingApikeysComponent,
    AccountSettingApikeyFormComponent,
    AccountSettingApikeySecretComponent,
  ],
  imports: [
    AppCommonModule,
    MatTabsModule,
    AccountSettingRoutingModule,
  ]
})
export class AccountSettingModule {}

import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { AppCommonModule } from '@common/common.module';

import { SettingSettingsComponent } from './setting-settings/setting-settings.component';
import { SettingSettingsRoutingModule } from './setting-settings-routing.module';

@NgModule({
  declarations: [SettingSettingsComponent],
  imports: [CommonModule, AppCommonModule, SettingSettingsRoutingModule],
})
export class SettingSettingsModule {}
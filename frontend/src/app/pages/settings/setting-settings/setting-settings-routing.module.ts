import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { SettingSettingsComponent } from './setting-settings/setting-settings.component';

const routes: Routes = [
  {
    path: '',
    component: SettingSettingsComponent,
    data: {
      hideBreadcrumb: true,
    },
  },
];

@NgModule({
  imports: [RouterModule.forChild(routes)],
  exports: [RouterModule],
})
export class SettingSettingsRoutingModule {}
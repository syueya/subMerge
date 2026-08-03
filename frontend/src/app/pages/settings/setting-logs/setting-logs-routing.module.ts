import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';

import { SettingLogsComponent } from './setting-logs/setting-logs.component';
const routes: Routes = [
  { path: '', component: SettingLogsComponent,
    data: {
      hideBreadcrumb: true //隐藏面包屑
    }
  }
];

@NgModule({
  imports: [RouterModule.forChild(routes)],
  exports: [RouterModule]
})
export class SettingLogsRoutingModule { }

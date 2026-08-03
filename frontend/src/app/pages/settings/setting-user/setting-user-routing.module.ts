import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';

import { SettingUserComponent } from './setting-user/setting-user.component';
const routes: Routes = [
  { path: '', component: SettingUserComponent,
    data: {
      hideBreadcrumb: true //隐藏面包屑
    }
  }
];

@NgModule({
  imports: [RouterModule.forChild(routes)],
  exports: [RouterModule]
})
export class SettingUserRoutingModule { }

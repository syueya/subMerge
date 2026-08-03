import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';

import { AccountSettingIndexComponent } from './account-setting-index/account-setting-index.component';

const routes: Routes = [
  {
    path: '',
    component: AccountSettingIndexComponent,
    data: {
      title: '我的信息',
      hideBreadcrumb: true,
      urls: [
        { title: '首页', url: '/main/dashboard' },
        { title: '我的信息' },
      ],
    },
  },
];

@NgModule({
  imports: [RouterModule.forChild(routes)],
  exports: [RouterModule]
})
export class AccountSettingRoutingModule { }

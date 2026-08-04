import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';

const routes: Routes = [
  {
    path: '',
    children: [
      {
        path: '',
        redirectTo: 'logs',
        pathMatch: 'full'
      },
      // 用户管理页为遗留脚手架（/api/v1），暂不挂载
      {
        path: 'logs',
        loadChildren: () => import('./setting-logs/setting-logs.module').then((m) => m.SettingLogsModule)
      }
    ]
  }
];

@NgModule({
  imports: [RouterModule.forChild(routes)],
  exports: [RouterModule]
})
export class SettingsRoutingModule {}

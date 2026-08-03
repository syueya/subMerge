import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { adminGuard } from '@common/guard/AdminGuard';

const routes: Routes = [
  {
    path: '',
    children: [
      {
        path: '',
        redirectTo: 'user',
        pathMatch: 'full'
      },
      { path: 'user', canActivate: [adminGuard], loadChildren: () => import('./setting-user/setting-user.module').then((m) => m.SettingUserModule) },
      { path: 'logs', canActivate: [adminGuard], loadChildren: () => import('./setting-logs/setting-logs.module').then((m) => m.SettingLogsModule) }
    ]
  }
];

@NgModule({
  imports: [RouterModule.forChild(routes)],
  exports: [RouterModule]
})
export class SettingsRoutingModule { }

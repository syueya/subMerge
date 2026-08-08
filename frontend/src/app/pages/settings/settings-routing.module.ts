import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';

const routes: Routes = [
  {
    path: '',
    children: [
      {
        path: '',
        redirectTo: 'setting-settings',
        pathMatch: 'full'
      },
      {
        path: 'setting-settings',
        loadChildren: () => import('./setting-settings/setting-settings.module').then((m) => m.SettingSettingsModule)
      },
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

import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';

const routes: Routes = [
  {
    path: '',
    children: [
      {
        path: '',
        redirectTo: 'outbound-proxy',
        pathMatch: 'full'
      },
      {
        path: 'outbound-proxy',
        loadChildren: () => import('./outbound-proxy/outbound-proxy.module').then((m) => m.OutboundProxyModule)
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

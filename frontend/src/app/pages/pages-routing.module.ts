import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';

const routes: Routes = [
  {
    path: '',
    redirectTo: 'dashboard',
    pathMatch: 'full'
  },
  {
    path: 'dashboard',
    loadChildren: () => import('./dashboard/dashboard.module').then((m) => m.DashboardModule),
    data: {
      hideBreadcrumb: true
    }
  },
  {
    path: 'sources',
    loadChildren: () => import('./sources/sources.module').then((m) => m.SourcesModule)
  },
  {
    path: 'groups',
    loadChildren: () => import('./groups/groups.module').then((m) => m.GroupsModule)
  },
  {
    path: 'rules',
    loadChildren: () => import('./rules/rules.module').then((m) => m.RulesModule)
  },
  {
    path: 'releases',
    loadChildren: () => import('./releases/releases.module').then((m) => m.ReleasesModule)
  },
  {
    path: 'tokens',
    loadChildren: () => import('./tokens/tokens.module').then((m) => m.TokensModule)
  },
  {
    path: 'geo',
    loadChildren: () => import('./geo/geo.module').then((m) => m.GeoModule)
  },
  {
    path: 'net-check',
    loadChildren: () => import('./net-check/net-check.module').then((m) => m.NetCheckModule)
  },
  {
    path: 'account-setting',
    loadChildren: () => import('./account-setting/account-setting.module').then(m => m.AccountSettingModule)
  },
  {
    path: 'settings',
    loadChildren: () => import('./settings/settings.module').then(m => m.SettingsModule)
  },
  // 快捷入口：/main/logs → 系统日志
  {
    path: 'logs',
    loadChildren: () =>
      import('./settings/setting-logs/setting-logs.module').then(m => m.SettingLogsModule)
  }
];

@NgModule({
  imports: [RouterModule.forChild(routes)],
  exports: [RouterModule]
})
export class PagesRoutingModule {}

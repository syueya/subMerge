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
    path: 'account-setting',
    loadChildren: () => import('./account-setting/account-setting.module').then(m => m.AccountSettingModule)
  }
];

@NgModule({
  imports: [RouterModule.forChild(routes)],
  exports: [RouterModule]
})
export class PagesRoutingModule {}

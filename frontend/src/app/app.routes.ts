import { Routes } from '@angular/router';

import { authGuard } from './common/guard/AuthGuard';
import { AppErrorComponent } from './common/error/error.component';
import { BlankComponent } from './layouts/blank/blank.component';
import { FullComponent } from './layouts/full/full.component';

export const routes: Routes = [
  {
    path: '',
    component: FullComponent,
    canActivate: [authGuard],
    children: [
      {
        path: '',
        redirectTo: 'main',
        pathMatch: 'full'
      },
      {
        path: 'main',
        loadChildren: () => import('./pages/pages.module').then((m) => m.PagesModule)
      }
    ]
  },
  {
    path: '',
    component: BlankComponent,
    children: [
      {
        path: 'auth',
        loadChildren: () => import('./pages/auth/auth.module').then((m) => m.AuthModule)
      }
    ]
  },
  {
    path: '',
    component: BlankComponent,
    children: [
    ]
  },
  /* 错误 */
  {
    path: 'error',
    component: AppErrorComponent
  },
  {
    path: '**',
    redirectTo: 'error'
  }
];

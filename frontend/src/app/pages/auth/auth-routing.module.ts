import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';

import { LoginComponent } from './login/login.component';
import { SystemInitConfigComponent } from './system-init-config/system-init-config.component';

const routes: Routes = [
  {
    path: '',
    redirectTo: 'login',
    pathMatch: 'full',
  },
  /* 登录 */
  {
    path: 'login',
    component: LoginComponent
  },
  /* 系统初始化配置 */
  {
    path: 'system-init-config',
    component: SystemInitConfigComponent
  }
];

@NgModule({
  imports: [RouterModule.forChild(routes)],
  exports: [RouterModule]
})
export class AuthRoutingModule { }

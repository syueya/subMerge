import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { AppCommonModule } from '@common/common.module';

import { AuthRoutingModule } from './auth-routing.module';
import { LoginComponent } from './login/login.component';
import { SystemInitConfigComponent } from './system-init-config/system-init-config.component';


@NgModule({
  declarations: [
    LoginComponent,
    SystemInitConfigComponent
  ],
  imports: [
    CommonModule,
    AppCommonModule,
    AuthRoutingModule
  ]
})
export class AuthModule { }

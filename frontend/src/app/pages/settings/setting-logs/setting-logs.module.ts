import { CdkAccordionModule } from '@angular/cdk/accordion';
import { ScrollingModule } from '@angular/cdk/scrolling';
import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { AppCommonModule } from '@common/common.module';
import { NgScrollbarModule } from 'ngx-scrollbar';

import { SettingLogsComponent } from './setting-logs/setting-logs.component';
import { SettingLogsRoutingModule } from './setting-logs-routing.module';


@NgModule({
  declarations: [
    SettingLogsComponent
  ],
  imports: [
    CommonModule,
    AppCommonModule,
    NgScrollbarModule,
    CdkAccordionModule,
    ScrollingModule,
    SettingLogsRoutingModule
  ]
})
export class SettingLogsModule { }

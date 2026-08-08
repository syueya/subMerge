import { NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { AppCommonModule } from '@common/common.module';

import { SourceFormComponent } from './source-form/source-form.component';
import { SourceListComponent } from './source-list/source-list.component';
import { SourceProxiesComponent } from './source-proxies/source-proxies.component';
import { SourcesRoutingModule } from './sources-routing.module';

@NgModule({
  declarations: [SourceFormComponent, SourceListComponent, SourceProxiesComponent],
  imports: [AppCommonModule, RouterModule, SourcesRoutingModule]
})
export class SourcesModule {}

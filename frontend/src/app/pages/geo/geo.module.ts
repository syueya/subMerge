import { NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { AppCommonModule } from '@common/common.module';
import { GeoEntriesComponent } from './geo-entries/geo-entries.component';
import { GeoQueryComponent } from './geo-query/geo-query.component';
import { GeoRoutingModule } from './geo-routing.module';

@NgModule({
  declarations: [GeoQueryComponent, GeoEntriesComponent],
  imports: [AppCommonModule, RouterModule, GeoRoutingModule]
})
export class GeoModule {}

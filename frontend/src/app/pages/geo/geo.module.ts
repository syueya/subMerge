import { NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { RuleFormModule } from '../rules/rule-form/rule-form.module';
import { AppCommonModule } from '@common/common.module';
import { GeoEntriesComponent } from './geo-entries/geo-entries.component';
import { GeoCategorySearchComponent } from './geo-query/geo-category-search/geo-category-search.component';
import { GeoDomainQueryComponent } from './geo-query/geo-domain-query/geo-domain-query.component';
import { GeoQueryComponent } from './geo-query/geo-query/geo-query.component';
import { GeoRoutingModule } from './geo-routing.module';

@NgModule({
  declarations: [GeoQueryComponent, GeoDomainQueryComponent, GeoCategorySearchComponent, GeoEntriesComponent],
  imports: [AppCommonModule, RuleFormModule, RouterModule, GeoRoutingModule]
})
export class GeoModule {}

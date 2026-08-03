import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { GeoQueryComponent } from './geo-query/geo-query.component';

const routes: Routes = [
  {
    path: '',
    component: GeoQueryComponent,
    data: { title: 'Geo 数据' }
  }
];

@NgModule({
  imports: [RouterModule.forChild(routes)],
  exports: [RouterModule]
})
export class GeoRoutingModule {}

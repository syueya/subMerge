import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';

import { SourceListComponent } from './source-list/source-list.component';

const routes: Routes = [
  {
    path: '',
    component: SourceListComponent,
    data: {
      title: '订阅源',
      hideBreadcrumb: true
    }
  }
];

@NgModule({
  imports: [RouterModule.forChild(routes)],
  exports: [RouterModule]
})
export class SourcesRoutingModule {}

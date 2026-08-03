import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { ReleaseListComponent } from './release-list/release-list.component';

const routes: Routes = [
  {
    path: '',
    component: ReleaseListComponent,
    data: {
      title: '发布',
      hideBreadcrumb: true
    }
  }
];

@NgModule({
  imports: [RouterModule.forChild(routes)],
  exports: [RouterModule]
})
export class ReleasesRoutingModule {}

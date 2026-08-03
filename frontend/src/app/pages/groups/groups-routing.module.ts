import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { GroupListComponent } from './group-list/group-list.component';

const routes: Routes = [
  {
    path: '',
    component: GroupListComponent,
    data: {
      title: '策略组',
      hideBreadcrumb: true
    }
  }
];

@NgModule({
  imports: [RouterModule.forChild(routes)],
  exports: [RouterModule]
})
export class GroupsRoutingModule {}

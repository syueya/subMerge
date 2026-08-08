import { NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { AppCommonModule } from '@common/common.module';

import { GroupFormComponent } from './group-form/group-form.component';
import { GroupListComponent } from './group-list/group-list.component';
import { GroupsRoutingModule } from './groups-routing.module';

@NgModule({
  declarations: [GroupListComponent, GroupFormComponent],
  imports: [AppCommonModule, RouterModule, GroupsRoutingModule]
})
export class GroupsModule {}

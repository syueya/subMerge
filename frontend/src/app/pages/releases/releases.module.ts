import { NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { AppCommonModule } from '@common/common.module';
import { SharedBusinessModule } from '../_shared/shared-business.module';
import { DraftChangesComponent } from './draft-changes/draft-changes.component';
import { ReleaseDetailComponent } from './release-detail/release-detail.component';
import { ReleaseListComponent } from './release-list/release-list.component';
import { ReleasesRoutingModule } from './releases-routing.module';

@NgModule({
  declarations: [ReleaseListComponent, ReleaseDetailComponent, DraftChangesComponent],
  imports: [AppCommonModule, SharedBusinessModule, RouterModule, ReleasesRoutingModule]
})
export class ReleasesModule {}

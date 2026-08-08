import { NgModule } from '@angular/core';
import { AppCommonModule } from '@common/common.module';

import { RuleMatchDialogComponent } from './rule-match-dialog/rule-match-dialog.component';

@NgModule({
  declarations: [RuleMatchDialogComponent],
  imports: [AppCommonModule],
  exports: [RuleMatchDialogComponent, AppCommonModule]
})
export class SharedBusinessModule {}

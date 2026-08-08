import { NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { AppCommonModule } from '@common/common.module';
import { RuleFormComponent } from './rule-form.component';

@NgModule({
  declarations: [RuleFormComponent],
  imports: [AppCommonModule, RouterModule],
  exports: [RuleFormComponent]
})
export class RuleFormModule {}

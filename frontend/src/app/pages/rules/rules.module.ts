import { NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { AppCommonModule } from '@common/common.module';
import { SharedBusinessModule } from '../_shared/shared-business.module';
import { BatchImportComponent } from './batch-import/batch-import.component';
import { NewCategoryFormComponent } from './new-category-form/new-category-form.component';
import { RuleHomeComponent } from './rule-home/rule-home.component';
import { RuleFormModule } from './rule-form/rule-form.module';
import { RulesRoutingModule } from './rules-routing.module';

@NgModule({
  declarations: [RuleHomeComponent, BatchImportComponent, NewCategoryFormComponent],
  imports: [AppCommonModule, SharedBusinessModule, RuleFormModule, RouterModule, RulesRoutingModule]
})
export class RulesModule {}

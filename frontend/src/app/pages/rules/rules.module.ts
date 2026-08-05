import { NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { AppCommonModule } from '@common/common.module';
import { SharedBusinessModule } from '../_shared/shared-business.module';
import { BatchImportComponent } from './batch-import/batch-import.component';
import { NewCategoryFormComponent } from './new-category-form/new-category-form.component';
import { RuleHomeComponent } from './rule-home/rule-home.component';
import { RuleFormComponent } from './rule-form/rule-form.component';
import { RulesRoutingModule } from './rules-routing.module';

@NgModule({
  declarations: [
    RuleHomeComponent,
    RuleFormComponent,
    BatchImportComponent,
    NewCategoryFormComponent
  ],
  imports: [AppCommonModule, SharedBusinessModule, RouterModule, RulesRoutingModule]
})
export class RulesModule {}

import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { RuleHomeComponent } from './rule-home/rule-home.component';

const routes: Routes = [
  {
    path: '',
    component: RuleHomeComponent,
    data: {
      title: '分流规则',
      hideBreadcrumb: true
    }
  }
];

@NgModule({
  imports: [RouterModule.forChild(routes)],
  exports: [RouterModule]
})
export class RulesRoutingModule {}

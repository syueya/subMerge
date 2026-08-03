import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { RuleEditorComponent } from './rule-editor/rule-editor.component';

const routes: Routes = [
  {
    path: '',
    component: RuleEditorComponent,
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

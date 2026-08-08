import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';

import { TokenListComponent } from './token-list/token-list.component';

const routes: Routes = [
  {
    path: '',
    component: TokenListComponent,
    data: {
      title: '令牌',
      hideBreadcrumb: true
    }
  }
];

@NgModule({
  imports: [RouterModule.forChild(routes)],
  exports: [RouterModule]
})
export class TokensRoutingModule {}

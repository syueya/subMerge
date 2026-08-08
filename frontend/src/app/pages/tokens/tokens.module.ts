import { NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { AppCommonModule } from '@common/common.module';

import { TokenFormComponent } from './token-form/token-form.component';
import { TokenListComponent } from './token-list/token-list.component';
import { TokensRoutingModule } from './tokens-routing.module';

@NgModule({
  declarations: [TokenListComponent, TokenFormComponent],
  imports: [AppCommonModule, RouterModule, TokensRoutingModule]
})
export class TokensModule {}

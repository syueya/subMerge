import { NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { AppCommonModule } from '@common/common.module';
import { TokenEditComponent } from './token-edit/token-edit.component';
import { TokenFormComponent } from './token-form/token-form.component';
import { TokenListComponent } from './token-list/token-list.component';
import { TokensRoutingModule } from './tokens-routing.module';

@NgModule({
  declarations: [TokenListComponent, TokenFormComponent, TokenEditComponent],
  imports: [AppCommonModule, RouterModule, TokensRoutingModule]
})
export class TokensModule {}

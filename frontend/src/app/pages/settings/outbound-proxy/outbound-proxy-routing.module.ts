import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { OutboundProxyComponent } from './outbound-proxy.component';

const routes: Routes = [
  {
    path: '',
    component: OutboundProxyComponent,
    data: {
      hideBreadcrumb: true,
    },
  },
];

@NgModule({
  imports: [RouterModule.forChild(routes)],
  exports: [RouterModule],
})
export class OutboundProxyRoutingModule {}

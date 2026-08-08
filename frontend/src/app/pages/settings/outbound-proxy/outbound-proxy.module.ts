import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { AppCommonModule } from '@common/common.module';
import { OutboundProxyComponent } from './outbound-proxy.component';
import { OutboundProxyRoutingModule } from './outbound-proxy-routing.module';

@NgModule({
  declarations: [OutboundProxyComponent],
  imports: [CommonModule, AppCommonModule, OutboundProxyRoutingModule],
})
export class OutboundProxyModule {}

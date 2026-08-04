import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { NetCheckPageComponent } from './net-check-page/net-check-page.component';

const routes: Routes = [
	{
		path: '',
		component: NetCheckPageComponent,
		data: { title: '网络检测', hideBreadcrumb: true },
	},
];

@NgModule({
	imports: [RouterModule.forChild(routes)],
	exports: [RouterModule],
})
export class NetCheckRoutingModule {}

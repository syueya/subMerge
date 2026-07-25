import { Routes } from '@angular/router';
import { authGuard } from './features/auth/auth.guard';
import { LoginComponent } from './features/auth/login.component';
import { DashboardComponent } from './features/dashboard/dashboard.component';
import { ShellComponent } from './features/layout/shell.component';
import { SourceListComponent } from './features/sources/source-list.component';
import { GroupListComponent } from './features/groups/group-list.component';
import { RuleEditorComponent } from './features/rules/rule-editor.component';
import { ReleaseListComponent } from './features/releases/release-list.component';
import { TokenListComponent } from './features/tokens/token-list.component';

export const routes: Routes = [
	{ path: 'login', component: LoginComponent },
	{
		path: '',
		component: ShellComponent,
		canActivate: [authGuard],
		children: [
			{ path: '', pathMatch: 'full', redirectTo: 'home' },
			{ path: 'home', component: DashboardComponent },
			{ path: 'sources', component: SourceListComponent },
			{ path: 'groups', component: GroupListComponent },
			{ path: 'rules', component: RuleEditorComponent },
			{ path: 'releases', component: ReleaseListComponent },
			{ path: 'tokens', component: TokenListComponent },
		],
	},
	{ path: '**', redirectTo: 'home' },
];

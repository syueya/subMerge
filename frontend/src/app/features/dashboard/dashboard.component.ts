import { Component, OnInit, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { catchError, forkJoin, of } from 'rxjs';
import { DialogService } from '../../common/dialog/dialog.service';
import { formatDateTime } from '../../common/format';
import { DraftStatus, Release } from '../../common/types';
import { ReleaseService } from '../releases/release.service';
import { RuleService } from '../rules/rule.service';
import { SourceService } from '../sources/source.service';
import { TokenService } from '../tokens/token.service';

@Component({
	selector: 'app-dashboard',
	standalone: true,
	imports: [RouterLink],
	templateUrl: './dashboard.component.html',
})
export class DashboardComponent implements OnInit {
	private readonly sources = inject(SourceService);
	private readonly rules = inject(RuleService);
	private readonly releases = inject(ReleaseService);
	private readonly tokens = inject(TokenService);
	private readonly dialog = inject(DialogService);

	loading = signal(true);

	sourceCount = signal(0);
	enabledSourceCount = signal(0);
	proxyCount = signal(0);
	groupCount = signal(0);
	ruleCount = signal(0);
	tokenCount = signal(0);
	activeTokenCount = signal(0);

	latestRelease = signal<Release | null>(null);
	draft = signal<DraftStatus | null>(null);

	readonly steps = [
		{
			n: '1',
			title: '订阅源',
			desc: '添加 Clash 订阅并拉取；节点会带地区前缀和源备注后缀',
			path: '/sources',
			action: '去管理',
		},
		{
			n: '2',
			title: '规则',
			desc: '策略组用直连/拒绝/各国即可；业务分流写规则（如 AI→US）',
			path: '/rules',
			action: '去编辑',
		},
		{
			n: '3',
			title: '发布',
			desc: '草稿不会自动生效，发布后订阅才更新；可回滚历史版本',
			path: '/releases',
			action: '去发布',
		},
		{
			n: '4',
			title: '令牌',
			desc: '生成独立链接发给朋友；重生后旧链接失效',
			path: '/tokens',
			action: '去创建',
		},
	];

	ngOnInit(): void {
		this.reload();
	}

	reload(): void {
		this.loading.set(true);
		forkJoin({
			sources: this.sources.list().pipe(catchError(() => of({ items: [] }))),
			proxies: this.sources.listProxies().pipe(catchError(() => of({ items: [] }))),
			groups: this.rules.listGroups().pipe(catchError(() => of({ items: [] }))),
			rules: this.rules.listRules().pipe(catchError(() => of({ items: [] }))),
			tokens: this.tokens.list().pipe(catchError(() => of({ items: [] }))),
			releases: this.releases.list().pipe(catchError(() => of({ items: [] }))),
			draft: this.releases.draftStatus().pipe(
				catchError(() =>
					of({
						hasPublished: false,
						dirty: false,
					} as DraftStatus),
				),
			),
		}).subscribe({
			next: (r) => {
				const src = r.sources.items || [];
				this.sourceCount.set(src.length);
				this.enabledSourceCount.set(src.filter((s) => s.enabled).length);
				this.proxyCount.set((r.proxies.items || []).length);
				this.groupCount.set((r.groups.items || []).length);
				this.ruleCount.set((r.rules.items || []).length);
				const toks = r.tokens.items || [];
				this.tokenCount.set(toks.length);
				this.activeTokenCount.set(toks.filter((t) => t.status === 'active').length);
				const rels = r.releases.items || [];
				this.latestRelease.set(rels.find((x) => x.status === 'published') || rels[0] || null);
				this.draft.set(r.draft);
				this.loading.set(false);
			},
			error: (e: Error) => {
				this.loading.set(false);
				void this.dialog.error(e.message);
			},
		});
	}

	draftLabel(): string {
		const d = this.draft();
		if (!d) return '—';
		if (d.buildError) return this.shortBuildError(d.buildError);
		if (!d.hasPublished) return '尚未发布';
		if (d.dirty) return '有未发布更改';
		return '已与线上一致';
	}

	draftClass(): string {
		const d = this.draft();
		if (!d) return 'badge-muted';
		if (d.buildError) return 'badge-err';
		if (!d.hasPublished || d.dirty) return 'badge-warn';
		return 'badge-ok';
	}

	/** 概览卡片用短文案；完整错误放 title */
	shortBuildError(err: string): string {
		const e = (err || '').toLowerCase();
		if (e.includes('no proxies') || e.includes('proxies available')) return '无可用节点';
		if (e.includes('proxy group') || e.includes('groups')) return '策略组不可用';
		if (e.includes('match')) return '缺少 MATCH 规则';
		if (e.includes('rule')) return '规则不完整';
		return '草稿异常';
	}

	buildErrorDetail(): string {
		return this.draft()?.buildError || '';
	}

	formatTime = formatDateTime;
}

import { Component, OnInit, inject, signal } from '@angular/core';
import { catchError, forkJoin, of } from 'rxjs';
import { DialogService } from '@common/services/dialog.service';
import { formatDateTime } from '@common/util/format';
import { Release } from '@data-struct';
import { DraftStatusStore } from '../../releases/services/draft-status.store';
import { ReleaseService } from '../../releases/services/release.service';
import { RuleService } from '../../rules/services/rule.service';
import { SourceService } from '../../sources/services/source.service';
import { TokenService } from '../../tokens/services/token.service';
import { CmParentComponent } from '@common/parents/parent/parent.component';
import { takeUntil } from 'rxjs';

@Component({
	selector: 'app-dashboard-home',
	templateUrl: './dashboard-home.component.html',
	standalone: false,
})
export class DashboardHomeComponent extends CmParentComponent implements OnInit {
	private readonly sources = inject(SourceService);
	private readonly rules = inject(RuleService);
	private readonly releases = inject(ReleaseService);
	private readonly tokens = inject(TokenService);
	private readonly draftStore = inject(DraftStatusStore);
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
	// 草稿状态来自共享 store（模板沿用 draft() 读取）
	draft = this.draftStore.status;
	draftSummary = this.draftStore.summary;

	readonly steps = [
		{
			n: '1',
			title: '订阅源',
			desc: '添加 Clash 订阅并拉取；节点会带地区前缀和源备注后缀',
			path: '/main/sources',
			action: '去管理',
			icon: 'cloud-download',
		},
		{
			n: '2',
			title: '策略组',
			desc: '配置出口容器：直连/拒绝/各国节点成员与测速方式',
			path: '/main/groups',
			action: '去配置',
			icon: 'hierarchy-2',
		},
		{
			n: '3',
			title: '分流规则',
			desc: '按业务写匹配（如广告→拒绝、AI→美国）；出口指向策略组',
			path: '/main/rules',
			action: '去编辑',
			icon: 'list-check',
		},
		{
			n: '4',
			title: '发布',
			desc: '草稿不会自动生效，发布后订阅才更新；可回滚历史版本',
			path: '/main/releases',
			action: '去发布',
			icon: 'rocket',
		},
		{
			n: '5',
			title: '令牌',
			desc: '生成独立链接发给朋友；重生后旧链接失效',
			path: '/main/tokens',
			action: '去创建',
			icon: 'key',
		},
	];

	override ngOnInit(): void {
		super.ngOnInit();
		this.reload();
	}

	reload(force = false): void {
		this.loading.set(true);
		this.draftStore.refresh();
		forkJoin({
			sources: this.sources.list(force).pipe(catchError(() => of({ items: [] }))),
			proxies: this.sources.listProxies(undefined, force).pipe(catchError(() => of({ items: [] }))),
			groups: this.rules.listGroups(force).pipe(catchError(() => of({ items: [] }))),
			rules: this.rules.listRules(force).pipe(catchError(() => of({ items: [] }))),
			tokens: this.tokens.list(force).pipe(catchError(() => of({ items: [] }))),
			releases: this.releases.list(force).pipe(catchError(() => of({ items: [] }))),
		})
			.pipe(takeUntil(this.$destroy))
			.subscribe({
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
					this.loading.set(false);
				},
				error: (e: Error) => {
					this.loading.set(false);
					void this.dialog.error(e.message);
				},
			});
	}

	/** 发布卡第三行：完整说明（与其它统计卡同为 f-s-12 一行） */
	publishStatusText(): string {
		const d = this.draft();
		if (!d) return '—';
		const err = String(d.buildError || '').trim();
		if (err) return err;
		if (!d.hasPublished) return '尚未发布';
		if (d.dirty) return this.draftSummary() || '有未发布更改';
		const time = this.formatTime(this.latestRelease()?.publishedAt || this.latestRelease()?.createdAt);
		return time ? `已与线上一致 · ${time}` : '已与线上一致';
	}

	buildErrorDetail(): string {
		return String(this.draft()?.buildError || '').trim();
	}

	formatTime = formatDateTime;
}

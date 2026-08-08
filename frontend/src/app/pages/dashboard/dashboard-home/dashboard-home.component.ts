import { Component, OnInit, inject, signal } from '@angular/core';
import { CmParentComponent } from '@common/parents/parent/parent.component';
import { DialogService } from '@common/services/dialog.service';
import { formatDateTime } from '@common/util';
import { BADGE_WARN, Release } from '@data-struct';
import { catchError, of, takeUntil } from 'rxjs';

import { DraftStatusStore } from '../../releases/services/draft-status.store';
import { ReleaseService } from '../../releases/services/release.service';
import { RuleService } from '../../rules/services/rule.service';
import { SourceService } from '../../sources/services/source.service';
import { TokenService } from '../../tokens/services/token.service';


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

	/** 首屏骨架：不挡整页，各卡片可独立显示 */
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
	readonly badgeWarn = BADGE_WARN;

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

	private pendingLoads = 0;

	override ngOnInit(): void {
		super.ngOnInit();
		this.reload();
	}

	/**
	 * 分批独立订阅，避免 forkJoin 等最慢接口才出数；
	 * 单路失败不影响其它卡片。
	 */
	reload(force = false): void {
		this.loading.set(true);
		this.pendingLoads = 6;
		this.draftStore.refresh(force);

		this.sources
			.list(force)
			.pipe(
				takeUntil(this.$destroy),
				catchError(() => of({ items: [] })),
			)
			.subscribe((r) => {
				const src = r.items || [];
				this.sourceCount.set(src.length);
				this.enabledSourceCount.set(src.filter((s) => s.enabled).length);
				this.markLoaded();
			});

		this.sources
			.listProxies(undefined, force)
			.pipe(
				takeUntil(this.$destroy),
				catchError(() => of({ items: [] })),
			)
			.subscribe((r) => {
				this.proxyCount.set((r.items || []).length);
				this.markLoaded();
			});

		this.rules
			.listGroups(force)
			.pipe(
				takeUntil(this.$destroy),
				catchError(() => of({ items: [] })),
			)
			.subscribe((r) => {
				this.groupCount.set((r.items || []).length);
				this.markLoaded();
			});

		this.rules
			.listRules(force)
			.pipe(
				takeUntil(this.$destroy),
				catchError(() => of({ items: [] })),
			)
			.subscribe((r) => {
				this.ruleCount.set((r.items || []).length);
				this.markLoaded();
			});

		this.tokens
			.list(force)
			.pipe(
				takeUntil(this.$destroy),
				catchError(() => of({ items: [] })),
			)
			.subscribe((r) => {
				const toks = r.items || [];
				this.tokenCount.set(toks.length);
				this.activeTokenCount.set(toks.filter((t) => t.status === 'active').length);
				this.markLoaded();
			});

		this.releases
			.list(force)
			.pipe(
				takeUntil(this.$destroy),
				catchError((e: Error) => {
					void this.dialog.error(e.message);
					return of({ items: [] as Release[] });
				}),
			)
			.subscribe((r) => {
				const rels = r.items || [];
				this.latestRelease.set(rels.find((x) => x.status === 'published') || rels[0] || null);
				this.markLoaded();
			});
	}

	private markLoaded(): void {
		this.pendingLoads = Math.max(0, this.pendingLoads - 1);
		if (this.pendingLoads === 0) {
			this.loading.set(false);
		}
	}

	/** 发布卡第三行：完整说明（与其它统计卡同为 f-s-12 一行） */
	publishStatusText(): string {
		const d = this.draft();
		if (!d) return '—';
		const err = String(d.buildError || '').trim();
		if (err) return err;
		if (!d.hasPublished) return '尚未发布';
		if (d.dirty) {
			// 轻量 draft-status 默认不带 changes；有 summary 再展示，否则通用文案
			return this.draftSummary() || '有未发布更改';
		}
		const time = this.formatTime(this.latestRelease()?.publishedAt || this.latestRelease()?.createdAt);
		return time ? `已与线上一致 · ${time}` : '已与线上一致';
	}

	buildErrorDetail(): string {
		return String(this.draft()?.buildError || '').trim();
	}

	formatTime = formatDateTime;
}

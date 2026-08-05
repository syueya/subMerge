import { Component, OnInit, inject, signal } from '@angular/core';
import {
		BADGE_MUTED,
		BADGE_OK,
		BADGE_WARN,
		GroupFormDialogData,
		PROXY_GROUP_TYPE_OPTIONS,
		ProxyGroup,
		Rule,
		enumText,
	} from '@data-struct';
	import { DialogService } from '@common/services/dialog.service';
	import { DraftStatusStore } from '../../releases/services/draft-status.store';
	import { SourceService } from '../../sources/services/source.service';
	import { RuleService } from '../../rules/services/rule.service';
	import { CM_DIALOG_WIDTH, CmDialogOpenService } from '@common/modules/dialog';
	import { takeUntil } from 'rxjs';
	import { CmParentComponent } from '@common/parents/parent/parent.component';
	import { GroupFormComponent } from '../group-form/group-form.component';

@Component({
	selector: 'app-group-list',
	templateUrl: './group-list.component.html',
	standalone: false,
})
export class GroupListComponent extends CmParentComponent implements OnInit {
	private svc = inject(RuleService);
	private draftStore = inject(DraftStatusStore);
	private sourceSvc = inject(SourceService);
	private dialog = inject(DialogService);
	private dialogOpen = inject(CmDialogOpenService);

	groups = signal<ProxyGroup[]>([]);
	rules = signal<Rule[]>([]);
	regionCatalog = signal<{ code: string; name: string }[]>([]);
	extraRegionCodes = signal<string[]>([]);
	knownSources = signal<{ id: number; name: string }[]>([]);
	override isLoading = signal(true);

	readonly defaultTestInterval = 300;
	readonly badgeWarn = BADGE_WARN;

	override ngOnInit(): void {
		super.ngOnInit();
		// 进页走会话缓存；工具栏刷新再 force
		this.reload();
	}

/** 写成功后静默同步草稿角标（进页仍不请求 draft-status） */
		private notifyDraftChanged(): void {
			this.draftStore.refresh();
		}

		/** force 绕过双层缓存；silent 不挡内容（写后对齐） */
		reload(force = false, silent = false): void {
			const groups$ = this.svc.listGroups(force).pipe(takeUntil(this.$destroy));
			const pipeGroups = silent ? groups$ : groups$.pipe(this.trackLoading());
			pipeGroups.subscribe({
				next: (r) => {
					this.groups.set(r.items || []);
				},
				error: (e: Error) => {
					void this.dialog.error(e.message);
				},
			});
			this.svc
				.listRules(force)
				.pipe(takeUntil(this.$destroy))
				.subscribe({
					next: (r) => this.rules.set(r.items || []),
					error: () => {},
				});
			this.sourceSvc
				.listRegions()
				.pipe(takeUntil(this.$destroy))
				.subscribe({
					next: (r) => {
						const items = (r.items || [])
							.map((item) => ({
								code: String(item.code || '').toUpperCase(),
								name: String(item.name || '').trim(),
							}))
							.filter((item) => item.code && item.code !== 'UNK' && item.code !== 'UNKNOWN');
						this.regionCatalog.set(items);
					},
					error: () => {},
				});
			this.sourceSvc
				.list(force)
				.pipe(takeUntil(this.$destroy))
				.subscribe({
					next: (r) => {
						const sources: { id: number; name: string }[] = [];
						const extras = new Set(this.extraRegionCodes());
						const catalogCodes = new Set(this.regionCatalog().map((x) => x.code));
						for (const s of r.items || []) {
							const c = String(s.region || '').toUpperCase();
							if (c && c !== 'UNK' && c !== 'UNKNOWN' && !catalogCodes.has(c)) extras.add(c);
							const name = String(s.name || '').trim();
							if (s.enabled && name) sources.push({ id: s.id, name });
						}
						this.extraRegionCodes.set([...extras].sort());
						sources.sort((a, b) => a.name.localeCompare(b.name, 'zh'));
						this.knownSources.set(sources);
					},
					error: () => {},
				});
			this.sourceSvc
				.listProxies(undefined, force)
				.pipe(takeUntil(this.$destroy))
				.subscribe({
					next: (r) => {
						const extras = new Set(this.extraRegionCodes());
						const catalogCodes = new Set(this.regionCatalog().map((x) => x.code));
						for (const p of r.items || []) {
							const c = String(p.region || '').toUpperCase();
							if (c && c !== 'UNK' && c !== 'UNKNOWN' && !catalogCodes.has(c)) extras.add(c);
						}
						this.extraRegionCodes.set([...extras].sort());
					},
					error: () => {},
				});
		}

	rulesOfGroup(name: string): number {
		return this.rules().filter((r) => r.target === name).length;
	}

	groupTypeText(v: string): string {
		return enumText(PROXY_GROUP_TYPE_OPTIONS, v);
	}

	groupTypeBadgeClass(type: string): string {
		switch (type) {
			case 'select':
				return BADGE_MUTED;
			case 'url-test':
				return BADGE_OK;
			case 'fallback':
				return BADGE_WARN;
			case 'load-balance':
				return BADGE_OK;
			default:
				return BADGE_MUTED;
		}
	}

	groupTypeTip(type: string): string {
		switch (type) {
			case 'select':
				return '手动选择：在客户端里自己挑节点';
			case 'url-test':
				return '自动测速：客户端定时测延迟，自动选最快节点';
			case 'fallback':
				return '故障转移：按成员顺序，挂了切下一个';
			case 'load-balance':
				return '负载均衡：在成员间分摊流量';
			default:
				return this.groupTypeText(type);
		}
	}

	openCreateGroup(): void {
		this.openForm(null);
	}

	openEditGroup(g: ProxyGroup): void {
		this.openForm(g);
	}

	private openForm(group: ProxyGroup | null): void {
		const data: GroupFormDialogData = {
			group,
			groups: this.groups(),
			regionCatalog: this.regionCatalog(),
			extraRegionCodes: this.extraRegionCodes(),
			knownSources: this.knownSources(),
		};
		const ref = this.dialogOpen.openForm(GroupFormComponent, data, {
			width: CM_DIALOG_WIDTH.form,
		});
ref.afterClosed().subscribe((ok) => {
				if (ok) {
					// 写操作已 invalidate 缓存；silent 对齐，不整页转圈
					this.reload(false, true);
					this.notifyDraftChanged();
				}
			});
		}

	async removeGroup(g: ProxyGroup): Promise<void> {
		const count = this.rulesOfGroup(g.name);
		let cascade = false;
		if (count > 0) {
			const delRules = await this.dialog.confirm(
				`策略组「${g.name}」被 ${count} 条规则引用。\n\n选「删除」：连同这些规则一起删掉。\n选「取消」后可再决定只删组（规则会变成出口失效）。\n\n是否删除组及其规则？`,
				'删除策略组',
				'删除组+规则',
			);
			if (!delRules) {
				const onlyGroup = await this.dialog.confirm(
					`仅删除策略组「${g.name}」，保留 ${count} 条规则（将显示为出口失效）？`,
					'仅删组',
					'仅删组',
				);
				if (!onlyGroup) return;
				cascade = false;
			} else {
				cascade = true;
			}
		} else {
			const ok = await this.dialog.confirm(`确认删除策略组「${g.name}」？`, '删除确认', '删除');
			if (!ok) return;
		}
		this.svc
			.deleteGroup(g.id, cascade)
			.pipe(takeUntil(this.$destroy))
			.subscribe({
next: () => {
						void this.dialog.success(
							cascade ? `已删除策略组「${g.name}」及其规则` : `已删除策略组「${g.name}」`,
						);
						this.reload(false, true);
						this.notifyDraftChanged();
					},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}
}

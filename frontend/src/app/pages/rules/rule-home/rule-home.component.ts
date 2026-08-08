import { Component, OnInit, computed, inject, signal } from '@angular/core';
import {
	BADGE_MUTED,
	BADGE_OK,
	BADGE_WARN,
	BatchImportDialogData,
	CategorySection,
	MatchableRule,
	ProxyGroup,
	RULE_TYPE_OPTIONS,
	Rule,
	RuleFormDialogData,
	RuleMatchDialogData,
	RuleMatchDialogResult,
	RuleType,
	enumText,
} from '@data-struct';
import { DialogService } from '@common/services/dialog.service';
	import { DraftStatusStore } from '../../releases/services/draft-status.store';
	import { RuleService } from '../services/rule.service';
import {
	buildCategorySections,
	buildTargetSections,
	canMoveRule,
	canMoveRuleGroup,
	canMoveRuleWithinGroup,
	categoryToRemember,
	defaultRuleTarget,
	isSystemRule,
	moveRuleGroup,
	moveRuleOrder,
	moveRuleWithinGroup,
	sortRules,
} from '../services/rule-ui';
import { CM_DIALOG_WIDTH, CmDialogOpenService } from '@common/modules/dialog';
import { CmParentComponent } from '@common/parents/parent/parent.component';
import { takeUntil, finalize } from 'rxjs';
import { RuleMatchDialogComponent } from '../../_shared/rule-match-dialog/rule-match-dialog.component';
import { BatchImportComponent } from '../batch-import/batch-import.component';
import { NewCategoryFormComponent } from '../new-category-form/new-category-form.component';
import { RuleFormComponent } from '../rule-form/rule-form.component';

@Component({
	selector: 'app-rule-home',
	templateUrl: './rule-home.component.html',
	standalone: false,
})
export class RuleHomeComponent extends CmParentComponent implements OnInit {
	private svc = inject(RuleService);
	private draftStore = inject(DraftStatusStore);
	private dialog = inject(DialogService);
	private dialogOpen = inject(CmDialogOpenService);

	rules = signal<Rule[]>([]);
	groups = signal<ProxyGroup[]>([]);
	extraCategories = signal<string[]>([]);
	override isLoading = signal(true);

	/** 搜索：匹配 payload / 类型 / 出口 / 备注 / 分类 */
	filterText = signal('');

	viewMode = signal<'category' | 'target'>('category');
	/** 默认全部折叠，展开后再渲染 table，减轻首屏 DOM */
	expandedCategoryKeys = signal<Set<string>>(new Set());
	expandedTargetKeys = signal<Set<string>>(new Set());

	selectedIds = signal<Set<number>>(new Set());
	batchTarget = signal('');
	batchBusy = signal(false);
	sortBusy = signal(false);
	selectedCount = computed(() => this.selectedIds().size);

	readonly badgeOk = BADGE_OK;
	readonly badgeWarn = BADGE_WARN;
	readonly badgeMuted = BADGE_MUTED;

	private readonly sorted = computed(() => sortRules(this.rules()));
	private readonly orderNoById = computed(() => {
		const map = new Map<number, number>();
		// 全局序号按完整列表，不受搜索过滤影响
		this.sorted().forEach((r, i) => map.set(r.id, i + 1));
		return map;
	});
	private readonly groupNames = computed(() => new Set(this.groups().map((g) => g.name)));

	private readonly filteredRules = computed(() => {
		const q = this.filterText().trim().toLowerCase();
		const list = this.sorted();
		if (!q) return list;
		return list.filter((r) => {
			const typeText = this.ruleTypeText(r.type).toLowerCase();
			const hay = [
				r.payload || '',
				String(r.type),
				typeText,
				r.target || '',
				r.note || '',
				r.category || '',
			]
				.join('\n')
				.toLowerCase();
			return hay.includes(q);
		});
	});

	/** 模板直接绑定，避免变更检测里反复调用方法 */
	readonly sections = computed<CategorySection[]>(() => {
		if (this.viewMode() === 'target') {
			return buildTargetSections(
				this.filteredRules(),
				this.groups().map((g) => g.name),
			);
		}
		return buildCategorySections(this.filteredRules(), this.extraCategories());
	});

	readonly allSections = computed<CategorySection[]>(() => {
		if (this.viewMode() === 'target') {
			return buildTargetSections(this.sorted(), this.groups().map((g) => g.name));
		}
		return buildCategorySections(this.sorted(), this.extraCategories());
	});

	readonly orphanCount = computed(
		() => this.sorted().filter((r) => this.isOrphanTarget(r.target)).length,
	);

	readonly filteredCount = computed(() => this.filteredRules().length);

	testMatchRules = computed<MatchableRule[]>(() =>
		this.sorted()
			.filter((r) => r.enabled)
			.map((r) => ({
				type: String(r.type),
				payload: r.payload || '',
				target: r.target,
				enabled: true,
			})),
	);

	override ngOnInit(): void {
		super.ngOnInit();
		// 进页走会话缓存；工具栏刷新再 force
		this.reload();
	}

	ruleTypeText(v: string | RuleType): string {
		return enumText(RULE_TYPE_OPTIONS, String(v));
	}

	globalOrderNo(rule: Rule): number {
		return this.orderNoById().get(rule.id) ?? 0;
	}

	isOrphanTarget(target: string): boolean {
		return !!target && !this.groupNames().has(target);
	}

	isSystemRule(rule: { type?: string; payload?: string } | null | undefined): boolean {
		return isSystemRule(rule);
	}

	canMoveRule(rule: Rule, action: 'up' | 'down'): boolean {
		return !this.sortBusy() && canMoveRule(this.rules(), rule.id, action);
	}

	canMoveRuleWithinGroup(rule: Rule): boolean {
		return (
			!this.sortBusy() &&
			canMoveRuleWithinGroup(this.rules(), rule.id, this.viewMode())
		);
	}

	canMoveRuleTop(rule: Rule): boolean {
		return this.canMoveRuleWithinGroup(rule);
	}

	hasMovableSection(section: CategorySection): boolean {
		return section.rules.some((rule) => !this.isSystemRule(rule));
	}

	canMoveSection(key: string, action: 'up' | 'down'): boolean {
		const order = this.allSections()
			.filter((section) => section.rules.some((rule) => !this.isSystemRule(rule)))
			.map((section) => section.key);
		return (
			!this.sortBusy() &&
			canMoveRuleGroup(this.rules(), key, this.viewMode(), action, order)
		);
	}

	private persistRuleOrder(previous: Rule[], next: Rule[]): void {
		const orderedIds = sortRules(next).map((item) => item.id);
		this.rules.set(next);
		this.sortBusy.set(true);
		this.svc
			.reorder(orderedIds)
			.pipe(
				takeUntil(this.$destroy),
				finalize(() => this.sortBusy.set(false)),
			)
			.subscribe({
				next: () => {
					this.reload(false, true);
					this.notifyDraftChanged();
				},
				error: (e: Error) => {
					this.rules.set(previous);
					void this.dialog.error(e.message);
				},
			});
	}

	moveRule(rule: Rule, action: 'up' | 'down' | 'top'): void {
		if (action === 'top') {
			if (!this.canMoveRuleWithinGroup(rule)) return;
			const previous = this.rules();
			this.persistRuleOrder(previous, moveRuleWithinGroup(previous, rule.id, this.viewMode()));
			return;
		}
		if (!this.canMoveRule(rule, action)) return;
		const previous = this.rules();
		this.persistRuleOrder(previous, moveRuleOrder(previous, rule.id, action));
	}

	moveSection(key: string, action: 'up' | 'down'): void {
		if (!this.canMoveSection(key, action)) return;
		const previous = this.rules();
		const order = this.allSections()
			.filter((section) => section.rules.some((rule) => !this.isSystemRule(rule)))
			.map((section) => section.key);
		this.persistRuleOrder(previous, moveRuleGroup(previous, key, this.viewMode(), action, order));
	}

	setViewMode(mode: 'category' | 'target'): void {
		if (this.viewMode() === mode) return;
		this.viewMode.set(mode);
	}

	onFilterInput(value: string): void {
		this.filterText.set(value);
	}

	clearFilter(): void {
		this.filterText.set('');
	}

	rememberCategory(name: string): void {
		const n = categoryToRemember(name, this.rules(), this.extraCategories());
		if (!n) return;
		this.extraCategories.update((xs) => [...xs, n]);
	}

	private expandedKeys(): Set<string> {
		return this.viewMode() === 'target' ? this.expandedTargetKeys() : this.expandedCategoryKeys();
	}

	isSectionExpanded(key: string): boolean {
		return this.expandedKeys().has(key);
	}

	setSectionExpanded(key: string, open: boolean): void {
		const cur = this.expandedKeys();
		const next = new Set(cur);
		if (open) next.add(key);
		else next.delete(key);
		if (this.viewMode() === 'target') this.expandedTargetKeys.set(next);
		else this.expandedCategoryKeys.set(next);
	}

	/** 当前视图下是否已全部展开（用于切换按钮文案） */
	allSectionsExpanded = computed(() => {
		const secs = this.sections();
		if (!secs.length) return false;
		const open = this.expandedKeys();
		return secs.every((s) => open.has(s.key));
	});

	/** 一个按钮切换：全展 ↔ 全折 */
	toggleExpandAll(): void {
		if (this.allSectionsExpanded()) {
			if (this.viewMode() === 'target') this.expandedTargetKeys.set(new Set());
			else this.expandedCategoryKeys.set(new Set());
			return;
		}
		const keys = new Set(this.sections().map((s) => s.key));
		if (this.viewMode() === 'target') this.expandedTargetKeys.set(keys);
		else this.expandedCategoryKeys.set(keys);
	}

	/** 写成功后静默同步草稿角标（不挡本页；进页仍不请求 draft-status） */
	private notifyDraftChanged(): void {
		this.draftStore.refresh();
	}

	/** force=true 时绕过双层缓存；silent 时不挡内容（写后本地已 patch 的补拉） */
	reload(force = false, silent = false): void {
		const rules$ = this.svc.listRules(force).pipe(takeUntil(this.$destroy));
		const pipeRules = silent ? rules$ : rules$.pipe(this.trackLoading());
		pipeRules.subscribe({
			next: (r) => {
				const list = r.items || [];
				this.rules.set(list);
				const valid = new Set(list.map((x) => x.id));
				this.selectedIds.update((prev) => {
					const next = new Set<number>();
					for (const id of prev) {
						if (valid.has(id)) next.add(id);
					}
					return next;
				});
			},
			error: (e: Error) => {
				void this.dialog.error(e.message);
			},
		});
		this.svc
			.listGroups(force)
			.pipe(takeUntil(this.$destroy))
			.subscribe({
				next: (r) => {
					this.groups.set(r.items || []);
					if (!this.batchTarget()) {
						this.batchTarget.set(this.defaultTarget() || '直连');
					}
				},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}

	isSelected(id: number): boolean {
		return this.selectedIds().has(id);
	}

	toggleSelect(id: number, checked: boolean): void {
		this.selectedIds.update((prev) => {
			const next = new Set(prev);
			if (checked) next.add(id);
			else next.delete(id);
			return next;
		});
	}

	isSectionAllSelected(sec: { rules: Rule[] }): boolean {
		if (!sec.rules.length) return false;
		const sel = this.selectedIds();
		return sec.rules.every((r) => sel.has(r.id));
	}

	isSectionSomeSelected(sec: { rules: Rule[] }): boolean {
		if (!sec.rules.length) return false;
		const sel = this.selectedIds();
		let n = 0;
		for (const r of sec.rules) {
			if (sel.has(r.id)) n++;
		}
		return n > 0 && n < sec.rules.length;
	}

	toggleSectionSelect(sec: { rules: Rule[] }, checked: boolean): void {
		this.selectedIds.update((prev) => {
			const next = new Set(prev);
			for (const r of sec.rules) {
				if (checked) next.add(r.id);
				else next.delete(r.id);
			}
			return next;
		});
	}

	clearSelection(): void {
		this.selectedIds.set(new Set());
	}

	async applyBatchTarget(): Promise<void> {
		const ids = [...this.selectedIds()];
		const target = this.batchTarget().trim();
		if (!ids.length) {
			void this.dialog.error('请先勾选规则');
			return;
		}
		if (!target) {
			void this.dialog.error('请选择出口');
			return;
		}
		const ok = await this.dialog.confirm(
			`将已选 ${ids.length} 条规则的出口改为「${target}」？`,
			'批量改出口',
			'确认',
		);
		if (!ok) return;
		const idSet = new Set(ids);
		const prev = this.rules();
		// 乐观更新
		this.rules.set(prev.map((r) => (idSet.has(r.id) ? { ...r, target } : r)));
		this.batchBusy.set(true);
		this.svc
			.batchUpdateTarget(ids, target)
			.pipe(takeUntil(this.$destroy))
			.subscribe({
				next: (res) => {
					this.batchBusy.set(false);
					void this.dialog.success(`已更新 ${res.updated} 条出口（草稿，需发布后生效）`);
					this.clearSelection();
					this.reload(false, true);
					this.notifyDraftChanged();
				},
				error: (e: Error) => {
					this.batchBusy.set(false);
					this.rules.set(prev);
					void this.dialog.error(e.message);
				},
			});
	}

	defaultTarget(): string {
		return defaultRuleTarget(this.groups().map((g) => g.name));
	}

	openCreateRule(opts?: { category?: string; target?: string } | string): void {
		let category = '';
		let target = '';
		if (typeof opts === 'string') {
			category = opts;
		} else {
			category = opts?.category ?? '';
			target = opts?.target ?? '';
		}
		this.openRuleForm(null, category, target || this.defaultTarget() || '直连');
	}

	openEditRule(rule: Rule): void {
		this.openRuleForm(rule, '', '');
	}

	private openRuleForm(rule: Rule | null, defaultCategory: string, defaultTarget: string): void {
		const data: RuleFormDialogData = {
			rule,
			groups: this.groups(),
			rules: this.rules(),
			extraCategories: this.extraCategories(),
			defaultCategory,
			defaultTarget,
		};
		const ref = this.dialogOpen.openForm(RuleFormComponent, data, {
			width: CM_DIALOG_WIDTH.form,
		});
		ref.afterClosed().subscribe((category) => {
			if (category === false || category === undefined) return;
			if (typeof category === 'string' && category) this.rememberCategory(category);
			if (this.viewMode() === 'category' && typeof category === 'string' && category) {
				const expanded = new Set(this.expandedCategoryKeys());
				expanded.add(category);
				this.expandedCategoryKeys.set(expanded);
			} else if (this.viewMode() === 'target' && defaultTarget) {
				const expanded = new Set(this.expandedTargetKeys());
				expanded.add(defaultTarget);
				this.expandedTargetKeys.set(expanded);
			}
			this.reload(false, true);
			this.notifyDraftChanged();
		});
	}

	openNewCategoryModal(): void {
		const ref = this.dialogOpen.openSmallForm(NewCategoryFormComponent, null, {
			width: CM_DIALOG_WIDTH.small,
		});
		ref.afterClosed().subscribe((name) => {
			if (!name) return;
			this.rememberCategory(name);
			this.viewMode.set('category');
			const expanded = new Set(this.expandedCategoryKeys());
			expanded.add(name);
			this.expandedCategoryKeys.set(expanded);
			this.openCreateRule({ category: name });
		});
	}

	openBatchImport(): void {
		const data: BatchImportDialogData = {
			groups: this.groups(),
			rules: this.rules(),
			extraCategories: this.extraCategories(),
			defaultTarget: this.defaultTarget() || '直连',
		};
		const ref = this.dialogOpen.openForm(BatchImportComponent, data, {
			width: CM_DIALOG_WIDTH.form,
		});
		ref.afterClosed().subscribe((category) => {
			if (category === false || category === undefined) return;
			if (typeof category === 'string' && category) this.rememberCategory(category);
			this.reload(false, true);
			this.notifyDraftChanged();
		});
	}

	openTestModal(): void {
		const data: RuleMatchDialogData = {
			title: '测试规则匹配',
			subtitle:
				'按当前启用规则从上到下由服务端模拟匹配（含 DOMAIN / GEOSITE / GEOIP）。使用面板已加载的 Geo 数据。',
			rules: this.testMatchRules(),
			typeText: (t) => this.ruleTypeText(t),
			targetText: (t) => t,
			showEditAction: true,
			showLocateAction: false,
		};
		const ref = this.dialogOpen.openContent(RuleMatchDialogComponent, data, {
			width: CM_DIALOG_WIDTH.large,
		});
		ref.afterClosed().subscribe((result: RuleMatchDialogResult) => {
			if (result?.action === 'edit') {
				const r = this.ruleFromMatchable(result.rule);
				if (r) this.openEditRule(r);
			}
		});
	}

	private ruleFromMatchable(rule: MatchableRule): Rule | null {
		return (
			this.rules().find(
				(r) =>
					String(r.type) === rule.type &&
					(r.payload || '') === (rule.payload || '') &&
					r.target === rule.target,
			) || null
		);
	}

	toggleRule(rule: Rule): void {
		const nextEnabled = !rule.enabled;
		const prev = this.rules();
		this.rules.set(prev.map((r) => (r.id === rule.id ? { ...r, enabled: nextEnabled } : r)));
		this.svc
			.updateRule(rule.id, {
				type: String(rule.type),
				payload: rule.payload,
				target: rule.target,
				enabled: nextEnabled,
				note: rule.note,
				category: rule.category,
				sortOrder: rule.sortOrder,
			})
			.pipe(takeUntil(this.$destroy))
			.subscribe({
				next: () => {
					// 写成功已 invalidate；静默对齐一次即可
					this.reload(false, true);
					this.notifyDraftChanged();
				},
				error: (e: Error) => {
					this.rules.set(prev);
					void this.dialog.error(e.message);
				},
			});
	}

	async removeRule(rule: Rule): Promise<void> {
		if (this.isSystemRule(rule)) {
			void this.dialog.error('系统规则不可删除（广告 / 国内 / 兜底）');
			return;
		}
		const ok = await this.dialog.confirm('确认删除该规则？', '删除确认', '删除');
		if (!ok) return;
		const prev = this.rules();
		this.rules.set(prev.filter((r) => r.id !== rule.id));
		this.selectedIds.update((s) => {
			const next = new Set(s);
			next.delete(rule.id);
			return next;
		});
		this.svc
			.deleteRule(rule.id)
			.pipe(takeUntil(this.$destroy))
.subscribe({
					next: () => {
						void this.dialog.success('规则已删除');
						this.reload(false, true);
						this.notifyDraftChanged();
					},
					error: (e: Error) => {
						this.rules.set(prev);
						void this.dialog.error(e.message);
					},
				});
	}
}

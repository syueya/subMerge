import { Component, OnInit, computed, inject, signal } from '@angular/core';
import {
	BADGE_MUTED,
	BADGE_OK,
	BADGE_WARN,
	RULE_TYPE_OPTIONS,
	ProxyGroup,
	Rule,
	RuleType,
	enumText,
} from '@data-struct';
import { MatchableRule } from '@common/util/rule-match';
import { DialogService } from '@common/services/dialog.service';
import { DraftStatusStore } from '../../releases/services/draft-status.store';
import { RuleService } from '../services/rule.service';
import {
	CategorySection,
	buildCategorySections,
	buildTargetSections,
	categoryToRemember,
	defaultRuleTarget,
	isSystemRule,
	sortRules,
} from '../services/rule-ui';
import { CM_DIALOG_WIDTH, CmDialogOpenService } from '@common/modules/dialog';
import { CmParentComponent } from '@common/parents/parent/parent.component';
import { takeUntil } from 'rxjs';
import {
	RuleMatchDialogComponent,
	RuleMatchDialogData,
	RuleMatchDialogResult,
} from '../../_shared/rule-match-dialog/rule-match-dialog.component';
import { BatchImportComponent, BatchImportDialogData } from '../batch-import/batch-import.component';
import { NewCategoryFormComponent } from '../new-category-form/new-category-form.component';
import { PublishFormComponent } from '../publish-form/publish-form.component';
import { RuleFormComponent, RuleFormDialogData } from '../rule-form/rule-form.component';

@Component({
	selector: 'app-rule-editor',
	templateUrl: './rule-editor.component.html',
	standalone: false,
})
export class RuleEditorComponent extends CmParentComponent implements OnInit {
	private svc = inject(RuleService);
	private draftStore = inject(DraftStatusStore);
	private dialog = inject(DialogService);
	private dialogOpen = inject(CmDialogOpenService);

	rules = signal<Rule[]>([]);
	groups = signal<ProxyGroup[]>([]);
	extraCategories = signal<string[]>([]);
	isLoading = false;

	draftDirty = this.draftStore.dirty;
	draftStatusNote = this.draftStore.note;

	viewMode = signal<'category' | 'target'>('category');
	expandedCategoryKeys = signal<Set<string>>(new Set());
	expandedTargetKeys = signal<Set<string>>(new Set());

	selectedIds = signal<Set<number>>(new Set());
	batchTarget = signal('');
	batchBusy = signal(false);
	selectedCount = computed(() => this.selectedIds().size);

	readonly badgeOk = BADGE_OK;
	readonly badgeWarn = BADGE_WARN;
	readonly badgeMuted = BADGE_MUTED;

	override ngOnInit(): void {
		super.ngOnInit();
		this.reload(true);
		this.draftStore.refresh();
	}

	ruleTypeText(v: string | RuleType): string {
		return enumText(RULE_TYPE_OPTIONS, String(v));
	}

	private readonly sorted = computed(() => sortRules(this.rules()));
	private readonly orderNoById = computed(() => {
		const map = new Map<number, number>();
		this.sorted().forEach((r, i) => map.set(r.id, i + 1));
		return map;
	});
	private readonly groupNames = computed(() => new Set(this.groups().map((g) => g.name)));
	private readonly sections = computed<CategorySection[]>(() => {
		if (this.viewMode() === 'target') {
			return buildTargetSections(
				this.rules(),
				this.groups().map((g) => g.name),
			);
		}
		return buildCategorySections(this.rules(), this.extraCategories());
	});

	testMatchRules = computed<MatchableRule[]>(() =>
		sortRules(this.rules())
			.filter((r) => r.enabled)
			.map((r) => ({
				type: String(r.type),
				payload: r.payload || '',
				target: r.target,
				enabled: true,
			})),
	);

	sortedRules(): Rule[] {
		return this.sorted();
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

	orphanRuleCount(): number {
		return this.sortedRules().filter((r) => this.isOrphanTarget(r.target)).length;
	}

	ruleSections(): CategorySection[] {
		return this.sections();
	}

	setViewMode(mode: 'category' | 'target'): void {
		if (this.viewMode() === mode) return;
		this.viewMode.set(mode);
	}

	rememberCategory(name: string): void {
		const n = categoryToRemember(name, this.rules(), this.extraCategories());
		if (!n) return;
		this.extraCategories.update((xs) => [...xs, n]);
	}

	private expandedKeys(): Set<string> {
		return this.viewMode() === 'target' ? this.expandedTargetKeys() : this.expandedCategoryKeys();
	}

	isSectionCollapsed(key: string): boolean {
		return !this.expandedKeys().has(key);
	}

	toggleSection(key: string): void {
		const cur = this.expandedKeys();
		const next = new Set(cur);
		if (next.has(key)) next.delete(key);
		else next.add(key);
		if (this.viewMode() === 'target') this.expandedTargetKeys.set(next);
		else this.expandedCategoryKeys.set(next);
	}

	reload(force = false): void {
		this.isLoading = true;
		this.svc
			.listRules(force)
			.pipe(takeUntil(this.$destroy))
			.subscribe({
				next: (r) => {
					const list = r.items || [];
					this.rules.set(list);
					this.isLoading = false;
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
					this.isLoading = false;
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
		this.batchBusy.set(true);
		this.svc
			.batchUpdateTarget(ids, target)
			.pipe(takeUntil(this.$destroy))
			.subscribe({
				next: (res) => {
					this.batchBusy.set(false);
					void this.dialog.success(`已更新 ${res.updated} 条出口（草稿，需发布后生效）`);
					this.clearSelection();
					this.reload();
					this.draftStore.refresh();
				},
				error: (e: Error) => {
					this.batchBusy.set(false);
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
			this.reload();
			this.draftStore.refresh();
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
			this.reload();
			this.draftStore.refresh();
		});
	}

	openPublishModal(): void {
		const ref = this.dialogOpen.openForm(PublishFormComponent, null, {
			width: CM_DIALOG_WIDTH.form,
		});
		ref.afterClosed().subscribe((ok) => {
			if (ok) this.draftStore.refresh();
		});
	}

	openTestModal(): void {
		const data: RuleMatchDialogData = {
			title: '测试规则匹配',
			subtitle:
				'按当前草稿从上到下模拟匹配（未发布也按草稿）。支持域名；IP 仅 IP-CIDR；GEOSITE/GEOIP 浏览器侧会跳过。',
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
		this.svc
			.updateRule(rule.id, {
				type: String(rule.type),
				payload: rule.payload,
				target: rule.target,
				enabled: !rule.enabled,
				note: rule.note,
				category: rule.category,
				sortOrder: rule.sortOrder,
			})
			.pipe(takeUntil(this.$destroy))
			.subscribe({
				next: () => {
					this.reload();
					this.draftStore.refresh();
				},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}

	async removeRule(rule: Rule): Promise<void> {
		if (this.isSystemRule(rule)) {
			void this.dialog.error('系统规则不可删除（广告 / 国内 / 兜底）');
			return;
		}
		const ok = await this.dialog.confirm('确认删除该规则？', '删除确认', '删除');
		if (!ok) return;
		this.svc
			.deleteRule(rule.id)
			.pipe(takeUntil(this.$destroy))
			.subscribe({
				next: () => {
					void this.dialog.success('规则已删除');
					this.reload();
					this.draftStore.refresh();
				},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}
}

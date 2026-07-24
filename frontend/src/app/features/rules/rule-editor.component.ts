import { Component, HostListener, OnInit, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import {
	RULE_TYPE_OPTIONS,
	ProxyGroup,
	ReleasePreview,
	Rule,
	RuleType,
	enumText,
} from '../../common/types';
import { MatchableRule } from '../../common/rule-match';
import { RuleMatchDialogComponent } from '../../common/rule-match-dialog/rule-match-dialog.component';
import { DialogService } from '../../common/dialog/dialog.service';
import { FieldTipComponent } from '../../common/field-tip/field-tip.component';
import { localizeBuildError } from '../../common/format';
import { ReleaseService } from '../releases/release.service';
import { RuleService } from './rule.service';
import { BatchImportModalComponent } from './batch-import-modal.component';
import { RuleFormModalComponent } from './rule-form-modal.component';
import {
	CATEGORY_NEW_VALUE,
	buildCategorySections,
	categoryToRemember,
	defaultRuleTarget,
	isSystemRule,
	sortRules,
} from './rule-ui';

@Component({
	selector: 'app-rule-editor',
	standalone: true,
	imports: [
		FormsModule,
		FieldTipComponent,
		RuleMatchDialogComponent,
		RouterLink,
		RuleFormModalComponent,
		BatchImportModalComponent,
	],
	templateUrl: './rule-editor.component.html',
})
export class RuleEditorComponent implements OnInit {
	private readonly svc = inject(RuleService);
	private readonly releaseSvc = inject(ReleaseService);
	private readonly dialog = inject(DialogService);

	rules = signal<Rule[]>([]);
	groups = signal<ProxyGroup[]>([]);

	// rule modal
	showRuleModal = signal(false);
	editingRule: Rule | null = null;
	createDefaultCategory = '';

	// batch import modal
	showBatchImportModal = signal(false);

	/** 用户新建、尚无规则的空分类（仅面板展示） */
	extraCategories = signal<string[]>([]);
	// 新增分类小弹窗
	showNewCategoryModal = signal(false);
	newCategoryName = '';

	readonly categoryNewValue = CATEGORY_NEW_VALUE;

	// publish modal
	showPublishModal = signal(false);
	publishNote = '';
	publishing = signal(false);
	preview = signal<ReleasePreview | null>(null);
	draftDirty = signal(false);
	draftStatusNote = signal('');

	// match test modal
	showTestModal = signal(false);
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

	/** 按业务分类折叠；空 = 全部展开首次加载时 */
	collapsedCategories = signal<Set<string>>(new Set());

	readonly tip = {
		publishNote: '可选。只写在发布历史里，方便以后回看这次改了什么。',
		newCategory: '新建一个空的业务分类分组；随后可在该分组下添加规则。分类名仅面板展示。',
	};

	ngOnInit(): void {
		this.reload();
		this.refreshDraftStatus();
	}

	refreshDraftStatus(): void {
		this.releaseSvc.draftStatus().subscribe({
			next: (s) => {
				this.draftDirty.set(!!s.dirty);
				if (s.buildError) {
					this.draftStatusNote.set(`草稿暂无法生成：${localizeBuildError(s.buildError)}`);
				} else if (!s.hasPublished) {
					this.draftStatusNote.set('尚未发布过配置，订阅链接在发布后才会有内容');
				} else if (s.dirty) {
					this.draftStatusNote.set(`有未发布更改（当前生效 v${s.publishedVersion || '?'}）`);
				} else {
					this.draftStatusNote.set(`已与 v${s.publishedVersion || '?'} 一致`);
				}
			},
			error: () => {
				this.draftDirty.set(false);
				this.draftStatusNote.set('');
			},
		});
	}

	@HostListener('document:keydown.escape')
	onEsc(): void {
		if (this.showPublishModal()) {
			this.closePublishModal();
			return;
		}
		if (this.showTestModal()) {
			this.closeTestModal();
			return;
		}
		if (this.showNewCategoryModal()) {
			this.closeNewCategoryModal();
			return;
		}
		if (this.showBatchImportModal()) {
			this.closeBatchImportModal();
			return;
		}
		if (this.showRuleModal()) {
			this.closeRuleModal();
		}
	}

	ruleTypeText(v: string | RuleType): string {
		return enumText(RULE_TYPE_OPTIONS, String(v));
	}

	/** 全局匹配顺序（与 Clash 一致） */
	sortedRules(): Rule[] {
		return sortRules(this.rules());
	}

	/** 界面显示用：全局第几条（1-based） */
	globalOrderNo(rule: Rule): number {
		const idx = this.sortedRules().findIndex((r) => r.id === rule.id);
		return idx >= 0 ? idx + 1 : 0;
	}

	/** 目标策略组已不存在 */
	isOrphanTarget(target: string): boolean {
		const names = new Set(this.groups().map((g) => g.name));
		return !!target && !names.has(target);
	}

	isSystemRule(rule: { type?: string; payload?: string } | null | undefined): boolean {
		return isSystemRule(rule);
	}

	orphanRuleCount(): number {
		return this.sortedRules().filter((r) => this.isOrphanTarget(r.target)).length;
	}

	categorySections() {
		return buildCategorySections(this.rules(), this.extraCategories());
	}

	rememberCategory(name: string): void {
		const n = categoryToRemember(name, this.rules(), this.extraCategories());
		if (!n) return;
		this.extraCategories.update((xs) => [...xs, n]);
	}

	isCategoryCollapsed(key: string): boolean {
		return this.collapsedCategories().has(key);
	}

	toggleCategory(key: string): void {
		const next = new Set(this.collapsedCategories());
		if (next.has(key)) next.delete(key);
		else next.add(key);
		this.collapsedCategories.set(next);
	}

	targetText(v: string): string {
		return v;
	}

	reload(): void {
		this.svc.listRules().subscribe({
			next: (r) => this.rules.set(r.items || []),
			error: (e: Error) => void this.dialog.error(e.message),
		});
		this.svc.listGroups().subscribe({
			next: (r) => this.groups.set(r.items || []),
			error: (e: Error) => void this.dialog.error(e.message),
		});
	}

	defaultTarget(): string {
		return defaultRuleTarget(this.groups().map((g) => g.name));
	}

	/** 添加规则；从某分类区块进入时默认带上该分类 */
	openCreateRule(defaultCategory?: string): void {
		this.editingRule = null;
		this.createDefaultCategory = defaultCategory ?? '';
		this.showRuleModal.set(true);
	}

	openNewCategoryModal(): void {
		this.newCategoryName = '';
		this.showNewCategoryModal.set(true);
	}

	closeNewCategoryModal(): void {
		this.showNewCategoryModal.set(false);
		this.newCategoryName = '';
	}

	submitNewCategory(): void {
		const name = this.newCategoryName.trim();
		if (!name) {
			void this.dialog.error('请填写分类名称');
			return;
		}
		if (name === this.categoryNewValue) {
			void this.dialog.error('分类名称不可用');
			return;
		}
		this.rememberCategory(name);
		// 展开该分类
		const collapsed = new Set(this.collapsedCategories());
		collapsed.delete(name);
		this.collapsedCategories.set(collapsed);
		this.closeNewCategoryModal();
		// 直接打开添加规则，分类默认新名字
		this.openCreateRule(name);
	}

	openBatchImport(): void {
		this.showBatchImportModal.set(true);
	}

	closeBatchImportModal(): void {
		this.showBatchImportModal.set(false);
	}

	onBatchImported(category: string | null): void {
		if (category) this.rememberCategory(category);
		this.closeBatchImportModal();
		this.reload();
		this.refreshDraftStatus();
	}

	openEditRule(rule: Rule): void {
		this.editingRule = rule;
		this.createDefaultCategory = '';
		this.showRuleModal.set(true);
	}

	closeRuleModal(): void {
		this.showRuleModal.set(false);
		this.editingRule = null;
		this.createDefaultCategory = '';
	}

	onRuleSaved(category: string | null): void {
		if (category) this.rememberCategory(category);
		this.closeRuleModal();
		this.reload();
		this.refreshDraftStatus();
	}

	openPublishModal(): void {
		this.publishNote = '';
		this.preview.set(null);
		this.showPublishModal.set(true);
	}

	closePublishModal(): void {
		if (this.publishing()) return;
		this.showPublishModal.set(false);
	}

	openTestModal(): void {
		this.showTestModal.set(true);
	}

	closeTestModal(): void {
		this.showTestModal.set(false);
	}

	matchTypeText = (type: string): string => this.ruleTypeText(type);
	matchTargetText = (target: string): string => this.targetText(target);

	canLocateMatchRule = (_rule: MatchableRule): boolean => false;

	onMatchEditRule(rule: MatchableRule): void {
		const r = this.ruleFromMatchable(rule);
		if (!r) return;
		this.closeTestModal();
		this.openEditRule(r);
	}

	onMatchLocateGroup(_rule: MatchableRule): void {
		// 规则页按业务分类展示，不再定位到策略组
		this.closeTestModal();
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
			.subscribe({
				next: () => {
					this.reload();
					this.refreshDraftStatus();
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
		this.svc.deleteRule(rule.id).subscribe({
			next: () => {
				void this.dialog.success('规则已删除');
				this.reload();
				this.refreshDraftStatus();
			},
			error: (e: Error) => void this.dialog.error(e.message),
		});
	}

	/** 按全局匹配顺序上移/下移（与 Clash 规则顺序一致；系统规则不可调） */
	move(rule: Rule, dir: -1 | 1): void {
		if (this.isSystemRule(rule)) {
			void this.dialog.error('系统规则顺序固定，不可调整');
			return;
		}
		const list = sortRules(this.rules());
		const idx = list.findIndex((r) => r.id === rule.id);
		const j = idx + dir;
		if (idx < 0 || j < 0 || j >= list.length) return;
		// 不允许与系统规则互换位置（后端也会钉死，这里直接拦）
		if (this.isSystemRule(list[j])) return;
		[list[idx], list[j]] = [list[j], list[idx]];
		this.svc.reorder(list.map((r) => r.id)).subscribe({
			next: () => {
				this.reload();
				this.refreshDraftStatus();
			},
			error: (e: Error) => void this.dialog.error(e.message),
		});
	}

	doPreview(): void {
		this.publishing.set(true);
		this.releaseSvc.preview().subscribe({
			next: (p) => {
				this.preview.set(p);
				this.publishing.set(false);
			},
			error: (e: Error) => {
				this.publishing.set(false);
				void this.dialog.error(e.message);
			},
		});
	}

	async publish(): Promise<void> {
		const ok = await this.dialog.confirm(
			'确认发布当前草稿配置？\n发布后订阅链接将立即使用新配置。',
			'发布确认',
			'发布',
		);
		if (!ok) return;
		this.publishing.set(true);
		this.releaseSvc.publish(this.publishNote).subscribe({
			next: (res) => {
				this.preview.set(res.preview);
				this.publishNote = '';
				this.publishing.set(false);
				void this.dialog.success(`已发布 v${res.release.version}`);
				this.showPublishModal.set(false);
				this.refreshDraftStatus();
			},
			error: (e: Error) => {
				this.publishing.set(false);
				void this.dialog.error(e.message);
			},
		});
	}
}

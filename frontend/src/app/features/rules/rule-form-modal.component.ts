import { Component, EventEmitter, Input, OnChanges, Output, SimpleChanges, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import {
	RULE_TYPE_OPTIONS,
	ProxyGroup,
	Rule,
	RuleType,
} from '../../common/types';
import { DialogService } from '../../common/dialog/dialog.service';
import { FieldTipComponent } from '../../common/field-tip/field-tip.component';
import { RuleService } from './rule.service';
import {
	CATEGORY_NEW_VALUE,
	buildCategoryOptions,
	isMatchType,
	isSystemRule,
	orphanTargetValue,
	payloadLabel,
	payloadPlaceholder,
	payloadTip,
	resolveSelectedCategory,
} from './rule-ui';

@Component({
	selector: 'app-rule-form-modal',
	standalone: true,
	imports: [FormsModule, FieldTipComponent, RouterLink],
	templateUrl: './rule-form-modal.component.html',
})
export class RuleFormModalComponent implements OnChanges {
	private readonly svc = inject(RuleService);
	private readonly dialog = inject(DialogService);

	@Input({ required: true }) open = false;
	/** null = 新建 */
	@Input() editingRule: Rule | null = null;
	@Input() groups: ProxyGroup[] = [];
	@Input() rules: Rule[] = [];
	@Input() extraCategories: string[] = [];
	/** 新建时默认分类 */
	@Input() defaultCategory = '';
	@Input() defaultTarget = '';

	@Output() closed = new EventEmitter<void>();
	@Output() saved = new EventEmitter<string | null>();

	ruleType: RuleType = RuleType.DOMAIN_SUFFIX;
	rulePayload = '';
	ruleTarget = '';
	ruleEnabled = true;
	ruleNote = '';
	ruleCategory = '';
	ruleCategoryCustom = '';
	ruleTypes = RULE_TYPE_OPTIONS;
	readonly categoryNewValue = CATEGORY_NEW_VALUE;

	readonly tip = {
		ruleType: '用什么方式匹配流量：域名、域名后缀、关键词、国家代码、IP 段等。',
		ruleTarget: '命中后走哪个策略组。\n只从策略组列表选择；策略组请到「策略组」页维护。',
		ruleNote: '仅后台备注，方便自己辨认，不会写入 Clash 配置。',
		ruleCategory:
			'业务分类仅用于面板分组浏览，不写入 Clash。\n可从已有分类选择，或选「＋ 新建分类…」输入新名字。\n「系统分类」= 广告 / 国内 GEOIP / 兜底 MATCH（顺序固定）。',
		ruleEnabled: '关闭后此规则不参与匹配，但仍保留在列表中。',
	};

	ngOnChanges(changes: SimpleChanges): void {
		if (changes['open'] && this.open) {
			this.hydrateFromInputs();
		}
	}

	private hydrateFromInputs(): void {
		const rule = this.editingRule;
		if (rule) {
			this.ruleType = rule.type as RuleType;
			this.rulePayload = rule.payload;
			this.ruleTarget = rule.target;
			this.ruleEnabled = rule.enabled;
			this.ruleNote = rule.note || '';
			this.ruleCategory = rule.category || '';
			this.ruleCategoryCustom = '';
			return;
		}
		this.ruleType = RuleType.DOMAIN_SUFFIX;
		this.rulePayload = '';
		this.ruleTarget = this.defaultTarget;
		this.ruleEnabled = true;
		this.ruleNote = '';
		this.ruleCategory = this.defaultCategory ?? '';
		this.ruleCategoryCustom = '';
	}

	get editingRuleId(): number | null {
		return this.editingRule?.id ?? null;
	}

	get systemLocked(): boolean {
		return this.editingRuleId != null && isSystemRule({ type: this.ruleType, payload: this.rulePayload });
	}

	categorySelectOptions() {
		return buildCategoryOptions(this.rules, this.extraCategories, {
			current: this.ruleCategory,
			allowNew: true,
			newValue: this.categoryNewValue,
		});
	}

	payloadLabel(): string {
		return payloadLabel(this.ruleType);
	}

	payloadTip(): string {
		return payloadTip(this.ruleType);
	}

	payloadPlaceholder(): string {
		return payloadPlaceholder(this.ruleType);
	}

	isMatchType(type: string = this.ruleType): boolean {
		return isMatchType(type);
	}

	orphanTarget(): string | null {
		return orphanTargetValue(
			this.ruleTarget,
			this.groups.map((g) => g.name),
		);
	}

	onRuleTypeChange(): void {
		if (this.isMatchType()) {
			this.rulePayload = '';
		}
	}

	close(): void {
		this.closed.emit();
	}

	saveRule(): void {
		if (!this.isMatchType() && !this.rulePayload.trim()) {
			void this.dialog.error(`请填写「${this.payloadLabel()}」`);
			return;
		}
		if (!this.ruleTarget.trim()) {
			void this.dialog.error('请选择目标出口');
			return;
		}
		const category = resolveSelectedCategory(
			this.ruleCategory,
			this.ruleCategoryCustom,
			this.categoryNewValue,
		);
		if (this.ruleCategory === this.categoryNewValue && !category) {
			void this.dialog.error('请填写新建分类名称');
			return;
		}
		const body = {
			type: this.ruleType,
			payload: this.isMatchType() ? '' : this.rulePayload.trim(),
			target: this.ruleTarget.trim(),
			enabled: this.ruleEnabled,
			note: this.ruleNote,
			category,
		};
		const req =
			this.editingRuleId == null
				? this.svc.createRule(body)
				: this.svc.updateRule(this.editingRuleId, body);
		req.subscribe({
			next: () => {
				void this.dialog.success('规则已保存（草稿，需发布后生效）');
				this.saved.emit(category || null);
			},
			error: (e: Error) => void this.dialog.error(e.message),
		});
	}
}

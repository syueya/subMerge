import { Component, inject } from '@angular/core';
import { FormBuilder, Validators } from '@angular/forms';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import {
	RULE_TYPE_OPTIONS,
	ProxyGroup,
	Rule,
	RuleType,
} from '@data-struct';
import { DialogService } from '@common/services/dialog.service';
import { RuleService } from '../services/rule.service';
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
} from '../services/rule-ui';
import { CmParentFormComponent } from '@common/parents/parent-form/parent-form.component';
import { finalize, takeUntil } from 'rxjs';

export interface RuleFormDialogData {
	rule: Rule | null;
	groups: ProxyGroup[];
	rules: Rule[];
	extraCategories: string[];
	defaultCategory?: string;
	defaultTarget?: string;
}

@Component({
	selector: 'app-rule-form',
	templateUrl: './rule-form.component.html',
	standalone: false,
})
export class RuleFormComponent extends CmParentFormComponent {
	dialogRef = inject<MatDialogRef<RuleFormComponent, string | null | false>>(MatDialogRef);
	data = inject<RuleFormDialogData>(MAT_DIALOG_DATA);
	private fb = inject(FormBuilder);
	private svc = inject(RuleService);
	private dialog = inject(DialogService);

	isUpdate: boolean;
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

	constructor() {
		super();
		const rule = this.data.rule;
		this.isUpdate = !!rule;
		const type = (rule?.type as RuleType) || RuleType.DOMAIN_SUFFIX;
		this.editForm = this.fb.group({
			type: [{ value: type, disabled: false }, [Validators.required]],
			payload: [rule?.payload || ''],
			target: [rule?.target || this.data.defaultTarget || '', [Validators.required]],
			enabled: [rule?.enabled ?? true],
			note: [rule?.note || ''],
			category: [rule?.category || this.data.defaultCategory || ''],
			categoryCustom: [''],
		});
		if (this.systemLocked) {
			this.editForm.get('type')?.disable({ emitEvent: false });
			this.editForm.get('payload')?.disable({ emitEvent: false });
			this.editForm.get('category')?.disable({ emitEvent: false });
		}
	}

	get systemLocked(): boolean {
		const rule = this.data.rule;
		if (!rule) return false;
		const type = this.editForm?.get('type')?.value || rule.type;
		const payload = this.editForm?.get('payload')?.value ?? rule.payload;
		return isSystemRule({ type, payload });
	}

	categorySelectOptions() {
		return buildCategoryOptions(this.data.rules, this.data.extraCategories, {
			current: this.editForm.get('category')?.value,
			allowNew: true,
			newValue: this.categoryNewValue,
		});
	}

	payloadLabelText(): string {
		return payloadLabel(this.editForm.get('type')?.value);
	}

	payloadTipText(): string {
		return payloadTip(this.editForm.get('type')?.value);
	}

	payloadPlaceholderText(): string {
		return payloadPlaceholder(this.editForm.get('type')?.value);
	}

	isMatch(): boolean {
		return isMatchType(this.editForm.get('type')?.value);
	}

	orphanTarget(): string | null {
		return orphanTargetValue(
			this.editForm.get('target')?.value,
			(this.data.groups || []).map((g) => g.name),
		);
	}

	onRuleTypeChange(): void {
		if (this.isMatch()) {
			this.editForm.patchValue({ payload: '' });
		}
	}

	submit(): void {
		if (this.isSubmitting) return;
		const raw = this.editForm.getRawValue();
		if (!this.isMatch() && !String(raw.payload || '').trim()) {
			void this.dialog.error(`请填写「${this.payloadLabelText()}」`);
			return;
		}
		if (!String(raw.target || '').trim()) {
			void this.dialog.error('请选择目标出口');
			return;
		}
		const category = resolveSelectedCategory(
			raw.category,
			raw.categoryCustom,
			this.categoryNewValue,
		);
		if (raw.category === this.categoryNewValue && !category) {
			void this.dialog.error('请填写新建分类名称');
			return;
		}
		const body = {
			type: raw.type,
			payload: this.isMatch() ? '' : String(raw.payload || '').trim(),
			target: String(raw.target || '').trim(),
			enabled: !!raw.enabled,
			note: raw.note || '',
			category,
		};
		this.isSubmitting = true;
		const req =
			this.data.rule == null
				? this.svc.createRule(body)
				: this.svc.updateRule(this.data.rule.id, body);
		req
			.pipe(
				takeUntil(this.$destroy),
				finalize(() => (this.isSubmitting = false)),
			)
			.subscribe({
				next: () => {
					void this.dialog.success('规则已保存（草稿，需发布后生效）');
					this.dialogRef.close(category || null);
				},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}
}

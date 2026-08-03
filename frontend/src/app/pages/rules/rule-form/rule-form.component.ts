import { Component, inject } from '@angular/core';
import { FormBuilder, Validators } from '@angular/forms';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import {
	RULE_TYPE_OPTIONS,
	RuleFormDialogData,
	RuleType,
} from '@data-struct';
import { DialogService } from '@common/services/dialog.service';
import { RuleService } from '../services/rule.service';
import {
	buildCategoryOptions,
	isMatchType,
	isSystemRule,
	orphanTargetValue,
	payloadLabel,
	payloadPlaceholder,
	payloadTip,
} from '../services/rule-ui';
import { CmParentFormComponent } from '@common/parents/parent-form/parent-form.component';
import { finalize, takeUntil } from 'rxjs';

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

/** 已有分类；新分类由输入框直接输入 */
		categorySelectOptions() {
			return buildCategoryOptions(this.data.rules, this.data.extraCategories, {
				current: this.editForm.get('category')?.value,
				allowNew: false,
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
		const category = String(raw.category || '').trim();
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

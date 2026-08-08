import { Component, inject } from '@angular/core';
import { FormBuilder, Validators } from '@angular/forms';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { CmParentFormComponent } from '@common/parents/parent-form/parent-form.component';
import { DialogService } from '@common/services/dialog.service';
import {
	BatchImportDialogData,
	RULE_TYPE_OPTIONS,
	RuleType,
} from '@data-struct';
import { finalize, takeUntil } from 'rxjs';

import { buildCategoryOptions } from '../services/rule-ui';
import { RuleService } from '../services/rule.service';


@Component({
	selector: 'app-batch-import',
	templateUrl: './batch-import.component.html',
	standalone: false,
})
export class BatchImportComponent extends CmParentFormComponent {
	dialogRef = inject<MatDialogRef<BatchImportComponent, string | null | false>>(MatDialogRef);
	data = inject<BatchImportDialogData>(MAT_DIALOG_DATA);
	private fb = inject(FormBuilder);
	private svc = inject(RuleService);
	private dialog = inject(DialogService);

ruleTypes = RULE_TYPE_OPTIONS;

	constructor() {
		super();
		this.editForm = this.fb.group({
			text: ['', [Validators.required]],
			defaultType: [RuleType.DOMAIN as RuleType, [Validators.required]],
			defaultTarget: [this.data.defaultTarget || '直连', [Validators.required]],
			defaultNote: [''],
			defaultCategory: [''],
			enabled: [true],
		});
	}

/** 已有分类；新分类由输入框直接输入 */
		batchCategorySelectOptions() {
			return buildCategoryOptions(this.data.rules, this.data.extraCategories, {
				allowNew: false,
			});
		}

	submit(): void {
		if (this.isSubmitting) return;
		this.editForm.markAllAsTouched();
		const raw = this.editForm.getRawValue();
		const text = String(raw.text || '').trim();
		if (!text) {
			void this.dialog.error('请粘贴要导入的规则（一行一条）');
			return;
		}
		if (!String(raw.defaultTarget || '').trim()) {
			void this.dialog.error('请选择默认出口（仅写域名时会用到）');
			return;
		}
		const defaultCategory = String(raw.defaultCategory || '').trim();
		this.isSubmitting = true;
		this.svc
			.batchImportRules({
				text,
				defaultType: raw.defaultType,
				defaultTarget: String(raw.defaultTarget).trim(),
				defaultNote: String(raw.defaultNote || '').trim(),
				defaultCategory,
				enabled: !!raw.enabled,
			})
			.pipe(
				takeUntil(this.$destroy),
				finalize(() => (this.isSubmitting = false)),
			)
			.subscribe({
				next: (res) => {
					const parts = [`已导入 ${res.created} 条`];
					if (res.skipped) parts.push(`跳过 ${res.skipped} 条`);
					if (res.errors?.length) parts.push(`提示 ${res.errors.length} 条`);
					void this.dialog.success(`${parts.join('，')  }（草稿，需发布后生效）`);
					if (res.errors?.length) {
						console.warn('batch import notes', res.errors);
					}
					this.dialogRef.close(defaultCategory || null);
				},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}
}

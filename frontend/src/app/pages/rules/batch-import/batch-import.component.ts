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
	resolveSelectedCategory,
} from '../services/rule-ui';
import { CmParentFormComponent } from '@common/parents/parent-form/parent-form.component';
import { finalize, takeUntil } from 'rxjs';

export interface BatchImportDialogData {
	groups: ProxyGroup[];
	rules: Rule[];
	extraCategories: string[];
	defaultTarget?: string;
}

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
	readonly categoryNewValue = CATEGORY_NEW_VALUE;
	readonly tipBatchImport =
		'一行一条，空行与 # 注释忽略。\n完整：分类,类型,匹配内容,出口[,备注]\n仅域名：example.com（用下方默认类型/出口/备注/分类）\n示例：\n个人,DOMAIN-SUFFIX,xiaxiazi.ccwu.cc,直连,域名\n海外AI,DOMAIN-SUFFIX,openai.com,美国US,AI-OpenAI\ngpt-api.xxww.online\n新规则会插在国内 GEOIP / 兜底 MATCH 之前。';

	constructor() {
		super();
		this.editForm = this.fb.group({
			text: ['', [Validators.required]],
			defaultType: [RuleType.DOMAIN as RuleType, [Validators.required]],
			defaultTarget: [this.data.defaultTarget || '直连', [Validators.required]],
			defaultNote: [''],
			defaultCategory: [''],
			defaultCategoryCustom: [''],
			enabled: [true],
		});
	}

	batchCategorySelectOptions() {
		return buildCategoryOptions(this.data.rules, this.data.extraCategories, {
			allowNew: true,
			newValue: this.categoryNewValue,
		});
	}

	submit(): void {
		if (this.isSubmitting) return;
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
		const defaultCategory = resolveSelectedCategory(
			raw.defaultCategory,
			raw.defaultCategoryCustom,
			this.categoryNewValue,
		);
		if (raw.defaultCategory === this.categoryNewValue && !defaultCategory) {
			void this.dialog.error('请填写新建分类名称');
			return;
		}
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
					void this.dialog.success(parts.join('，') + '（草稿，需发布后生效）');
					if (res.errors?.length) {
						console.warn('batch import notes', res.errors);
					}
					this.dialogRef.close(defaultCategory || null);
				},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}
}

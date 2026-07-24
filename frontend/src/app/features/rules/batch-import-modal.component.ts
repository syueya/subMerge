import { Component, EventEmitter, Input, OnChanges, Output, SimpleChanges, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
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
	resolveSelectedCategory,
} from './rule-ui';

@Component({
	selector: 'app-batch-import-modal',
	standalone: true,
	imports: [FormsModule, FieldTipComponent],
	templateUrl: './batch-import-modal.component.html',
})
export class BatchImportModalComponent implements OnChanges {
	private readonly svc = inject(RuleService);
	private readonly dialog = inject(DialogService);

	@Input({ required: true }) open = false;
	@Input() groups: ProxyGroup[] = [];
	@Input() rules: Rule[] = [];
	@Input() extraCategories: string[] = [];
	@Input() defaultTarget = '直连';

	@Output() closed = new EventEmitter<void>();
	@Output() imported = new EventEmitter<string | null>();

	batchImportText = '';
	batchImportDefaultType: RuleType = RuleType.DOMAIN;
	batchImportDefaultTarget = '直连';
	batchImportDefaultNote = '';
	batchImportDefaultCategory = '';
	batchImportDefaultCategoryCustom = '';
	batchImportEnabled = true;
	batchImporting = signal(false);

	ruleTypes = RULE_TYPE_OPTIONS;
	readonly categoryNewValue = CATEGORY_NEW_VALUE;

	readonly tipBatchImport =
		'一行一条，空行与 # 注释忽略。\n完整：分类,类型,匹配内容,出口[,备注]\n仅域名：example.com（用下方默认类型/出口/备注/分类）\n示例：\n个人,DOMAIN-SUFFIX,xiaxiazi.ccwu.cc,直连,域名\n海外AI,DOMAIN-SUFFIX,openai.com,美国US,AI-OpenAI\ngpt-api.xxww.online\n新规则会插在国内 GEOIP / 兜底 MATCH 之前。';

	ngOnChanges(changes: SimpleChanges): void {
		if (changes['open'] && this.open) {
			this.resetForm();
		}
	}

	private resetForm(): void {
		this.batchImportText = '';
		this.batchImportDefaultType = RuleType.DOMAIN;
		this.batchImportDefaultTarget = this.defaultTarget || '直连';
		this.batchImportDefaultNote = '';
		this.batchImportDefaultCategory = '';
		this.batchImportDefaultCategoryCustom = '';
		this.batchImportEnabled = true;
		this.batchImporting.set(false);
	}

	batchCategorySelectOptions() {
		return buildCategoryOptions(this.rules, this.extraCategories, {
			allowNew: true,
			newValue: this.categoryNewValue,
		});
	}

	close(): void {
		if (this.batchImporting()) return;
		this.closed.emit();
	}

	submitBatchImport(): void {
		const text = this.batchImportText.trim();
		if (!text) {
			void this.dialog.error('请粘贴要导入的规则（一行一条）');
			return;
		}
		if (!this.batchImportDefaultTarget.trim()) {
			void this.dialog.error('请选择默认出口（仅写域名时会用到）');
			return;
		}
		const defaultCategory = resolveSelectedCategory(
			this.batchImportDefaultCategory,
			this.batchImportDefaultCategoryCustom,
			this.categoryNewValue,
		);
		if (this.batchImportDefaultCategory === this.categoryNewValue && !defaultCategory) {
			void this.dialog.error('请填写新建分类名称');
			return;
		}
		this.batchImporting.set(true);
		this.svc
			.batchImportRules({
				text,
				defaultType: this.batchImportDefaultType,
				defaultTarget: this.batchImportDefaultTarget.trim(),
				defaultNote: this.batchImportDefaultNote.trim(),
				defaultCategory,
				enabled: this.batchImportEnabled,
			})
			.subscribe({
				next: (res) => {
					this.batchImporting.set(false);
					const parts = [`已导入 ${res.created} 条`];
					if (res.skipped) parts.push(`跳过 ${res.skipped} 条`);
					if (res.errors?.length) parts.push(`提示 ${res.errors.length} 条`);
					void this.dialog.success(parts.join('，') + '（草稿，需发布后生效）');
					if (res.errors?.length) {
						console.warn('batch import notes', res.errors);
					}
					this.imported.emit(defaultCategory || null);
				},
				error: (e: Error) => {
					this.batchImporting.set(false);
					void this.dialog.error(e.message);
				},
			});
	}
}

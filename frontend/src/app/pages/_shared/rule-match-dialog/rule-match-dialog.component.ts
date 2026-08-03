import { Component, inject, signal } from '@angular/core';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import {
	MatchableRule,
	RuleMatchDialogData,
	RuleMatchDialogResult,
	RuleMatchResult,
} from '@data-struct';
import { runRuleMatch } from '@common/util/rule-match';

@Component({
	selector: 'app-rule-match-dialog',
	templateUrl: './rule-match-dialog.component.html',
	standalone: false,
})
export class RuleMatchDialogComponent {
	dialogRef = inject<MatDialogRef<RuleMatchDialogComponent, RuleMatchDialogResult>>(MatDialogRef);
	data = inject<RuleMatchDialogData>(MAT_DIALOG_DATA);

	testInput = '';
	testResult = signal<RuleMatchResult | null>(null);

	readonly testExamples = [
		'chat.openai.com',
		'www.google.com',
		'github.com',
		't.me',
	];

	get title(): string {
		return this.data.title || '测试规则匹配';
	}

	get subtitle(): string {
		return (
			this.data.subtitle ||
			'按规则从上到下模拟匹配。支持域名；IP 仅 IP-CIDR；GEOSITE/GEOIP 浏览器侧会跳过。'
		);
	}

	get loading(): boolean {
		return !!this.data.loading;
	}

	get rules(): MatchableRule[] {
		return this.data.rules || [];
	}

	typeText(type: string): string {
		return (this.data.typeText || ((t) => t))(type);
	}

	targetText(target: string): string {
		return (this.data.targetText || ((t) => t))(target);
	}

	get showEditAction(): boolean {
		return !!this.data.showEditAction;
	}

	get showLocateAction(): boolean {
		return !!this.data.showLocateAction;
	}

	canLocate(rule: MatchableRule): boolean {
		return (this.data.canLocate || (() => false))(rule);
	}

	close(): void {
		this.dialogRef.close(null);
	}

	runTest(): void {
		this.testResult.set(runRuleMatch(this.testInput, this.rules));
	}

	useExample(ex: string): void {
		this.testInput = ex;
		this.runTest();
	}

	onEdit(rule: MatchableRule): void {
		this.dialogRef.close({ action: 'edit', rule });
	}

	onLocate(rule: MatchableRule): void {
		this.dialogRef.close({ action: 'locate', rule });
	}

	kindLabel(kind: RuleMatchResult['kind']): string {
		switch (kind) {
			case 'domain':
				return '域名';
			case 'ipv4':
				return 'IPv4';
			case 'ipv6':
				return 'IPv6';
			case 'empty':
				return '空';
			default:
				return '无效';
		}
	}

	/** 结果徽章文案：具体命中 / MATCH 兜底 / 完全未命中 */
	statusLabel(r: RuleMatchResult): string {
		if (r.matched) return '已命中';
		return '未命中';
	}

	statusBadgeClass(r: RuleMatchResult): string {
		if (r.matched) return 'bg-light-success text-success rounded f-s-12 f-w-600 p-x-8 p-y-2';
		return 'bg-light-warning text-warning rounded f-s-12 f-w-600 p-x-8 p-y-2';
	}

	/** MATCH 兜底：模板用方法访问，避免 strictTemplates 对字段误报 */
	isFallbackMatch(r: RuleMatchResult): boolean {
		return !!r.fallbackMatch;
	}

	/** 有关联规则即可编辑（含 MATCH 兜底） */
	canEdit(r: RuleMatchResult): boolean {
		return this.showEditAction && !!r.rule;
	}

	canShowLocate(r: RuleMatchResult): boolean {
		return this.showLocateAction && !!r.rule && this.canLocate(r.rule!);
	}
}

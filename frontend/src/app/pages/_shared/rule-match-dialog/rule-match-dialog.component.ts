import { Component, inject, signal } from '@angular/core';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import {
	MatchableRule,
	RuleMatchResult,
	runRuleMatch,
} from '@common/util/rule-match';

export interface RuleMatchDialogData {
	title?: string;
	subtitle?: string;
	rules: MatchableRule[];
	loading?: boolean;
	typeText?: (type: string) => string;
	targetText?: (target: string) => string;
	showEditAction?: boolean;
	showLocateAction?: boolean;
	canLocate?: (rule: MatchableRule) => boolean;
}

export type RuleMatchDialogResult =
	| { action: 'edit'; rule: MatchableRule }
	| { action: 'locate'; rule: MatchableRule }
	| null;

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
		'https://www.google.com',
		'www.bilibili.com',
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
}

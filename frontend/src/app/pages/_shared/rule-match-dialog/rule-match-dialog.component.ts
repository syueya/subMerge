import { Component, inject, signal } from '@angular/core';
import { MatChipListboxChange } from '@angular/material/chips';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { ApiService } from '@common/net/api.service';
import {
	MatchableRule,
	RuleMatchDialogData,
	RuleMatchDialogResult,
	RuleMatchResult,
} from '@data-struct';

@Component({
	selector: 'app-rule-match-dialog',
	templateUrl: './rule-match-dialog.component.html',
	standalone: false,
})
export class RuleMatchDialogComponent {
	dialogRef = inject<MatDialogRef<RuleMatchDialogComponent, RuleMatchDialogResult>>(MatDialogRef);
	data = inject<RuleMatchDialogData>(MAT_DIALOG_DATA);
	private readonly api = inject(ApiService);

	testInput = '';
	testResult = signal<RuleMatchResult | null>(null);
	testing = signal(false);

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
			'按规则从上到下由服务端模拟匹配（含 DOMAIN / GEOSITE / GEOIP / IP-CIDR）。使用面板已加载的 Geo 数据，可能与客户端本地 dat 不完全一致。'
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
		const input = this.testInput.trim();
		if (!input || this.testing()) return;

		const needResolve = this.rules.some((r) => {
			if (r.enabled === false) return false;
			return String(r.type || '').toUpperCase() === 'GEOIP';
		});

		this.testing.set(true);
		this.api
			.post<RuleMatchResult>('/rules/match', {
				input,
				rules: this.rules,
				resolve: needResolve,
			})
			.subscribe({
				next: (r) => {
					this.testing.set(false);
					this.testResult.set(this.normalizeResult(r));
				},
				error: (e: Error) => {
					this.testing.set(false);
					this.testResult.set({
						input,
						host: '',
						kind: 'invalid',
						matched: false,
						fallbackMatch: false,
						rule: null,
						skipped: 0,
						note: e?.message || '规则匹配请求失败',
					});
				},
			});
	}

	/** 保证模板所需字段齐全（后端 omitempty 时补默认） */
	private normalizeResult(r: RuleMatchResult): RuleMatchResult {
		return {
			...r,
			matched: !!r.matched,
			fallbackMatch: !!r.fallbackMatch,
			rule: r.rule ?? null,
			skipped: r.skipped ?? 0,
			note: r.note || '',
			host: r.host || '',
			kind: r.kind || 'invalid',
			input: r.input ?? this.testInput,
		};
	}

	get selectedExample(): string | undefined {
		const v = this.testInput.trim();
		return this.testExamples.includes(v) ? v : undefined;
	}

	onExampleChipChange(ev: MatChipListboxChange): void {
		const v = typeof ev.value === 'string' ? ev.value : '';
		if (!v || this.testing()) return;
		this.useExample(v);
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
		if (r.fallbackMatch) return '最终匹配';
		return '未命中';
	}

	statusBadgeClass(r: RuleMatchResult): string {
		if (r.matched) return 'bg-light-success text-success rounded f-s-12 f-w-600 p-x-8 p-y-2';
		if (r.fallbackMatch) return 'bg-light-primary text-primary rounded f-s-12 f-w-600 p-x-8 p-y-2';
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

	geoHitText(r: RuleMatchResult): string {
		const g = r.geoHit;
		if (!g) return '';
		const parts: string[] = [];
		if (g.type && g.value) parts.push(`${g.type} ${g.value}`);
		else if (g.value) parts.push(g.value);
		if (g.cidr) parts.push(g.cidr);
		if (g.ip) parts.push(`IP ${g.ip}`);
		return parts.join(' · ') || g.category;
	}
}

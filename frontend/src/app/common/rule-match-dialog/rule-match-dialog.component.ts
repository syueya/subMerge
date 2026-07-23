import { Component, input, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import {
	MatchableRule,
	RuleMatchResult,
	runRuleMatch,
} from '../rule-match';

@Component({
	selector: 'app-rule-match-dialog',
	standalone: true,
	imports: [FormsModule],
	templateUrl: './rule-match-dialog.component.html',
})
export class RuleMatchDialogComponent {
	/** 是否显示弹窗 */
	open = input(false);
	title = input('测试规则匹配');
	subtitle = input(
		'按规则从上到下模拟匹配。支持域名；IP 仅 IP-CIDR；GEOSITE/GEOIP 浏览器侧会跳过。',
	);
	rules = input<MatchableRule[]>([]);
	loading = input(false);
	/** 类型展示文案 */
	typeText = input<(type: string) => string>((t) => t);
	/** 出口展示文案 */
	targetText = input<(target: string) => string>((t) => t);
	/** 是否显示「编辑此规则」 */
	showEditAction = input(false);
	/** 是否显示「定位到策略组」 */
	showLocateAction = input(false);
	canLocate = input<(rule: MatchableRule) => boolean>(() => false);

	closed = output<void>();
	editRule = output<MatchableRule>();
	locateGroup = output<MatchableRule>();

	testInput = '';
	testResult = signal<RuleMatchResult | null>(null);

	readonly testExamples = [
		'chat.openai.com',
		'https://www.google.com',
		'www.bilibili.com',
		't.me',
	];

	close(): void {
		this.testInput = '';
		this.testResult.set(null);
		this.closed.emit();
	}

	runTest(): void {
		this.testResult.set(runRuleMatch(this.testInput, this.rules()));
	}

	useExample(ex: string): void {
		this.testInput = ex;
		this.runTest();
	}

	onEdit(rule: MatchableRule): void {
		this.editRule.emit(rule);
	}

	onLocate(rule: MatchableRule): void {
		this.locateGroup.emit(rule);
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

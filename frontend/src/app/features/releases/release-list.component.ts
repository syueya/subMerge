import { Component, OnInit, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import {
	RELEASE_STATUS_BADGE,
	RELEASE_STATUS_OPTIONS,
	Release,
	ReleaseDetail,
	ReleaseRuleLine,
	enumBadgeClass,
	enumText,
} from '../../common/types';
import { MatchableRule } from '../../common/rule-match';
	import { RuleMatchDialogComponent } from '../../common/rule-match-dialog/rule-match-dialog.component';
	import { DialogService } from '../../common/dialog/dialog.service';
	import { draftStatusNote } from '../../common/format';
	import { ReleaseService } from './release.service';

@Component({
	selector: 'app-release-list',
	standalone: true,
	imports: [RouterLink, FormsModule, RuleMatchDialogComponent],
	templateUrl: './release-list.component.html',
})
export class ReleaseListComponent implements OnInit {
	private readonly svc = inject(ReleaseService);
	private readonly dialog = inject(DialogService);

	items = signal<Release[]>([]);
	loading = signal(false);
	publishing = signal(false);
	draftDirty = signal(false);
	draftNote = signal('');

	/** 查看历史版本详情 */
	viewing = signal<ReleaseDetail | null>(null);
	viewTab = signal<'rules' | 'yaml'>('rules');
	viewLoading = signal(false);

	/** 某版本规则匹配测试 */
	testOpen = signal(false);
	testLoading = signal(false);
	testVersion = signal<number | null>(null);
	testRules = signal<MatchableRule[]>([]);

	ngOnInit(): void {
		this.reload();
		this.refreshDraft();
	}

	refreshDraft(): void {
		this.svc.draftStatus().subscribe({
			next: (s) => {
				this.draftDirty.set(!!s.dirty);
				this.draftNote.set(draftStatusNote(s));
			},
			error: () => {
				this.draftDirty.set(false);
				this.draftNote.set('');
			},
		});
	}

	async publishNow(): Promise<void> {
		const ok = await this.dialog.confirm(
			'确认发布当前草稿配置？\n发布后「全部源 + 自动」的订阅链接将使用新配置。',
			'发布确认',
			'发布',
		);
		if (!ok) return;
		this.publishing.set(true);
		this.svc.publish('').subscribe({
			next: (res) => {
				this.publishing.set(false);
				void this.dialog.success(`已发布 v${res.release.version}`);
				this.reload();
				this.refreshDraft();
			},
			error: (e: Error) => {
				this.publishing.set(false);
				void this.dialog.error(e.message);
			},
		});
	}

	statusText(v: string): string {
		return enumText(RELEASE_STATUS_OPTIONS, v);
	}

	statusClass(v: string): string {
		return enumBadgeClass(RELEASE_STATUS_BADGE, v);
	}

	reload(): void {
		this.loading.set(true);
		this.svc.list().subscribe({
			next: (r) => {
				this.items.set(r.items || []);
				this.loading.set(false);
			},
			error: (e: Error) => {
				this.loading.set(false);
				void this.dialog.error(e.message);
			},
		});
	}

	async rollback(item: Release): Promise<void> {
		const ok = await this.dialog.confirm(
			`回滚到 v${item.version}？\n将基于该版本生成新的发布记录。`,
			'回滚确认',
			'回滚',
		);
		if (!ok) return;
		this.svc.rollback(item.id).subscribe({
			next: (r) => {
				void this.dialog.success(`已回滚，新版本 v${r.version}`);
				this.reload();
				this.refreshDraft();
			},
			error: (e: Error) => void this.dialog.error(e.message),
		});
	}

	openView(item: Release): void {
		this.viewLoading.set(true);
		this.viewTab.set('rules');
		this.viewing.set(null);
		this.svc.get(item.id).subscribe({
			next: (d) => {
				this.viewing.set(d);
				this.viewLoading.set(false);
			},
			error: (e: Error) => {
				this.viewLoading.set(false);
				void this.dialog.error(e.message);
			},
		});
	}

	closeView(): void {
		this.viewing.set(null);
		this.viewLoading.set(false);
	}

	openTest(item: Release): void {
		this.testOpen.set(true);
		this.testLoading.set(true);
		this.testRules.set([]);
		this.testVersion.set(item.version);
		this.svc.get(item.id).subscribe({
			next: (d) => {
				this.testRules.set(this.toMatchRules(d.rules || []));
				this.testVersion.set(d.version);
				this.testLoading.set(false);
			},
			error: (e: Error) => {
				this.testLoading.set(false);
				this.testOpen.set(false);
				void this.dialog.error(e.message || '加载版本规则失败');
			},
		});
	}

	closeTest(): void {
		this.testOpen.set(false);
		this.testRules.set([]);
		this.testVersion.set(null);
	}

	testTitle(): string {
		const v = this.testVersion();
		return v != null ? `测试 v${v} 规则` : '测试规则匹配';
	}

	testSubtitle(): string {
		const v = this.testVersion();
		const base =
			'按该版本已发布规则从上到下模拟匹配。支持域名；IP 仅 IP-CIDR；GEOSITE/GEOIP 浏览器侧会跳过。';
		return v != null ? `v${v} · ${base}` : base;
	}

	matchTypeText = (type: string): string => type;
	matchTargetText = (target: string): string => target;

	ruleLineText(r: ReleaseRuleLine): string {
		return r.raw || [r.type, r.payload, r.target].filter(Boolean).join(',');
	}

	private toMatchRules(rules: ReleaseRuleLine[]): MatchableRule[] {
		return rules.map((r) => ({
			type: r.type,
			payload: r.payload || '',
			target: r.target,
			raw: r.raw,
			enabled: true,
		}));
	}
}

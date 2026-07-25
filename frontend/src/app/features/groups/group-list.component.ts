import { Component, HostListener, OnInit, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import {
	PROXY_GROUP_TYPE_OPTIONS,
	ProxyGroup,
	Rule,
	enumText,
	regionLabel,
} from '../../common/types';
import { DialogService } from '../../common/dialog/dialog.service';
import { FieldTipComponent } from '../../common/field-tip/field-tip.component';
import { draftStatusNote as buildDraftStatusNote } from '../../common/format';
import { ReleaseService } from '../releases/release.service';
import { SourceService } from '../sources/source.service';
import { RuleService } from '../rules/rule.service';

@Component({
	selector: 'app-group-list',
	standalone: true,
	imports: [FormsModule, FieldTipComponent, RouterLink],
	templateUrl: './group-list.component.html',
})
export class GroupListComponent implements OnInit {
	private readonly svc = inject(RuleService);
	private readonly releaseSvc = inject(ReleaseService);
	private readonly sourceSvc = inject(SourceService);
	private readonly dialog = inject(DialogService);

	groups = signal<ProxyGroup[]>([]);
	rules = signal<Rule[]>([]);
	regionCatalog = signal<{ code: string; name: string }[]>([]);
	extraRegionCodes = signal<string[]>([]);
	knownSources = signal<{ id: number; name: string }[]>([]);

	draftDirty = signal(false);
	draftStatusNote = signal('');

	showGroupModal = signal(false);
	editingGroupId: number | null = null;
	groupName = '';
	groupType = 'select';
	groupMembers: string[] = ['ALL'];
	groupCustomMember = '';
	groupEnabled = true;
	groupUrl = 'https://www.gstatic.com/generate_204';
	groupInterval: number | null = 300;
	groupTypes = PROXY_GROUP_TYPE_OPTIONS;
	readonly defaultTestURL = 'https://www.gstatic.com/generate_204';
	readonly defaultTestInterval = 300;

	memberPresets = computed(() => {
		const base: { value: string; text: string }[] = [
			{ value: 'ALL', text: '全部节点 (ALL)' },
			{ value: 'DIRECT', text: '引擎直连 (DIRECT)' },
			{ value: 'REJECT', text: '引擎拒绝 (REJECT)' },
		];
		const sourceItems = this.knownSources().map((s) => ({
			value: `SOURCE:${s.name}`,
			text: `源「${s.name}」全部节点 (SOURCE:${s.name})`,
		}));

		const labels: Record<string, string> = {};
		const codes = new Set<string>();
		for (const r of this.regionCatalog()) {
			const code = String(r.code || '').toUpperCase();
			const name = String(r.name || '').trim();
			if (!code || code === 'UNKNOWN') continue;
			codes.add(code);
			if (name) labels[code] = name;
		}
		for (const c of this.extraRegionCodes()) {
			const code = String(c || '').toUpperCase();
			if (code && code !== 'UNKNOWN') codes.add(code);
		}
		const regionItems = [...codes]
			.sort()
			.map((code) => ({
				value: `REGION:${code}`,
				text: `${regionLabel(code, labels)}节点 (REGION:${code})`,
			}));
		return [...base, ...sourceItems, ...regionItems];
	});

	readonly tip = {
		groupName: '组名会被规则引用。中文也可以，如「直连」「拒绝」；地区组常用 美国US、日本JP。',
		groupType:
			'手动选择：客户端里自己挑节点。\n自动测速(url-test)：客户端定时探测 url，自动选延迟最低节点。\n故障转移：按顺序，挂了切下一个。\n负载均衡：在成员间分摊。\n测速在 Clash 客户端执行，不是本面板。',
		groupProxies:
			'勾选成员。\nALL=全部节点\nSOURCE:源名=该订阅源全部节点\nSOURCE:id:3=按源 ID\nREGION:US=美国前缀节点\nDIRECT/REJECT=引擎关键字\n也可勾选其它策略组，或手动添加具体节点名。\nurl-test 建议只放节点（SOURCE:… / REGION:XX / ALL）。',
		groupUrl: '测速探测地址。客户端用各节点访问此 URL 测延迟。\n常用：https://www.gstatic.com/generate_204',
		groupInterval: '重新测速间隔（秒）。例如 300 = 每 5 分钟测一次。',
		groupEnabled: '关闭后此策略组不写入发布配置。',
	};

	ngOnInit(): void {
		this.reload();
		this.refreshDraftStatus();
	}

	refreshDraftStatus(): void {
		this.releaseSvc.draftStatus().subscribe({
			next: (s) => {
				this.draftDirty.set(!!s.dirty);
				this.draftStatusNote.set(buildDraftStatusNote(s));
			},
			error: () => {
				this.draftDirty.set(false);
				this.draftStatusNote.set('');
			},
		});
	}

	@HostListener('document:keydown.escape')
	onEsc(): void {
		if (this.showGroupModal()) this.closeGroupModal();
	}

	reload(): void {
		this.svc.listGroups().subscribe({
			next: (r) => this.groups.set(r.items || []),
			error: (e: Error) => void this.dialog.error(e.message),
		});
		this.svc.listRules().subscribe({
			next: (r) => this.rules.set(r.items || []),
			error: () => {},
		});
		this.sourceSvc.listRegions().subscribe({
			next: (r) => {
				const items = (r.items || [])
					.map((item) => ({
						code: String(item.code || '').toUpperCase(),
						name: String(item.name || '').trim(),
					}))
					.filter((item) => item.code && item.code !== 'UNKNOWN');
				this.regionCatalog.set(items);
			},
			error: () => {},
		});
		this.sourceSvc.list().subscribe({
			next: (r) => {
				const sources: { id: number; name: string }[] = [];
				const extras = new Set(this.extraRegionCodes());
				const catalogCodes = new Set(this.regionCatalog().map((x) => x.code));
				for (const s of r.items || []) {
					const c = String(s.region || '').toUpperCase();
					if (c && c !== 'UNKNOWN' && !catalogCodes.has(c)) extras.add(c);
					const name = String(s.name || '').trim();
					if (s.enabled && name) sources.push({ id: s.id, name });
				}
				this.extraRegionCodes.set([...extras].sort());
				sources.sort((a, b) => a.name.localeCompare(b.name, 'zh'));
				this.knownSources.set(sources);
			},
			error: () => {},
		});
		this.sourceSvc.listProxies().subscribe({
			next: (r) => {
				const extras = new Set(this.extraRegionCodes());
				const catalogCodes = new Set(this.regionCatalog().map((x) => x.code));
				for (const p of r.items || []) {
					const c = String(p.region || '').toUpperCase();
					if (c && c !== 'UNKNOWN' && !catalogCodes.has(c)) extras.add(c);
				}
				this.extraRegionCodes.set([...extras].sort());
			},
			error: () => {},
		});
	}

	rulesOfGroup(name: string): number {
		return this.rules().filter((r) => r.target === name).length;
	}

	groupTypeText(v: string): string {
		return enumText(PROXY_GROUP_TYPE_OPTIONS, v);
	}

	groupTypeBadgeClass(type: string): string {
		switch (type) {
			case 'select':
				return 'badge-muted';
			case 'url-test':
				return 'badge-ok';
			case 'fallback':
				return 'badge-warn';
			case 'load-balance':
				return 'badge-ok';
			default:
				return 'badge-muted';
		}
	}

	groupTypeTip(type: string): string {
		switch (type) {
			case 'select':
				return '手动选择：在客户端里自己挑节点';
			case 'url-test':
				return '自动测速：客户端定时测延迟，自动选最快节点';
			case 'fallback':
				return '故障转移：按成员顺序，挂了切下一个';
			case 'load-balance':
				return '负载均衡：在成员间分摊流量';
			default:
				return this.groupTypeText(type);
		}
	}

	memberGroupOptions(): ProxyGroup[] {
		const selfId = this.editingGroupId;
		return this.groups().filter((g) => g.id !== selfId);
	}

	customMembers(): string[] {
		const preset = new Set(this.memberPresets().map((p) => p.value));
		const groupNames = new Set(this.memberGroupOptions().map((g) => g.name));
		return this.groupMembers.filter((m) => !preset.has(m) && !groupNames.has(m));
	}

	isMemberSelected(value: string): boolean {
		return this.groupMembers.includes(value);
	}

	toggleMember(value: string): void {
		if (this.isMemberSelected(value)) {
			this.groupMembers = this.groupMembers.filter((m) => m !== value);
		} else {
			this.groupMembers = [...this.groupMembers, value];
		}
	}

	addCustomMember(): void {
		const v = this.groupCustomMember.trim();
		if (!v) return;
		if (!this.isMemberSelected(v)) {
			this.groupMembers = [...this.groupMembers, v];
		}
		this.groupCustomMember = '';
	}

	removeMember(value: string): void {
		this.groupMembers = this.groupMembers.filter((m) => m !== value);
	}

	needsTestParams(type: string = this.groupType): boolean {
		return type === 'url-test' || type === 'fallback';
	}

	onGroupTypeChange(type: string): void {
		this.groupType = type;
		if (this.needsTestParams(type)) {
			if (!this.groupUrl.trim()) this.groupUrl = this.defaultTestURL;
			if (this.groupInterval == null || this.groupInterval < 1) {
				this.groupInterval = this.defaultTestInterval;
			}
		}
	}

	resetGroupForm(): void {
		this.editingGroupId = null;
		this.groupName = '';
		this.groupType = 'select';
		this.groupMembers = ['ALL'];
		this.groupCustomMember = '';
		this.groupEnabled = true;
		this.groupUrl = this.defaultTestURL;
		this.groupInterval = this.defaultTestInterval;
	}

	openCreateGroup(): void {
		this.resetGroupForm();
		this.showGroupModal.set(true);
	}

	openEditGroup(g: ProxyGroup): void {
		this.editingGroupId = g.id;
		this.groupName = g.name;
		this.groupType = String(g.type || 'select');
		this.groupMembers = [...(g.proxies || [])];
		this.groupCustomMember = '';
		this.groupEnabled = g.enabled;
		this.groupUrl = g.url || this.defaultTestURL;
		this.groupInterval = g.interval ?? this.defaultTestInterval;
		this.showGroupModal.set(true);
	}

	closeGroupModal(): void {
		this.showGroupModal.set(false);
		this.resetGroupForm();
	}

	saveGroup(): void {
		if (!this.groupName.trim()) {
			void this.dialog.error('请填写策略组名称');
			return;
		}
		if (this.groupMembers.length === 0) {
			void this.dialog.error('请至少选择一个成员');
			return;
		}
		if (this.needsTestParams()) {
			if (!this.groupUrl.trim()) {
				void this.dialog.error('自动测速/故障转移请填写测速 URL');
				return;
			}
			if (this.groupInterval == null || this.groupInterval < 1) {
				void this.dialog.error('请填写有效的测速间隔（秒）');
				return;
			}
		}
		const body: {
			name: string;
			type: string;
			proxies: string[];
			enabled: boolean;
			url?: string;
			interval?: number;
		} = {
			name: this.groupName.trim(),
			type: this.groupType,
			proxies: [...this.groupMembers],
			enabled: this.groupEnabled,
		};
		if (this.needsTestParams()) {
			body.url = this.groupUrl.trim();
			body.interval = Number(this.groupInterval);
		} else {
			body.url = '';
		}
		const req =
			this.editingGroupId == null
				? this.svc.createGroup(body)
				: this.svc.updateGroup(this.editingGroupId, body);
		req.subscribe({
			next: () => {
				void this.dialog.success(this.editingGroupId == null ? '策略组已创建' : '策略组已更新');
				this.closeGroupModal();
				this.reload();
				this.refreshDraftStatus();
			},
			error: (e: Error) => void this.dialog.error(e.message),
		});
	}

	async removeGroup(g: ProxyGroup): Promise<void> {
		const count = this.rulesOfGroup(g.name);
		let cascade = false;
		if (count > 0) {
			const delRules = await this.dialog.confirm(
				`策略组「${g.name}」被 ${count} 条规则引用。\n\n选「删除」：连同这些规则一起删掉。\n选「取消」后可再决定只删组（规则会变成出口失效）。\n\n是否删除组及其规则？`,
				'删除策略组',
				'删除组+规则',
			);
			if (!delRules) {
				const onlyGroup = await this.dialog.confirm(
					`仅删除策略组「${g.name}」，保留 ${count} 条规则（将显示为出口失效）？`,
					'仅删组',
					'仅删组',
				);
				if (!onlyGroup) return;
				cascade = false;
			} else {
				cascade = true;
			}
		} else {
			const ok = await this.dialog.confirm(`确认删除策略组「${g.name}」？`, '删除确认', '删除');
			if (!ok) return;
		}
		this.svc.deleteGroup(g.id, cascade).subscribe({
			next: () => {
				void this.dialog.success(
					cascade ? `已删除策略组「${g.name}」及其规则` : `已删除策略组「${g.name}」`,
				);
				this.reload();
				this.refreshDraftStatus();
			},
			error: (e: Error) => void this.dialog.error(e.message),
		});
	}
}

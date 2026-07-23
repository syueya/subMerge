import { Component, HostListener, OnInit, computed, inject, signal } from '@angular/core';
	import { FormsModule } from '@angular/forms';
	import {
		 PROXY_GROUP_TYPE_OPTIONS,
		 REGION_OPTIONS,
		 RULE_TYPE_OPTIONS,
		 ProxyGroup,
		 ReleasePreview,
		 Rule,
		 RuleType,
		 enumText,
		 regionLabel,
	} from '../../common/types';
	import { MatchableRule } from '../../common/rule-match';
	import { RuleMatchDialogComponent } from '../../common/rule-match-dialog/rule-match-dialog.component';
	import { DialogService } from '../../common/dialog/dialog.service';
	import { FieldTipComponent } from '../../common/field-tip/field-tip.component';
	import { ReleaseService } from '../releases/release.service';
	import { SourceService } from '../sources/source.service';
	import { RuleService } from './rule.service';
	
	@Component({
	 selector: 'app-rule-editor',
	 standalone: true,
	 imports: [FormsModule, FieldTipComponent, RuleMatchDialogComponent],
	 templateUrl: './rule-editor.component.html',
	})
	export class RuleEditorComponent implements OnInit {
		 private readonly svc = inject(RuleService);
		 private readonly releaseSvc = inject(ReleaseService);
		 private readonly sourceSvc = inject(SourceService);
		 private readonly dialog = inject(DialogService);
	
		 rules = signal<Rule[]>([]);
		 groups = signal<ProxyGroup[]>([]);
		 /** 系统中出现过的地区码（源默认 + 节点） */
		 knownRegions = signal<string[]>([]);
	
	/** 展开的策略组名；未归属用 __orphan__ */
		expandedGroups = signal<Set<string>>(new Set());

// rule modal
	 showRuleModal = signal(false);
	 editingRuleId: number | null = null;
	 ruleType: RuleType = RuleType.DOMAIN_SUFFIX;
	 rulePayload = '';
	 ruleTarget = 'PROXY';
	 ruleEnabled = true;
	 ruleNote = '';
	 ruleTypes = RULE_TYPE_OPTIONS;

	// batch import modal
	showBatchImportModal = signal(false);
	batchImportText = '';
	batchImportDefaultType: RuleType = RuleType.DOMAIN;
	batchImportDefaultTarget = '直连';
	batchImportDefaultNote = '';
	batchImportEnabled = true;
	batchImporting = signal(false);

// publish modal
	 showPublishModal = signal(false);
	 publishNote = '';
	 publishing = signal(false);
	 preview = signal<ReleasePreview | null>(null);
	 /** 相对已发布是否有未发布更改 */
	 draftDirty = signal(false);
	 draftStatusNote = signal('');

 // match test modal（公共组件）
	 showTestModal = signal(false);
	 /** 测试用规则列表（启用 + 排序，与发布一致） */
	 testMatchRules = computed<MatchableRule[]>(() =>
		 [...this.rules()]
			 .filter((r) => r.enabled)
			 .sort((a, b) => a.sortOrder - b.sortOrder || a.id - b.id)
			 .map((r) => ({
				 type: String(r.type),
				 payload: r.payload || '',
				 target: r.target,
				 enabled: true,
			 })),
	 );
	
	// group modal
	 showGroupModal = signal(false);
	 editingGroupId: number | null = null;
	 groupName = '';
	 groupType = 'select';
	 /** 多选成员（预设 / 其它策略组 / 自定义） */
	 groupMembers: string[] = ['ALL'];
	 groupCustomMember = '';
	 groupEnabled = true;
	 /** url-test / fallback 测速地址；空则发布时不写 url */
	 groupUrl = 'https://www.gstatic.com/generate_204';
	 /** 测速间隔（秒） */
	 groupInterval: number | null = 300;
	 groupTypes = PROXY_GROUP_TYPE_OPTIONS;
	 readonly defaultTestURL = 'https://www.gstatic.com/generate_204';
	 readonly defaultTestInterval = 300;

/** 策略组成员预设：引擎关键字 + 常用地区 + 系统实际出现的地区 */
		memberPresets = computed(() => {
			const base: { value: string; text: string }[] = [
				{ value: 'ALL', text: '全部节点 (ALL)' },
				{ value: 'DIRECT', text: '引擎直连 (DIRECT)' },
				{ value: 'REJECT', text: '引擎拒绝 (REJECT)' },
			];
		const codes = new Set<string>();
		for (const r of REGION_OPTIONS) codes.add(String(r.value).toUpperCase());
		for (const r of this.knownRegions()) codes.add(String(r).toUpperCase());
		const regionItems = [...codes]
			.filter(Boolean)
			.sort()
			.map((code) => ({
				value: `REGION:${code}`,
				text: `${regionLabel(code)}节点 (REGION:${code})`,
			}));
		return [...base, ...regionItems];
	 });

readonly tip = {
		publishNote: '可选。只写在发布历史里，方便以后回看这次改了什么。',
		ruleType: '用什么方式匹配流量：域名、域名后缀、关键词、国家代码、IP 段等。',
		ruleTarget: '命中后走哪个策略组。\n只从策略组列表选择，不要写引擎关键字。',
		ruleNote: '仅后台备注，方便自己辨认，不会写入 Clash 配置。',
ruleEnabled: '关闭后此规则不参与匹配，但仍保留在列表中。',
			batchImport:
				'一行一条，空行与 # 注释忽略。\n完整：TYPE,匹配内容,出口[,备注]\n仅域名：example.com（用下方默认类型/出口/备注）\n示例：\nDOMAIN-SUFFIX,xiaxiazi.ccwu.cc,直连,个人-小鸡\ngpt-api.xxww.online\nDOMAIN-KEYWORD,hybgzs,直连,个人-中转\n新规则会插在国内 GEOIP / 兜底 MATCH 之前。',
			groupName: '组名会被规则引用。中文也可以，如「直连」「拒绝」；地区组常用 US、JP。',
		groupType:
			'手动选择：客户端里自己挑节点。\n自动测速(url-test)：客户端定时探测 url，自动选延迟最低节点。\n故障转移：按顺序，挂了切下一个。\n负载均衡：在成员间分摊。\n测速在 Clash 客户端执行，不是本面板。',
		groupProxies:
			'勾选成员。\nALL=全部节点\nREGION:US=美国前缀节点\nDIRECT/REJECT=引擎直连/拒绝（给「直连」「拒绝」组用）\n也可勾选其它策略组，或手动添加具体节点名。\nurl-test 建议只放节点（REGION:XX / ALL）。',
		groupUrl:
			'测速探测地址。客户端用各节点访问此 URL 测延迟。\n常用：https://www.gstatic.com/generate_204',
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
				if (s.buildError) {
					this.draftStatusNote.set(`草稿暂无法生成：${s.buildError}`);
				} else if (!s.hasPublished) {
					this.draftStatusNote.set('尚未发布过配置，订阅链接在发布后才会有内容');
				} else if (s.dirty) {
					this.draftStatusNote.set(`有未发布更改（当前生效 v${s.publishedVersion || '?'}）`);
				} else {
					this.draftStatusNote.set(`已与 v${s.publishedVersion || '?'} 一致`);
				}
			},
			error: () => {
				this.draftDirty.set(false);
				this.draftStatusNote.set('');
			},
		});
	}

 @HostListener('document:keydown.escape')
 onEsc(): void {
if (this.showPublishModal()) {
	 this.closePublishModal();
	 return;
	 }
	 if (this.showTestModal()) {
	 this.closeTestModal();
	 return;
	 }
	 if (this.showBatchImportModal()) {
	 this.closeBatchImportModal();
	 return;
	 }
	 if (this.showRuleModal()) {
	 this.closeRuleModal();
	 return;
	 }
	 if (this.showGroupModal()) {
	 this.closeGroupModal();
	 }
	 }

 ruleTypeText(v: string | RuleType): string {
 return enumText(RULE_TYPE_OPTIONS, String(v));
 }

groupTypeText(v: string): string {
	 return enumText(PROXY_GROUP_TYPE_OPTIONS, String(v));
	 }

	/** 策略组类型徽章样式：手动 / 测速 / 故障转移 / 负载均衡 */
	groupTypeBadgeClass(type: string): string {
		switch (String(type || '')) {
			case 'select':
				return 'badge-type-select';
			case 'url-test':
				return 'badge-type-url-test';
			case 'fallback':
				return 'badge-type-fallback';
			case 'load-balance':
				return 'badge-type-load-balance';
			default:
				return 'badge-muted';
		}
	}

	groupTypeTip(type: string): string {
		switch (String(type || '')) {
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

/** 指向某策略组的规则（按全局匹配顺序） */
		rulesOfGroup(groupName: string): Rule[] {
			return this.sortedRules().filter((r) => r.target === groupName);
		}

		/** 目标策略组已不存在的规则（归入未归属） */
		orphanRules(): Rule[] {
			const names = new Set(this.groups().map((g) => g.name));
			return this.sortedRules().filter((r) => !names.has(r.target));
		}

		/** 全局匹配顺序（与 Clash 一致） */
		sortedRules(): Rule[] {
			return [...this.rules()].sort((a, b) => a.sortOrder - b.sortOrder || a.id - b.id);
		}

		/** 界面显示用：全局第几条（1-based），不暴露内部 sortOrder 数字 */
		globalOrderNo(rule: Rule): number {
			const idx = this.sortedRules().findIndex((r) => r.id === rule.id);
			return idx >= 0 ? idx + 1 : 0;
		}

	expandKeyForTarget(target: string): string {
		if (this.groups().some((g) => g.name === target)) return target;
		return '__orphan__';
	}

 isGroupExpanded(key: string): boolean {
 return this.expandedGroups().has(key);
 }

 toggleGroupExpand(key: string): void {
 const next = new Set(this.expandedGroups());
 if (next.has(key)) next.delete(key);
 else next.add(key);
 this.expandedGroups.set(next);
 }

 ensureGroupExpanded(key: string): void {
 if (this.expandedGroups().has(key)) return;
 const next = new Set(this.expandedGroups());
 next.add(key);
 this.expandedGroups.set(next);
 }

payloadLabel(type: string = this.ruleType): string {
		switch (type) {
			case RuleType.DOMAIN:
				return '完整域名';
			case RuleType.DOMAIN_SUFFIX:
				return '域名后缀';
			case RuleType.DOMAIN_KEYWORD:
				return '域名关键词';
			case RuleType.GEOSITE:
				return 'GeoSite 分类';
			case RuleType.GEOIP:
				return '国家/地区代码';
			case RuleType.IP_CIDR:
				return 'IPv4 网段';
			case RuleType.IP_CIDR6:
				return 'IPv6 网段';
			case RuleType.MATCH:
				return '匹配内容';
			default:
				return '匹配内容';
		}
	}

	payloadTip(type: string = this.ruleType): string {
		switch (type) {
			case RuleType.DOMAIN:
				return '填写完整域名（不含协议）。\n例如：www.google.com';
			case RuleType.DOMAIN_SUFFIX:
				return '填写域名后缀。\n例如：google.com、openai.com';
			case RuleType.DOMAIN_KEYWORD:
				return '填写域名中包含的关键词。\n例如：google、openai';
			case RuleType.GEOSITE:
				return 'Clash Meta 域名分类（客户端 geosite.dat）。\n广告拦截常用：category-ads-all\n其它如：youtube、google、cn';
			case RuleType.GEOIP:
				return '填写 GeoIP 国家代码。\n例如：CN（国内）、US、JP。\n国内直连常用 CN。';
			case RuleType.IP_CIDR:
				return '填写 IPv4 CIDR。\n例如：10.0.0.0/8、1.1.1.1/32';
			case RuleType.IP_CIDR6:
				return '填写 IPv6 CIDR。\n例如：2001:db8::/32';
			case RuleType.MATCH:
				return '最终匹配无需填写匹配内容，会兜底所有剩余流量。';
			default:
				return '按规则类型填写对应匹配值。';
		}
	}

	payloadPlaceholder(type: string = this.ruleType): string {
		switch (type) {
			case RuleType.DOMAIN:
				return 'www.example.com';
			case RuleType.DOMAIN_SUFFIX:
				return 'openai.com';
			case RuleType.DOMAIN_KEYWORD:
				return 'google';
			case RuleType.GEOSITE:
				return 'category-ads-all';
			case RuleType.GEOIP:
				return 'CN';
			case RuleType.IP_CIDR:
				return '10.0.0.0/8';
			case RuleType.IP_CIDR6:
				return '2001:db8::/32';
			default:
				return '';
		}
	}

 isMatchType(type: string = this.ruleType): boolean {
 return type === RuleType.MATCH;
 }

targetText(v: string): string {
		return v;
	}

	/** 当前目标不在策略组列表中时，下拉仍保留该项 */
	orphanTarget(): string | null {
		const v = (this.ruleTarget || '').trim();
		if (!v) return null;
		if (this.groups().some((g) => g.name === v)) return null;
		return v;
	}

	reload(): void {
		this.svc.listRules().subscribe({
			next: (r) => {
				const items = r.items || [];
				this.rules.set(items);
			},
			error: (e: Error) => void this.dialog.error(e.message),
		});
this.svc.listGroups().subscribe({
				next: (r) => {
					const items = r.items || [];
					this.groups.set(items);
					// 默认全部收缩，需要时点标题展开
				},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	 // 地区目录 + 源/节点实际地区，供 REGION:* 预设
	 this.sourceSvc.listRegions().subscribe({
	 next: (r) => {
	 const codes = new Set(this.knownRegions());
	 for (const item of r.items || []) {
	 const c = String(item.code || '').toUpperCase();
	 if (c && c !== 'UNKNOWN') codes.add(c);
	 }
	 this.knownRegions.set([...codes].sort());
	 },
	 error: () => {},
	 });
	 this.sourceSvc.list().subscribe({
	 next: (r) => {
	 const codes = new Set(this.knownRegions());
	 for (const s of r.items || []) {
	 const c = String(s.region || '').toUpperCase();
	 if (c && c !== 'UNKNOWN') codes.add(c);
	 }
	 this.knownRegions.set([...codes].sort());
	 },
	 error: () => {},
	 });
	 this.sourceSvc.listProxies().subscribe({
	 next: (r) => {
	 const codes = new Set(this.knownRegions());
	 for (const p of r.items || []) {
	 const c = String(p.region || '').toUpperCase();
	 if (c && c !== 'UNKNOWN') codes.add(c);
	 }
	 this.knownRegions.set([...codes].sort());
	 },
	 error: () => {},
	 });
 }

 /** 可选为成员的其它策略组（编辑时排除自身，避免自引用） */
 memberGroupOptions(): ProxyGroup[] {
 const selfId = this.editingGroupId;
 return this.groups().filter((g) => g.id !== selfId);
 }

/** 已选但不在预设/策略组列表中的成员（自定义节点名等） */
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

/** 是否需要测速 URL / 间隔（url-test、fallback） */
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
	 const name = this.groupName.trim();
	 const body: {
	 name: string;
	 type: string;
	 proxies: string[];
	 enabled: boolean;
	 url?: string;
	 interval?: number;
	 } = {
	 name,
	 type: this.groupType,
	 proxies: [...this.groupMembers],
	 enabled: this.groupEnabled,
	 };
	 if (this.needsTestParams()) {
	 body.url = this.groupUrl.trim();
	 body.interval = Number(this.groupInterval);
	 } else {
	 // 手动/负载均衡：清空测速 URL；interval 不传则后端置 nil
	 body.url = '';
	 }
	 const req =
	 this.editingGroupId == null
	 ? this.svc.createGroup(body)
	 : this.svc.updateGroup(this.editingGroupId, body);
req.subscribe({
			next: () => {
				void this.dialog.success(
					this.editingGroupId == null ? '策略组已创建' : '策略组已更新',
				);
				this.ensureGroupExpanded(name);
				this.closeGroupModal();
				this.reload();
				this.refreshDraftStatus();
			},
			error: (e: Error) => void this.dialog.error(e.message),
		});
	}

 resetRuleForm(target?: string): void {
 this.editingRuleId = null;
 this.ruleType = RuleType.DOMAIN_SUFFIX;
 this.rulePayload = '';
 this.ruleTarget = target || this.defaultTarget();
 this.ruleEnabled = true;
 this.ruleNote = '';
 }

defaultTarget(): string {
		if (this.groups().some((g) => g.name === '直连')) return '直连';
		if (this.groups().some((g) => g.name === 'PROXY')) return 'PROXY';
		if (this.groups().length > 0) return this.groups()[0].name;
		return '';
	}

openCreateRule(groupName?: string): void {
	 this.resetRuleForm(groupName);
	 this.showRuleModal.set(true);
	 }

	openBatchImport(groupName?: string): void {
		this.batchImportText = '';
		this.batchImportDefaultType = RuleType.DOMAIN;
		this.batchImportDefaultTarget = groupName || this.defaultTarget() || '直连';
		this.batchImportDefaultNote = '';
		this.batchImportEnabled = true;
		this.batchImporting.set(false);
		this.showBatchImportModal.set(true);
	}

	closeBatchImportModal(): void {
		if (this.batchImporting()) return;
		this.showBatchImportModal.set(false);
		this.batchImportText = '';
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
		this.batchImporting.set(true);
		this.svc
			.batchImportRules({
				text,
				defaultType: this.batchImportDefaultType,
				defaultTarget: this.batchImportDefaultTarget.trim(),
				defaultNote: this.batchImportDefaultNote.trim(),
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
						// 解析/重复提示不阻断成功，控制台可看
						console.warn('batch import notes', res.errors);
					}
					this.ensureGroupExpanded(this.expandKeyForTarget(this.batchImportDefaultTarget.trim()));
					this.closeBatchImportModal();
					this.reload();
					this.refreshDraftStatus();
				},
				error: (e: Error) => {
					this.batchImporting.set(false);
					void this.dialog.error(e.message);
				},
			});
	}

 openEditRule(rule: Rule): void {
 this.editingRuleId = rule.id;
 this.ruleType = rule.type as RuleType;
 this.rulePayload = rule.payload;
 this.ruleTarget = rule.target;
 this.ruleEnabled = rule.enabled;
 this.ruleNote = rule.note || '';
 this.showRuleModal.set(true);
 }

 closeRuleModal(): void {
 this.showRuleModal.set(false);
 this.resetRuleForm();
 }

 openPublishModal(): void {
 this.publishNote = '';
 this.preview.set(null);
 this.showPublishModal.set(true);
 }

 closePublishModal(): void {
 if (this.publishing()) return;
 this.showPublishModal.set(false);
 }

	 openTestModal(): void {
	 this.showTestModal.set(true);
	 }
	
	 closeTestModal(): void {
	 this.showTestModal.set(false);
	 }
	
	 /** 公共弹窗：规则类型文案 */
	 matchTypeText = (type: string): string => this.ruleTypeText(type);
	
	 /** 公共弹窗：出口文案 */
	 matchTargetText = (target: string): string => this.targetText(target);
	
	 canLocateMatchRule = (rule: MatchableRule): boolean => {
		 const r = this.ruleFromMatchable(rule);
		 return !!r && this.groups().some((g) => g.name === r.target);
	 };
	
	 onMatchEditRule(rule: MatchableRule): void {
		 const r = this.ruleFromMatchable(rule);
		 if (!r) return;
		 this.closeTestModal();
		 this.openEditRule(r);
	 }
	
	 onMatchLocateGroup(rule: MatchableRule): void {
		 const r = this.ruleFromMatchable(rule);
		 if (!r) return;
		 this.ensureGroupExpanded(r.target);
		 this.closeTestModal();
	 }
	
	 private ruleFromMatchable(rule: MatchableRule): Rule | null {
		 return (
			 this.rules().find(
				 (r) =>
					 String(r.type) === rule.type &&
					 (r.payload || '') === (rule.payload || '') &&
					 r.target === rule.target,
			 ) || null
		 );
	 }
	
	 onRuleTypeChange(): void {
 if (this.isMatchType()) {
 this.rulePayload = '';
 }
 }

 saveRule(): void {
 if (!this.isMatchType() && !this.rulePayload.trim()) {
 void this.dialog.error(`请填写「${this.payloadLabel()}」`);
 return;
 }
 if (!this.ruleTarget.trim()) {
 void this.dialog.error('请选择或填写目标出口');
 return;
 }
 const body = {
 type: this.ruleType,
 payload: this.isMatchType() ? '' : this.rulePayload.trim(),
 target: this.ruleTarget.trim(),
 enabled: this.ruleEnabled,
 note: this.ruleNote,
 };
 const req =
 this.editingRuleId == null
 ? this.svc.createRule(body)
 : this.svc.updateRule(this.editingRuleId, body);
req.subscribe({
			next: () => {
				void this.dialog.success('规则已保存（草稿，需发布后生效）');
				this.ensureGroupExpanded(this.expandKeyForTarget(body.target));
				this.closeRuleModal();
				this.reload();
				this.refreshDraftStatus();
			},
			error: (e: Error) => void this.dialog.error(e.message),
		});
	}

	toggleRule(rule: Rule): void {
		this.svc
			.updateRule(rule.id, {
				type: String(rule.type),
				payload: rule.payload,
				target: rule.target,
				enabled: !rule.enabled,
				note: rule.note,
				sortOrder: rule.sortOrder,
			})
			.subscribe({
				next: () => {
					this.reload();
					this.refreshDraftStatus();
				},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}

	async removeRule(rule: Rule): Promise<void> {
		const ok = await this.dialog.confirm('确认删除该规则？', '删除确认', '删除');
		if (!ok) return;
		this.svc.deleteRule(rule.id).subscribe({
			next: () => {
				void this.dialog.success('规则已删除');
				this.reload();
				this.refreshDraftStatus();
			},
			error: (e: Error) => void this.dialog.error(e.message),
		});
	}

/** 按全局匹配顺序上移/下移（与 Clash 规则顺序一致，跨策略组） */
	move(rule: Rule, dir: -1 | 1): void {
		const list = [...this.rules()].sort((a, b) => a.sortOrder - b.sortOrder || a.id - b.id);
		const idx = list.findIndex((r) => r.id === rule.id);
		const j = idx + dir;
		if (idx < 0 || j < 0 || j >= list.length) return;
		[list[idx], list[j]] = [list[j], list[idx]];
		this.svc.reorder(list.map((r) => r.id)).subscribe({
			next: () => {
				this.reload();
				this.refreshDraftStatus();
			},
			error: (e: Error) => void this.dialog.error(e.message),
		});
	}

	async removeGroup(g: ProxyGroup): Promise<void> {
		const count = this.rulesOfGroup(g.name).length;
		let cascade = false;
		if (count > 0) {
			const delRules = await this.dialog.confirm(
				`策略组「${g.name}」下有 ${count} 条规则。\n\n选「删除」：连同这些规则一起删掉。\n选「取消」后可再决定只删组（规则会变成未归属，发布可能失败）。\n\n是否删除组及其规则？`,
				'删除策略组',
				'删除组+规则',
			);
			if (!delRules) {
				const onlyGroup = await this.dialog.confirm(
					`仅删除策略组「${g.name}」，保留 ${count} 条规则（将显示为未归属）？\n发布前请改掉这些规则的目标出口。`,
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

 doPreview(): void {
 this.publishing.set(true);
 this.releaseSvc.preview().subscribe({
 next: (p) => {
 this.preview.set(p);
 this.publishing.set(false);
 },
 error: (e: Error) => {
 this.publishing.set(false);
 void this.dialog.error(e.message);
 },
 });
 }

 async publish(): Promise<void> {
 const ok = await this.dialog.confirm(
 '确认发布当前草稿配置？\n发布后订阅链接将立即使用新配置。',
 '发布确认',
 '发布',
 );
 if (!ok) return;
 this.publishing.set(true);
this.releaseSvc.publish(this.publishNote).subscribe({
			next: (res) => {
				this.preview.set(res.preview);
				this.publishNote = '';
				this.publishing.set(false);
				void this.dialog.success(`已发布 v${res.release.version}`);
				this.showPublishModal.set(false);
				this.refreshDraftStatus();
			},
			error: (e: Error) => {
				this.publishing.set(false);
				void this.dialog.error(e.message);
			},
		});
	}
}

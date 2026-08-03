import { Component, computed, inject, signal } from '@angular/core';
import { FormBuilder, Validators } from '@angular/forms';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import {
	PROXY_GROUP_TYPE_OPTIONS,
	ProxyGroup,
	regionLabel,
} from '@data-struct';
import { DialogService } from '@common/services/dialog.service';
import { RuleService } from '../../rules/services/rule.service';
import { CmParentFormComponent } from '@common/parents/parent-form/parent-form.component';
import { finalize, takeUntil } from 'rxjs';

export interface GroupFormDialogData {
	group: ProxyGroup | null;
	groups: ProxyGroup[];
	regionCatalog: { code: string; name: string }[];
	extraRegionCodes: string[];
	knownSources: { id: number; name: string }[];
}

@Component({
	selector: 'app-group-form',
	templateUrl: './group-form.component.html',
	standalone: false,
})
export class GroupFormComponent extends CmParentFormComponent {
	dialogRef = inject<MatDialogRef<GroupFormComponent, boolean>>(MatDialogRef);
	data = inject<GroupFormDialogData>(MAT_DIALOG_DATA);
	private fb = inject(FormBuilder);
	private svc = inject(RuleService);
	private dialog = inject(DialogService);

	isUpdate: boolean;
	groupTypes = PROXY_GROUP_TYPE_OPTIONS;
	readonly defaultTestURL = 'https://www.gstatic.com/generate_204';
	readonly defaultTestInterval = 300;

	members = signal<string[]>([]);
	customMember = signal('');

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

	memberPresets = computed(() => {
		const base: { value: string; text: string }[] = [
			{ value: 'ALL', text: '全部节点 (ALL)' },
			{ value: 'DIRECT', text: '引擎直连 (DIRECT)' },
			{ value: 'REJECT', text: '引擎拒绝 (REJECT)' },
		];
		const sourceItems = (this.data.knownSources || []).map((s) => ({
			value: `SOURCE:${s.name}`,
			text: `源「${s.name}」全部节点 (SOURCE:${s.name})`,
		}));
		const labels: Record<string, string> = {};
		const codes = new Set<string>();
		for (const r of this.data.regionCatalog || []) {
			const code = String(r.code || '').toUpperCase();
			const name = String(r.name || '').trim();
			if (!code || code === 'UNKNOWN') continue;
			codes.add(code);
			if (name) labels[code] = name;
		}
		for (const c of this.data.extraRegionCodes || []) {
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

	constructor() {
		super();
		const g = this.data.group;
		this.isUpdate = !!g;
		const type = String(g?.type || 'select');
		this.members.set([...(g?.proxies || (g ? [] : ['ALL']))]);
		this.editForm = this.fb.group({
			name: [g?.name || '', [Validators.required, Validators.maxLength(64)]],
			type: [type, [Validators.required]],
			enabled: [g?.enabled ?? true],
			url: [g?.url || this.defaultTestURL],
			interval: [g?.interval ?? this.defaultTestInterval],
		});
	}

	memberGroupOptions(): ProxyGroup[] {
		const selfId = this.data.group?.id;
		return (this.data.groups || []).filter((g) => g.id !== selfId);
	}

	customMembers(): string[] {
		const preset = new Set(this.memberPresets().map((p) => p.value));
		const groupNames = new Set(this.memberGroupOptions().map((g) => g.name));
		return this.members().filter((m) => !preset.has(m) && !groupNames.has(m));
	}

	isMemberSelected(value: string): boolean {
		return this.members().includes(value);
	}

	toggleMember(value: string): void {
		if (this.isMemberSelected(value)) {
			this.members.update((prev) => prev.filter((m) => m !== value));
		} else {
			this.members.update((prev) => [...prev, value]);
		}
		this.editForm.markAsDirty();
	}

	addCustomMember(): void {
		const v = this.customMember().trim();
		if (!v) return;
		if (!this.isMemberSelected(v)) {
			this.members.update((prev) => [...prev, v]);
			this.editForm.markAsDirty();
		}
		this.customMember.set('');
	}

	removeMember(value: string): void {
		this.members.update((prev) => prev.filter((m) => m !== value));
		this.editForm.markAsDirty();
	}

	needsTestParams(type: string = this.editForm.get('type')?.value): boolean {
		return type === 'url-test' || type === 'fallback';
	}

	onGroupTypeChange(type: string): void {
		this.editForm.patchValue({ type });
		if (this.needsTestParams(type)) {
			if (!String(this.editForm.get('url')?.value || '').trim()) {
				this.editForm.patchValue({ url: this.defaultTestURL });
			}
			const interval = this.editForm.get('interval')?.value;
			if (interval == null || interval < 1) {
				this.editForm.patchValue({ interval: this.defaultTestInterval });
			}
		}
	}

	submit(): void {
		if (this.isSubmitting) return;
		const raw = this.editForm.getRawValue();
		const name = String(raw.name || '').trim();
		const members = this.members();
		if (!name) {
			void this.dialog.error('请填写策略组名称');
			return;
		}
		if (members.length === 0) {
			void this.dialog.error('请至少选择一个成员');
			return;
		}
		const interval = raw.interval;
		if (this.needsTestParams(raw.type)) {
			if (!String(raw.url || '').trim()) {
				void this.dialog.error('自动测速/故障转移请填写测速 URL');
				return;
			}
			if (interval == null || interval < 1) {
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
			name,
			type: raw.type,
			proxies: [...members],
			enabled: !!raw.enabled,
		};
		if (this.needsTestParams(raw.type)) {
			body.url = String(raw.url || '').trim();
			body.interval = Number(interval);
		} else {
			body.url = '';
		}

		this.isSubmitting = true;
		const req =
			this.data.group == null
				? this.svc.createGroup(body)
				: this.svc.updateGroup(this.data.group.id, body);
		req
			.pipe(
				takeUntil(this.$destroy),
				finalize(() => (this.isSubmitting = false)),
			)
			.subscribe({
				next: () => {
					void this.dialog.success(this.data.group == null ? '策略组已创建' : '策略组已更新');
					this.dialogRef.close(true);
				},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}
}

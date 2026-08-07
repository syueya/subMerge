import { Component, computed, inject, signal } from '@angular/core';
import { FormBuilder, Validators } from '@angular/forms';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import {
	GroupFormDialogData,
	PROXY_GROUP_TYPE_OPTIONS,
	ProxyGroup,
	regionLabel,
} from '@data-struct';
import { DialogService } from '@common/services/dialog.service';
import { RuleService } from '../../rules/services/rule.service';
import { CmParentFormComponent } from '@common/parents/parent-form/parent-form.component';
import { finalize, takeUntil } from 'rxjs';

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
if (!code || code === 'UNK' || code === 'UNKNOWN') continue;
				codes.add(code);
				if (name) labels[code] = name;
			}
			for (const c of this.data.extraRegionCodes || []) {
				const code = String(c || '').toUpperCase();
				if (code && code !== 'UNK' && code !== 'UNKNOWN') codes.add(code);
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
		} else if (value === 'ALL') {
			this.members.set(['ALL']);
		} else {
			this.members.update((prev) => [...prev.filter((m) => m !== 'ALL'), value]);
		}
		this.editForm.markAsDirty();
	}

	addCustomMember(): void {
		const v = this.customMember().trim();
		if (!v) return;
		if (!this.isMemberSelected(v)) {
			if (v === 'ALL') {
				this.members.set(['ALL']);
			} else {
				this.members.update((prev) => [...prev.filter((m) => m !== 'ALL'), v]);
			}
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

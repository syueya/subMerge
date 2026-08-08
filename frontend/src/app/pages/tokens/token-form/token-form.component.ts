import { Component, WritableSignal, inject, signal } from '@angular/core';
import { FormBuilder, Validators } from '@angular/forms';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { CmParentFormComponent } from '@common/parents/parent-form/parent-form.component';
import { DialogService } from '@common/services/dialog.service';
import { MSG_NAME_REQUIRED } from '@common/util';
import {
	BADGE_WARN,
	ProxyGroup,
	ShareToken,
	SubscriptionSource,
	TOKEN_GROUP_MODE_OPTIONS,
	TokenFormDialogData,
	TokenGroupMode,
	enumText,
} from '@data-struct';
import { finalize, takeUntil } from 'rxjs';

import { TokenService } from '../services/token.service';

@Component({
	selector: 'app-token-form',
	templateUrl: './token-form.component.html',
	standalone: false,
})
export class TokenFormComponent extends CmParentFormComponent {
	dialogRef = inject<MatDialogRef<TokenFormComponent, boolean>>(MatDialogRef);
	data = inject<TokenFormDialogData>(MAT_DIALOG_DATA);
	private fb = inject(FormBuilder);
	private svc = inject(TokenService);
	private dialog = inject(DialogService);

	readonly groupModeOptions = TOKEN_GROUP_MODE_OPTIONS;
	readonly badgeWarn = BADGE_WARN;

	/** 编辑模式：data.token 有值 */
	readonly isEdit = !!this.data.token;

	allSources = signal(true);
	sourceIds = signal<Set<number>>(new Set());
	groupMode = signal<TokenGroupMode>('auto');
	groupNames = signal<Set<string>>(new Set());

	constructor() {
		super();
		if (this.isEdit) {
			this.editForm = this.fb.group({});
			this.initFromToken(this.data.token!);
		} else {
			this.editForm = this.fb.group({
				name: ['', [Validators.required, Validators.maxLength(64)]],
			});
			const list = this.data.sourceList || [];
			this.sourceIds.set(new Set(list.map((s) => s.id)));
			this.allSources.set(true);
		}
	}

	get token(): ShareToken | null {
		return this.data.token || null;
	}

	get modalTitle(): string {
		if (this.isEdit && this.token) {
			return `编辑链接范围 · ${this.token.name}`;
		}
		return '创建令牌';
	}

	get sourceList(): SubscriptionSource[] {
		return this.data.sourceList || [];
	}

	get groupList(): ProxyGroup[] {
		return this.data.groupList || [];
	}

	groupModeText(v?: string): string {
		return enumText(TOKEN_GROUP_MODE_OPTIONS, v || 'auto');
	}

	private initFromToken(item: ShareToken): void {
		const ids = item.sourceIds || [];
		if (ids.length === 0) {
			this.allSources.set(true);
			this.sourceIds.set(this.allSourceIdSet());
		} else {
			this.allSources.set(false);
			this.sourceIds.set(new Set(ids));
		}
		const mode = (item.groupMode || 'auto') as TokenGroupMode;
		this.groupMode.set(mode);
		this.groupNames.set(new Set(item.groupNames || []));
		if (mode === 'custom' && this.groupNames().size === 0) {
			this.groupNames.set(new Set(this.groupList.filter((g) => g.enabled).map((g) => g.name)));
		}
	}

	private allSourceIdSet(): Set<number> {
		return new Set(this.sourceList.map((s) => s.id));
	}

	private setAllSources(allFlag: WritableSignal<boolean>, ids: WritableSignal<Set<number>>, all: boolean): void {
		allFlag.set(all);
		if (all || ids().size === 0) {
			ids.set(this.allSourceIdSet());
		}
	}

	private toggleSource(
		allFlag: WritableSignal<boolean>,
		ids: WritableSignal<Set<number>>,
		id: number,
		checked: boolean,
	): void {
		const next = new Set(ids());
		if (checked) next.add(id);
		else next.delete(id);
		ids.set(next);
		const allIds = this.sourceList.map((s) => s.id);
		allFlag.set(allIds.length > 0 && next.size === allIds.length && allIds.every((sid) => next.has(sid)));
	}

	private isSourceChecked(allFlag: WritableSignal<boolean>, ids: WritableSignal<Set<number>>, id: number): boolean {
		return allFlag() || ids().has(id);
	}

	private applyGroupMode(names: WritableSignal<Set<string>>, mode: TokenGroupMode): void {
		if (mode === 'custom' && names().size === 0) {
			names.set(new Set(this.groupList.filter((g) => g.enabled).map((g) => g.name)));
		}
	}

	private toggleGroup(names: WritableSignal<Set<string>>, name: string, checked: boolean): void {
		const next = new Set(names());
		if (checked) next.add(name);
		else next.delete(name);
		names.set(next);
	}

	setAll(all: boolean): void {
		this.setAllSources(this.allSources, this.sourceIds, all);
		this.editForm.markAsDirty();
	}

	toggleSourceId(id: number, checked: boolean): void {
		this.toggleSource(this.allSources, this.sourceIds, id, checked);
		this.editForm.markAsDirty();
	}

	isSourceIdChecked(id: number): boolean {
		return this.isSourceChecked(this.allSources, this.sourceIds, id);
	}

	setGroupMode(mode: TokenGroupMode): void {
		this.groupMode.set(mode);
		this.applyGroupMode(this.groupNames, mode);
		this.editForm.markAsDirty();
	}

	toggleGroupName(name: string, checked: boolean): void {
		this.toggleGroup(this.groupNames, name, checked);
		this.editForm.markAsDirty();
	}

	isGroupNameChecked(name: string): boolean {
		return this.groupNames().has(name);
	}

	submit(): void {
		if (this.isSubmitting) return;
		this.editForm.markAllAsTouched();

		let name = '';
		if (!this.isEdit) {
			name = String(this.editForm.get('name')?.value || '').trim();
			if (!name) {
				void this.dialog.error(MSG_NAME_REQUIRED);
				return;
			}
		}

		const all = this.allSources();
		const ids = all ? [] : Array.from(this.sourceIds());
		if (!all && ids.length === 0) {
			void this.dialog.error('请至少选择一个订阅源，或勾选「全部源」');
			return;
		}
		const groupMode = this.groupMode();
		const groupNames = groupMode === 'custom' ? Array.from(this.groupNames()) : [];
		if (groupMode === 'custom' && groupNames.length === 0) {
			void this.dialog.error('自定义模式请至少勾选一个策略组');
			return;
		}

		this.isSubmitting = true;
		const req$ = this.isEdit
			? this.svc.update(this.token!.id, { sourceIds: ids, groupMode, groupNames })
			: this.svc.create(name, ids, groupMode, groupNames);

		req$
			.pipe(
				takeUntil(this.$destroy),
				finalize(() => (this.isSubmitting = false)),
			)
			.subscribe({
				next: (t) => {
					const scope =
						(t.sourceIds || []).length === 0 ? '全部源' : `指定 ${(t.sourceIds || []).length} 个源`;
					const gscope = this.groupModeText(t.groupMode);
					if (this.isEdit) {
						void this.dialog.success(`已更新「${t.name}」（${scope} · ${gscope}）`);
					} else {
						void this.dialog.success(
							t.subscribeUrl
								? `令牌「${t.name}」已创建（${scope} · ${gscope}），可直接复制订阅链接`
								: `令牌「${t.name}」已创建（${scope} · ${gscope}）`,
						);
					}
					this.dialogRef.close(true);
				},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}
}

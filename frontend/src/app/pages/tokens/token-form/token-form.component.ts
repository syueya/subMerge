import { Component, WritableSignal, inject, signal } from '@angular/core';
import { FormBuilder, Validators } from '@angular/forms';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import {
	BADGE_WARN,
	ProxyGroup,
	SubscriptionSource,
	TOKEN_GROUP_MODE_OPTIONS,
	TokenGroupMode,
	enumText,
} from '@data-struct';
import { DialogService } from '@common/services/dialog.service';
import { TokenService } from '../services/token.service';
import { CmParentFormComponent } from '@common/parents/parent-form/parent-form.component';
import { finalize, takeUntil } from 'rxjs';

export interface TokenFormDialogData {
	sourceList: SubscriptionSource[];
	groupList: ProxyGroup[];
}

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
	readonly tip = {
		name: '朋友备注名，仅用于管理识别，不会出现在订阅内容中。',
		sources: '勾选「全部源」时，之后新增的启用源也会自动纳入该链接；否则仅下发勾选的源。',
		groupMode:
			'自动：按该链接实际节点去掉空地区组（推荐）。\n全部：保留模板中的策略组。\n自定义：只下发勾选的策略组。',
	};

	allSources = signal(true);
	sourceIds = signal<Set<number>>(new Set());
	groupMode = signal<TokenGroupMode>('auto');
	groupNames = signal<Set<string>>(new Set());

	constructor() {
		super();
		this.editForm = this.fb.group({
			name: ['', [Validators.required, Validators.maxLength(64)]],
		});
		const list = this.data.sourceList || [];
		this.sourceIds.set(new Set(list.map((s) => s.id)));
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
		const name = String(this.editForm.get('name')?.value || '').trim();
		if (!name) {
			void this.dialog.error('请填写朋友备注名');
			return;
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
		this.svc
			.create(name, ids, groupMode, groupNames)
			.pipe(
				takeUntil(this.$destroy),
				finalize(() => (this.isSubmitting = false)),
			)
			.subscribe({
				next: (t) => {
					const scope =
						(t.sourceIds || []).length === 0 ? '全部源' : `指定 ${(t.sourceIds || []).length} 个源`;
					const gscope = this.groupModeText(t.groupMode);
					void this.dialog.success(
						t.subscribeUrl
							? `令牌「${t.name}」已创建（${scope} · ${gscope}），可直接复制订阅链接`
							: `令牌「${t.name}」已创建（${scope} · ${gscope}）`,
					);
					this.dialogRef.close(true);
				},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}
}

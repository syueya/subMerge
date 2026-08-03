import { Component, WritableSignal, inject, signal } from '@angular/core';
import { FormBuilder } from '@angular/forms';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import {
	BADGE_WARN,
	ProxyGroup,
	ShareToken,
	SubscriptionSource,
	TOKEN_GROUP_MODE_OPTIONS,
	TokenEditDialogData,
	TokenGroupMode,
	enumText,
} from '@data-struct';
import { DialogService } from '@common/services/dialog.service';
import { TokenService } from '../services/token.service';
import { CmParentFormComponent } from '@common/parents/parent-form/parent-form.component';
import { finalize, takeUntil } from 'rxjs';

@Component({
	selector: 'app-token-edit',
	templateUrl: './token-edit.component.html',
	standalone: false,
})
export class TokenEditComponent extends CmParentFormComponent {
	dialogRef = inject<MatDialogRef<TokenEditComponent, boolean>>(MatDialogRef);
	data = inject<TokenEditDialogData>(MAT_DIALOG_DATA);
	private fb = inject(FormBuilder);
	private svc = inject(TokenService);
	private dialog = inject(DialogService);

readonly groupModeOptions = TOKEN_GROUP_MODE_OPTIONS;
	readonly badgeWarn = BADGE_WARN;

	allSources = signal(true);
	sourceIds = signal<Set<number>>(new Set());
	groupMode = signal<TokenGroupMode>('auto');
	groupNames = signal<Set<string>>(new Set());

	constructor() {
		super();
		// 仅用于 dirty 标记与表单外壳
		this.editForm = this.fb.group({});
		const item = this.data.token;
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

	get token(): ShareToken {
		return this.data.token;
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
			.update(this.token.id, {
				sourceIds: ids,
				groupMode,
				groupNames,
			})
			.pipe(
				takeUntil(this.$destroy),
				finalize(() => (this.isSubmitting = false)),
			)
			.subscribe({
				next: (t) => {
					const scope =
						(t.sourceIds || []).length === 0 ? '全部源' : `指定 ${(t.sourceIds || []).length} 个源`;
					void this.dialog.success(`已更新「${t.name}」（${scope} · ${this.groupModeText(t.groupMode)}）`);
					this.dialogRef.close(true);
				},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}
}

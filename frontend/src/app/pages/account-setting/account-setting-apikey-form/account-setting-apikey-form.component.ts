import { Component, inject, signal } from '@angular/core';
import { FormBuilder, Validators } from '@angular/forms';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { MatChipListboxChange } from '@angular/material/chips';
import {
	APIKey,
	APIKeyFormDialogData,
	APIKeyScope,
	API_KEY_SCOPE_HINTS,
	API_KEY_SCOPE_OPTIONS,
} from '@data-struct';
import { DialogService } from '@common/services/dialog.service';
import { CmParentFormComponent } from '@common/parents/parent-form/parent-form.component';
import { finalize, takeUntil } from 'rxjs';

import { ApiKeyService } from '../services/apikey.service';

@Component({
	selector: 'app-account-setting-apikey-form',
	templateUrl: './account-setting-apikey-form.component.html',
	standalone: false,
})
export class AccountSettingApikeyFormComponent extends CmParentFormComponent {
	dialogRef = inject<MatDialogRef<AccountSettingApikeyFormComponent, APIKey | boolean>>(MatDialogRef);
	data = inject<APIKeyFormDialogData>(MAT_DIALOG_DATA);
	private fb = inject(FormBuilder);
	private svc = inject(ApiKeyService);
	private dialog = inject(DialogService);

	readonly scopeOptions = API_KEY_SCOPE_OPTIONS;
	readonly isEdit = !!this.data.key;

	scopeHint(scope: APIKeyScope): string {
		return API_KEY_SCOPE_HINTS[scope] || '';
	}

	/** chip-listbox 当前选中值（多选） */
	selectedScopes = signal<APIKeyScope[]>(['read']);

	constructor() {
		super();
		if (this.isEdit) {
			const k = this.data.key!;
			this.editForm = this.fb.group({
				name: [k.name, [Validators.required, Validators.maxLength(128)]],
				note: [k.note || '', [Validators.maxLength(512)]],
			});
			this.selectedScopes.set(this.normalizeDisplayScopes((k.scopes || []) as APIKeyScope[]));
		} else {
			this.editForm = this.fb.group({
				name: ['', [Validators.required, Validators.maxLength(128)]],
				note: ['', [Validators.maxLength(512)]],
			});
		}
	}

	get modalTitle(): string {
		if (this.isEdit && this.data.key) {
			return `编辑 API 密钥 · ${this.data.key.name}`;
		}
		return '创建 API 密钥';
	}

	onScopeChange(ev: MatChipListboxChange): void {
		const raw = ev.value;
		const values: APIKeyScope[] = Array.isArray(raw)
			? (raw as APIKeyScope[])
			: raw
				? [raw as APIKeyScope]
				: [];
		this.selectedScopes.set(this.resolveScopeSelection(this.selectedScopes(), values));
		this.editForm.markAsDirty();
	}

	/**
	 * 「全部」与单项互斥：
	 * - 新勾选 * → 只保留 *
	 * - 在 * 基础上再勾单项 → 取消 *，保留单项
	 * - 仅单项多选
	 */
	private resolveScopeSelection(prev: APIKeyScope[], next: APIKeyScope[]): APIKeyScope[] {
		const prevHadAll = prev.includes('*');
		const nextSet = new Set(next.filter(Boolean));
		const nextHasAll = nextSet.has('*');
		const order: APIKeyScope[] = ['read', 'write', 'publish'];

		if (nextHasAll && !prevHadAll) {
			return ['*'];
		}
		if (nextHasAll && prevHadAll && nextSet.size > 1) {
			nextSet.delete('*');
			return order.filter((s) => nextSet.has(s));
		}
		if (nextHasAll) {
			return ['*'];
		}
		return order.filter((s) => nextSet.has(s));
	}

	private normalizeDisplayScopes(scopes: APIKeyScope[]): APIKeyScope[] {
		const set = new Set(scopes.filter(Boolean));
		if (set.has('*')) {
			return ['*'];
		}
		const order: APIKeyScope[] = ['read', 'write', 'publish'];
		return order.filter((s) => set.has(s));
	}

	private scopesForSubmit(): APIKeyScope[] {
		return this.normalizeDisplayScopes(this.selectedScopes());
	}

	submit(): void {
		if (this.isSubmitting) return;
		if (this.editForm.invalid) {
			this.editForm.markAllAsTouched();
			return;
		}
		const selected = this.scopesForSubmit();
		if (selected.length === 0) {
			void this.dialog.error('请至少选择一个权限');
			return;
		}

		const v = this.editForm.getRawValue();
		const name = String(v.name || '').trim();
		const note = String(v.note || '').trim();
		if (!name) {
			void this.dialog.error('请填写名称');
			return;
		}

		this.isSubmitting = true;
		// 过期时间前端不展示：创建默认不过期；编辑不改 expiresAt
		const req$ = this.isEdit
			? this.svc.update(this.data.key!.id, {
					name,
					scopes: selected,
					note,
				})
			: this.svc.create(name, selected, note, null);

		req$
			.pipe(
				takeUntil(this.$destroy),
				finalize(() => (this.isSubmitting = false)),
			)
			.subscribe({
				next: (item) => {
					if (this.isEdit) {
						void this.dialog.success(`已更新「${item.name}」`);
						this.dialogRef.close(true);
					} else {
						void this.dialog.success(`密钥「${item.name}」已创建`);
						this.dialogRef.close(item);
					}
				},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}
}

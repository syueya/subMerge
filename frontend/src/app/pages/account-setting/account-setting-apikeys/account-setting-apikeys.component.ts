import { AfterViewInit, Component, inject, signal } from '@angular/core';
import { MatTableDataSource } from '@angular/material/table';
import {
	APIKey,
	APIKeyFormDialogData,
	APIKeySecretDialogData,
	APIKeyStatus,
	API_KEY_SCOPE_OPTIONS,
	API_KEY_STATUS_BADGE,
	API_KEY_STATUS_OPTIONS,
	enumBadgeClass,
	enumText,
} from '@data-struct';
import { DialogService } from '@common/services/dialog.service';
import { formatDateTime } from '@common/util';
import { CmDialogOpenService } from '@common/modules/dialog';
import { CmParentTableComponent } from '@common/parents/parent-table/parent-table.component';
import { finalize, takeUntil } from 'rxjs';

import { AccountSettingApikeyFormComponent } from '../account-setting-apikey-form/account-setting-apikey-form.component';
import { AccountSettingApikeySecretComponent } from '../account-setting-apikey-secret/account-setting-apikey-secret.component';
import { ApiKeyService } from '../services/apikey.service';

@Component({
	selector: 'app-account-setting-apikeys',
	templateUrl: './account-setting-apikeys.component.html',
	standalone: false,
})
export class AccountSettingApikeysComponent extends CmParentTableComponent implements AfterViewInit {
	private svc = inject(ApiKeyService);
	private dialog = inject(DialogService);
	private dialogOpen = inject(CmDialogOpenService);

	dataSource = new MatTableDataSource<APIKey>([]);
	override displayedColumns: string[] = ['name', 'key', 'note', 'scopes', 'used', 'action'];

	busy = signal(false);
	formatTime = formatDateTime;

	constructor() {
		super();
		this.rememberPageSize(this.constructor.name);
	}

	override handlerAfterViewInit(): void {
		super.handlerAfterViewInit();
		this.reloadTableDataByFirstPage();
	}

	override reloadTableData(): void {
		this.isLoading = true;
		this.svc
			.list(true)
			.pipe(
				takeUntil(this.$destroy),
				finalize(() => (this.isLoading = false)),
			)
			.subscribe({
				next: (r) => {
					const list = r.items || [];
					this.dataSource.data = list;
					this.paginatorProps.length = list.length;
				},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}

	statusText(v: string): string {
		return enumText(API_KEY_STATUS_OPTIONS, v);
	}

	statusClass(v: string): string {
		return enumBadgeClass(API_KEY_STATUS_BADGE, v);
	}

	scopesText(scopes: string[] | undefined): string {
		if (!scopes?.length) return '—';
		// 展示顺序与 chips 一致：全部 → 读取 → 写入 → 发布
		const order = ['*', 'read', 'write', 'publish'];
		const sorted = [...scopes].sort(
			(a, b) => (order.indexOf(a) + 99) - (order.indexOf(b) + 99),
		);
		return sorted.map((s) => enumText(API_KEY_SCOPE_OPTIONS, s, s)).join('、');
	}

	openCreate(): void {
		const data: APIKeyFormDialogData = { key: null };
		const ref = this.dialogOpen.openSmallForm(AccountSettingApikeyFormComponent, data);
		ref.afterClosed().subscribe((result) => {
			if (!result) return;
			this.reloadTableDataByFirstPage();
			if (typeof result === 'object' && result.key) {
				this.showSecret(result.name, result.key, '密钥已创建，请复制保存。之后也可在列表中再次查看。');
			}
		});
	}

	openEdit(item: APIKey): void {
		const data: APIKeyFormDialogData = { key: item };
		const ref = this.dialogOpen.openSmallForm(AccountSettingApikeyFormComponent, data);
		ref.afterClosed().subscribe((ok) => {
			if (ok) this.reloadTableDataByFirstPage();
		});
	}

	viewSecret(item: APIKey): void {
		this.busy.set(true);
		this.svc
			.secret(item.id)
			.pipe(
				takeUntil(this.$destroy),
				finalize(() => this.busy.set(false)),
			)
			.subscribe({
				next: (res) => this.showSecret(item.name, res.key),
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}

	private showSecret(name: string, key: string, hint?: string): void {
		const data: APIKeySecretDialogData = { name, key, hint };
		this.dialogOpen.openSmallForm(AccountSettingApikeySecretComponent, data);
	}

	disable(item: APIKey): void {
		this.busy.set(true);
		this.svc
			.update(item.id, { status: APIKeyStatus.Disabled })
			.pipe(
				takeUntil(this.$destroy),
				finalize(() => this.busy.set(false)),
			)
			.subscribe({
				next: () => {
					void this.dialog.success(`已禁用「${item.name}」`);
					this.reloadTableDataByFirstPage();
				},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}

	enable(item: APIKey): void {
		this.busy.set(true);
		this.svc
			.update(item.id, { status: APIKeyStatus.Active })
			.pipe(
				takeUntil(this.$destroy),
				finalize(() => this.busy.set(false)),
			)
			.subscribe({
				next: () => {
					void this.dialog.success(`已启用「${item.name}」`);
					this.reloadTableDataByFirstPage();
				},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}

	async revoke(item: APIKey): Promise<void> {
		const ok = await this.dialog.confirm(
			`作废「${item.name}」？\n旧密钥立即失效，记录保留，之后可重新生成。`,
			'作废确认',
			'作废',
		);
		if (!ok) return;
		this.busy.set(true);
		this.svc
			.revoke(item.id)
			.pipe(
				takeUntil(this.$destroy),
				finalize(() => this.busy.set(false)),
			)
			.subscribe({
				next: () => {
					void this.dialog.success(`已作废「${item.name}」`);
					this.reloadTableDataByFirstPage();
				},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}

	async regenerate(item: APIKey): Promise<void> {
		const ok = await this.dialog.confirm(
			`重新生成「${item.name}」？\n旧密钥立即失效，请把新密钥更新到 agent。`,
			'重新生成',
			'重新生成',
		);
		if (!ok) return;
		this.busy.set(true);
		this.svc
			.regenerate(item.id)
			.pipe(
				takeUntil(this.$destroy),
				finalize(() => this.busy.set(false)),
			)
			.subscribe({
				next: (k) => {
					this.reloadTableDataByFirstPage();
					if (k.key) {
						this.showSecret(k.name, k.key, '已重新生成，请复制新密钥。');
					} else {
						void this.dialog.success(`已重新生成「${item.name}」`);
					}
				},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}

	async remove(item: APIKey): Promise<void> {
		const ok = await this.dialog.confirm(
			`永久删除「${item.name}」？\n记录不可恢复（与「作废」不同）。`,
			'删除确认',
			'删除',
		);
		if (!ok) return;
		this.busy.set(true);
		this.svc
			.delete(item.id)
			.pipe(
				takeUntil(this.$destroy),
				finalize(() => this.busy.set(false)),
			)
			.subscribe({
				next: () => {
					void this.dialog.success(`已删除「${item.name}」`);
					this.reloadTableDataByFirstPage();
				},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}
}

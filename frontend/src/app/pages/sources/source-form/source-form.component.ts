import { Component, inject } from '@angular/core';
import { FormBuilder, Validators } from '@angular/forms';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import {
	DEFAULT_EXCLUDE_NAME_REGEX,
	DEFAULT_EXCLUDE_SERVERS,
	FALLBACK_REGION,
	RegionCatalogEntry,
	RegionMode,
	SubscriptionSource,
	regionOptionText,
} from '@data-struct';
import { DialogService } from '@common/services/dialog.service';
import { SourceService } from '../services/source.service';
import { formatRefreshMsg } from '../services/source-refresh.util';
import { CmParentFormComponent } from '@common/parents/parent-form/parent-form.component';
import { concatMap, finalize, takeUntil } from 'rxjs';

export interface SourceFormDialogData {
	source: SubscriptionSource | null;
	regionCatalog: RegionCatalogEntry[];
	fallbackRegion: string;
}

@Component({
	selector: 'app-source-form',
	templateUrl: './source-form.component.html',
	standalone: false,
})
export class SourceFormComponent extends CmParentFormComponent {
	dialogRef = inject<MatDialogRef<SourceFormComponent, boolean>>(MatDialogRef);
	data = inject<SourceFormDialogData>(MAT_DIALOG_DATA);
	private fb = inject(FormBuilder);
	private svc = inject(SourceService);
	private dialog = inject(DialogService);

	isUpdate: boolean;
	showAdvanced = false;

	readonly tip = {
		name: '备注名会拼到节点名末尾（如「良心云」→ …-良心云），请起简短可辨认的名字。',
		region:
			'固定模式：从目录选择地区，该源全部节点加此前缀。\n自动识别：回退建议选「未知 (UNKNOWN)」，仅当节点名识别失败时使用。',
		regionMode:
			'自动识别（机场推荐）：从节点名 emoji/关键词推断 JP/HK/US…，混合地区订阅一条即可；回退默认 UNKNOWN。\n固定地区：整源强制同一前缀（单地区订阅更合适）。',
		url: '上游 Clash 订阅地址。加密存储，界面只显示脱敏结果。编辑时留空表示不修改。',
		excludeName:
			'名称匹配此正则的节点会被丢弃（不区分大小写）。\n默认含：剩余流量、套餐到期、过滤掉… 等机场信息节点。\n拉取成功后弹窗会列出被过滤的名称。',
		excludeServers: 'server 黑名单，逗号/换行分隔。如 127.0.0.1,localhost',
		includeName: '可选。非空时只保留名称匹配的节点（白名单）。',
		enabled: '关闭后此源不参与拉取与发布。',
	};

	constructor() {
		super();
		const item = this.data.source;
		this.isUpdate = !!item;
		const mode: RegionMode = item?.regionMode || 'auto';
		const fallback = this.data.fallbackRegion || FALLBACK_REGION;
		const region =
			item?.region ||
			(mode === 'auto' ? fallback : this.fixedRegionOptions()[0]?.code || 'US');

		this.editForm = this.fb.group({
			name: [item?.name || '', [Validators.required, Validators.maxLength(64)]],
			regionMode: [mode as RegionMode, [Validators.required]],
			region: [region, [Validators.required]],
			url: [''],
			enabled: [item?.enabled ?? true],
			excludeNameRegex: [item?.excludeNameRegex ?? DEFAULT_EXCLUDE_NAME_REGEX],
			excludeServers: [item?.excludeServers ?? DEFAULT_EXCLUDE_SERVERS],
			includeNameRegex: [item?.includeNameRegex ?? ''],
		});

		if (!this.isUpdate) {
			this.editForm.get('url')?.setValidators([Validators.required]);
			this.editForm.get('url')?.updateValueAndValidity();
		}

		this.editForm
			.get('regionMode')
			?.valueChanges.pipe(takeUntil(this.$destroy))
			.subscribe((m: RegionMode) => this.onRegionModeChange(m));
	}

	fixedRegionOptions(): RegionCatalogEntry[] {
		return (this.data.regionCatalog || []).filter(
			(r) => String(r.code).toUpperCase() !== FALLBACK_REGION,
		);
	}

	autoRegionOptions(): RegionCatalogEntry[] {
		const list = [...(this.data.regionCatalog || [])];
		list.sort((a, b) => {
			const au = String(a.code).toUpperCase() === FALLBACK_REGION ? 0 : 1;
			const bu = String(b.code).toUpperCase() === FALLBACK_REGION ? 0 : 1;
			if (au !== bu) return au - bu;
			return String(a.code).localeCompare(String(b.code));
		});
		return list;
	}

	regionOptionLabel(r: RegionCatalogEntry): string {
		return regionOptionText(r.code, r.name);
	}

	onRegionModeChange(mode: RegionMode): void {
		const fallback = this.data.fallbackRegion || FALLBACK_REGION;
		const current = String(this.editForm.get('region')?.value || '').toUpperCase();
		if (mode === 'auto') {
			if (!current || current === 'US') {
				this.editForm.patchValue({ region: fallback }, { emitEvent: false });
			}
		} else if (mode === 'fixed') {
			if (!current || current === FALLBACK_REGION) {
				const first = this.fixedRegionOptions()[0];
				this.editForm.patchValue({ region: first?.code || 'US' }, { emitEvent: false });
			}
		}
	}

	get regionMode(): RegionMode {
		return this.editForm.get('regionMode')?.value || 'auto';
	}

	submit(): void {
		if (this.isSubmitting) return;
		const raw = this.editForm.getRawValue();
		const name = String(raw.name || '').trim();
		const region = String(raw.region || '')
			.trim()
			.toUpperCase();
		const url = String(raw.url || '').trim();
		const mode = raw.regionMode as RegionMode;

		if (!name) {
			void this.dialog.error('请填写名称');
			return;
		}
		if (!this.isUpdate && !url) {
			void this.dialog.error('请填写订阅 URL');
			return;
		}
		if (!/^[A-Z0-9]{1,16}$/.test(region)) {
			void this.dialog.error('请选择有效地区（如 UNKNOWN、US、JP、HK）');
			return;
		}
		if (mode === 'fixed' && region === FALLBACK_REGION) {
			void this.dialog.error('固定地区模式请选择具体国家/地区，不要用 UNKNOWN');
			return;
		}

		const body = {
			name,
			region,
			enabled: !!raw.enabled,
			regionMode: mode,
			excludeNameRegex: String(raw.excludeNameRegex ?? ''),
			excludeServers: String(raw.excludeServers ?? ''),
			includeNameRegex: String(raw.includeNameRegex ?? ''),
			...(url ? { url } : {}),
		};

		this.isSubmitting = true;
		if (!this.isUpdate) {
			this.svc
				.create({ ...body, url })
				.pipe(
					concatMap((source) => this.svc.refresh(source.id)),
					takeUntil(this.$destroy),
					finalize(() => (this.isSubmitting = false)),
				)
				.subscribe({
					next: (res) => {
						void this.dialog.success(formatRefreshMsg(res, '已创建并拉取完成'));
						this.dialogRef.close(true);
					},
					error: (err: Error) => {
						void this.dialog.error(
							`自动拉取失败：${err.message}\n若列表中已出现该源，可稍后点「拉取」重试。`,
						);
						this.dialogRef.close(true);
					},
				});
			return;
		}

		this.svc
			.update(this.data.source!.id, body)
			.pipe(
				takeUntil(this.$destroy),
				finalize(() => (this.isSubmitting = false)),
			)
			.subscribe({
				next: (src) => {
					void this.dialog.success(`已更新「${src.name}」`);
					this.dialogRef.close(true);
				},
				error: (err: Error) => void this.dialog.error(err.message),
			});
	}
}

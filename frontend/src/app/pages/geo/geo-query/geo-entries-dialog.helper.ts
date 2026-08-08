import { Injectable, inject } from '@angular/core';
import { MatDialogRef } from '@angular/material/dialog';
import { CM_DIALOG_WIDTH, CmDialogOpenService } from '@common/modules/dialog';
import { GeoEntriesDialogData, GeoEntriesDialogResult } from '@data-struct';

import { RuleCreateService } from '../../_shared/rule-create.service';
import { GeoEntriesComponent } from '../geo-entries/geo-entries.component';

export interface GeoEntriesDialogOptions {
	/** 打开/更新弹窗数据 */
	data: GeoEntriesDialogData;
	/** 是否强制打开（即使 items 为空） */
	forceOpen: boolean;
	/** 翻页回调 */
	onPage: (pageIndex: number, pageSize: number) => void;
	/** 打开成功后回调（用于联动 rule 创建） */
	onOpen?: (ref: MatDialogRef<GeoEntriesComponent, GeoEntriesDialogResult>) => void;
}

/**
 * Geo 条目弹窗生命周期管理：打开、按需更新、关闭。
 * 供 geo-domain-query 与 geo-category-search 共用，避免重复实现。
 */
@Injectable({ providedIn: 'root' })
export class GeoEntriesDialogHelper {
	private readonly dialogOpen = inject(CmDialogOpenService);
	private readonly ruleCreate = inject(RuleCreateService);

	private entriesDialogRef: MatDialogRef<GeoEntriesComponent, GeoEntriesDialogResult> | null = null;

	/** 打开或就地更新条目弹窗 */
	openOrUpdate(opts: GeoEntriesDialogOptions): void {
		const payload: GeoEntriesDialogData = {
			...opts.data,
			onPage: opts.onPage
		};

		if (this.entriesDialogRef) {
			this.entriesDialogRef.componentInstance.applyData(payload);
			return;
		}
		if (!opts.forceOpen && !payload.items.length && !payload.loading) return;

		const ref = this.dialogOpen.openContent(GeoEntriesComponent, payload, {
			width: CM_DIALOG_WIDTH.large
		});
		this.entriesDialogRef = ref;
		opts.onOpen?.(ref);
		ref.afterClosed().subscribe((result: GeoEntriesDialogResult) => {
			if (this.entriesDialogRef === ref) {
				this.entriesDialogRef = null;
			}
			if (result?.action === 'add') {
				this.ruleCreate.open(result.context);
			}
		});
	}

	/** 更新已打开弹窗的 loading 状态（不关窗） */
	markLoading(loading: boolean, getData: () => GeoEntriesDialogData): void {
		const ref = this.entriesDialogRef;
		if (!ref) return;
		const inst = ref.componentInstance;
		inst.applyData({
			...getData(),
			loading
		});
	}

	close(): void {
		this.entriesDialogRef?.close(null);
		this.entriesDialogRef = null;
	}

	/** 是否已打开弹窗 */
	isOpen(): boolean {
		return !!this.entriesDialogRef;
	}
}
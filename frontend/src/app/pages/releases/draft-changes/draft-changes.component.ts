import { Component, inject } from '@angular/core';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import {
	BADGE_ERR,
	BADGE_MUTED,
	BADGE_OK,
	BADGE_WARN,
	DraftChange,
	DraftChangesDialogData,
} from '@data-struct';

@Component({
	selector: 'app-draft-changes',
	templateUrl: './draft-changes.component.html',
	standalone: false,
})
export class DraftChangesComponent {
	dialogRef = inject<MatDialogRef<DraftChangesComponent>>(MatDialogRef);
	data = inject<DraftChangesDialogData>(MAT_DIALOG_DATA);

	get changes(): DraftChange[] {
		return this.data.changes || [];
	}

	get summary(): string {
		return this.data.summary || '';
	}

	get publishedVersion(): number | undefined {
		return this.data.publishedVersion;
	}

	get modalTitle(): string {
		const n = this.changes.length;
		const ver = this.publishedVersion;
		if (ver != null) return `未发布更改（相对 v${ver} · ${n}）`;
		return `未发布更改（${n}）`;
	}

	changeActionText(a: string): string {
		switch (a) {
			case 'added':
				return '新增';
			case 'removed':
				return '删除';
			case 'modified':
				return '修改';
			default:
				return a;
		}
	}

	changeKindText(k: string): string {
		switch (k) {
			case 'proxy':
				return '节点';
			case 'group':
				return '策略组';
			case 'rule':
				return '规则';
			default:
				return k;
		}
	}

	changeActionClass(a: string): string {
		switch (a) {
			case 'added':
				return BADGE_OK;
			case 'removed':
				return BADGE_ERR;
			case 'modified':
				return BADGE_WARN;
			default:
				return BADGE_MUTED;
		}
	}

	close(): void {
		this.dialogRef.close();
	}
}

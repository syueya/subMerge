import { Component, inject, signal } from '@angular/core';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import {
	RELEASE_STATUS_OPTIONS,
	ReleaseDetail,
	ReleaseDetailDialogData,
	enumText,
} from '@data-struct';

@Component({
	selector: 'app-release-detail',
	templateUrl: './release-detail.component.html',
	standalone: false,
})
export class ReleaseDetailComponent {
	dialogRef = inject<MatDialogRef<ReleaseDetailComponent>>(MatDialogRef);
	data = inject<ReleaseDetailDialogData>(MAT_DIALOG_DATA);

	viewTab = signal<'rules' | 'yaml'>('rules');

	get detail(): ReleaseDetail {
		return this.data.detail;
	}

	statusText(v: string): string {
		return enumText(RELEASE_STATUS_OPTIONS, v);
	}

	close(): void {
		this.dialogRef.close();
	}
}

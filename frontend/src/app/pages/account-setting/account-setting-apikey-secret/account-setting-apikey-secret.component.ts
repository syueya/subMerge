import { Component, inject } from '@angular/core';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { CmParentComponent } from '@common/parents/parent/parent.component';
import { DialogService } from '@common/services/dialog.service';
import { copyToClipboard, MSG_COPIED, MSG_COPY_FAILED } from '@common/util';
import { APIKeySecretDialogData } from '@data-struct';

@Component({
	selector: 'app-account-setting-apikey-secret',
	templateUrl: './account-setting-apikey-secret.component.html',
	standalone: false,
})
export class AccountSettingApikeySecretComponent extends CmParentComponent {
	dialogRef = inject<MatDialogRef<AccountSettingApikeySecretComponent>>(MatDialogRef);
	data = inject<APIKeySecretDialogData>(MAT_DIALOG_DATA);
	private dialog = inject(DialogService);

	async copy(): Promise<void> {
		try {
			await copyToClipboard(this.data.key);
			void this.dialog.success(MSG_COPIED);
		} catch {
			void this.dialog.error(MSG_COPY_FAILED);
		}
	}
}

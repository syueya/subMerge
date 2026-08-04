import { Component, inject } from '@angular/core';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { APIKeySecretDialogData } from '@data-struct';
import { DialogService } from '@common/services/dialog.service';
import { CmParentComponent } from '@common/parents/parent/parent.component';

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
			await this.writeClipboard(this.data.key);
			void this.dialog.success('已复制到剪贴板');
		} catch {
			void this.dialog.error('复制失败，请手动选择文本');
		}
	}

	private async writeClipboard(text: string): Promise<void> {
		if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText && window.isSecureContext) {
			await navigator.clipboard.writeText(text);
			return;
		}
		const ta = document.createElement('textarea');
		ta.value = text;
		ta.setAttribute('readonly', '');
		ta.style.position = 'fixed';
		ta.style.left = '-9999px';
		ta.style.top = '0';
		document.body.appendChild(ta);
		ta.select();
		ta.setSelectionRange(0, text.length);
		try {
			if (!document.execCommand('copy')) {
				throw new Error('execCommand copy failed');
			}
		} finally {
			document.body.removeChild(ta);
		}
	}
}

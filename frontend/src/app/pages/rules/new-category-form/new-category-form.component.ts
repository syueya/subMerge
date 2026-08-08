import { Component, inject } from '@angular/core';
import { FormBuilder, Validators } from '@angular/forms';
import { MatDialogRef } from '@angular/material/dialog';
import { CmParentFormComponent } from '@common/parents/parent-form/parent-form.component';
import { DialogService } from '@common/services/dialog.service';

import { CATEGORY_NEW_VALUE } from '../services/rule-ui';

@Component({
	selector: 'app-new-category-form',
	templateUrl: './new-category-form.component.html',
	standalone: false,
})
export class NewCategoryFormComponent extends CmParentFormComponent {
	dialogRef = inject<MatDialogRef<NewCategoryFormComponent, string | null>>(MatDialogRef);
	private fb = inject(FormBuilder);
private dialog = inject(DialogService);

	constructor() {
		super();
		this.editForm = this.fb.group({
			name: ['', [Validators.required, Validators.maxLength(64)]],
		});
	}

	submit(): void {
		const name = String(this.editForm.get('name')?.value || '').trim();
		if (!name) {
			void this.dialog.error('请填写分类名称');
			return;
		}
		if (name === CATEGORY_NEW_VALUE) {
			void this.dialog.error('分类名称不可用');
			return;
		}
		this.dialogRef.close(name);
	}
}

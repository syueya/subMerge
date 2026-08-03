import { Component, inject, signal } from '@angular/core';
import { FormBuilder } from '@angular/forms';
import { MatDialogRef } from '@angular/material/dialog';
import { ReleasePreview } from '@data-struct';
import { DialogService } from '@common/services/dialog.service';
import { DraftStatusStore } from '../../releases/services/draft-status.store';
import { ReleaseService } from '../../releases/services/release.service';
import { CmParentFormComponent } from '@common/parents/parent-form/parent-form.component';
import { finalize, takeUntil } from 'rxjs';

@Component({
	selector: 'app-publish-form',
	templateUrl: './publish-form.component.html',
	standalone: false,
})
export class PublishFormComponent extends CmParentFormComponent {
	dialogRef = inject<MatDialogRef<PublishFormComponent, boolean>>(MatDialogRef);
	private fb = inject(FormBuilder);
	private releaseSvc = inject(ReleaseService);
	private draftStore = inject(DraftStatusStore);
	private dialog = inject(DialogService);

preview = signal<ReleasePreview | null>(null);
	previewing = false;

	constructor() {
		super();
		this.editForm = this.fb.group({
			note: [''],
		});
	}

	doPreview(): void {
		this.previewing = true;
		this.releaseSvc
			.preview()
			.pipe(
				takeUntil(this.$destroy),
				finalize(() => (this.previewing = false)),
			)
			.subscribe({
				next: (p) => this.preview.set(p),
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}

	async submit(): Promise<void> {
		if (this.isSubmitting) return;
		const ok = await this.dialog.confirm(
			'确认发布当前草稿配置？\n发布后订阅链接将立即使用新配置。',
			'发布确认',
			'发布',
		);
		if (!ok) return;
		this.isSubmitting = true;
		this.releaseSvc
			.publish(String(this.editForm.get('note')?.value || ''))
			.pipe(
				takeUntil(this.$destroy),
				finalize(() => (this.isSubmitting = false)),
			)
			.subscribe({
				next: (res) => {
					this.preview.set(res.preview);
					void this.dialog.success(`已发布 v${res.release.version}`);
					this.draftStore.refresh();
					this.dialogRef.close(true);
				},
				error: (e: Error) => void this.dialog.error(e.message),
			});
	}
}

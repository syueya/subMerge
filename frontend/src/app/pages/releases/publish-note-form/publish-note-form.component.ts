import { Component, inject } from '@angular/core';
import { FormBuilder, Validators } from '@angular/forms';
import { MatDialogRef } from '@angular/material/dialog';
import { CmParentFormComponent } from '@common/parents/parent-form/parent-form.component';

@Component({
  selector: 'app-publish-note-form',
  templateUrl: './publish-note-form.component.html',
  standalone: false,
})
export class PublishNoteFormComponent extends CmParentFormComponent {
  dialogRef = inject<MatDialogRef<PublishNoteFormComponent, string | undefined>>(MatDialogRef);
  private fb = inject(FormBuilder);

  constructor() {
    super();
    this.editForm = this.fb.group({
      note: ['', [Validators.maxLength(255)]],
    });
  }

  submit(): void {
    if (this.editForm.invalid) {
      this.editForm.markAllAsTouched();
      return;
    }
    this.dialogRef.close(String(this.editForm.get('note')?.value || '').trim());
  }
}

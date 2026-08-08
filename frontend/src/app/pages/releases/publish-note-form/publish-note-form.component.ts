import { Component, inject } from '@angular/core';
import { FormBuilder, Validators } from '@angular/forms';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { CmParentFormComponent } from '@common/parents/parent-form/parent-form.component';
import { BADGE_ERR, BADGE_MUTED, BADGE_OK, BADGE_WARN, DraftChange } from '@data-struct';

export interface PublishNoteFormData {
  changes: DraftChange[];
  summary: string;
}

export interface PublishNoteFormResult {
  confirmed: true;
  note: string;
}

@Component({
  selector: 'app-publish-note-form',
  templateUrl: './publish-note-form.component.html',
  standalone: false,
})
export class PublishNoteFormComponent extends CmParentFormComponent {
  readonly data = inject<PublishNoteFormData>(MAT_DIALOG_DATA);
  dialogRef = inject<MatDialogRef<PublishNoteFormComponent, PublishNoteFormResult | undefined>>(MatDialogRef);
  private fb = inject(FormBuilder);

  constructor() {
    super();
    this.editForm = this.fb.group({
      note: ['', [Validators.maxLength(255)]],
    });
  }

  get changes(): DraftChange[] {
    return this.data.changes || [];
  }

  get summary(): string {
    return this.data.summary || '';
  }

  changeActionText(a: string): string {
    switch (a) {
      case 'added': return '新增';
      case 'removed': return '删除';
      case 'modified': return '修改';
      default: return a;
    }
  }

  changeKindText(k: string): string {
    switch (k) {
      case 'proxy': return '节点';
      case 'group': return '策略组';
      case 'rule': return '规则';
      default: return k;
    }
  }

  changeActionClass(a: string): string {
    switch (a) {
      case 'added': return BADGE_OK;
      case 'removed': return BADGE_ERR;
      case 'modified': return BADGE_WARN;
      default: return BADGE_MUTED;
    }
  }

  submit(): void {
    if (this.editForm.invalid) {
      this.editForm.markAllAsTouched();
      return;
    }
    this.dialogRef.close({
      confirmed: true,
      note: String(this.editForm.get('note')?.value || '').trim(),
    });
  }
}

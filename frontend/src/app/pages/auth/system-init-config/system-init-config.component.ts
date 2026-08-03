import { Component, inject } from '@angular/core';
import { Validators, FormBuilder } from '@angular/forms';
import { MatDialogRef } from '@angular/material/dialog';
import { CmMessageService } from '@common/modules/message';
import { CmParentFormComponent } from '@common/parents/parent-form/parent-form.component';
import { AuthService } from '@common/services';
import { validateConfirmPassword, validateUserName } from '@common/util';
import { takeUntil } from 'rxjs';

@Component({
  selector: 'app-system-init-config',
  standalone: false,
  templateUrl: './system-init-config.component.html'
})
export class SystemInitConfigComponent extends CmParentFormComponent {
  dialogRef = inject<MatDialogRef<SystemInitConfigComponent>>(MatDialogRef);
  private fb = inject(FormBuilder);
  private auth = inject(AuthService);
  private message = inject(CmMessageService);

  override isSubmitting = false;

  constructor() {
    super();
    this.editForm = this.fb.group({
      nickName: ['', [Validators.maxLength(32)]],
      userName: ['', [Validators.required, Validators.minLength(1), Validators.maxLength(32), validateUserName]],
      password: ['', [Validators.required, Validators.minLength(10), Validators.maxLength(72)]],
      passwordSecond: ['', [Validators.required, Validators.minLength(10), Validators.maxLength(72), validateConfirmPassword('password')]]
    });
  }

  submit() {
    if (this.editForm.invalid) {
      this.editForm.markAllAsTouched();
      return;
    }
    const v = this.editForm.getRawValue();
    this.isSubmitting = true;
    this.auth
      .bootstrap({
        username: String(v.userName || '').trim(),
        password: String(v.password || ''),
        displayName: String(v.nickName || '').trim() || undefined
      })
      .pipe(takeUntil(this.$destroy))
      .subscribe({
        next: () => {
          this.isSubmitting = false;
          this.message.success('系统初始化配置保存成功！请使用管理员账号登录。');
          this.dialogRef.close(true);
        },
        error: (err: Error) => {
          this.isSubmitting = false;
          this.message.error(err.message || '初始化失败');
        }
      });
  }
}

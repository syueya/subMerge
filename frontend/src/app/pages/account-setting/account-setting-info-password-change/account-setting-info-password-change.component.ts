import { Component, inject } from '@angular/core';
import { FormGroup, FormControl, Validators } from '@angular/forms';
import { CmMessageService } from '@common/modules/message';
import { CmParentFormComponent } from '@common/parents/parent-form/parent-form.component';
import { AuthService } from '@common/services';
import { validateConfirmPassword } from '@common/util';
import { takeUntil } from 'rxjs';

@Component({
  selector: 'app-account-setting-info-password-change',
  templateUrl: './account-setting-info-password-change.component.html',
  standalone: false
})
export class AccountSettingInfoPasswordChangeComponent extends CmParentFormComponent {
  private authService = inject(AuthService);
  private message = inject(CmMessageService);

  override isSubmitting = false;

  override editForm = new FormGroup({
    oldPassword: new FormControl('', [Validators.required, Validators.minLength(6), Validators.maxLength(72)]),
    password: new FormControl('', [Validators.required, Validators.minLength(6), Validators.maxLength(72)]),
    passwordSecond: new FormControl('', [
      Validators.required,
      Validators.minLength(6),
      Validators.maxLength(72),
      validateConfirmPassword('password')
    ])
  });

  submit() {
    if (this.editForm.invalid) {
      this.editForm.markAllAsTouched();
      return;
    }
    this.isSubmitting = true;
    const v = this.editForm.getRawValue();
    this.authService
      .changePassword(String(v.oldPassword || ''), String(v.password || ''))
      .pipe(takeUntil(this.$destroy))
      .subscribe({
        next: () => {
          this.message.success('用户密码修改成功！请使用新密码重新登录！');
          this.editForm.reset();
          this.isSubmitting = false;
          void this.authService.logout(false);
        },
        error: (err: Error) => {
          this.message.error(err.message || '修改密码失败');
          this.isSubmitting = false;
        }
      });
  }
}

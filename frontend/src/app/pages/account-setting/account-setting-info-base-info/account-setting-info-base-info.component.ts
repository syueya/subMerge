import { Component, inject } from '@angular/core';
import { FormGroup, FormControl, Validators } from '@angular/forms';
import { CmMessageService } from '@common/modules/message';
import { CmParentFormComponent } from '@common/parents/parent-form/parent-form.component';
import { AuthService } from '@common/services';
import { takeUntil } from 'rxjs';

@Component({
  selector: 'app-account-setting-info-base-info',
  templateUrl: './account-setting-info-base-info.component.html',
  standalone: false
})
export class AccountSettingInfoBaseInfoComponent extends CmParentFormComponent {
  private authService = inject(AuthService);
  private message = inject(CmMessageService);

  override isSubmitting = false;

  override editForm = new FormGroup({
    userName: new FormControl('', [Validators.required, Validators.minLength(1), Validators.maxLength(32)]),
    nickName: new FormControl('', [Validators.maxLength(32)])
  });

  constructor() {
    super();
    this.authService.$userInfo.pipe(takeUntil(this.$destroy)).subscribe(userInfo => {
      if (userInfo) {
        this.editForm.patchValue({
          userName: userInfo.userName,
          nickName: userInfo.nickName
        });
      }
    });
  }

  submit() {
    if (this.editForm.invalid) {
      this.editForm.markAllAsTouched();
      return;
    }
    this.isSubmitting = true;
    const v = this.editForm.getRawValue();
    this.authService
      .updateProfile({
        username: v.userName || undefined,
        displayName: v.nickName || undefined
      })
      .pipe(takeUntil(this.$destroy))
      .subscribe({
        next: () => {
          this.message.success('用户基本信息修改成功！');
          this.isSubmitting = false;
        },
        error: (error: Error) => {
          console.error('Error updating user information:', error);
          this.message.error(error.message || '修改用户信息时发生错误');
          this.isSubmitting = false;
        }
      });
  }
}

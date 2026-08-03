import { HttpClient } from '@angular/common/http';
import { Component, inject } from '@angular/core';
import { Validators, FormBuilder } from '@angular/forms';
import { MatDialogRef, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { HttpRespone } from '@common/interfaces';
import { CmMessageService } from '@common/modules/message';
import { CmParentFormComponent } from '@common/parents/parent-form/parent-form.component';
import { validateConfirmPassword, validateUserName } from '@common/util';
import { debounceTime, takeUntil, finalize } from 'rxjs';

import { User } from '../interfaces/User';

@Component({
  selector: 'app-setting-user-update',
  standalone: false,
  templateUrl: './setting-user-update.component.html'
})

export class SettingUserUpdateComponent extends CmParentFormComponent {
  dialogRef = inject<MatDialogRef<SettingUserUpdateComponent>>(MatDialogRef);
  data = inject<User>(MAT_DIALOG_DATA);
  private fb = inject(FormBuilder);
  private httpClient = inject(HttpClient);
  private message = inject(CmMessageService);

  isUpdate: boolean;

  constructor() {
    super(); // 调用父类的构造函数
    this.isUpdate = !!this.data; // 判断是否是编辑模式

    // 初始化表单
    this.editForm = this.fb.group({
      id: [null, []],
      userName: ['', [Validators.required, Validators.minLength(2), Validators.maxLength(20), validateUserName]],
      nickName: ['', [Validators.required, Validators.minLength(2), Validators.maxLength(20)]],
      password: ['', []], // 新增时必填，编辑时可选
      passwordSecond: ['', []] // 新增时必填，编辑时可选
    });

    // 如果是编辑模式
    if (this.isUpdate) {
      //编辑时密码可不填
      this.editForm.get('password')?.clearValidators();
      this.editForm.get('passwordSecond')?.clearValidators();
      this.editForm.patchValue(this.data);
    } else {
      //新增时密码必填
      this.editForm.get('password')?.setValidators([Validators.required, Validators.minLength(8), Validators.maxLength(72)]);
      this.editForm.get('passwordSecond')?.setValidators([Validators.required, Validators.minLength(8), Validators.maxLength(72), validateConfirmPassword('password')]);
    }
    this.editForm.get('password')?.updateValueAndValidity();
    this.editForm.get('passwordSecond')?.updateValueAndValidity();

    //密码修改后更新密码校验
    this.editForm
      .get('password')
      ?.valueChanges.pipe(takeUntil(this.$destroy)).pipe(debounceTime(500))
      .subscribe(value => {
        this.checkPasswords();
      });

    this.editForm
      .get('passwordSecond')
      ?.valueChanges.pipe(takeUntil(this.$destroy)).pipe(debounceTime(500))
      .subscribe(value => {
        this.checkPasswords();
      });
  }

  /*
  * 密码修改后更新密码校验
  */
  checkPasswords(): void {
    const password = this.editForm.get('password')?.value;
    const passwordSecond = this.editForm.get('passwordSecond')?.value;

    if (password || passwordSecond) {
      this.editForm.get('password')?.setValidators([Validators.required, Validators.minLength(8), Validators.maxLength(72)]);
      this.editForm.get('passwordSecond')?.setValidators([Validators.required, Validators.minLength(8), Validators.maxLength(72), validateConfirmPassword('password')]);
    } else {
      this.editForm.get('password')?.clearValidators();
      this.editForm.get('passwordSecond')?.clearValidators();
    }
    this.editForm.get('password')?.updateValueAndValidity();
    this.editForm.get('passwordSecond')?.updateValueAndValidity();
  }

  // 提交保存
  submit() {
    if (!this.canSubmit) {
      return;
    }
    this.isSubmitting = true;
    this.httpClient
      .post<HttpRespone<boolean>>('/api/v1/user/save', this.editForm.getRawValue())
      .pipe(takeUntil(this.$destroy), finalize(() => this.isSubmitting = false))
      .subscribe(res => {
        if (res.code === 20000) {
          this.message.success('用户信息保存成功');
          this.dialogRef.close(true);
        }
      });
  }
}

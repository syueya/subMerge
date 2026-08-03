import { HttpClient } from '@angular/common/http';
import { Component, inject } from '@angular/core';
import { MatDialogRef } from '@angular/material/dialog';
import { AvatarList } from '@common/data/avatar-list';
import { HttpRespone } from '@common/interfaces';
import { CmMessageService } from '@common/modules/message';
import { CmParentComponent } from '@common/parents/parent/parent.component';
import { AuthService } from '@common/services';
import { takeUntil } from 'rxjs';

@Component({
    selector: 'app-account-setting-info-avatar-setting',
    templateUrl: './account-setting-info-avatar-setting.component.html',
    styleUrl: './account-setting-info-avatar-setting.component.scss',
    standalone: false
})
export class AccountSettingInfoAvatarSettingComponent extends CmParentComponent {
  dialogRef = inject<MatDialogRef<AccountSettingInfoAvatarSettingComponent>>(MatDialogRef);
  private authService = inject(AuthService);
  private httpClient = inject(HttpClient);
  private message = inject(CmMessageService);

  /**
   * 是否正在提交
   */
  isSubmitting = false;

  /**
   * 头像列表
   */
  avatarList = AvatarList;

  /**
   * 跟踪头像
   */
  trackByAvatar = (
    _: number,
    item: {
      value: string;
      src: string;
    }
  ) => item.value;

  /**
   * 激活的图片
   */
  activeAvatar = '1';

  constructor() {
    super();
    this.authService.$userInfo.pipe(takeUntil(this.$destroy)).subscribe((userInfo) => {
      this.activeAvatar = userInfo!.avatar;
      //用户头像
      if (userInfo?.avatar) {
        this.activeAvatar = userInfo.avatar;
      } else {
        //默认用户头像
        this.activeAvatar = '1';
      }
    });
  }

  /**
   * 提交表单
   */
  submit() {
    if (!this.activeAvatar) {
      return;
    }
    this.isSubmitting = true;

    this.httpClient
      .put<HttpRespone<boolean>>('/api/v1/user/updateAvatar', {
        avatar: this.activeAvatar
      }).pipe(takeUntil(this.$destroy))
      .subscribe({
        next: (res) => {
          if (res.code === 20000) {
            this.message.success('用户头像更新成功');
            this.authService.refreshUserInfo();
            this.dialogRef.close(true);
          } else {
            this.isSubmitting = false;
          }
        },
        error: (error) => {
          console.error('Error updating user avatar:', error);
          this.message.error('更新用户头像时发生错误');
          this.isSubmitting = false;
        }
      });
  }
}

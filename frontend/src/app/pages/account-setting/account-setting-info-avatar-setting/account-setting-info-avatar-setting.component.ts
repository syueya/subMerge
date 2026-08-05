import { Component, inject } from '@angular/core';
import { MatDialogRef } from '@angular/material/dialog';
import { AvatarList } from '@common/data/avatar-list';
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
  private message = inject(CmMessageService);

  /** 是否正在提交 */
  isSubmitting = false;

  /** 预设头像列表 */
  avatarList = AvatarList;

  trackByAvatar = (_: number, item: { value: string; src: string }) => item.value;

  /** 当前选中的预设 value，或 data URL */
  activeAvatar = '1';

  constructor() {
    super();
    this.authService.$userInfo.pipe(takeUntil(this.$destroy)).subscribe((userInfo) => {
      const avatar = userInfo?.avatar || '';
      if (!avatar) {
        this.activeAvatar = '1';
        return;
      }
      // 已存 data URL：尽量反查对应预设，否则保留 data URL 本身
      if (avatar.startsWith('data:')) {
        const matched = AvatarList.find((item) => item.value === avatar || item.src === avatar);
        this.activeAvatar = matched?.value || avatar;
        return;
      }
      // 历史模板可能只存了 '1'..'12'
      if (AvatarList.some((item) => item.value === avatar)) {
        this.activeAvatar = avatar;
        return;
      }
      this.activeAvatar = '1';
    });
  }

  async submit() {
    if (!this.activeAvatar || this.isSubmitting) {
      return;
    }
    this.isSubmitting = true;
    try {
      const dataURL = await this.resolveAvatarDataURL(this.activeAvatar);
      await new Promise<void>((resolve, reject) => {
        this.authService
          .updateProfile({ avatar: dataURL })
          .pipe(takeUntil(this.$destroy))
          .subscribe({
            next: () => resolve(),
            error: (err: Error) => reject(err)
          });
      });
      this.message.success('用户头像更新成功');
      this.dialogRef.close(true);
    } catch (error) {
      console.error('Error updating user avatar:', error);
      const msg = error instanceof Error && error.message ? error.message : '更新用户头像时发生错误';
      this.message.error(msg);
      this.isSubmitting = false;
    }
  }

  /** 将预设 value / 路径 / 已有 data URL 统一为后端可接受的 data URL */
  private async resolveAvatarDataURL(avatar: string): Promise<string> {
    const trimmed = (avatar || '').trim();
    if (!trimmed) {
      throw new Error('请选择头像');
    }
    if (trimmed.startsWith('data:image/')) {
      return trimmed;
    }
    const preset = AvatarList.find((item) => item.value === trimmed || item.src === trimmed);
    const src = preset?.src || (trimmed.startsWith('/') || trimmed.startsWith('./') ? trimmed : '');
    if (!src) {
      throw new Error('无效的头像选择');
    }
    return this.fetchAsDataURL(src);
  }

  private fetchAsDataURL(src: string): Promise<string> {
    return new Promise((resolve, reject) => {
      const img = new Image();
      img.crossOrigin = 'anonymous';
      img.onload = () => {
        try {
          const canvas = document.createElement('canvas');
          canvas.width = img.naturalWidth || img.width;
          canvas.height = img.naturalHeight || img.height;
          const ctx = canvas.getContext('2d');
          if (!ctx) {
            reject(new Error('无法处理头像图片'));
            return;
          }
          ctx.drawImage(img, 0, 0);
          // JPEG 预设体积更小，满足后端 data URL 大小限制
          resolve(canvas.toDataURL('image/jpeg', 0.92));
        } catch (e) {
          reject(e instanceof Error ? e : new Error('头像转换失败'));
        }
      };
      img.onerror = () => reject(new Error('头像资源加载失败'));
      img.src = src;
    });
  }
}

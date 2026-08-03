import { Component, inject } from '@angular/core';
import { UserInfo } from '@common/interfaces';
import { CM_DIALOG_WIDTH, CmDialogOpenService } from '@common/modules/dialog';
import { CmParentComponent } from '@common/parents/parent/parent.component';
import { AuthService } from '@common/services';
import { takeUntil } from 'rxjs';

import { AccountSettingInfoAvatarSettingComponent } from '../account-setting-info-avatar-setting/account-setting-info-avatar-setting.component';

@Component({
    selector: 'app-account-setting-info',
    templateUrl: './account-setting-info.component.html',
    standalone: false
})
export class AccountSettingInfoComponent extends CmParentComponent {
  private authService = inject(AuthService);
  private dialogOpen = inject(CmDialogOpenService);

  userInfo!: UserInfo;

  constructor() {
    super();
    this.authService.$userInfo.pipe(takeUntil(this.$destroy)).subscribe((userinfo) => {
      this.userInfo = userinfo!;
    });
  }

  /**
   * 打开头像设置弹窗
   */
  openAvatarSettingDialog() {
    // 高度/滚动由 CmDialogOpenService 统一注入的 cm-dialog-panel 处理，不要再单独写 maxHeight / panelClass
    this.dialogOpen.openPicker(AccountSettingInfoAvatarSettingComponent, {}, {
      width: CM_DIALOG_WIDTH.medium
    });
  }
}

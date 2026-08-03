import { NgOptimizedImage } from '@angular/common';
import { Component, inject } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatTooltipModule } from '@angular/material/tooltip';
import { RouterModule } from '@angular/router';
import { DirectivesModule } from '@common/directives/directives.module';
import { UserInfo } from '@common/interfaces';
import { CM_DIALOG_WIDTH } from '@common/modules/dialog';
import { IconsModule } from '@common/modules/icons/icons.module';
import { CmSharedDialogService, CmSharedDialogDataModel } from '@common/modules/shared-dialog';
import { CmParentComponent } from '@common/parents/parent/parent.component';
import { AuthService } from '@common/services';
import { UserRoleEnum, UserRoleEnumToName } from '@data-struct';
import { takeUntil } from 'rxjs';

import { AppVersionComponent } from '../app-version/app-version.component';

@Component({
    selector: 'app-sidenav-account-info',
    imports: [RouterModule, DirectivesModule, IconsModule, MatButtonModule, MatTooltipModule, NgOptimizedImage, AppVersionComponent],
    templateUrl: './sidenav-account-info.component.html'
})
export class SidenavAccountInfoComponent  extends CmParentComponent{
  private authService = inject(AuthService);
  private dialogService = inject(CmSharedDialogService);

  userInfo!:UserInfo;

  constructor() {

    super();
    this.authService.$userInfo.pipe(takeUntil(this.$destroy)).subscribe(userinfo=>{
      this.userInfo = userinfo!;
    })
  }

  roleLabel(): string {
    return UserRoleEnumToName[Number(this.userInfo?.role)] || UserRoleEnumToName[UserRoleEnum.Normal];
  }

  
  /**
   * 退出登录
   */
  logOut() {
    this.dialogService.open({
      model: CmSharedDialogDataModel.confirm,
      title: '确认',
      content: '确认注销当前设备登录？',
      width: CM_DIALOG_WIDTH.small
    })
    .afterClosed()
    .subscribe((result) => {
      if(result){
        this.authService.logout();
      }

    });

  }
}

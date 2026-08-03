import { HttpClient } from '@angular/common/http';
import { Component, inject } from '@angular/core';
import { HttpRespone } from '@common/interfaces';
import { CM_DIALOG_WIDTH } from '@common/modules/dialog';
import { CmMessageService } from '@common/modules/message';
import { CmSharedDialogDataModel, CmSharedDialogService } from '@common/modules/shared-dialog';
import { CmParentComponent } from '@common/parents/parent/parent.component';
import { AuthService } from '@common/services';
import { formatDate } from '@common/util';
import { takeUntil } from 'rxjs';

import { DeviceItem } from '@data-struct';

@Component({
    selector: 'app-account-setting-security-device-manage',
    templateUrl: './account-setting-security-device-manage.component.html',
    standalone: false
})
export class AccountSettingSecurityDeviceManageComponent extends CmParentComponent {
  private authService = inject(AuthService);
  private httpClient = inject(HttpClient);
  private message = inject(CmMessageService);
  private dialogService = inject(CmSharedDialogService);

  /**
   * 登录设备列表
   */
  deviceList!: DeviceItem[];

  /**
   * 跟踪头像
   */
  trackByDeviceItem = (_: number, item: DeviceItem) => item.token;

  constructor() {
    super();
    this.getDeviceList();
  }

  getDeviceList() {
    this.httpClient.get<HttpRespone<DeviceItem[]>>('/api/v1/user/tokens').pipe(takeUntil(this.$destroy)).subscribe((res) => {
      if (res.code === 20000) {
        const data = res?.data?.length ? res?.data : [];
        data.forEach((item: DeviceItem) => {
          if (item.loginTime) {
            item.loginTimeStr = formatDate(item.loginTime, 'yyyy-MM-dd HH:mm');
          } else {
            item.loginTimeStr = '未知登录时间';
          }
        });
        this.deviceList = data;
      }else{
        this.deviceList = [];
      }
    });
  }

  logOutDevice(token: string) {
    this.dialogService
      .open({
        model: CmSharedDialogDataModel.confirm,
        title: '注销设备',
        content: '确认注销该设备? 注销后该设备将被退出登录状态',
        width: CM_DIALOG_WIDTH.small
      })
      .afterClosed()
      .subscribe((result) => {
        if (result) {
          this.httpClient
            .post<HttpRespone<boolean>>('/api/v1/user/logoutByTokens', {
              tokens: [token]
            }).pipe(takeUntil(this.$destroy))
            .subscribe((res) => {
              if (res.code === 20000) {
                this.message.success('设备注销成功');
                this.getDeviceList();
              }
            });
        }
      });
  }

  logOutAllDevice(){
    if (!this.deviceList?.length) {
      this.message.warning('暂无设备可注销！');
      return;
    }
    const tokens = this.deviceList.filter((item) => !item.current).map((item) => item.token) || [];
    if (!tokens.length) {
      this.message.warning('暂无设备可注销！');
      return;
    }
    this.dialogService
    .open({
      model: CmSharedDialogDataModel.confirm,
      title: '注销所有设备',
      content: '确认注销所有设备(非当前设备)? 注销后这些设备将被退出登录状态',
      width: CM_DIALOG_WIDTH.small
    })
    .afterClosed()
    .subscribe((result) => {
      if (result) {
        this.httpClient
          .post<HttpRespone<boolean>>('/api/v1/user/logoutByTokens', {
            tokens: tokens
          }).pipe(takeUntil(this.$destroy))
          .subscribe((res) => {
            if (res.code === 20000) {
              this.message.success('所有设备已注销成功！(不包含当前设备，当前设备可直接退出登录)');
              this.getDeviceList();
            }
          });
      }
    });
  }
}

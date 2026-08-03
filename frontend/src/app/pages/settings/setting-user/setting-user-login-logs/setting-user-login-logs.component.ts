import { HttpClient } from '@angular/common/http';
import { Component, OnInit, inject } from '@angular/core';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { PageEvent } from '@angular/material/paginator';
import { MatTableDataSource } from '@angular/material/table';
import { ListRespone } from '@common/interfaces';
import { CmParentComponent } from '@common/parents/parent/parent.component';
import { formatDate } from '@common/util';
import { takeUntil, finalize } from 'rxjs';

import { LoginLog } from '../interfaces/LoginLog';

@Component({
  selector: 'app-setting-user-login-logs',
  standalone: false,
  templateUrl: './setting-user-login-logs.component.html'
})
export class SettingUserLoginLogsComponent extends CmParentComponent implements OnInit {
  dialogRef = inject<MatDialogRef<SettingUserLoginLogsComponent>>(MatDialogRef);
  data = inject<{
    userId: number;
    userName: string;
    nickName: string;
}>(MAT_DIALOG_DATA);
  private httpClient = inject(HttpClient);

  /** 登录日志数据源 */
  dataSource = new MatTableDataSource<LoginLog>();

  /** 表格列 */
  displayedColumns: string[] = ['loginTime', 'success', 'ip', 'deviceType', 'deviceName'];

  /** 分页参数 */
  pageNum = 1;
  pageSize = 20;
  total = 0;

  /** 页大小选项 */
  pageSizeOptions = [20, 50, 100];

  /** 加载状态 */
  isLoading = false;

  override ngOnInit(): void {
    super.ngOnInit();
    this.loadLoginLogs();
  }

  /** 加载登录日志 */
  loadLoginLogs(): void {
    this.isLoading = true;
    this.httpClient
      .post<ListRespone<LoginLog>>('/api/v1/user/loginLogs', {
        userId: this.data.userId,
        pageNum: this.pageNum,
        pageSize: this.pageSize
      })
      .pipe(takeUntil(this.$destroy), finalize(() => this.isLoading = false))
      .subscribe((res) => {
        if (res.code === 20000 && res.data?.list?.length) {
          this.dataSource.data = res.data.list;
          this.total = res.data.total;
        } else {
          this.dataSource.data = [];
          this.total = 0;
        }
      });
  }

  /** 分页变化 */
  onPaginatorChange(e: PageEvent): void {
    this.pageNum = e.pageIndex + 1;
    this.pageSize = e.pageSize;
    this.loadLoginLogs();
  }

  /** 行唯一标识 */

  /** 格式化登录时间 */
  formatLoginTime(loginTime: number): string {
    if (!loginTime) {
      return '未知';
    }
    return formatDate(loginTime, 'yyyy-MM-dd HH:mm:ss');
  }

  /** 设备类型展示 */
  deviceTypeText(deviceType: string): string {
    if (!deviceType) {
      return '-';
    }
    const map: Record<string, string> = {
      web: 'Web',
      ios: 'iOS',
      android: 'Android',
      browser_extension: '浏览器扩展'
    };
    return map[deviceType] || deviceType;
  }

  tableTrackBy(idx: number, item: { id?: number | string }) {
    return item?.id ?? idx;
  }
}

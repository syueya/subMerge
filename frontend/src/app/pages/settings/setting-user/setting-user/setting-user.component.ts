import { HttpClient } from '@angular/common/http';
import { AfterViewInit, Component, inject } from '@angular/core';
import { MatTableDataSource } from '@angular/material/table';
import { HttpRespone, ListRespone, ServiceQueryParams } from '@common/interfaces';
import { CM_DIALOG_WIDTH, CmDialogOpenService } from '@common/modules/dialog';
import { CmMessageService } from '@common/modules/message';
import { CmSharedDialogDataModel, CmSharedDialogService } from '@common/modules/shared-dialog';
import { CmParentTableComponent } from '@common/parents/parent-table/parent-table.component';
import { formatDate } from '@common/util';
import { UserRoleEnumToName } from '@data-struct';
import { takeUntil, finalize } from 'rxjs';

import { User } from '@data-struct';
import { SettingUserLoginLogsComponent } from '../setting-user-login-logs/setting-user-login-logs.component';
import { SettingUserUpdateComponent } from '../setting-user-update/setting-user-update.component';

@Component({
  selector: 'app-setting-user',
  standalone: false,
  templateUrl: './setting-user.component.html'
})

export class SettingUserComponent extends CmParentTableComponent implements AfterViewInit {
  private httpClient = inject(HttpClient);
  private dialogOpen = inject(CmDialogOpenService);
  private message = inject(CmMessageService);
  private dialogService = inject(CmSharedDialogService);
  // 初始化数据源
  dataSource = new MatTableDataSource<User>();

  // 当前选中的行
  selectedTableRow!: User;

  // 表格列名
  override displayedColumns: string[] = ['userName', 'nickName', 'role', 'lastDevice', 'action'];


  constructor() {
    super();
    //初始化记住分页大小
    this.rememberPageSize(this.constructor.name);
  }

  // 初始化后置
  override handlerAfterViewInit() {
    super.handlerAfterViewInit();  // 调用基类的 handlerAfterViewInit()
    this.reloadTableDataByFirstPage(); // 管理员加载用户列表
  }

  /* 加载表格数据 */
  override reloadTableData(query?: ServiceQueryParams): void {
    this.isLoading = true;
    this.httpClient
    .post<ListRespone<User>>('/api/v1/user/page', {...query})
      .pipe(takeUntil(this.$destroy), finalize(() => this.isLoading = false))
      .subscribe((res) => {
        if (res.code === 20000 && res.data?.list?.length) {
          this.dataSource.data = res.data.list; // 更新数据源
          this.paginatorProps.length = res.data.total; // 更新总数
          this.paginatorProps.pageIndex = res.data.pageNum-1;
          this.paginatorProps.pageSize = res.data.pageSize;
        } else {
          this.paginatorProps.length = 0;
          this.paginatorProps.pageIndex = 0;
          this.paginatorProps.pageSize = 0;
          this.dataSource.data = []; // 没有数据时，设置空数组
        }
      });
  }

  // 表格行的唯一标识

  /** 角色展示文案 */
  roleLabel(role?: number | string): string {
    return UserRoleEnumToName[Number(role)] || '-';
  }

  /** 设备类型展示文案 */
  deviceTypeText(deviceType?: string): string {
    if (!deviceType) {
      return '';
    }
    const map: Record<string, string> = {
      web: 'Web',
      ios: 'iOS',
      android: 'Android',
      browser_extension: '浏览器扩展'
    };
    return map[deviceType] || deviceType;
  }

  /** 格式化登录时间(与管理员设备管理页保持一致) */
  formatLoginTime(loginTime?: number): string {
    if (!loginTime) {
      return '';
    }
    return formatDate(loginTime, 'yyyy-MM-dd HH:mm');
  }


  // 将选中的行数据赋值给变量selectedTableRow

  // 新增和修改用户
  addAndEditUser(row: User | null = null): void {
    const data = row || null;
    const dialogRef = this.dialogOpen.openForm(SettingUserUpdateComponent, data, {
      width: CM_DIALOG_WIDTH.form
    });

    dialogRef.afterClosed().subscribe((result) => {
      if(result){
        this.reloadTableDataByFirstPage();
      }
    });
  }

  // 查看用户登录日志
  viewLoginLogs(row: User): void {
    this.dialogOpen.openContent(SettingUserLoginLogsComponent, { userId: row.id, userName: row.userName, nickName: row.nickName }, {
      width: CM_DIALOG_WIDTH.large
    });
  }

  // 删除用户
  deleteUser(row: User): void {
    this.dialogService.open({
      model: CmSharedDialogDataModel.confirm,
      title: '确认删除',
      content: `确认删除用户【${row.nickName}】？`,
      width: CM_DIALOG_WIDTH.small
    })
    .afterClosed()
    .subscribe((result) => {
      if(result){
        this.httpClient
        .delete<HttpRespone<boolean>>(`/api/v1/user/delete/${row.id}`)
        .pipe(takeUntil(this.$destroy))
        .subscribe((res) => {
          if (res.code === 20000) {
            this.message.success(`用户 【${  row.nickName  }】 删除成功`);
            this.reloadTableDataByFirstPage();
          }
        });
      }
    });
  }

  override selectTableRow(row: any) {
    this.selectedTableRow = row;
    this.activeTableDataItem = row;
  }
}

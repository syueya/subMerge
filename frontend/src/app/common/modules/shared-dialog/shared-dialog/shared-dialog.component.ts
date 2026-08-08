/* eslint-disable @typescript-eslint/no-explicit-any */
import { Component, inject } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatDialogModule, MatDialogRef, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { MatDividerModule } from '@angular/material/divider';
import { MatTooltipModule } from '@angular/material/tooltip';
import { IconsModule } from '@common/modules/icons/icons.module';
import { CmSharedDialogDataModel } from '@common/modules/shared-dialog/enum/CmSharedDialogDataModel';
import { CmSharedDialogData } from '@common/modules/shared-dialog/interfaces/CmSharedDialogData';

/**
 * 确认/提示弹窗。
 * 不要 import AppCommonModule：会与根公共模块形成运行时循环依赖（NG0919），
 * 打开 confirm 时无法读取 @Component metadata。
 * 标题区对齐 cm-dialog-header（垂直居中 + 分割线），但不直接引用该组件以免循环依赖。
 */
@Component({
  selector: 'cm-shared-dialog',
  imports: [MatDialogModule, MatButtonModule, MatDividerModule, MatTooltipModule, IconsModule],
  templateUrl: './shared-dialog.component.html'
})
export class CmSharedDialogComponent {
  dialogRef = inject<MatDialogRef<CmSharedDialogComponent>>(MatDialogRef);
  data = inject<CmSharedDialogData>(MAT_DIALOG_DATA);

  static instance: CmSharedDialogComponent;

  modelType = CmSharedDialogDataModel;
  /** 当前弹窗类型 */
  model: CmSharedDialogDataModel | string = CmSharedDialogDataModel.confirm;

  constructor() {
    const data = this.data;

    CmSharedDialogComponent.instance = this;
    if (data.model) {
      this.model = data.model;
      // 仅在调用方未指定按钮文案时填默认值，保留 confirmText 等自定义
      switch (data.model) {
        case CmSharedDialogDataModel.delete:
          data.sureStr ||= '删除';
          break;
        case CmSharedDialogDataModel.confirm:
          data.sureStr ||= '确认';
          break;
        case CmSharedDialogDataModel.info:
          data.sureStr ||= '好的';
          break;
        default:
          break;
      }
    }
  }

  onNoClick(param?: any): void {
    if (this?.dialogRef) {
      this.dialogRef.close(param);
    } else {
      CmSharedDialogComponent.instance.dialogRef.close(param);
    }
  }
}

/* eslint-disable @typescript-eslint/no-explicit-any */
import { Component, inject } from '@angular/core';
import { MatDialogRef, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { AppCommonModule } from '@common/common.module';
import { CmSharedDialogDataModel } from '@common/modules/shared-dialog/enum/CmSharedDialogDataModel';
import { CmSharedDialogData } from '@common/modules/shared-dialog/interfaces/CmSharedDialogData';

@Component({
    selector: 'cm-shared-dialog',
    imports: [AppCommonModule],
    templateUrl: './shared-dialog.component.html',
    styleUrl: './shared-dialog.component.scss'
})
export class CmSharedDialogComponent {
  dialogRef = inject<MatDialogRef<CmSharedDialogComponent>>(MatDialogRef);
  data = inject<CmSharedDialogData>(MAT_DIALOG_DATA);

  static instance: CmSharedDialogComponent;

  modelType = CmSharedDialogDataModel;
  // 当前组件类型
  model: CmSharedDialogDataModel | string = CmSharedDialogDataModel.confirm;

  constructor() {
    const data = this.data;

    CmSharedDialogComponent.instance = this;
    if (data.model) {
      this.model = data.model;
      switch (data.model) {
        case CmSharedDialogDataModel.delete:
          // 如果为删除时，修改确定按钮名称
          data.sureStr = '删除';
          break;
        case CmSharedDialogDataModel.confirm:
          // 如果为确认时，修改确定按钮名称
          data.sureStr = '确认';
          break;
        case CmSharedDialogDataModel.info:
          // 如果为提示时，修改确定按钮名称
          data.sureStr = '好的';
          break;

        default:
          break;
      }
    }
  }


  onNoClick(param?: any): void {
    if (this && this.dialogRef) {
      this.dialogRef.close(param);
    } else {
      CmSharedDialogComponent.instance.dialogRef.close(param);
    }
  }
}

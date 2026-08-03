import { Injectable, inject } from '@angular/core';
import { CmMessageService } from '@common/modules/message';
import { CmSharedDialogDataModel, CmSharedDialogService } from '@common/modules/shared-dialog';
import { firstValueFrom } from 'rxjs';

/**
 * 兼容旧 SubMerge 前端的 DialogService API。
 * 底层使用模板的 CmMessageService（短提示）与 CmSharedDialogService（确认/长文案）。
 */
@Injectable({ providedIn: 'root' })
export class DialogService {
  private readonly message = inject(CmMessageService);
  private readonly sharedDialog = inject(CmSharedDialogService);

  info(message: string, title = '提示'): Promise<void> {
    if (preferModal(message)) {
      return this.openInfo(title, message);
    }
    this.message.info(message);
    return Promise.resolve();
  }

  success(message: string, title = '成功'): Promise<void> {
    if (preferModal(message)) {
      return this.openInfo(title, message);
    }
    this.message.success(message);
    return Promise.resolve();
  }

  error(message: string, title = '错误'): Promise<void> {
    if (preferModal(message)) {
      return this.openInfo(title, message);
    }
    this.message.error(message);
    return Promise.resolve();
  }

  async confirm(message: string, title = '请确认', confirmText = '确定'): Promise<boolean> {
    const ref = this.sharedDialog.open({
      model: CmSharedDialogDataModel.confirm,
      title,
      content: message,
      sureStr: confirmText,
      cancelStr: '取消'
    });
    const result = await firstValueFrom(ref.afterClosed());
    return !!result;
  }

  private async openInfo(title: string, message: string): Promise<void> {
    const ref = this.sharedDialog.open({
      model: CmSharedDialogDataModel.info,
      title,
      content: message,
      sureStr: '知道了'
    });
    await firstValueFrom(ref.afterClosed());
  }
}

function preferModal(message: string): boolean {
  const m = message ?? '';
  if (m.includes('\n')) return true;
  if (m.length > 96) return true;
  return false;
}

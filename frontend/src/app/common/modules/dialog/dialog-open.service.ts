import { Overlay } from '@angular/cdk/overlay';
import { ComponentType } from '@angular/cdk/portal';
import { Injectable, inject } from '@angular/core';
import { MatDialog, MatDialogConfig } from '@angular/material/dialog';

/** 弹窗宽度统一用视口百分比，避免写死 px。 */
export const CM_DIALOG_WIDTH = {
  small: '40%',
  medium: '50%',
  form: '60%',
  large: '70%',
  xlarge: '75%',
  report: '80%',
  xwide: '85%',
  wide: '90%'
} as const;


@Injectable({ providedIn: 'root' })
export class CmDialogOpenService {
  private dialog = inject(MatDialog);
  private overlay = inject(Overlay);

  openForm<T, D = unknown>(component: ComponentType<T>, data: D, config: MatDialogConfig<D> = {}) {
    return this.open(component, data, {
      width: CM_DIALOG_WIDTH.form,
      disableClose: true,
      autoFocus: false,
      ...config
    });
  }

  openSmallForm<T, D = unknown>(component: ComponentType<T>, data: D, config: MatDialogConfig<D> = {}) {
    return this.openForm(component, data, {
      width: CM_DIALOG_WIDTH.small,
      ...config
    });
  }

  openLargeForm<T, D = unknown>(component: ComponentType<T>, data: D, config: MatDialogConfig<D> = {}) {
    return this.openForm(component, data, {
      width: CM_DIALOG_WIDTH.large,
      ...config
    });
  }

  /** 内容查看弹窗（非表单）：允许点遮罩/ESC 关闭 */
  openContent<T, D = unknown>(component: ComponentType<T>, data: D, config: MatDialogConfig<D> = {}) {
    return this.open(component, data, {
      width: CM_DIALOG_WIDTH.form,
      disableClose: false,
      autoFocus: false,
      ...config
    });
  }

  openWideContent<T, D = unknown>(component: ComponentType<T>, data: D, config: MatDialogConfig<D> = {}) {
    return this.openContent(component, data, {
      width: CM_DIALOG_WIDTH.report,
      ...config
    });
  }

  openNoopScrollWideContent<T, D = unknown>(component: ComponentType<T>, data: D, config: MatDialogConfig<D> = {}) {
    return this.openWideContent(component, data, {
      width: CM_DIALOG_WIDTH.wide,
      scrollStrategy: this.overlay.scrollStrategies.noop(),
      ...config
    });
  }

  openPicker<T, D = unknown>(component: ComponentType<T>, data: D, config: MatDialogConfig<D> = {}) {
    return this.open(component, data, {
      width: CM_DIALOG_WIDTH.medium,
      disableClose: false,
      autoFocus: false,
      ...config
    });
  }

  private withPanelClass<D>(config: MatDialogConfig<D>): MatDialogConfig<D> {
    const customPanelClasses = Array.isArray(config.panelClass) ? config.panelClass : config.panelClass ? [config.panelClass] : [];
    return {
      ...config,
      panelClass: ['cm-dialog-panel', ...customPanelClasses]
    };
  }

  openWithConfig<T, D = unknown>(component: ComponentType<T>, config: MatDialogConfig<D> = {}) {
    return this.dialog.open(component, this.withPanelClass(config));
  }

  open<T, D = unknown>(component: ComponentType<T>, data: D, config: MatDialogConfig<D> = {}) {
    return this.dialog.open(component, {
      ...this.withPanelClass(config),
      data
    });
  }
}

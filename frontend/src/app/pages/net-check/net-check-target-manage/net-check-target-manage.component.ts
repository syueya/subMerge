import { Component, inject, signal } from '@angular/core';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { CmParentComponent } from '@common/parents/parent/parent.component';
import { DialogService } from '@common/services/dialog.service';
import {
  NetCheckConfig,
  NetCheckTarget,
  NetCheckTargetManageDialogData,
  NetCheckTargetManageDialogResult,
} from '@data-struct';
import { takeUntil } from 'rxjs';

import { NetCheckService } from '../services/net-check.service';

@Component({
  selector: 'app-net-check-target-manage',
  templateUrl: './net-check-target-manage.component.html',
  standalone: false,
})
export class NetCheckTargetManageComponent extends CmParentComponent {
  private readonly svc = inject(NetCheckService);
  private readonly dialog = inject(DialogService);

  readonly dialogRef = inject<MatDialogRef<NetCheckTargetManageComponent, NetCheckTargetManageDialogResult | null>>(MatDialogRef);
  readonly data = inject<NetCheckTargetManageDialogData>(MAT_DIALOG_DATA);
  readonly config = signal<NetCheckConfig>(this.cloneConfig(this.data.config));
  readonly saving = signal(false);
  readonly resetting = signal(false);

  updateField(field: 'timeout' | 'autoRefresh', value: string | number): void {
    const parsed = Number(value);
    this.config.update((config) => ({ ...config, [field]: Number.isFinite(parsed) ? parsed : 0 }));
  }

  updateTarget(index: number, field: 'name' | 'url', value: string): void {
    this.config.update((config) => ({
      ...config,
      targets: config.targets.map((target, i) => (i === index ? { ...target, [field]: value } : target)),
    }));
  }

  toggleTarget(index: number, enabled: boolean): void {
    this.config.update((config) => ({
      ...config,
      targets: config.targets.map((target, i) => (i === index ? { ...target, enabled } : target)),
    }));
  }

  addTarget(): void {
    this.config.update((config) => ({
      ...config,
      targets: [...config.targets, { name: '自定义目标', url: 'https://example.com/', enabled: true }],
    }));
  }

  removeTarget(index: number): void {
    this.config.update((config) => ({ ...config, targets: config.targets.filter((_, i) => i !== index) }));
  }

  resetConfig(): void {
    if (this.resetting()) return;
    this.resetting.set(true);
    this.svc
      .reset()
      .pipe(takeUntil(this.$destroy))
      .subscribe({
        next: (config) => {
          this.config.set(this.cloneConfig(config));
          this.resetting.set(false);
          void this.dialog.success('已恢复默认检测目标');
        },
        error: (err: Error) => {
          this.resetting.set(false);
          void this.dialog.error(err.message);
        },
      });
  }

  saveConfig(): void {
    if (this.saving()) return;
    const config = this.config();
    if (config.targets.some((target) => !target.name.trim() || !target.url.trim())) {
      void this.dialog.error('请填写完整的目标名称和 URL');
      return;
    }

    this.saving.set(true);
    this.svc
      .saveConfig(config)
      .pipe(takeUntil(this.$destroy))
      .subscribe({
        next: (savedConfig) => {
          this.saving.set(false);
          void this.dialog.success('检测设置已保存');
          this.dialogRef.close({
            config: this.cloneConfig(savedConfig),
          });
        },
        error: (err: Error) => {
          this.saving.set(false);
          void this.dialog.error(err.message);
        },
      });
  }

  private cloneConfig(config: NetCheckConfig): NetCheckConfig {
    return {
      timeout: config.timeout,
      autoRefresh: config.autoRefresh,
      targets: (config.targets || []).map((target: NetCheckTarget) => ({ ...target })),
    };
  }
}

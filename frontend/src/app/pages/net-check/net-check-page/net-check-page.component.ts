import { Component, OnDestroy, OnInit, inject, signal } from '@angular/core';
import { CmDialogOpenService, CM_DIALOG_WIDTH } from '@common/modules/dialog/dialog-open.service';
import { CmParentComponent } from '@common/parents/parent/parent.component';
import { DialogService } from '@common/services/dialog.service';
import {
  BADGE_ERR,
  BADGE_MUTED,
  BADGE_OK,
  NetCheckConfig,
  NetCheckRequest,
  NetCheckResult,
  NetCheckSummary,
} from '@data-struct';
import { takeUntil } from 'rxjs';

import { NetCheckTargetManageComponent } from '../net-check-target-manage/net-check-target-manage.component';
import { NetCheckService } from '../services/net-check.service';

@Component({
  selector: 'app-net-check-page',
  templateUrl: './net-check-page.component.html',
  standalone: false,
})
export class NetCheckPageComponent extends CmParentComponent implements OnInit, OnDestroy {
  private readonly svc = inject(NetCheckService);
  private readonly dialog = inject(DialogService);
  private readonly dialogOpen = inject(CmDialogOpenService);

  readonly config = signal<NetCheckConfig>({ timeout: 10, autoRefresh: 0, targets: [] });
  readonly results = signal<NetCheckResult[]>([]);
  readonly summary = signal<NetCheckSummary | null>(null);
  readonly proxyInfo = signal('');
  readonly proxyMode = signal('');
  readonly loading = signal(false);
  readonly badgeOk = BADGE_OK;
  readonly badgeErr = BADGE_ERR;
  readonly badgeMuted = BADGE_MUTED;
  readonly displayedColumns = ['name', 'url', 'result', 'code', 'time', 'note'];

  /** 仅当前页会话有效，不落库、不随路由保留 */
  proxyEnabled = false;
  proxyURL = '';
  private autoTimer: ReturnType<typeof setInterval> | null = null;

  /** 填写代理地址时自动勾选启用；清空则取消勾选（仍可手动改勾选） */
  onProxyURLChange(value: string): void {
    this.proxyURL = value;
    this.proxyEnabled = !!value.trim();
  }

  override ngOnInit(): void {
    super.ngOnInit();
    this.loadConfig();
  }

  override ngOnDestroy(): void {
    this.clearAutoTimer();
    super.ngOnDestroy();
  }

  loadConfig(): void {
    this.svc
      .config()
      .pipe(takeUntil(this.$destroy))
      .subscribe({
        next: (config) => {
          this.config.set(this.cloneConfig(config));
          this.updateAutoTimer();
        },
        error: (err: Error) => void this.dialog.error(err.message),
      });
  }

  openTargetManage(): void {
    const ref = this.dialogOpen.openLargeForm(
      NetCheckTargetManageComponent,
      { config: this.cloneConfig(this.config()) },
      { width: CM_DIALOG_WIDTH.large },
    );

    ref.afterClosed().pipe(takeUntil(this.$destroy)).subscribe((result) => {
      if (!result) {
        this.loadConfig();
        return;
      }
      this.config.set(this.cloneConfig(result.config));
      this.updateAutoTimer();
    });
  }

  runCheck(targets?: NetCheckConfig['targets']): void {
    if (this.loading()) return;

    const request: NetCheckRequest = {
      proxy: { enabled: this.proxyEnabled, url: this.proxyURL.trim() },
      timeout: this.config().timeout,
      autoRefresh: this.config().autoRefresh,
      targets: targets || this.config().targets,
    };
    this.loading.set(true);
    this.svc
      .check(request)
      .pipe(takeUntil(this.$destroy))
      .subscribe({
next: (response) => {
                  this.results.set(response.results || []);
                  this.summary.set(response.summary);
                  this.proxyInfo.set(response.proxyInfo || '');
                  this.proxyMode.set(response.proxyMode || '');
                  this.loading.set(false);
                },
        error: (err: Error) => {
          this.loading.set(false);
          void this.dialog.error(err.message);
        },
      });
  }

  retryFailed(): void {
    const failed = this.results().filter((result) => result.status === 'FAIL');
    if (failed.length) {
      this.runCheck(failed.map((result) => ({ name: result.name, url: result.url, enabled: true })));
    }
  }

  failedCount(): number {
    return this.results().filter((result) => result.status === 'FAIL').length;
  }

  formatMs(ms: number): string {
    return ms < 1000 ? `${ms} ms` : `${(ms / 1000).toFixed(2)} s`;
  }

  private updateAutoTimer(): void {
    this.clearAutoTimer();
    const seconds = Number(this.config().autoRefresh || 0);
    if (seconds > 0) {
      this.autoTimer = setInterval(() => this.runCheck(), seconds * 1000);
    }
  }

  private clearAutoTimer(): void {
    if (this.autoTimer !== null) {
      clearInterval(this.autoTimer);
      this.autoTimer = null;
    }
  }

  private cloneConfig(config: NetCheckConfig): NetCheckConfig {
    return {
      timeout: config.timeout,
      autoRefresh: config.autoRefresh,
      targets: (config.targets || []).map((target) => ({ ...target })),
    };
  }
}

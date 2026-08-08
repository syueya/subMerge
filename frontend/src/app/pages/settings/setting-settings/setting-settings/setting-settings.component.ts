import { Component, inject, signal } from '@angular/core';
import { FormBuilder, Validators } from '@angular/forms';
import { CmParentFormComponent } from '@common/parents/parent-form/parent-form.component';
import { DialogService } from '@common/services/dialog.service';
import { publicBaseUrlValidator } from '@common/util';
import { SystemSettingsView } from '@data-struct';
import { finalize, takeUntil } from 'rxjs';

import { SystemSettingsService } from '../../services/system-settings.service';


@Component({
  selector: 'app-setting-settings',
  templateUrl: './setting-settings.component.html',
  styleUrls: ['./setting-settings.component.scss'],
  standalone: false,
})
export class SettingSettingsComponent extends CmParentFormComponent {
  private readonly fb = inject(FormBuilder);
  private readonly svc = inject(SystemSettingsService);
  private readonly dialog = inject(DialogService);

  readonly loading = signal(true);
  readonly saving = signal(false);
  readonly view = signal<SystemSettingsView | null>(null);
  readonly form = this.fb.nonNullable.group({
    sourceFetchUA: ['', Validators.required],
    sourceFetchTimeout: [30, [Validators.required, Validators.min(1)]],
    sourceMaxBytes: [5, [Validators.required, Validators.min(1)]],
    refreshInterval: [24, [Validators.required, Validators.min(1), Validators.max(720)]],
    geoipUrl: ['', Validators.required],
    geositeUrl: ['', Validators.required],
    geodbUrl: ['', Validators.required],
    geoasnUrl: ['', Validators.required],
    ipGeoUrl: ['', Validators.required],
    ipGeoTimeout: [5, [Validators.required, Validators.min(1)]],
    logOutput: ['both' as 'console' | 'file' | 'both' | 'none'],
    debugLogging: false,
    logRetentionDays: [7, [Validators.required, Validators.min(0), Validators.max(3650)]],
    proxyEnabled: false,
    proxyUrl: ['', Validators.pattern(/^(https?|socks5h?):\/\/[^\s]+$/)],
    publicBaseUrl: ['http://localhost:8080', [Validators.required, publicBaseUrlValidator()]],
    trustedProxies: [''],
    cookieSecure: false,
  });

  override handlerAfterViewInit(): void { super.handlerAfterViewInit(); this.load(); }

  load(): void {
    this.loading.set(true);
    this.svc.get().pipe(takeUntil(this.$destroy), finalize(() => this.loading.set(false))).subscribe({
      next: (view) => { this.view.set(view); this.form.patchValue({ ...view.settings, sourceMaxBytes: Math.round(view.settings.sourceMaxBytes / 1048576) }, { emitEvent: false }); this.form.patchValue({ proxyUrl: '' }, { emitEvent: false }); },
      error: (err: Error) => void this.dialog.error(err.message),
    });
  }

  save(): void {
    if (this.saving()) return;
    if (this.form.invalid) { this.form.markAllAsTouched(); void this.dialog.error('请检查系统设置中的无效字段'); return; }
    this.saving.set(true);
    const raw = this.form.getRawValue();
    const payload = { ...raw, sourceMaxBytes: raw.sourceMaxBytes * 1048576 };
    this.svc.save(payload).pipe(takeUntil(this.$destroy), finalize(() => this.saving.set(false))).subscribe({
      next: (view) => { this.view.set(view); this.form.patchValue({ ...view.settings, sourceMaxBytes: Math.round(view.settings.sourceMaxBytes / 1048576) }, { emitEvent: false }); this.form.patchValue({ proxyUrl: '' }, { emitEvent: false }); void this.dialog.success(view.restartRequired ? '系统设置已保存，可信代理配置将在重启服务后生效' : '系统设置已保存'); },
      error: (err: Error) => void this.dialog.error(err.message),
    });
  }

  reset(): void {
    if (this.saving()) return;
    void this.dialog.confirm('确定要恢复系统默认吗？当前已保存的设置将被覆盖。', '恢复系统默认').then((confirmed) => {
      if (!confirmed) return;
      this.saving.set(true);
      this.svc.reset().pipe(takeUntil(this.$destroy), finalize(() => this.saving.set(false))).subscribe({
        next: (view) => { this.view.set(view); this.form.patchValue({ ...view.settings, sourceMaxBytes: Math.round(view.settings.sourceMaxBytes / 1048576) }, { emitEvent: false }); this.form.patchValue({ proxyUrl: '' }, { emitEvent: false }); void this.dialog.success(view.restartRequired ? '系统设置已恢复，可信代理配置将在重启服务后生效' : '系统设置已恢复为默认值'); },
        error: (err: Error) => void this.dialog.error(err.message),
      });
    });
  }

  source(key: string): string { return this.view()?.source[key] === 'web' ? '网页设置' : '系统默认'; }

  /** 组装字段 tooltip：功能说明 + 当前值来源 */
  help(explanation: string, key: string): string {
    return `${explanation}`;
  }
}

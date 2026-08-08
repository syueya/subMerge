import { Component, inject, signal } from '@angular/core';
import { FormBuilder, Validators } from '@angular/forms';
import { finalize, takeUntil } from 'rxjs';
import { DialogService } from '@common/services/dialog.service';
import { CmParentFormComponent } from '@common/parents/parent-form/parent-form.component';
import { SystemSettingsService } from '../services/system-settings.service';
import { SystemSettingsView } from '@data-struct';

@Component({ selector: 'app-outbound-proxy', templateUrl: './outbound-proxy.component.html', standalone: false })
export class OutboundProxyComponent extends CmParentFormComponent {
  private readonly fb = inject(FormBuilder);
  private readonly svc = inject(SystemSettingsService);
  private readonly dialog = inject(DialogService);

  readonly loading = signal(true);
  readonly saving = signal(false);
  readonly view = signal<SystemSettingsView | null>(null);
  readonly form = this.fb.nonNullable.group({
    sourceFetchUA: ['', Validators.required],
    sourceFetchTimeout: ['30s', Validators.required],
    sourceMaxBytes: [8388608, [Validators.required, Validators.min(1024)]],
    refreshInterval: [24, [Validators.required, Validators.min(1), Validators.max(720)]],
    geoipUrl: ['', Validators.required],
    geositeUrl: ['', Validators.required],
    geodbUrl: ['', Validators.required],
    geoasnUrl: ['', Validators.required],
    ipGeoUrl: ['', Validators.required],
    ipGeoTimeout: ['5s', Validators.required],
    logOutput: ['both' as 'console' | 'file' | 'both' | 'none'],
    debugLogging: false,
    logRetentionDays: [7, [Validators.required, Validators.min(0), Validators.max(3650)]],
    proxyEnabled: false,
    proxyUrl: ['', Validators.pattern(/^(https?|socks5h?):\/\/[^\s]+$/)],
    publicBaseUrl: ['http://localhost:8080', Validators.required],
    trustedProxies: [''],
    cookieSecure: false,
  });

  override handlerAfterViewInit(): void { super.handlerAfterViewInit(); this.load(); }

  load(): void {
    this.loading.set(true);
    this.svc.get().pipe(takeUntil(this.$destroy), finalize(() => this.loading.set(false))).subscribe({
      next: (view) => { this.view.set(view); this.form.patchValue(view.settings, { emitEvent: false }); this.form.patchValue({ proxyUrl: '' }, { emitEvent: false }); },
      error: (err: Error) => void this.dialog.error(err.message),
    });
  }

  save(): void {
    if (this.saving()) return;
    if (this.form.invalid) { this.form.markAllAsTouched(); void this.dialog.error('请检查系统设置中的无效字段'); return; }
    this.saving.set(true);
    const raw = this.form.getRawValue();
    this.svc.save(raw).pipe(takeUntil(this.$destroy), finalize(() => this.saving.set(false))).subscribe({
      next: (view) => { this.view.set(view); this.form.patchValue(view.settings, { emitEvent: false }); this.form.patchValue({ proxyUrl: '' }, { emitEvent: false }); void this.dialog.success(view.restartRequired ? '系统设置已保存，可信代理配置将在重启服务后生效' : '系统设置已保存'); },
      error: (err: Error) => void this.dialog.error(err.message),
    });
  }

  reset(): void {
    if (this.saving()) return;
    this.saving.set(true);
    this.svc.reset().pipe(takeUntil(this.$destroy), finalize(() => this.saving.set(false))).subscribe({
      next: (view) => { this.view.set(view); this.form.patchValue(view.settings, { emitEvent: false }); this.form.patchValue({ proxyUrl: '' }, { emitEvent: false }); void this.dialog.success(view.restartRequired ? '系统设置已恢复，可信代理配置将在重启服务后生效' : '系统设置已恢复为默认值'); },
      error: (err: Error) => void this.dialog.error(err.message),
    });
  }

  source(key: string): string { return this.view()?.source[key] === 'web' ? '网页设置' : '系统默认'; }
}

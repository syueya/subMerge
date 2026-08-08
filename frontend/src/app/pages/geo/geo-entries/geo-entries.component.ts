import { Component, inject, signal } from '@angular/core';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { PageEvent } from '@angular/material/paginator';
import { CmParentComponent } from '@common/parents/parent/parent.component';
import { BADGE_MUTED, GeoEntriesAddRuleResult, GeoEntriesDialogData, GeoEntriesDialogResult, GeoEntryRow } from '@data-struct';

@Component({
  selector: 'app-geo-entries',
  templateUrl: './geo-entries.component.html',
  standalone: false
})
export class GeoEntriesComponent extends CmParentComponent {
  dialogRef = inject<MatDialogRef<GeoEntriesComponent, GeoEntriesDialogResult>>(MatDialogRef);
  private readonly initial = inject<GeoEntriesDialogData>(MAT_DIALOG_DATA);

  readonly badgeMuted = BADGE_MUTED;
  readonly pageSizeOptions = [20, 50, 100];

  /** 弹窗内可变状态：父组件翻页后通过 applyData 就地刷新 */
  private readonly state = signal<GeoEntriesDialogData>({ ...this.initial });

  constructor() {
    super();
  }

  /** 父组件更新当前页数据（不关弹窗） */
  applyData(data: GeoEntriesDialogData): void {
    this.state.set({
      ...data,
      onPage: data.onPage ?? this.state().onPage ?? this.initial.onPage
    });
  }

  get title(): string {
    return this.state().title || '条目';
  }

  get subtitle(): string {
    const s = this.state();
    return s.subtitle || (s.total ? `共 ${s.total} 条` : '');
  }

  get loading(): boolean {
    return !!this.state().loading;
  }

  get paginated(): boolean {
    return !!this.state().paginated;
  }

  get total(): number {
    return this.state().total || 0;
  }

  get pageSize(): number {
    return this.state().limit || 50;
  }

  get pageIndex(): number {
    const s = this.state();
    const size = s.limit || 50;
    return size > 0 ? Math.floor((s.offset || 0) / size) : 0;
  }

  get canAddRule(): boolean {
    return !this.loading && !!this.state().addRule;
  }

  get items(): GeoEntryRow[] {
    return this.state().items || [];
  }

  onPage(e: PageEvent): void {
    const cb = this.state().onPage || this.initial.onPage;
    cb?.(e.pageIndex, e.pageSize);
  }

  addRule(): void {
    const context = this.state().addRule;
    if (!context) return;
    const result: GeoEntriesAddRuleResult = { action: 'add', context };
    this.dialogRef.close(result);
  }

  close(): void {
    this.dialogRef.close(null);
  }

  trackBy = (_: number, row: GeoEntryRow) => `${row.type}|${row.value}|${row.detail || ''}`;
}

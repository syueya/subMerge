import { Component, contentChild, input, TemplateRef, TrackByFunction } from '@angular/core';

export interface ResponsiveCardField {
  label: string;
  prop?: string;
  value?: (item: unknown) => unknown;
  fullWidth?: boolean;
}

@Component({
  selector: 'cm-responsive-table-list',
  standalone: false,
  templateUrl: './responsive-table-list.component.html'
})
export class ResponsiveTableListComponent {
  /** 列表数据；空且非 loading 时展示 noDataTitle */
  readonly data = input<unknown[]>([]);
  readonly isMobile = input(false);
  readonly isLoading = input(false);
  readonly tableHeight = input('calc(100vh - 275px)');
  readonly noDataTitle = input('暂无数据');
  readonly contentOnly = input(false);
  readonly trackBy = input<TrackByFunction<any> | undefined>(undefined);
  readonly cardTitle = input<string | ((item: unknown) => unknown) | undefined>(undefined);
  readonly cardBadge = input<string | ((item: unknown) => unknown) | undefined>(undefined);
  readonly cardFields = input<ResponsiveCardField[]>([]);

  readonly mobileCardTpl = contentChild<TemplateRef<unknown>>('mobileCard');
  readonly mobileSelectTpl = contentChild<TemplateRef<unknown>>('mobileSelect');
  readonly mobileExtraTpl = contentChild<TemplateRef<unknown>>('mobileExtra');
  readonly mobileActionsTpl = contentChild<TemplateRef<unknown>>('mobileActions');
  readonly desktopTableTpl = contentChild<TemplateRef<unknown>>('desktopTable');

  trackItem(index: number, item: unknown) {
    const trackBy = this.trackBy();
    return trackBy ? trackBy(index, item) : index;
  }

  fieldValue(item: unknown, field: ResponsiveCardField) {
    if (field.value) {
      return field.value(item);
    }
    if (field.prop && item && typeof item === 'object') {
      return (item as Record<string, unknown>)[field.prop];
    }
    return '';
  }

  headerValue(item: unknown, value?: string | ((item: unknown) => unknown)) {
    if (!value) {
      return '';
    }
    if (typeof value === 'function') {
      return value(item);
    }
    if (item && typeof item === 'object') {
      return (item as Record<string, unknown>)[value] ?? value;
    }
    return value;
  }

  hasValue(value: unknown) {
    return value !== null && value !== undefined && value !== '';
  }
}

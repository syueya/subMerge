import { Component, ContentChild, Input, TemplateRef, TrackByFunction } from '@angular/core';

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
  @Input() data: unknown[] = [];
  @Input() isMobile = false;
  @Input() isLoading = false;
  @Input() tableHeight = 'calc(100vh - 275px)';
  @Input() noDataTitle = '暂无数据';
  @Input() contentOnly = false;
  @Input() trackBy?: TrackByFunction<any>;
  @Input() cardTitle?: string | ((item: unknown) => unknown);
  @Input() cardBadge?: string | ((item: unknown) => unknown);
  @Input() cardFields: ResponsiveCardField[] = [];

  @ContentChild('mobileCard', { static: false }) mobileCardTpl?: TemplateRef<unknown>;
  @ContentChild('mobileSelect', { static: false }) mobileSelectTpl?: TemplateRef<unknown>;
  @ContentChild('mobileExtra', { static: false }) mobileExtraTpl?: TemplateRef<unknown>;
  @ContentChild('mobileActions', { static: false }) mobileActionsTpl?: TemplateRef<unknown>;
  @ContentChild('desktopTable', { static: false }) desktopTableTpl?: TemplateRef<unknown>;

  trackItem(index: number, item: unknown) {
    return this.trackBy ? this.trackBy(index, item) : index;
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

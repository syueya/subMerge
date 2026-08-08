import { NgOptimizedImage } from '@angular/common';
import { LOCALE_ID, NgModule } from '@angular/core';
// Material Form Controls
import { MatAutocompleteModule } from '@angular/material/autocomplete';
import { MatButtonModule } from '@angular/material/button';
import { MatButtonToggleModule } from '@angular/material/button-toggle';
import { MatCardModule } from '@angular/material/card';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatChipsModule } from '@angular/material/chips';
import { MAT_DATE_FORMATS, MAT_DATE_LOCALE, MatRippleModule } from '@angular/material/core';
import { MAT_DIALOG_DEFAULT_OPTIONS, MatDialogModule } from '@angular/material/dialog';
import { MatDividerModule } from '@angular/material/divider';
import { MatExpansionModule } from '@angular/material/expansion';
import { MAT_FORM_FIELD_DEFAULT_OPTIONS, MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatListModule } from '@angular/material/list';
import { MatMenuModule } from '@angular/material/menu';
import { MatPaginatorIntl, MatPaginatorModule } from '@angular/material/paginator';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatRadioModule } from '@angular/material/radio';
import { MatSelectModule } from '@angular/material/select';
import { MatSidenavModule } from '@angular/material/sidenav';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatTableModule } from '@angular/material/table';
import { MatTabsModule } from '@angular/material/tabs';
import { MatToolbarModule } from '@angular/material/toolbar';
// Material Layout
import { MAT_TOOLTIP_DEFAULT_OPTIONS, MatTooltipModule } from '@angular/material/tooltip';
// Material Buttons & Indicators
// Material Popups & Modals
// Material Data tables

import { CustomPaginatorIntl } from './util/CustomPaginator';

export const MY_DATE_FORMATS = {
  parse: {
    dateInput: 'YYYY-MM-DD'
  },
  display: {
    dateInput: 'YYYY-MM-DD',
    monthYearLabel: 'YYYY MMM',
    dateA11yLabel: 'LL',
    monthYearA11yLabel: 'YYYY MMMM'
  }
};

export const MATERIAL_PROVIDERS = [
  { provide: MAT_DATE_FORMATS, useValue: MY_DATE_FORMATS },
  { provide: LOCALE_ID, useValue: 'zh' },
  { provide: MAT_DATE_LOCALE, useValue: 'zh-CN' },
  { provide: MatPaginatorIntl, useClass: CustomPaginatorIntl },
  { provide: MAT_DIALOG_DEFAULT_OPTIONS, useValue: { disableClose: false, autoFocus: false } },
  { provide: MAT_TOOLTIP_DEFAULT_OPTIONS, useValue: { touchGestures: 'auto' } },
  // 无 hint/error 时不占底部空行；有 mat-error / mat-hint 时再展开
  {
    provide: MAT_FORM_FIELD_DEFAULT_OPTIONS,
    useValue: { appearance: 'outline', subscriptSizing: 'dynamic' }
  }
];

@NgModule({
  declarations: [],
  imports: [NgOptimizedImage],
  exports: [
    MatCheckboxModule,
    MatAutocompleteModule,
    MatFormFieldModule,
    MatInputModule,
    MatRadioModule,
    MatSelectModule,
    MatMenuModule,
    MatSidenavModule,
    MatSlideToggleModule,
    MatToolbarModule,
    MatCardModule,
    MatDividerModule,
    MatExpansionModule,
    MatListModule,
    MatButtonModule,
    MatButtonToggleModule,
    MatChipsModule,
    MatProgressSpinnerModule,
    MatProgressBarModule,
    MatRippleModule,
    MatDialogModule,
    MatTooltipModule,
    MatPaginatorModule,
    MatTableModule,
    MatTabsModule,
    NgOptimizedImage
  ],
  providers: MATERIAL_PROVIDERS
})
export class MaterialModule {}

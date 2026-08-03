import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { DirectivesModule } from '@common/directives/directives.module';
import { MaterialModule } from '@common/material.module';
import { IconsModule } from '@common/modules/icons/icons.module';

import { CmCheckboxItemGroupComponent } from './checkbox-item-group/checkbox-item-group.component';
import { CmCollapsePanelComponent } from './collapse-panel/collapse-panel.component';
import { CmDialogHeaderComponent } from './dialog-header/dialog-header.component';
import { FilterActionsComponent } from './filter-actions/filter-actions.component';
import { FilterResultCountComponent } from './filter-result-count/filter-result-count.component';
import { FormFieldErrorComponent } from './form-field-error/form-field-error.component';
import { CmInputSeletedSearchComponent } from './input-seleted-search/input-seleted-search.component';
import { NoDataComponent } from './no-data/no-data.component';
import { PageToolbarComponent } from './page-toolbar/page-toolbar.component';
import { RequiredMarkComponent } from './required-mark/required-mark.component';
import { ResponsiveTableListComponent } from './responsive-table-list/responsive-table-list.component';
import { SelectWithCreateComponent } from './select-with-create/select-with-create.component';
import { SmilesPreviewComponent } from './smiles-preview/smiles-preview.component';

@NgModule({
  declarations: [
    FormFieldErrorComponent,
    RequiredMarkComponent,
    CmInputSeletedSearchComponent,
    NoDataComponent,
    CmCheckboxItemGroupComponent,
    CmCollapsePanelComponent,
    CmDialogHeaderComponent,
    ResponsiveTableListComponent,
    SmilesPreviewComponent,
    PageToolbarComponent,
    FilterActionsComponent,
    FilterResultCountComponent,
    SelectWithCreateComponent
  ],
  imports: [
    CommonModule,
    MaterialModule,
    FormsModule,
    ReactiveFormsModule,
    DirectivesModule,
    IconsModule
  ],
  exports: [
    FormFieldErrorComponent,
    RequiredMarkComponent,
    CmInputSeletedSearchComponent,
    NoDataComponent,
    CmCheckboxItemGroupComponent,
    CmCollapsePanelComponent,
    CmDialogHeaderComponent,
    ResponsiveTableListComponent,
    SmilesPreviewComponent,
    PageToolbarComponent,
    FilterActionsComponent,
    FilterResultCountComponent,
    SelectWithCreateComponent
  ],
})
export class ComponentsModule { }

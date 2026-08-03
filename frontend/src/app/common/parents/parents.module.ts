import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';

import { CmMatContorlParentComponent } from './mat-contorl-parent/mat-contorl-parent.component';
import { CmOnDestoryComponent } from './on-destory/on-destory.component';
import { CmParentComponent } from './parent/parent.component';
import { CmParentFormComponent } from './parent-form/parent-form.component';
import { CmParentTableComponent } from './parent-table/parent-table.component';

// CmParentSelectableTableComponent 仅作继承基类，不进入 declarations

@NgModule({
  declarations: [
    CmParentComponent,
    CmParentFormComponent,
    CmOnDestoryComponent,
    CmMatContorlParentComponent,
    CmParentTableComponent
  ],
  imports: [
    CommonModule
  ],
  exports: [
    CmParentComponent,
    CmParentFormComponent,
    CmOnDestoryComponent,
    CmMatContorlParentComponent,
    CmParentTableComponent
  ]
})
export class ParentsModule { }

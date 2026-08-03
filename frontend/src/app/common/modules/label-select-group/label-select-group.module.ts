import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { ReactiveFormsModule } from '@angular/forms';
import {MatChipsModule} from '@angular/material/chips';
import { ComponentsModule } from '@common/components/components.module';
import { MaterialModule } from '@common/material.module';

import { LabelSelectGroupComponent } from './label-select-group/label-select-group.component';
@NgModule({
  declarations: [LabelSelectGroupComponent],
  imports: [
    CommonModule,
    MatChipsModule,
    ReactiveFormsModule,
    ComponentsModule,
    MaterialModule
  ],
  exports: [LabelSelectGroupComponent]
})
export class LabelSelectGroupModule { }

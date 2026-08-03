import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { MaterialModule } from '@common/material.module';

import { CmFormFieldComponent } from './cm-form-field/cm-form-field.component';
import { IconsModule } from '../icons/icons.module';


@NgModule({
  declarations: [CmFormFieldComponent],
  imports: [
    CommonModule,MaterialModule,IconsModule
  ],
  exports:[CmFormFieldComponent]
})
export class FormLayoutModule { }

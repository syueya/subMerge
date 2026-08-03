/* eslint-disable @typescript-eslint/no-explicit-any */
import { Component, Input, TemplateRef } from '@angular/core';

@Component({
    selector: 'cm-dialog-header',
    templateUrl: './dialog-header.component.html',
    standalone: false
})
export class CmDialogHeaderComponent {
  @Input() modalTitle = '';

  
  @Input()
  titleTemplateRef!: TemplateRef<any>

}

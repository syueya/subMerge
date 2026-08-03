import { Injectable, inject } from '@angular/core';
import { CM_DIALOG_WIDTH, CmDialogOpenService } from '@common/modules/dialog';
import { CmSharedDialogData } from '@common/modules/shared-dialog/interfaces/CmSharedDialogData';
import { CmSharedDialogComponent } from '@common/modules/shared-dialog/shared-dialog/shared-dialog.component';


@Injectable({
  providedIn: 'root'
})
export class CmSharedDialogService {
  private dialogOpen = inject(CmDialogOpenService);


  open(data:CmSharedDialogData){
   return this.dialogOpen.open(CmSharedDialogComponent, data, {
      width: data?.width || CM_DIALOG_WIDTH.small
    });
  }
}

import { Injectable, inject } from '@angular/core';
import { CM_DIALOG_WIDTH, CmDialogOpenService } from '@common/modules/dialog';
import { DialogService } from '@common/services/dialog.service';
import { RuleFormDialogData } from '@data-struct';
import { forkJoin, take } from 'rxjs';

import { DraftStatusStore } from '../releases/services/draft-status.store';
import { RuleFormComponent } from '../rules/rule-form/rule-form.component';
import { defaultRuleTarget } from '../rules/services/rule-ui';
import { RuleService } from '../rules/services/rule.service';

@Injectable({ providedIn: 'root' })
export class RuleCreateService {
  private readonly ruleService = inject(RuleService);
  private readonly dialogOpen = inject(CmDialogOpenService);
  private readonly draftStore = inject(DraftStatusStore);
  private readonly dialog = inject(DialogService);

  open(context: { type: 'GEOSITE' | 'GEOIP'; payload: string }): void {
    forkJoin({
      rules: this.ruleService.listRules(),
      groups: this.ruleService.listGroups()
    })
      .pipe(take(1))
      .subscribe({
        next: ({ rules, groups }) => this.openForm(rules.items || [], groups.items || [], context),
        error: (e: Error) => void this.dialog.error(e.message)
      });
  }

  private openForm(rules: RuleFormDialogData['rules'], groups: RuleFormDialogData['groups'], context: { type: 'GEOSITE' | 'GEOIP'; payload: string }): void {
    const data: RuleFormDialogData = {
      rule: null,
      groups,
      rules,
      extraCategories: [],
      defaultTarget: defaultRuleTarget(groups.map(group => group.name)) || '直连',
      defaultType: context.type,
      defaultPayload: context.payload,
      defaultCategory: 'GEO'
    };
    const ref = this.dialogOpen.openForm(RuleFormComponent, data, {
      width: CM_DIALOG_WIDTH.form
    });
    ref.afterClosed().subscribe(result => {
      if (result === false || result === undefined) return;
      this.draftStore.refresh();
    });
  }
}

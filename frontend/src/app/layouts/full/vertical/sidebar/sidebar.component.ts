import {
  Component,
  EventEmitter,
  Input,
  Output,
} from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatTooltipModule } from '@angular/material/tooltip';
import { IconsModule } from '@common/modules/icons/icons.module';

import { BrandingComponent } from './branding.component';

@Component({
    selector: 'app-sidebar',
    imports: [BrandingComponent, IconsModule, MatButtonModule, MatTooltipModule],
    templateUrl: './sidebar.component.html'
})
export class SidebarComponent {
  constructor() { }
  @Input() showToggle = true;
  @Output() readonly toggleMobileNav = new EventEmitter<void>();
  @Output() readonly toggleCollapsed = new EventEmitter<void>();
}
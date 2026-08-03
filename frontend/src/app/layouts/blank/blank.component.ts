import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { MatSidenavModule } from '@angular/material/sidenav';
import { RouterOutlet } from '@angular/router';

import { CoreService } from '../../common/services/core.service';
import { AppSettings } from '../../config';

@Component({
    selector: 'app-blank',
    templateUrl: './blank.component.html',
    styleUrls: [],
    imports: [RouterOutlet, MatSidenavModule, CommonModule]
})
export class BlankComponent {
  private settings = inject(CoreService);

  private htmlElement!: HTMLHtmlElement;

  options: AppSettings;

  constructor() {
    this.options = this.settings.getOptions();
    this.htmlElement = document.querySelector('html')!;
    this.applyTheme(this.options);
  }

  /**
   * 应用浅/深色与主题色（主题色 class 挂在 html，不在容器上重复绑定）
   */
  applyTheme(options: AppSettings): void {
    this.options = options;
    if (options.theme === 'dark') {
      this.htmlElement.classList.add('dark-theme');
      this.htmlElement.classList.remove('light-theme');
    } else {
      this.htmlElement.classList.remove('dark-theme');
      this.htmlElement.classList.add('light-theme');
    }

    this.htmlElement.classList.forEach(className => {
      if (className.endsWith('_theme')) {
        this.htmlElement.classList.remove(className);
      }
    });
    if (options.activeTheme) {
      this.htmlElement.classList.add(options.activeTheme);
    }
  }
}

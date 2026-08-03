import { CommonModule } from '@angular/common';
import { Component } from '@angular/core';
import { environment } from '@env/environment';

@Component({
    selector: 'app-app-version',
    imports: [CommonModule],
    templateUrl: './app-version.component.html',
    styleUrl: './app-version.component.scss'
})
export class AppVersionComponent {
  /**
   * 当前版本号
   */
  currentVersion: string = environment.version;

  /**
   * 是否有新版本
   */
  ishasNewVersion = false;

  constructor() {
  }

}

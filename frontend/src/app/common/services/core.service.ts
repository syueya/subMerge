import { Injectable, signal } from '@angular/core';

import { AppSettings, defaults } from '../../config';

@Injectable({
  providedIn: 'root'
})
export class CoreService {
  private optionsSignal = signal<AppSettings>(defaults);

  constructor() {
    // 初始化时从localStorage加载状态
    this.loadStateFromLocalStorage();
  }

  getOptions() {
    return this.optionsSignal();
  }

  private loadStateFromLocalStorage(): void {
    const actThemeOptionStr = localStorage.getItem('wt_current_theme_option');
    if (actThemeOptionStr) {
      let options: Partial<AppSettings> | null = null;
      try {
        options = JSON.parse(actThemeOptionStr) as Partial<AppSettings>;
      } catch (error) {
        console.error('从本地存储解析主题自定义设置时出错:', error);
        options = null;
      }
      if (options) {
        this.setOptions(options);
      }
    }
  }

  /** 只保留当前 AppSettings 字段，丢弃历史 localStorage 中的 navPos 等废弃项 */
  private normalizeOptions(options: Partial<AppSettings>): AppSettings {
    return {
      theme: options.theme ?? defaults.theme,
      sidenavOpened: options.sidenavOpened ?? defaults.sidenavOpened,
      sidenavCollapsed: options.sidenavCollapsed ?? defaults.sidenavCollapsed,
      activeTheme: options.activeTheme ?? defaults.activeTheme
    };
  }

  setOptions(options: Partial<AppSettings>) {
    this.optionsSignal.update(current => this.normalizeOptions({
      ...current,
      ...options
    }));
    // 保存主题配置
    localStorage.setItem('wt_current_theme_option', JSON.stringify(this.optionsSignal()));
  }
}

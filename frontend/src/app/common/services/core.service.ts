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
      let options = null;
      try {
        options = JSON.parse(actThemeOptionStr) as AppSettings;
      } catch (error) {
        console.error('从本地存储解析主题自定义设置时出错:', error);
        options = null;
      }
      this.setOptions(options!);
    }
  }

  setOptions(options: Partial<AppSettings>) {
    this.optionsSignal.update(current => ({
      ...current,
      ...options
    }));
    // 保存主题配置
    localStorage.setItem('wt_current_theme_option', JSON.stringify(this.optionsSignal()));
  }
}

import { BreakpointObserver, BreakpointState } from '@angular/cdk/layout';
import { Injectable, inject } from '@angular/core';
import { BehaviorSubject, Observable } from 'rxjs';
import { distinctUntilChanged, filter, map } from 'rxjs/operators';

/**
 * 屏幕尺寸枚举
 * 定义了不同屏幕尺寸的名称
 */
export enum ScreenSize {
  XSmall = 'XSmall', // 超小屏幕
  Small = 'Small',   // 小屏幕
  Medium = 'Medium', // 中屏幕
  Large = 'Large',   // 大屏幕
  XLarge = 'XLarge', // 超大屏幕
  XXLarge = 'XXLarge', // 超超大屏幕
  XXXLarge = 'XXXLarge' // 超超超大屏幕
}

/**
 * 屏幕尺寸信息接口
 * 包含屏幕尺寸枚举值、是否为手机屏幕以及屏幕宽度
 */
export interface ScreenSizeInfo {
  size: ScreenSize; // 当前屏幕尺寸
  isMobile: boolean; // 是否为手机屏幕
  width: number; // 当前屏幕宽度
}

/**
 * 屏幕断点定义
 * 定义了不同屏幕尺寸的媒体查询条件
 */
const BREAKPOINTS = {
  XSmall: '(max-width: 599px)',
  Small: '(min-width: 600px) and (max-width: 959px)',
  Medium: '(min-width: 960px) and (max-width: 1279px)',
  Large: '(min-width: 1280px) and (max-width: 1699px)',
  XLarge: '(min-width: 1700px) and (max-width: 2099px)',
  XXLarge: '(min-width: 2100px) and (max-width: 2999px)',
  XXXLarge: '(min-width: 3000px)'
};

/**
 * 屏幕尺寸服务
 * 用于监听屏幕尺寸变化并提供屏幕尺寸信息
 */
@Injectable({
  providedIn: 'root'
})
export class ScreenSizeService {
  private breakpointObserver = inject(BreakpointObserver);

  private screenSizeSubject = new BehaviorSubject<ScreenSizeInfo | null>(null); // 屏幕尺寸的行为主题
  screenSize$: Observable<ScreenSizeInfo> = this.screenSizeSubject.asObservable().pipe(filter((v): v is ScreenSizeInfo => v !== null), distinctUntilChanged()); // 屏幕尺寸的可观察对象

  constructor() {
    // 监听屏幕断点变化
    this.breakpointObserver
      .observe(Object.values(BREAKPOINTS))
      .pipe(
        map((state: BreakpointState) => {
          const width = window.innerWidth; // 获取当前屏幕宽度
          let size: ScreenSize = ScreenSize.XSmall; // 默认屏幕尺寸
          let isMobile = false; // 默认非手机屏幕

          // 根据断点状态判断当前屏幕尺寸
          if (state.breakpoints[BREAKPOINTS.XSmall]) {
            size = ScreenSize.XSmall;
            isMobile = true;
          } else if (state.breakpoints[BREAKPOINTS.Small]) {
            size = ScreenSize.Small;
            isMobile = true;
          } else if (state.breakpoints[BREAKPOINTS.Medium]) {
            size = ScreenSize.Medium;
          } else if (state.breakpoints[BREAKPOINTS.Large]) {
            size = ScreenSize.Large;
          } else if (state.breakpoints[BREAKPOINTS.XLarge]) {
            size = ScreenSize.XLarge;
          } else if (state.breakpoints[BREAKPOINTS.XXLarge]) {
            size = ScreenSize.XXLarge;
          } else if (state.breakpoints[BREAKPOINTS.XXXLarge]) {
            size = ScreenSize.XXXLarge;
          }

          return { size, isMobile, width }; // 返回屏幕尺寸信息
        })
      )
      .subscribe(info => {
        if (info) {
          this.screenSizeSubject.next(info); // 更新屏幕尺寸信息
        }
      });
  }
}

import { Injectable } from '@angular/core';
import { Subject } from 'rxjs';

export interface ScrollEvent {
  scrollTop: number; // 滚动条距离顶部的距离
  scrollHeight: number; // 滚动区域的总高度
  clientHeight: number; // 可视区域的高度
}

/**
 * 窗口滚动监听服务
 * 提供滚动事件的通知和触发返回顶部的功能
 */
@Injectable({
  providedIn: 'root'
})
export class ScrollService {
  private scrollSubject = new Subject<ScrollEvent>(); // 滚动事件的 Subject
  scroll$ = this.scrollSubject.asObservable(); // 滚动事件的可观察对象

  private triggerToTopSubject = new Subject<void>(); // 返回顶部事件的 Subject
  triggerToTop$ = this.triggerToTopSubject.asObservable(); // 返回顶部事件的可观察对象

  /**
   * 通知滚动事件
   * @param position 滚动事件的位置信息
   */
  notifyScroll(position: ScrollEvent) {
    this.scrollSubject.next(position);
  }

  /**
   * 触发返回顶部事件
   */
  triggerToTop() {
    this.triggerToTopSubject.next();
  }
}

/**
 * 该类为 SpinnerVisibilityService 类，用于管理加载动画的显示与隐藏
 */
import { Injectable, inject } from '@angular/core';
import { Observable, ReplaySubject } from 'rxjs';

import { PendingRequestsInterceptor } from './pending-requests-interceptor.service';

@Injectable({
    providedIn: 'root'
})
export class SpinnerVisibilityService {
    private pendingRequestsInterceptor = inject(PendingRequestsInterceptor);


    private _visibility$ = new ReplaySubject<boolean>(1);

    // 获取加载动画显示状态的可观察对象
    get visibility$(): Observable<boolean> {
        return this._visibility$.asObservable();
    }

    // 显示加载动画
    show(): void {
        this.pendingRequestsInterceptor.forceByPass = true; // 强制绕过拦截器处理
        this._visibility$.next(true); // 设置加载动画显示状态为true
    }

    // 隐藏加载动画
    hide(): void {
        this._visibility$.next(false); // 设置加载动画显示状态为false
        this.pendingRequestsInterceptor.forceByPass = false; // 取消强制绕过拦截器处理
    }
}
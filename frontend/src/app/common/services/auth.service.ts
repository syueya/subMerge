import { Injectable, inject } from '@angular/core';
import { Router } from '@angular/router';
import { AvatarList } from '@common/data/avatar-list';
import { UserInfo } from '@common/interfaces/UserInfo';
import { CmMessageService } from '@common/modules/message';
import { SessionHttpCacheService } from '../net/session-http-cache';
import { STORAGE_USERINFO_KEY } from '@common/util';
import { BehaviorSubject, Observable, lastValueFrom, map, tap } from 'rxjs';

import { ApiService } from '../net/api.service';
import { AdminUser } from '@data-struct';

/**
 * SubMerge 鉴权（单管理员）：
 * - 真实会话在 HttpOnly Cookie `submerge_session`（JS 读不到）
 * - 本地仅缓存非敏感 AdminUser，供 UI/Guard 快速判断
 * - token 为 Cookie 会话占位，不向 Authorization 注入真实值
 */
@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private message = inject(CmMessageService);
  private router = inject(Router);
  private sessionHttpCache = inject(SessionHttpCacheService);
  private api = inject(ApiService);

  /** Cookie 会话占位，满足模板 guard 对 token 的检查 */
  private static readonly COOKIE_TOKEN = 'cookie-session';

  private token: string | null = null;
  private userInfo: UserInfo | null = null;
  private userInfoSubject = new BehaviorSubject<UserInfo | null>(null);
  private readonly stateReady: Promise<void>;

  constructor() {
    this.stateReady = this.loadStateFromLocalStorage();
  }

  waitForState(): Promise<void> {
    return this.stateReady;
  }

  get $userInfo(): Observable<UserInfo | null> {
    return this.userInfoSubject.asObservable();
  }

  private async loadStateFromLocalStorage(): Promise<void> {
    const userInfoStr = localStorage.getItem(STORAGE_USERINFO_KEY);
    if (userInfoStr) {
      try {
        this.userInfo = this.processUserInfo(JSON.parse(userInfoStr) as UserInfo);
        this.validateUserInfo(this.userInfo);
        this.token = AuthService.COOKIE_TOKEN;
        this.userInfoSubject.next(this.userInfo);
      } catch (error) {
        console.error('从本地存储解析userInfo时出错:', error);
        this.userInfo = null;
        this.token = null;
        localStorage.removeItem(STORAGE_USERINFO_KEY);
      }
    }
  }

  getToken(): string | null {
    // Cookie 会话：有用户缓存时返回占位，不向 Authorization 注入真实 token
    return this.token;
  }

  setToken(token: string | null): void {
    this.token = token;
  }

  getUserInfo(): UserInfo | null {
    return this.userInfo;
  }

  setUserInfo(userInfo: UserInfo | null): void {
    const normalizedUserInfo = userInfo ? this.processUserInfo(userInfo) : null;
    this.userInfo = normalizedUserInfo;
    this.userInfoSubject.next(normalizedUserInfo);
    if (normalizedUserInfo) {
      localStorage.setItem(STORAGE_USERINFO_KEY, JSON.stringify(normalizedUserInfo));
      this.token = AuthService.COOKIE_TOKEN;
    } else {
      localStorage.removeItem(STORAGE_USERINFO_KEY);
      this.token = null;
    }
  }

  /** 将后端 AdminUser 写入会话缓存 */
  setSessionFromAdmin(user: AdminUser): void {
    this.setUserInfo(this.adminToUserInfo(user));
  }

  async refreshUserInfo(): Promise<void> {
    try {
      const data = await lastValueFrom(this.api.get<{ user: AdminUser }>('/auth/me'));
      if (data?.user) {
        this.setSessionFromAdmin(data.user);
      }
    } catch (error) {
      console.error('刷新用户信息时发生错误:', error);
      this.message.error('无法获取用户信息，请稍后再试。');
    }
  }

  setupStatus(): Observable<{ needsSetup: boolean }> {
    return this.api.get<{ needsSetup: boolean }>('/auth/setup-status');
  }

  bootstrap(body: { username: string; password: string; displayName?: string }): Observable<AdminUser> {
    return this.api.post<{ user: AdminUser }>('/auth/bootstrap', body).pipe(
      tap(data => this.setSessionFromAdmin(data.user)),
      map(data => data.user)
    );
  }

  login(username: string, password: string): Observable<AdminUser> {
    return this.api.post<{ user: AdminUser }>('/auth/login', { username, password }).pipe(
      tap(data => this.setSessionFromAdmin(data.user)),
      map(data => data.user)
    );
  }

  changePassword(oldPassword: string, newPassword: string): Observable<{ success: boolean }> {
    return this.api.post<{ success: boolean }>('/auth/password', { oldPassword, newPassword });
  }

  updateProfile(body: { username?: string; displayName?: string; avatar?: string }): Observable<AdminUser> {
    return this.api.put<{ user: AdminUser }>('/auth/profile', body).pipe(
      tap(data => this.setSessionFromAdmin(data.user)),
      map(data => data.user)
    );
  }

  private processUserInfo(userInfo: UserInfo): UserInfo {
    if (userInfo.avatar && !userInfo.avatar.startsWith('data:') && !userInfo.avatar.startsWith('http') && !userInfo.avatar.startsWith('/')) {
      const avatar = AvatarList.find(item => item.value === userInfo.avatar);
      userInfo.avatarSrc = avatar?.src || '/assets/images/avatar/default.jpg';
    } else if (userInfo.avatar) {
      userInfo.avatarSrc = userInfo.avatar;
    } else {
      userInfo.avatarSrc = '/assets/images/avatar/default.jpg';
    }
    return userInfo;
  }

  clear(): void {
    this.setToken(null);
    this.setUserInfo(null);
    this.sessionHttpCache.clear();
    this.router.navigate(['/auth/login']);
  }

  /** 仅清本地会话，不跳转（供 401 拦截器使用） */
  clearSessionOnly(): void {
    this.setToken(null);
    this.setUserInfo(null);
    this.sessionHttpCache.clear();
  }

  async logout(tips = true): Promise<void> {
    try {
      if (this.userInfo) {
        await lastValueFrom(this.api.post<{ success: boolean }>('/auth/logout', {}));
      }
      if (tips) {
        this.message.success('当前设备已退出登录！');
      }
    } catch (error) {
      console.error('登出时发生错误:', error);
      // 即便接口失败也清本地
    }
    this.clear();
  }

  isLoggedIn(): boolean {
    return !!this.userInfo;
  }

  displayName(): string {
    const u = this.userInfo;
    return (u?.nickName || u?.userName || 'admin').trim() || 'admin';
  }

  private validateUserInfo(userInfo: UserInfo): void {
    if (!userInfo.id || !userInfo.userName) {
      throw new Error('用户信息不完整');
    }
  }

  private adminToUserInfo(user: AdminUser): UserInfo {
    return {
      id: user.id,
      userName: user.username,
      nickName: user.displayName || user.username,
      avatar: user.avatar || '',
      createdAt: Date.parse(user.createdAt) || Date.now(),
      updatedAt: Date.now(),
      avatarSrc: user.avatar || '/assets/images/avatar/default.jpg'
    };
  }
}

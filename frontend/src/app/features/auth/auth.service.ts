import { Injectable, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { Observable, map, tap } from 'rxjs';
import { ApiService } from '../../core/api.service';
import { AdminUser, LoginResponse } from '../../common/types';

// 会话令牌只存于 HttpOnly cookie（submerge_session），JS 读不到，可抵御 XSS 窃取。
// 此处仅缓存非敏感的用户资料用于快速渲染 UI；登录态以是否有 user 判断，真正鉴权靠 cookie。
const USER_KEY = 'submerge_user';

@Injectable({ providedIn: 'root' })
export class AuthService {
	private readonly api = inject(ApiService);
	private readonly router = inject(Router);

	readonly user = signal<AdminUser | null>(this.readUser());

	setupStatus(): Observable<{ needsSetup: boolean }> {
		return this.api.get<{ needsSetup: boolean }>('/auth/setup-status');
	}

	bootstrap(body: { username: string; password: string; displayName?: string }): Observable<LoginResponse> {
		return this.api
			.post<LoginResponse>('/auth/bootstrap', body)
			.pipe(tap((data) => this.setSession(data.user)));
	}

	login(username: string, password: string): Observable<LoginResponse> {
		return this.api
			.post<LoginResponse>('/auth/login', { username, password })
			.pipe(tap((data) => this.setSession(data.user)));
	}

	logout(): void {
		if (this.user()) {
			// 会话 cookie 由浏览器随请求自动带上（withCredentials）
			this.api.post('/auth/logout', {}).subscribe({ error: () => undefined });
		}
		this.clearSession();
		void this.router.navigateByUrl('/login');
	}

	me(): Observable<AdminUser> {
		return this.api.get<{ user: AdminUser }>('/auth/me').pipe(
			tap((data) => this.setSession(data.user)),
			map((data) => data.user),
		);
	}

	changePassword(oldPassword: string, newPassword: string): Observable<{ success: boolean }> {
		return this.api.post<{ success: boolean }>('/auth/password', { oldPassword, newPassword });
	}

	updateProfile(body: { username?: string; displayName?: string; avatar?: string }): Observable<AdminUser> {
		return this.api.put<{ user: AdminUser }>('/auth/profile', body).pipe(
			tap((data) => this.setSession(data.user)),
			map((data) => data.user),
		);
	}

	/** 展示名：昵称优先，否则用户名 */
	displayName(): string {
		const u = this.user();
		return (u?.displayName || u?.username || 'admin').trim() || 'admin';
	}

	/** 头像字母：取展示名首字 */
	avatarLetter(): string {
		return this.displayName().slice(0, 1).toUpperCase();
	}

	isLoggedIn(): boolean {
		return !!this.user();
	}

	/**
	 * 记录会话：鉴权凭据是后端下发的 HttpOnly cookie（JS 读不到，抗 XSS 窃取）。
	 * 这里只缓存非敏感的用户资料到 localStorage，用于刷新后快速渲染 UI。
	 */
	setSession(user: AdminUser): void {
		localStorage.setItem(USER_KEY, JSON.stringify(user));
		this.user.set(user);
	}

	clearSession(): void {
		localStorage.removeItem(USER_KEY);
		this.user.set(null);
	}

	private readUser(): AdminUser | null {
		const raw = localStorage.getItem(USER_KEY);
		if (!raw) return null;
		try {
			return JSON.parse(raw) as AdminUser;
		} catch {
			return null;
		}
	}
}

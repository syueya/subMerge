import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Router } from '@angular/router';
import { Observable, catchError, map, tap, throwError } from 'rxjs';
import { AdminUser, ApiResponse, LoginResponse } from '../../common/types';

const TOKEN_KEY = 'submerge_token';
const USER_KEY = 'submerge_user';

@Injectable({ providedIn: 'root' })
export class AuthService {
	private readonly http = inject(HttpClient);
	private readonly router = inject(Router);

	readonly token = signal<string | null>(localStorage.getItem(TOKEN_KEY));
	readonly user = signal<AdminUser | null>(this.readUser());

	setupStatus(): Observable<{ needsSetup: boolean }> {
		return this.http.get<ApiResponse<{ needsSetup: boolean }>>('/api/auth/setup-status').pipe(
			map((r) => {
				if (!r.ok || !r.data) {
					throw new Error(r.error?.message || 'setup status failed');
				}
				return r.data;
			}),
			catchError((err) => throwError(() => new Error(this.errorMessage(err, 'setup status failed')))),
		);
	}

	bootstrap(body: {
		username: string;
		password: string;
		displayName?: string;
	}): Observable<LoginResponse> {
		return this.http.post<ApiResponse<LoginResponse>>('/api/auth/bootstrap', body).pipe(
			map((r) => {
				if (!r.ok || !r.data) {
					throw new Error(r.error?.message || 'setup failed');
				}
				return r.data;
			}),
			tap((data) => this.setSession(data.token, data.user)),
			catchError((err) => throwError(() => new Error(this.errorMessage(err, 'setup failed')))),
		);
	}

	login(username: string, password: string): Observable<LoginResponse> {
		return this.http.post<ApiResponse<LoginResponse>>('/api/auth/login', { username, password }).pipe(
			map((r) => {
				if (!r.ok || !r.data) {
					throw new Error(r.error?.message || 'login failed');
				}
				return r.data;
			}),
			tap((data) => this.setSession(data.token, data.user)),
			catchError((err) => throwError(() => new Error(this.errorMessage(err, 'login failed')))),
		);
	}

	logout(): void {
		if (this.token()) {
			// Authorization 由 authInterceptor 注入
			this.http.post('/api/auth/logout', {}).subscribe({ error: () => undefined });
		}
		this.clearSession();
		void this.router.navigateByUrl('/login');
	}

	me(): Observable<AdminUser> {
		return this.http.get<ApiResponse<{ user: AdminUser }>>('/api/auth/me').pipe(
			map((r) => {
				if (!r.ok || !r.data) {
					throw new Error(r.error?.message || 'unauthorized');
				}
				this.user.set(r.data.user);
				localStorage.setItem(USER_KEY, JSON.stringify(r.data.user));
				return r.data.user;
			}),
			catchError((err) => throwError(() => new Error(this.errorMessage(err, 'unauthorized')))),
		);
	}

	changePassword(oldPassword: string, newPassword: string): Observable<{ success: boolean }> {
		return this.http
			.post<ApiResponse<{ success: boolean }>>('/api/auth/password', {
				oldPassword,
				newPassword,
			})
			.pipe(
				map((r) => {
					if (!r.ok || !r.data) {
						throw new Error(r.error?.message || 'change password failed');
					}
					return r.data;
				}),
				catchError((err) => throwError(() => new Error(this.errorMessage(err, 'change password failed')))),
			);
	}

	updateProfile(body: {
		username?: string;
		displayName?: string;
		avatar?: string;
	}): Observable<AdminUser> {
		return this.http.put<ApiResponse<{ user: AdminUser }>>('/api/auth/profile', body).pipe(
			map((r) => {
				if (!r.ok || !r.data) {
					throw new Error(r.error?.message || 'update profile failed');
				}
				this.user.set(r.data.user);
				localStorage.setItem(USER_KEY, JSON.stringify(r.data.user));
				return r.data.user;
			}),
			catchError((err) => throwError(() => new Error(this.errorMessage(err, 'update profile failed')))),
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
		return !!this.token();
	}

	setSession(token: string, user: AdminUser): void {
		localStorage.setItem(TOKEN_KEY, token);
		localStorage.setItem(USER_KEY, JSON.stringify(user));
		this.token.set(token);
		this.user.set(user);
	}

	clearSession(): void {
		localStorage.removeItem(TOKEN_KEY);
		localStorage.removeItem(USER_KEY);
		this.token.set(null);
		this.user.set(null);
	}

	private errorMessage(err: unknown, fallback: string): string {
		const response = err as {
			error?: { error?: { message?: string }; message?: string };
			message?: string;
		};
		return response.error?.error?.message || response.error?.message || response.message || fallback;
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

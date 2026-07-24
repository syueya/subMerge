import { Component, ElementRef, HostListener, OnDestroy, OnInit, inject, signal, viewChild } from '@angular/core';
	import { FormsModule } from '@angular/forms';
	import { Router, RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
	import { DialogService } from '../../common/dialog/dialog.service';
	import { HealthResponse } from '../../common/types';
	import { ApiService } from '../../core/api.service';
	import { ThemeMode, ThemeService } from '../../core/theme.service';
	import { AuthService } from '../auth/auth.service';
	
	@Component({
		selector: 'app-shell',
		standalone: true,
		imports: [RouterOutlet, RouterLink, RouterLinkActive, FormsModule],
		templateUrl: './shell.component.html',
		styleUrl: './shell.component.scss',
	})
	export class ShellComponent implements OnInit, OnDestroy {
		readonly auth = inject(AuthService);
		readonly theme = inject(ThemeService);
		private readonly api = inject(ApiService);
		private readonly dialog = inject(DialogService);
		private readonly router = inject(Router);
	
		private readonly menuRoot = viewChild<ElementRef<HTMLElement>>('userMenuRoot');
	
		// 与后端 isAllowedAvatarDataURL 对齐：仅位图，拒绝 svg（防 XSS）
		private static readonly avatarMime = new Set([
			'image/png',
			'image/jpeg',
			'image/webp',
			'image/gif',
		]);
	
		menuOpen = signal(false);
		profileOpen = signal(false);
		busy = signal(false);
		appVersion = signal('');

	// profile form
	formUsername = '';
	formDisplayName = '';
	formOldPassword = '';
	formNewPassword = '';
	formNewPassword2 = '';
	formAvatarPreview = '';
	avatarDirty = false;

nav = [
			{ path: '/home', label: '概览' },
			{ path: '/sources', label: '订阅源' },
			{ path: '/groups', label: '策略组' },
			{ path: '/rules', label: '分流规则' },
			{ path: '/releases', label: '发布' },
			{ path: '/tokens', label: '令牌' },
		];

		ngOnInit(): void {
			// 刷新后用服务端资料覆盖 localStorage；401 由 interceptor 处理，其它错误忽略
			if (this.auth.isLoggedIn()) {
				this.auth.me().subscribe({ error: () => undefined });
			}
			// 顶部 logo 旁展示应用版本（失败则不显示）
			this.api.get<HealthResponse>('/health').subscribe({
				next: (h) => {
					const v = (h.version || '').trim();
					if (v) this.appVersion.set(v);
				},
				error: () => undefined,
			});
		}

	ngOnDestroy(): void {
		// 弹窗开启时若路由离开，避免 body 残留 overflow:hidden
		this.lockBodyScroll(false);
	}

	@HostListener('document:click', ['$event'])
	onDocClick(ev: MouseEvent): void {
		if (!this.menuOpen()) return;
		const root = this.menuRoot()?.nativeElement;
		if (root && !root.contains(ev.target as Node)) {
			this.menuOpen.set(false);
		}
	}

	@HostListener('document:keydown.escape')
	onEsc(): void {
		if (this.profileOpen()) {
			this.closeProfile();
			return;
		}
		if (this.menuOpen()) {
			this.menuOpen.set(false);
		}
	}

	toggleMenu(ev: Event): void {
		ev.stopPropagation();
		this.menuOpen.update((v) => !v);
	}

	closeMenu(): void {
		this.menuOpen.set(false);
	}

	setTheme(mode: ThemeMode): void {
		this.theme.setTheme(mode);
	}

	openProfile(): void {
		const u = this.auth.user();
		this.formUsername = u?.username || '';
		this.formDisplayName = u?.displayName || u?.username || '';
		this.formOldPassword = '';
		this.formNewPassword = '';
		this.formNewPassword2 = '';
		this.formAvatarPreview = u?.avatar || '';
		this.avatarDirty = false;
		// 先关下拉，避免菜单按钮叠在弹窗下面透出
		this.menuOpen.set(false);
		this.profileOpen.set(true);
		this.lockBodyScroll(true);
	}

	closeProfile(): void {
		if (this.busy()) return;
		this.profileOpen.set(false);
		this.lockBodyScroll(false);
	}

	private lockBodyScroll(lock: boolean): void {
		if (typeof document === 'undefined') return;
		document.body.style.overflow = lock ? 'hidden' : '';
	}

	onAvatarFile(ev: Event): void {
		const input = ev.target as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) return;
		if (!ShellComponent.avatarMime.has(file.type)) {
			void this.dialog.error('头像仅支持 PNG / JPEG / WebP / GIF');
			input.value = '';
			return;
		}
		if (file.size > 150 * 1024) {
			void this.dialog.error('头像请小于 150KB');
			input.value = '';
			return;
		}
		const reader = new FileReader();
		reader.onload = () => {
			const result = String(reader.result || '');
			if (result.length > 180 * 1024) {
				void this.dialog.error('头像编码后过大，请换更小的图');
				return;
			}
			this.formAvatarPreview = result;
			this.avatarDirty = true;
		};
		reader.readAsDataURL(file);
		input.value = '';
	}

	clearAvatar(): void {
		this.formAvatarPreview = '';
		this.avatarDirty = true;
	}

	saveProfile(): void {
		const username = this.formUsername.trim();
		const name = this.formDisplayName.trim();
		if (!username) {
			void this.dialog.error('请填写登录名');
			return;
		}
		if (!/^[A-Za-z0-9_.-]{1,32}$/.test(username)) {
			void this.dialog.error('登录名仅限字母、数字、_ - .，最长 32');
			return;
		}
		if (!name) {
			void this.dialog.error('请填写昵称');
			return;
		}
		if (name.length > 32) {
			void this.dialog.error('昵称最多 32 个字符');
			return;
		}

		const wantPwd = !!(this.formOldPassword || this.formNewPassword || this.formNewPassword2);
		if (wantPwd) {
			if (!this.formOldPassword) {
				void this.dialog.error('请填写当前密码');
				return;
			}
			if (this.formNewPassword.length < 8) {
				void this.dialog.error('新密码至少 8 位');
				return;
			}
			if (this.formNewPassword !== this.formNewPassword2) {
				void this.dialog.error('两次输入的新密码不一致');
				return;
			}
		}

		const profileBody: { username: string; displayName: string; avatar?: string } = {
			username,
			displayName: name,
		};
		if (this.avatarDirty) {
			profileBody.avatar = this.formAvatarPreview || '';
		}

		this.busy.set(true);
		this.auth.updateProfile(profileBody).subscribe({
			next: () => {
				if (!wantPwd) {
					this.busy.set(false);
					this.profileOpen.set(false);
					this.lockBodyScroll(false);
					void this.dialog.success('资料已保存');
					return;
				}
				this.auth.changePassword(this.formOldPassword, this.formNewPassword).subscribe({
					next: () => {
						this.busy.set(false);
						this.profileOpen.set(false);
						this.lockBodyScroll(false);
						void this.dialog.success('资料与密码已更新，请使用新密码重新登录');
						this.auth.logout();
					},
					error: (err: Error) => {
						this.busy.set(false);
						void this.dialog.error(`资料已保存，但改密失败：${err.message}`);
					},
				});
			},
			error: (err: Error) => {
				this.busy.set(false);
				void this.dialog.error(err.message);
			},
		});
	}

	logout(): void {
		this.menuOpen.set(false);
		this.auth.logout();
	}
}

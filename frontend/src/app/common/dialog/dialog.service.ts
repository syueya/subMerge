import { Injectable, signal } from '@angular/core';

export type DialogKind = 'info' | 'success' | 'error' | 'confirm';

export interface DialogState {
	open: boolean;
	kind: DialogKind;
	title: string;
	message: string;
	confirmText: string;
	cancelText: string;
	showCancel: boolean;
}

export type ToastKind = 'info' | 'success' | 'error';

export interface ToastState {
	open: boolean;
	kind: ToastKind;
	title: string;
	message: string;
}

const defaultState = (): DialogState => ({
	open: false,
	kind: 'info',
	title: '',
	message: '',
	confirmText: '确定',
	cancelText: '取消',
	showCancel: false,
});

const defaultToast = (): ToastState => ({
	open: false,
	kind: 'info',
	title: '',
	message: '',
});

/** 需要看完整内容时用弹窗：多行、或明显偏长的结果文案 */
function preferModal(message: string): boolean {
	const m = message ?? '';
	if (m.includes('\n')) return true;
	// 刷新/导入类汇总往往较长，留给弹窗
	if (m.length > 96) return true;
	return false;
}

@Injectable({ providedIn: 'root' })
export class DialogService {
	readonly state = signal<DialogState>(defaultState());
	readonly toast = signal<ToastState>(defaultToast());

	private resolver: ((ok: boolean) => void) | null = null;
	private toastTimer: ReturnType<typeof setTimeout> | null = null;
	private toastResolver: (() => void) | null = null;

	/** 顶部轻提示，默认约 1.6s 自动消失，不挡操作 */
	showToast(
		kind: ToastKind,
		message: string,
		title?: string,
		durationMs = 1600,
	): Promise<void> {
		const t =
			title ??
			(kind === 'success' ? '成功' : kind === 'error' ? '错误' : '提示');
		if (this.toastTimer != null) {
			clearTimeout(this.toastTimer);
			this.toastTimer = null;
		}
		if (this.toastResolver) {
			const prev = this.toastResolver;
			this.toastResolver = null;
			prev();
		}
		this.toast.set({ open: true, kind, title: t, message });
		return new Promise<void>((resolve) => {
			this.toastResolver = resolve;
			this.toastTimer = setTimeout(() => {
				this.toastTimer = null;
				this.toast.set(defaultToast());
				const r = this.toastResolver;
				this.toastResolver = null;
				r?.();
			}, Math.max(800, durationMs));
		});
	}

	dismissToast(): void {
		if (this.toastTimer != null) {
			clearTimeout(this.toastTimer);
			this.toastTimer = null;
		}
		const r = this.toastResolver;
		this.toastResolver = null;
		this.toast.set(defaultToast());
		r?.();
	}

	info(message: string, title = '提示'): Promise<void> {
		if (preferModal(message)) {
			return this.open({
				kind: 'info',
				title,
				message,
				showCancel: false,
				confirmText: '知道了',
			}).then(() => undefined);
		}
		return this.showToast('info', message, title);
	}

	success(message: string, title = '成功'): Promise<void> {
		if (preferModal(message)) {
			return this.open({
				kind: 'success',
				title,
				message,
				showCancel: false,
				confirmText: '确定',
			}).then(() => undefined);
		}
		return this.showToast('success', message, title);
	}

	error(message: string, title = '错误'): Promise<void> {
		if (preferModal(message)) {
			return this.open({
				kind: 'error',
				title,
				message,
				showCancel: false,
				confirmText: '知道了',
			}).then(() => undefined);
		}
		return this.showToast('error', message, title, 2000);
	}

	confirm(message: string, title = '请确认', confirmText = '确定'): Promise<boolean> {
		return this.open({
			kind: 'confirm',
			title,
			message,
			showCancel: true,
			confirmText,
			cancelText: '取消',
		});
	}

	resolve(ok: boolean): void {
		const r = this.resolver;
		this.resolver = null;
		this.state.set(defaultState());
		r?.(ok);
	}

	private open(partial: Partial<DialogState> & Pick<DialogState, 'kind' | 'title' | 'message'>): Promise<boolean> {
		if (this.resolver) {
			this.resolver(false);
			this.resolver = null;
		}
		return new Promise<boolean>((resolve) => {
			this.resolver = resolve;
			this.state.set({
				...defaultState(),
				...partial,
				open: true,
			});
		});
	}
}

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

const defaultState = (): DialogState => ({
	open: false,
	kind: 'info',
	title: '',
	message: '',
	confirmText: '确定',
	cancelText: '取消',
	showCancel: false,
});

@Injectable({ providedIn: 'root' })
export class DialogService {
	readonly state = signal<DialogState>(defaultState());

	private resolver: ((ok: boolean) => void) | null = null;

	info(message: string, title = '提示'): Promise<void> {
		return this.open({ kind: 'info', title, message, showCancel: false, confirmText: '知道了' }).then(
			() => undefined,
		);
	}

	success(message: string, title = '成功'): Promise<void> {
		return this.open({
			kind: 'success',
			title,
			message,
			showCancel: false,
			confirmText: '确定',
		}).then(() => undefined);
	}

	error(message: string, title = '错误'): Promise<void> {
		return this.open({
			kind: 'error',
			title,
			message,
			showCancel: false,
			confirmText: '知道了',
		}).then(() => undefined);
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

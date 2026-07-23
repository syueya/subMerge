import { NgClass } from '@angular/common';
import { Component, HostListener, inject } from '@angular/core';
import { DialogService } from './dialog.service';

@Component({
	selector: 'app-dialog',
	standalone: true,
	imports: [NgClass],
	templateUrl: './dialog.component.html',
	styleUrl: './dialog.component.scss',
})
export class DialogComponent {
	readonly dialog = inject(DialogService);

	iconClass(kind: string): string {
		switch (kind) {
			case 'success':
				return 'icon-success';
			case 'error':
				return 'icon-error';
			case 'confirm':
				return 'icon-confirm';
			default:
				return 'icon-info';
		}
	}

	onBackdrop(): void {
		const s = this.dialog.state();
		if (s.showCancel) {
			this.dialog.resolve(false);
		}
	}

	@HostListener('document:keydown.escape')
	onEsc(): void {
		if (!this.dialog.state().open) return;
		this.dialog.resolve(false);
	}
}

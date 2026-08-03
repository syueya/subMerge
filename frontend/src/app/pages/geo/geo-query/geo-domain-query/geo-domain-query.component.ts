import { Component, EventEmitter, Output, inject, signal } from '@angular/core';
import { DialogService } from '@common/services/dialog.service';
import { BADGE_MUTED, BADGE_WARN, GeoQueryResponse } from '@data-struct';
import { CmParentComponent } from '@common/parents/parent/parent.component';
import { takeUntil } from 'rxjs';
import { GeoService } from '../../services/geo.service';

@Component({
	selector: 'app-geo-domain-query',
	templateUrl: './geo-domain-query.component.html',
	standalone: false,
})
export class GeoDomainQueryComponent extends CmParentComponent {
	private readonly svc = inject(GeoService);
	private readonly dialog = inject(DialogService);

	@Output() openCategory = new EventEmitter<string>();

	readonly queryResult = signal<GeoQueryResponse | null>(null);
	readonly loading = signal(false);

	readonly badgeMuted = BADGE_MUTED;
	readonly badgeWarn = BADGE_WARN;

	domain = '';
	resolve = false;

	query(): void {
		if (!this.domain.trim()) {
			void this.dialog.error('请输入域名');
			return;
		}
		this.loading.set(true);
		this.svc
			.query(this.domain, this.resolve)
			.pipe(takeUntil(this.$destroy))
			.subscribe({
				next: (result) => {
					this.queryResult.set(result);
					this.loading.set(false);
				},
				error: (err: Error) => {
					this.loading.set(false);
					void this.dialog.error(err.message);
				},
			});
	}

	clearResult(): void {
		this.queryResult.set(null);
	}

	openGeoSiteCategoryEntries(category: string): void {
		this.openCategory.emit(category);
	}
}

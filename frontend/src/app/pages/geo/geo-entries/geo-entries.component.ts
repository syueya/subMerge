import { Component, inject } from '@angular/core';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { MatTableDataSource } from '@angular/material/table';
import { GeoEntryRow } from '@data-struct';
import { CmParentComponent } from '@common/parents/parent/parent.component';

export interface GeoEntriesDialogData {
	title: string;
	subtitle: string;
	items: GeoEntryRow[];
	total: number;
	offset: number;
	limit: number;
	loading: boolean;
	paginated: boolean;
}

export type GeoEntriesDialogResult = 'previous' | 'next' | null;

@Component({
	selector: 'app-geo-entries',
	templateUrl: './geo-entries.component.html',
	standalone: false,
})
export class GeoEntriesComponent extends CmParentComponent {
	dialogRef = inject<MatDialogRef<GeoEntriesComponent, GeoEntriesDialogResult>>(MatDialogRef);
	data = inject<GeoEntriesDialogData>(MAT_DIALOG_DATA);

	dataSource = new MatTableDataSource<GeoEntryRow>([]);
	displayedColumns = ['type', 'value', 'detail'];

	constructor() {
		super();
		this.dataSource.data = this.data.items || [];
	}

	get title(): string {
		return this.data.title || '条目';
	}

	get subtitle(): string {
		return this.data.subtitle || (this.data.total ? `共 ${this.data.total} 条` : '');
	}

	get loading(): boolean {
		return !!this.data.loading;
	}

	get paginated(): boolean {
		return !!this.data.paginated;
	}

	get total(): number {
		return this.data.total || 0;
	}

	get offset(): number {
		return this.data.offset || 0;
	}

	get items(): GeoEntryRow[] {
		return this.data.items || [];
	}

	canPrevious(): boolean {
		return this.offset > 0 && !this.loading;
	}

	canNext(): boolean {
		return this.offset + this.items.length < this.total && !this.loading;
	}

	rangeText(): string {
		if (!this.total) return `0 / ${this.total}`;
		return `${this.offset + 1}-${this.offset + this.items.length} / ${this.total}`;
	}

	previous(): void {
		if (!this.canPrevious()) return;
		this.dialogRef.close('previous');
	}

	next(): void {
		if (!this.canNext()) return;
		this.dialogRef.close('next');
	}

	close(): void {
		this.dialogRef.close(null);
	}

	tableTrackBy = (_: number, row: GeoEntryRow) => `${row.type}|${row.value}|${row.detail || ''}`;
}

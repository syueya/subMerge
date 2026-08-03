import { Pipe, PipeTransform } from '@angular/core';

@Pipe({ standalone: true, name: 'seasonFormat' })
export class SeasonFormatPipePipe implements PipeTransform {
  transform(season: number): string {
    return `S${String(season).padStart(2, '0')}`;
  }
}

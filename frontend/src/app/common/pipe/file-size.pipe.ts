import { Pipe, PipeTransform } from '@angular/core';
import { formatBytes } from '@common/util';

@Pipe({
  name: 'fileSize',
  standalone: true
})
export class FileSizePipe implements PipeTransform {

  transform(size: number): string {
    return formatBytes(size);
  }

}

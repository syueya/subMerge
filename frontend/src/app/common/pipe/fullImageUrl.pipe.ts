import { Pipe, PipeTransform } from '@angular/core';
import { getFullUrl } from '@common/util/urlUtils';

@Pipe({ standalone: true, name: 'getFullImageUrl' })
export class GetFullImageUrlPipe implements PipeTransform {
  transform(url: string): string {
    if (!url) {
      return './assets/images/no-image.png';
    }
    // 判断 url 是否为有效地址，如果是则直接返回
    if (url.startsWith('http://') || url.startsWith('https://')) {
      return url;
    }
    return getFullUrl(`/api/v1${url}`);
  }
}

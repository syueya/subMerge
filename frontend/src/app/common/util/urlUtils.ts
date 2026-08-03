import { environment } from '@env/environment';

/**
 * 拼接相对地址为完整的 URL。
 * SubMerge 开发走 proxy、生产同域托管时 backEndUrl 为空，应保持相对路径或用当前 origin。
 */
export function getFullUrl(relativeUrl: string): string {
  const base = (environment.backEndUrl || '').trim();
  if (!base) {
    // 空 base：相对路径即可（ng serve proxy / 同域静态托管）
    // 若调用方需要绝对 URL，退回当前页面 origin
    if (relativeUrl.startsWith('/')) {
      return relativeUrl;
    }
    try {
      return new URL(relativeUrl, window.location.origin).href;
    } catch {
      return relativeUrl;
    }
  }

  try {
    return new URL(relativeUrl, base.endsWith('/') ? base : `${base}/`).href;
  } catch {
    // base 非法时不抛到业务层
    return relativeUrl.startsWith('/') ? relativeUrl : `/${relativeUrl}`;
  }
}

/**
 * 处理请求的URL，确保其格式正确
 */
export function processUrl(reqUrl: string): string {
  // 处理静态资源请求
  if (reqUrl.startsWith('assets')) {
    return `./${reqUrl.startsWith('/') ? reqUrl.substring(1) : reqUrl}`;
  }

  // 处理非绝对URL
  if (!reqUrl.startsWith('https://') && !reqUrl.startsWith('http://')) {
    return getFullUrl(reqUrl);
  }

  return reqUrl;
}

/**
 * 拼接文件夹路径
 */
export function joinPath(pathArray: string[]): string {
  return `/${pathArray.join('/')}`;
}

/**
 * 获取父级路径
 */
export function getParentPath(path: string): string {
  const normalizedPath = path.replace(/\/+$/, '/');
  const lastSlashIndex = normalizedPath.lastIndexOf('/');
  if (lastSlashIndex > 0) {
    return normalizedPath.substring(0, lastSlashIndex);
  }
  return '/';
}

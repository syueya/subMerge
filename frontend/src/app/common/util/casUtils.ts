/**
 * 规范化 CAS 号：去掉首尾空白与中间多余空格
 */
export function formatCas(cas: string | null | undefined): string {
  if (!cas) {
    return '';
  }
  return String(cas).trim().replace(/\s+/g, '');
}

import { formatTimeAgo } from './dateUtils';

/**
 * 为列表项补充 updatedAtStr（多页 reloadTableData 共用）
 */
export function withUpdatedAtStr<T extends { updatedAt?: number | string | null; updatedAtStr?: string }>(list: T[] | null | undefined): T[] {
  if (!list?.length) {
    return [];
  }
  return list.map(item => {
    if (item.updatedAt) {
      item.updatedAtStr = formatTimeAgo(item.updatedAt as any);
    }
    return item;
  });
}

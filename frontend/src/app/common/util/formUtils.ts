/* eslint-disable @typescript-eslint/no-explicit-any */

/**
 * 将对象的所有key中的.替换为__
 * @param obj 输入对象
 * @returns 新对象
 */
export function convertDotKeysToUnderscore(obj: Record<string, any>): Record<string, any> {
  if (!obj || typeof obj !== 'object') return obj;
  const result: Record<string, any> = {};
  for (const key in obj) {
    if (!Object.prototype.hasOwnProperty.call(obj, key)) continue;
    const newKey = key.replace(/\./g, '__');
    const value = obj[key];
    result[newKey] = (value && typeof value === 'object' && !Array.isArray(value))
      ? convertDotKeysToUnderscore(value)
      : value;
  }
  return result;
}

/**
 * 将对象的所有key中的__替换为.
 * @param obj 输入对象
 * @returns 新对象
 */
export function convertUnderscoreKeysToDot(obj: Record<string, any>): Record<string, any> {
  if (!obj || typeof obj !== 'object') return obj;
  const result: Record<string, any> = {};
  for (const key in obj) {
    if (!Object.prototype.hasOwnProperty.call(obj, key)) continue;
    const newKey = key.replace(/__/g, '.');
    const value = obj[key];
    result[newKey] = (value && typeof value === 'object' && !Array.isArray(value))
      ? convertUnderscoreKeysToDot(value)
      : value;
  }
  return result;
}

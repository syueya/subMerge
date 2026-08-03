/* eslint-disable @typescript-eslint/no-explicit-any */
import { coerceCssPixelValue } from '@angular/cdk/coercion';

/**
 * 将数字转换为 CSS 像素值字符串
 * @param value 要转换的数字
 * @returns CSS 像素值字符串
 */
export function toCssPixel(value: number | string): string {
  return coerceCssPixelValue(value);
}

/**
 * 将字符串转换为 Base64 编码
 * @param input 要编码的字符串
 * @returns Base64 编码后的字符串
 */
export function stringToBase64(input: string): string {
  if (typeof window !== 'undefined' && typeof window.btoa === 'function') {
    // 在浏览器环境中
    return window.btoa(input);
  } else {
    throw new Error('Environment not supported.');
  }
}

/**
 * 将 Base64 编码转换为字符串
 * @param base64 要解码的 Base64 编码
 * @returns 解码后的字符串
 */
export function base64ToString(base64: string): string {
  if (typeof window !== 'undefined' && typeof window.atob === 'function') {
    // 在浏览器环境中
    return window.atob(base64);
  } else {
    throw new Error('Environment not supported.');
  }
}

/**
 * 将字节数转换为可读文件大小格式
 * @param size 要转换的字节数
 * @returns 可读格式的字符串
 */
export function formatBytes(size: number): string {
  // 检查输入是否为 null 或 undefined
  if (size === null || size === 0 || size === undefined) {
    return `0 B`;
  }

  // 检查输入是否为数字
  if (typeof size !== 'number' || isNaN(size)) {
    return `0 B`;
  }
  // 定义单位
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB'];
  let unitIndex = 0;

  // 转换大小
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex++;
  }

  // 保留两位小数并省略多余的0
  const formattedSize = size % 1 === 0 ? size.toString() : size.toFixed(2).replace(/\.?0+$/, '');

  return `${formattedSize} ${units[unitIndex]}`;
}

/**
 * 将字节每秒数转换为可读文件大小格式
 * @param size 要转换的字节每秒数
 * @returns 可读格式的字符串
 */
export function formatBytesPerSecond(size: number): string {
  // 检查输入是否为 null 或 undefined
  if (size === null || size === 0 || size === undefined) {
    return `0 B/s`;
  }

  // 检查输入是否为数字
  if (typeof size !== 'number' || isNaN(size)) {
    return `0 B/s`;
  }
  // 定义单位
  const units = ['B/s', 'KB/s', 'MB/s', 'GB/s', 'TB/s', 'PB/s', 'EB/s'];
  let unitIndex = 0;

  // 转换大小
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex++;
  }

  // 保留两位小数并省略多余的0
  const formattedSize = size % 1 === 0 ? size.toString() : size.toFixed(2).replace(/\.?0+$/, '');

  return `${formattedSize} ${units[unitIndex]}`;
}


/**
 * 计算传输速度
 *
 * @param startTime 开始时间戳
 * @param endTime 结束时间戳
 * @param totalSize 传输的总大小（字节）
 * @returns 传输速度（字节/秒）
*/
export function calculateSpeed(startTime: number, endTime: number, totalSize: number): string {
    if (!startTime || !endTime || !totalSize) {
      return `${0  } B/s`;
    }
    // 计算花费的时间
    const timeSpent = endTime - startTime;
    if (timeSpent <= 0) {
        throw new Error("结束时间必须大于开始时间");
    }

    // 计算速度
    const speed = totalSize / timeSpent;
    return formatBytesPerSecond(speed);// 格式化速度
  }

/**
 * 将数字转换为可读数字格式
 * @param size 要转换的数字
 * @returns 可读格式的字符串
 */
export function formatNumberWithUnits(size: number): string {
  // 检查输入是否为 null 或 undefined
  if (size === null || size === undefined) {
    return '0';
  }

  // 检查输入是否为数字
  if (typeof size !== 'number' || isNaN(size)) {
    return '0';
  }

  // 定义单位
  const units = ['', 'k', 'w', 'kw'];
  let unitIndex = 0;

  // 转换大小
  if (size >= 1000 && size < 10000) {
    size /= 1000;
    unitIndex = 1; // 'k'
  } else if (size >= 10000 && size < 10000000) {
    size /= 10000;
    unitIndex = 2; // 'w'
  } else if (size >= 10000000) {
    size /= 10000000;
    unitIndex = 3; // 'kw'
  }

  // 保留两位小数并省略多余的0
  const formattedSize = size % 1 === 0 ? size.toFixed(0) : size.toFixed(2).replace(/\.?0+$/, '');

  return `${formattedSize}${units[unitIndex]}`;
}


/**
 * 将数字枚举转换为数字数组
 * @param enumObj
 * @returns
 */
export function getEnumNumberValues(enumObj: any): number[] {
  return Object.values(enumObj).filter(value => typeof value === 'number') as number[];
}

/**
 * 将数字扩大指定倍数，并保留小数点后的指定位数
 * @param value 原始数字
 * @param multiplier 要扩大的倍数,默认为100倍
 * @param decimalPlaces 小数点后保留的位数，默认为1位小数点
 * @returns 扩大后的数字
 */
export function multiplyAndFormat(value: number, multiplier = 100, decimalPlaces = 1): number {
  if (value === null || value === undefined) {
    return NaN; // 返回NaN表示不是一个数字
  }
  const multipliedValue = value * multiplier; // 乘以倍数
  const formatResult = parseFloat(multipliedValue.toFixed(decimalPlaces)); // 格式化小数点并转换回数字类型
  return formatResult;
}

/**
 * 将文本输入框中的文本转换为数组
 * @param input 文本输入框中的文本
 * @returns 数组
 */
export function textAreaToArray(input: string): string[] {
  if (!input) {
    return [];
  }
  return input
    .split(/\r?\n/) // 根据换行符分割文本
    .filter(line => line.trim() !== ''); // 过滤掉空行
}


/**
 * 将对象转换为扁平化的对象
 * @param obj 要转换的对象
 * @returns 扁平化的对象
 */
export function flattenObject(obj: Record<string, any>, prefix = ''): Record<string, any> {
  return Object.keys(obj).reduce((acc, key) => {
      const newKey = prefix ? `${prefix}.${key}` : key;
      if (typeof obj[key] === 'object' && obj[key] !== null && !Array.isArray(obj[key])) {
          Object.assign(acc, flattenObject(obj[key], newKey));
      } else {
          acc[newKey] = obj[key];
      }
      return acc;
  }, {} as Record<string, any>);
}

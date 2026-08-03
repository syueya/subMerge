/**
 * 获取文件扩展名
 * @param filename
 * @returns
 */
export function getFileExtension(filename: string): string {
  // 找到最后一个点的位置
  const lastDotIndex = filename.lastIndexOf('.');
  // 如果找不到点，或者点是第一个字符，返回空字符串
  if (lastDotIndex === -1 || lastDotIndex === 0) {
    return '';
  }
  // 返回从最后一个点到字符串末尾的子字符串
  return filename.substring(lastDotIndex + 1);
}

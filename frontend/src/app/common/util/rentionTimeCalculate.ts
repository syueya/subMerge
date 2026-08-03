export interface AlkaneTime {
  carbon: number;
  time: number;
}


// 根据保留时间找前后烷烃
function findNeighboursByTime(alkanes: AlkaneTime[], targetTime: number): [AlkaneTime, AlkaneTime] {
  let lower!: AlkaneTime;
  let upper!: AlkaneTime;
  for (const item of alkanes) {
    if (item.time <= targetTime) {
      lower = item;
    } else if (!upper) {
      upper = item;
    }
  }
  if (!lower || !upper) throw new Error('找不到前后烷烃');
  return [lower, upper];
}


// 根据保留指数找前后烷烃
function findNeighboursByIndex(alkanes: AlkaneTime[], targetIndex: number): [AlkaneTime, AlkaneTime] {
  let lower!: AlkaneTime;
  let upper!: AlkaneTime;
  for (const item of alkanes) {
    if (item.carbon * 100 <= targetIndex) {
      lower = item;
    } else if (!upper) {
      upper = item;
    }
  }
  if (!lower || !upper) throw new Error('找不到前后烷烃');
  return [lower, upper];
}

// 保留时间 → 保留指数
export function retentionIndexFromRetentionTime(alkanes: AlkaneTime[], retentionTime: number): number {
  const [lower, upper] = findNeighboursByTime(alkanes, retentionTime);
  const ratio = (retentionTime - lower.time) / (upper.time - lower.time);
  return lower.carbon * 100 + Math.round(ratio * 100);
}

// 保留指数 → 保留时间
export function retentionTimeFromRetentionIndex(alkanes: AlkaneTime[], retentionIndex: number): number {
  const [lower, upper] = findNeighboursByIndex(alkanes, retentionIndex);
  const ratio = (retentionIndex - lower.carbon * 100) / 100;
  const calculatedTime =
    lower.time + ratio * (upper.time - lower.time);
  return Number(calculatedTime.toFixed(3));
}

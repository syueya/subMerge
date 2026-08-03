// 定义支持的元素类型
type ElementSymbol = 'C' | 'H' | 'O' | 'N' | 'S';

// 定义化学式解析结果的接口
export type ParsedFormula = Record<string, number>;


// 定义化学式反应计算的接口
export interface ReactionFormula {
  formula: string; // 分子式
  weight: number // 分子量
}


// 原子量表
const ATOMIC_WEIGHTS: Record<ElementSymbol, number> = {
  C: 12,
  H: 1,
  O: 16,
  N: 14,
  S: 32,
};

/**
 * 类型守卫：验证元素符号
 */
function isElementSymbol(symbol: string): symbol is ElementSymbol {
  return ['C', 'H', 'O', 'N', 'S'].includes(symbol as ElementSymbol);
}


/**
 * 解析简单化学式（如C6H12O6）
 * @param formula 化学式字符串
 * @returns 解析结果
 */
export function parseChemicalFormula(formula: string): ParsedFormula {
  if (!formula) throw new Error('化学式不能为空');

  const result: ParsedFormula = {};

   // 匹配一个元素符号后跟任意数量的数字（可选）
  const regex = /([CHONS])(\d*)/g;

  let match: RegExpExecArray | null;
  while ((match = regex.exec(formula)) !== null) {
    const element = match[1] as ElementSymbol;
    const count = match[2] ? parseInt(match[2]) : 1;  // 提取数量，默认为1
    result[element] = (result[element] || 0) + count;
  }

  if (Object.keys(result).length === 0) {
    throw new Error('无效的化学式格式');
  }

  return result;
}

/**
 * 计算化学式的分子量
 * @param formula 化学式字符串
 * @returns 分子量
 */
export function calculateMolecularWeight(formula: string): number {
  const elements = parseChemicalFormula(formula);
  let totalWeight = 0;

  for (const element in elements) {
    if (isElementSymbol(element)) {
      totalWeight += ATOMIC_WEIGHTS[element] * elements[element];
    } else {
      throw new Error(`不支持的元素: ${element}`);
    }
  }

  return Math.round(totalWeight); // 确保结果为整数
}

/**
 * 将解析后的化学式对象转换回字符串表示
 * @param parsedFormula 解析后的化学式对象
 * @returns 化学式字符串
 */
export function formatFormula(parsedFormula: ParsedFormula): string {
  // 按元素符号排序（C、H、O、N、S顺序）
  const elementsOrder: ElementSymbol[] = ['C', 'H', 'O', 'N', 'S'];

  return elementsOrder
    .filter(element => parsedFormula[element] > 0)
    .map(element =>
      parsedFormula[element] === 1 ? element : `${element}${parsedFormula[element]}`
    )
    .join('');
}


/**
 * 将多个化学式相加，返回新的化学式和分子量
 * @param baseFormula 基础化学式
 * @param ...restFormulas 要添加的化学式
 * @returns 包含新化学式和分子量的对象
 */
export function addFormulas(baseFormula: string, ...restFormulas: string[]): ReactionFormula {
  const merged: ParsedFormula = parseChemicalFormula(baseFormula);

  // 累加其余所有化学式
  for (const f of restFormulas) {
    const parsed = parseChemicalFormula(f);
    for (const el in parsed) {
      merged[el] = (merged[el] || 0) + parsed[el];
    }
  }

  const newFormula = formatFormula(merged);
  const weight     = calculateMolecularWeight(newFormula);

  return { formula: newFormula, weight };
}


/**
 * 从化学式中脱去指定数量的水分子，返回新的化学式和分子量
 * @param baseFormula 基础化学式
 * @param waterMolecules 要脱去的水分子数量，默认为1
 * @returns 包含新化学式和分子量的对象
 */
export function removeWater(baseFormula: string, waterMolecules = 1): ReactionFormula {
  const original = parseChemicalFormula(baseFormula);
  const waterElements = parseChemicalFormula('H2O');

  // 从原始化学式中减去水分子的元素组成
  const modified: ParsedFormula = { ...original };
  for (const element in waterElements) {
    const countToRemove = waterElements[element] * waterMolecules;
    if (!modified[element] || modified[element] < countToRemove) {
      throw new Error(`无法从 ${baseFormula} 中脱去 ${waterMolecules} 分子水：元素 ${element} 不足`);
    }
    modified[element] -= countToRemove;

    // 如果元素计数为0，从对象中删除
    if (modified[element] === 0) {
      delete modified[element];
    }
  }

  // 生成新的化学式字符串
  const newFormula = formatFormula(modified);

  // 计算新的分子量
  const weight = calculateMolecularWeight(newFormula);

  return {
    formula: newFormula,
    weight: weight
  };
}

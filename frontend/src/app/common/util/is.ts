
export function isFunction<T>(val: T) {
  return Object.prototype.toString.call(val) === '[object Function]';
}

export function isObject<T>(val: T)  {
  return Object.prototype.toString.call(val) === '[object Object]';
}

export function isArray<T>(val: T) {
  return Array.isArray(val);
}

export function isSet<T>(val: T) {
  return Object.prototype.toString.call(val) === '[object Set]';
}

export function isMap<T>(val: T) {
  return Object.prototype.toString.call(val) === '[object Map]';
}

export function isSymbol<T>(val: T) {
  return Object.prototype.toString.call(val) === '[object Symbol]';
}

export function isDate<T>(val: T) {
  return Object.prototype.toString.call(val) === '[object Date]';
}

export function isRegExp<T>(val: T) {
  return Object.prototype.toString.call(val) === '[object RegExp]';
}

export function isArrayBuffer<T>(val: T) {
  return Object.prototype.toString.call(val) === '[object ArrayBuffer]';
}

export function isNull<T>(val: T) {
  return val === null;
}

export function isUndefined<T>(val: T) {
  return val === undefined;
}

export function isNullOrUndefined<T>(val: T) {
  return val === undefined || val === null;
}

export function isNumber<T>(val: T) {
  return typeof val === 'number';
}

export function isString<T>(val: T) {
  return typeof val === 'string';
}

export function isBoolean<T>(val: T) {
  return typeof val === 'boolean';
}

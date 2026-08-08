import { AbstractControl, ValidationErrors, ValidatorFn } from '@angular/forms';

/**
 * 验证确认密码
 * @param controlNameToCompare
 * @returns
 */
export function validateConfirmPassword(controlNameToCompare: string): ValidatorFn {
  return (control: AbstractControl): ValidationErrors | null => {
    if (!control.parent) {
      return null; // 如果不是表单的一部分，则不进行验证
    }
    const passwordControl = control.parent.get(controlNameToCompare);
    if (passwordControl && passwordControl.value !== control.value) {
      return { confirmPasswordMismatch: true };
    }
    return null;
  };
}

/**
 * 验证用户名
 * @param control
 * @returns
 */
export function validateUserName(control: AbstractControl): ValidationErrors | null {
  if (!/^[a-zA-Z][a-zA-Z0-9]*$/.test(control.value)) {
    return { userNameInvalid: true };
  }
  return null;
}

/**
 * 验证正则表达式
 * @param pattern
 * @param errorKey
 * @param errorMessage
 * @returns
 */
export function patternValidator(pattern: RegExp, errorKey: string, errorMessage: string): ValidatorFn {
  let regex: RegExp;
  try {
    regex = new RegExp(pattern);
  } catch (e) {
    console.error('无效正则表达式:', e);
    return () => ({ [errorKey]: '无效正则表达式' });
  }
  return (control: AbstractControl): ValidationErrors | null => {
    if (!control.value) {
      return null; // 如果控件值为空，不进行验证
    }
    const valid = regex.test(control.value);
    return valid ? null : { [errorKey]: errorMessage };
  };
}

/**
 * 验证cron格式
 * @returns
 */
export function cronValidator(): ValidatorFn {
  return (control: AbstractControl): ValidationErrors | null => {
    const cronExpression = control.value;
    if (!control.value) {
      return null; // 如果控件值为空，不进行验证
    }
    const cronParts = cronExpression.split(' ');
    // 仅检查标准 Unix/Linux 风格的 cron 表达式
    if (cronParts.length !== 5) {
      return { cronInvalid: 'cron表达式无效，必须是空格分隔的5段' };
    }

    const [minute, hour, dayOfMonth, month, dayOfWeek] = cronParts;

    const isValidPart = (part: string, pattern: RegExp) => pattern.test(part);

    const minutePattern = /^(\*|([0-5]?\d)(\/[0-5]?\d)?|\*\/[0-5]?\d)(,(\*|([0-5]?\d)(\/[0-5]?\d)?|\*\/[0-5]?\d))*$/;
    const hourPattern = /^(\*|([01]?\d|2[0-3])(\/([01]?\d|2[0-3]))?|\*\/([01]?\d|2[0-3]))(,(\*|([01]?\d|2[0-3])(\/([01]?\d|2[0-3]))?|\*\/([01]?\d|2[0-3])))*$/;
    const dayOfMonthPattern = /^(\*|([01]?\d|2[0-9]|3[01])(\/([01]?\d|2[0-9]|3[01]))?|\*\/([01]?\d|2[0-9]|3[01]))(,(\*|([01]?\d|2[0-9]|3[01])(\/([01]?\d|2[0-9]|3[01]))?|\*\/([01]?\d|2[0-9]|3[01])))*$/;
    const monthPattern = /^(\*|(0?[1-9]|1[0-2])(\/(0?[1-9]|1[0-2]))?|\*\/(0?[1-9]|1[0-2]))(,(\*|(0?[1-9]|1[0-2])(\/(0?[1-9]|1[0-2]))?|\*\/(0?[1-9]|1[0-2])))*$/;
    const dayOfWeekPattern = /^(\*|([0-7]|(SUN|MON|TUE|WED|THU|FRI|SAT))(\/([0-7]|(SUN|MON|TUE|WED|THU|FRI|SAT)))?|\*\/([0-7]|(SUN|MON|TUE|WED|THU|FRI|SAT)))(,(\*|([0-7]|(SUN|MON|TUE|WED|THU|FRI|SAT))(\/([0-7]|(SUN|MON|TUE|WED|THU|FRI|SAT)))?|\*\/([0-7]|(SUN|MON|TUE|WED|THU|FRI|SAT))))*$/;
    const dayOfWeekExtendedPattern = /^(\*|([0-7]|(SUN|MON|TUE|WED|THU|FRI|SAT))([#L][1-5])?)(,(\*|([0-7]|(SUN|MON|TUE|WED|THU|FRI|SAT))([#L][1-5])?))*$/;
    const dayOfWeekRangePattern = /^(\*|([0-7]|(SUN|MON|TUE|WED|THU|FRI|SAT))(-([0-7]|(SUN|MON|TUE|WED|THU|FRI|SAT)))?)(,(\*|([0-7]|(SUN|MON|TUE|WED|THU|FRI|SAT))(-([0-7]|(SUN|MON|TUE|WED|THU|FRI|SAT)))?))*$/;
    const dayOfWeekLastPattern = /^(\*|([0-7]|(SUN|MON|TUE|WED|THU|FRI|SAT))L)(,(\*|([0-7]|(SUN|MON|TUE|WED|THU|FRI|SAT))L))*$/;

    if (!isValidPart(minute, minutePattern)) {
      return { cronInvalid: 'cron表达式【分钟】部分格式错误' };
    }
    if (!isValidPart(hour, hourPattern)) {
      return { cronInvalid: 'cron表达式【小时】部分格式错误' };
    }
    if (!isValidPart(dayOfMonth, dayOfMonthPattern)) {
      return { cronInvalid: 'cron表达式【日期】部分格式错误' };
    }
    if (!isValidPart(month, monthPattern)) {
      return { cronInvalid: 'cron表达式【月份】部分格式错误' };
    }
    if (!isValidPart(dayOfWeek, dayOfWeekPattern) && !isValidPart(dayOfWeek, dayOfWeekExtendedPattern) && !isValidPart(dayOfWeek, dayOfWeekRangePattern) && !isValidPart(dayOfWeek, dayOfWeekLastPattern)) {
      return { cronInvalid: 'cron表达式星期部分格式错误' };
    }

    return null;
  };
}

/**
 * 验证正整数
 * @returns
 */
export function positiveIntegerValidator(): ValidatorFn {
  return (control: AbstractControl): ValidationErrors | null => {
    if (!control.value) {
      return null; // 如果控件值为空，不进行验证
    }

    if (!!control.value || control.value === 0) {
      const v = /^[1-9]\d*$/;

      // 如果输入非正整数
      if (!v.test(control.value)) {
        return { positiveInteger: '请输入正整数！' };
      }

      return null;
    }
    return null;
  };
}

/**
 * 自定义验证器：路径必须以/开头
 */
export function filePathValidator(): ValidatorFn {
  return (control: AbstractControl): ValidationErrors | null => {
    if (!control.value) {
      return null; // 如果控件值为空，不进行验证
    }
    if (!control.value.startsWith('/')) {
      return { filePath: '路径不合法,必须以/开头' };
    }
    if (control.value.includes('..')) {
      return { filePath: '路径不合法,不能包含相对路径' };
    }
    if (control.value.endsWith('/')) {
      return { filePath: '路径不合法,路径不能以/结尾' };
    }
    if (control.value === '/') {
      return { filePath: '路径不合法,不能为根目录' };
    }
    return null;
  };
}

/**
 * URL验证器：校验 http/https 地址合法性
 * 使用 URL 构造函数解析，可校验协议、主机名、路径、查询参数、端口
 * @param base 为 true 时仅校验协议前缀
 * @returns
 */
export function hostValidator(base = false): ValidatorFn {
  return (control: AbstractControl): ValidationErrors | null => {
    const raw = String(control.value || '').trim();
    if (!raw) {
      return null; // 如果控件值为空，不进行验证
    }

    // 基础模式：仅检查是否以http或https开头
    if (base) {
      if (!/^(https?:\/\/).*$/i.test(raw)) {
        return { url: '域名必须以http或https开头' };
      }
      return null;
    }

    // 完整模式：使用 URL 构造函数解析，校验协议、主机名、路径、查询参数、端口
    try {
      const url = new URL(raw);
      const protocol = url.protocol.toLowerCase();
      if (
        (protocol !== 'http:' && protocol !== 'https:') ||
        !url.hostname ||
        url.pathname.startsWith('//') ||
        url.search ||
        url.hash
      ) {
        return { url: '请输入合法的 HTTP/HTTPS 地址' };
      }
      if (url.port && (!/^\d+$/.test(url.port) || Number(url.port) < 1 || Number(url.port) > 65535)) {
        return { url: '端口号必须在 1-65535 范围内' };
      }
    } catch {
      return { url: '请输入合法的 HTTP/HTTPS 地址' };
    }

    // 检查URL结尾是否包含/
    if (raw.endsWith('/')) {
      return { url: '域名结尾不能出现/' };
    }

    // 使用正则表达式检查URL是否包含中文标点符号和其他特殊字符
    const specialCharPattern = /[!#$%^&*()+=[\]{};'"\\|,<>?·~！#￥%……&*（）——+【】{}；：”“’‘、|，。《》？、]/;
    if (specialCharPattern.test(raw)) {
      return { url: '域名中不允许包含中文标点符号和其他特殊字符' };
    }

    return null;
  };
}

/**
 * 代理验证器：代理URL必须以http、https、socks5h、socks5开头，后面可以跟冒号和端口号，端口号必须是数字，且不大于5位
 * @returns
 */
export function proxyHostValidator(): ValidatorFn {
  return (control: AbstractControl): ValidationErrors | null => {
    if (!control.value) {
      return null; // 如果控件值为空，不进行验证
    }

    // 使用正则表达式检查URL是否以http、https、socks5h、socks5开头，后面可以跟冒号和端口号
    /*     const urlPattern = /^(https?|socks5h?|socks5):\/\/.*$/i;
    if (!urlPattern.test(control.value)) {
      return { url: '域名必须以http、https、socks5h、socks5开头' };
    } */

    // 检查URL结尾是否包含/
    if (control.value.endsWith('/')) {
      return { url: '域名结尾不能出现/' };
    }

    // 使用正则表达式检查URL是否包含中文标点符号和其他特殊字符
    const specialCharPattern = /[!#$%^&*()+=[\]{};'"\\|,<>?·~！#￥%……&*（）——+【】{}；：”“’‘、|，。《》？、]/;
    if (specialCharPattern.test(control.value)) {
      return { url: '域名中不允许包含中文标点符号和其他特殊字符' };
    }

    // 如果有两个冒号，则需要验证端口号格式
    const colonCount = (control.value.match(/:/g) || []).length;
    if (colonCount === 2) {
      // 冒号后面的端口号必须是数字，且不大于5位
      const portPattern = /^(https?|socks5h?|socks5):\/\/.*:(\d{1,5})$/i;
      if (!portPattern.test(control.value)) {
        return { url: '端口号必须是数字，且不大于5位' };
      }
    }

    return null;
  };
}



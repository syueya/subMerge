const uaMathList = [
  {
    name: '苹果',
    matchList: [/iPhone/i, /Macintosh/i, /like Mac OS X/i, /iPad/i, /iPod/i]
  },
  {
    name: '华为',
    matchList: [/HUAWEI/i, /SPN-AL00/i, /GLK-AL00/i, /Huawei/i, /HMSCore/i, /HW/]
  },
  {
    name: '荣耀',
    matchList: [/HONOR/i]
  },
  {
    name: 'oppo',
    matchList: [/PCAM10/i, /OPPO/i, /PCH/, /PBAM00/, /PBEM00/, /HeyTapBrowser/, /PADT00/, /PCDM10/]
  },
  {
    name: 'vivo',
    matchList: [/V1981A/i, /vivo/i, /V1818A/, /V1838A/, /V19/, /VivoBrowser/]
  },
  {
    name: '小米',
    matchList: [/Redmi/i, /HM/, /MIX/i, /MI/, /XiaoMi/]
  },
  {
    name: '金利',
    matchList: [/GN708T/i]
  },
  {
    name: 'oneplus',
    matchList: [/GM1910/i, /ONEPLUS/i]
  },
  {
    name: 'sony',
    matchList: [/SOV/i, /LT29i/, /Xperia/]
  },
  {
    name: '三星',
    matchList: [/SAMSUNG/i, /SM-/, /GT/, /SCH-I939I/]
  },
  {
    name: '魅族',
    matchList: [/MZ-/, /MX4/i, /M355/, /M353/, /M351/, /M811/, /PRO 7-H/]
  },
  {
    name: '华硕',
    matchList: [/ASUS/]
  },
  {
    name: '美图',
    matchList: [/MP/]
  },
  {
    name: '天语',
    matchList: [/K-Touch/]
  },
  {
    name: '联想',
    matchList: [/Lenovo/i]
  },
  {
    name: '宇飞来',
    matchList: [/YU FLY/i]
  },
  {
    name: '糖果',
    matchList: [/SUGAR/i]
  },
  {
    name: '酷派',
    matchList: [/Coolpad/i]
  },
  {
    name: 'ecell',
    matchList: [/ecell/i]
  },
  {
    name: '詹姆士',
    matchList: [/A99A/i]
  },
  {
    name: 'tcl',
    matchList: [/TCL/i]
  },
  {
    name: '捷语',
    matchList: [/6000/i, /V1813A/]
  },
  {
    name: '8848',
    matchList: [/8848/i]
  },
  {
    name: 'H6',
    matchList: [/H6/]
  },
  {
    name: '中兴',
    matchList: [/ZTE/i]
  },
  {
    name: '努比亚',
    matchList: [/NX/]
  },
  {
    name: '努比亚',
    matchList: [/NX/]
  },
  {
    name: '海信',
    matchList: [/HS/]
  },
  {
    name: 'HTC',
    matchList: [/HTC/]
  }
];

const browserList = [
  { name: 'Edge', matchList: [/(Edge|Edg\/|EdgA|EdgiOS)/] },
  { name: 'Safari', matchList: [/Safari/] },
  { name: 'Chrome', matchList: [/(Chrome|CriOS)/] },
  { name: 'IE', matchList: [/(MSIE|Trident)/] },
  { name: 'Firefox', matchList: [/(Firefox|FxiOS)/] },
  { name: 'Firefox Focus', matchList: [/Focus/] },
  { name: 'Chromium', matchList: [/Chromium/] },
  { name: 'Opera', matchList: [/(Opera|OPR|OPT)/] },
  { name: 'Vivaldi', matchList: [/Vivaldi/] },
  { name: 'Yandex', matchList: [/YaBrowser/] },
  { name: 'Arora', matchList: [/Arora/] },
  { name: 'Lunascape', matchList: [/Lunascape/] },
  { name: 'QupZilla', matchList: [/QupZilla/] },
  { name: 'Coc Coc', matchList: [/coc_coc_browser/] },
  { name: 'Kindle', matchList: [/(Kindle|Silk\/)/] },
  { name: 'Iceweasel', matchList: [/Iceweasel/] },
  { name: 'Konqueror', matchList: [/Konqueror/] },
  { name: 'Iceape', matchList: [/Iceape/] },
  { name: 'SeaMonkey', matchList: [/SeaMonkey/] },
  { name: 'Epiphany', matchList: [/Epiphany/] },
  { name: '360', matchList: [/(QihooBrowser|QHBrowser)/] },
  { name: '360EE', matchList: [/360EE/] },
  { name: '360SE', matchList: [/360SE/] },
  { name: 'UC', matchList: [/(UCBrowser|UBrowser|UCWEB)/] },
  { name: 'QQBrowser', matchList: [/QQBrowser/] },
  { name: 'QQ', matchList: [/QQ\//] },
  { name: 'Baidu', matchList: [/(Baidu|BIDUBrowser|baidubrowser|baiduboxapp|BaiduHD)/] },
  { name: 'Maxthon', matchList: [/Maxthon/] },
  { name: 'Sogou', matchList: [/(MetaSr|Sogou)/] },
  { name: 'Liebao', matchList: [/(LBBROWSER|LieBaoFast)/] },
  { name: '2345Explorer', matchList: [/2345Explorer/] },
  { name: '115Browser', matchList: [/115Browser/] },
  { name: 'TheWorld', matchList: [/TheWorld/] },
  { name: 'XiaoMi', matchList: [/MiuiBrowser/] },
  { name: 'Vivo', matchList: [/VivoBrowser/] },
  { name: 'Huawei', matchList: [/HuaweiBrowser/] },
  { name: 'OPPO', matchList: [/HeyTapBrowser/] },
  { name: 'Quark', matchList: [/Quark/] },
  { name: 'Qiyu', matchList: [/Qiyu/] },
  { name: 'Wechat', matchList: [/MicroMessenger/] },
  { name: 'WechatWork', matchList: [/wxwork\//] },
  { name: 'Taobao', matchList: [/AliApp\(TB/] },
  { name: 'Alipay', matchList: [/AliApp\(AP/] },
  { name: 'Weibo', matchList: [/Weibo/] },
  { name: 'Douban', matchList: [/com\.douban\.frodo/] },
  { name: 'Suning', matchList: [/SNEBUY-APP/] },
  { name: 'iQiYi', matchList: [/IqiyiApp/] },
  { name: 'DingTalk', matchList: [/DingTalk/] },
  { name: 'Douyin', matchList: [/aweme/] }
];

const osList = [
  {
    name: 'Windows',
    matchList: [/Windows/]
  },
  {
    name: 'Linux',
    matchList: [/(Linux|X11)/]
  },
  {
    name: 'MacOS',
    matchList: [/Macintosh/]
  },
  {
    name: 'Android',
    matchList: [/(Android|Adr)/]
  },
  {
    name: 'HarmonyOS',
    matchList: [/HarmonyOS/]
  },
  {
    name: 'Ubuntu',
    matchList: [/Ubuntu/]
  },
  {
    name: 'FreeBSD',
    matchList: [/FreeBSD/]
  },
  {
    name: 'Debian',
    matchList: [/Debian/]
  },
  {
    name: 'Windows Phone',
    matchList: [/(IEMobile|Windows Phone)/]
  },
  {
    name: 'BlackBerry',
    matchList: [/(BlackBerry|RIM)/]
  },
  {
    name: 'MeeGo',
    matchList: [/MeeGo/]
  },
  {
    name: 'Symbian',
    matchList: [/Symbian/]
  },
  {
    name: 'iOS',
    matchList: [/like Mac OS X/]
  },
  {
    name: 'Chrome OS',
    matchList: [/CrOS/]
  },
  {
    name: 'WebOS',
    matchList: [/hpwOS/]
  }
];
interface UaDetail {
  name: string;
  matchList: RegExp[];
}
function matchUaDetail(ua: string, uaList: UaDetail[]): string {
  for (const uaDetail of uaList) {
    for (const re of uaDetail.matchList) {
      if (re.test(ua)) {
        return uaDetail.name;
      }
    }
  }
  const brandMatch = /; ([^;]+) Build/.exec(ua);
  return brandMatch ? brandMatch[1] : '';
}

function getBrand(ua: string): string {
  return matchUaDetail(ua, uaMathList);
}

function getBrowser(ua: string): string {
  return matchUaDetail(ua, browserList);
}

function getOs(ua: string): string {
  return matchUaDetail(ua, osList);
}

interface UserAgentData {
  getHighEntropyValues(properties: string[]): Promise<Record<string, string | null>>;
  platform: string;
}

interface Navigator extends NavigatorID, NavigatorOnLine, NavigatorConcurrentHardware, NavigatorStorage, NavigatorLanguage, NavigatorPlugins {
  userAgentData: UserAgentData;
}

async function getWindowsVersion(ua: string) {
  const userAgent = ua.toLowerCase();
  const windows11Regex = /windows nt 10.0/; // Windows 11的用户代理可能与Windows 10相似，可能需要更精细的检查
  const windows81Regex = /windows 8.1/;
  const windows8Regex = /windows 8/;
  const windows7Regex = /windows 7/;
  const windowsVistaRegex = /windows vista/;
  const windowsXPRegex = /windows xp/;

  if (windows11Regex.test(userAgent)) {
    const na = navigator as unknown as Navigator;
    if ('userAgentData' in na) {
      try {
        const uaData = (await na.userAgentData!.getHighEntropyValues(['platformVersion'])) as Record<string, string>;
        if (na.userAgentData!.platform === 'Windows') {
          const majorPlatformVersion = parseInt(uaData["platformVersion"].split('.')[0]);
          if (majorPlatformVersion >= 13) {
            return 'Windows 11';
          } else if (majorPlatformVersion > 0) {
            return 'Windows 10';
          } else {
            return 'Windows';
          }
        } else {
          return 'Windows 11 or 10';
        }
      } catch (error) {
        return 'Windows 11 or 10';
      }
    } else {
      return 'Windows 11 or 10';
    }
  } else if (windows81Regex.test(userAgent)) {
    return 'Windows 8.1';
  } else if (windows8Regex.test(userAgent)) {
    return 'Windows 8';
  } else if (windows7Regex.test(userAgent)) {
    return 'Windows 7';
  } else if (windowsVistaRegex.test(userAgent)) {
    return 'Windows Vista';
  } else if (windowsXPRegex.test(userAgent)) {
    return 'Windows XP';
  }

  return '';
}

// 设备信息工具函数
async function getDeviceInfo() {
  const userAgent = navigator.userAgent;

  if (!userAgent) {
    return '未知浏览器';
  }

  const band = getBrand(userAgent) || '';
  let os = getOs(userAgent) || '';
  if (os === 'Windows') {
    os = await getWindowsVersion(userAgent);
  }

  const browser = getBrowser(userAgent) || '';
  let deviceInfo = '';
  if (band && os && browser) {
    deviceInfo = `${band} ${os} ${browser}`;
  } else if (band && browser) {
    deviceInfo = `${band} ${browser}`;
  } else if (os && browser) {
    deviceInfo = `${os} ${browser}`;
  } else if (browser) {
    deviceInfo = `${browser}`;
  }

  if (deviceInfo) {
    return `${deviceInfo}浏览器`;
  } else {
    // 其他设备
    return '未知浏览器';
  }
}

export { getDeviceInfo };

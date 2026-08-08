import { MenuItem } from '../interfaces/MenuItem';

export const MenuList: MenuItem[] = [
  {
    displayName: '工作概览',
    iconName: 'layout-dashboard',
    route: '/main/dashboard'
  },
  {
    displayName: '配置',
    children: [
      {
        displayName: '订阅管理',
        iconName: 'cloud-download',
        route: '/main/sources'
      },
      {
        displayName: '策略分组',
        iconName: 'hierarchy-2',
        route: '/main/groups'
      },
      {
        displayName: '分流规则',
        iconName: 'list-check',
        route: '/main/rules'
      },
      {
        displayName: '版本发布',
        iconName: 'rocket',
        route: '/main/releases'
      },
      {
        displayName: '订阅链接',
        iconName: 'key',
        route: '/main/tokens'
      }
    ]
  },
  {
    displayName: '运维',
    children: [
      {
        displayName: 'Geo 数据',
        iconName: 'world',
        route: '/main/geo'
      },
      {
        displayName: '网络检测',
        iconName: 'brand-speedtest',
        route: '/main/net-check'
      },
      {
        displayName: '系统设置',
        iconName: 'settings',
        route: '/main/settings/setting-settings'
      },
      {
        displayName: '系统日志',
        iconName: 'list-details',
        route: '/main/logs'
      }
    ]
  }
];

export default MenuList;

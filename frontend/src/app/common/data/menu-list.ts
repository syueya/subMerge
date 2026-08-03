import { MenuItem } from '../interfaces/MenuItem';

export const MenuList: MenuItem[] = [
  {
    displayName: '概览',
    iconName: 'layout-dashboard',
    route: '/main/dashboard'
  },
  {
    displayName: '订阅源',
    iconName: 'activity',
    route: '/main/sources'
  },
  {
    displayName: '策略组',
    iconName: 'components',
    route: '/main/groups'
  },
  {
    displayName: '分流规则',
    iconName: 'settings',
    route: '/main/rules'
  },
  {
    displayName: '发布',
    iconName: 'history',
    route: '/main/releases'
  },
  {
    displayName: '令牌',
    iconName: 'key',
    route: '/main/tokens'
  },
  {
    displayName: 'Geo 数据',
    iconName: 'world',
    route: '/main/geo'
  }
];

export default MenuList;

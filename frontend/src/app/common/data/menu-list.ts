import { MenuItem } from '../interfaces/MenuItem';

export const MenuList: MenuItem[] = [
  {
    displayName: '概览',
    iconName: 'layout-dashboard',
    route: '/main/dashboard'
  },
  {
    displayName: '订阅源',
    iconName: 'cloud-download',
    route: '/main/sources'
  },
  {
    displayName: '策略组',
    iconName: 'hierarchy-2',
    route: '/main/groups'
  },
  {
    displayName: '分流规则',
    iconName: 'list-check',
    route: '/main/rules'
  },
  {
    displayName: 'Geo 数据',
    iconName: 'world',
    route: '/main/geo'
  },
  {
    displayName: '发布',
    iconName: 'rocket',
    route: '/main/releases'
  },
  {
    displayName: '令牌',
    iconName: 'key',
    route: '/main/tokens'
  }
];

export default MenuList;

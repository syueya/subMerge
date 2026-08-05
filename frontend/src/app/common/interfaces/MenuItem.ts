export interface MenuItem {
    displayName?: string; // 显示名称
    disabled?: boolean; // 是否禁用
    external?: boolean; // 是否外部链接
    twoLines?: boolean; // 是否两行显示
    chip?: boolean; // 是否显示小圆点
    iconName?: string;  // 图标名称
    navCap?: string;    // 导航标题
    chipContent?: string;   // 小圆点内容
    chipClass?: string;  // 小圆点样式
    subtext?: string;   // 子文本
    route?: string;   // 路由地址
    children?: MenuItem[];  // 子菜单
    ddType?: string;    // 下拉菜单类型
}
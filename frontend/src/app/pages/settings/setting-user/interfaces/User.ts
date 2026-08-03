export interface User {
  id: number; //用户ID
  userName: string; //用户名
  nickName: string; //昵称
  password: string; //密码
  passwordSecond: string; //确认密码
  avatar?: string; //头像
  role?: number | string; //角色
  createdAt?: number; //创建时间
  updatedAt?: number; //更新时间
  lastDeviceType?: string; //最近一次成功登录设备类型
  lastDeviceName?: string; //最近一次成功登录设备名称
  lastIp?: string; //最近一次成功登录IP
  lastLoginTime?: number; //最近一次成功登录时间戳
}

export interface LoginLog {
  id: number;          // 主键
  userId: number;      // 用户ID(用户不存在时为0)
  userName: string;    // 登录用户名(失败时也记录)
  success: boolean;    // 是否登录成功
  reason: string;      // 失败原因(成功为空)
  ip: string;          // 登录IP
  deviceType: string;  // 设备类型
  deviceName: string;  // 设备名称
  loginTime: number;   // 登录时间戳
}

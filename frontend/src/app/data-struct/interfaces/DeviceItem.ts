export interface DeviceItem {
  userId: number;
  userName: string;
  nickName: string;
  avatar: string;
  role: number;
  createdAt: number;
  updatedAt: number;
  ip: string;
  deviceType: string; //设备类型 'web' 'ios' 'android'
  deviceName: string; //设备名称
  loginTime: number;
  loginTimeStr: string;
  token: string;
  current: boolean; //是否当前设备
}

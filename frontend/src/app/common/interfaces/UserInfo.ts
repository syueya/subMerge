/**
 * 用户信息接口
 */
export interface UserInfo {
  id: number;
  userName: string;
  nickName: string;
  role: number;
  avatar: string;
  createdAt: number;
  updatedAt: number;
  avatarSrc?: string;
}

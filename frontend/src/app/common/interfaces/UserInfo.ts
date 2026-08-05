/**
 * 登录管理员的前端展示缓存（由后端 AdminUser 映射，非多用户体系）
 */
export interface UserInfo {
  id: number;
  userName: string;
  nickName: string;
  avatar: string;
  createdAt: number;
  updatedAt: number;
  avatarSrc?: string;
}

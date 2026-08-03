import { Option } from '@common/interfaces';

/** 用户角色，与后端 UserRoleEnum 对齐 */
export enum UserRoleEnum {
  Admin = 1,
  Normal = 2
}

export const UserRoleEnumToName: Record<number, string> = {
  [UserRoleEnum.Admin]: '管理员',
  [UserRoleEnum.Normal]: '普通用户'
};

export const UserRoleList: Array<Option<number>> = [
  { value: UserRoleEnum.Admin, text: UserRoleEnumToName[UserRoleEnum.Admin] },
  { value: UserRoleEnum.Normal, text: UserRoleEnumToName[UserRoleEnum.Normal] }
];

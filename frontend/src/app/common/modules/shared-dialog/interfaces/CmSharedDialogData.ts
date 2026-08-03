import { CmSharedDialogDataModel } from "../enum/CmSharedDialogDataModel";

export interface CmSharedDialogData {
    /* 类型 */
    model:CmSharedDialogDataModel;
    /* 标题 */
    title?: string;
    /* 内容 */
    content?: string;
    /* 确认按钮文字 */
    sureStr?: string;
    /* 取消按钮文字 */
    cancelStr?: string;

    /** 宽度 */
    width?:string;
}
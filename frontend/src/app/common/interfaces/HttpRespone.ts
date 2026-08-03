/* eslint-disable @typescript-eslint/no-explicit-any */

import { SortDirection } from "@angular/material/sort";

/** Value of key-value pairs object */
export type PlainJsonValue = boolean | number | string | null | undefined;
/**
 * Typeof JSON object parsed from Response data
 * simple key-value pairs object.
 */
export interface JsonType {
  [key: string]: PlainJsonValue | PlainJsonValue[] | JsonType | JsonType[];
}


/** Custom response json data structure */
export interface HttpRespone<T extends JsonType | PlainJsonValue | PlainJsonValue[] | JsonType[] | any = any> {
  /** 0: no error */
  code: number;
  /** payload */
  data?: T;
  message?: string | null;
}


export interface ListRespone<T extends JsonType | PlainJsonValue | PlainJsonValue[] | JsonType[] | any = any> {
  /** 0: no error */
  code: number;
  /** payload */
  data?: {
    list: T[];
    total:number;
    pageNum:number;
    pageSize:number;
    [key: string]: any;
  };
  message?: string | null;
}

export interface TableSortData {

  // 排序字段

  orderBy: string;

  // 升序还是降序

  orderDirection: SortDirection;

}

/**
 * 列表查询，基础 query params
 *
 * @export
 * @interface ServiceQueryParams
 */
export interface ServiceQueryParams {
  pageNum: number;
  pageSize: number;
}

export const BasicQueryParams: ServiceQueryParams = {
  pageNum: 1,
  pageSize: 20
};

/**
 * 用于设置table分页表，属性类型
 *
 * @export
 * @interface PaginatorPropsType
 */
export interface PaginatorPropsType {
  pageIndex: number;
  pageSize: number;
  length: number;
}

export function initialPaginatorProps(): PaginatorPropsType {
  return {
      pageIndex: 0,
      pageSize: 20,
      length: 0,
  }
}

// 泛型响应接口
export interface BaseResp<T> {
  data: T;  // 泛型数据
}

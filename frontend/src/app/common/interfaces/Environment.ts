/* eslint-disable @typescript-eslint/no-explicit-any */
export interface Environment {
    [key: string]: any;
    production:boolean;
    mockBackend:boolean;
    backEndUrl:string;
    version:string;
}
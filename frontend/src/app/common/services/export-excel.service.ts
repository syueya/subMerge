import { Injectable } from '@angular/core';

type XLSXModule = typeof import('xlsx');
type ExcelCellValue = unknown;
type ExcelRow = Record<string, ExcelCellValue>;
interface ExcelColumnConfig {
  field: string;
  header: string;
  format?: (value: ExcelCellValue) => ExcelCellValue;
}

@Injectable({
  providedIn: 'root'
})
export class ExportExcelService {

  constructor() { }

  /**
   * 动态加载 xlsx 模块，避免全量引入导致首屏体积过大（约 400KB）
   */
  private async loadXlsx(): Promise<XLSXModule> {
    return await import('xlsx');
  }

  /**
   * 将数据导出为Excel文件（支持Promise回调）
   */
  async exportToExcel<T extends object>(data: T[], fileName = '报告', sheetName = 'Sheet1'): Promise<void> {
    try {
      const XLSX = await this.loadXlsx();
      const ws = XLSX.utils.json_to_sheet(data);
      const wb = XLSX.utils.book_new();
      XLSX.utils.book_append_sheet(wb, ws, sheetName);
      const excelBuffer = XLSX.write(wb, { bookType: 'xlsx', type: 'array' }) as BlobPart;
      this.saveExcelFile(excelBuffer, fileName);
    } catch (error) {
      console.error('导出Excel失败:', error);
      throw error;
    }
  }

  /**
   * 保存Excel文件到本地
   */
  private saveExcelFile(buffer: BlobPart, fileName: string): void {
    const data: Blob = new Blob([buffer], { type: 'application/octet-stream' });
    const url = window.URL.createObjectURL(data);
    const link = document.createElement('a');
    link.href = url;
    link.download = `${fileName}.xlsx`;
    link.click();
    window.URL.revokeObjectURL(url);
  }

  /**
   * 导出带表头映射的Excel，只导出映射中定义的列（支持Promise回调）
   */
  exportToExcelWithHeaderMap<T extends object>(data: T[], headerMap: Record<string, string>, fileName = '报告', sheetName = 'Sheet1'): Promise<void> {
    const formattedData = data.map(item => {
      const formattedItem: ExcelRow = {};
      Object.keys(headerMap).forEach(key => {
        if (Object.hasOwn(item, key)) {
          formattedItem[headerMap[key]] = (item as ExcelRow)[key];
        }
      });
      return formattedItem;
    });

    return this.exportToExcel(formattedData, fileName, sheetName);
  }

  /**
   * 导出带表头映射的Excel，支持自定义列顺序（支持Promise回调）
   */
  exportToExcelWithColumnConfig<T extends object>(
    data: T[],
    columnConfig: ExcelColumnConfig[],
    fileName = '报告',
    sheetName = 'Sheet1'
  ): Promise<void> {
    const formattedData = data.map(item => {
      const formattedItem: ExcelRow = {};
      columnConfig.forEach(({ field, header, format }) => {
        if (Object.hasOwn(item, field)) {
          const value = (item as ExcelRow)[field];
          formattedItem[header] = format ? format(value) : value;
        }
      });
      return formattedItem;
    });

    return this.exportToExcel(formattedData, fileName, sheetName);
  }

  /**
   * 导出复杂Excel，支持多工作表、自定义样式和合并单元格（支持Promise回调）
   */
  async exportComplexExcel(workbookConfig: {
    fileName: string;
    sheets: Array<{
      sheetName: string;
      data: object[];
      columnConfig: ExcelColumnConfig[];
      mergeCells?: Array<{ s: { r: number; c: number }; e: { r: number; c: number } }>;
      styles?: Record<string, Record<string, unknown>>;
    }>;
  }): Promise<void> {
    try {
      const XLSX = await this.loadXlsx();
      const wb = XLSX.utils.book_new();

      workbookConfig.sheets.forEach(sheet => {
        const formattedData = sheet.data.map(item => {
          const formattedItem: ExcelRow = {};
          sheet.columnConfig.forEach(({ field, header, format }) => {
            if (Object.hasOwn(item, field)) {
              const value = (item as ExcelRow)[field];
              formattedItem[header] = format ? format(value) : value;
            }
          });
          return formattedItem;
        });

        const ws = XLSX.utils.json_to_sheet(formattedData);

        if (sheet.mergeCells) {
          ws['!merges'] = sheet.mergeCells;
        }

        if (sheet.styles) {
          Object.keys(sheet.styles).forEach(cellAddress => {
            if (ws[cellAddress]) {
              ws[cellAddress].s = sheet.styles![cellAddress];
            }
          });
        }

        XLSX.utils.book_append_sheet(wb, ws, sheet.sheetName);
      });

      const excelBuffer = XLSX.write(wb, { bookType: 'xlsx', type: 'array' }) as BlobPart;
      this.saveExcelFile(excelBuffer, workbookConfig.fileName);
    } catch (error) {
      console.error('导出Excel失败:', error);
      throw error;
    }
  }
}

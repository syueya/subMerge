import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { HttpRespone } from '@common/interfaces';
import { catchError, forkJoin, map, of } from 'rxjs';

export interface UploadFilesResult {
  paths: string[];
  successFiles: string[];
  failedFiles: string[];
}

@Injectable({ providedIn: 'root' })
export class SystemUploadService {
  private httpClient = inject(HttpClient);

  uploadFile(file: File) {
    const fileFormData = new FormData();
    fileFormData.append('file', file);
    return this.httpClient.post<HttpRespone<string>>('/api/v1/system/upload', fileFormData);
  }

  uploadFiles(files: File[]) {
    const tasks = files.map(file => this.uploadFile(file).pipe(
      map(response => ({ file, response })),
      catchError(() => of({ file, response: null }))
    ));

    return forkJoin(tasks).pipe(
      map(results => results.reduce<UploadFilesResult>((acc, item) => {
        if (item.response?.code === 20000 && item.response.data) {
          acc.paths.push(item.response.data);
          acc.successFiles.push(item.file.name);
        } else {
          acc.failedFiles.push(item.file.name);
        }
        return acc;
      }, { paths: [], successFiles: [], failedFiles: [] }))
    );
  }
}

import {ErrorHandler, Injectable} from '@angular/core';


@Injectable()
export class CmGlobalErrorHandler implements ErrorHandler {

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  handleError(error: any): void {
    const chunkFailedMessage = /Loading chunk [\d]+ failed/;
    // 网速卡时 加载资源失败
    const loadResourceFailed = /Cannot activate an already activated outlet/;
    console.error(error);
    if (chunkFailedMessage.test(error.message) || loadResourceFailed.test(error.message)) {
      alert('未能加载网站资源文件。程序可能已更新版本，需要重新刷新页面。必要时请清空浏览器缓存，刷新尝试！');
    }
  }
}

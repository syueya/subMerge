// 样品报告：文档 + 色谱峰波形（样品/谱图语义）
export const IconCustomReportSample =
  '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">' +
  '<path stroke="none" d="M0 0h24v24H0z" fill="none"/>' +
  // 文档外形
  '<path d="M14 3v4a1 1 0 0 0 1 1h4"/>' +
  '<path d="M17 21h-10a2 2 0 0 1 -2 -2v-14a2 2 0 0 1 2 -2h7l5 5v11a2 2 0 0 1 -2 2z"/>' +
  // 基线
  '<path d="M8.5 17.2h7"/>' +
  // 色谱峰：矮-高-中
  '<path d="M10 17.2v-2.4"/>' +
  '<path d="M12 17.2v-4.2"/>' +
  '<path d="M14 17.2v-3"/>' +
  // 峰顶小点，增强“谱图”识别
  '<circle cx="12" cy="12.6" r="0.55" fill="currentColor" stroke="none"/>' +
  '</svg>';

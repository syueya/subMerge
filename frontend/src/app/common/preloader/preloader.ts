import { isPlatformServer } from '@angular/common';
import { PLATFORM_ID, inject, DOCUMENT } from '@angular/core';

export function stepPreloader(): () => void {
  const doc: Document = inject(DOCUMENT);
  const ssr = isPlatformServer(inject(PLATFORM_ID));
  if (ssr) {
    return () => {};
  }
  const body = doc.querySelector<HTMLBodyElement>('body')!;
  body.style.overflow = 'hidden';
  let done = false;

  return () => {
    if (done) return;

    done = true;

    const el = doc.querySelector<HTMLElement>('.preloader');
    if (!el) return;
    el.style.transition = 'opacity 0.8s';
    el.style.opacity = '0';

    const logo = el.querySelector<HTMLElement>('.logo-zoom-svg');
    if (logo) {
      logo.style.animation = 'none'; // 移除原有的动画
      logo.style.transition = 'transform 0.8s';
      logo.style.transform = 'scale(0)';
    }

    setTimeout(() => {
      el?.parentNode?.removeChild(el);
      body.style.overflow = '';
    }, 800); // 等待0.8秒（与过渡时间一致）后移除元素
  
  };
}
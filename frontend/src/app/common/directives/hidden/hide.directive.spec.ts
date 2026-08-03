import { ElementRef } from '@angular/core';
import { TestBed } from '@angular/core/testing';

import { CmHideDirective } from './hide.directive';

describe('CmHideDirective', () => {
  it('should create an instance', () => {
    const elementRef = new ElementRef<HTMLElement>(document.createElement('div'));
    TestBed.configureTestingModule({
      providers: [{ provide: ElementRef, useValue: elementRef }]
    });
    const directive = TestBed.runInInjectionContext(() => new CmHideDirective());
    expect(directive).toBeTruthy();
  });
});

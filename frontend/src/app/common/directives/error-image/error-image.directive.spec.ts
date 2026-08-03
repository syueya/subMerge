import { ElementRef } from '@angular/core';
import { TestBed } from '@angular/core/testing';

import { ErrorImageDirective } from './error-image.directive';

describe('ErrorImageDirective', () => {
  it('should create an instance', () => {
    const el = new ElementRef<HTMLImageElement>(document.createElement('img'));
    TestBed.configureTestingModule({
      providers: [{ provide: ElementRef, useValue: el }]
    });
    const directive = TestBed.runInInjectionContext(() => new ErrorImageDirective());
    expect(directive).toBeTruthy();
  });
});

import { ComponentFixture, TestBed } from '@angular/core/testing';

import { FormLayoutModule } from '../form-layout.module';
import { CmFormFieldComponent } from './cm-form-field.component';

describe('CmFormFieldComponent', () => {
  let component: CmFormFieldComponent;
  let fixture: ComponentFixture<CmFormFieldComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [FormLayoutModule]
    }).compileComponents();

    fixture = TestBed.createComponent(CmFormFieldComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});

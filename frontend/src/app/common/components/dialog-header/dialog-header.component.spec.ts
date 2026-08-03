import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ComponentsModule } from '@common/components/components.module';

import { CmDialogHeaderComponent } from './dialog-header.component';

describe('CmDialogHeaderComponent', () => {
  let component: CmDialogHeaderComponent;
  let fixture: ComponentFixture<CmDialogHeaderComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ComponentsModule]
    }).compileComponents();

    fixture = TestBed.createComponent(CmDialogHeaderComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});

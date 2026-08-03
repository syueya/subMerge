import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ParentsModule } from '@common/parents/parents.module';

import { CmMatContorlParentComponent } from './mat-contorl-parent.component';

describe('MatContorlParentComponent', () => {
  let component: CmMatContorlParentComponent;
  let fixture: ComponentFixture<CmMatContorlParentComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ParentsModule]
    }).compileComponents();

    fixture = TestBed.createComponent(CmMatContorlParentComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});

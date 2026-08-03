import { AfterViewInit, Component, ElementRef, Input, OnChanges, SimpleChanges, ViewChild } from '@angular/core';
import SmilesDrawer from 'smiles-drawer';

@Component({
  selector: 'cm-smiles-preview',
  standalone: false,
  templateUrl: './smiles-preview.component.html'
})
export class SmilesPreviewComponent implements AfterViewInit, OnChanges {
  @Input() smiles = '';
  @Input() width = 300;
  @Input() height = 180;

  @ViewChild('svgElement') svgElement?: ElementRef<SVGElement>;

  errorMessage = '';

  ngAfterViewInit() {
    this.render();
  }

  ngOnChanges(changes: SimpleChanges) {
    if (changes['smiles']) {
      queueMicrotask(() => this.render());
    }
  }

  private render() {
    const svg = this.svgElement?.nativeElement;
    if (!svg) {
      return;
    }
    svg.replaceChildren();
    this.errorMessage = '';
    const value = this.smiles?.trim();
    if (!value) {
      return;
    }
    const drawer = new SmilesDrawer.SvgDrawer({ width: this.width, height: this.height });
    SmilesDrawer.parse(
      value,
      tree => drawer.draw(tree, svg, 'light'),
      () => this.errorMessage = 'SMILES 格式无法识别，请检查结构式'
    );
  }
}

import { Component } from '@angular/core';


@Component({
  selector: 'app-branding',
  standalone: true,
  template: `
    <a href="/" class="branding-link d-flex align-items-center text-decoration-none">
      <img
        src="./assets/images/logos/logoIcon.svg"
        class="branding-logo"
        alt="SubMerge"
      />
      <span class="branding-title hide-menu">SubMerge</span>
    </a>
  `
})
export class BrandingComponent {}

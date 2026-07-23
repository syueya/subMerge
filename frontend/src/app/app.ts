import { Component, inject } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { DialogComponent } from './common/dialog/dialog.component';
import { ThemeService } from './core/theme.service';

@Component({
 selector: 'app-root',
 imports: [RouterOutlet, DialogComponent],
 templateUrl: './app.html',
 styleUrl: './app.css',
})
export class App {
 /** 注入以在启动时应用主题 */
 private readonly theme = inject(ThemeService);
}

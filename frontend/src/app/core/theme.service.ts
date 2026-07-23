import { Injectable, signal } from '@angular/core';

export type ThemeMode = 'light' | 'dark';

const STORAGE_KEY = 'submerge-theme';

/** 黑白主题：默认浅色，localStorage 持久化 */
@Injectable({ providedIn: 'root' })
export class ThemeService {
 readonly theme = signal<ThemeMode>(this.readInitial());

 constructor() {
 this.apply(this.theme());
 }

 isDark(): boolean {
 return this.theme() === 'dark';
 }

 setTheme(mode: ThemeMode): void {
 this.theme.set(mode);
 try {
 localStorage.setItem(STORAGE_KEY, mode);
 } catch {
 // ignore
 }
 this.apply(mode);
 }

 toggle(): void {
 this.setTheme(this.isDark() ? 'light' : 'dark');
 }

 private readInitial(): ThemeMode {
 try {
 const v = localStorage.getItem(STORAGE_KEY);
 if (v === 'dark' || v === 'light') return v;
 } catch {
 // ignore
 }
 return 'light';
 }

 private apply(mode: ThemeMode): void {
 const root = document.documentElement;
 root.setAttribute('data-theme', mode);
 root.classList.toggle('dark', mode === 'dark');
 root.style.colorScheme = mode;
 }
}

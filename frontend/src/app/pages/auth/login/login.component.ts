import { Component, OnInit, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { CmParentComponent } from '@common/parents/parent/parent.component';
import { AuthService } from '@common/services';
import { CmMessageService } from '@common/modules/message';

@Component({
  selector: 'app-login',
  standalone: false,
  templateUrl: './login.component.html',
  styleUrl: './login.component.scss'
})
export class LoginComponent extends CmParentComponent implements OnInit {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly message = inject(CmMessageService);

  /** true = 首次创建管理员；false = 登录 */
  needsSetup = signal(false);
  checking = signal(true);

  username = '';
  password = '';
  password2 = '';
  displayName = '';
  loading = signal(false);
  error = signal('');

  override ngOnInit(): void {
    if (this.auth.isLoggedIn()) {
      void this.router.navigateByUrl('/main/dashboard');
      return;
    }
    this.auth.setupStatus().subscribe({
      next: s => {
        this.needsSetup.set(!!s.needsSetup);
        this.checking.set(false);
      },
      error: () => {
        this.needsSetup.set(false);
        this.checking.set(false);
      }
    });
  }

  submit(): void {
    if (this.needsSetup()) {
      this.bootstrap();
    } else {
      this.login();
    }
  }

  private login(): void {
    if (!this.username.trim() || !this.password) {
      this.error.set('请填写用户名和密码');
      return;
    }
    this.loading.set(true);
    this.error.set('');
    this.auth.login(this.username.trim(), this.password).subscribe({
      next: () => {
        this.loading.set(false);
        this.message.success('登录成功');
        void this.router.navigateByUrl('/main/dashboard');
      },
      error: (err: Error) => {
        this.loading.set(false);
        const msg = err.message || '登录失败';
        if (msg.toLowerCase().includes('setup') || msg.includes('创建')) {
          this.needsSetup.set(true);
          this.error.set('请先创建管理员账号');
          return;
        }
        this.error.set(msg);
      }
    });
  }

  private bootstrap(): void {
    const u = this.username.trim();
    if (!u) {
      this.error.set('请填写登录名');
      return;
    }
    if (!/^[A-Za-z0-9_.-]{1,32}$/.test(u)) {
      this.error.set('登录名仅限字母、数字、_ - .，最长 32');
      return;
    }
    if (this.password.length < 10) {
      this.error.set('密码至少 10 位');
      return;
    }
    if (this.password !== this.password2) {
      this.error.set('两次密码不一致');
      return;
    }
    this.loading.set(true);
    this.error.set('');
    this.auth
      .bootstrap({
        username: u,
        password: this.password,
        displayName: this.displayName.trim() || u
      })
      .subscribe({
        next: () => {
          this.loading.set(false);
          this.message.success('管理员已创建');
          void this.router.navigateByUrl('/main/dashboard');
        },
        error: (err: Error) => {
          this.loading.set(false);
          this.error.set(err.message || '创建失败');
        }
      });
  }
}

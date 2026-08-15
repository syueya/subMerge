# SubMerge

自托管的 Clash 订阅合并与配置下发面板。

- 合并多个 Clash 订阅，节点按地区码添加前缀（如 `US-xxx`）
- 在网页中维护策略组、分流规则、订阅源和分享 Token
- 通过 `/subscribe/{token}` 向客户端提供订阅
- 提供 Geo 数据查询和更新页面（`/geo`）

推荐使用 Docker Compose 部署。应用运行后，首次在网页中创建管理员账号，再添加订阅源并发布配置即可。

## Docker Compose 部署（推荐）

已安装 Docker Engine 和 Compose 插件，并确认 `docker compose version` 能正常输出版本后，进入部署目录执行：

```bash
curl -fsSL https://github.com/syueya/subMerge/releases/latest/download/docker-compose.yml -o docker-compose.yml
docker compose up -d
```

`docker-compose.yml` 使用 `latest` 镜像标签，会拉取最新稳定版容器镜像。首次启动会把数据库、密钥和 Geo 数据写入当前目录的 `data/`，不需要额外创建 `.env`。

如需修改端口、时区或自行保管加密密钥，请在**首次启动前**于同一目录创建 `.env`：

```dotenv
SUBMERGE_PORT=8080
TZ=Asia/Shanghai

# 可选；至少 32 个字符。已有部署不能随意更换。
ENCRYPTION_KEY=
```

需要自行生成密钥时，可执行：

```bash
printf 'ENCRYPTION_KEY=%s\n' "$(openssl rand -hex 32)" > .env
```

启动后验活：

```bash
docker compose ps
curl http://127.0.0.1:8080/api/health
```

最后一条应返回包含 `"version"` 的 JSON。若未正常启动，查看日志：

```bash
docker compose logs --tail=200 submerge
```

浏览器访问 `http://服务器地址:8080`，首次打开时创建管理员账号。

首次使用空目录时，Geo 数据会在后台自动下载，通常需要 1–3 分钟；可在管理端 `/geo` 查看四个数据文件是否均为「可用」。下载失败不会阻止面板启动，可在 `/geo` 重新执行「更新数据」。

### 持久化数据与备份

Compose 文件使用相对于 `docker-compose.yml` 的目录挂载；不要删除 `data/`：

| 本机目录 | 容器目录 | 用途 |
|---|---|---|
| `./data` | `/app/data` | SQLite 数据库、加密盐和自动生成的密钥 |
| `./data/log` | `/app/log` | 应用日志 |
| `./data/geo` | `/app/defaults/geo` | Geo 数据 |
| `./bin` | `/app/runtime` | 在线更新下载的二进制及回滚文件 |

升级、迁移或修改密钥前，至少备份 `data/`：

```bash
tar -czf submerge-data-$(date +%F).tar.gz data
```

`ENCRYPTION_KEY`、`data/crypto.key` 和 `data/crypto.salt` 均与已加密的订阅源、Token 和 API Key 有关。更换或丢失它们会导致既有加密数据无法解密。

### 上线后的必要设置

登录后打开「设置 → 系统设置」：

1. 若通过域名或反向代理访问，设置「公开访问地址」为完整地址，例如 `https://sub.example.com`。
2. 填写反向代理的可信 IP/CIDR；保存后重启容器：`docker compose restart`。
3. HTTPS 反向代理场景开启 Cookie Secure；仅 HTTP 访问时保持关闭。
4. 按需调整订阅源请求、Geo 下载、出站代理、日志和刷新周期。

这些网页设置保存在 SQLite 中；`PUBLIC_BASE_URL`、`TRUSTED_PROXIES`、`COOKIE_SECURE`、Geo 和日志等同名环境变量不会生效。

不要把管理端口直接暴露到不受信任的公网。使用反向代理时，应同时配置 HTTPS、访问控制和防火墙规则；分享订阅仅使用 `/subscribe/{token}`。

### 更新

未使用管理端在线更新时，更新镜像只需：

```bash
docker compose pull && docker compose up -d
```

管理端在线更新会校验正式 Release 的签名，更新完成后容器自动重启，并将新二进制保存到 `./bin`。此后的 `./bin/submerge` 会优先于镜像内二进制；使用过在线更新的实例请继续在管理端完成后续更新。**不要删除 `data/`。**

## 不使用 Docker：Linux 二进制部署

Release 同时提供 `submerge-linux-amd64` 和 `submerge-linux-arm64`。下载与服务器架构相符的文件并改名为 `submerge`。以下示例使用 systemd；它会在在线更新完成后负责重启进程。

```bash
sudo install -d -m 755 /opt/submerge
sudo install -m 755 ./submerge-linux-amd64 /opt/submerge/submerge
if ! id -u submerge >/dev/null 2>&1; then
  sudo useradd --system --home /opt/submerge --shell /usr/sbin/nologin submerge
fi
sudo chown -R submerge:submerge /opt/submerge
```

创建 `/etc/systemd/system/submerge.service`：

```ini
[Unit]
Description=SubMerge
After=network-online.target
Wants=network-online.target

[Service]
User=submerge
Group=submerge
WorkingDirectory=/opt/submerge
Environment=APP_ENV=production
Environment=HTTP_ADDR=:8080
Environment=TZ=Asia/Shanghai
ExecStart=/opt/submerge/submerge
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
```

启动并验活：

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now submerge
curl http://127.0.0.1:8080/api/health
sudo journalctl -u submerge -f
```

二进制部署的数据目录分别为 `/opt/submerge/data`、`/opt/submerge/log` 和 `/opt/submerge/backend/defaults/geo`；首次启动会自动创建并初始化。请确保运行用户对它们有写权限，并将这几个目录纳入备份。

## 本地开发

需要 Go（版本以 [`backend/go.mod`](backend/go.mod) 为准）和 Node.js 24。

```bash
cp .env.example .env

# 终端一：后端 API
cd backend && go run .

# 终端二：React 前端 http://localhost:4202
cd fronted && npm ci && npm start
```

Windows 可双击 [`test/test-run.bat`](test/test-run.bat) 启动本地测试环境；它会将测试二进制和 npm 缓存写到 `test/bin/`。

本地开发路径由工作目录决定：从项目根运行后端时使用 `backend/data`、`backend/log`、`backend/defaults/geo`；在 `backend/` 目录运行时使用 `data`、`log`、`defaults/geo`。

## 发布与版本

根目录 [`VERSION`](VERSION) 是唯一版本来源。向 `main` 提交该文件的版本变更会触发 GitHub Actions：构建 React、Linux amd64/arm64 二进制、多架构容器镜像，生成签名更新清单，并创建 `v<版本>` Tag 与 GitHub Release。

如需同步发布 Docker Hub 镜像，可在 GitHub Actions 手动运行 [Docker Hub 工作流](.github/workflows/dockerhub.yml)。该工作流使用 `docker/Dockerfile` 构建并推送 `DOCKER_USERNAME/submerge:<版本>`；稳定版本还会更新 `latest`。运行前只需要配置 `DOCKER_USERNAME` 和 `DOCKER_PASSWORD` 两个 Actions secret。Docker Hub 镜像不包含在线更新签名公钥，管理端在线更新不会启用；升级请使用 `docker compose pull && docker compose up -d`。默认部署仍使用 GHCR，Docker Hub 镜像可通过 `SUBMERGE_IMAGE` 和 `SUBMERGE_TAG` 覆盖 Compose 中的默认值。

Release 附件包括：

- `docker-compose.yml`：使用 `latest` 容器镜像，可直接部署
- `submerge-linux-amd64`、`submerge-linux-arm64`：Linux 二进制
- `update-manifest.json` 与 `update-manifest.json.sig`：在线更新清单及签名

本地构建生产镜像时，保持仓库根目录作为构建上下文：

```bash
docker build -f docker/Dockerfile .
```

## 功能边界

- 单机 SQLite，适合自托管，不是多租户 SaaS
- 单管理员账号；自动化可使用 API Key，客户端使用分享 Token
- 分流规则在 Clash Meta / Clash Verge 客户端执行

## 目录

```text
submerge/
├── backend/          Go 服务与默认规则、Geo 数据
├── fronted/          React 管理面板
├── deploy/           Compose 模板
├── docker/           容器 Dockerfile、入口脚本和测试
├── scripts/          自动发布和工作流校验脚本
├── test/             本地测试启动脚本与可再生测试产物
├── VERSION
└── .github/workflows/ CI 与发布工作流
```

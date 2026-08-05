# SubMerge

小型 Clash 订阅合并与配置下发面板。

- 合并多源 Clash 订阅（节点按地区码加前缀，如 `US-xxx`）
- 网页分开展示：策略组管出口成员，分流规则管业务匹配；预览后发布
- 独立订阅链接，隐藏上游地址；令牌可绑定部分源
- 分流在 Clash Meta / Clash Verge 客户端执行
- 管理端提供 Geo 数据查询与更新页面（`/geo`）

版本见 `frontend/version.ts`（唯一来源；Docker/CI 构建时同步到 `backend/version/VERSION` 供后端 embed）。

## 本地开发

```bash
cp .env.example .env   # 改 ENCRYPTION_KEY（≥32 字符）

cd backend && go run .

# http://localhost:4200 → API :8080
cd frontend && npm start
```

首次打开网页创建管理员。探活：`GET /api/health`。

路径由工作目录决定（无 `DATA_DIR` 等环境变量）：

| 工作目录 | 数据 | 日志 |
|----------|------|------|
| 项目根 | `backend/data` | `backend/log` |
| `backend/` 或 Docker `/app` | `./data` | `./log` |

## Geo 数据

管理端 `/geo` 页面读取 Geo 目录下的四个文件，并显示文件大小、SHA256、MMDB build 时间或数据哈希版本。域名查询默认只匹配 `geosite.dat` 的分类；勾选 DNS 解析后，还会用解析出的 IP 查询 `geoip.dat`、`geoip.metadb` 和 `GeoLite2-ASN.mmdb`。

| 运行方式 | Geo 目录 |
|----------|----------|
| 本地开发（项目根 / `backend/`） | `backend/defaults/geo/`（仓库可带默认文件） |
| Docker（`WORKDIR=/app`） | `/app/defaults/geo`（**镜像不含默认数据**） |

页面的「更新数据」会从以下地址下载并校验后原子覆盖该目录，服务随后重载索引。地址可通过环境变量覆盖：`GEOIP_URL`、`GEOSITE_URL`、`GEODB_URL`、`GEOASN_URL`。运行进程必须对该目录有写权限。

Docker 部署请挂载 volume 到 `/app/defaults/geo`（见 compose 的 `./geo`）。首次启动若目录为空（或任一必需文件不可用），进程会**后台自动拉取一次**；失败只记日志，可稍后在 `/geo` 点「更新数据」或重启。不挂 volume 时容器重建会丢已下载文件并再次自动拉取。

`geosite.dat` 支持按分类反查域名条目；GeoIP 文件反查的是其实际保存的 CIDR。`geoip.metadb` 与 ASN 数据库不保存域名，因此不能从分类反查域名。


```bash
cd frontend && npm ci && npm run build
cd backend && go build -o ../bin/submerge .
../bin/submerge   # Windows: ..\bin\submerge.exe
```


## 主流程

```text
登录 → 配置订阅源 → 拉取合并 → 改策略组 → 改分流规则 → 发布
     → 创建分享 token → 客户端 /subscribe/{token}
```

- **策略组**（`/groups`）：出口容器。成员、测速方式；被规则引用为「目标出口」
- **分流规则**（`/rules`）：匹配条件。按业务分类在面板浏览；`category` 仅后台用，不写入 Clash

## 默认规则

空库时从 `backend/defaults/` 写入（已有数据不覆盖；改 YAML 需重新编译）：

- **策略组**：直连 / 拒绝 / 常用国家 / 其他国家
- **分流**：业务规则见 `defaults/rules.yaml`；系统规则由代码固定生成（广告→拒绝、国内 GEOIP→直连、MATCH→美国）

成员语法：`ALL`、`REGION:US`、`REGION:OTHER`、`SOURCE:源名`、`SOURCE:id:N`、`DIRECT`/`REJECT`、组名或节点名。

Clash 规则行只含 `TYPE,payload,target`（`MATCH,target`），不含业务分类。

## 安全

- 管理后台登录；上游 URL 加密存储；token 只存哈希
- 会话 Cookie：HttpOnly + SameSite=Lax；是否 Secure 由 `COOKIE_SECURE` 控制（默认 `false`）
- 数据库与密钥不要放静态目录；Docker 用独立 volume；管理端口勿裸暴露公网
- `ENCRYPTION_KEY` 至少 32 字符（`openssl rand -hex 32`）；`PUBLIC_BASE_URL` 填对外访问根地址（生成订阅链接用）
- `TRUSTED_PROXIES`：可信反代 IP/CIDR。管限流/审计的真实客户端 IP；无反代留空
- `COOKIE_SECURE`：`true` 时 Cookie 仅 HTTPS 发送。与 `TRUSTED_PROXIES` **独立**——HTTPS 反代通常两个都配；`http://IP` 访问保持默认 `false`
- 单管理员模型：首次 bootstrap 建号；无多用户 / 角色体系。自动化用 API Key 作用域，客户端用分享 Token

## 上线检查清单

### 1. 环境变量（必改）

| 变量 | 生产建议 |
|------|----------|
| `ENCRYPTION_KEY` | `openssl rand -hex 32`，≥32 字符；**切勿**沿用 compose 示例占位 |
| `APP_ENV` | `production` |
| `PUBLIC_BASE_URL` | 浏览器实际访问根地址（如 `https://sub.example.com`），影响订阅链接生成 |
| `COOKIE_SECURE` | 纯 HTTP/`IP:端口` 保持 `false`；HTTPS 反代设 `true` |
| `TRUSTED_PROXIES` | 有 Nginx/Caddy/NPM/Docker 网桥时填反代 CIDR；裸跑留空 |
| `SOURCE_REFRESH_INTERVAL` | 按需（小时）；改后重启 |
| `LOG_RETENTION_DAYS` | 建议 ≥7；磁盘紧可缩短 |
| `TZ` | 与运维时区一致，默认 `Asia/Shanghai` |

本地可直接改 `.env`；Docker 改 `docker-public/docker-submerge/docker-compose.yaml` 的 `environment`。

### 2. Docker Compose 示例改法

```yaml
environment:
  - APP_ENV=production
  - ENCRYPTION_KEY=<openssl rand -hex 32 的结果>
  - PUBLIC_BASE_URL=https://你的域名
  - TRUSTED_PROXIES=172.16.0.0/12   # 按实际反代网段改；无反代删此行或留空
  - COOKIE_SECURE=true              # 仅 HTTPS 时
  - SOURCE_REFRESH_INTERVAL=24
  - LOG_OUTPUT=both
  - LOG_RETENTION_DAYS=7
  - TZ=Asia/Shanghai
volumes:
  - ./data:/app/data   # SQLite 与密钥材料，务必备份
  - ./log:/app/log
  - ./geo:/app/defaults/geo  # 首次 /geo「更新数据」后保留；镜像不自带
```

管理端口不要直接裸暴露公网；更稳妥是反代只对内网开放面板，公网仅放行 `/subscribe/*`（按你的反代策略裁剪）。

### 3. 发布前冒烟

```text
1. GET /api/health → ok
2. 首次打开 → bootstrap 管理员（或已有账号登录）
3. （Docker）等约 1–3 分钟后管理端 /geo 四个资源 Available（空 volume 会自动 bootstrap；失败可手动「更新数据」）
4. 添加订阅源 → 拉取成功 → 节点有地区前缀
5. 策略组 / 分流规则可编辑
6. 发布版本成功
7. 创建分享 Token → 客户端 /subscribe/{token} 能拉到 YAML
8. 账户：改昵称 / 改密码 / 改头像成功
9. （可选）API Key 只读 scope 能调列表接口
```

### 4. 运维与备份

- **备份**：定期拷贝 `data/`（含 SQLite 与加密 salt）；升级或改 `ENCRYPTION_KEY` 前必须先备份
- **日志**：`log/` 按 `LOG_RETENTION_DAYS` 清理；排障时看审计与应用日志
- **Geo**：Docker 镜像不含默认 geo；挂载 `/app/defaults/geo`；首次缺失时启动后台自动下载，也可在 `/geo` 手动更新；进程需对该目录有写权限（entrypoint 会 chown）
- **密钥轮换**：更换 `ENCRYPTION_KEY` 会导致已加密字段无法解密，需有迁移方案；日常不要随意改

### 5. 已知产品边界

- 单机 SQLite，适合自托管；非多租户 SaaS
- 单管理员；协作可用共享账号或 API Key，无「普通成员」角色
- Docker/CI 构建文件在同级 `docker-public/docker-submerge/`

## 技术栈

Angular + Material + Tailwind · Go + Gin + SQLite · Docker Compose

## 目录

```text
submerge/
├── backend/          Go 服务、defaults/、data/、log/
├── frontend/         Angular 面板
├── bin/              编译产物（gitignore）
├── .env.example
└── （Docker 部署文件见同级 docker-public/docker-submerge/）
```

Docker 部署文件：

- `docker-public/docker-submerge/Dockerfile`
- `docker-public/docker-submerge/docker-compose.yaml`
- `docker-public/.github/workflows/submerge-build.yml`

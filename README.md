# SubMerge

小型 Clash 订阅合并与配置下发面板。

- 合并多源 Clash 订阅（节点按地区码加前缀，如 `US-xxx`）
- 网页分开展示：策略组管出口成员，分流规则管业务匹配；预览后发布
- 独立订阅链接，隐藏上游地址；令牌可绑定部分源
- 分流在 Clash Meta / Clash Verge 客户端执行
- 管理端提供 Geo 数据查询与更新页面（`/geo`）

版本唯一来源是根目录 [`VERSION`](VERSION)。React、正式 Go 二进制、GitHub Release 和容器镜像均从这里读取。`main` 上提交的版本变更会自动创建对应的 `v<版本>` Tag 并发布；构建时版本通过链接参数写入二进制，部署和运行时不需要额外复制该文件。

## 本地开发

```bash
cp .env.example .env   # 新环境可留空 ENCRYPTION_KEY；已有部署请保留原密钥

cd backend && go run .

# React 重构版：http://localhost:4202 → API :8080
cd fronted && npm install && npm start
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

页面的「更新数据」会从系统设置中配置的地址下载并校验后原子覆盖该目录，服务随后重载索引。Geo 下载地址、订阅源 User-Agent、超时、刷新间隔、日志和出站代理统一在管理端「设置 → 系统设置」中修改，并保存到 SQLite；不再读取同名环境变量。运行进程必须对该目录有写权限。

升级前如果 `.env` 中配置过这些网页设置项（包括公开访问地址、可信反向代理、Cookie Secure、订阅源、Geo、日志和出站代理），升级后不会继续生效，请登录系统设置页面重新保存。已有数据库中的出站代理配置会保留；没有数据库配置时使用代码默认值。
Docker 部署请挂载 volume 到 `/app/defaults/geo`（见 compose 的 `./geo`）。首次启动若目录为空（或任一必需文件不可用），进程会**后台自动拉取一次**；失败只记日志，可稍后在 `/geo` 点「更新数据」或重启。不挂 volume 时容器重建会丢已下载文件并再次自动拉取。

`geosite.dat` 支持按分类反查域名条目；GeoIP 文件反查的是其实际保存的 CIDR。`geoip.metadb` 与 ASN 数据库不保存域名，因此不能从分类反查域名。


```bash
cd fronted && npm ci && npm run build
cd backend && go build -ldflags "-X github.com/submerge/submerge/backend/version.Value=$(tr -d '[:space:]' < ../VERSION)" -o ../bin/submerge .
../bin/submerge   # Windows: ..\bin\submerge.exe
```

React 产物会直接写入 `backend/internal/webui/dist/` 并由 Go `embed.FS` 打进二进制。部署时只需要 `submerge` 可执行文件，不需要额外复制网页文件。Linux amd64/arm64 二进制、签名更新清单和多架构容器镜像均由 GitHub Actions 自动发布。

使用 Docker 时，可直接从对应 GitHub Release 下载 `docker-compose.yml`，然后在该目录运行 `docker compose up -d`；该文件会拉取与该 Release 匹配的镜像版本。


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
- **分流**：业务规则见 `defaults/rules.yaml`；系统规则由代码固定生成（广告→拒绝、国内 GEOIP→直连、MATCH→香港）

成员语法：`ALL`、`REGION:US`、`REGION:OTHER`、`SOURCE:源名`、`SOURCE:id:N`、`DIRECT`/`REJECT`、组名或节点名。

Clash 规则行只含 `TYPE,payload,target`（`MATCH,target`），不含业务分类。

## 安全

- 管理后台登录；上游 URL 加密存储；token 只存哈希
- 会话 Cookie：HttpOnly + SameSite=Lax；Secure 开关在「设置 → 系统设置」中配置，HTTPS 反代通常开启
- 数据库与密钥不要放静态目录；Docker 用独立 volume；管理端口勿裸暴露公网
- `ENCRYPTION_KEY` 可显式配置（建议生产环境使用外部 Secret）；留空时首次启动会在 `data/crypto.key` 自动生成 32 字节随机密钥。已有部署不要清空或更换原密钥
- 可信反代 IP/CIDR 和公开访问地址在「设置 → 系统设置」中配置；可信代理变更后需重启服务。无反代时可信代理留空。
- Cookie Secure 与可信代理独立：HTTPS 反代通常开启 Secure，纯 HTTP 访问保持关闭。
- 单管理员模型：首次 bootstrap 建号；无多用户 / 角色体系。自动化用 API Key 作用域，客户端用分享 Token

## 上线检查清单

### 1. 环境变量（必改）

| 变量 | 生产建议 |
|------|----------|
| `ENCRYPTION_KEY` | 新环境可省略，首次启动自动生成并保存到 `data/crypto.key`；已有部署继续使用原密钥，生产环境建议使用外部 Secret |
| `APP_ENV` | `production` |
| `TZ` | 与运维时区一致，默认 `Asia/Shanghai` |

本地可直接改 `.env`；Docker 改仓库内 [`deploy/docker-compose.yml`](deploy/docker-compose.yml) 的 `environment`。公开访问地址、可信反向代理和 Cookie Secure，以及订阅源、Geo、日志和出站代理请在「设置 → 系统设置」修改；可信反向代理变更后需要重启服务。

### 2. Docker Compose 示例改法

```yaml
environment:
  - APP_ENV=production
  # 新环境可省略 ENCRYPTION_KEY；首次启动会在 /app/data/crypto.key 生成
  # 已有部署请继续注入原 ENCRYPTION_KEY，不要改成空值
  - TZ=Asia/Shanghai
volumes:
  - ./data:/app/data       # SQLite 与密钥材料，务必备份
  - ./data/log:/app/log
  - ./data/geo:/app/defaults/geo  # 首次 /geo「更新数据」后保留；镜像不自带
  - ./bin:/app/runtime     # 在线更新后的二进制；必须持久化
```

管理端口不要直接裸暴露公网；更稳妥是反代只对内网开放面板，公网仅放行 `/subscribe/*`（按你的反代策略裁剪）。

### 3. 发布前冒烟

```text
1. GET /api/health → ok
2. 首次打开 → bootstrap 管理员（或已有账号登录）
3. （Docker）等待约 1–3 分钟，直到管理端 /geo 的四个资源显示“可用”（空 volume 会自动初始化；失败时可手动点击「更新数据」）
4. 添加订阅源 → 拉取成功 → 节点有地区前缀
5. 策略组 / 分流规则可编辑
6. 发布版本成功
7. 创建分享 Token → 客户端 /subscribe/{token} 能拉到 YAML
8. 账户：改昵称 / 改密码 / 改头像成功
9. （可选）API Key 只读 scope 能调列表接口
```

### 4. 运维与备份

- **备份**：定期拷贝 `data/`（含 SQLite、`crypto.salt` 与自动生成的 `crypto.key`）；升级或改 `ENCRYPTION_KEY` 前必须先备份
- **日志**：`log/` 的保留天数在「设置 → 系统设置」中配置；排障时看审计与应用日志
- **Geo**：Docker 镜像不含默认 geo；挂载 `/app/defaults/geo`；首次缺失时启动后台自动下载，也可在 `/geo` 手动更新；进程需对该目录有写权限（entrypoint 会 chown）
- **密钥轮换**：更换 `ENCRYPTION_KEY` 或删除自动生成的 `data/crypto.key` 会导致已加密字段无法解密，需有迁移方案；日常不要随意改

### 5. 已知产品边界

- 单机 SQLite，适合自托管；非多租户 SaaS
- 单管理员；协作可用共享账号或 API Key，无「普通成员」角色
- 在线更新目前发布 Linux amd64/arm64 资产；其他平台的开发构建不会安装不匹配资产

## 技术栈

React + shadcn/ui + Tailwind · Go + Gin + SQLite · Docker Compose

## 目录

```text
submerge/
├── backend/          Go 服务、defaults/、data/、log/
├── fronted/          React + shadcn/ui 面板
├── docker/           容器入口脚本
├── scripts/          可复现发布构建脚本
├── Dockerfile
├── deploy/           Docker Compose 配置
│   └── docker-compose.yml
├── test/             本地测试启动脚本与可再生测试产物
│   └── test-run.bat
├── .env.example
└── .github/workflows CI 与发布工作流
```

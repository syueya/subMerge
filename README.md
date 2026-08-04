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
cd frontend && npm start   # http://localhost:4200 → API :8080
```

首次打开网页创建管理员。探活：`GET /api/health`。

路径由工作目录决定（无 `DATA_DIR` 等环境变量）：

| 工作目录 | 数据 | 日志 |
|----------|------|------|
| 项目根 | `backend/data` | `backend/log` |
| `backend/` 或 Docker `/app` | `./data` | `./log` |

## Geo 数据

管理端 `/geo` 页面读取 `backend/defaults/geo/` 下的四个文件，并显示文件大小、SHA256、MMDB build 时间或数据哈希版本。域名查询默认只匹配 `geosite.dat` 的分类；勾选 DNS 解析后，还会用解析出的 IP 查询 `geoip.dat`、`geoip.metadb` 和 `GeoLite2-ASN.mmdb`。

页面的「更新数据」会从以下地址下载并校验后原子覆盖 `backend/defaults/geo/`，服务随后重载索引。地址可通过环境变量覆盖：`GEOIP_URL`、`GEOSITE_URL`、`GEODB_URL`、`GEOASN_URL`。运行进程必须对该目录有写权限。

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

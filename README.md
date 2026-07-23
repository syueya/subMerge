# SubMerge

小型 Clash 订阅合并与配置下发面板。

- 合并多源 Clash 订阅（节点按地区码加前缀，如 `US-xxx`）
- 网页维护策略组 / 分流规则，预览后发布
- 独立订阅链接，隐藏上游地址；令牌可绑定部分源
- 分流在 Clash Meta / Clash Verge 客户端执行

版本见 `backend/version/VERSION`。

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

## 本机二进制

```bash
cd frontend && npm ci && npm run build
cd backend && go build -o ../bin/submerge .
./bin/submerge   # Windows: bin\submerge.exe
```

## Docker

先编产物，再打镜像（镜像内不编 Node/Go）：

```bash
cp .env.example .env

cd frontend && npm ci && npm run build
cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -ldflags="-s -w" -o ../bin/submerge-linux .
docker compose build && docker compose up -d
```

| 路径 | 说明 |
|------|------|
| `/app/submerge` | 二进制 |
| `/app/frontend/dist/submerge/browser` | 前端 |
| `/app/data` | SQLite（volume `submerge-data`） |
| `/app/log` | 日志（volume `submerge-log`） |

```bash
docker compose logs -f
docker compose down       # 保留数据
docker compose down -v    # 清空 volume
```

## 主流程

```text
登录 → 配置订阅源 → 拉取合并 → 改策略组/规则 → 发布
     → 创建分享 token → 客户端 /subscribe/{token}
```

## 默认规则

空库时从 `backend/defaults/` 写入（已有数据不覆盖；改 YAML 需重新编译）：

- **策略组**：直连 / 拒绝 / 常用国家 / 其他国家
- **分流**：广告→拒绝；海外 AI/Netflix→美国；YouTube/GitHub→日本；电报/社交→香港；国内→直连；MATCH→日本

成员语法：`ALL`、`REGION:US`、`REGION:OTHER`、`DIRECT`/`REJECT`、组名或节点名。

## 安全

- 管理后台登录；上游 URL 加密存储；token 只存哈希
- 生产 `APP_ENV=production` 时 Cookie 带 Secure（需 HTTPS）
- 数据库与密钥不要放静态目录；Docker 用独立 volume

## 技术栈

Angular + Material + Tailwind · Go + Gin + SQLite · Docker Compose

## 目录

```text
submerge/
├── backend/          Go 服务、defaults/、data/、log/
├── frontend/         Angular 面板
├── bin/              编译产物（gitignore）
├── Dockerfile
├── docker-compose.yml
└── .env.example
```

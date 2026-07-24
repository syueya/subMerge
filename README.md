# SubMerge

小型 Clash 订阅合并与配置下发面板。

- 合并多源 Clash 订阅（节点按地区码加前缀，如 `US-xxx`）
- 网页分开展示：策略组管出口成员，分流规则管业务匹配；预览后发布
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

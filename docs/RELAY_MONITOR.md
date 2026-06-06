# 中转站监控（Relay Monitor）

监控**外部**中转站（运行 sub2api / newapi 的站点）对外公布的**分组倍率**，定时抓取、
对比、记录涨/跌变化。**红色代表涨、绿色代表跌**。

> 与「渠道监控（Channel Monitor）」相互独立：渠道监控测的是自己挂的上游账号心跳；
> 中转站监控抓的是别家中转站的公开分组倍率。两者复用同一套 SSRF 防护与 HTTP client。

入口：admin 侧边栏 **渠道管理 → 中转站监控**。

---

## 功能

三个 tab：

- **倍率总览（默认页）**：列出所有被监控分组的**当前倍率**（即使从未变化）；
  变化过的显示涨/跌幅并排在最前（按变化时间倒序）。支持按**套餐**筛选。涨红跌绿。
- **变化公告**：倍率变化历史表（类型、站点、系统、厂商、分组、套餐、原倍率、当前倍率、
  涨跌幅、时间、公告内容）；全部 / 只看涨 / 只看跌 + 搜索 + 「探测全部」。顶部三张卡
  （涨公告数 / 跌公告数 / 最近刷新）。
- **站点管理**：增删改被监控站点；每个站点可**只勾选要监控的分组**（不是全站所有分组）；
  支持「拉取分组列表」勾选。

附加：

- **厂商下拉**：datalist 预置 OpenAI / Claude / Gemini / Grok / DeepSeek，仍可自定义输入。
- **套餐档位**：按分组名自动识别 `team / enterprise / max / ultra / pro / plus / free`，
  以彩色徽章展示，可在总览页按档位筛选。
- **定时探测**：每个启用的站点按各自间隔独立定时抓取。

---

## 探测原理

按站点的「系统类型」选择解析器：

| 系统 | 接口 | 凭证 | 解析字段 |
|------|------|------|----------|
| **sub2api** | 先 `POST {base}/api/v1/auth/login` 拿 JWT，再 `GET {base}/api/v1/groups/available` | **邮箱 + 密码**（别人的站不开放 API 密钥，只能账号登录） | `data[].rate_multiplier` |
| **newapi** | `GET {base}/api/pricing` | 不需要（公开） | 顶层 `group_ratio` map |

- sub2api 的分组倍率接口需要**登录态**：探测时用配置的**邮箱+密码**登录目标站
  （`/api/v1/auth/login` → `data.access_token`），再用该 JWT 抓分组。JWT 过期由每次
  探测重新登录自动处理；目标站开启 2FA 时无法自动登录，会返回探测失败。
- newapi 的「模型广场」`/api/pricing` 公开，留空凭证即可。
- 密码以 **AES-256-GCM** 加密存库（复用项目现有 `SecretEncryptor`）；登录邮箱
  存 `auth_account` 列（明文）。
- 抓取走 SSRF-safe HTTP client：强制 https、拦截 loopback/私网/云元数据地址。

### 涨跌判定

每次探测把当前倍率与上次**快照**对比：
- 分组首次见到 → 只写快照，不算变化；
- 倍率变大 → `up`（涨），变小 → `down`（跌），生成一条公告
  `分组倍率从 0.005x 变为 0.001x`；
- 浮点容差 `1e-9`，避免抖动误报。

每站变化历史保留最新 **500** 条，超出探测后自动裁剪。

---

## 数据表（迁移 `149` + `150`）

| 表 | 说明 |
|----|------|
| `relay_monitors` | 站点配置（name, system, base_url, vendor, **auth_account**, credential_encrypted, watched_groups, enabled, interval_seconds, last_checked_at, last_error） |
| `relay_rate_snapshots` | 当前倍率快照，每监控每分组一行，唯一 `(monitor_id, group_name)`，用于对比 |
| `relay_rate_changes` | 倍率变化历史（涨/跌公告） |

- `149_add_relay_monitors.sql`：建上述 3 张表。
- `150_relay_monitor_auth_account.sql`：给 `relay_monitors` 加 `auth_account` 列（sub2api 登录邮箱）。

子表对 `relay_monitors` 外键 `ON DELETE CASCADE`：删站点自动清理快照与历史。
两个迁移都是纯新增（`CREATE TABLE IF NOT EXISTS` / `ADD COLUMN IF NOT EXISTS`），对既有数据无副作用。

---

## API（admin，`/api/v1/admin/relay-monitors`）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `` | 列出监控站点（分页） |
| POST | `` | 新增站点 |
| GET | `/:id` | 查询站点 |
| PUT | `/:id` | 更新站点（credential 空串=不改） |
| DELETE | `/:id` | 删除站点 |
| POST | `/:id/probe` | 探测单个站点，返回当前倍率 + 本次变化 |
| POST | `/probe-all` | 探测全部启用站点 |
| POST | `/fetch-groups` | 用给定配置抓目标站全部分组+倍率（不落库），供勾选 |
| GET | `/overview` | 倍率总览：所有被跟踪分组的当前倍率 + 最近一次变化（变化过的排前），受 `search` 过滤 |
| GET | `/changes` | 倍率变化历史（`direction` / `search` / `monitor_id` 过滤 + 分页） |
| GET | `/summary` | 涨/跌公告数量（顶部统计卡，受 `search` 过滤） |

> sub2api 站点的请求体含 `auth_account`（登录邮箱）+ `credential`（登录密码，更新时留空=不改）。

---

## 本地联调 runbook

```bash
# 1. 起 PostgreSQL + Redis（确认 5432 / 6379 在监听）
#    DEV_GUIDE 默认凭据：db=sub2api user=sub2api pass=sub2api

# 2. 准备配置
cd backend
cp ../deploy/config.example.yaml config.yaml
#    编辑 config.yaml：database 段填 sub2api/sub2api/sub2api；redis 填 localhost:6379

# 3. 启动服务（首次启动自动应用迁移 149/150，建表 + 加 auth_account 列）
go run ./cmd/server
```

启动后：

1. 进 admin → 侧边栏 **渠道管理 → 中转站监控**。
2. 切到 **站点管理** → 新增站点：
   - system = `sub2api`
   - base_url = 目标站（如 `https://yysubapi.yangyangnj.top`）
   - **登录邮箱 + 密码** = 你在该站的账号密码
   - 厂商 = 下拉选 OpenAI / Claude / …
3. 点 **拉取分组列表** —— 能正确列出分组+倍率即说明登录与解析都正确。
4. 勾选要盯的分组保存 → 点 **探测** → 改一次该站某分组倍率 → 再 **探测全部**，
   看「倍率总览」与「变化公告」是否出现涨/跌（红/绿）。

> 若目标站返回结构与假设不同，把实际响应贴出来即可快速调整解析器
> （`backend/internal/service/relay_monitor_probe.go` 的 `parseSub2APILoginToken` /
> `parseSub2APIGroups` / `parseNewAPIGroups`，已有对应单元测试 `relay_monitor_probe_test.go`）。

---

## Docker 部署 / 更新（生产）

线上以官方镜像 `weishaw/sub2api:latest` 跑在 compose（`/volume2/docker/sub2api`）。
部署改造版 = 用本分支构建一个自定义镜像替换。

```bash
# 0. 先备份（DB 逻辑导出 + app 数据）
TS=$(date +%Y%m%d-%H%M%S); BK=/volume2/docker/sub2api/backups/relay-$TS; mkdir -p "$BK"
docker exec sub2api-postgres sh -c 'PGPASSWORD=$POSTGRES_PASSWORD pg_dump -U sub2api -d sub2api' | gzip > "$BK/db.sql.gz"
tar czf "$BK/app-data.tar.gz" -C /volume2/docker/sub2api data
cp /volume2/docker/sub2api/docker-compose.yaml "$BK/"

# 1. 拉取最新分支并构建镜像
cd /root && rm -rf sub2api-build
git clone --depth 1 -b feature/relay-monitor https://github.com/yy1105384898/sub2api.git sub2api-build
cd sub2api-build && docker build -t weishaw/sub2api:relay-monitor .

# 2. compose 指向新镜像（首次切换时改，后续重建可跳过）
cd /volume2/docker/sub2api
sed -i 's#image: weishaw/sub2api:latest#image: weishaw/sub2api:relay-monitor#' docker-compose.yaml

# 3. 重建容器（迁移 149/150 启动自动应用）
docker compose up -d sub2api

# 4. 验证
docker logs sub2api 2>&1 | grep -iE 'relay_monitor|migrat|started' | tail
docker exec sub2api-postgres sh -c 'PGPASSWORD=$POSTGRES_PASSWORD psql -U sub2api -d sub2api -tAc "select tablename from pg_tables where tablename like '\''relay_%'\''"'
```

**回滚**：把 compose 的 `weishaw/sub2api:relay-monitor` 换回 `weishaw/sub2api:latest`，
`docker compose up -d sub2api` 即可；迁移只增表/列，回滚旧镜像无副作用。

> Dockerfile 默认 `GOPROXY=goproxy.cn`，国内服务器构建无需额外配置。前端用
> `pnpm install --frozen-lockfile`，依赖 `frontend/pnpm-lock.yaml`（勿用 pnpm 11 改写它）。

---

## 代码位置

| 层 | 文件 |
|----|------|
| ent schema | `backend/ent/schema/relay_monitor.go` `relay_rate_snapshot.go` `relay_rate_change.go` |
| 迁移 | `backend/migrations/149_add_relay_monitors.sql` `150_relay_monitor_auth_account.sql` |
| repository | `backend/internal/repository/relay_monitor_repo.go` |
| service | `backend/internal/service/relay_monitor*.go`（探测/登录在 `relay_monitor_probe.go`） |
| handler | `backend/internal/handler/admin/relay_monitor_handler.go` |
| 路由 | `backend/internal/server/routes/admin.go`（`registerRelayMonitorRoutes`） |
| 前端页面 | `frontend/src/views/admin/RelayMonitorView.vue` |
| 前端 API | `frontend/src/api/admin/relayMonitor.ts` |
| i18n | `frontend/src/i18n/locales/{zh,en}.ts`（`admin.relayMonitor.*` / `nav.relayMonitor`） |

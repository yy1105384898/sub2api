# 中转站监控（Relay Monitor）

监控**外部**中转站（运行 sub2api / newapi 的站点）对外公布的**分组倍率**，定时抓取、
对比、记录涨/跌变化。**红色代表涨、绿色代表跌**。

> 与「渠道监控（Channel Monitor）」相互独立：渠道监控测的是自己挂的上游账号心跳；
> 中转站监控抓的是别家中转站的公开分组倍率。两者复用同一套 SSRF 防护与 HTTP client。

入口：admin 侧边栏 **渠道管理 → 中转站监控**。

---

## 功能

- **公告看板**：顶部三张卡（涨公告数 / 跌公告数 / 最近刷新），下面是倍率变化历史表
  （类型、站点、系统、厂商、分组、原倍率、当前倍率、涨跌幅、时间、公告内容），涨红跌绿。
- **筛选**：全部 / 只看涨 / 只看跌 + 站点/厂商/分组搜索 + 「探测全部」按钮。
- **站点管理**：增删改被监控站点；每个站点可**只勾选要监控的分组**（不是全站所有分组）。
- **定时探测**：每个启用的站点按各自间隔独立定时抓取。

---

## 探测原理

按站点的「系统类型」选择解析器：

| 系统 | 接口 | 凭证 | 解析字段 |
|------|------|------|----------|
| **sub2api** | `GET {base}/api/v1/groups/available` | **需要** `Authorization: Bearer <token>` | `data[].rate_multiplier` |
| **newapi** | `GET {base}/api/pricing` | 不需要（公开） | 顶层 `group_ratio` map |

- sub2api 的分组倍率接口需要**登录态**，所以监控 sub2api 站点时必须给该站配一个访问 token。
  token 形式与 sub2api 这边一致（用户登录后拿到的 JWT）。
- newapi 的「模型广场」`/api/pricing` 公开，留空凭证即可。
- 凭证以 **AES-256-GCM** 加密存库（复用项目现有 `SecretEncryptor`）。
- 抓取走 SSRF-safe HTTP client：强制 https、拦截 loopback/私网/云元数据地址。

### 涨跌判定

每次探测把当前倍率与上次**快照**对比：
- 分组首次见到 → 只写快照，不算变化；
- 倍率变大 → `up`（涨），变小 → `down`（跌），生成一条公告
  `分组倍率从 0.005x 变为 0.001x`；
- 浮点容差 `1e-9`，避免抖动误报。

每站变化历史保留最新 **500** 条，超出探测后自动裁剪。

---

## 数据表（迁移 `149_add_relay_monitors.sql`）

| 表 | 说明 |
|----|------|
| `relay_monitors` | 站点配置（name, system, base_url, vendor, credential_encrypted, watched_groups, enabled, interval_seconds, last_checked_at, last_error） |
| `relay_rate_snapshots` | 当前倍率快照，每监控每分组一行，唯一 `(monitor_id, group_name)`，用于对比 |
| `relay_rate_changes` | 倍率变化历史（涨/跌公告） |

子表对 `relay_monitors` 外键 `ON DELETE CASCADE`：删站点自动清理快照与历史。

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
| GET | `/changes` | 倍率变化历史（`direction` / `search` / `monitor_id` 过滤 + 分页） |
| GET | `/summary` | 涨/跌公告数量（顶部统计卡，受 `search` 过滤） |

---

## 本地联调 runbook

```bash
# 1. 起 PostgreSQL + Redis（确认 5432 / 6379 在监听）
#    DEV_GUIDE 默认凭据：db=sub2api user=sub2api pass=sub2api

# 2. 准备配置
cd backend
cp ../deploy/config.example.yaml config.yaml
#    编辑 config.yaml：database 段填 sub2api/sub2api/sub2api；redis 填 localhost:6379

# 3. 启动服务（首次启动自动应用迁移 149，建出 3 张 relay_ 表）
go run ./cmd/server
```

启动后：

1. 进 admin → 侧边栏 **渠道管理 → 中转站监控**。
2. 切到 **站点管理** → 新增站点：
   - system = `sub2api`
   - base_url = 你的实例（如 `https://yysubapi.yangyangnj.top`）
   - 凭证 = 在该站登录后的 token
3. 点 **拉取分组列表** —— 能正确列出分组+倍率即说明解析正确。
4. 勾选要盯的分组保存 → 点 **探测** → 改一次该站某分组倍率 → 再 **探测全部**，
   看公告看板是否出现涨/跌记录（红/绿）。

> 若目标站的 token 形式或返回结构与上述假设不同，把实际响应贴出来即可快速调整解析器
> （`backend/internal/service/relay_monitor_probe.go` 的 `parseSub2APIGroups` /
> `parseNewAPIGroups`，已有对应单元测试 `relay_monitor_probe_test.go`）。

---

## 代码位置

| 层 | 文件 |
|----|------|
| ent schema | `backend/ent/schema/relay_monitor.go` `relay_rate_snapshot.go` `relay_rate_change.go` |
| 迁移 | `backend/migrations/149_add_relay_monitors.sql` |
| repository | `backend/internal/repository/relay_monitor_repo.go` |
| service | `backend/internal/service/relay_monitor*.go` |
| handler | `backend/internal/handler/admin/relay_monitor_handler.go` |
| 路由 | `backend/internal/server/routes/admin.go`（`registerRelayMonitorRoutes`） |
| 前端页面 | `frontend/src/views/admin/RelayMonitorView.vue` |
| 前端 API | `frontend/src/api/admin/relayMonitor.ts` |
| i18n | `frontend/src/i18n/locales/{zh,en}.ts`（`admin.relayMonitor.*` / `nav.relayMonitor`） |

# Quality Audit

审计日期：2026-06-28

当前结论：

```text
tictick-hi 当前是 scaffold。
它不是合格 demo。
它不是 usable。
它更不是 production-safe。
```

本审计用于约束后续推进：先修质量底座，再做业务扩展。

## 1. 分级规则

```text
scaffold        有骨架，但不可作为 demo
demo            能演示主链路
usable          能支撑真实工作
production-safe 可安全长期运行
done            用户确认关闭
```

任何模块如果没有对应验收和检查，不能升级等级。

## 2. 当前模块评级

| 模块 | 当前等级 | 处理 | 主要问题 |
| --- | --- | --- | --- |
| 架构文档 | usable | 保留 | 还需要随实现持续校准 |
| Go 子命令 | scaffold | 保留后收敛 | 入口可用，但配置、日志、错误边界粗 |
| Docker Compose | demo | 保留 | 运行形态对，`scripts/stage8-smoke.sh` 已覆盖一键构建启动和全链路 smoke，`scripts/stage8-sigterm-smoke.sh` 已覆盖 data sync / backtest / trading / notify 容器 SIGTERM 收尾；仍缺生产运行手册、备份/恢复和外部依赖韧性验证 |
| PostgreSQL migrations | scaffold | 保留后加强 | `0011_domain_constraints.sql` 已补充核心 domain CHECK，`0012_referential_constraints.sql` 已补充核心事实表 FK / composite unique，`0016_worker_lease_constraints.sql` 已补充 worker lease 字段一致性 CHECK，`0017_strategy_intent_parent_constraints.sql` 已补充 `strategy_intents` 新增/更新时的多态父任务归属约束，`0018_strategy_intent_parent_delete_guards.sql` 已补充父任务删除防 orphan 保护，`0019_task_terminal_timestamp_constraints.sql` 已补充任务终态 `finished_at` 一致性约束，`0020_validate_worker_lease_constraints.sql` 已修补历史半截 lease 并 VALIDATE worker lease CHECK，`0021_task_status_transition_guards.sql` 已补充 data sync / backtest / trading 核心状态流转 trigger，`scripts/stage8-migration-audit.sh` 已进入 Stage 8 smoke 并校验状态流转 trigger；仍缺完整统一状态机、数据迁移/回滚策略和全量历史数据验证 |
| API server | scaffold | 保留后加强 | 已按领域拆分，`/api/candles` 已返回 metadata，回测 / 交易创建已复用策略 schema 校验，系统写请求已有 CSRF 检查，错误响应已统一为 `code/message/error` 且 500 响应不再泄露内部错误；已知 API 资源路径的方法错误会返回 `405 method_not_allowed` 和 `Allow` header；`GET /api/system/api-contract` 已暴露基础 OpenAPI 3.1 request / response schema contract；`scripts/quality-gate.sh` 已纳入前端 API route 和核心 TypeScript DTO 字段与后端 contract 漂移硬检查；登录和系统管理写操作已有基础操作审计日志；仍缺 TS 类型自动生成、外部 OpenAPI validator、更细错误分类和生产级审计边界 |
| 登录会话 | demo | 保留后加强 | HttpOnly session cookie、CSRF double-submit 写保护、登录失败节流、当前操作员 session 列表和非当前 session 撤销已进入 API / 系统管理边界；登录成功 / 失败、退出和会话撤销会进入基础操作审计；仍缺持久化限流、密码策略、RBAC / 自保护规则和生产级设备上下文 |
| 数据同步 worker | demo | 保留后加强 | 能 claim、拉取、upsert 1m K 线并恢复游标，运行中会持续刷新 heartbeat / locked_until，heartbeat 丢失后会停止保存结果；临时市场数据错误记录为 retry 并释放 lease，永久失败会停用 sync / realtime 期望；用户可从研究页 retry failed 任务，retry 只接受 failed 状态并清理错误和 lease；用户 stop sync / realtime、runner 上下文取消和容器 SIGTERM 会释放 active lease；release / fail / pause 清锁语义已收敛到共享 helper；仍缺完整统一状态机、外部网络限流和真实恢复压测 |
| CandleProvider | demo | 保留后加强 | 已统一 native / 1m 聚合、来源和缺口 metadata，查询 limit 已有显式默认/上限，`from/to` 已校验顺序并按 interval 限制最大闭区间跨度，聚合 fallback 会返回 coverage 并标记基础窗口受限，PostgreSQL 集成测试覆盖基础聚合、缺口、默认最新窗口查询、超大 limit clamp 和 runner 侧闭合信号过滤；仍缺大范围性能压测、分页/游标和更多异常数据边界 |
| Binance / OKX K 线 adapter | demo | 保留后加强 | 能拉 K 线，Binance 支持多 base URL fallback，EOF/超时/429/5xx/OKX 50011 已分类为临时错误并由 sync runner 有限重试，错误摘要不泄露完整请求 URL；仍缺全局限流、真实网络韧性和更完整交易所业务码分类 |
| 研究页 | demo | 保留后打磨 | 列表在上、图表在下，任务表格错误列、failed retry 操作和图表高度已有前端约束；图表面板已用固定 flex 剩余空间和面板边界切断高度反馈，lightweight-charts 外层视口使用 `contain: strict`、固定 100% CSS 边界、视口 `clientWidth/clientHeight` 单向输入和面板可用高度上限，JS 不再信任 `ResizeObserver` / bounds height 作为图表高度输入，也不再把 chart 内部高度反写到 root/canvas，并由 headless 页面连续采样验证不再增高；显示 source / health / base interval；但交易对仍硬编码、图表研究能力仍薄 |
| 策略 registry / runtime | demo | 保留后加强 | 已有策略 schema 校验、默认参数规范化、order / notification intent 和边界门禁，仍缺策略沙箱、参数版本迁移和更多真实策略 |
| 回测 | demo | 保留后加强 | 已通过 CandleProvider 执行、`minute_replay` 以 `1m` 推进，策略输入前会丢弃未闭合 K 线，且 `gap/insufficient/limitedByBaseWindow` 不再进入策略输入；intent / order / result 落库，详情页展示 intent 和买卖点；runner 上下文取消和容器 SIGTERM 会释放 active lease 并复位为 pending；撮合模型、费用/滑点曲线、指标体系仍不可信 |
| 交易 runner | demo | 保留后加强 | 已通过 CandleProvider 取 K 线，策略输入前会丢弃未闭合 K 线，且 `gap/insufficient/limitedByBaseWindow` 不再进入策略输入；paper executor 落库 intent / order / execution / position / notification，running task claim 已按 `updated_at` 轮转避免旧任务长期占用队列，用户 pause、runner 上下文取消和容器 SIGTERM 会释放 active lease，live execute 已禁用；通知 intent 可经 local / webhook / email / Telegram / 飞书 provider 投递；仍缺可信风控、完整统一 worker lease 和实盘安全边界 |
| 实盘安全 | demo | 保留后加强 | 新建交易所账号凭据使用 `ENCRYPTION_KEY` + AES-GCM 加密保存，列表/API 不返回明文，live 任务创建校验账号启用和凭据状态；真实 testnet/sandbox live executor、幂等提交和生产密钥管理仍未完成 |
| 通知 | demo | 保留后加强 | NotificationIntent 已进入 notification outbox，`hi notify` 支持 local / webhook-demo / webhook / email / Telegram / 飞书 provider、失败重试和系统页 retry，delivered / failed / retry / runner 上下文取消会通过共享 lease helper 释放 outbox lock；真实 provider 采用 env-reference 凭据模型，密钥不进入 channel target；webhook / Telegram / 飞书支持真实 HTTP POST，email 支持 SMTP；notify 容器 SIGTERM 已由慢 webhook smoke 证明会释放 outbox lock；通道更新/删除、生产级模板/限流/回执、完整统一 worker lease 仍未完成 |
| 前端基础设施 | scaffold | 保留后加强 | Vue/Naive/Pinia/i18n/主题骨架存在，策略任务表单已由 schema 驱动并校验参数，路由页面已懒加载且生产入口 chunk 降到 500 kB 以下；概览页已改为真实聚合视图；仍缺系统性桌面/移动/主题视觉回归，整体业务体验仍需继续打磨 |
| 概览页 | demo | 保留后加强 | 已从现有 API 读取系统健康、数据同步、回测、交易和通知记录，展示关键数量、异常提醒、worker 健康和最近活动；仍缺时间窗口筛选、趋势图、操作入口和生产级监控语义 |
| 系统管理 / 运维健康 | demo | 保留后加强 | 操作台账号可创建和启停，当前操作员 session 可查看和撤销非当前会话，基础操作审计页/API 可查看登录和系统管理写操作，运维健康页/API 展示数据库、api、worker count、heartbeat 和 locked_until；仍缺 RBAC、自保护规则、不可篡改审计和生产监控 |
| 质量门禁 | demo | 保留后加强 | 阶段 0 硬门禁、策略边界检查、API contract route / field drift 检查、整体 scaffold 声明检查、Stage 8 smoke gate 和 data sync / backtest / trading / notify SIGTERM smoke 已通过；live executor/testnet、完整统一 worker lease、真实通知 provider 的生产启用边界和生产级登录安全作为后续风险审计保留 |

## 3. 必须先修的问题

### 阶段 0 Definition of Done：质量底座

目标等级：scaffold

范围内：

- 关闭当前质量门禁中的基础工程失败项：文件过大、`PageStub` 占位路由、质量脚本不可持续。
- 拆分 `internal/web/api/server.go`，让 API server 按领域组织，单文件低于硬上限。
- 拆分 `web/frontend/src/i18n/messages.ts`，让 i18n 文案按语言或领域组织，单文件低于硬上限。
- 建立并保留 `scripts/quality-gate.sh` 作为轻量质量门禁入口。
- README 明确当前整体等级只能是 `scaffold`，并指向交付协议、质量审计和实施计划。
- 本审计文档保留模块等级表，并在阶段结束时更新阶段 0 验收结果。

范围外：

- CandleProvider 和研究核心语义升级。
- 回测撮合可信度升级。
- paper/live executor 分离。
- 交易所账号真加密。
- 通知 provider/outbox。
- CSRF、防暴力破解、完整会话审计。

用户可见行为：

- 概览页不再是空泛 `PageStub`，但仍只能是 scaffold 级状态面板。
- 研究、回测、交易等既有页面行为不因文件拆分回退。

后端验收：

- API server 拆分后现有 API 路由行为不变。
- `go test ./...` 和 `go vet ./...` 通过。

前端验收：

- i18n 拆分后中英切换行为不变。
- `PageStub` 不再被路由引用。
- `pnpm run typecheck`、`pnpm run test`、`pnpm run build` 通过。

数据验收：

- 不引入新的 migration。
- 不改变现有数据语义。

安全验收：

- 不把 scaffold 说成 demo、usable 或 production-safe。
- 实盘密钥、live executor 和生产安全风险不在阶段 0 冒充关闭。

测试验收：

- `scripts/quality-gate.sh` 能稳定执行。
- 如果质量门禁仍因后续阶段风险失败，必须在阶段 0 验收结果中列为未关闭失败项。

质量门禁：

- `go test ./...`
- `go vet ./...`
- `scripts/quality-gate.sh`
- `cd web/frontend && pnpm run typecheck`
- `cd web/frontend && pnpm run test`
- `cd web/frontend && pnpm run build`

### 当前质量门禁结果

执行命令：

```text
scripts/quality-gate.sh
```

当前结果：通过。

已关闭基础失败项：

- `internal/web/api/server.go` 已拆分，生产文件低于 Go 文件硬上限。
- `web/frontend/src/i18n/messages.ts` 已拆分，i18n 入口和语言文件低于 TypeScript 文件硬上限。
- `internal/backtest/runner.go` 不再使用 `float64` / `ParseFloat` / `FormatFloat` 处理交易事实。
- 前端路由不再引用 `PageStub`。
- `scripts/quality-gate.sh` 已建立，并能稳定执行 file size、trading float、strategy boundary、API contract route drift、阶段 0 scaffold marker 检查。

后续风险审计仍保留：

- 交易所账号密钥 digest 风险已在阶段 6 切片关闭到 `demo`：新建账号使用 `ENCRYPTION_KEY` + AES-GCM 加密，历史非 AES-GCM 行标记为 `legacy`。
- live executor 仍禁用，testnet/sandbox、幂等提交、真实交易所提交和生产密钥管理仍未建立。

这些风险关闭前，项目整体仍为 `scaffold`，但它们不阻断阶段 0 的基础工程质量验收。

### 阶段 0 当前验收快照

执行时间：2026-06-27

通过：

- `go test ./...`
- `go vet ./...`
- `scripts/quality-gate.sh`
- `cd web/frontend && pnpm run typecheck`
- `cd web/frontend && pnpm run test`
- `cd web/frontend && pnpm run build`

失败：

- 无。

后续风险审计：

- 交易所账号密钥 digest 风险已在阶段 6 切片关闭到 `demo`；历史非 AES-GCM 行标记为 `legacy`。
- live executor 仍禁用，testnet/sandbox、幂等提交、真实交易所提交和生产密钥管理仍未建立。

阶段 0 结论：

- 阶段 0 质量底座验收通过。
- 项目整体仍为 `scaffold`，不能称为 demo、usable、production-safe 或完成。

### P0：不能把 scaffold 说成 demo

项目状态必须被明确标为 `scaffold`。

关闭条件：

- README 指向 AI 交付协议。
- 每次最终回复使用固定格式。
- `docs/quality-audit.md` 持续更新。

### P0：CandleProvider 必须独立成核心服务

阶段 1 状态：

- 已建立 `internal/data.CandleProvider`。
- `/api/candles` 返回 `candles`、`source`、`requestedInterval`、`baseInterval`、`health`、`gaps`。
- `1m` 返回 native；`5m / 15m / 1h / 4h / 1d` 可从 `1m` 聚合。
- 同周期 native 不健康时会尝试回退到 `1m` 聚合；无法回退时保留 native + gap 状态。
- 回测 runner 和交易 runner 已改为通过 CandleProvider 结果取 K 线。
- PostgreSQL 集成测试已覆盖从 `market_candles` 聚合 `1m -> 5m`、基础缺口 metadata、native gap 查询和无时间范围时默认返回最新窗口。
- 查询 limit 已收敛到 `DefaultCandleLimit=1000`、`MaxCandleLimit=5000`；API 超过上限返回 `400`，store 直接调用时 clamp 到最大上限而不是静默降回默认值。
- `/api/candles` 已校验 interval、`from <= to`，并按闭区间语义限制 `from/to` 最大跨度为 `(MaxCandleLimit - 1) * interval`；倒置或超大时间范围返回 `400`。
- CandleProvider 返回 `coverage` 元数据；高周期从 `1m` 聚合且基础窗口被 `MaxCandleLimit` 截断时，`limitedByBaseWindow=true`，研究页显示窗口受限，避免静默冒充完整窗口。
- 仍缺大范围性能压测、分页/游标和更多异常数据边界；闭合周期信号已有 runner 侧基础过滤，未闭合 K 线不再进入策略输入。

关闭条件：

- 建立独立 CandleProvider 或等价服务。
- 返回 candles、source、base_interval、gap 信息。
- native 数据不健康时可回退到更小周期聚合。
- 数据不足时明确返回缺口，不伪造 K 线。
- 图表、回测、交易 runner 统一调用。
- 覆盖单元测试和 PostgreSQL 集成测试。

### P0：worker lease 不能只有 claim

现状问题：

- claim 时写入 `heartbeat_at`。
- 数据同步、回测、交易 worker 均通过 `internal/workerlease.RunWithHeartbeat` 运行同一套 heartbeat loop。
- heartbeat 丢失后，数据同步 worker 会在保存 K 线前重新确认 lease，避免继续写入已失去租约的结果。
- 数据同步 stop sync / stop realtime 和交易 pause 会清理 `locked_by`、`locked_until`、`heartbeat_at`，Stage 8 smoke 通过真实 API + PostgreSQL 断言覆盖。
- data sync、backtest、trading、notification outbox 的 release / fail / pause 清锁 SQL 已收敛到 `internal/store/postgres/lease.go` 共享 helper。
- data sync、backtest、trading、notification outbox 的 claim id 查询、claim 状态更新、claim 过期条件和 claim 锁字段写入已收敛到 `internal/store/postgres/lease.go` 共享 helper。
- data sync、backtest、trading、notification outbox 的非 claim 状态更新清锁 SQL 已开始收敛到 `internal/store/postgres/lease.go` 共享 helper。
- data sync、backtest、trading 的 PostgreSQL heartbeat SQL 已收敛到 `internal/store/postgres/lease.go` 共享 helper，notification outbox 因无 `heartbeat_at` 字段会被 helper 拒绝 heartbeat。
- data sync、backtest、trading、notification runner 在父上下文取消时会释放当前 active lease，不再把 shutdown 误记为任务失败；backtest 会从 `running` 复位为 `pending`，避免清锁后无法再次 claim。
- data sync / backtest / trading / notify 容器级 SIGTERM 已由 `scripts/stage8-sigterm-smoke.sh` 通过真实 Docker Compose stop、受控阻塞点和 PostgreSQL 锁字段断言覆盖。
- claim 的领域候选条件、排序和非 claim 状态切换仍分散在各自 store 方法中，还不是完整统一状态机。
- 停止状态机不完整。
- shutdown 判定已收敛为父 context 取消后任何 work error 都走 release，避免外部库返回非标准取消错误时误记失败或遗留锁。

关闭条件：

- 提取统一 lease 包。
- 支持 claim、heartbeat、release、fail、pause。
- worker 运行长任务时持续刷新 `locked_until`。
- heartbeat 失败达到阈值后停止外部副作用。
- 数据同步、回测、交易 worker 都走统一实现。

### P0：实盘密钥不能用 digest 冒充加密

阶段 6 状态：

- 新建 `exchange_accounts` 凭据已从 `secretDigest` 改为 AES-GCM ciphertext，前缀为 `v1:aesgcm:`。
- `ENCRYPTION_KEY` 来源已定义为 32 字节 base64/hex 环境变量；`.env.example` 和 compose 已暴露配置入口。
- 列表/API 响应只返回账号元数据和 `credentialStatus`，不返回完整 API key/secret。
- 历史非 AES-GCM 行标记为 `legacy`，不能用于创建新的 live 任务。

继续加强条件：

- testnet/sandbox live executor 能用加密凭据解密后提交测试订单。
- 订单先落库再提交交易所，且幂等键贯穿 retry。
- 生产密钥来源升级为 KMS/secret manager 或等价方案。
- 轮换 `ENCRYPTION_KEY` 和历史 `legacy` 账号迁移策略明确。

### P0：交易事实不能用 float64

阶段 0 状态：

- `internal/backtest`、`internal/trading`、`internal/data` 已纳入质量脚本检查。
- 回测交易事实已从 `float64` / `ParseFloat` / `FormatFloat` 迁出。
- 策略指标仍可使用浮点计算，但订单、资金、仓位、成交、PnL 不能用浮点事实。

关闭条件：

- 建立 decimal / money / quantity 类型边界。
- 回测撮合和汇总不再使用 `float64` 表示交易事实。
- 质量脚本能检查敏感目录中的 `float64` 使用。

### P1：API server 必须拆分

阶段 0 状态：

- `internal/web/api/server.go` 已按领域拆分。
- auth、system、data、backtest、trading、strategy、validation、static 已拆入独立文件。
- API server 仍需要继续加强 request / response mapping 和错误边界。

关闭条件：

- 按领域拆成多个 handler 文件。
- validation 独立。
- request / response mapping 独立。
- 单文件低于工程约束建议值。

### 阶段 1 Definition of Done：研究核心

目标等级：demo

范围内：

- 建立后端 CandleProvider 或等价查询服务，作为 `/api/candles`、回测 runner、交易 runner 的统一 K 线查询入口。
- `1m` 请求返回 native K 线，并返回数据来源、基础周期、健康状态。
- `5m / 15m / 1h / 4h / 1d` 请求在没有健康同周期 native K 线时，从 `1m` 聚合。
- 查询结果返回 `source: native / aggregated / none`、`baseInterval`、`health: ok / gap / insufficient`、缺口列表。
- 后端缺口检测基于 UTC open_time 和周期长度，不依赖前端或浏览器时区。
- 研究页布局调整为数据同步任务列表在上、K 线图表在下。
- 研究页展示当前数据源、数据来源、基础周期、数据健康和缺口摘要。

范围外：

- worker lease 统一状态机。
- 数据同步长任务运行中的 heartbeat loop。
- 聚合 K 线持久化缓存。
- 回测详情买卖点叠加。
- 策略指标叠加。
- 实盘下单、通知 provider、账号安全加固。

用户可见行为：

- 研究页顶部是同步任务列表，点击“查看图表”后下方图表加载该数据源。
- 图表下方或工具区能看到当前 K 线来自 native 还是 aggregated。
- 数据不足或存在缺口时，研究页明确展示健康状态，不把空图表伪装成正常。
- 创建、同步、实时、删除同步任务的既有交互不回退。

后端验收：

- `/api/candles` 返回带 metadata 的结果对象，不再只返回裸 K 线数组。
- CandleProvider 单元测试覆盖 native、aggregated、gap、insufficient。
- API route 测试覆盖 `/api/candles` metadata。
- 回测 runner 和交易 runner 通过统一 CandleProvider 结果取 K 线。

前端验收：

- 前端 API client 能解析新的 candles result。
- 研究页显示 source、base interval、health、gap count。
- 研究页布局在桌面和移动宽度下均为列表在上、图表在下。
- `pnpm run typecheck`、`pnpm run test`、`pnpm run build` 通过。

数据验收：

- 不引入新的 migration。
- 不改变 `market_candles` 作为原始事实表的语义。
- 聚合结果不持久化为事实数据。

安全验收：

- 不把阶段 1 说成 usable。
- 实盘真实下单、testnet/sandbox 和幂等提交继续保留为后续阶段风险。

测试验收：

- `go test ./...`
- `go vet ./...`
- `scripts/quality-gate.sh`
- `cd web/frontend && pnpm run typecheck`
- `cd web/frontend && pnpm run test`
- `cd web/frontend && pnpm run build`

### 阶段 1 当前验收快照

执行时间：2026-06-27

通过：

- `go test ./...`
- `go vet ./...`
- `scripts/quality-gate.sh`
- `cd web/frontend && pnpm run typecheck`
- `cd web/frontend && pnpm run test`
- `cd web/frontend && pnpm run build`
- `docker compose up --build -d`
- `curl -fsS http://127.0.0.1:8080/readyz`
- 登录后请求 `/api/candles?exchange=binance&symbol=BTCUSDT&interval=5m&limit=3` 返回 `source=aggregated`、`baseInterval=1m`、`health=ok`。
- Playwright headless Chrome 桌面宽度验证：同步任务列表 y=169，图表 y=315，图表高 680，metadata 显示 `K 线来源: 内部聚合 / 数据健康: 正常 / 基础周期: 1m`。
- Playwright headless Chrome 移动宽度验证：同步任务列表 y=251，图表 y=418，图表高 624，metadata 显示同上。

失败：

- 无硬失败。

警告：

- Vite 构建仍提示主 chunk 超过 500 kB，后续需要做路由级 code split。
- 系统截图工具受 macOS 权限限制，最终视觉验证使用 headless Chrome screenshot 和 bounding box 断言完成。

后续风险审计：

- 交易所账号密钥 digest 风险已在阶段 6 切片关闭到 `demo`；历史非 AES-GCM 行标记为 `legacy`。
- live executor 仍禁用，testnet/sandbox、幂等提交、真实交易所提交和生产密钥管理仍未建立。
- worker lease 仍没有运行中 heartbeat loop 和完整停止状态机。

阶段 1 结论：

- 研究核心达到 `demo` 检查点。
- 项目整体仍为 `scaffold`，不能称为 usable、production-safe 或完成。

### 阶段 1 研究页 K 线高度稳定性补充

执行时间：2026-06-27

触发问题：

- 研究页 K 线图表存在高度反馈风险，用户侧观察到图表区域持续拉高并可能拖崩页面。

修复范围：

- `TradingViewChart` 不再使用图表库内部 `autoSize`。
- 图表改为按容器实际 `getBoundingClientRect()` 尺寸显式初始化和 resize。
- resize 观察目标从可能受图表子节点影响的 chart body 收敛到固定 `.chart-panel`。
- 图表高度不再信任 `.research-chart-body` 的实时高度，统一从固定 `.chart-panel` 的 computed/client/bounds 最小有效高度扣除图表槽 top offset，避免被图表库子节点污染后继续写回更大高度。
- 图表 canvas 在 `createChart` 和 `resize` 前写入显式像素宽高，避免 lightweight-charts 内部 DOM 反向改变宿主测量结果。
- resize 通过 `requestAnimationFrame` 合并，并在尺寸未变化时跳过。
- 组件卸载时断开 `ResizeObserver`、窗口 resize 事件和待执行 animation frame。
- 研究页图表区域新增固定 flex body，工具栏之外的剩余空间才是 K 线图表高度来源。
- `.trading-chart` 明确使用 `height: 100%`，不再以 `auto` 高度参与布局反馈。

验证：

- `cd web/frontend && pnpm run typecheck`
- `cd web/frontend && pnpm run test -- TradingViewChart`
- `cd web/frontend && pnpm run build`
- `cd web/frontend && pnpm run test`
- `go test ./...`
- `go vet ./...`
- `git diff --check`
- `scripts/quality-gate.sh`
- `docker compose up -d --build api`
- `curl -fsS http://127.0.0.1:8080/readyz`
- Headless Chrome 桌面 `2048x1024` 打开 `/research`，30 次采样 `scrollHeight=2026`、`panelHeight=760`、`bodyHeight=683`、`chartHeight=683`、`canvasHeight=681`、`tvHeight=680`，无增长。
- Headless Chrome 桌面 `2048x1024` 打开 `/research?exchange=binance&symbol=BTCUSDT&interval=5m`，30 次采样 `scrollHeight=2026`、`panelHeight=760`、`bodyHeight=683`、`chartHeight=683`、`canvasHeight=681`、`tvHeight=680`，无增长。
- Headless Chrome 移动 `390x844` 打开 `/research`，30 次采样 `scrollHeight=1984`、`panelHeight=624`、`bodyHeight=457`、`chartHeight=457`、`canvasHeight=455`、`tvHeight=454`，无增长。
- 浏览器采样未捕获 `ResizeObserver`、JS exception 或 console error。
- 本轮追加回归验证：`pnpm run test` 覆盖 26 个前端测试，其中 `TradingViewChart` 新增 polluted host / inflated panel 两个高度污染场景。
- 本轮追加本地 8080 验证：`docker compose up -d --build api` 后，Headless Chrome `2048x1024` 登录并打开 `/research`，30 次采样 first/last 均为 `scrollHeight=2354`、`panel=717`、`body=640`、`chart=640`、`canvas=638`、`tv=638`，`uniqueCount=1`，无 runtime/log error。

失败：

- 无硬失败。

警告：

- Vite 构建仍提示主 chunk 超过 500 kB，后续需要做路由级 code split。

### 阶段 1 研究页 K 线高度稳定性二次加固

执行时间：2026-06-27

触发问题：

- 用户侧继续观察到 K 线图表存在持续拉高页面并最终拖崩浏览器的风险。

修复范围：

- `TradingViewChart` 尺寸读取优先使用固定宿主的 `clientWidth/clientHeight`，不再把可能被图表内部 DOM 污染的 `getBoundingClientRect().height` 当作主要高度来源。
- 图表高度继续由最近的 `.chart-panel` 做边界，但边界读取改为优先使用 `clientHeight`，再回退 computed height 和 bounds height。
- 根节点 `.trading-chart` 和 `.trading-chart__canvas` 同步写入同一组显式像素尺寸，避免根节点和 canvas 宿主之间出现尺寸漂移。
- 绝对定位从 `inset: 0` 收敛为 `top/left + explicit width/height`，减少右/下约束与显式尺寸并存时的布局解算不确定性。
- 新增单测覆盖宿主 bounds 被污染到 `3200px` 但 client 高度仍固定的场景。

验证：

- `cd web/frontend && pnpm run test -- src/components/chart/TradingViewChart.test.ts`
- `cd web/frontend && pnpm run typecheck`
- `cd web/frontend && pnpm run build`
- `docker compose up -d --build api`
- `curl -fsS http://127.0.0.1:8080/readyz`
- Headless Chrome 桌面 `2048x1034` 登录并打开 `/research`，45 次采样 first/last 均为 `scrollHeight=1318`、`panel=760`、`body=683`、`chart=682`、`canvasHost=682`、`tv=682`，`uniqueCount=1`，无 console error。
- Headless Chrome 桌面 `2048x1034` 登录并打开 `/research?exchange=binance&symbol=BTCUSDT&interval=5m`，45 次采样 first/last 均为 `scrollHeight=1318`、`panel=760`、`body=683`、`chart=682`、`canvasHost=682`、`tv=682`，`uniqueCount=1`，无 console error。
- Headless Chrome 移动 `390x844` 登录并打开 `/research`，45 次采样 first/last 均为 `scrollHeight=1256`、`panel=624`、`body=457`、`chart=456`、`canvasHost=456`、`tv=456`，`uniqueCount=1`，无 console error。
- `go test ./...`
- `go vet ./...`
- `scripts/quality-gate.sh`
- `git diff --check`

失败：

- 无硬失败。

警告：

- Vite 构建仍提示主 chunk 超过 500 kB，后续需要做路由级 code split。

### 阶段 1 研究页 K 线高度稳定性三次加固

执行时间：2026-06-27

触发问题：

- 用户侧继续反馈 K 线图表界面会无限拉高，直到页面崩掉。
- 前两轮 JS 尺寸 clamp 已覆盖常见反馈路径，但 CSS 仍需要彻底阻断 lightweight-charts 内部 DOM 参与父容器固有尺寸计算。

修复范围：

- `.trading-chart` 和 `.trading-chart__canvas` 改为明确 `width: 100%`、`height: 100%`、`max-width/max-height: 100%`，不再依赖 `auto` 尺寸解算。
- 图表根、canvas 宿主和 `.tv-lightweight-charts` 内部容器统一使用 `contain: strict`，防止内部 canvas / pane DOM 把外层容器撑高。
- `.tv-lightweight-charts` 改为 absolute + `inset: 0`，固定在 canvas 宿主内，避免内部普通流高度反向影响宿主。
- 新增单测覆盖无 `.chart-panel` 边界时宿主高度被图表内部污染到 `5000px/8000px` 的场景，验证最多 resize 到 viewport，且不会继续追涨。

验证：

- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `go test ./...`
- `go vet ./...`
- `scripts/quality-gate.sh`
- `git diff --check`
- `docker compose up -d --build api`
- `curl -fsS http://127.0.0.1:8080/readyz`
- Headless Chrome `1365x768` 登录并打开 `/research`，24 次、12 秒采样 first/last 均为 `body=1118`、`panel=560`、`chartBody=483`、`chart=483`、`tv=483`，`grew=false`，`contain=strict`。

失败：

- 无硬失败。

警告：

- Vite 构建仍提示主 chunk 超过 500 kB，后续需要做路由级 code split。

### 阶段 1 研究页 K 线高度稳定性四次加固

执行时间：2026-06-28

触发问题：

- 用户侧继续反馈 K 线图表界面存在无限拉高直到页面崩掉的风险。
- 本地 headless Chrome 未复现持续增长，但当前代码仍观察 `.research-chart-body`，并且 root/canvas 尺寸完全依赖 CSS `100%` 解算，真实浏览器环境下仍有 resize feedback 入口。

修复范围：

- `TradingViewChart` 初始化和 resize 前同步写入 `.trading-chart` 与 `.trading-chart__canvas` 的显式像素宽高。
- `ResizeObserver` 在存在 `.chart-panel` 时改为观察固定面板，不再观察可能受图表内部 DOM 影响的 `.research-chart-body`。
- `.trading-chart` 和 `.trading-chart__canvas` 绝对定位从 `inset: 0` 收敛为 `top/left + 显式宽高`，减少四边约束与显式尺寸并存时的布局反馈风险。
- `.research-chart-body` 升级为 `contain: strict`，阻断 lightweight-charts 内部 table/canvas 对父级固有高度的贡献。
- 单测改为验证 root/canvas 必须固定到稳定宿主像素尺寸，并新增固定 `.chart-panel` 观察目标断言。

验证：

- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `git diff --check`
- `scripts/quality-gate.sh`
- `docker compose up -d --build api`
- `curl -fsS http://127.0.0.1:8080/readyz`
- Headless Chrome `2048x1034` 登录并打开 `/research?exchange=binance&symbol=BTCUSDT&interval=1m`，180 次、约 45 秒采样 first/last 均为 `body=1285`、`panel=727`、`bodyEl=650`、`chart=649`、`canvas=649`、`tv=649`、`maxInner=649`，`unique=1`、`grew=false`，且 `chart.inlineH=649px`、`canvas.inlineH=649px`。

失败：

- 无硬失败。

未执行：

- 本轮只改前端图表和审计文档，未执行 `go test ./...`、`go vet ./...`、`scripts/stage8-smoke.sh`。

### 阶段 1 研究页 K 线高度稳定性五次加固

执行时间：2026-06-28

触发问题：

- 用户侧继续反馈 K 线图表界面会无限拉高直到页面崩掉。
- 本地 `8080/research` headless Chrome 未复现持续增长，但真实浏览器中 `ResizeObserver` 高度回调仍可能被图表内部 DOM 撑大后写回图表。

修复范围：

- `TradingViewChart` 在存在 `.chart-panel` 边界时，面板高度读取改为优先信任固定 CSS computed height，再回退 client / observed / bounds。
- 新增单测模拟 `.chart-panel` CSS 高度为 `760px`，但 `ResizeObserver` 回报 `5200px` 的异常场景，验证图表不会 resize 到异常高度。
- 重新构建本地镜像并替换 API 容器，使 `http://127.0.0.1:8080/research` 服务本轮修复后的前端产物。

验证：

- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `go test ./...`
- `go vet ./...`
- `scripts/quality-gate.sh`
- `git diff --check`
- `docker compose build api`
- `docker compose up -d --no-deps api`
- `docker compose ps api`
- Headless Chrome `1440x900` 登录并打开 `/research`，30 次采样 first/last 均为 `documentHeight=1238`、`panel=680`、`chartBody=603`、`chart=603`、`canvasHost=603`、`tv=603`，无增长。

失败：

- 无硬失败。

未执行：

- 本轮未执行 `scripts/stage8-smoke.sh`、`scripts/stage8-sigterm-smoke.sh`，因为修复范围限定为前端图表高度反馈和本地 API 静态资源更新。

### 阶段 1 研究页 K 线高度稳定性六次加固

执行时间：2026-06-28

触发问题：

- 用户侧继续反馈 K 线图表界面会无限拉高直到页面崩掉。
- 前几轮修复仍存在尺寸反馈路径：组件既读取外部 DOM 高度，又把计算后的像素高度写回 `.trading-chart` / `.trading-chart__canvas`，真实浏览器中仍可能把 chart 内部高度纳入下一轮 resize 输入。

修复范围：

- `TradingViewChart` 不再把测得的像素宽高反写到 `.trading-chart` 和 `.trading-chart__canvas` inline style，root/canvas 尺寸只由固定 CSS viewport 承载。
- ResizeObserver 观察目标改为实际 chart viewport（例如 `.research-chart-body`），不再观察上一层 `.chart-panel` 后再推导图表高度。
- 图表尺寸读取优先使用 viewport `clientWidth/clientHeight`，再回退 ResizeObserver content box、computed height、bounds，避免 polluted bounds 成为主输入。
- 最近 `.chart-panel` 只作为可用高度硬上限；即使 viewport 高度被污染到数千像素，也只会被截断到面板可用高度，不会持续追涨。
- 单测契约改为验证 chart 组件不写 root/canvas inline 高度、优先使用 viewport client size、污染高度被 panel cap 截断。

验证：

- `pnpm --dir web/frontend run test -- src/components/chart/TradingViewChart.test.ts`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `go test ./...`
- `go vet ./...`
- `scripts/quality-gate.sh`
- `git diff --check`
- `docker compose up -d --build api`
- `curl -fsS http://127.0.0.1:8080/readyz`
- Headless Chrome 桌面 `2048x1034` 登录并打开 `/research`，80 次采样 first/last 均为 `scrollHeight=1318`、`panel=760`、`body=683`、`chart=683`、`canvasHost=683`、`tv=683`；root/canvas inline 高度为空，无高度增长。
- Headless Chrome 移动 `390x844` 登录并打开 `/research`，80 次采样 first/last 均为 `scrollHeight=1256`、`panel=624`、`body=457`、`chart=457`、`canvasHost=457`、`tv=457`；root/canvas inline 高度为空，无高度增长。

失败：

- 旧图表单测首次运行失败 8 项，原因是测试仍断言上一轮实现细节（观察 panel、写 root/canvas inline 像素）。已按新契约改写并重新通过。

未执行：

- 本轮未执行 `scripts/stage8-smoke.sh`、`scripts/stage8-sigterm-smoke.sh`；本次修复范围限定为前端图表高度反馈和本地 API 静态资源更新。
- 浏览器采样仍出现登录前 `/api/auth/me` 的预期 `401` 和 Chrome 对 password autocomplete 的提示；未捕获 JS runtime error。

### 阶段 1 研究页 K 线高度稳定性七次加固

执行时间：2026-06-28

触发问题：

- 用户侧继续反馈 K 线图表界面会无限拉高，直到页面崩掉。
- 本地 `8080/research` 当前构建 headless Chrome 未复现持续增长，但旧实现仍允许在没有可信 `clientHeight` 时，把 `ResizeObserver` height 或 bounds height 作为图表高度输入。

修复范围：

- `TradingViewChart` 高度读取收敛为：优先使用 chart viewport `clientHeight`；如果存在 `.chart-panel`，只使用固定面板可用高度作为上限和兜底；最后才退回固定 fallback。
- `ResizeObserver` 只保留宽度辅助输入，observer height 不再参与图表高度计算。
- `.chart-panel` 可用高度读取优先使用 CSS computed height，再回退 client / bounds，避免 observer height 污染面板高度。
- 研究页图表面板从 grid 百分比行改为 flex 列布局，`.research-chart-body` 只占工具栏之外的剩余空间，不再用 `height: 100%` 参与自身高度解算。
- 图表单测更新为新契约：没有可信高度边界时不追随图表驱动的宿主增高；有固定面板时忽略膨胀 observer height。

验证：

- `pnpm --dir web/frontend run test -- src/components/chart/TradingViewChart.test.ts`
- 本轮通用门禁见最终回复。
- Headless Chrome `1440x900` 登录并打开 `/research`，80 次采样 first/last 均为 `documentHeight=1238`、`panel=680`、`chartBody=603`、`chart=603`、`canvasHost=603`、`tv=603`，`uniqueCount=1`，无增长。
- Headless Chrome mobile `390x844` 登录并打开 `/research`，80 次采样 first/last 均为 `documentHeight=1256`、`panel=624`、`chartBody=457`、`chart=457`、`canvasHost=457`、`tv=457`，`uniqueCount=1`，无增长。

失败：

- 旧图表单测首次运行失败 4 项，原因是测试仍断言旧策略允许无可信边界时追随 bounds/viewport 高度；已按新高度输入契约改写并重新通过。

剩余风险：

- 本地 headless Chrome 仍未复现用户侧持续增长；本轮通过删除 observer/bounds height 输入来关闭主要反馈入口，但仍需用户在真实可见浏览器中确认。

### 阶段 1 研究页 K 线高度稳定性八次加固

执行时间：2026-06-28

触发问题：

- 用户侧继续反馈前端 K 线图表界面会无限拉高，直到页面崩掉。
- 本地 headless Chrome 在当前构建仍未复现持续增长，但旧高度读取契约仍允许 `.chart-panel` 直接宿主场景信任被污染的 `clientHeight`，无固定面板场景也仍可能用宿主高度作为图表高度输入。

修复范围：

- `TradingViewChart` 高度读取改为：存在 `.chart-panel` 时只以面板可用高度作为硬上限，chart viewport 的 `clientHeight` 只允许缩小最终高度，不能放大最终高度。
- `.chart-panel` 高度读取使用 client / computed / bounds 的最小正值，避免任一测量来源被图表内部 DOM 污染后把高度放大。
- 图表直接挂在 `.chart-panel` 时也走固定面板边界，不再因为 `panel === host` 回退到宿主 `clientHeight`。
- 无 `.chart-panel` 的非生产挂载场景只使用固定 fallback 高度，不再从 host bounds 追随增长。
- 研究页图表面板从 flex 列布局收敛为固定两行 grid：工具栏 `auto`，图表槽 `minmax(0, 1fr)`，图表 body 明确占满固定槽位。
- 图表根节点和 canvas 宿主继续不写 inline 宽高，由固定 CSS viewport 承载尺寸，避免反向参与父级布局。

验证：

- `pnpm --dir web/frontend run test -- src/components/chart/TradingViewChart.test.ts`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `docker compose build api`
- `docker compose up -d --no-deps api`
- `curl -fsS http://127.0.0.1:8080/readyz`
- Headless Chrome 桌面 `1440x900` 登录并打开 `/research`，80 次采样 first/last 均为 `documentHeight=1238`、`panel=680`、`chartBody=603`、`chart=603`、`tv=603`，`uniqueDocs=[1238]`、`uniqueChart=[603]`、`grew=false`，无 runtime/log error。
- Headless Chrome 移动 `390x844` 登录并打开 `/research`，80 次采样 first/last 均为 `documentHeight=1256`、`panel=624`、`chartBody=457`、`chart=457`、`tv=457`，`uniqueDocs=[1256]`、`uniqueChart=[457]`、`grew=false`，无 runtime/log error。
- `go test ./...`
- `go vet ./...`
- `git diff --check`
- `scripts/quality-gate.sh`

失败：

- 无硬失败。

剩余风险：

- 本轮仍未在用户的可见 Chrome 会话里复现原始无限增长，只能通过源码约束、自动化污染输入测试和本地 headless Chrome 采样关闭已知反馈入口。

### 阶段 1 Candle 查询 limit 边界补充

执行时间：2026-06-27

触发问题：

- `/api/candles` 接受任意 `limit`，而 PostgreSQL 查询层对 `limit > 5000` 会静默降回默认 `1000`。
- 这会让大范围查询看似被接受，实际只返回默认窗口，属于阶段 1 查询边界不清。

修复范围：

- 新增 `data.DefaultCandleLimit=1000` 和 `data.MaxCandleLimit=5000`。
- 新增 `data.NormalizeCandleLimit`，供存储层直接调用时统一规范化。
- `/api/candles` 对超过 `MaxCandleLimit` 的请求返回 `400`，不再把不可控大范围请求传入 store。
- PostgreSQL `ListNativeCandles` 对直接调用的超大 limit clamp 到 `MaxCandleLimit`，不再静默降回默认 1000。
- CandleProvider 聚合基础 K 线窗口使用同一组 limit 常量，移除硬编码。

验证：

- `go test ./internal/data ./internal/web/api ./internal/store/postgres`
- `TestNormalizeCandleLimit` 覆盖默认、负数、正常值、最大值和超大值。
- `TestCandlesRouteRejectsOversizedLimit` 覆盖 API 超大 limit 返回 `400`。
- `TestIntegrationListNativeCandlesClampsOversizedLimit` 覆盖直接 store 查询超大 limit 时不再降回默认 1000。

失败：

- 无硬失败。

后续风险：

- 这不是完整 cursor pagination；阶段 1 仍需要明确大范围时间查询和翻页协议。

### 阶段 1 Candle 查询时间范围边界补充

执行时间：2026-06-27

触发问题：

- `/api/candles` 之前只解析 `from/to`，没有拒绝倒置区间，也没有限制可表达的时间跨度。
- 这会让一个已受 `limit` 限制的请求仍能表达超大历史区间，查询语义和性能边界不清。

修复范围：

- `internal/data` 新增 `ValidateCandleQueryRange`，作为 Candle 查询输入边界。
- API parse 阶段校验 interval、`from <= to`。
- 当 `from/to` 同时存在时，按闭区间语义限制最大跨度为 `(MaxCandleLimit - 1) * IntervalDuration(interval)`；超过边界返回 `400`。
- 单端 `from` 或 `to` 查询仍允许，由 SQL `LIMIT` 约束返回窗口。

验证：

- `go test ./internal/data ./internal/web/api`
- `TestValidateCandleQueryRange` 覆盖无边界、单端边界、相同边界、最大合法跨度、倒置、超大跨度和无效 interval。
- `TestCandlesRouteRejectsInvertedRange` 覆盖倒置区间 API 返回 `400`。
- `TestCandlesRouteRejectsOversizedRange` 覆盖超大区间 API 返回 `400`。
- `TestCandlesRouteRejectsUnsupportedInterval` 覆盖不支持 interval API 返回 `400`。

失败：

- 无硬失败。

后续风险：

- 这仍不是完整 cursor pagination；大范围历史查询需要后续明确游标协议和性能压测。
- 高周期从 `1m` 聚合时仍受基础 K 线窗口限制，聚合缓存或分页读取仍未完成。

### 阶段 1 Candle 聚合窗口覆盖补充

执行时间：2026-06-27

触发问题：

- 高周期 K 线从 `1m` fallback 聚合时，请求 `limit=1000` 的 `1h` 实际需要 60000 根 `1m` 基础 K 线。
- 现有基础查询上限为 `MaxCandleLimit=5000`，Provider 会只聚合有限基础窗口，但响应里没有说明窗口被截断，研究页可能把少量聚合 K 线误看成完整健康窗口。

修复范围：

- `CandleResult` 新增 `coverage` 元数据，包含 requested / returned 数量、required / actual base window 和 `limitedByBaseWindow`。
- CandleProvider 在聚合 fallback 时记录基础窗口需求和实际返回基础 K 数。
- 当基础窗口被 `MaxCandleLimit` 截断且返回聚合数量不足请求 limit 时，结果 `health=insufficient`。
- 研究页元信息显示窗口受限标签，前端 API wrapper 保留 coverage。

验证：

- `go test ./internal/data`
- `pnpm run test -- data`
- `TestCandleProviderReportsLimitedAggregationCoverage` 覆盖 `1h limit=1000` 需要 60000 根基础 K 线但只能读取 5000 根时的 coverage 和 health。
- `data api` metadata 测试覆盖前端 wrapper 不丢失 coverage。
- 本地 API smoke：`/api/candles?exchange=binance&symbol=BTCUSDT&interval=1h&limit=1000` 返回 `health=insufficient`、`coverage.requiredBaseCandles=60000`、`coverage.baseLimit=5000`、`coverage.limitedByBaseWindow=true`。

失败：

- 无硬失败。

后续风险：

- 这仍不是 cursor pagination；它只让受限窗口可观察，尚未解决长区间完整读取。
- 回测和交易 runner 已拒绝 `gap/insufficient/limitedByBaseWindow` 数据，但长区间完整读取仍未解决。

### 阶段 1/3/4 策略输入数据健康门禁补充

执行时间：2026-06-27

触发问题：

- 回测和交易 runner 之前只把 `candleHealth` 写入摘要或忽略 metadata，仍会把 `gap`、`insufficient` 或 `limitedByBaseWindow` 的 K 线送入策略。
- 这会让策略在缺口、不足或基础聚合窗口受限的数据上产生 intent / order / notification，结果看起来像真实信号但数据前提不成立。

修复范围：

- 新增 `data.ValidateStrategyCandleResult` 作为策略输入前共享门禁。
- 回测 runner 在 `ClosedCandles` 和 `strategy.GenerateIntents` 前校验 CandleProvider 结果；不健康数据会 mark failed，不保存 backtest result / intent / order。
- 交易 runner 在 `ClosedCandles` 和 `strategy.GenerateIntents` 前校验 CandleProvider 结果；不健康数据会 mark failed，不保存 trading result / order / execution / notification。
- CandleProvider `limitedByBaseWindow` 只在理论基础窗口超限且实际基础查询打满 `BaseLimit` 时标记，避免短 `from/to` 区间被误判。

验证：

- `go test ./internal/data ./internal/backtest ./internal/trading`
- `TestValidateStrategyCandleResult` 覆盖 healthy、gap、insufficient、limited coverage。
- `TestRunnerRunOnceFailsOnUnhealthyCandles` 覆盖回测遇到 `health=gap` 时 mark failed 且不保存结果。
- `TestRunnerRunOnceFailsOnLimitedCoverage` 覆盖交易遇到 `limitedByBaseWindow=true` 时 mark failed 且不保存结果。
- `docker compose up -d --build backtest trading` 后 backtest / trading worker 容器均能启动。

失败：

- 无硬失败。

后续风险：

- 这只是策略输入前门禁，不提供自动补数、分页读取或重试策略。
- 已失败任务需要用户或后续运维能力介入，自动恢复策略仍未定义。

### 阶段 1/3/4 闭合 K 线信号补充

执行时间：2026-06-27

触发问题：

- 计划要求“未闭合的聚合 K 线可以用于图表展示，但不能被当成闭合周期信号”。
- 回测 runner 和交易 runner 之前直接把 `CandleProvider` 返回的全部 K 线交给策略，未闭合的最后一根 K 线可能触发 order / notification intent。

修复范围：

- 新增 `data.ClosedCandles` 作为 runner 侧共享过滤边界。
- 回测 runner 在 `strategy.GenerateIntents` 前只传入 `IsClosed=true` 的 K 线。
- 交易 runner 在 `strategy.GenerateIntents` 前只传入 `IsClosed=true` 的 K 线。
- 回测结果摘要新增 `inputCandleCount`、`strategyCandleCount`、`droppedOpenCandleCount`，用于审计策略实际使用的 K 线数量。

验证：

- `go test ./internal/data ./internal/backtest ./internal/trading`
- `TestClosedCandlesFiltersOpenCandles` 覆盖共享过滤函数。
- `TestRunnerRunOnceIgnoresUnclosedCandleSignals` 覆盖回测：未闭合最后一根 K 线即使会产生突破信号，也不会生成 intent / order，并记录 dropped count。
- `TestRunnerRunOnceIgnoresUnclosedCandleSignals` 覆盖交易：未闭合最后一根 K 线不会生成 intent / order / execution。

失败：

- 无硬失败。

后续风险：

- 这只是闭合信号基础防护，不提升回测撮合可信度，不代表交易风控或实盘安全边界完成。

### 阶段 1 数据同步临时错误收敛补充

执行时间：2026-06-27

触发问题：

- 研究页同步任务曾展示 Binance K 线请求 `EOF` 类错误，错误摘要可读性不足，临时网络失败会直接进入失败展示。

修复范围：

- Binance / OKX K 线 adapter 将 transport、超时、HTTP 429、HTTP 5xx 归类为临时错误。
- Binance 多 endpoint fallback 保留，所有 endpoint 都是临时失败时返回脱敏的 temporary unavailable 摘要，不包含完整 query URL。
- 数据同步 runner 对临时 market data 错误做短重试，默认 2 次，延迟默认 250ms，并支持 `SYNC_FETCH_RETRIES` / `SYNC_RETRY_DELAY` 配置。
- 临时错误重试后仍失败时记录 `last_error`，但对仍启用的 sync / realtime 任务回到 `pending` 等待下一轮领取，不把临时错误长期固定为 `failed`。
- 本轮追加收敛：临时 market data 错误不再走 `MarkDataSyncFailed`，runner 改走 `RecordDataSyncRetry`，释放当前 lease 并保留 sync / realtime 期望；realtime 任务保持 `running`，一次性同步任务回到 `pending`。
- 本轮追加收敛：永久数据同步失败会进入 `failed` 并关闭 `sync_enabled` / `realtime_enabled`，同时 claim 只选择 `pending` / `running` 任务，避免旧的 failed+enabled 行被 worker 反复领取。
- 数据同步失败错误文本在落库前做空白规范化和 500 rune 截断。
- 研究页最近错误列强化单行截断、title 和 tooltip 换行，避免长错误撑爆表格。

验证：

- `go test ./...`
- `go vet ./...`
- `cd web/frontend && pnpm run typecheck`
- `cd web/frontend && pnpm run test`
- `cd web/frontend && pnpm run build`
- `git diff --check`
- `scripts/quality-gate.sh`
- `docker compose up -d --build api sync`
- `curl -fsS http://127.0.0.1:8080/readyz`
- 本地 API `/api/data/tasks` 返回同步任务 `status=running`、`latestSyncedAt=2026-06-27T05:19:00Z`、无 `lastError`。
- PostgreSQL 查询显示任务 `status=running`、`last_synced_open_time=2026-06-27 05:19:00+00`、`last_error=''`。
- Headless Chrome 打开 `/research`，同步任务表显示 `最新同步时间=2026-06-27T05:20:00Z`、`实时=运行中`、`同步=运行中`、`最近错误=-`，无 console / page error。
- 本轮追加目标验证：`go test ./internal/datasync ./internal/store/postgres ./internal/web/api` 覆盖临时错误耗尽重试后记录 retry 且不标 failed，永久错误仍走 failed。
- 本轮追加前端验证：`cd web/frontend && pnpm run typecheck` 确认 `DataSyncTask.attemptCount` 类型可被前端接收。

失败：

- 无硬失败。

警告：

- 数据同步 worker 仍没有统一 lease 包和运行中 heartbeat loop，本补充不把数据同步升级为 usable。
- Vite 构建仍提示主 chunk 超过 500 kB，后续需要做路由级 code split。

### 阶段 1 PostgreSQL 集成证据补充

执行时间：2026-06-27

新增验收入口：

- `TICTICK_TEST_DATABASE_URL=... go test ./internal/store/postgres -run Integration -count=1 -v`

覆盖：

- `TestIntegrationCandleProviderAggregatesAndReportsGaps` 使用真实 PostgreSQL `market_candles` 表，验证 `Store.GetCandles` 经 CandleProvider 从 `1m` 聚合出 `5m`，返回 `source=aggregated`、`baseInterval=1m`，并报告底层 `1m` 缺口。
- 同一测试验证 `1m` native 查询从 PostgreSQL 行扫描后返回 `source=native`、`health=gap` 和缺口数量。
- `TestIntegrationListNativeCandlesUsesLatestWindowWithoutRange` 验证无 `from/to` 的研究页默认查询返回最新 N 根 K 线并按时间升序输出；带 `from` 的查询仍从区间起点升序返回。
- 同一测试验证 `5m` 聚合在无时间范围时使用最新 `1m` 窗口，避免研究页默认图表展示数据库最早一段历史。
- `TestIntegrationDataSyncRetryReleasesAndReclaimsTask` 使用真实 `data_sync_tasks` 表，验证 `RecordDataSyncRetry` 会清理 `locked_by` / `locked_until` / `heartbeat_at`、保留 sync / realtime 期望，并能被 `ClaimDataSyncTask` 再次领取。
- `TestIntegrationDataSyncPermanentFailureStopsTask` 验证 `MarkDataSyncFailed` 会进入 `failed`、关闭 `sync_enabled` / `realtime_enabled`、清理 lease，并不再满足 claim 条件。

本轮本地执行：

- `docker run --rm --network tictick-hi_default -v "$PWD":/src -w /src -e TICTICK_TEST_DATABASE_URL='postgresql://tictick:...@postgres:5432/tictick_hi?sslmode=disable' golang:1.26-bookworm go test ./internal/store/postgres -run Integration -count=1 -v`

结果：

- 4 个 PostgreSQL 集成测试全部通过。
- 普通 `go test ./internal/store/postgres` 在未设置 `TICTICK_TEST_DATABASE_URL` 时通过，集成测试默认跳过，避免误连非测试数据库。

剩余风险：

- 集成测试覆盖基础聚合、缺口、默认最新窗口和同步 retry 状态机，不代表大范围 K 线查询性能已达 usable。
- data sync / backtest / trading 容器级 SIGTERM 后数据库断言已由 Stage 8 专用 smoke 覆盖；仍缺真实交易所网络稳定性证明。

### 阶段 2 Definition of Done：策略沉淀

目标等级：demo

范围内：

- 后端策略 registry 明确列出策略 ID、名称、版本、描述、支持周期、支持 intent 类型和参数 schema。
- 参数 schema 支持 number / select / boolean / text，并包含 required、default、min、max、step、options、description。
- 创建回测和交易任务时，后端按策略 schema 校验参数：必填、未知参数、类型、数值范围、select options 都必须被检查。
- 策略 runtime 只接收 candles 和参数快照，只返回结构化 intent，不允许下单、发通知或写库。
- 策略 intent 至少支持 `order` 和 `notification` 两类，并用 payload 表达 side、price、quantity、symbol、occurredAt、message 等结构化字段。
- 交易 runner 只负责把策略 intent 落成任务观察记录；不把策略函数变成执行器。
- 前端能从 `/api/strategies` 获取 schema，选择策略并按 schema 填写参数。
- 前端提交回测 / 交易任务时保存参数快照，且 UI 能展示策略描述、支持 intent 和参数摘要。
- 质量门禁加入策略边界检查，阻止 `internal/strategy` 反向依赖 store/web/trading/backtest 或网络发送能力。

范围外：

- 回测 worker 的撮合可信度升级。
- 回测详情买卖点叠加。
- PaperExecutor、持仓、成交和订单簿。
- Notification outbox / provider / retry。
- live executor、实盘密钥、实盘确认护栏。

用户可见行为：

- 新建回测和新建交易页面能加载策略列表。
- 选择不同策略时，参数表单根据后端 schema 切换。
- 策略摘要显示策略描述、支持 intent 和当前参数快照。
- 不符合 schema 的参数不能绕过前端 / 后端进入任务。

后端验收：

- `internal/strategy` 有 registry/schema 单元测试。
- `internal/strategy` 有 order intent 和 notification intent 单元测试。
- API route 测试覆盖无效策略参数会被拒绝。
- `scripts/quality-gate.sh` 执行策略边界检查。

前端验收：

- strategies API client 测试覆盖参数 schema 正规化。
- `useStrategyTaskForm` 测试覆盖 default param values 和参数范围校验。
- `pnpm run typecheck`、`pnpm run test`、`pnpm run build` 通过。

数据验收：

- 不引入新的 migration。
- 创建任务仍保存 strategyId 和 strategyParams 快照。
- 策略 registry 不写数据库。

安全验收：

- 策略代码不能直接下单、通知或写库。
- 阶段 2 不升级实盘安全等级，不声明 usable。

测试验收：

- `go test ./...`
- `go vet ./...`
- `scripts/quality-gate.sh`
- `cd web/frontend && pnpm run typecheck`
- `cd web/frontend && pnpm run test`
- `cd web/frontend && pnpm run build`

### 阶段 2 当前验收快照

执行时间：2026-06-27

通过：

- `git diff --check`
- `go test ./...`
- `go vet ./...`
- `scripts/quality-gate.sh`
- `cd web/frontend && pnpm run typecheck`
- `cd web/frontend && pnpm run test`
- `cd web/frontend && pnpm run build`
- `docker compose up --build -d`
- `curl -fsS http://127.0.0.1:8080/readyz`
- 登录后 `GET /api/strategies` 返回 `order / notification` intent 和 `signalMode` 参数 schema。
- 登录后 `POST /api/backtests` 只传 `fastPeriod` 时，后端保存补全后的 `strategyParams` 快照。
- 登录后 `POST /api/backtests` 传 `fastPeriod=1` 时返回 `400`。
- Headless Chrome 打开 `/backtests/new` 和 `/trading/new`，能看到策略 schema 参数表单、intent 标签和 `Signal Mode` 下拉选项。

失败：

- 无硬失败。

警告：

- Vite 构建仍提示主 chunk 超过 500 kB，后续需要做路由级 code split。
- 策略仅是 registry/runtime demo 边界，没有沙箱、版本迁移、权限隔离或真实策略库。
- 这次本地 API smoke 创建了一条 `Stage2 EMA defaults` 回测任务，用于确认后端参数快照规范化。

后续风险审计：

- 交易所账号密钥 digest 风险已在阶段 6 切片关闭到 `demo`；历史非 AES-GCM 行标记为 `legacy`。
- live executor 仍禁用，testnet/sandbox、幂等提交、真实交易所提交和生产密钥管理仍未建立。
- worker lease 仍没有运行中 heartbeat loop 和完整停止状态机。
- 通知仍没有 outbox / provider / retry。

阶段 2 结论：

- 策略沉淀达到 `demo` 检查点。
- 项目整体仍为 `scaffold`，不能称为 usable、production-safe 或完成。

### 阶段 3 Definition of Done：回测

目标等级：demo

范围内：

- 回测任务从前端创建，经 API 保存到 PostgreSQL，并由 `hi backtest` worker 领取执行。
- 回测 worker 必须通过 CandleProvider 读取 K 线；`closed_candle` 使用任务周期，`minute_replay` 使用 `1m` 作为推进周期，并在结果摘要中记录执行周期和 K 线来源。
- 回测运行后必须保存策略 intent、订单和结果摘要，`order` intent 进入撮合，`notification` intent 只记录为 intent。
- 回测详情页必须读取任务、K 线、intent、订单，并在图表上展示买卖点标记。
- 回测订单、资金、仓位、PnL 继续禁止使用 `float64` 作为交易事实。
- 回测 worker 在任务运行期间至少具备 heartbeat 刷新能力，避免长任务只 claim 不续租。

范围外：

- 可信撮合引擎、成交簿、部分成交、滑点曲线和复杂手续费模型。
- 回测指标体系、风险指标、收益曲线和策略指标叠加。
- 全系统统一 worker lease 状态机关闭。
- PaperExecutor、LiveExecutor、通知 provider/outbox。
- 实盘密钥和实盘下单安全边界。

用户可见行为：

- 用户能创建回测任务，并在 worker 运行后看到任务状态、结果摘要、策略 intent、订单列表。
- 回测详情图表使用任务周期 K 线，并展示 buy / sell 标记。
- `minute_replay` 回测结果能明确显示执行周期来自 `1m`。

后端验收：

- API 提供 `/api/backtests/:id/intents`。
- BacktestRepository 保存并读取 strategy intents。
- Backtest worker 单元测试覆盖 order intent、notification intent、minute replay 执行周期和 heartbeat。
- API route 测试覆盖 backtest intents 路由。

前端验收：

- Backtest API client 支持读取 intents。
- 图表组件支持订单 markers。
- 回测详情页展示 intents，并把订单映射为图表 marker。
- `pnpm run typecheck`、`pnpm run test`、`pnpm run build` 通过。

数据验收：

- `strategy_intents` 继续作为回测和交易共用 intent 表，通过 `task_type` 区分。
- 不引入假的回测事实表；订单仍写入 `backtest_orders`。
- 结果摘要记录 `candleSource`、`executionInterval`、`triggerMode`、`totalIntents`、`totalOrders`。

安全验收：

- 阶段 3 不声明回测可信或 usable。
- 策略仍不能直接下单、通知或写库。
- 实盘风险继续保留为后续阶段风险。

测试验收：

- `go test ./...`
- `go vet ./...`
- `scripts/quality-gate.sh`
- `cd web/frontend && pnpm run typecheck`
- `cd web/frontend && pnpm run test`
- `cd web/frontend && pnpm run build`

### 阶段 3 当前验收快照

执行时间：2026-06-27

通过：

- `git diff --check`
- `go test ./...`
- `go vet ./...`
- `scripts/quality-gate.sh`
- `cd web/frontend && pnpm run typecheck`
- `cd web/frontend && pnpm run test`
- `cd web/frontend && pnpm run build`
- `docker compose up --build -d`
- `curl -fsS http://127.0.0.1:8080/readyz`

本地 smoke：

- 登录本地 API 后创建 `Stage3 smoke minute replay` 回测，任务 ID：`bt_35934289802b746157d95471`。
- worker 执行后任务状态为 `succeeded`。
- 结果摘要包含 `triggerMode=minute_replay`、`executionInterval=1m`、`requestedInterval=1m`、`baseInterval=1m`、`candleSource=native`、`candleHealth=ok`。
- `GET /api/backtests/:id/intents` 返回 232 条 strategy intent。
- `GET /api/backtests/:id/orders` 返回 232 条 backtest order，订单 `intentId` 指向已落库的 strategy intent。
- 详情图表按任务周期 `5m` 请求 K 线，CandleProvider 从 `1m` 聚合出 201 根 K 线，`source=aggregated`、`health=ok`。

浏览器验收：

- Headless Chrome 打开 `/backtests/bt_35934289802b746157d95471`，能看到策略意图、订单、执行周期和 `1m` 摘要。
- 回测详情图表 6 次采样高度稳定：panel 780px、canvas 750px、`maxPanelDelta=0`、`maxBodyDelta=0`。
- 回测详情长列表已限制为局部滚动：intent list 280px、order list 280px，页面高度稳定为 1624px。
- Headless Chrome 打开 `/research`，同步任务列表在上、K 线图表在下。
- 研究页图表 6 次采样高度稳定：panel 740px、canvas 634px、`maxPanelDelta=0`、`maxBodyDelta=0`。
- 浏览器验收没有 console error 或 page error。
- 截图保留在 `/tmp/tictick-stage3-backtest-detail.png` 和 `/tmp/tictick-stage3-research.png`。

失败：

- 无硬失败。

警告：

- Vite 构建仍提示主 chunk 超过 500 kB，后续需要做路由级 code split。
- 当前回测 smoke 使用本地已有 `binance / BTCUSDT / 1m` 数据，不代表外部交易所稳定性。
- 回测撮合仍是 demo 级顺序撮合，不具备可信回测指标和真实成交语义。

后续风险审计：

- 交易所账号密钥 digest 风险已在阶段 6 切片关闭到 `demo`；历史非 AES-GCM 行标记为 `legacy`。
- live executor 仍禁用，testnet/sandbox、幂等提交、真实交易所提交和生产密钥管理仍未建立。
- 通知仍没有 outbox / provider / retry。

阶段 3 结论：

- 回测链路达到 `demo` 检查点。
- 项目整体仍为 `scaffold`，不能称为 usable、production-safe 或完成。

### 阶段 4 Definition of Done：模拟盘

目标等级：demo

范围内：

- 交易任务 `paper` 类型从前端创建，经 API 保存到 PostgreSQL，并由 `hi trading` worker 领取执行。
- 交易 runner 继续通过 CandleProvider 读取 K 线并生成策略 intent。
- paper executor 和 live executor 在代码边界上明确分离；阶段 4 禁止 live executor 真实下单，也不再用本地 `pending_submission` 冒充 live 执行。
- `order` intent 在 paper executor 中生成 paper order 和 execution；`notification` intent 或 `notify` policy 生成 notification 记录。
- 持仓从 paper executions 可重复计算并写入 position 事实表，worker 重跑不能重复累加。
- 交易详情页必须读取任务、K 线、intent、order、execution、position、notification，并能观察 worker heartbeat / lease 状态。
- 订单、成交、持仓继续禁止使用 `float64` 作为交易事实。
- trading worker 在任务运行期间至少具备 heartbeat 刷新能力，避免只 claim 不续租。

范围外：

- 真实交易所下单。
- 真实成交回报、部分成交、撤单、订单簿和撮合深度。
- 可信 PnL、保证金、杠杆、资金费率、手续费模型。
- Notification provider / outbox / retry。
- 统一 worker lease 包完全抽取。
- 实盘密钥真加密和 testnet / sandbox live executor。

用户可见行为：

- 用户能创建并启动 paper trading task。
- worker 运行后，交易详情页能看到策略 intent、paper order、execution、position 和 notification。
- 交易详情页图表展示 paper buy / sell 标记。
- live execute 在阶段 4 明确被拒绝或失败，不能伪装成已提交交易所。

后端验收：

- API 提供 `/api/trading/tasks/:id/executions` 和 `/api/trading/tasks/:id/positions`。
- TradingRepository 保存并读取 paper executions / positions。
- Trading worker 单元测试覆盖 paper order+execution+position、notification intent、live execute 禁止、heartbeat。
- API route 测试覆盖 trading executions / positions 路由。

前端验收：

- Trading API client 支持读取 executions / positions。
- 交易详情页展示 positions / executions / intents / orders / notifications 和 worker 状态。
- 交易详情图表支持 paper order markers。
- `pnpm run typecheck`、`pnpm run test`、`pnpm run build` 通过。

数据验收：

- 新增 paper execution / position 事实表必须有幂等约束。
- position 必须从 executions 重算或等价可重复机制更新，不能依赖内存累计。
- 不引入 live 真实下单路径。

安全验收：

- 阶段 4 不声明模拟盘 usable。
- live executor 不允许真实下单，不允许把 pending 本地记录说成交易所提交。
- 实盘风险继续保留为后续阶段风险。

测试验收：

- `go test ./...`
- `go vet ./...`
- `scripts/quality-gate.sh`
- `cd web/frontend && pnpm run typecheck`
- `cd web/frontend && pnpm run test`
- `cd web/frontend && pnpm run build`

### 阶段 4 当前验收快照

执行时间：2026-06-27

通过：

- `git diff --check`
- `go test ./...`
- `go vet ./...`
- `scripts/quality-gate.sh`
- `cd web/frontend && pnpm run typecheck`
- `cd web/frontend && pnpm run test`
- `cd web/frontend && pnpm run build`
- `docker compose up --build -d`
- `curl -fsS http://127.0.0.1:8080/readyz`
- PostgreSQL migration 已执行到 `0007_paper_trading_facts.sql`

本地 smoke：

- 登录本地 API 后创建 `Stage4 paper smoke` paper trading task，任务 ID：`tt_4ba4b4e5eb78a7900dbaef64`。
- `POST /api/trading/tasks/:id/start` 后任务进入 `running`。
- trading worker 写入 111 条 strategy intent、111 条 paper order、111 条 execution、1 条 position、0 条 notification。
- 等待一个 worker 周期后，`attemptCount` 从 4 到 5，但 intent / order / execution / position 数量保持 111 / 111 / 111 / 1，证明幂等约束没有重复累加。
- `heartbeatAt` 从 `2026-06-27T04:40:18.298269Z` 更新到 `2026-06-27T04:40:28.305982Z`。
- `POST /api/trading/tasks` 创建 live + execute 返回 `400`，错误为 `live execution is disabled until the live safety stage`。
- smoke 结束后任务已暂停，保留结果供页面检查。

浏览器验收：

- Headless Chrome 打开 `/trading/tt_4ba4b4e5eb78a7900dbaef64`，能看到模拟盘、持仓、成交、最近心跳。
- 交易详情图表 6 次采样高度稳定：panel 780px、canvas 750px、`maxPanelDelta=0`、`maxBodyDelta=0`。
- 浏览器验收没有 console error 或 page error。
- 截图保留在 `/tmp/tictick-stage4-trading-detail.png`。

失败：

- 无硬失败。

警告：

- Vite 构建仍提示主 chunk 超过 500 kB，后续需要做路由级 code split。
- 当前 position 的 `realizedPnl` 仍为占位级 `0`，阶段 4 只保证 position 从 execution 可重复计算，不声明 PnL 可信。
- Notification provider/outbox/retry 已进入阶段 5 demo，真实第三方 provider 基础发送路径已在后续阶段 8 阻断项补充中接入。

后续风险审计：

- 交易所账号密钥 digest 风险已在阶段 6 切片关闭到 `demo`；历史非 AES-GCM 行标记为 `legacy`。
- live executor 仍禁用，testnet/sandbox、幂等提交、真实交易所提交和生产密钥管理仍未建立。
- 统一 worker lease 包仍未抽取，当前只是 trading/backtest 局部 heartbeat。

阶段 4 结论：

- 模拟盘 paper 链路达到 `demo` 检查点。
- 项目整体仍为 `scaffold`，不能称为 usable、production-safe 或完成。

## 4. 阶段 5：通知 demo 链路

目标等级：`demo`，不是 usable。

Definition of Done：

- NotificationIntent 从策略 intent 进入 notification outbox。
- notification provider 抽象明确，阶段 5 启用安全的本地 / webhook-demo / webhook provider，不接入真实敏感凭据。
- 通知发送状态、失败原因、重试次数可追踪。
- 交易详情和系统通知页能观察 notification 状态流。
- 后端、前端和质量门禁检查通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。

范围外：

- 实盘下单。
- tick 数据。
- 交易所账号真加密。
- 真实第三方通知凭据和生产通知通道。
- 回测撮合可信度升级。

### 阶段 5 当前验收快照

执行时间：2026-06-27

通过：

- `go test ./...`
- `go vet ./...`
- `scripts/quality-gate.sh`
- `cd web/frontend && pnpm run typecheck`
- `cd web/frontend && pnpm run test`
- `cd web/frontend && pnpm run build`
- `git diff --check`
- `NotificationIntent` 由 trading runner 保存到 `notifications` 和 `notification_outbox`。
- `hi notify` 已加入单二进制子命令和 Docker Compose service。
- provider 抽象已建立，阶段 5 启用 `local` / `webhook-demo` / `webhook` provider；`webhook` 使用真实 HTTP POST 和请求上下文取消，不持有第三方敏感凭据。
- notify worker 常驻模式会 drain 当前可领取 outbox，避免一次策略输出大量通知时每 10 秒只处理 1 条。
- 系统通知 API 支持 `GET /api/system/notifications` 和 `POST /api/system/notifications/:id/retry`。
- 系统通知页展示通知状态、provider、attempt、nextAttempt、错误和 retry 操作。
- 交易详情页通知 tab 展示通知状态、provider、attempt 和错误 / 发送时间。
- Docker 本地 smoke：`docker compose up -d --build` 成功，`curl -fsS http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`。
- migration smoke：`schema_migrations` 最新包含 `0009_notification_outbox_terminal_state.sql` 和 `0008_notification_outbox.sql`。
- 成功投递 smoke：paper trading notification-only 任务生成 111 条通知；`notifications` 全部 `sent`，`notification_outbox` 全部 `delivered`，attemptCount 为 1。
- 失败重试 smoke：目标为 `fail-target` 的 demo 通道生成 111 条 `retry_scheduled` 通知，错误为 `demo provider rejected target "fail-target"`。
- 手动 retry smoke：`POST /api/system/notifications/:id/retry` 后，该通知重新投递失败，attemptCount 从 1 增至 2，`notifications` 和 `notification_outbox` 同步记录 `retry_scheduled`、错误和下一次重试时间。
- 终态失败 smoke：重试耗尽后失败任务的 111 条通知全部进入 `failed`，`notification_outbox` 保持 `failed`，notify worker 保持 `running`，不再因终态 `next_attempt_at = NULL` 重启。
- 前端 DOM smoke：`/system/notifications` 显示 `sent` / `retry_scheduled` 和 stage5 目标；`/trading/:id` 可切换到通知 tab 并看到 `stage5-smoke` / `sent`。
- 容器 SIGTERM smoke：慢 webhook 投递期间 `docker compose stop -t 10 notify` 后，`notification_outbox` 清空 `locked_by` / `locked_until` 且不记录投递失败。

失败：

- 无当前硬失败。

警告：

- 真实邮件、Telegram、飞书 provider 已在后续阶段 8 阻断项补充中接入基础发送路径；阶段 5 仍只声明 demo。
- 通知通道只有创建和读取，没有更新、删除、启停编辑；凭据采用 env-reference 模型但还不是生产级密钥治理。
- `hi notify` 已有 outbox claim/lock，但仍未抽取全系统统一 worker lease 包。
- 通知 provider 未实现生产级限流、熔断、模板、审计签名或外部回执。
- Vite 构建仍提示主 chunk 超过 500 kB，后续需要做路由级 code split。

后续风险审计：

- 交易所账号密钥 digest 风险已在阶段 6 切片关闭到 `demo`；历史非 AES-GCM 行标记为 `legacy`。
- live executor 仍禁用，testnet/sandbox、幂等提交、真实交易所提交和生产密钥管理仍未建立。
- 登录会话仍缺持久化限流、会话审计、密码策略和生产级设备上下文。

### 阶段 8 通知真实 provider 基础发送路径补充

执行时间：2026-06-28

目标等级：demo 增量。

范围内：

- `hi notify` provider registry 新增 `email`、`telegram`、`feishu`。
- Telegram provider 使用 `telegram://send?chat_id=<chat-id>&token_env=TELEGRAM_BOT_TOKEN`，bot token 只从环境变量读取，请求 `sendMessage` JSON payload。
- 飞书 provider 使用 `feishu://webhook?url_env=FEISHU_WEBHOOK_URL`，webhook URL 只从环境变量读取，请求文本消息 JSON payload。
- Email provider 使用 `smtp://host:port?from=...&to=...&username_env=SMTP_USERNAME&password_env=SMTP_PASSWORD`，SMTP password 只从环境变量读取，支持 opportunistic / required / disabled STARTTLS。
- Docker Compose notify service 和 `.env.example` 暴露常用 provider secret 环境变量。
- 系统通知通道创建 API 和前端 provider 下拉允许 `email`、`telegram`、`feishu`。
- provider 错误会脱敏 token / webhook URL / SMTP password，避免写入 notification error 时泄露密钥。

范围外：

- 真实第三方账号联网验收、生产级模板系统、限流 / 熔断、外部回执、密钥轮换、通道更新 / 删除、凭据加密迁移、审计签名。

验证：

- `go test ./internal/notification ./internal/web/api`
- `go test ./internal/notification ./internal/web/api ./internal/store/postgres`
- `pnpm --dir web/frontend exec vitest run src/services/api/system.test.ts`
- Docker Compose mock 飞书 provider smoke：临时 mock 容器加入 `tictick-hi_default` 网络，seed `feishu://webhook?url_env=FEISHU_SMOKE_WEBHOOK` notification outbox，运行 `hi notify --once` 后 mock 收到 1 次 `/feishu` POST，payload 为文本消息；`notification_outbox` 状态为 `delivered|1|t|`，`notifications` 状态为 `sent|1|t|`。本轮证据：`taskID=tt_provider_smoke_1782584148459`、`notificationID=nt_provider_smoke_1782584148459`、`outboxID=no_provider_smoke_1782584148459`。

失败：

- 首轮 `go test ./internal/notification ./internal/web/api` 失败：测试替身 `captureMailSender` 使用值传递但 `Send` 为指针接收器。
- 已修正为指针传递，重跑通过。
- 首轮 Compose mock 飞书 smoke 使用宿主机 `host.docker.internal` 时未收到请求；诊断确认 `hi notify --once` 已 claim outbox，但 HTTP POST 因容器到宿主 mock server 超时进入 `retry_scheduled`，错误中 webhook URL 已脱敏。已改为同 Docker 网络内 mock 容器后重跑通过。

剩余风险：

- 本补充只证明 provider 构造 payload、读取 env secret、错误脱敏和 API / 前端 provider 名称可用；未使用真实 Telegram / 飞书 / SMTP 账号做外部联网验收。
- 通知模块仍为 `demo`，不能声明 usable 或 production-safe。

阶段 5 结论：

- 通知链路达到 `demo` 检查点。
- 项目整体仍为 `scaffold`，不能称为 usable、production-safe 或完成。

### 阶段 6 当前验收快照

执行时间：2026-06-27

通过：

- `go test ./...`
- `go vet ./...`
- `scripts/quality-gate.sh`
- `cd web/frontend && pnpm run typecheck`
- `cd web/frontend && pnpm run test`
- `cd web/frontend && pnpm run build`
- `git diff --check`

本地 smoke：

- 使用 `ENCRYPTION_KEY` 重建并启动 `api` 服务。
- `POST /api/system/exchange-accounts` 新建账号返回 `credentialStatus=encrypted`，响应不包含 `apiKey` / `apiSecret`。
- PostgreSQL 中新建账号 `encrypted_api_key` 和 `encrypted_api_secret` 均为 `v1:aesgcm:` 前缀，且不包含 smoke 明文 key / secret。
- 使用 encrypted enabled 账号创建 live notify 任务返回 `201`。
- 使用 disabled 账号创建 live notify 任务返回 `400`。
- 使用 legacy 非 AES-GCM 账号创建 live notify 任务返回 `400`。
- 使用 encrypted enabled 账号创建 live execute 任务返回 `400`，live execute 仍默认禁用。

失败：

- 无。

后续风险审计：

- 阶段 6 只达到 `demo` 检查点，不能声明实盘可用。
- 真实 testnet/sandbox live executor 未实现。
- 订单先落库再提交交易所、交易所响应回写和幂等 retry 仍未完成。
- 生产级 KMS / secret manager、密钥轮换和历史 `legacy` 账号迁移策略仍未完成。
- 全系统 worker lease、登录会话持久化限流 / 密码策略 / 生产级审计仍未完成。

阶段 6 结论：

- 实盘安全边界达到 `demo` 检查点。
- 项目整体仍为 `scaffold`，不能称为 usable、production-safe 或完成。

### 阶段 7 当前验收快照

执行时间：2026-06-27

通过：

- `go test ./...`
- `go vet ./...`
- `scripts/quality-gate.sh`
- `cd web/frontend && pnpm run typecheck`
- `cd web/frontend && pnpm run test`
- `cd web/frontend && pnpm run build`
- `git diff --check`

本地 smoke：

- `docker compose up -d --build` 成功，`curl -fsS http://127.0.0.1:8080/readyz` 返回健康。
- 登录成功后同时设置 `tictick_hi_session` 和 `tictick_hi_csrf` cookie。
- 不带 `X-CSRF-Token` 的 `POST /api/system/operators` 返回 `403`。
- 带 CSRF header 创建操作台账号返回 `201`，随后 `POST /api/system/operators/:id/disable` 返回 `enabled=false`，`POST /api/system/operators/:id/enable` 返回 `enabled=true`。
- 使用不存在账号连续失败登录后触发节流，最终返回 `429`。
- `GET /api/system/health` 返回 `sync-worker`、`backtest-worker`、`trading-worker`、`notify-worker`，并为 worker 暴露 `pendingCount`、`runningCount`、`lockedCount`、`staleLeaseCount`、heartbeat / locked_until 字段。
- 前端 DOM smoke：`/system/health` 渲染 worker 统计字段，`/system/operators` 渲染创建和启停操作。

失败：

- 无。

警告：

- CSRF 采用 double-submit cookie，本阶段只到本地 demo 边界，不是完整生产 CSRF/session 防护。
- 登录失败节流为 API 进程内存态，多实例、重启后持久化和全局限流未实现。
- 操作台账号启停没有 RBAC、自保护规则或强密码策略；基础操作审计已在后续阶段 7 补充中覆盖到 `demo` 边界。
- 运维健康能观察现有 task lease 字段，但全系统统一 worker lease、持续 heartbeat loop 和优雅停止状态机仍未完成。
- Vite 构建仍提示主 chunk 超过 500 kB，后续需要做路由级 code split。

阶段 7 结论：

- 运维健康和操作台账号达到 `demo` 检查点。
- 项目整体仍为 `scaffold`，不能称为 usable、production-safe 或完成。

### 阶段 7 登录会话管理补充

执行时间：2026-06-28

目标等级：demo 增量。

范围内：

- 为 `operator_sessions` 增加非敏感公开 session id，避免前端暴露 token hash。
- 登录时创建随机 `os_...` session id。
- `GET /api/auth/sessions` 返回当前操作员有效 session 列表，并标记当前 session。
- `DELETE /api/auth/sessions/:id` 撤销当前操作员的非当前 session，写请求继续要求 CSRF。
- 系统管理菜单新增登录会话页面，前端可查看 session、识别当前 session 并撤销非当前 session。

范围外：

- RBAC、自保护规则、生产级会话审计、持久化登录限流、密码策略、设备指纹、IP / UA 变更告警。

验证：

- `go test ./internal/web/api`
- `go test ./internal/data ./internal/store/postgres ./internal/web/api`
- `pnpm --dir web/frontend exec vitest run src/services/api/auth.test.ts src/router/routes.test.ts`
- `pnpm --dir web/frontend run typecheck`
- `go test ./...`
- `go vet ./...`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `scripts/quality-gate.sh`
- `git diff --check`
- `docker compose up -d --build api`
- `curl -fsS http://127.0.0.1:8080/readyz`
- 本地 HTTP smoke：两次登录后 `GET /api/auth/sessions` 能返回当前和非当前 session；`DELETE /api/auth/sessions/:id` 撤销非当前 session 后对应 cookie 请求 `/api/auth/me` 返回 `401`；撤销当前 session 返回 `409 invalid_state`；响应未暴露 `tokenHash`。
- Headless Chrome DOM smoke：登录后打开 `/system/sessions`，页面标题为 `登录会话`，表格渲染 session 行，显示当前会话标记，无错误状态。

失败：

- 首轮 `scripts/quality-gate.sh` 失败：`internal/web/api/server_test.go` 超过 700 行、`internal/store/postgres/system_store.go` 超过 500 行。
- 已通过拆分 `internal/web/api/auth_session_test.go` 和 `internal/store/postgres/auth_session_store.go` 关闭，重跑 `scripts/quality-gate.sh` 通过。

剩余风险：

- 这只是基础 session 管理，不是生产级登录安全；仍缺持久化限流、密码策略、RBAC / 自保护、设备上下文和生产级审计。
- 本轮未运行完整 `scripts/stage8-smoke.sh`；session 路由用本地 HTTP / DOM smoke 覆盖。

### 阶段 7 系统操作审计日志补充

执行时间：2026-06-28

目标等级：demo 增量。

范围内：

- 新增 `audit_events` PostgreSQL 表，记录操作者、动作、资源、结果、请求路径、来源地址、User-Agent、元数据和创建时间。
- `POST /api/auth/login` 成功 / 失败、`POST /api/auth/logout`、`DELETE /api/auth/sessions/:id` 写入基础认证审计事件。
- 系统管理写操作写入基础操作审计事件：通知重试、通知通道创建、交易所账号创建、操作员创建、操作员启用 / 禁用。
- 新增 `GET /api/system/audit-events`，按时间倒序返回最近审计事件。
- 系统管理菜单新增“操作审计”页面，前端能查看时间、操作者、动作、资源、结果、请求和元数据。
- 审计元数据不记录操作员密码、API secret、session token hash。

范围外：

- RBAC、自保护规则、不可篡改审计、集中日志、签名 / hash chain、审计留存策略、完整审计 taxonomy、全量 request / response schema。

验证：

- `go test ./internal/web/api ./internal/store/postgres`
- `pnpm --dir web/frontend exec vitest run src/services/api/system.test.ts src/router/routes.test.ts`
- `go test ./...`
- `go vet ./...`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `scripts/quality-gate.sh`
- `docker compose up -d --build api`
- `curl -fsS http://127.0.0.1:8080/readyz`
- 本地 HTTP smoke：登录后创建并禁用操作员，`GET /api/system/audit-events?limit=30` 返回 `auth.login`、`operator.create`、`operator.disable`，响应不包含 `secret123`、`apiSecret`、`tokenHash`、`tictick_hi_session`。
- Headless Chrome DOM smoke：登录后打开 `/system/audit-events`，页面标题为 `操作审计`，表格渲染 6 行，包含 `auth.login`。

失败：

- 无。

剩余风险：

- 审计写入当前和业务写操作不是同一数据库事务，部分成功 / 审计失败的边界还不是 production-safe。
- 审计日志可被数据库管理员直接修改，没有不可篡改链、签名、外部集中日志或留存策略。
- `X-Forwarded-For` 仅做基础读取，生产代理信任边界未定义。
- 本轮未运行完整 `scripts/stage8-smoke.sh` 或 `scripts/stage8-sigterm-smoke.sh`。

## 5. 下一条可推进切片

下一步只推进：

```text
阶段 8：整体 usable 验收
```

目标等级：尝试从 `scaffold` 推进到 `usable`，但只有全链路 smoke 和完整质量门禁证明后才能升级。

Definition of Done：

- Docker Compose 一键启动，迁移、api、sync、backtest、trading、notify 服务均健康或可观察。
- 研究 -> 策略 -> 回测 -> 模拟盘 -> 通知全链路 smoke 通过，并能从前端观察关键结果。
- 数据同步、CandleProvider、策略、回测、模拟盘、通知和系统管理文档与实现一致。
- 所有模块等级重新审计，不能把 demo 风险冒充 usable。
- 完整质量门禁通过。

范围外：

- 真实实盘 production 下单。
- 企业级 SSO / RBAC / 审计。
- Kubernetes / 多实例生产编排。
- 真实第三方通知 provider 生产启用边界。

### 阶段 8 当前验收快照：全链路 smoke gate

执行时间：2026-06-27

新增验收入口：

- `scripts/stage8-smoke.sh`
- `scripts/stage8-sigterm-smoke.sh`

通过：

- `scripts/stage8-smoke.sh`
- `scripts/stage8-sigterm-smoke.sh`
- `go test ./...`
- `go vet ./...`
- `scripts/quality-gate.sh`
- `cd web/frontend && pnpm run typecheck`
- `cd web/frontend && pnpm run test`
- `cd web/frontend && pnpm run build`
- `git diff --check`

全链路 smoke 覆盖：

- `docker compose up -d --build` 成功，`/readyz` 返回健康。
- 登录 API 成功并设置 session / CSRF cookie；后续写请求带 `X-CSRF-Token`。
- `GET /api/strategies` 能观察到 `ema-cross` 策略 registry。
- 通过 API 创建专用数据同步任务，并向 `market_candles` 写入专用 `1m` K 线事实数据。
- 通过 API 创建被强制置为 active lease 的 sync / realtime / paper 任务，调用 stop / pause 后 PostgreSQL 断言 `status=paused` 且 `locked_by`、`locked_until`、`heartbeat_at` 均为空。
- `GET /api/candles?interval=5m` 返回 `source=aggregated`、`health=ok`、`baseInterval=1m`，证明研究页和 CandleProvider 可用同一数据源。
- 通过 API 创建 `webhook-demo` 通知通道。
- 通过 API 创建回测任务，`hi backtest --once` 执行后任务进入 `succeeded`，并能读取 strategy intents 和 backtest orders；PostgreSQL 断言 backtest task lease 已释放。
- 通过 API 同时启动 paper execute 和 paper notification 两个 `running` 交易任务；`hi trading --once` 多次执行后两个任务都被 claim，并分别产生 paper orders / executions / positions 和 notification records。
- `hi notify --once` 投递后所有通知进入 `sent`。
- PostgreSQL 断言 notification outbox delivered rows 已释放 lock。
- `GET /api/system/health` 能观察 `sync-worker`、`backtest-worker`、`trading-worker`、`notify-worker`。

本轮 smoke 证据：

- symbol：`S81782570320USDT`
- data task：`dst_901766ee25f5c725d68ef668`
- backtest：`bt_e5852bb9f6c1bfd860528642`
- paper execute：`tt_dbb6e69ac20fad0f9c528d1a`
- paper notify：`tt_22a103a75713aee9b630061a`
- notification channel：`stage8-smoke-1782570320`

前端 DOM smoke：

- `/research?exchange=binance&symbol=S81782549588USDT&interval=5m` 显示专用 symbol、`K 线来源: 内部聚合` 和数据健康。
- `/backtests/bt_fdd2e012fe2b539b9e8bfabc` 显示专用 symbol、完成状态、订单数和回测详情。
- `/trading/tt_c83e8ebd5ec045feaf0849b6` 显示专用 symbol 和通知通道。
- `/system/notifications` 显示 `stage8-smoke-1782549588` 和 `sent` 状态。
- 浏览器 DOM smoke 未捕获 console error 或 page error。

前端图表回归修复：

- K 线图表 resize 不再读取图表库挂载节点或图表组件自身高度，观察目标收敛到页面提供的布局宿主。
- 研究页图表面板改为固定两行 grid：任务列表在上，图表在下，`.research-chart-body` 使用 `minmax(0, 1fr)` 分配剩余高度。
- 图表尺寸读取后会被夹在所在 `.chart-panel` 的可用高度和 `1200px` 安全上限内；即使 ResizeObserver 收到异常膨胀的 viewport 高度，也不会把异常高度写回 lightweight-charts。
- `.trading-chart` 脱离普通文档流并明确 `height: 100%` 铺满固定宿主，lightweight-charts 根节点和 canvas 被约束在宿主尺寸内，避免内部 DOM 高度反向撑开页面。
- headless Chrome 本地采样 `/research` 桌面 `2048x1034`：`scrollHeight=1318`、`panel=760`、`chart=683`、`canvas=682`、`tv=682` 连续 16 次稳定。
- headless Chrome 本地采样 `/research` 移动 `390x844`：`scrollHeight=1256`、`panel=624`、`chart=457`、`canvas=456`、`tv=456` 连续 10 次稳定。

已修正的 trading claim 公平性问题：

- 早期 trading claim 只按 `created_at ASC` 选择 `running` 任务，paper execute 任务保存结果释放锁后仍会继续排在队首，后续 paper notification 任务可能长期无法被 claim。
- `ClaimTradingTask` 改为 `ORDER BY updated_at ASC, created_at ASC`；`SaveTradingRunResult` 释放锁时会更新 `updated_at=now()`，让刚处理过的 task 回到队尾。
- 新增 migration `0010_trading_claim_fairness.sql` 为 `running` task claim 建立 `(status, updated_at, created_at)` 索引。
- Stage 8 smoke 不再用“先 pause execute 再创建 notification”规避问题，而是同时启动两个 `running` paper 任务并验证二者都被处理。

已修正的数据同步 heartbeat 问题：

- `SyncRepository` 增加 `HeartbeatDataSyncTask`，PostgreSQL 实现按 `task_id + locked_by + running` 条件刷新 `heartbeat_at` 和 `locked_until`。
- `hi sync` 增加 `SYNC_HEARTBEAT_INTERVAL` 配置，默认随 `SYNC_LEASE_TTL / 3`，compose 和 `.env.example` 已暴露入口。
- data sync runner 在 fetch / retry / save 期间运行 heartbeat loop；保存 K 线结果前会再次 heartbeat，lease 丢失时不写入结果。
- 单元测试覆盖长 fetch 期间 heartbeat 刷新，以及 heartbeat lease lost 后不保存 K 线结果。
- 登录后 `/api/system/health` 返回 `sync-worker` 健康摘要：`pending=0 running=5 locked=0 stale=0`。

已收敛的 worker heartbeat 实现：

- 新增 `internal/workerlease.RunWithHeartbeat`，统一执行初始 heartbeat、周期性 heartbeat、heartbeat 失败后取消任务上下文和错误传播。
- data sync、backtest、trading runner 已复用同一 helper，不再各自复制 heartbeat loop。
- `internal/workerlease` 单元测试覆盖初始 heartbeat、运行中刷新和 heartbeat 失败取消任务。

已修正的用户停止释放 lease 问题：

- `SetSyncEnabled(false)` / `SetRealtimeEnabled(false)` 进入 `paused` 时清理 `locked_by`、`locked_until`、`heartbeat_at` 并写入 `finished_at`。
- `SetTradingTaskStatus(paused|failed|cancelled)` 清理 active lease；trading runner 正常释放和 failed 路径也会清理 `heartbeat_at`。
- Stage 8 smoke 使用真实 API 创建任务、直接模拟 active lease、再调用 stop / pause API，并用 PostgreSQL 断言锁字段为空。

已收敛的 worker lease 终态清理：

- 新增 `internal/store/postgres/lease.go`，集中定义 task / outbox lease 字段和清锁 SQL 片段。
- data sync、backtest、trading、notification outbox 的 release / fail / pause 路径复用共享 helper，不再各自手写 `locked_by` / `locked_until` / `heartbeat_at` 清理片段。
- Stage 8 smoke 额外断言 backtest succeeded 后 task lease 释放、notification delivered 后 outbox lock 释放。
- 单元测试覆盖 task 表必须清理 `heartbeat_at`、notification outbox 不引用不存在的 `heartbeat_at`、条件清锁保留非终态 lease。

已收敛的 worker lease claim 共享字段：

- `internal/store/postgres/lease.go` 新增 `claimableLeasePredicate`，集中表达 `locked_until IS NULL OR locked_until < now()` 可 claim 条件。
- `internal/store/postgres/lease.go` 新增 `claimLeaseAssignments`，集中写入 `locked_by`、`locked_until`、可选 `heartbeat_at`、额外 claim 字段、`attempt_count` 和 `updated_at`。
- data sync、backtest、trading、notification outbox 的 claim 更新复用共享 helper；各自的领域候选条件、排序、公平性和返回字段保持不变。
- 单元测试覆盖 task 表 claim 必须写入 `heartbeat_at`、notification outbox claim 不引用不存在的 `heartbeat_at`、额外字段和过期谓词。

本轮追加收敛：

- `internal/store/postgres/lease.go` 新增 `claimLeaseID` / `claimLeaseIDSQL`，集中生成 `SELECT <id> FROM <lease table> ... locked_until IS NULL OR locked_until < now() ... FOR UPDATE SKIP LOCKED` 查询。
- data sync、backtest、trading、notification outbox 的 claim id 查询复用 `claimLeaseID`；领域候选条件、排序和返回字段保持原语义。
- `internal/store/postgres/lease.go` 新增 `heartbeatLease`，data sync、backtest、trading 的 PostgreSQL heartbeat 刷新复用同一 helper；notification outbox 没有 `heartbeat_at` 字段，helper 会拒绝 heartbeat。
- 单元测试覆盖 claim SQL 的资源表 / 候选条件 / 过期谓词 / 排序、heartbeat SQL 字段、无行影响时报告 lease lost，以及 notification outbox 不允许 heartbeat。

本轮继续收敛：

- `internal/store/postgres/lease.go` 新增 `claimLeaseUpdateSQL` / `claimLeaseRow`，集中执行“选中候选 id 后更新状态、写入 lease 字段、递增 attempt、返回领域行”的 claim 状态更新路径。
- data sync、backtest、trading、notification outbox 的 claim 更新路径复用 `claimLeaseRow`；领域候选条件、排序、返回字段、扫描函数和 notification 主表同步更新保持原语义。
- 单元测试覆盖 claim update SQL 的状态 / lease / attempt / returning 字段、先 select 再 update 的调用顺序，以及没有候选行时不得执行 update。

本轮非 claim 转移收敛：

- `internal/store/postgres/lease.go` 新增 `leaseTransitionUpdateSQL`，集中生成“状态/结果字段更新 + 清理 lease 字段 + `updated_at`”的 transition update SQL。
- data sync 保存结果 / 永久失败 / 临时重试、backtest 成功 / 失败 / shutdown release、trading 失败、notification delivered / failed outbox 更新复用 `leaseTransitionUpdateSQL`；领域特有字段、返回字段和 notification 主表同步更新保持原语义。
- 单元测试覆盖 task 表 transition 必须清理 `heartbeat_at`、notification outbox transition 不引用 `heartbeat_at`、`RETURNING` 和带状态守卫的 `WHERE` 必须保留。

已收敛的 worker 取消释放 lease 路径：

- `internal/workerlease` 新增 shutdown 判定和不继承父取消的 release context，避免用已取消的请求上下文做收尾写库。
- data sync、backtest、trading、notification runner 在父上下文取消时释放当前 active lease，并跳过 MarkFailed / MarkNotificationFailed。
- backtest shutdown release 会把仍处于 `running` 的任务复位为 `pending`，否则 backtest claim 只选 `pending` 会导致任务卡死。
- 单元测试覆盖 sync/backtest/trading/notification 的 shutdown release、不保存部分结果、不误标失败；同时覆盖普通 `context.Canceled` 业务错误不会被误判为进程 shutdown，以及父 context 已取消后外部库返回非标准错误也会进入 shutdown release。

已补充的 worker 容器 SIGTERM 收尾证据：

- 新增 `scripts/stage8-sigterm-smoke.sh`，通过临时 Docker Compose override 注入 `sigterm-market` 慢速 Binance-compatible mock，不依赖真实交易所网络。
- 脚本创建专用 `S8TERM...` realtime 数据同步任务，只清理同命名空间的历史 smoke 任务，不暂停普通用户同步任务。
- sync 容器使用 `SYNC_WORKER_ID=stage8-sigterm-...`、1s heartbeat 和 mock `BINANCE_BASE_URLS`，脚本等待 PostgreSQL 显示任务已被 claim、`locked_until` 有效、`heartbeat_at` 存在，同时 mock `/api/v3/klines` 请求保持 pending。
- 通过 `docker compose stop -t 10 sync` 发送容器级 SIGTERM 后，脚本断言该任务仍为 `running` / `realtime_enabled=true`，且 `locked_by`、`locked_until`、`heartbeat_at` 已全部清空、`last_error` 为空、`attempt_count > 0`。
- 脚本写入确定性 `1m` K 线后，使用 PostgreSQL `ACCESS EXCLUSIVE` 表锁阻塞 backtest / trading 的 CandleProvider 查询，避免依赖睡眠型测试钩子。
- backtest 容器使用 `BACKTEST_WORKER_ID=stage8-sigterm-backtest-...` 和 6s lease；脚本等待任务进入 `running`、heartbeat 写入后停容器，并断言任务复位为 `pending` 且锁字段清空。
- trading 容器使用 `TRADING_WORKER_ID=stage8-sigterm-trading-...` 和 6s lease；脚本等待任务进入 `running`、heartbeat 写入后停容器，并断言任务保持 `running` 且锁字段清空。
- notify 容器使用 `NOTIFY_WORKER_ID=stage8-sigterm-notify-...` 和 6s lease；脚本 seed `webhook` outbox 并让 provider 阻塞在慢 HTTP POST，停容器后断言 `notification_outbox.locked_by` / `locked_until` 清空且没有记录投递错误。
- 本轮 smoke 证据：symbol `S8TERM1782573041USDT`、data task `dst_6784171c4299cbd8456a980f`、sync worker `stage8-sigterm-1782573041`、backtest `bt_9f1b9e3d09fa9fcc711a0b9f`、backtest worker `stage8-sigterm-backtest-1782573041`、trading `tt_55bb977ce4d28b3bd16cfcd6`、trading worker `stage8-sigterm-trading-1782573041`、notification `nt_s8term_1782573041`、notify worker `stage8-sigterm-notify-1782573041`。

已收敛的交易所 K 线错误边界：

- `internal/exchange` 提供共享 HTTP status / transport error 分类和 endpoint 错误摘要，避免 adapter 泄露完整请求路径和 query 参数。
- Binance / OKX adapter 统一将 EOF、deadline、transport error、HTTP 429、HTTP 5xx 识别为临时错误；OKX 业务码 `50011` 识别为临时限流错误，`51001` 等配置 / symbol 错误不重试。
- `hi sync` 继续通过 `SYNC_FETCH_RETRIES` / `SYNC_RETRY_DELAY` 对临时 market data 错误做有限重试；`last_error` 保持规范化和 500 rune 截断。
- 单元测试覆盖 URL 摘要脱敏、临时 / 永久错误分类、Binance fallback、OKX rate-limit 码和 sync runner 临时错误重试。

已补充的数据同步失败恢复入口：

- 新增 `POST /api/data/tasks/:id/retry`，用于从研究页恢复失败的数据同步任务。
- PostgreSQL `RetryDataSyncTask` 只接受 `failed` 任务，并会将任务恢复为 `pending`、重新打开 `sync_enabled`、清理 `last_error`、`locked_by`、`locked_until`、`heartbeat_at` 和 `finished_at`，使 `hi sync` 能再次 claim。
- 研究页失败任务的同步操作位显示明确的 retry 按钮，不再让用户依赖含义不清的重新同步按钮。
- 前端 API wrapper、研究页 composable 和表格事件已接入 retry 后刷新任务列表。
- 后端 route 测试覆盖 retry API 返回 `pending`、`syncEnabled=true`、`lastError=""`。
- PostgreSQL 集成测试覆盖 failed task retry 后 lease 释放、错误清理、后续 `ClaimDataSyncTask` 可重新领取，以及 running task retry 不会清理 active lease。
- 前端测试覆盖 `/api/data/tasks/:id/retry` 调用和失败行 retry 事件。
- 本轮完整验证通过：`go test ./...`、`go vet ./...`、`scripts/quality-gate.sh`、`cd web/frontend && pnpm run typecheck`、`cd web/frontend && pnpm run test`、`cd web/frontend && pnpm run build`、`scripts/stage8-smoke.sh`、`git diff --check`。
- 本轮 stage8 smoke 证据：symbol `S81782567804USDT`、data task `dst_6b653f85c1c419c924bfeafd`、backtest `bt_9b646cb1533bd879b44b2acf`、paper execute `tt_37a4340193eb71bd62a8d242`、paper notify `tt_e4bb9739cf5a05237761f9ef`、notification channel `stage8-smoke-1782567804`。

阶段 8 usable readiness 重审计：

| 模块 | 重审计等级 | 可用证据 | usable 阻断项 |
| --- | --- | --- | --- |
| 架构文档 | usable | 主计划、交付协议和质量审计能约束实现顺序与等级声明 | 需要随实现持续校准，不阻断阶段 8 |
| Go 子命令 | scaffold | `hi api/sync/backtest/trading/notify/migrate` 可由 compose 和 smoke 调用 | 日志、配置错误边界、运行手册和优雅停止证据不足 |
| Docker Compose | demo | `scripts/stage8-smoke.sh` 从 compose build/up 进入并完成全链路 smoke；`scripts/stage8-sigterm-smoke.sh` 从 compose stop 进入并验证 data sync / backtest / trading / notify 收尾 | 缺备份/恢复、资源限制、外部依赖失败策略和共享环境部署说明 |
| PostgreSQL migrations | scaffold | 当前 smoke 可从 migrations 建库并运行；`0011_domain_constraints.sql` 已补充核心状态、类型、数值和时间范围 CHECK，`0012_referential_constraints.sql` 已补充 orders / executions / positions / notifications / outbox / backtest_orders 的核心 FK 和同 task composite FK，`0016_worker_lease_constraints.sql` 已补充 task/outbox lease 字段一致性 CHECK，`0017_strategy_intent_parent_constraints.sql` 已补充 `strategy_intents` 新增/更新父任务归属约束，`0018_strategy_intent_parent_delete_guards.sql` 已补充父任务删除防 orphan 保护，`0019_task_terminal_timestamp_constraints.sql` 已补充任务终态 `finished_at` 一致性约束，`0020_validate_worker_lease_constraints.sql` 已修补历史半截 lease 并 VALIDATE worker lease CHECK，`0021_task_status_transition_guards.sql` 已补充 data sync / backtest / trading 核心状态流转 trigger；`scripts/stage8-migration-audit.sh` 已校验迁移全量应用、worker lease CHECK validated、状态流转 trigger、终态 finished_at、lease、intent parent 和核心事实 orphan | 完整统一状态机、全量历史数据验证、数据迁移/回滚策略不足 |
| API server | scaffold | 核心路由已拆分，CSRF 写保护、策略参数校验、retry API、结构化错误响应和基础操作审计可测；前端 API client 会读取服务端 `message/error` 并保留 `code`；已知 API 路径的方法错误返回 405 和 `Allow` header；`GET /api/system/api-contract` 返回基础 OpenAPI 3.1 contract，覆盖当前前端路由、request body、success schema、错误 schema、session cookie 和 CSRF header；`TestFrontendAPI*` 和 `scripts/check-api-contract-drift.sh` 会阻止前端 service route、request DTO、核心 response DTO、adapter response 字段和 candle query 参数漂移 | TS 类型自动生成、全量错误分类、生产级审计边界和 OpenAPI 外部校验不足 |
| 登录会话 | demo | HttpOnly session、CSRF double-submit、登录失败节流、session 列表和撤销有 route / smoke 覆盖；登录成功 / 失败、退出、session 撤销已进入基础操作审计 | 限流内存态、无密码策略/RBAC、自保护规则和生产级设备上下文 |
| 数据同步 worker | demo | claim/heartbeat/upsert/retry/release、失败后 UI retry、Stage 8 smoke 和容器 SIGTERM smoke 有覆盖 | 未证明真实交易所网络下长期恢复、全局限流和完整状态机 |
| CandleProvider | demo | native/aggregated/gap/coverage metadata、runner 健康门禁和集成测试已覆盖 | 大范围分页/游标、性能压测、异常数据修复策略不足 |
| Binance / OKX adapter | demo | 临时错误分类、Binance fallback、OKX rate-limit 码和 URL 脱敏有测试 | 无全局限流器、真实网络压测、代理/地域策略和完整业务码审计 |
| 研究页 | demo | 数据源 metadata、列表在上图表在下、图表高度稳定、失败任务 retry 已覆盖 | 交易对硬编码、图表工具薄、缺时间范围/指标/缺口修复工作流 |
| 策略 registry / runtime | demo | schema 驱动参数、intent 输出和策略边界门禁已覆盖 | 缺策略沙箱、版本迁移、权限隔离和真实策略库 |
| 回测 | demo | CandleProvider、closed/minute replay、intent/order/result、买卖点展示和容器 SIGTERM release 已走通 | 撮合、费用/滑点曲线、指标体系和结果可信度不足 |
| 交易 runner | demo | paper execute/notification、position/order/execution/outbox、claim 公平性和容器 SIGTERM release 已走通；通知 intent 可进入 email / Telegram / 飞书 provider 基础发送路径 | 风控、PnL 可信度、通知 provider 生产启用边界、统一状态机和实盘隔离不足 |
| 实盘安全 | demo | 凭据 AES-GCM、本地 live 任务创建护栏和 live execute 禁用已验证 | testnet/sandbox executor、订单先落库再提交、幂等 retry、KMS/轮换未完成 |
| 通知 | demo | outbox、local/webhook-demo/webhook/email/Telegram/飞书 provider、失败重试、系统页 retry 和 notify 容器 SIGTERM release 已覆盖 | 真实第三方账号联网验收、模板、限流、审计和通道管理不足 |
| 前端基础设施 | scaffold | Vue/Naive/Pinia/i18n/主题/API wrapper/图表封装已存在并通过测试；路由级 code split 已让生产入口 chunk 降至 437.44 kB，构建不再出现 Vite 大 chunk 警告；概览页已接入真实 API 聚合 | 缺系统性桌面/移动/主题视觉回归 |
| 概览页 | demo | 从现有 API 聚合系统健康、data sync、backtest、trading 和 notification，展示关键计数、异常提醒、worker 状态和最近活动；`useOverviewWorkspace` 单测覆盖聚合契约 | 缺时间窗口筛选、趋势图、关键操作入口和生产级监控语义 |
| 系统管理 / 运维健康 | demo | 操作台账号启停、当前操作员 session 管理、基础操作审计页、健康页 worker 统计和通知/账号管理可用 | 无 RBAC、自保护、不可篡改审计和生产监控 |
| 质量门禁 | demo | 通用门禁、API contract route / field drift gate、stage8 smoke、data sync/backtest/trading/notify SIGTERM smoke、scaffold 声明检查可重复运行 | 尚未把真实网络压测、TS 类型生成校验、视觉回归和安全审计纳入硬门禁 |

重审计结论：

- Stage 8 全链路 smoke gate 已建立，但多个核心模块仍只到 `demo` 或 `scaffold`。
- 当前没有足够证据把项目整体升级为 `usable`。
- 下一步必须优先关闭能支撑真实工作的 blocker，而不是继续铺新页面或新 provider 空壳。
- 本轮重审计验证通过：`go test ./...`、`go vet ./...`、`scripts/quality-gate.sh`、`cd web/frontend && pnpm run typecheck`、`cd web/frontend && pnpm run test`、`cd web/frontend && pnpm run build`、`scripts/stage8-sigterm-smoke.sh`、`scripts/stage8-smoke.sh`、`git diff --check`。
- 本轮重审计 smoke 证据：symbol `S81782573245USDT`、data task `dst_f50d94d45951e6efd06e39fb`、backtest `bt_421f6f5b1770e6681318ce3a`、paper execute `tt_92ee30d68ee5fa7ab7a3e2f8`、paper notify `tt_abd4aa87976068292c816fa2`、notification channel `stage8-smoke-1782573245`。
- 本轮 worker SIGTERM smoke 证据：symbol `S8TERM1782573041USDT`、data task `dst_6784171c4299cbd8456a980f`、sync worker `stage8-sigterm-1782573041`、backtest `bt_9f1b9e3d09fa9fcc711a0b9f`、backtest worker `stage8-sigterm-backtest-1782573041`、trading `tt_55bb977ce4d28b3bd16cfcd6`、trading worker `stage8-sigterm-trading-1782573041`、notification `nt_s8term_1782573041`、notify worker `stage8-sigterm-notify-1782573041`。

失败：

- 无当前硬失败。

警告：

- Stage 8 当前已建立可重复全链路 smoke gate，并完成 usable readiness 重审计；重审计显示多个核心模块仍为 `demo` 或 `scaffold`，不能把整体升级为 `usable`。
- 全链路 smoke 使用确定性 seed K 线，不依赖真实交易所网络；它证明内部链路，不证明 Binance / OKX 外部稳定性。
- 交易所 adapter 仍缺全局限流器、代理 / 地域网络策略、更多 OKX / Binance 业务错误码审计和真实网络压测。
- worker claim id 查询、claim 状态更新、部分非 claim 状态更新、共享字段、过期谓词和 PostgreSQL heartbeat 刷新已收敛，runner 级 shutdown release 已有单元证明，data sync / backtest / trading / notify 容器 SIGTERM 数据库断言已补齐，数据库已拒绝半截 lease 和非 running 加锁；但领域候选条件、排序、完整状态流转约束和部分业务状态切换仍未抽取为完整统一状态机。
- 回测撮合、paper position PnL、真实通知 provider 生产启用边界、实盘 testnet/sandbox 和生产级会话/RBAC/审计仍是后续风险。
- Vite 主入口 chunk 过大已由路由级 code split 关闭；前端仍缺系统性桌面 / 移动 / 主题视觉回归。

阶段 8 当前结论：

- 整体全链路 smoke gate 达到 `demo` 证据增强。
- 项目整体仍为 `scaffold`，不能称为 usable、production-safe 或完成。

### 阶段 8 前端基础设施 code split 补充

执行时间：2026-06-27

触发问题：

- Stage 8 readiness 重审计将前端基础设施列为 `scaffold`，其中一个明确 blocker 是 Vite 构建主 chunk 超过 500 kB。
- 页面组件全部静态 import 到 `routes.ts`，研究、回测、交易和系统管理页面一起进入入口 bundle。

修复范围：

- `routes.ts` 中 AppShell 和所有页面组件改为动态 import，实现路由级 lazy loading。
- `routes.test.ts` 新增断言：所有带 component 的 route 都必须保持 lazy component 函数，防止后续回退到静态页面 import。
- 不新增业务页面，不调整路由语义，不把前端基础设施升级为 usable。

验证：

- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `go test ./...`
- `go vet ./...`
- `scripts/quality-gate.sh`

构建证据：

- 生产入口 JS 从此前约 `1,219.02 kB` 拆分为 `index-DYiXLYv6.js 437.44 kB`。
- 研究页、回测详情、交易详情、系统管理页面均生成独立 lazy chunks。
- 本轮 `pnpm run build` 未再出现 Vite `Some chunks are larger than 500 kB` 警告。

失败：

- 无硬失败。

剩余风险：

- 前端基础设施仍为 `scaffold`；缺系统性桌面 / 移动 / 主题视觉回归，不能因 code split 声明 usable。

### 阶段 8 概览页真实聚合补充

执行时间：2026-06-28

触发问题：

- Stage 8 readiness 重审计将概览页列为 `scaffold`，因为页面只有 scaffold 状态和基础健康入口，不是真实业务概览。
- 概览页无法回答当前同步、回测、交易、通知和 worker 健康的整体状态。

修复范围：

- `OverviewPage.vue` 改为真实聚合视图，从现有 API 加载系统健康、数据同步任务、回测任务、交易任务和系统通知记录。
- 新增 `useOverviewWorkspace`，集中生成关键计数、异常提醒、worker health 摘要和最近活动列表，避免在页面模板中堆业务计算。
- 概览页继续显示整体 `scaffold` 等级，不把局部改进冒充整体可用。
- 不新增后端汇总接口，不改变数据库 schema，不展示通知 target、交易所 API key 或 secret。

验证：

- `pnpm --dir web/frontend run test -- src/composables/useOverviewWorkspace.test.ts`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `go test ./...`
- `go vet ./...`
- `scripts/quality-gate.sh`
- `git diff --check`
- `docker compose up -d --build api`
- `curl -fsS http://127.0.0.1:8080/readyz`
- 新增 `useOverviewWorkspace.test.ts` 覆盖五类 API 聚合、summary card、alert、recent activity 和加载失败状态。
- Headless Chrome 桌面 `1440x900` 登录并打开 `/overview`，渲染 5 个真实指标卡：数据同步、回测任务、交易任务、通知、后台服务；渲染系统健康、异常提醒、最近活动 3 个面板，无页面错误。
- Headless Chrome 移动 `390x844` 登录并打开 `/overview`，`documentWidth=390` 等于视口宽度，渲染 5 个指标卡和 3 个面板，无横向溢出。

阶段结论：

- 概览页从 `scaffold` 升级为 `demo`。
- 项目整体仍为 `scaffold`，不能称为 usable 或 production-safe。

剩余风险：

- 概览页仍缺时间窗口筛选、趋势图、关键操作入口、视觉回归和生产级监控语义。

### 阶段 8 PostgreSQL domain constraints 补充

执行时间：2026-06-27

触发问题：

- Stage 8 readiness 重审计将 PostgreSQL migrations 维持为 `scaffold`，其中一个明确 blocker 是核心表缺少数据库层 domain 约束。
- 仅靠应用层枚举和参数校验不能阻止直接写库、历史脚本或未来 worker bug 写入非法状态、非法类型或明显不可能的数值。

修复范围：

- 新增 `0011_domain_constraints.sql`，为 data sync、market candles、backtest、trading、strategy intents、orders、notifications、exchange accounts、operators、outbox、executions 和 positions 补充核心 CHECK 约束。
- 约束范围覆盖状态枚举、任务类型、订单方向、通知 provider、K 线 OHLC 边界、时间范围、尝试次数和基础价格/数量非负边界。
- 集成测试新增非法 domain value 写入拒绝断言，并要求错误中包含具体约束名。
- 不新增业务字段，不改变 API 行为，不把 migrations 升级为 usable。

验证：

- `go test ./internal/store/postgres`
- Docker network PostgreSQL 集成测试：`go test ./internal/store/postgres -run Integration -count=1 -v`
- `go test ./...`
- `go vet ./...`
- `scripts/quality-gate.sh`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `scripts/stage8-smoke.sh`
- 当前 compose 数据库已记录 `schema_migrations.version = '0011_domain_constraints.sql'`

本轮 smoke 证据：

- symbol `S81782575757USDT`
- data task `dst_cc86b8502161bd81c70bbd73`
- backtest `bt_c75662af21a13631d37e1d19`
- paper execute `tt_996b4185ac27573ca9d9d514`
- paper notify `tt_d26ba1ae3d68c27c76d6f110`
- notification channel `stage8-smoke-1782575757`

过程发现：

- 首次用 `127.0.0.1:5432` 跑集成测试失败，因为本地 compose PostgreSQL 端口未发布；改为 Docker network 内连接后通过。
- 原计划把 `attempt_count <= max_attempts` 作为通知历史记录硬约束，但现有失败通知存在 `attempt_count > max_attempts` 的合法历史终态；迁移改为保留 `attempt_count >= 0` 和 `max_attempts > 0`，避免破坏历史数据。

失败：

- 无当前硬失败。

剩余风险：

- PostgreSQL migrations 仍为 `scaffold`；核心事实表 FK、worker lease 一致性 CHECK、`strategy_intents` 新增/更新父任务归属约束和父任务删除防 orphan 保护已补，但完整状态流转约束、历史数据验证、数据修复迁移和 rollback 策略仍未关闭。

### 阶段 8 PostgreSQL referential constraints 补充

执行时间：2026-06-28

触发问题：

- `0011_domain_constraints.sql` 只约束了状态、类型、数值和时间范围，仍允许下游事实表直接写入不存在的 task、intent、order 或 notification 引用。
- Stage 8 readiness 重审计中的 PostgreSQL migrations 仍不能因为有 CHECK 就升级；真实可用前至少要阻止 orphan 交易事实和通知事实。

修复范围：

- 新增 `0012_referential_constraints.sql`，为 trading tasks、strategy intents、orders、notifications 增加必要 composite unique key，以支持同 task 参照校验。
- 为 orders / executions / positions 增加到 trading_tasks 的 FK，并让 task_type 参与校验。
- 为 orders / executions / notifications / notification_outbox / backtest_orders 增加到 strategy_intents 的同 task FK。
- 为 executions 增加到 orders 的同 task FK；为 notification_outbox 增加到 notifications 的同 task FK。
- `SaveBacktestResult` 改为先删除旧 backtest orders，再删除旧 backtest intents，避免新增 FK 后重跑结果保存时违反引用顺序。
- 约束相关 integration tests 拆分到 `integration_constraints_test.go`，避免 `integration_test.go` 超过 700 行硬上限。
- 不引入触发器，不在本轮强行解决 `strategy_intents` 同时服务 backtest / paper / live 的多态 task FK。

验证：

- `go test ./internal/store/postgres`
- Docker network PostgreSQL 集成测试：`go test ./internal/store/postgres -run Integration -count=1 -v`
- `go test ./...`
- `go vet ./...`
- `scripts/quality-gate.sh`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `scripts/stage8-smoke.sh`
- 当前 compose 数据库已记录 `schema_migrations.version = '0012_referential_constraints.sql'`

本轮 smoke 证据：

- symbol `S81782576597USDT`
- data task `dst_9ab919b74de8a05d94023de5`
- backtest `bt_c8c2209e0cfa660ac26f0b1b`
- paper execute `tt_7ce89f624091ca213a3bb7db`
- paper notify `tt_e661d7e225b05bf72579499a`
- notification channel `stage8-smoke-1782576597`

过程发现：

- 初版约束测试直接放入 `integration_test.go` 后，`scripts/quality-gate.sh` 失败：该文件达到 948 行，超过 700 行硬上限。
- 已通过拆分 `integration_constraints_test.go` 修复，拆分后 `integration_test.go` 为 554 行、`integration_constraints_test.go` 为 403 行。

失败：

- 无当前硬失败。

剩余风险：

- PostgreSQL migrations 仍为 `scaffold`；完整状态流转约束、历史数据验证、数据修复迁移、rollback 策略和生产数据备份/恢复验证仍未关闭。

### 阶段 8 PostgreSQL worker lease constraints 补充

执行时间：2026-06-28

触发问题：

- Worker lease 的应用层 claim / heartbeat / release helper 已持续收敛，但数据库仍允许直接写入半截 lease 字段，例如只有 `locked_by` 没有 `locked_until` / `heartbeat_at`。
- 数据库也允许非 `running` 状态持有 active lock，这会削弱恢复、观测和后续状态机审计。

修复范围：

- 新增 `0016_worker_lease_constraints.sql`，为 `data_sync_tasks`、`backtest_tasks`、`trading_tasks` 和 `notification_outbox` 增加 lease 字段一致性 CHECK。
- task 表要求 lock 三元组 `locked_by` / `locked_until` / `heartbeat_at` 要么全部为空，要么全部非空且状态为 `running`。
- notification outbox 无 `heartbeat_at` 字段，要求 `locked_by` / `locked_until` 要么全部为空，要么全部非空且状态为 `running`。
- 约束使用 `NOT VALID`，避免已有历史脏行阻断迁移；新写入和后续更新仍会被约束。
- 集成测试新增非法 lease 写入拒绝断言，覆盖 partial lock、非 running lock、缺 heartbeat lock 和 outbox 非 running lock。

验证：

- `go test ./internal/store/postgres -run 'TestIntegrationDatabaseConstraintsRejectInvalidDomainValues|TestIntegrationNotificationProviderConstraintsAllowExternalProviders'`
- 本轮通用门禁见最终回复。

失败：

- 无当前硬失败。

剩余风险：

- PostgreSQL migrations 仍为 `scaffold`；本轮只约束 lease 字段一致性，未关闭完整状态流转、父任务删除级联/历史数据验证、数据修复迁移、rollback 策略和生产备份/恢复验证。

### 阶段 8 PostgreSQL worker lease validation 补充

执行时间：2026-06-28

触发问题：

- `0016_worker_lease_constraints.sql` 使用 `NOT VALID`，只保护新写入和后续更新，没有验证历史数据。
- 本地 migration audit 首次运行发现 14 行历史 `trading_tasks` 为 `paused`，`locked_by` / `locked_until` 已清空但 `heartbeat_at` 残留，属于历史半截 lease。

修复范围：

- 新增 `0020_validate_worker_lease_constraints.sql`。
- 迁移先清理 `data_sync_tasks`、`backtest_tasks`、`trading_tasks` 和 `notification_outbox` 中不满足 worker lease 一致性的历史锁字段。
- 随后对 `data_sync_tasks_lease_consistency_check`、`backtest_tasks_lease_consistency_check`、`trading_tasks_lease_consistency_check` 和 `notification_outbox_lease_consistency_check` 执行 `VALIDATE CONSTRAINT`。
- 新增 `scripts/stage8-migration-audit.sh`，校验所有 migration 已记录、worker lease CHECK 已 validated、终态任务有 `finished_at`、lease 字段一致、strategy intent parent 存在且类型匹配、核心事实表不存在 orphan。
- `scripts/stage8-smoke.sh` 在 compose 启动和 `/readyz` 后调用 migration audit，使 Stage 8 smoke 包含迁移历史不变量验证。
- 新增 `TestIntegrationWorkerLeaseConstraintsAreValidated`，防止新库迁移后 CHECK 继续停留在 `NOT VALID`。

验证：

- `go test ./internal/store/postgres -run 'TestIntegrationWorkerLeaseConstraintsAreValidated|TestIntegrationTaskTerminalStatusesRequireFinishedAt|TestIntegrationFailureTransitionsSetFinishedAt' -count=1`
- `docker compose up -d --build migrate api`
- `scripts/stage8-migration-audit.sh`
- 本轮通用门禁见最终回复。

失败：

- 首次 `scripts/stage8-migration-audit.sh` 失败：`trading inconsistent lease rows has 14 violating rows`。已通过 `0020_validate_worker_lease_constraints.sql` 修补历史半截 heartbeat 后重跑通过。

剩余风险：

- PostgreSQL migrations 仍为 `scaffold`；本轮只把 worker lease 历史修补和部分历史不变量审计纳入可重复脚本，未关闭完整状态流转、全量历史数据验证、数据迁移 rollback 策略和生产备份/恢复验证。

### 阶段 8 PostgreSQL task status transition guard 补充

执行时间：2026-06-28

触发问题：

- `data_sync_tasks`、`backtest_tasks` 和 `trading_tasks` 已有状态枚举 CHECK、worker lease CHECK 和终态 `finished_at` CHECK，但数据库仍允许直接把任务从不合理状态跳到另一状态。
- 用户命令路径也可能绕过业务语义，例如对 failed data sync 任务直接 start sync，或对 failed trading task 直接 start，导致任务被非 retry / 非创建路径重开。

修复范围：

- 新增 `0021_task_status_transition_guards.sql`。
- 为 `data_sync_tasks` 增加 `data_sync_tasks_status_transition_guard`：允许 pending/running/paused 的运行态切换，允许 running 进入 succeeded/failed，允许 failed 仅通过 retry 语义回到 pending，禁止 pending 直达 succeeded/failed 和 succeeded/cancelled 重开。
- 为 `backtest_tasks` 增加 `backtest_tasks_status_transition_guard`：允许 pending -> running、running -> pending/succeeded/failed，禁止 failed/succeeded/cancelled 重开。
- 为 `trading_tasks` 增加 `trading_tasks_status_transition_guard`：允许 pending -> running/paused、running -> paused/failed、paused -> running，禁止 failed/succeeded/cancelled 重开。
- `SetSyncEnabled` / `SetRealtimeEnabled` 只允许操作 pending/running/paused 的 data sync 任务；failed 任务必须走 `RetryDataSyncTask`。
- `SetTradingTaskStatus` 只允许 pending/running/paused 范围内的 start/pause/stop 转换；非法状态转换返回 `ErrInvalidState`，API 会映射为 `409 invalid_state`。
- `scripts/stage8-migration-audit.sh` 新增三张任务表状态流转 trigger 启用校验。
- 集成测试修正旧夹具：非 running 任务不再插入 active lease，terminal 任务插入时必须携带 `finished_at`。

验证：

- `TICTICK_TEST_DATABASE_URL='postgresql://tictick:tictick-local-postgres-password@127.0.0.1:55432/tictick_hi?sslmode=disable' go test ./internal/store/postgres -run 'TestIntegration(TaskTerminalStatusesRequireFinishedAt|WorkerLeaseConstraintsAreValidated|FailureTransitionsSetFinishedAt|TaskStatusTransitionGuards|TaskCommandsRejectInvalidStatusTransitions|RetryDataSyncTaskRestoresFailedTask|RetryDataSyncTaskRejectsRunningTask)' -count=1`
- 本轮通用门禁见最终回复。

失败：

- 首次带本地临时 Postgres 的目标集成测试失败：旧测试夹具仍插入 `failed + locked_by`，违反已 validated 的 `data_sync_tasks_lease_consistency_check`。已修正为只有 running 夹具携带 active lease。
- 第二次目标集成测试失败：旧测试夹具插入 failed 任务时缺少 `finished_at`，违反已有 `data_sync_tasks_terminal_finished_at_check`。已修正 terminal 夹具必须携带 `finished_at`。

剩余风险：

- PostgreSQL migrations 仍为 `scaffold`；本轮只约束三张核心任务表的当前业务状态流转。完整统一状态机、notification outbox / notifications 更细状态流转、全量历史数据验证、数据迁移 rollback 策略和生产备份/恢复验证仍未关闭。

### 阶段 8 PostgreSQL strategy intent parent constraints 补充

执行时间：2026-06-28

触发问题：

- `strategy_intents` 同时服务 backtest、paper 和 live 任务，普通 FK 无法直接表达 `task_type='backtest'` 指向 `backtest_tasks`，`task_type IN ('paper','live')` 指向同 type 的 `trading_tasks`。
- 下游 orders / executions / notifications 已通过同 task FK 约束引用 intent，但 intent 本身仍可能被直接写成 orphan 或 task type 错配。

修复范围：

- 新增 `0017_strategy_intent_parent_constraints.sql`，通过 deferrable constraint trigger 校验 `strategy_intents` 新增/更新时的父任务存在性和类型匹配。
- `backtest` intent 必须指向存在的 `backtest_tasks.id`。
- `paper` / `live` intent 必须指向存在且 `type` 匹配的 `trading_tasks(id, type)`。
- 集成测试覆盖合法 backtest intent、缺失 backtest 父任务、缺失 trading 父任务和 trading task type mismatch。
- 本轮不改变父任务删除语义，不处理历史 orphan 数据验证。

验证：

- `go test ./internal/store/postgres -run TestIntegrationDatabaseReferentialConstraintsRejectOrphans -count=1`
- 本轮通用门禁见最终回复。

失败：

- 无当前硬失败。

剩余风险：

- PostgreSQL migrations 仍为 `scaffold`；本轮只约束新写入/更新的 intent 父任务归属，父任务删除防 orphan 保护见下一小节；历史数据验证、完整状态流转、数据修复迁移、rollback 策略和生产备份/恢复验证仍未关闭。

### 阶段 8 PostgreSQL strategy intent parent delete guards 补充

执行时间：2026-06-28

触发问题：

- `0017_strategy_intent_parent_constraints.sql` 只约束 `strategy_intents` 新增/更新，仍允许直接删除已被 intent 引用的 backtest / trading 父任务并留下 orphan intent。
- 项目当前没有公开删除 backtest / trading task 的业务 API，但数据库层仍应拒绝会破坏事实归属的直接写库操作。

修复范围：

- 新增 `0018_strategy_intent_parent_delete_guards.sql`，为 `backtest_tasks` 和 `trading_tasks` 增加 BEFORE DELETE guard trigger。
- 当父任务仍被 `strategy_intents` 引用时，数据库拒绝删除并返回明确约束名。
- 集成测试覆盖 backtest/trading 父任务被引用时删除失败，以及先删除 intent 后父任务可删除。
- 调整约束测试辅助数据清理顺序，先清理下游 facts 和 intents，再删除 trading parent，避免测试库遗留数据。
- 本轮不引入隐式级联删除，保持删除语义保守。

验证：

- `go test ./internal/store/postgres -run 'TestIntegrationStrategyIntentParentDeleteIsRestricted|TestIntegrationDatabaseReferentialConstraintsRejectOrphans' -count=1`
- 本轮通用门禁见最终回复。

失败：

- 无当前硬失败。

剩余风险：

- PostgreSQL migrations 仍为 `scaffold`；本轮只补父任务删除防 orphan 保护，未关闭历史数据验证、完整状态流转、数据修复迁移、rollback 策略和生产备份/恢复验证。

### 阶段 8 PostgreSQL task terminal timestamp constraints 补充

执行时间：2026-06-28

触发问题：

- 当前数据库只限制任务状态取值和 worker lease 字段一致性，仍允许直接写入 `succeeded` / `failed` / `cancelled` 终态但不写 `finished_at`。
- 没有终态完成时间会破坏任务审计、失败恢复判断和后续历史数据校验。

修复范围：

- 新增 `0019_task_terminal_timestamp_constraints.sql`。
- 迁移先将 `data_sync_tasks`、`backtest_tasks`、`trading_tasks` 中已有终态且 `finished_at IS NULL` 的历史行修补为 `COALESCE(finished_at, updated_at, now())`。
- 为三张任务表增加 `*_terminal_finished_at_check`，要求 `succeeded` / `failed` / `cancelled` 必须有 `finished_at`。
- `MarkDataSyncFailed` 和 `MarkTradingTaskFailed` 写入失败状态时同步写 `finished_at = now()`，避免应用层新写入违反约束。
- 新增 `integration_state_constraints_test.go`，覆盖终态缺失 `finished_at` 被拒绝，以及 data sync / trading failure transition 会写入 `finished_at`。

验证：

- `go test ./internal/store/postgres -run 'TestIntegrationTaskTerminalStatusesRequireFinishedAt|TestIntegrationFailureTransitionsSetFinishedAt|TestIntegrationDataSyncPermanentFailureStopsTask|TestIntegrationRetryDataSyncTaskRestoresFailedTask|TestIntegrationDatabaseConstraintsRejectInvalidDomainValues' -count=1`
- 本轮通用门禁见最终回复。

失败：

- 无当前硬失败。

剩余风险：

- PostgreSQL migrations 仍为 `scaffold`；本轮只覆盖任务终态完成时间一致性，未关闭完整状态流转约束、全量历史数据审计、数据修复回放、rollback 策略和生产备份/恢复验证。

### 阶段 8 API error model 补充

执行时间：2026-06-28

触发问题：

- Stage 8 readiness 重审计将 API server 维持为 `scaffold`，其中一个明确 blocker 是错误响应不一致、前端拿不到稳定错误 code，且 500 路径会把内部错误文本直接返回给客户端。
- 前端 `ApiError` 之前主要依赖 HTTP `statusText`，用户可见错误容易退化成泛化的 `Bad Request` / `Conflict`。

修复范围：

- 后端 `writeError` / `writeStoreError` / `writeAuthError` 统一输出 `{code, message, error}`，保留旧 `error` 字段兼容现有调用。
- `data.ErrNotFound`、`data.ErrInvalidState`、auth、CSRF、method、rate limit 和 generic bad request 映射为稳定 code。
- 500 响应统一返回 `code=internal_error`、`message=internal server error`，不再把 repository / driver 细节直接暴露给前端。
- 前端 `ApiError` 新增 `code` 字段，并优先使用服务端 `message`，兼容旧 `{error}` payload。
- 不新增 API 路由，不改变正常成功响应，不在本轮补审计日志或全量错误 taxonomy。

验证：

- `go test ./internal/web/api`
- `pnpm --dir web/frontend exec vitest run src/services/api/client.test.ts`
- `go test ./...`
- `go vet ./...`
- `scripts/quality-gate.sh`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `scripts/stage8-smoke.sh`

本轮 smoke 证据：

- symbol `S81782577037USDT`
- data task `dst_c1cf309192ba00847e2e70e3`
- backtest `bt_e4183ad2b59d06999fd7ec28`
- paper execute `tt_0122cf220fc3f22141d9f1a4`
- paper notify `tt_f16ac0c66bf0e519aa063a69`
- notification channel `stage8-smoke-1782577037`

失败：

- 无当前硬失败。

剩余风险：

- API server 仍为 `scaffold`；还缺完整 request / response mapping、生产级审计边界、跨路由错误 taxonomy、错误响应文档和 OpenAPI / schema 级契约校验。

### 阶段 8 API method contract 补充

执行时间：2026-06-28

触发问题：

- API server 虽然已有结构化错误响应，但不少“资源路径存在、HTTP 方法不对”的请求会落到 route not found，返回 `404 not_found`。
- 调用方无法稳定区分“路径不存在”和“方法不允许”，也拿不到标准 `Allow` header。

修复范围：

- 新增 `writeMethodNotAllowed`，统一输出 `405`、`code=method_not_allowed`、`message=method not allowed` 和 `Allow` header。
- auth、data tasks、candles、strategies、backtests、trading tasks、system health / notifications / audit / accounts / operators 的已知路径按资源方法收敛为 405。
- 未知 API 路径仍保持 404，避免把拼错路径误报为方法错误。
- 新增 `TestAPIMethodNotAllowedContracts` 表驱动测试，覆盖 auth、data、candles、backtest、trading、system 的关键路径，以及未知路径不设置 `Allow`。

验证：

- `go test ./internal/web/api -run 'TestAPIMethodNotAllowedContracts|TestAPIStructuredErrorResponses|TestAPIStructuredInternalErrorDoesNotLeakDetails' -count=1`
- 本轮通用门禁见最终回复。

失败：

- 无当前硬失败。

剩余风险：

- API server 仍为 `scaffold`；本轮只补 HTTP method contract，仍缺完整 request / response schema、OpenAPI / schema 级契约校验、跨路由错误 taxonomy 和生产级审计边界。

### 阶段 8 API schema contract 补充

执行时间：2026-06-28

触发问题：

- Stage 8 readiness 重审计将 API server 维持为 `scaffold`，其中一个明确 blocker 是缺少 request / response schema 和 OpenAPI / schema 级契约。
- 前端 typed client 和后端 Go model 之前没有同一份可请求、可测试的 API contract，路由或字段漂移只能靠页面测试间接发现。

修复范围：

- 新增 `GET /api/system/api-contract`，通过已登录 session 读取基础 OpenAPI 3.1 contract。
- contract 声明 `sessionCookie` 和 `csrfHeader` security scheme，写请求会标出 CSRF header 要求。
- contract 覆盖当前前端实际调用的 auth、data、candles、strategies、backtests、trading、system 路由，声明 path/query 参数、request body、success response 和稳定错误 response。
- response / request schema 由 Go struct 的 JSON tag 反射生成，覆盖 `data` 和 `strategy` 核心模型，避免手写字段列表和实现漂移。
- secret 边界纳入测试：`ExchangeAccount` response schema 不暴露 `apiKey` / `apiSecret`，`CreateExchangeAccount` request schema 仍声明必要密钥输入，`OperatorSession` response schema 不暴露 `tokenHash`。
- 新增 `TestAPIContractRouteExposesOpenAPIContract`、`TestAPIContractCoversCurrentFrontendRoutes`、`TestAPIContractDeclaresWriteSecurityAndErrorShape`、`TestAPIContractSchemasProtectSecretBoundary`。
- `TestAPIMethodNotAllowedContracts` 纳入 `/api/system/api-contract` 的 405 / `Allow` contract。

验证：

- `go test ./internal/web/api`
- 本轮通用门禁见最终回复。

失败：

- 无当前硬失败。

剩余风险：

- API server 仍为 `scaffold`；本轮只补基础 OpenAPI/schema contract，还没有前端 TypeScript 类型自动生成 / 字段级 diff gate、外部 OpenAPI validator、全量错误 taxonomy、生产级审计边界或 RBAC。

### 阶段 8 API contract drift gate 补充

执行时间：2026-06-28

触发问题：

- `/api/system/api-contract` 已存在，但如果前端 service 新增或修改 `apiClient.get/post/delete` 调用而后端 contract 没有同步声明，之前没有硬门禁阻止漂移。
- 前端写请求依赖 CSRF header，contract 若漏声明 `csrfHeader`，后续 API 使用者会拿到不完整安全契约。

修复范围：

- 新增 `TestFrontendAPIRoutesAreCoveredByContract`，扫描 `web/frontend/src/services/api` 非测试 service 文件，提取 `apiClient.get/post/delete` 调用。
- 测试会把前端路径补齐 `/api` 前缀、去除 query string、把模板字符串动态段归一化后，与后端 OpenAPI contract 的 method/path 做段级匹配。
- 需要 session 的前端写请求必须在 contract security 中声明 `csrfHeader`；公开登录 POST 不强制 CSRF。
- 新增 `scripts/check-api-contract-drift.sh`，运行 `TestFrontendAPI*` drift 测试集合。
- `scripts/quality-gate.sh` 新增硬检查 `api contract drift`，漂移会让质量门禁失败。

验证：

- `scripts/check-api-contract-drift.sh`
- `go test ./internal/web/api -count=1`
- 本轮通用门禁见最终回复。

失败：

- 初版测试误把公开登录 POST 也要求 CSRF；已收敛为“需要 session 的写请求必须声明 CSRF”。
- 初版路径匹配不能处理前端动态 `action` 对应后端显式 `/enable`、`/disable`；已改为路径段兼容匹配。

剩余风险：

- API server 仍为 `scaffold`；本轮只防止前端 API route 与后端 contract 漂移，字段级 contract drift 已在后续补充中覆盖；仍未引入 TypeScript 类型自动生成或外部 OpenAPI schema validator。

### 阶段 8 API schema field drift gate 补充

执行时间：2026-06-28

触发问题：

- route drift gate 只能证明前端调用路径存在于后端 contract，不能证明前端 request / response DTO 字段仍与 OpenAPI contract 对齐。
- API server 仍被 Stage 8 readiness 标记为 `scaffold`，其中一个明确风险是字段级 schema 漂移没有硬门禁。

修复范围：

- 新增 `api_schema_drift_test.go`，解析 `web/frontend/src/types/app.ts` 和 `web/frontend/src/services/api/data.ts` 中简单对象型 TypeScript DTO。
- `TestFrontendAPIRequestTypesMatchContractSchemas` 校验前端 request DTO 与后端 OpenAPI schema 字段集合和 required / optional 完全一致，覆盖 login、data sync、backtest、trading、notification channel、exchange account、operator 创建请求。
- `TestFrontendAPIResponseTypesMatchContractFields` 校验核心前端 response DTO 字段集合与后端 contract schema 一致，覆盖 data sync、candle metadata、backtest、trading、notification、system health、audit、strategy schema 等模型。
- `TestFrontendAPIAdapterResponseFieldsExistInContract` 校验前端 adapter 内部 response projection 不读取 contract 中不存在的字段。
- `TestFrontendAPICandleQueryMatchesContractParameters` 校验前端 candle query 字段和 `/api/candles` query parameters 的字段集合与必填性一致。
- `scripts/check-api-contract-drift.sh` 从只跑 route drift 扩大为运行全部 `TestFrontendAPI*` drift 测试，因此 `scripts/quality-gate.sh` 会硬性执行字段级检查。

验证：

- `go test ./internal/web/api -run 'TestFrontendAPI(Request|Response|Adapter|CandleQuery)' -count=1`
- `scripts/check-api-contract-drift.sh`
- `go test ./internal/web/api -count=1`
- 本轮通用门禁见最终回复。

失败：

- 无当前硬失败。

剩余风险：

- API server 仍为 `scaffold`；本轮字段级 drift gate 依赖当前项目简单 TypeScript type 语法解析，不是通用 TypeScript compiler AST，也没有自动生成前端类型或引入外部 OpenAPI validator。

## 6. 保留 / 返工 / 删除 / 延后

保留：

- 单二进制多子命令方向。
- Docker Compose 运行形态。
- Vue 3 + Naive UI + Pinia + i18n + lightweight-charts。
- PostgreSQL 作为唯一数据库和协调中心。
- 现有 migration 作为草稿基线。
- 现有研究页骨架。

返工：

- API server 文件组织。
- CandleProvider。
- worker lease。
- exchange account 密钥处理。
- backtest executor。
- trading runner。
- system health。

删除或替换：

- 阶段 6 前用 digest 冒充 encrypted secret 的实现已替换为本地 AES-GCM 边界。
- 回测中的交易事实 `float64`。
- 只返回裸 candles 的 `/api/candles` 语义。
- 空泛的 “external worker health”。

延后：

- 实盘真实下单。
- 通知真实第三方 provider 生产启用边界。
- 概览页深度指标。
- 聚合 K 线持久化缓存。
- tick / trade 级数据。

## 7. 当前不能做的声明

在上述 P0 关闭前，禁止对外声明：

- “系统已实现”
- “demo 已完成”
- “实盘可用”
- “回测可信”
- “数据同步稳定”
- “质量已达标”

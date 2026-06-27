# Quality Audit

审计日期：2026-06-27

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
| Docker Compose | demo | 保留 | 运行形态对，但还缺完整 smoke gate |
| PostgreSQL migrations | scaffold | 保留后加强 | 表基本有了，约束、外键、状态约束不足 |
| API server | scaffold | 保留后加强 | 已按领域拆分，`/api/candles` 已返回 metadata，回测 / 交易创建已复用策略 schema 校验，系统写请求已有 CSRF 检查；仍缺统一 request / response mapping 和更强错误边界 |
| 登录会话 | demo | 保留后加强 | HttpOnly session cookie、CSRF double-submit 写保护、登录失败节流已进入 API 边界；仍缺持久化限流、会话管理、审计和密码策略 |
| 数据同步 worker | demo | 保留后加强 | 能 claim、拉取、upsert 1m K 线并恢复游标，运行中会持续刷新 heartbeat / locked_until，heartbeat 丢失后会停止保存结果；临时市场数据错误记录为 retry 并释放 lease，永久失败会停用 sync / realtime 期望；用户 stop sync / realtime 和 runner 上下文取消会释放 active lease；release / fail / pause 清锁语义已收敛到共享 helper；仍缺完整统一状态机和容器级 SIGTERM smoke |
| CandleProvider | demo | 保留后加强 | 已统一 native / 1m 聚合、来源和缺口 metadata，查询 limit 已有显式默认/上限，PostgreSQL 集成测试覆盖基础聚合、缺口、默认最新窗口查询、超大 limit clamp 和 runner 侧闭合信号过滤；仍缺大范围性能压测、分页/游标和更多异常数据边界 |
| Binance / OKX K 线 adapter | demo | 保留后加强 | 能拉 K 线，Binance 支持多 base URL fallback，EOF/超时/429/5xx/OKX 50011 已分类为临时错误并由 sync runner 有限重试，错误摘要不泄露完整请求 URL；仍缺全局限流、真实网络韧性和更完整交易所业务码分类 |
| 研究页 | demo | 保留后打磨 | 列表在上、图表在下，任务表格错误列和图表高度已有前端约束；图表面板已用固定 grid 行和面板边界 clamp 切断高度反馈，显示 source / health / base interval；但交易对仍硬编码、图表研究能力仍薄 |
| 策略 registry / runtime | demo | 保留后加强 | 已有策略 schema 校验、默认参数规范化、order / notification intent 和边界门禁，仍缺策略沙箱、参数版本迁移和更多真实策略 |
| 回测 | demo | 保留后加强 | 已通过 CandleProvider 执行、`minute_replay` 以 `1m` 推进，策略输入前会丢弃未闭合 K 线，intent / order / result 落库，详情页展示 intent 和买卖点；撮合模型、费用/滑点曲线、指标体系仍不可信 |
| 交易 runner | demo | 保留后加强 | 已通过 CandleProvider 取 K 线，策略输入前会丢弃未闭合 K 线，paper executor 落库 intent / order / execution / position / notification，running task claim 已按 `updated_at` 轮转避免旧任务长期占用队列，用户 pause 和 runner 上下文取消会释放 active lease，live execute 已禁用；仍缺可信风控、真实第三方通知 provider、完整统一 worker lease 和实盘安全边界 |
| 实盘安全 | demo | 保留后加强 | 新建交易所账号凭据使用 `ENCRYPTION_KEY` + AES-GCM 加密保存，列表/API 不返回明文，live 任务创建校验账号启用和凭据状态；真实 testnet/sandbox live executor、幂等提交和生产密钥管理仍未完成 |
| 通知 | demo | 保留后加强 | NotificationIntent 已进入 notification outbox，`hi notify` 支持 local / webhook-demo provider、失败重试和系统页 retry，delivered / failed / retry / runner 上下文取消会通过共享 lease helper 释放 outbox lock；真实第三方 provider、通道更新/删除、完整统一 worker lease 仍未完成 |
| 前端基础设施 | scaffold | 保留后加强 | Vue/Naive/Pinia/i18n/主题骨架存在，策略任务表单已由 schema 驱动并校验参数，整体业务体验仍需继续打磨 |
| 概览页 | scaffold | 保留后加强 | 有 scaffold 状态面板和基础健康信息，不是完整概览 |
| 系统管理 / 运维健康 | demo | 保留后加强 | 操作台账号可创建和启停，运维健康页/API 展示数据库、api、worker count、heartbeat 和 locked_until；仍缺 RBAC、审计、完整 session 管理和生产监控 |
| 质量门禁 | scaffold | 保留后加强 | 阶段 0 硬门禁和策略边界检查已通过，live executor/testnet、统一 worker lease 和生产级登录安全作为后续风险审计保留 |

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
- `scripts/quality-gate.sh` 已建立，并能稳定执行 file size、trading float、阶段 0 scaffold marker 检查。

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
- 仍缺较大时间范围性能边界、分页/游标和更多异常数据边界；闭合周期信号已有 runner 侧基础过滤，未闭合 K 线不再进入策略输入。

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
- data sync、backtest、trading、notification outbox 的 claim 过期条件和 claim 锁字段写入已收敛到 `internal/store/postgres/lease.go` 共享 helper。
- data sync、backtest、trading、notification runner 在父上下文取消时会释放当前 active lease，不再把 shutdown 误记为任务失败；backtest 会从 `running` 复位为 `pending`，避免清锁后无法再次 claim。
- claim 的领域筛选、排序和状态切换仍分散在各自 store 方法中，还不是完整统一状态机。
- 停止状态机不完整。
- 容器级 SIGTERM smoke 尚未证明任务能收尾。

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
- 仍缺容器级 SIGTERM 后数据库断言和真实交易所网络稳定性证明。

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
- Notification provider/outbox/retry 已进入阶段 5 demo，真实第三方 provider 仍未接入。

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
- notification provider 抽象明确，阶段 5 只启用安全的本地 / webhook-like demo provider，不接入真实敏感凭据。
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
- provider 抽象已建立，阶段 5 只启用 `local` / `webhook-demo` demo provider，不访问真实第三方网络。
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

失败：

- 无当前硬失败。

警告：

- 真实邮件、Telegram、飞书 provider 未接入；阶段 5 只声明 demo。
- 通知通道只有创建和读取，没有更新、删除、启停编辑和凭据脱敏模型。
- `hi notify` 已有 outbox claim/lock，但仍未抽取全系统统一 worker lease 包。
- 通知 provider 未实现生产级限流、熔断、模板、审计签名或外部回执。
- Vite 构建仍提示主 chunk 超过 500 kB，后续需要做路由级 code split。

后续风险审计：

- 交易所账号密钥 digest 风险已在阶段 6 切片关闭到 `demo`；历史非 AES-GCM 行标记为 `legacy`。
- live executor 仍禁用，testnet/sandbox、幂等提交、真实交易所提交和生产密钥管理仍未建立。
- 登录会话仍缺 CSRF、防暴力破解和更完整 session 管理。

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
- 全系统 worker lease、登录会话 CSRF / 防暴力破解仍未完成。

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
- 操作台账号启停没有 RBAC、自保护规则、审计记录或强密码策略。
- 运维健康能观察现有 task lease 字段，但全系统统一 worker lease、持续 heartbeat loop 和优雅停止状态机仍未完成。
- Vite 构建仍提示主 chunk 超过 500 kB，后续需要做路由级 code split。

阶段 7 结论：

- 运维健康和操作台账号达到 `demo` 检查点。
- 项目整体仍为 `scaffold`，不能称为 usable、production-safe 或完成。

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
- 真实第三方通知 provider。

### 阶段 8 当前验收快照：全链路 smoke gate

执行时间：2026-06-27

新增验收入口：

- `scripts/stage8-smoke.sh`

通过：

- `scripts/stage8-smoke.sh`
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

- symbol：`S81782558204USDT`
- data task：`dst_9cb68c4565588802c954e287`
- backtest：`bt_b85822a7a5ec1f6e0cf8764e`
- paper execute：`tt_b0dc4463cf489d480a00027a`
- paper notify：`tt_b4f1e26746e22d2d74997b10`
- notification channel：`stage8-smoke-1782558204`

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

已收敛的 worker 取消释放 lease 路径：

- `internal/workerlease` 新增 shutdown 判定和不继承父取消的 release context，避免用已取消的请求上下文做收尾写库。
- data sync、backtest、trading、notification runner 在父上下文取消时释放当前 active lease，并跳过 MarkFailed / MarkNotificationFailed。
- backtest shutdown release 会把仍处于 `running` 的任务复位为 `pending`，否则 backtest claim 只选 `pending` 会导致任务卡死。
- 单元测试覆盖 sync/backtest/trading/notification 的 shutdown release、不保存部分结果、不误标失败；同时覆盖普通 `context.Canceled` 业务错误不会被误判为进程 shutdown。

已收敛的交易所 K 线错误边界：

- `internal/exchange` 提供共享 HTTP status / transport error 分类和 endpoint 错误摘要，避免 adapter 泄露完整请求路径和 query 参数。
- Binance / OKX adapter 统一将 EOF、deadline、transport error、HTTP 429、HTTP 5xx 识别为临时错误；OKX 业务码 `50011` 识别为临时限流错误，`51001` 等配置 / symbol 错误不重试。
- `hi sync` 继续通过 `SYNC_FETCH_RETRIES` / `SYNC_RETRY_DELAY` 对临时 market data 错误做有限重试；`last_error` 保持规范化和 500 rune 截断。
- 单元测试覆盖 URL 摘要脱敏、临时 / 永久错误分类、Binance fallback、OKX rate-limit 码和 sync runner 临时错误重试。

失败：

- 无当前硬失败。

警告：

- Stage 8 当前只建立了可重复全链路 smoke gate；还没有完成所有模块等级重审计，不能把整体升级为 `usable`。
- 全链路 smoke 使用确定性 seed K 线，不依赖真实交易所网络；它证明内部链路，不证明 Binance / OKX 外部稳定性。
- 交易所 adapter 仍缺全局限流器、代理 / 地域网络策略、更多 OKX / Binance 业务错误码审计和真实网络压测。
- worker claim 的共享字段和过期谓词已收敛，runner 级 shutdown release 已有单元证明，但领域候选选择仍未抽取为完整统一状态机，容器级 SIGTERM 数据库断言仍未补齐。
- 回测撮合、paper position PnL、真实通知 provider、实盘 testnet/sandbox 和生产级会话/RBAC/审计仍是后续风险。
- Vite 构建仍提示主 chunk 超过 500 kB，后续需要做路由级 code split。

阶段 8 当前结论：

- 整体全链路 smoke gate 达到 `demo` 证据增强。
- 项目整体仍为 `scaffold`，不能称为 usable、production-safe 或完成。

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
- 通知真实第三方 provider。
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

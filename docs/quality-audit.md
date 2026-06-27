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
| API server | scaffold | 保留后加强 | 已按领域拆分，`/api/candles` 已返回 metadata，回测 / 交易创建已复用策略 schema 校验，仍缺统一 request / response mapping 和更强错误边界 |
| 登录会话 | scaffold | 返工加强 | 有 cookie session，但 CSRF、防暴力破解、会话审计不足 |
| 数据同步 worker | scaffold | 返工加强 | 能 claim、拉取、upsert 1m K 线并恢复游标，但没有真正 heartbeat loop、优雅停止状态机 |
| CandleProvider | demo | 保留后加强 | 已统一 native / 1m 聚合、来源和缺口 metadata，仍缺 PostgreSQL 集成测试、性能边界和闭合信号硬化 |
| Binance / OKX K 线 adapter | scaffold | 保留后加强 | 能拉 K 线，但 symbol 规范、限流、错误分类不完整 |
| 研究页 | demo | 保留后打磨 | 列表在上、图表在下，显示 source / health / base interval，但交易对仍硬编码、图表研究能力仍薄 |
| 策略 registry / runtime | demo | 保留后加强 | 已有策略 schema 校验、默认参数规范化、order / notification intent 和边界门禁，仍缺策略沙箱、参数版本迁移和更多真实策略 |
| 回测 | demo | 保留后加强 | 已通过 CandleProvider 执行、`minute_replay` 以 `1m` 推进、intent / order / result 落库，详情页展示 intent 和买卖点；撮合模型、费用/滑点曲线、指标体系仍不可信 |
| 交易 runner | demo | 保留后加强 | 已通过 CandleProvider 取 K 线，paper executor 落库 intent / order / execution / position / notification，live execute 已禁用；仍缺可信风控、真实通知 provider、统一 worker lease 和实盘安全边界 |
| 实盘安全 | below-scaffold | 延后 | 密钥字段名叫 encrypted，实际是 digest，不是真加密也不能解密 |
| 通知 | scaffold | 返工 | 有通知记录雏形，但 provider/outbox/retry 不完整 |
| 前端基础设施 | scaffold | 保留后加强 | Vue/Naive/Pinia/i18n/主题骨架存在，策略任务表单已由 schema 驱动并校验参数，整体业务体验仍需继续打磨 |
| 概览页 | scaffold | 保留后加强 | 有 scaffold 状态面板和基础健康信息，不是完整概览 |
| 质量门禁 | scaffold | 保留后加强 | 阶段 0 硬门禁和策略边界检查已通过，实盘安全和 live executor 作为后续风险审计保留 |

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
- 实盘密钥和 live executor 风险继续保留为 scaffold / below-scaffold，不在阶段 0 冒充关闭。

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

- 交易所账号密钥仍然使用 `secretDigest`，不是真加密。
- live executor 仍禁用，实盘安全边界未建立。

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

- `internal/store/postgres/system_store.go` 仍使用 `secretDigest` 处理交易所账号密钥。
- live executor 仍禁用，实盘安全边界未建立。

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
- 仍缺 PostgreSQL 集成测试、较大时间范围性能边界、闭合周期信号的后续硬化。

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
- 运行过程中没有周期性 heartbeat。
- 停止状态机不完整。
- 容器退出时没有证明任务能收尾。

关闭条件：

- 提取统一 lease 包。
- 支持 claim、heartbeat、release、fail、pause。
- worker 运行长任务时持续刷新 `locked_until`。
- heartbeat 失败达到阈值后停止外部副作用。
- 数据同步、回测、交易 worker 都走统一实现。

### P0：实盘密钥不能用 digest 冒充加密

现状问题：

- `encrypted_api_key` / `encrypted_api_secret` 存的是 SHA-256 digest。
- digest 不能解密，未来 live executor 无法拿到密钥。
- 字段名会误导后续实现。

关闭条件：

- 定义真正加密边界。
- `ENCRYPTION_KEY` 来源明确。
- 使用成熟 AEAD 加密。
- 列表和日志永不展示完整密钥。
- 禁用账号后不能提交新实盘订单。

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
- 实盘密钥和 live executor 风险继续保留为后续阶段风险。

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

- `internal/store/postgres/system_store.go` 仍使用 `secretDigest` 处理交易所账号密钥。
- live executor 仍禁用，实盘安全边界未建立。
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
- resize 通过 `requestAnimationFrame` 合并，并在尺寸未变化时跳过。
- 组件卸载时断开 `ResizeObserver`、窗口 resize 事件和待执行 animation frame。
- 研究页图表区域新增固定 flex body，工具栏之外的剩余空间才是 K 线图表高度来源。

验证：

- `cd web/frontend && pnpm run typecheck`
- `cd web/frontend && pnpm run test`
- `cd web/frontend && pnpm run build`
- `git diff --check`
- `scripts/quality-gate.sh`
- `docker compose up -d --build api`
- `curl -fsS http://127.0.0.1:8080/readyz`
- Headless Chrome 桌面 `2048x1024` 打开 `/research`，30 次采样 `scrollHeight=1099`、`panelHeight=760`、`bodyHeight=683`、`chartHeight=683`、`canvasHeight=683`，无增长。
- Headless Chrome 桌面 `2048x1024` 打开 `/research?exchange=binance&symbol=BTCUSDT&interval=5m`，30 次采样 `scrollHeight=1099`、`panelHeight=760`、`bodyHeight=683`、`chartHeight=683`、`canvasHeight=683`，无增长。
- Headless Chrome 移动 `390x844` 打开 `/research`，30 次采样 `scrollHeight=1058`、`panelHeight=624`、`bodyHeight=457`、`chartHeight=457`、`canvasHeight=457`，无增长。
- 浏览器采样未捕获 `ResizeObserver`、JS exception 或 console error。

失败：

- 无硬失败。

警告：

- Vite 构建仍提示主 chunk 超过 500 kB，后续需要做路由级 code split。

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

- `internal/store/postgres/system_store.go` 仍使用 `secretDigest` 处理交易所账号密钥。
- live executor 仍禁用，实盘安全边界未建立。
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

- `internal/store/postgres/system_store.go` 仍使用 `secretDigest` 处理交易所账号密钥。
- live executor 仍禁用，实盘安全边界未建立。
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
- Notification 仍只是本地记录，没有 provider/outbox/retry。

后续风险审计：

- `internal/store/postgres/system_store.go` 仍使用 `secretDigest` 处理交易所账号密钥。
- live executor 仍禁用，实盘安全边界未建立。
- 统一 worker lease 包仍未抽取，当前只是 trading/backtest 局部 heartbeat。

阶段 4 结论：

- 模拟盘 paper 链路达到 `demo` 检查点。
- 项目整体仍为 `scaffold`，不能称为 usable、production-safe 或完成。

## 4. 下一条可推进切片

下一步只推进：

```text
阶段 5：通知 demo 链路
```

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

## 5. 保留 / 返工 / 删除 / 延后

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

- 用 digest 冒充 encrypted secret 的实现。
- 回测中的交易事实 `float64`。
- 只返回裸 candles 的 `/api/candles` 语义。
- 空泛的 “external worker health”。

延后：

- 实盘真实下单。
- 通知真实 provider。
- 概览页深度指标。
- 聚合 K 线持久化缓存。
- tick / trade 级数据。

## 6. 当前不能做的声明

在上述 P0 关闭前，禁止对外声明：

- “系统已实现”
- “demo 已完成”
- “实盘可用”
- “回测可信”
- “数据同步稳定”
- “质量已达标”

# tictick-hi 实施计划

## 0. 阅读约定与硬约束

本文档不是愿景稿，而是 `tictick-hi` 第一版实现的工程契约。后续代码实现、重构和功能扩展都必须以本文档为边界。如果实现中发现本文档不够精确，先更新计划并确认边界，再写代码。

术语约定：

- `MUST` 表示必须满足，不满足不能合并。
- `SHOULD` 表示默认必须满足，只有明确记录原因才能例外。
- `MUST NOT` 表示禁止实现。
- “任务”特指 `strategy_tasks` 中的一条可运行策略任务。
- “运行”特指 `task_runs` 中由 daemon 实际启动的一次 runtime 实例。

不可妥协的硬约束：

- MUST 保持单体 Go 项目，不拆微服务。
- MUST 使用 PostgreSQL，不能引入第二套生产数据库。
- MUST 提供 Docker / Docker Compose 单机部署能力，包含 app、PostgreSQL、migration、healthcheck 和示例配置。
- MUST 第一版只启用 Binance 和 OKX；交易所层必须通过小接口和 adapter registry 保持可扩展，新增交易所需要新增 adapter、测试和明确设计记录，但不能要求重写核心交易链路。
- MUST 让 backtest、paper、live 共享同一套 Strategy、Risk、Order、Fill、Portfolio 语义。
- MUST 让每笔订单、成交、仓位、事件都能追溯到 `task_id`。
- MUST 让 live task 可被单独启动、暂停、停止，不能实现成一个全局单脚本。
- MUST 让控制命令只改变数据库期望状态，不能在 CLI 里直接启动 live 下单循环。
- MUST 让策略只返回 `OrderIntent` / `NotificationIntent` 等意图，不能直接访问交易所、数据库、通知服务或 portfolio writer。
- MUST 支持系统级通知和策略级通知路由；通知发送必须可追溯、可去重、可关闭。
- MUST 让操作台保持简单，尤其数据同步页只能展示同步、检查、修复的必要信息。
- MUST 让操作台具备基础登录鉴权、会话安全、CSRF 防护和操作审计，不能裸露管理接口。
- MUST 让实盘相关危险操作具备防盗护栏：默认关闭交易、默认开启 kill switch、二次确认、必要时重新验证当前密码、API 密钥不入库不展示。
- MUST NOT 为了“以后可能用到”提前引入复杂抽象、插件系统、权限系统或数据治理模型。
- MUST NOT 在核心包中 import 具体交易所适配器、PostgreSQL 实现、HTTP handler 或 CLI。
- MUST NOT 通过全局变量保存任务状态、仓位、订单序列或 exchange client。
- MUST NOT 在 live 未知执行结果时直接标记失败；必须记录 uncertain / pending 状态并等待 reconcile。
- MUST 对价格、数量、金额、手续费、PnL 使用明确类型和资产单位；不该使用 `float64` 的交易事实不能使用 `float64`，也不能让语义不明的 decimal 在核心链路中流动。

## 1. 项目定位

`tictick-hi` 是一个小而美的多交易所交易机器人系统。它的目标不是复刻 `tictick-lite` 或 `tictick-pro` 的平台化复杂度，而是在 `tictick-bot` 的清爽运行模型上补齐多交易所、K 线数据同步和基于同步数据的回测能力。

一句话定位：

```text
tictick-hi = tictick-bot 的多交易所版本 + K 线数据同步 + 数据库回测
```

核心目标：

- 使用 Go 实现。
- 使用 PostgreSQL 作为唯一持久化数据库。
- 第一版只启用 Binance 和 OKX；后续可按 adapter 契约扩展更多交易所。
- 支持同步多个交易所、多个交易对的 1m 历史 K 线。
- 支持策略和回测使用多个周期，周期数据由 1m 内部聚合生成。
- 第一版支持以 1m closed candle 作为最小时间钟触发策略执行；TimeClock 抽象不能把 1m 写死为架构上限，未来接入 tick / trade 级数据时可以使用更细的基础事件推进。
- 支持简单数据完整性检测与修复。
- 支持基于 PostgreSQL 中已同步 K 线做回测。
- 支持多个模拟盘任务同时运行。
- 支持多个实盘任务同时运行，并能按任务启动、暂停、停止。
- 支持系统级通知和策略级信号通知。
- 保持代码结构清爽、边界明确、工程实践可靠。

## 2. 非目标

第一版明确不做：

- 不做微服务。
- 不做多租户。
- 不做策略插件市场。
- 不做复杂权限系统、MFA、审批流；第一版只做单管理员基础登录鉴权和操作防护。
- 不做重型 Web 控制台。
- 不做复杂数据治理中心。
- 不做多语言策略。
- 不做高频交易、低延迟撮合或复杂路由。
- 第一版不做 tick / trade 级行情同步、存储和逐笔成交回测；但 TimeClock 抽象不能把 1m 固化成长期上限。
- 第一版不做 Kubernetes、Helm、服务网格或复杂集群部署；Docker Compose 是标准部署路径。
- 不做通用研究平台。
- 不做过度抽象的交易所能力矩阵；交易所扩展点保持小而明确。
- 不做多套订单状态机、多套账本或多套风控。

所有功能必须服务于一条清楚的交易链路。

## 3. 核心形态

系统由一个常驻 daemon 管理多个任务：

```text
PostgreSQL
  -> hi daemon
      -> TaskSupervisor
          -> PaperRunner(task A)
          -> PaperRunner(task B)
          -> LiveRunner(task C)
          -> LiveRunner(task D)
```

任务示例：

```text
task A: binance-main + BTCUSDT  + ema-cross + paper
task B: okx-main     + ETH-USDT + grid      + paper
task C: binance-main + SOLUSDT  + trend     + live
task D: okx-main     + BTC-USDT + ema-cross + live
```

每个任务独立拥有：

- `task_id`
- exchange account
- strategy
- symbol
- interval
- mode: `backtest` / `paper` / `live`
- desired status: `running` / `paused` / `stopped` / `dry_run`
- risk limits
- runtime state

控制命令只修改数据库中的期望状态；daemon 观察状态变化并负责启动、暂停或停止 runner。

## 4. 核心交易链路

每个任务内部保持同一条短链路：

```text
Candle
  -> Strategy
  -> StrategyDecision
      -> OrderIntent
      -> NotificationIntent
  -> Risk
  -> Executor
  -> ExecutionReport / Fill
  -> Portfolio
  -> NotificationRouter
  -> Recorder
```

边界规则：

- 策略只产生 `OrderIntent` 和 `NotificationIntent`，不能直接下单、不能直接发通知。
- 策略不能写数据库。
- 策略不能修改仓位。
- 风控只做准入判断，不修改订单、不修改仓位。
- 执行层只负责订单提交、撤单、状态查询和交易所回报。
- 通知层只负责路由和投递，不参与交易决策。
- Portfolio 是仓位、成本、手续费、PnL 的唯一事实源。
- Backtest、Paper、Live 尽量共用订单、成交和账本语义。
- 每笔订单必须能追溯到唯一 `task_id` 和 `strategy_id`。

## 5. 推荐目录结构

```text
tictick-hi/
  cmd/hi/                  CLI 入口
  cmd/hid/                 daemon 入口，也可后续合并到 hi daemon
  internal/core/           clock、id、component、基础错误
  internal/config/         配置加载与校验
  internal/auth/           Web 登录、会话、CSRF、密码校验
  internal/model/          Candle、Order、Fill、Position、Portfolio
  internal/exchange/       交易所接口定义
  internal/adapter/binance Binance 适配器
  internal/adapter/okx     OKX 适配器
  internal/data/           K 线同步、检查、修复、查询、聚合
  internal/strategy/       策略接口、注册表、内置策略
  internal/risk/           全局、账户、任务、订单风控
  internal/execution/      paper/live executor、订单状态语义
  internal/portfolio/      账本、仓位、PnL
  internal/notification/   通知事件、路由、投递 provider
  internal/backtest/       回测 runner
  internal/runtime/        task runner、supervisor、daemon loop
  internal/store/          PostgreSQL repository 接口
  internal/store/postgres  PostgreSQL 实现与 migrations
  internal/web/            极简本机操作台，后置实现
  web/frontend/            Vue 3 + Vite + TypeScript 操作台
  docker/                  Docker entrypoint、healthcheck、部署辅助配置
  deploy/compose/          docker-compose 示例和生产覆盖配置
  Dockerfile               多阶段构建 app 镜像
  .dockerignore
  .env.example             Docker/本地部署环境变量示例，不含真实密钥
  docs/                    设计和实施文档
  scripts/                 smoke、test、migration 辅助脚本
```

依赖方向：

```text
cmd
  -> config / runtime / store / adapter
runtime
  -> model / strategy / risk / execution / portfolio / store interfaces
adapter
  -> exchange / model
store/postgres
  -> store interfaces / model
model
  -> standard library only
```

核心包不能反向依赖 `cmd`、`web`、具体数据库实现或具体交易所实现。

## 6. PostgreSQL 数据模型

第一版保留必要事实，不做复杂数据治理。

主要表：

```text
exchange_catalog
exchange_accounts
instrument_rules
market_candles
data_sync_jobs
data_quality_reports
strategies
strategy_tasks
task_runs
orders
fills
positions
portfolio_snapshots
events
settings
notification_channels
notification_routes
notification_events
notification_deliveries
web_users
web_sessions
```

### 6.0 exchange_catalog

交易所目录表，用来避免在所有业务表里硬编码交易所枚举。

第一版 seed：

- `binance`
- `okx`

字段：

- `id`: 稳定小写标识，例如 `binance`、`okx`
- `name`
- `enabled`
- `created_at`
- `updated_at`

规则：

- 第一版产品能力只暴露 Binance 和 OKX。
- 业务表的 `exchange` 字段 SHOULD 引用 `exchange_catalog(id)`，不要到处写 `CHECK (exchange IN (...))`。
- 新增交易所必须新增 adapter、symbol normalization、instrument rules 同步、数据同步测试、paper/live fake 测试和 live safety 检查。
- 不允许为了未来交易所做巨大能力矩阵；暂时不支持的能力返回明确 `ErrUnsupported`。

### 6.1 exchange_accounts

记录交易所账号配置：

- `id`
- `exchange`: 第一版为 `binance` / `okx`，必须存在于 `exchange_catalog`
- `market`: 第一版默认 `spot`
- `environment`: `testnet` / `production` / `sandbox`
- `api_key_env`
- `secret_key_env`
- `passphrase_env`: OKX 使用
- `trade_enabled`
- `created_at`
- `updated_at`

密钥不入库，只保存环境变量名。

### 6.1.1 instrument_rules

交易所交易规则表，避免价格、数量和最小名义价值在代码中散落：

- `exchange`
- `market`
- `symbol`
- `base_asset`
- `quote_asset`
- `tick_size`
- `step_size`
- `min_quantity`
- `min_notional`
- `notional_asset`
- `updated_at`

用途：

- live 下单前按交易所规则舍入和校验。
- backtest / paper 记录本次运行使用的规则。
- 风控按舍入后的订单重新计算 notional。

### 6.2 market_candles

历史 K 线事实表。第一版只保存交易所原始 1m K 线：

- `exchange`
- `market`
- `symbol`
- `interval`: 第一版必须为 `1m`
- `open_time`
- `open`
- `high`
- `low`
- `close`
- `volume`
- `complete`
- `source`: 第一版固定为 `exchange`
- `ingested_at`

唯一键：

```text
(exchange, market, symbol, interval, open_time)
```

查询索引：

```text
(exchange, market, symbol, interval, open_time)
```

禁止事项：

- 禁止把内部聚合出的 5m/15m/1h/4h/1d 写入 `market_candles`。
- 禁止在同一张表混用交易所原生高周期和内部聚合高周期。
- 禁止回测直接读取交易所原生高周期绕过内部聚合器。
- 禁止把 tick / trade 级数据塞进 `market_candles`。未来如果支持更细粒度行情，必须新增 `market_trades`、`market_ticks` 或等价事实表，并更新 TimeClock 设计。

### 6.3 data_sync_jobs

只保存同步任务的简要状态，不做复杂流程编排：

- `id`
- `exchange`
- `market`
- `symbol`
- `interval`
- `start_time`
- `end_time`
- `status`: `queued` / `running` / `succeeded` / `failed`
- `error`
- `started_at`
- `finished_at`

### 6.4 data_quality_reports

保存最近一次检查结果：

- `id`
- `exchange`
- `market`
- `symbol`
- `interval`
- `start_time`
- `end_time`
- `status`: `healthy` / `missing_gaps` / `invalid_candles`
- `missing_count`
- `invalid_count`
- `checked_at`
- `summary_json`

操作台只展示 status 和少量数字。

### 6.5 strategies

策略定义：

- `id`
- `name`
- `version`
- `kind`
- `config_json`
- `created_at`
- `updated_at`

### 6.6 strategy_tasks

可运行任务：

- `id`
- `strategy_id`
- `exchange_account_id`
- `exchange`
- `market`
- `symbol`
- `interval`
- `trigger_clock`: `strategy_close` / `base`
- `mode`: `paper` / `live`
- `desired_status`: `running` / `paused` / `stopped` / `dry_run`
- `capital_limit`
- `max_order_notional`
- `max_position_notional`
- `signal_only`
- `enabled`
- `created_at`
- `updated_at`

### 6.7 task_runs

每次 daemon 实际启动任务时生成：

- `id`
- `task_id`
- `mode`
- `observed_status`: `starting` / `running` / `stopping` / `stopped` / `failed`
- `started_at`
- `stopped_at`
- `error`

### 6.8 orders / fills

所有订单和成交必须带：

- `task_id`
- `strategy_id`
- `exchange_account_id`
- `exchange`
- `symbol`
- `client_order_id`
- `exchange_order_id`

`client_order_id` 必须可追踪，建议格式：

```text
hi-{taskShort}-{symbolShort}-{yyyymmdd}-{seq}
```

### 6.9 notification_channels

通知通道配置。通道只保存非密钥配置，真实 token / secret 通过环境变量提供：

- `id`
- `type`: `email` / `telegram` / `feishu` / `webhook`
- `name`
- `enabled`
- `config_json`
- `secret_env`
- `created_at`
- `updated_at`

### 6.10 notification_routes

通知路由规则。系统级通知和策略级通知共用同一套路由：

- `id`
- `scope`: `system` / `strategy` / `task`
- `strategy_id`
- `task_id`
- `min_severity`: `info` / `warn` / `error` / `critical`
- `channel_id`
- `enabled`
- `created_at`
- `updated_at`

规则：

- `system` route 不绑定 strategy/task。
- `strategy` route 绑定 `strategy_id`。
- `task` route 绑定 `task_id`。
- task route 优先于 strategy route，strategy route 优先于 system route。

### 6.11 notification_events / notification_deliveries

`notification_events` 保存需要通知的事实，`notification_deliveries` 保存每个通道的投递结果。

通知事件来源：

- system：daemon 启停、task failed、live uncertain order、kill switch、reconcile 异常。
- strategy：策略监测到信号，希望人工核对。
- risk：风控拒单、风险阈值接近或触发。
- execution：订单 rejected、uncertain、部分成交长时间未完成。

要求：

- 通知事件必须先落库，再投递。
- 投递失败必须记录并可重试。
- 同一个 dedupe key 在冷却窗口内不得重复轰炸。
- 策略不能直接调用 provider，只能返回 `NotificationIntent`。

### 6.12 web_users / web_sessions

操作台第一版使用单管理员登录模型，不做 RBAC。

`web_users` 保存管理员账号：

- `id`
- `username`
- `password_hash`
- `password_algo`: 第一版 `argon2id`，也可接受经过文档确认的 `bcrypt`
- `disabled`
- `failed_login_count`
- `locked_until`
- `last_login_at`
- `created_at`
- `updated_at`

`web_sessions` 保存 Web 会话：

- `id`
- `user_id`
- `token_hash`
- `csrf_token_hash`
- `user_agent_hash`
- `ip`
- `created_at`
- `last_seen_at`
- `expires_at`
- `revoked_at`

规则：

- 系统不得提供默认用户名和默认密码。
- 第一个管理员必须通过 CLI 初始化。
- 密码明文不得入库、不得写日志、不得进入 events。
- session cookie MUST 使用 `HttpOnly`、`SameSite=Strict`；HTTPS 部署时 MUST 使用 `Secure`。
- 所有写操作 API MUST 校验 CSRF token。
- 登录失败 MUST 限速；连续失败后临时锁定。
- logout、密码修改、管理员初始化、登录失败锁定、危险操作必须写入 `events` 或 auth audit event。
- 第一版可以只有一个管理员账号；后续如要多用户、角色、MFA，必须单独设计，不能在 handler 中零散加字段。

## 7. K 线同步与聚合设计

### 7.1 目标

K 线同步只做简单数据工具，不做复杂行情数据平台。

操作台只回答：

- 哪些交易所、交易对的 1m 原始 K 线已经同步过。
- 当前数据是否完整。
- 有缺口时能否一键修复。
- 哪些高周期可以由 1m 数据内部聚合得到。

第一版推荐且默认的数据策略：

```text
exchange native 1m candles
  -> market_candles(raw 1m)
  -> CandleAggregator
  -> derived 5m / 15m / 30m / 1h / 4h / 1d candles
  -> backtest / paper / live strategy
```

原则：

- 交易所同步层默认只拉 1m。
- 高周期 K 线默认不从交易所重复拉取。
- 高周期 K 线默认不落入 `market_candles`，而是在查询时或 runner 内部由 1m 聚合生成。
- 如果未来确实要缓存聚合结果，必须使用独立表和 `derived_from_interval` 字段，不能和原始交易所 K 线混在一起。
- 回测、模拟盘、实盘必须使用同一个聚合器，不能各自实现一套聚合逻辑。
- 时间钟模型不等同于 1m K 线模型。第一版的基础事件是 1m closed candle；未来如果引入 tick / trade 数据，基础事件可以比 1m 更细，但高周期 K 线仍必须由统一聚合器或 candle builder 生成。

### 7.2 CLI

```sh
hi data sync \
  --exchange binance \
  --symbol BTCUSDT \
  --interval 1m \
  --from 2026-01-01T00:00:00Z \
  --to 2026-02-01T00:00:00Z

hi data sync \
  --exchange okx \
  --symbol BTC-USDT \
  --interval 1m \
  --from 2026-01-01T00:00:00Z \
  --to 2026-02-01T00:00:00Z

hi data check --exchange binance --symbol BTCUSDT --interval 1m
hi data repair --exchange binance --symbol BTCUSDT --interval 1m --from ... --to ...
hi data aggregate-check --exchange binance --symbol BTCUSDT --from ... --to ... --target-interval 15m
```

批量同步：

```sh
hi data sync \
  --exchange binance \
  --symbols BTCUSDT,ETHUSDT,SOLUSDT \
  --interval 1m \
  --from ... \
  --to ...
```

约束：

- `hi data sync` 第一版只允许 `--interval 1m`。
- `hi data check` 检查原始 1m 完整性。
- `hi data aggregate-check` 检查目标周期是否能由完整的 1m 数据聚合得到。
- 回测和任务可以选择 `--interval 5m/15m/30m/1h/4h/1d`，但数据来源仍然是 1m 聚合结果。

### 7.3 完整性检查

第一版只检查必要项：

- open time 是否按 1m 连续。
- 是否存在缺口。
- 是否存在重复，数据库唯一键兜底。
- OHLC 是否合法：
  - `high >= low`
  - `open` 在 high/low 范围内
  - `close` 在 high/low 范围内
  - 价格为正
  - volume 非负
- 未收盘 K 线不作为错误。

对于高周期：

- 先检查覆盖区间内 1m 是否完整。
- 再检查目标周期的边界是否能整除。
- 聚合结果不直接与交易所高周期结果比较。

### 7.4 修复

修复流程：

```text
scan range
  -> find missing windows / invalid windows
  -> fetch from exchange
  -> upsert market_candles
  -> check again
  -> save latest data_quality_report
```

前端和 CLI 只显示最终结果，不展示复杂内部事件流。

### 7.5 操作台页面

只做一个简表：

```text
Exchange | Symbol  | Base | First | Last | 1m Candles | Status | Derived | Actions
binance  | BTCUSDT | 1m   | ...   | ...  | 43200      | healthy | 5m,15m,30m,1h | sync / check
okx      | BTC-USDT| 1m   | ...   | ...  | 43190      | gaps    | -          | repair
```

状态只保留：

- `healthy`
- `missing_gaps`
- `invalid_candles`
- `syncing`
- `failed`

## 8. 回测设计

第一版回测基于 PostgreSQL 中的 1m `market_candles`，再按请求 interval 内部聚合，不依赖临时 CSV。这里的 1m 是 MVP 的基础数据粒度，不是 TimeClock 的永久上限；如果未来接入 tick / trade 级事实数据，回测可以使用更细的 `base_clock` 推进。

回测必须显式区分三个时间概念：

- `base_clock`：基础时间钟，表示回测引擎每次推进的最小市场数据事件。第一版为 `closed_candle:1m`；未来可以是 `trade` 或 `tick`。
- `strategy_interval`：策略观察周期，可以是 `1m`、`5m`、`15m`、`30m`、`1h`、`4h`、`1d`。
- `trigger_clock`：策略触发时钟，决定策略是在每个基础事件都执行，还是只在策略周期 K 线收盘时执行。

第一版支持两种 `trigger_clock`：

```text
strategy_close  只在 strategy_interval K 线收盘时触发策略
base            每个 base_clock 事件都触发策略；第一版等价于每根 1m closed candle
```

示例：

- `base_clock=closed_candle:1m, strategy_interval=15m, trigger_clock=strategy_close`：每 15 分钟触发一次策略。
- `base_clock=closed_candle:1m, strategy_interval=15m, trigger_clock=base`：每 1 分钟触发一次策略，策略上下文里能看到截至当前 1m 的 15m forming candle。
- 未来如果 `base_clock=trade, strategy_interval=15m, trigger_clock=base`：每笔 trade 事件都可以触发策略，策略上下文里的 15m forming candle 只能由截至当前 trade 的数据构成。

CLI 示例：

```sh
hi backtest run \
  --strategy ema-cross \
  --config ./configs/ema-cross.json \
  --exchange binance \
  --symbol BTCUSDT \
  --interval 1m \
  --trigger-clock strategy_close \
  --from 2026-01-01T00:00:00Z \
  --to 2026-02-01T00:00:00Z
```

回测输出：

- candle count
- order count
- fill count
- realized PnL
- fee
- win/loss count
- max drawdown，第一版可简化
- final equity

回测记录入库，便于后续比较，但第一版不做复杂可视化。

如果请求高周期回测：

```sh
hi backtest run \
  --strategy ema-cross \
  --exchange binance \
  --symbol BTCUSDT \
  --interval 15m \
  --trigger-clock base \
  --from 2026-01-01T00:00:00Z \
  --to 2026-02-01T00:00:00Z
```

第一版 runtime MUST 从 1m 数据加载并聚合 15m K 线。回测结果必须记录：

- `base_interval = 1m`
- `strategy_interval = 15m`
- `trigger_clock = base`
- `aggregation_method = ohlcv_v1`

## 9. 模拟盘设计

Paper task 由 daemon 管理，不是单次命令。

运行逻辑：

```text
daemon
  -> load running paper tasks
  -> each task pulls latest base event; v1 uses latest closed 1m candle / synced 1m data stream
  -> CandleAggregator builds task interval candle
  -> TimeClock decides whether to trigger strategy
  -> strategy generates intents
  -> risk checks
  -> paper executor simulates fill
  -> portfolio updates
  -> facts persisted
```

Paper executor 第一版支持：

- market order 按 close price 成交。
- limit order 基于 OHLC 判断是否成交。
- 固定手续费率。
- 固定滑点 bps。

不要提前实现复杂撮合、盘口深度、延迟模型。

Paper runner 约束：

- task interval 为 `1m` 时直接使用 1m candle。
- task interval 为高周期且 `trigger_clock=strategy_close` 时，必须等待目标周期 closed 后再调用策略。
- task interval 为高周期且 `trigger_clock=base` 时，每个 base event 都可以调用策略；第一版 base event 是 1m closed candle，策略上下文包含当前 forming target candle。
- 不允许直接请求交易所高周期 K 线。
- 同一批 1m 输入在 backtest 和 paper 中聚合出的高周期 candle 必须一致。

## 10. 实盘设计

实盘必须像模拟盘一样由 daemon 管理多个任务。

### 10.1 Live task 生命周期

```text
stopped
  -> starting
  -> running
  -> stopping
  -> stopped

running
  -> failed
  -> paused
```

`strategy_tasks.desired_status` 表示用户想要什么。  
`task_runs.observed_status` 表示系统实际状态。

### 10.2 控制命令

```sh
hi task start task-binance-btc-ema
hi task pause task-binance-btc-ema
hi task stop task-binance-btc-ema
hi task dry-run task-okx-eth-grid
hi task list
```

这些命令只写 PostgreSQL，不直接下单。

### 10.3 Live safety

实盘任务启动前必须满足：

- global kill switch 为 false。
- exchange account `trade_enabled = true`。
- task `mode = live`。
- task `desired_status = running`。
- symbol 在白名单内。
- API key 存在。
- 单笔 notional 不超过限制。
- task 持仓 notional 不超过限制。
- account 级别总风险不超过限制。

防盗护栏：

- exchange API key 只从环境变量读取，禁止通过 Web 页面查看、下载或复制。
- exchange account 默认 `trade_enabled = false`。
- global kill switch 默认开启，必须显式关闭后才允许 live 下单。
- Web 上关闭 kill switch、启用 account trading、启动 live task、提高风险额度时，MUST 要求二次确认；高风险操作 SHOULD 要求输入当前登录密码或使用短期 re-auth token。
- 可检查 API key 权限的交易所必须检查；如果发现 withdrawal / transfer 权限开启，live gate MUST 拒绝启动并发出系统通知。
- 检查不到 API key 权限时，MUST 在事件中明确记录，并在操作台显示风险提示。
- 所有 live start / stop / pause / dry-run 切换、kill switch 变更、risk limit 变更都必须写审计事件。

### 10.4 Reconcile

第一版只做最小 reconcile：

- 周期性查询 open orders。
- 周期性查询 recent fills。
- 将交易所状态合并到本地订单和成交事实。
- 发现未知订单或状态不一致时记录事件，不自动做激进修复。

Live runner 的行情输入：

- 第一版使用 closed 1m candle poll；未来支持 tick / trade 时，live runner 应切换为更细的 base event stream，但不能改变策略、风控、订单和账本边界。
- 高周期策略必须通过 `CandleAggregator` 聚合。
- `trigger_clock=strategy_close` 时，策略只在目标周期 candle closed 后运行。
- `trigger_clock=base` 时，策略可以在每个 base event 后运行；第一版等价于每根 1m closed candle。
- 不使用交易所原生高周期 K 线作为策略输入。

## 11. 交易所适配器

第一版只实现 Binance 和 OKX，但交易所层必须能扩展更多 adapter。扩展能力来自清晰的小接口和注册表，不来自一个臃肿的能力矩阵。

包边界：

```text
internal/exchange
  -> 交易所接口、错误分类、symbol normalization 契约

internal/adapter/binance
  -> Binance 实现

internal/adapter/okx
  -> OKX 实现
```

交易所接口第一版保持小：

```go
type MarketDataClient interface {
    HistoricalCandles(ctx context.Context, query KlineQuery) ([]Candle, error)
    LatestCandle(ctx context.Context, query LatestCandleQuery) (Candle, error)
}

type TradingClient interface {
    SubmitOrder(ctx context.Context, order OrderRequest) (OrderReport, error)
    CancelOrder(ctx context.Context, query OrderQuery) (OrderReport, error)
    OpenOrders(ctx context.Context, query OpenOrdersQuery) ([]OrderReport, error)
    RecentFills(ctx context.Context, query RecentFillsQuery) ([]Fill, error)
}
```

Adapter factory：

```go
type AdapterFactory interface {
    Exchange() string
    BuildMarketDataClient(cfg AccountConfig) (MarketDataClient, error)
    BuildTradingClient(cfg AccountConfig) (TradingClient, error)
    NormalizeSymbol(raw string) (Symbol, error)
}
```

Adapter registry：

```text
registry.Register(binance.Factory{})
registry.Register(okx.Factory{})

factory := registry.Get(exchangeID)
```

规则：

- `Exchange()` 返回值 MUST 等于 `exchange_catalog.id`。
- `internal/runtime` 只能依赖 `internal/exchange` 接口，不能 import 具体 adapter。
- 具体 adapter 只能在 `cmd` 或 composition root 中注册和装配。
- symbol normalization 归 adapter 管，例如 Binance `BTCUSDT`、OKX `BTC-USDT`。
- adapter 必须把交易所错误分类为 retryable、rejected、uncertain、unsupported。
- 暂不支持的能力返回明确 `ErrUnsupported`，不能静默降级。

`MarketDataClient` 约束：

- `HistoricalCandles` 第一版只请求交易所 1m K 线。
- `LatestCandle` 第一版只返回最近一根已收盘 1m K 线。
- 高周期 K 线由 `internal/data` 聚合，adapter 不负责聚合。

新增交易所流程：

1. 在 `exchange_catalog` 中加入新的 exchange id。
2. 新增 `internal/adapter/{exchange}` 包，实现 `AdapterFactory`、`MarketDataClient`、需要实盘时实现 `TradingClient`。
3. 增加 symbol normalization、instrument rules 拉取和权限检查。
4. 增加 httptest 覆盖 public candles、order submit/cancel、open orders、recent fills。
5. 增加 fake adapter runner 测试，证明 backtest/paper/live 语义不变。
6. 更新 CLI/Web 可选交易所列表。

不要为了未来交易所提前做巨大接口。接口以当前 Binance / OKX 的必要能力为准，但核心表结构和 runtime 不能写死只能支持这两个 adapter。

### 11.1 CandleAggregator

内部聚合接口：

```go
type CandleAggregator interface {
    Aggregate(ctx context.Context, req AggregateRequest) ([]model.Candle, error)
}

type AggregateRequest struct {
    Exchange       string
    Market         string
    Symbol         string
    BaseInterval   string // 1m in v1; this is data source granularity, not TimeClock's permanent limit
    TargetInterval string
    From           time.Time
    To             time.Time
}
```

实现规则：

- `BaseInterval` 第一版固定为 `1m`，因为第一版只持久化 1m K 线。
- `TargetInterval` 可以为 `1m`，此时直接返回原始 1m。
- `TargetInterval` 大于 1m 时，必须先检查 1m 完整性。
- backtest、paper、live 必须通过该接口拿策略周期 K 线。
- 如果未来引入 tick / trade 级基础事件，必须新增 MarketEventReader / CandleBuilder 或等价组件，把更细事件构造成策略所需 K 线；不能让各 runner 自己手写一套 tick 到 K 线逻辑。

## 11.5 通知设计

通知是第一版必须具备的能力，但必须保持边界清楚：策略不能直接发邮件、Telegram 或飞书；策略只能产出 `NotificationIntent`，由 runtime 落库并交给 notification worker 投递。

### 11.5.1 通知类型

系统级通知：

- daemon 启动失败。
- daemon 获取数据库锁失败。
- task 启动失败。
- task 异常退出。
- live order 进入 `uncertain`。
- reconcile 发现本地与交易所状态不一致。
- global kill switch 被开启或关闭。
- 数据同步失败。
- 数据检查发现缺口或非法 K 线。

策略级通知：

- 策略检测到买入/卖出信号，但任务配置为只通知人工核对。
- 策略检测到重要指标穿越。
- 策略检测到风险接近阈值。
- 策略进入观望状态但希望提醒。

风控/执行通知：

- 风控拒单。
- 单笔订单超过 task 限制。
- account 风险接近上限。
- 订单被交易所拒绝。
- 部分成交长时间未完成。

### 11.5.2 通知通道

第一版通道类型固定：

```text
email
telegram
feishu
webhook
```

Provider 接口：

```go
type Provider interface {
    Type() string
    Send(ctx context.Context, msg Message) error
}
```

Message 结构：

```go
type Message struct {
    EventID   string
    Severity  string
    Title     string
    Body      string
    Payload   map[string]string
    CreatedAt time.Time
}
```

密钥规则：

- SMTP password、Telegram bot token、Feishu webhook secret MUST 来自环境变量。
- `notification_channels.config_json` 只保存非密钥配置。
- 日志、events、deliveries 中 MUST NOT 输出 secret。

### 11.5.3 路由优先级

通知路由顺序：

```text
task route
  -> strategy route
  -> system route
```

规则：

- task route 命中时，仍可继续投递 strategy/system route，除非 route 配置 `exclusive = true`。第一版不实现 exclusive，全部命中 route 都投递。
- route 的 `min_severity` 会过滤低等级通知。
- route disabled 时不投递。
- channel disabled 时对应 delivery 标记为 `skipped`。

### 11.5.4 去重与冷却

去重以 `dedupe_key` 为核心。

规则：

- runtime 接收 `NotificationIntent` 后 MUST 生成稳定 `dedupe_key`。
- 在同一 route 的 cooldown 窗口内，重复 dedupe key 不再创建新的 delivery。
- 被冷却跳过的事件可以写入 `notification_events`，但 delivery 标记为 `skipped`，reason 写入 payload。
- critical 通知默认不冷却，除非 route 明确配置 cooldown。

### 11.5.5 投递与重试

notification worker 流程：

```text
poll pending deliveries
  -> mark sending
  -> provider.Send
  -> sent or failed
  -> failed with next_attempt_at
```

重试策略：

- 最大 3 次。
- backoff：30s、2m、10m。
- 失败后保留 last_error。
- worker 重启后从 `notification_deliveries` 恢复。

### 11.5.6 人工核对模式

策略可以只发通知，不下单：

```text
StrategyDecision{
  Orders: nil,
  Notifications: []NotificationIntent{signal}
}
```

也可以同时发通知和订单意图：

```text
StrategyDecision{
  Orders: []OrderIntent{order},
  Notifications: []NotificationIntent{signal}
}
```

如果任务配置为 `signal_only = true`，runtime MUST 丢弃订单意图，只保留通知意图，并写 event 说明订单被 signal-only 模式拦截。

第一版不做复杂人工审批下单。人工核对后的操作是用户通过 CLI/Web 手动调整 task 状态或策略配置；后续如要做“通知中审批下单”，必须单独设计，不能偷偷塞进 notification provider。

## 12. 配置方式

配置文件只放非密钥信息：

```yaml
database:
  dsn_env: TICTICK_HI_DATABASE_DSN

runtime:
  bind_addr: 127.0.0.1:8090
  task_poll_interval: 2s
  reconcile_interval: 30s
  kill_switch_default: true

web:
  allow_non_loopback_bind: false
  session_ttl: 12h
  reauth_ttl: 5m
  login_lockout_after: 5
  login_lockout_duration: 15m
  cookie_secure: auto

exchange_accounts:
  - id: binance-main
    exchange: binance
    market: spot
    environment: testnet
    api_key_env: TICTICK_HI_BINANCE_API_KEY
    secret_key_env: TICTICK_HI_BINANCE_SECRET_KEY
    trade_enabled: false

  - id: okx-main
    exchange: okx
    market: spot
    environment: sandbox
    api_key_env: TICTICK_HI_OKX_API_KEY
    secret_key_env: TICTICK_HI_OKX_SECRET_KEY
    passphrase_env: TICTICK_HI_OKX_PASSPHRASE
    trade_enabled: false

notification_channels:
  - id: ops-email
    type: email
    name: Ops Email
    enabled: true
    secret_env: TICTICK_HI_SMTP_PASSWORD
    config:
      smtp_host: smtp.example.com
      smtp_port: 587
      username_env: TICTICK_HI_SMTP_USERNAME
      from: bot@example.com
      to:
        - owner@example.com

  - id: strategy-telegram
    type: telegram
    name: Strategy Telegram
    enabled: true
    secret_env: TICTICK_HI_TELEGRAM_BOT_TOKEN
    config:
      chat_id_env: TICTICK_HI_TELEGRAM_CHAT_ID

notification_routes:
  - id: system-critical-email
    scope: system
    min_severity: error
    channel_id: ops-email
    cooldown: 5m

  - id: ema-signal-telegram
    scope: strategy
    strategy_id: ema-btc
    min_severity: info
    channel_id: strategy-telegram
    cooldown: 15m
```

生产实盘配置文件不得 group/world writable。

## 12.5 Docker 部署

第一版必须支持单机 Docker Compose 部署。目标是让本机、VPS 或小型服务器能稳定运行，不做 Kubernetes 化。

必须提供的文件：

```text
Dockerfile
.dockerignore
.env.example
config/docker.yaml
deploy/compose/docker-compose.yml
deploy/compose/docker-compose.prod.yml
docker/entrypoint.sh
docker/healthcheck.sh
scripts/docker-smoke-test.sh
```

镜像要求：

- 使用多阶段构建：前端 build、Go build、最小 runtime image。
- 最终镜像只包含 `hi` binary、前端静态资源、必要证书和默认配置模板。
- 最终镜像 MUST 使用非 root 用户运行。
- 镜像内不得包含 `.env`、API key、数据库密码、Telegram token、Feishu secret、SMTP password。
- binary SHOULD 带版本信息：git commit、build time、version。
- 容器日志输出到 stdout/stderr，禁止默认写本地滚动日志文件。

Compose 服务：

```text
postgres
  -> PostgreSQL 数据库，使用持久化 volume

migrate
  -> 一次性执行 hi migrate，成功后退出

app
  -> hi daemon --config /app/config/docker.yaml
  -> 暴露 Web/health API
```

第一版 `docker-compose.yml` 默认用于本机安全运行：

- PostgreSQL 不暴露到宿主机公网端口，只在 compose network 内可见。
- app 端口默认绑定 `127.0.0.1:8090:8090`。
- `TICTICK_HI_DATABASE_DSN` 指向 compose 内部 PostgreSQL。
- `app` 依赖 `migrate` 成功完成。
- `postgres` 必须有 healthcheck。
- `app` 必须有 healthcheck。

生产覆盖文件 `docker-compose.prod.yml`：

- 可以配置反向代理网络，但 app 默认仍不直接承担 TLS。
- 如果绑定非 loopback 地址，必须显式设置 `web.allow_non_loopback_bind=true` 或等价环境变量。
- cookie secure 在 HTTPS 反向代理后必须启用。
- 生产环境建议使用 Docker secrets 或宿主机只读 env file 注入密钥。

迁移规则：

- migration 可以通过 `migrate` 一次性服务执行，也可以手动运行：

```sh
docker compose -f deploy/compose/docker-compose.yml run --rm migrate
```

- `app` 启动时可以检查 migration 是否已完成，但不应该在未知状态下偷偷执行破坏性迁移。
- migration 必须幂等、按顺序、可重复运行。

管理员初始化：

```sh
docker compose -f deploy/compose/docker-compose.yml run --rm app hi auth init-admin --username admin
```

约束：

- `init-admin` 必须交互式读取密码；不得通过 compose 环境变量传入管理员明文密码。
- `.env.example` 只能提供变量名和占位说明，不能包含真实密钥。
- exchange API key、SMTP password、Telegram token、Feishu secret 必须通过环境变量或 Docker secrets 注入。
- Docker 部署默认 `trade_enabled=false`、`kill_switch=true`。

健康检查：

- `/healthz`：进程存活，不要求数据库完全可用。
- `/readyz`：数据库连接、migration 状态、daemon lock、配置加载状态可用。
- Docker healthcheck SHOULD 调用 `/readyz` 或 `hi health --ready`。

备份与恢复：

- 第一版至少提供 PostgreSQL 备份/恢复脚本说明。
- 推荐脚本：

```text
scripts/backup-postgres.sh
scripts/restore-postgres.sh
```

- 备份文件不得包含在镜像内。
- restore 必须要求显式确认目标数据库，禁止误覆盖生产库。

Docker smoke test：

```sh
scripts/docker-smoke-test.sh
```

至少验证：

- `docker compose config` 通过。
- app image 能构建。
- postgres healthcheck 通过。
- migrate 服务执行成功。
- app `/readyz` 通过。
- `hi exchange list` 显示 Binance / OKX。
- 未登录访问受保护 API 返回 401。

## 13. 极简操作台

Web 操作台后置实现，且保持简单，但必须有基础登录鉴权和防盗护栏。前端技术栈默认：

```text
Vue 3 + Vite + TypeScript
TradingView Lightweight Charts
```

如果完整 TradingView Charting Library 能明显提升回测复盘、指标叠加、画线标注或多面板体验，并且项目具备合法可用的 Charting Library 资源，也可以选择完整 Charting Library。

约束：

- 使用 `web/frontend` 存放前端源码。
- 默认 K 线图表使用 TradingView 的轻量版本 `lightweight-charts`。
- 完整 TradingView Charting Library 是可选实现，不作为第一版默认依赖。
- 前端必须提供 `ChartAdapter` 边界，页面只依赖统一的 chart API，不能直接散落调用具体图表库。
- 不在前端实现交易逻辑。
- 图表数据来自后端 API，后端负责 1m 查询和高周期聚合。
- 除登录、登出、健康检查外，所有 Web API 默认需要登录态。
- Web API 默认不接受跨站请求；CORS 默认关闭。
- 所有写操作必须带 CSRF token。
- 前端不得持久化 session token 到 localStorage；使用 `HttpOnly` cookie 承载 session。

图表库选择原则：

- 如果只是展示 K 线、订单点、成交点、简单指标，优先 Lightweight Charts。
- 如果需要完整绘图工具、复杂指标面板、类似专业交易终端的交互，再选择完整 Charting Library。
- 无论选择哪一种，后端 API 和回测事实模型不能因此改变。
- 图表库切换不能影响 Data、Backtest、Orders 页面之外的业务逻辑。

第一版页面：

- Overview：任务状态、kill switch、今日 PnL 简表。
- Data：K 线同步、检查、修复。
- Tasks：任务列表，start / pause / stop。
- Orders：最近订单和成交。
- Notifications：通道、路由、最近通知和失败投递。

鉴权页面：

- Login：用户名、密码。
- Session：显示当前登录状态、最近登录时间、登出。
- 首次使用前必须通过 CLI 创建管理员；Web 不提供公开注册入口。

基础安全规则：

- 管理员密码使用 Argon2id 或经确认的 bcrypt 哈希。
- session token 使用高强度随机数，只保存 hash。
- session 默认 12 小时过期，可配置。
- 登录失败按 username + IP 限速；连续失败后锁定一段时间。
- 所有状态变更 API 记录操作者、IP、user agent hash、动作、目标对象 ID。
- Web 默认只监听 `127.0.0.1`；如果配置为非 loopback bind，启动时必须打印醒目警告，并要求配置明确确认。
- 部署到公网或局域网共享入口时，必须放在 HTTPS / 可信反向代理之后；应用层 cookie 在 HTTPS 下必须启用 `Secure`。
- 响应头 SHOULD 包含 `Content-Security-Policy`、`X-Frame-Options: DENY`、`Referrer-Policy`，降低页面被嵌入或脚本注入后误操作风险。

K 线图表要求：

- Data 页面可以查看 1m 原始 K 线和内部聚合出的 5m/15m/30m/1h/4h/1d。
- Backtest detail 可以显示回测使用的 strategy interval K 线、订单点和成交点。
- 图表不展示复杂数据治理信息。
- 图表 interval 切换必须调用后端聚合 API，不能前端自行聚合。

所有危险操作需要确认文本。  
危险操作包括：

- 关闭 kill switch。
- 启用 exchange account 的 `trade_enabled`。
- 启动 live task。
- 从 `dry_run` 切到 `running`。
- 提高 `capital_limit`、`max_order_notional`、`max_position_notional`。

危险操作确认要求：

- 确认文本必须包含 exchange、account、symbol、task id、mode。
- 操作 `关闭 kill switch`、`启用交易账号`、`启动 live task`、`提高风险额度` 时，MUST 要求当前密码或短期 re-auth token。
- 确认通过后只能在短时间窗口内使用，默认 5 分钟。

## 14. 工程实践要求

### 14.1 代码边界和依赖规则

代码边界必须能被静态检查，不能只靠自觉。

允许的依赖方向：

```text
cmd
  -> config, runtime, store/postgres, adapter, web

runtime
  -> model, data, strategy, risk, execution, portfolio, notification, store interfaces

auth
  -> model, store interfaces

data
  -> model, exchange interfaces, store interfaces

strategy
  -> model only

risk
  -> model only

execution
  -> model, exchange interfaces

portfolio
  -> model only

notification
  -> model, store interfaces

adapter/*
  -> exchange interfaces, model

store/postgres
  -> model, store interfaces

web
  -> auth, runtime/query service interfaces only
```

禁止的依赖：

- `internal/model` MUST NOT import 任何业务包。
- `internal/strategy` MUST NOT import store、adapter、execution、notification provider、web。
- `internal/risk` MUST NOT import store、adapter、execution、web。
- `internal/portfolio` MUST NOT import store、adapter、execution、web。
- `internal/runtime` MUST NOT import `internal/store/postgres`。
- `internal/runtime` MUST NOT import `internal/adapter/*`。
- `internal/auth` MUST NOT import `internal/store/postgres`、adapter、runtime、web。
- `internal/store/postgres` MUST NOT import runtime、adapter、web。
- `cmd` 只负责依赖装配和参数解析，不写业务逻辑。
- `web/frontend` MUST NOT 自行实现交易、聚合、风控、PnL 计算。

必须提供 import boundary check：

```sh
scripts/check-boundaries.sh
```

该脚本至少检查：

- strategy 包没有导入 store / adapter / execution / notification provider / web。
- runtime 包没有导入 postgres / concrete adapter。
- auth 包没有导入 postgres / adapter / runtime / web。
- model 包没有导入 internal 其它包。
- web/frontend 没有复制核心交易计算逻辑。

### 14.2 错误处理

- 所有外部调用必须带 context。
- 交易所错误需要分类：可重试、拒绝、未知执行状态。
- 未知执行状态不能直接当作失败。
- 网络超时后必须记录 pending / uncertain 事件。
- goroutine 顶层必须 recover 并写 error event，不能静默退出。
- 所有 error wrap 必须包含动作和关键对象 ID，例如 task_id、order_id、exchange。
- 禁止 `panic` 处理业务错误；panic 只允许在测试或不可恢复的编程错误中出现。
- 禁止吞掉错误后只写日志继续执行。
- live 下单、撤单、reconcile 的错误必须写入 `events`。

### 14.3 Context、并发和生命周期

- 所有数据库、HTTP、交易所、通知 provider 调用 MUST 接收 `context.Context`。
- daemon 下所有长期 goroutine MUST 由 supervisor 持有 cancel function。
- task runner MUST 响应 context cancellation。
- ticker MUST `Stop()`。
- channel send/receive 不能永久阻塞，必须受 context 或 buffer 策略控制。
- 禁止在策略中启动 goroutine。
- 禁止包级 mutable 全局变量保存 runtime 状态。
- rate limiter、sequence generator、client cache 必须挂在明确 owner 结构体下。

### 14.4 数据库和事务

- 不使用 ORM。
- PostgreSQL 使用 `pgx/v5` 和 `pgxpool`。
- migration 使用手写 SQL，按顺序执行。
- 每个 migration 在单独事务中执行。
- repository 方法必须小而明确，不做业务编排。
- 跨多表事实写入必须使用事务。
- 订单和成交写入必须幂等。
- client order sequence 必须由数据库事务生成。
- SQL 查询必须有测试覆盖，尤其是 upsert、reconcile、task polling。
- instrument_rules 的 tick_size、step_size、min_quantity、min_notional 应用必须有测试。
- 禁止在业务代码中拼接未校验 SQL。

### 14.5 包、接口、函数和文件大小

硬性代码规模约束：

- 单个 Go 文件 SHOULD 小于 400 行。
- 单个 Go 文件超过 600 行 MUST 拆分或写明原因。
- 单个函数 SHOULD 小于 60 行。
- 单个函数超过 100 行 MUST 拆分。
- 单个 interface SHOULD 不超过 5 个方法。
- 单个 struct SHOULD 不超过 12 个字段；配置 struct 和 DTO 可以例外，但必须保持分组清晰。
- 构造函数参数 SHOULD 不超过 5 个；更多参数必须使用 config struct。
- 单个 package SHOULD 聚焦一个领域；如果 package 名开始变成 `service`、`manager`、`common`、`utils`，必须重新命名或拆分。

禁止：

- 禁止 `internal/app` 这类万能胶水包在第一版出现。
- 禁止 `utils`、`common`、`helper` 这类无边界包。
- 禁止复制粘贴同一套订单状态转换。
- 禁止同一个文件同时处理 CLI、数据库、交易所和策略。
- 禁止在测试之外使用 sleep 等待异步结果；必须用 context、fake clock 或显式同步。

### 14.6 领域不变量

代码必须用测试保护这些不变量：

- 策略只返回意图，不直接产生订单事实。
- 风控拒绝的 intent 不得提交交易所。
- 每个 order 必须归属一个 task。
- 每个 fill 必须归属一个 order 和 task。
- fill 累计数量不得超过 order quantity。
- position 只能由 fill 推导更新。
- 所有 Quantity / Money / Price / Fee / PnL 必须带 asset。
- 订单 quantity asset 必须等于 base asset。
- 订单 quote amount asset 必须等于 quote asset。
- 订单 price 必须是 quote/base。
- fee asset 不得被默认成 quote asset。
- 风控必须使用交易所规则舍入后的 notional。
- live 下单前必须满足 tick_size、step_size、min_quantity、min_notional。
- backtest/paper/live 使用同一个时间钟和聚合器语义。
- `trigger_clock=base` 不得看到未来 1m 数据。
- `trigger_clock=strategy_close` 不得看到 forming strategy candle。
- notification event 必须先落库再投递。
- notification delivery 必须记录 sent/failed/skipped。
- Web 写操作必须有已登录 session 和 CSRF 校验。
- 高风险 Web 操作必须有二次确认，必要时必须通过 re-auth。
- API 密钥、session token、CSRF token、密码明文不得写日志、events 或前端状态。

### 14.7 前端代码约束

- 前端使用 Vue 3 + Vite + TypeScript。
- 必须开启 TypeScript strict。
- 所有 API 类型 SHOULD 由后端 OpenAPI 或共享 schema 生成；手写类型必须有测试或契约校验。
- 页面组件不直接调用图表库，必须通过 `ChartAdapter`。
- 页面组件不直接拼装交易决策。
- 前端只展示后端返回的事实和派生结果。
- 前端不得把 session token、CSRF token 或密码持久化到 localStorage / IndexedDB。
- 前端所有写操作请求必须走统一 API client，由 API client 附带 CSRF header。
- 前端接收金额、价格、数量时必须使用 `{ value, asset }` 或 `{ value, base, quote }` 结构。
- 前端不能把金额事实转成 number 后再参与计算；展示格式化使用字符串/decimal helper。
- K 线 interval 切换必须请求后端聚合 API。
- 危险操作组件必须复用统一 confirmation 组件。
- 前端测试至少覆盖登录态、任务控制、数据同步动作、通知路由展示、危险操作确认和图表 adapter 基础渲染。

### 14.8 测试要求

每个阶段必须有测试：

- model validation 单元测试。
- numeric domain type / rounding / asset consistency 单元测试。
- strategy 单元测试。
- risk 单元测试。
- data check / repair 单元测试。
- exchange adapter 使用 httptest，不依赖真实网络。
- PostgreSQL repository 使用测试数据库或容器化测试。
- runner 使用 fake exchange / fake store 测试多任务启动停止。
- live submit 默认只跑 fake adapter，真实 testnet 单独手动 gate。
- TimeClock / CandleAggregator 必须有表格测试。
- auth 必须测试密码哈希校验、登录失败锁定、session 过期、logout revoke、CSRF 校验、re-auth TTL。
- notification router / cooldown / retry 必须有单元测试。
- frontend 至少有类型检查和核心组件测试。

### 14.9 质量门禁

基础检查：

```sh
go test ./...
go vet ./...
gofmt
scripts/check-boundaries.sh
```

后续提供脚本：

```sh
scripts/smoke-test.sh
scripts/release-check.sh
```

release check 至少包括：

- 单元测试。
- race-sensitive package 的 `go test -race`，至少覆盖 runtime、notification、execution。
- import boundary check。
- gofmt / go vet。
- PostgreSQL migration up。
- K 线同步 fake exchange smoke。
- CandleAggregator / TimeClock smoke。
- backtest smoke。
- paper task smoke。
- notification route / fake provider smoke。
- task start / pause / stop smoke。
- frontend typecheck。
- frontend component smoke。

### 14.10 代码评审准入标准

每次合并前必须回答：

- 这个变更属于哪个明确领域包？
- 有没有引入新的事实源？
- 有没有破坏 Strategy / Risk / Execution / Portfolio / Notification 边界？
- 是否新增或修改了状态机？如果是，是否更新文档和测试？
- 是否新增数据库字段或 migration？如果是，是否有 migration 测试？
- 是否新增交易所行为？如果是，是否有 httptest 或 fake adapter 测试？
- 是否破坏 exchange adapter 契约，或把某个交易所特殊逻辑塞进 runtime/data/execution？
- 是否影响 backtest/paper/live 共用语义？
- 是否影响 TimeClock 或 CandleAggregator？
- 是否影响前端图表数据契约？
- 是否有可重复的本地验证命令？
- 提交是否原子，是否没有夹带无关文件？
- 本地提交是否已经推送到远程分支？

没有清楚答案时，不合并。

### 14.11 Git 提交与远程同步

代码提交必须原子化，并同步推送到远程分支。

分支规则：

- 不直接在 `main` 上堆实现。
- 每个阶段或明确功能使用独立分支，例如 `feature/phase-3-data-sync`、`feature/phase-5-paper-runner`、`docs/implementation-plan`。
- 第一次推送分支时使用 `git push -u origin <branch>`。
- 后续每完成一个可验证的原子提交，都要同步 `git push` 到同一远程分支。
- 禁止在未确认的情况下 force push；确需改写远程历史时，只能使用 `--force-with-lease`，并必须先说明原因。

原子提交规则：

- 一个 commit 只表达一个清晰意图：一个模型变更、一个 migration、一个 adapter 行为、一个 runner 行为、一个前端页面切片或一个文档契约变更。
- 不把无关修改混在同一个 commit 中。
- 不把纯格式化和行为变更混在同一个 commit 中，除非格式化只影响同一小范围文件。
- 数据库 migration、对应 model/store 变更和测试可以作为一个原子提交。
- API contract、后端实现、前端调用和契约测试可以作为一个端到端原子提交，但范围必须小。
- 修复 review 意见时优先追加小提交；只有在明确要求整理历史时才 squash。

提交信息：

- 使用稳定格式：

```text
<type>(<scope>): <summary>
```

- `type` 第一版使用：`feat`、`fix`、`refactor`、`test`、`docs`、`chore`、`build`。
- `scope` 使用明确领域：`model`、`data`、`backtest`、`runtime`、`execution`、`portfolio`、`notification`、`auth`、`web`、`docker`、`docs`。
- 非平凡提交的 commit body MUST 写明验证命令；如果没有运行测试，必须说明原因。

提交前检查：

- commit 前必须查看 `git diff --stat` 和关键 diff，确认没有夹带无关文件。
- 不提交真实 `.env`、API key、数据库 dump、私钥、token、备份文件。
- 不提交本地 IDE 噪声、临时文件、日志文件。
- 提交前至少运行与改动范围匹配的检查；跨领域改动必须运行 `scripts/release-check.sh` 或记录未运行原因。

推送规则：

- 每个原子提交完成并通过对应检查后，应推送到远程分支。
- push 失败必须停止后续提交，先处理远程同步问题。
- 远程分支是协作事实源之一；本地长期不推送的大批量提交不允许进入主线。
- 合并前远程分支必须包含所有本地提交，且工作区不得有未说明的脏改动。

## 15. 实施阶段

### Phase 0: 文档和边界确认

产出：

- 本实施计划。
- README 项目定位。
- 初始工程约束文档。

验收：

- 方向确认：`tictick-bot` 多交易所版。
- PostgreSQL 确认。
- Binance / OKX 范围确认。
- 数据同步操作台简化原则确认。

### Phase 1: 项目骨架

产出：

- Go module。
- `cmd/hi` CLI。
- config loader。
- PostgreSQL connection。
- migration runner。
- Dockerfile、`.dockerignore`、`.env.example`。
- `deploy/compose/docker-compose.yml`，包含 postgres、migrate、app。
- 基础测试脚本。

验收：

```sh
go test ./...
go run ./cmd/hi --help
go run ./cmd/hi migrate
docker compose -f deploy/compose/docker-compose.yml config
```

### Phase 2: 核心模型与存储

产出：

- Candle、OrderIntent、Order、Fill、Position、Portfolio。
- exchange_catalog。
- PostgreSQL migrations。
- repository 接口和 postgres 实现。

验收：

- model validation 测试通过。
- repository CRUD 测试通过。

### Phase 3: K 线同步、检查、修复

产出：

- Binance public klines。
- OKX public candles。
- exchange adapter registry。
- market_candles upsert。
- data check。
- data repair。
- CandleAggregator。
- aggregate-check。
- 批量 symbols sync。

验收：

```sh
hi data sync --exchange binance ...
hi data sync --exchange okx ...
hi data check ...
hi data repair ...
hi data aggregate-check ...
```

### Phase 4: 策略和回测

产出：

- Strategy 接口。
- Strategy registry。
- `StrategyDecision`、`OrderIntent`、`NotificationIntent`。
- 内置 `ema-cross` 和测试策略。
- 基于 PostgreSQL K 线的 backtest runner。
- 高周期回测通过 1m 聚合生成策略 K 线。

验收：

```sh
hi strategy list
hi backtest run --strategy ema-cross ...
```

### Phase 5: Daemon 与 Paper 多任务

产出：

- strategy_tasks。
- task supervisor。
- paper runner。
- notification worker。
- fake provider、email、telegram、feishu provider 骨架。
- task start / pause / stop CLI。
- notification channel / route / test CLI。

验收：

- 同时启动多个 paper task。
- 单独暂停其中一个 task 不影响其它 task。
- 所有订单、成交、事件都带 task 归因。
- 策略能发出 `NotificationIntent`，并通过 fake provider 投递。
- 同一个 dedupe key 在 cooldown 内不会重复投递。

### Phase 6: Live 多任务

产出：

- Binance live trading adapter。
- OKX live trading adapter。
- live runner。
- live safety gate。
- 最小 reconcile。

验收：

- dry-run live task 可运行。
- testnet / sandbox 小额订单手动 gate 可通过。
- 同一 daemon 可管理 Binance 和 OKX 的 live task。

### Phase 7: 极简操作台

产出：

- `internal/auth`。
- `web_users` / `web_sessions` migration 和 repository。
- `hi auth init-admin` / `change-password` / `session list` / `session revoke`。
- Web login / logout / session API。
- CSRF middleware 和 auth middleware。
- `/healthz` 和 `/readyz`。
- Vue 3 + Vite + TypeScript 前端骨架。
- ChartAdapter 图表适配层。
- 默认 TradingView Lightweight Charts K 线实现，必要时可替换为完整 TradingView Charting Library。
- Overview。
- Data。
- Tasks。
- Orders。
- Notifications。

验收：

- 未登录时除 login/health 外不能访问 API。
- 登录失败限速、session 过期、logout revoke、CSRF 校验测试通过。
- Data 页面只展示简表和 sync/check/repair。
- K 线图能展示 1m 原始数据和后端聚合出的高周期数据。
- Tasks 页面可 start/pause/stop。
- live 相关危险操作需要确认和 re-auth。
- Notifications 页面只展示通道、路由、最近通知、失败投递和 test/retry 动作。
- 默认仅本机访问。

### Phase 8: Docker 部署验收

产出：

- 多阶段 Dockerfile。
- `deploy/compose/docker-compose.yml`。
- `deploy/compose/docker-compose.prod.yml`。
- `config/docker.yaml`。
- `docker/entrypoint.sh`。
- `docker/healthcheck.sh`。
- `scripts/docker-smoke-test.sh`。
- PostgreSQL backup / restore 脚本或明确文档。

验收：

```sh
docker compose -f deploy/compose/docker-compose.yml config
docker compose -f deploy/compose/docker-compose.yml build
scripts/docker-smoke-test.sh
```

验收标准：

- PostgreSQL volume 持久化。
- migration 服务能成功执行。
- app ready 后 `/readyz` 返回成功。
- 默认只绑定 `127.0.0.1:8090`。
- 未登录不能访问受保护 API。
- 镜像和 compose 文件中不包含真实密钥。

## 16. 第一版成功标准

第一版完成时，系统应该能做到：

- 一个 PostgreSQL 数据库保存全部事实。
- 能同步 Binance / OKX 历史 K 线。
- 能检查并修复 K 线缺口。
- 能从 1m K 线内部聚合出 5m/15m/1h/4h/1d。
- 能用同步数据跑回测。
- 能同时运行多个 paper task。
- 能同时运行多个 live task。
- 操作台有基础登录鉴权、session 过期、CSRF 防护和登出撤销。
- live 危险操作默认受 kill switch、trade_enabled、确认文本、re-auth 和风险额度保护。
- 能发送系统级通知和策略级信号通知。
- 能通过 Docker Compose 一键启动 PostgreSQL、migration 和 app，并通过健康检查。
- 每个 task 可独立启动、暂停、停止。
- 每笔订单和成交都有明确 task / strategy / exchange 归因。
- 操作台清爽，不暴露内部复杂性。
- 代码结构清楚，新人能从目录和接口理解系统。

## 17. 长期维护原则

任何新功能进入前必须回答：

- 它是否服务于数据、策略、回测、模拟、实盘、通知这六件事？
- 它会不会让操作台变复杂？
- 它是否引入第二套事实源？
- 它是否破坏策略、风控、执行、账本边界？
- 它是否能用测试证明？

如果答案不清楚，就不加。

## 18. 单一事实源矩阵

系统中每类状态只能有一个权威来源。任何实现如果引入第二个事实源，必须先停下来重写设计。

| 领域 | 权威事实源 | 允许的缓存/派生 | 禁止 |
| --- | --- | --- | --- |
| 交易所目录 | `exchange_catalog` + adapter registry | CLI/Web 可选列表 | UI 或 runtime 私自硬编码完整交易所列表 |
| 交易所账号配置 | `exchange_accounts` | 进程内只读配置副本 | 把密钥明文写入库 |
| 全局开关 | `settings` | daemon 本地缓存，最多缓存一个 poll interval | CLI 直接改内存状态 |
| Web 管理员 | `web_users` | 当前请求 user context | 默认账号或配置文件明文密码 |
| Web session | `web_sessions` | HttpOnly session cookie 中的 opaque token | localStorage token 或纯内存 session |
| 用户期望任务状态 | `strategy_tasks.desired_status` | Web/CLI 展示副本 | runner 自己随意改 desired status |
| 实际运行状态 | `task_runs.observed_status` + daemon 内存 goroutine | dashboard 展示状态 | 只用内存状态判断历史运行 |
| 原始历史 K 线 | `market_candles` 中的 1m exchange candles | 查询结果缓存 | CSV 临时文件作为生产事实 |
| 高周期 K 线 | 由 1m `market_candles` 通过 `CandleAggregator` 派生 | 可选聚合缓存 | 交易所原生高周期和内部聚合混用 |
| 策略定义 | `strategies` | strategy registry 中的 factory | 策略代码读写自己的运行状态文件 |
| 订单事实 | `orders` | open order 查询视图 | portfolio 内部私藏订单状态 |
| 成交事实 | `fills` | 最近成交查询视图 | 只靠交易所 recent fills 不落库 |
| 仓位 | `positions` 最新快照，来源于 fills replay | dashboard summary | 手工 UPDATE 仓位绕过 fill |
| PnL | `portfolio_snapshots`，来源于 portfolio ledger | overview 聚合 | execution 层直接计算并写 PnL |
| 事件 | `events` | 操作台最近事件列表 | 只写日志不写事件 |
| 通知事件 | `notification_events` | 最近通知列表 | 策略直接调用通知 provider |
| 通知投递 | `notification_deliveries` | provider response 摘要 | 发完不记录结果 |

规则：

- Facts first：订单、成交、K 线、任务状态必须先落库，再给 UI 或 summary 使用。
- Derived second：positions、portfolio snapshots 可以是快照，但必须能由 orders / fills 解释。
- Logs are not facts：日志用于排查，不能替代 events 和 database facts。
- Memory is not authority：进程重启后必须能从 PostgreSQL 恢复可运行状态。

## 19. 核心类型契约

### 19.1 model.Candle

第一版 Candle 字段固定如下：

```go
type Candle struct {
    Exchange   string
    Market     string
    Symbol     string
    BaseAsset  Asset
    QuoteAsset Asset
    Interval   string
    OpenTime   time.Time
    Open       Price
    High       Price
    Low        Price
    Close      Price
    Volume     Quantity
    Complete   bool
}
```

`model.Candle` 可以表示原始 1m K 线，也可以表示内部聚合出的高周期 K 线。是否落库由数据层决定；第一版只有原始 1m exchange candle 落入 `market_candles`。

验证规则：

- `Exchange` MUST 是 `binance` 或 `okx`。
- `Market` 第一版 MUST 是 `spot`。
- `Symbol` MUST 非空。
- `BaseAsset` 和 `QuoteAsset` MUST 非空。
- `Interval` MUST 属于第一版 interval 白名单。
- `OpenTime` MUST 按 interval 对齐。
- `Open`、`High`、`Low`、`Close` MUST 大于 0。
- `Volume` MUST 大于等于 0。
- OHLC 的 `Base` MUST 等于 `BaseAsset`。
- OHLC 的 `Quote` MUST 等于 `QuoteAsset`。
- `Volume.Asset` MUST 等于 `BaseAsset`。
- `High >= Low`。
- `Open` 和 `Close` MUST 位于 `[Low, High]`。

价格和数量类型：

- PostgreSQL 中用 `NUMERIC(38, 18)`。
- Go 中 SHOULD 使用 `github.com/shopspring/decimal` 封装领域类型，禁止用 `float64` 保存订单价格、数量、金额事实。
- `float64` 不是禁用品，但只能用于适合近似计算的场景，例如指标、统计、图表坐标、展示型比例；它不能承载订单、成交、仓位、资金、手续费、PnL、风控额度等交易事实。

### 19.1.1 数值与精度契约

交易系统里的数值必须带语义，不能只传一个裸 decimal。

核心数值类型：

```go
type Asset string

type Quantity struct {
    Asset Asset
    Value decimal.Decimal
}

type Money struct {
    Asset Asset
    Value decimal.Decimal
}

type Price struct {
    Base  Asset
    Quote Asset
    Value decimal.Decimal // quote per 1 base
}

type Notional struct {
    Asset Asset
    Value decimal.Decimal
}

type Fee struct {
    Asset Asset
    Value decimal.Decimal
}

type PnL struct {
    Asset Asset
    Value decimal.Decimal
}
```

语义规则：

- `Asset` 使用交易所标准资产代码的大写形式，例如 `BTC`、`USDT`。OKX symbol 可以是 `BTC-USDT`，但 asset 仍然是 `BTC` 和 `USDT`。
- `Quantity` 表示 base asset 数量，例如 `BTC 0.01`。
- `Price` 表示 `quote/base`，例如 `USDT per BTC = 65000`。
- `Money` 表示某个资产的金额，例如 `USDT 1000`。
- `Notional` 表示订单或仓位名义价值，第一版统一使用 quote asset。
- `Fee` 必须带 fee asset，不能默认都是 USDT。
- `PnL` 必须带 asset，第一版 portfolio summary 统一折算到 quote asset。
- 任何函数如果接收 `decimal.Decimal`，必须在函数名或参数名中体现单位；核心交易链路 SHOULD 使用上面的领域类型。

禁止：

- 禁止在 model、risk、execution、portfolio、runtime 中使用 `float64` 保存交易事实。
- 禁止使用裸 `decimal.Decimal` 表示语义不明的金额。
- 禁止把 base quantity 和 quote amount 放进同一个字段。
- 禁止假定所有 fee 都是 quote asset。
- 禁止在前端重新计算 PnL、notional、fee。

`float64` 允许范围：

- 指标内部临时计算，例如 EMA、标准差、回归等。
- 策略信号判断中的指标值，例如 RSI、EMA 差值、z-score；它可以影响是否产生信号，但不能直接成为订单价格、数量或金额。
- 回测报告中的展示型百分比，例如胜率、收益率、回撤率；底层金额仍必须来自 decimal 或领域数值类型。
- 图表库坐标适配层，因为前端图表库通常使用 number。

`float64` 使用规则：

- 使用 float64 的函数必须位于明显的指标、统计或图表适配包内。
- `float64` 可以参与信号判断，但不能作为 `OrderIntent` 的 price、quantity、quote amount、risk limit、portfolio balance、PnL、fee 字段。
- 如果一个近似计算结果需要变成交易事实，必须在边界处显式转换为 `Price`、`Quantity`、`Money`、`Fee`、`PnL` 等领域类型，并经过精度、资产和舍入校验。
- 新增持久化字段、API 字段、核心模型字段时，默认不用 `float64`；确实需要时必须能说明它不是交易事实。

舍入规则：

- 价格下单前按交易所 `tick_size` 舍入。
- 数量下单前按交易所 `step_size` 舍入。
- BUY limit 价格默认向下舍入，避免超出预期价格。
- SELL limit 价格默认向上舍入，避免低于预期价格。
- BUY quantity 默认向下舍入，避免超出 notional 上限。
- SELL quantity 默认向下舍入，避免卖出超过持仓。
- 手续费和 PnL 计算保留内部高精度，展示层再按资产精度格式化。
- 风控使用舍入后的订单值重新计算 notional，不能只检查舍入前意图。

交易所规则：

- `instrument_rules` 必须记录 `base_asset`、`quote_asset`、`tick_size`、`step_size`、`min_qty`、`min_notional`。
- live 下单前必须应用交易所规则。
- paper/backtest 可以使用同一套 instrument rules；缺失时必须明确使用默认规则并在 run summary 记录。

数据库规则：

- 存储字段仍使用 `NUMERIC(38, 18)`，但表字段名必须体现语义，例如 `quantity`、`quote_amount`、`limit_price`、`fee`。
- 涉及资产的表必须带 `base_asset`、`quote_asset` 或具体 `*_asset` 字段。
- API 返回金额必须同时返回 `value` 和 `asset`。

### 19.2 Strategy 接口

策略接口保持小：

```go
type Strategy interface {
    Name() string
    Version() string
    Warmup() int
    OnTick(ctx context.Context, c StrategyContext) (model.StrategyDecision, error)
}
```

`StrategyContext` 只读：

```go
type StrategyContext struct {
    TaskID           string
    StrategyID       string
    Exchange         string
    Market           string
    Symbol           string
    BaseKind         string // closed_candle in v1; future trade/tick
    BaseInterval     string // 1m when BaseKind is closed_candle in v1
    BaseTime         time.Time
    StrategyInterval string
    TriggerClock     string // strategy_close or base
    BaseCandle       *model.Candle
    StrategyCandle   model.Candle
    StrategyClosed   bool
    Candles          []model.Candle
    Position         model.PositionSnapshot
    Portfolio        model.PortfolioSnapshot
    Now              time.Time
}
```

字段语义：

- `BaseKind` 表示当前时间钟由什么基础事件推进。第一版固定为 `closed_candle`，未来可以扩展为 `trade` 或 `tick`。
- `BaseInterval` 第一版为 `1m`；当未来基础事件不是 K 线时，`BaseInterval` 可以为空或使用稳定枚举，不能被策略假定为永远 `1m`。
- `BaseTime` 是当前基础事件的事件时间，不是机器当前时间。
- `BaseCandle` 在第一版 MUST 非空，表示当前时间钟对应的 1m closed candle；未来 tick / trade 模式下可能为空。
- `StrategyCandle` 是当前策略周期 K 线；在 `trigger_clock=base` 时可能是 forming candle。
- `StrategyClosed` 表示 `StrategyCandle` 是否已经收盘。
- `Candles` 是策略周期的历史 closed candles，不包含未来 candle。

策略禁止事项：

- MUST NOT import `internal/store`。
- MUST NOT import `internal/adapter`。
- MUST NOT import `internal/execution`。
- MUST NOT import `internal/notification` provider。
- MUST NOT 读取环境变量。
- MUST NOT 启动 goroutine。
- MUST NOT 持久化文件。
- MUST NOT 使用当前真实时间做交易判断，必须使用 `StrategyContext.Now`。

### 19.3 StrategyDecision

`StrategyDecision` 是策略唯一返回值：

```go
type StrategyDecision struct {
    Orders        []OrderIntent
    Notifications []NotificationIntent
}
```

规则：

- `Orders` 进入 risk / execution 链路。
- `Notifications` 进入 notification router 链路。
- 空 decision 合法。
- strategy error 不应被伪装成 notification；runtime 负责把 strategy error 转成 system notification。

### 19.4 OrderIntent

`OrderIntent` 是策略可返回的交易意图：

```go
type OrderIntent struct {
    ID          string
    TaskID      string
    StrategyID  string
    Exchange    string
    Market      string
    Symbol      string
    BaseAsset   Asset
    QuoteAsset  Asset
    Side        Side
    Type        OrderType
    Quantity    Quantity
    QuoteAmount Money
    LimitPrice  Price
    Reason      string
    CreatedAt   time.Time
}
```

约束：

- market order 必须且只能使用 `Quantity` 或 `QuoteAmount` 其中一种。
- limit order MUST 有 `Quantity` 和 `LimitPrice`。
- `Quantity.Asset` MUST 等于 `BaseAsset`。
- `QuoteAmount.Asset` MUST 等于 `QuoteAsset`。
- `LimitPrice.Base` MUST 等于 `BaseAsset`。
- `LimitPrice.Quote` MUST 等于 `QuoteAsset`。
- `Reason` MUST 非空，用于解释策略为什么下单。
- `TaskID`、`StrategyID` 由 runtime 补齐，策略不得伪造其它任务 ID。

### 19.5 NotificationIntent

`NotificationIntent` 表示策略希望系统发出一条可路由通知，例如“出现某个信号，需要人工核对”：

```go
type NotificationIntent struct {
    ID          string
    TaskID      string
    StrategyID  string
    Exchange    string
    Market      string
    Symbol      string
    Severity    NotificationSeverity
    Category    string
    Title       string
    Message     string
    DedupeKey   string
    Cooldown    time.Duration
    Payload     map[string]string
    CreatedAt   time.Time
}
```

Severity 固定为：

```text
info, warn, error, critical
```

规则：

- `Title` 和 `Message` MUST 非空。
- `Category` SHOULD 使用稳定枚举，如 `signal`, `risk`, `execution`, `system`。
- `DedupeKey` SHOULD 非空；为空时 runtime 用 task/symbol/category/title 生成。
- `Cooldown` 为空时使用 route 默认冷却时间。
- 策略只能声明通知意图，不能选择具体 provider token。
- runtime MUST 补齐 `TaskID`、`StrategyID`、`Exchange`、`Market`、`Symbol`。

## 20. 状态机契约

### 20.1 task desired status

`strategy_tasks.desired_status` 只表达用户期望：

| 当前 | 允许变更为 | 含义 |
| --- | --- | --- |
| `stopped` | `dry_run`, `running` | 让 daemon 启动任务 |
| `dry_run` | `paused`, `stopped`, `running` | dry-run 可升级到 live running，但必须重新过 live gate |
| `running` | `paused`, `stopped`, `dry_run` | 停止真实提交或降级 dry-run |
| `paused` | `running`, `dry_run`, `stopped` | 恢复或停止 |

CLI / Web 只能改 desired status。runner 不允许直接把 desired status 改成 `running`。

### 20.2 task observed status

`task_runs.observed_status` 表达 daemon 实际状态：

```text
starting -> running -> stopping -> stopped
starting -> failed
running  -> failed
stopping -> failed
```

规则：

- 每次 runner 启动 MUST 创建新的 `task_runs`。
- runner 正常退出 MUST 写 `stopped_at` 和 `observed_status = stopped`。
- runner 因错误退出 MUST 写 `observed_status = failed` 和 `error`。
- daemon 重启时，如果发现旧 run 仍是 `running`，MUST 标记为 `failed` 或 `stopped_unclean`，再按 desired status 决定是否新建 run。

### 20.3 order status

订单状态第一版固定：

```text
intent_created
  -> risk_rejected
  -> pending_submit
  -> submitted
  -> partially_filled
  -> filled
  -> cancel_requested
  -> canceled
  -> rejected
  -> uncertain
```

关键规则：

- `risk_rejected` 不得产生交易所订单。
- `pending_submit` 表示准备提交但还没有确定交易所结果。
- 网络超时、连接断开、交易所返回不确定错误时 MUST 进入 `uncertain`。
- `uncertain` 只能通过 reconcile 变为 `submitted`、`filled`、`canceled`、`rejected` 或人工处理状态。
- 成交数量不得倒退。
- 累计成交数量不得超过订单数量。
- 同一个 `client_order_id` 只能归属一个 task。

## 21. PostgreSQL DDL 草案

第一版 migration MUST 至少包含以下结构。字段可以在实现时补充，但不能删除这些核心约束。

```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
  id TEXT PRIMARY KEY,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE web_users (
  id TEXT PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  password_algo TEXT NOT NULL CHECK (password_algo IN ('argon2id', 'bcrypt')),
  disabled BOOLEAN NOT NULL DEFAULT false,
  failed_login_count INTEGER NOT NULL DEFAULT 0 CHECK (failed_login_count >= 0),
  locked_until TIMESTAMPTZ,
  last_login_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE web_sessions (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES web_users(id),
  token_hash TEXT NOT NULL UNIQUE,
  csrf_token_hash TEXT NOT NULL,
  user_agent_hash TEXT,
  ip INET,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ
);

CREATE INDEX web_sessions_user_idx ON web_sessions (user_id, expires_at DESC);
CREATE INDEX web_sessions_active_idx ON web_sessions (expires_at DESC)
  WHERE revoked_at IS NULL;

CREATE TABLE exchange_catalog (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO exchange_catalog (id, name) VALUES
  ('binance', 'Binance'),
  ('okx', 'OKX')
ON CONFLICT (id) DO NOTHING;

CREATE TABLE exchange_accounts (
  id TEXT PRIMARY KEY,
  exchange TEXT NOT NULL REFERENCES exchange_catalog(id),
  market TEXT NOT NULL DEFAULT 'spot' CHECK (market IN ('spot')),
  environment TEXT NOT NULL CHECK (environment IN ('testnet', 'sandbox', 'production')),
  api_key_env TEXT NOT NULL,
  secret_key_env TEXT NOT NULL,
  passphrase_env TEXT,
  trade_enabled BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE instrument_rules (
  exchange TEXT NOT NULL REFERENCES exchange_catalog(id),
  market TEXT NOT NULL DEFAULT 'spot' CHECK (market IN ('spot')),
  symbol TEXT NOT NULL,
  base_asset TEXT NOT NULL,
  quote_asset TEXT NOT NULL,
  tick_size NUMERIC(38, 18) NOT NULL CHECK (tick_size > 0),
  step_size NUMERIC(38, 18) NOT NULL CHECK (step_size > 0),
  min_quantity NUMERIC(38, 18) NOT NULL DEFAULT 0 CHECK (min_quantity >= 0),
  min_notional NUMERIC(38, 18) NOT NULL DEFAULT 0 CHECK (min_notional >= 0),
  notional_asset TEXT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (exchange, market, symbol)
);

CREATE TABLE market_candles (
  exchange TEXT NOT NULL REFERENCES exchange_catalog(id),
  market TEXT NOT NULL DEFAULT 'spot' CHECK (market IN ('spot')),
  symbol TEXT NOT NULL,
  base_asset TEXT NOT NULL,
  quote_asset TEXT NOT NULL,
  interval TEXT NOT NULL CHECK (interval = '1m'),
  open_time TIMESTAMPTZ NOT NULL,
  open NUMERIC(38, 18) NOT NULL CHECK (open > 0),
  high NUMERIC(38, 18) NOT NULL CHECK (high > 0),
  low NUMERIC(38, 18) NOT NULL CHECK (low > 0),
  close NUMERIC(38, 18) NOT NULL CHECK (close > 0),
  volume NUMERIC(38, 18) NOT NULL CHECK (volume >= 0),
  complete BOOLEAN NOT NULL DEFAULT true,
  source TEXT NOT NULL DEFAULT 'exchange' CHECK (source = 'exchange'),
  ingested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (exchange, market, symbol, interval, open_time),
  CHECK (high >= low),
  CHECK (open >= low AND open <= high),
  CHECK (close >= low AND close <= high)
);

CREATE INDEX market_candles_lookup_idx
  ON market_candles (exchange, market, symbol, interval, open_time);

CREATE TABLE strategies (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  version TEXT NOT NULL,
  kind TEXT NOT NULL,
  config_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  config_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (name, version, config_hash)
);

CREATE TABLE strategy_tasks (
  id TEXT PRIMARY KEY,
  strategy_id TEXT NOT NULL REFERENCES strategies(id),
  exchange_account_id TEXT NOT NULL REFERENCES exchange_accounts(id),
  exchange TEXT NOT NULL REFERENCES exchange_catalog(id),
  market TEXT NOT NULL DEFAULT 'spot' CHECK (market IN ('spot')),
  symbol TEXT NOT NULL,
  base_asset TEXT NOT NULL,
  quote_asset TEXT NOT NULL,
  interval TEXT NOT NULL,
  trigger_clock TEXT NOT NULL DEFAULT 'strategy_close'
    CHECK (trigger_clock IN ('strategy_close', 'base')),
  mode TEXT NOT NULL CHECK (mode IN ('paper', 'live')),
  desired_status TEXT NOT NULL DEFAULT 'stopped'
    CHECK (desired_status IN ('running', 'paused', 'stopped', 'dry_run')),
  capital_limit NUMERIC(38, 18) NOT NULL CHECK (capital_limit > 0),
  capital_asset TEXT NOT NULL,
  max_order_notional NUMERIC(38, 18) NOT NULL CHECK (max_order_notional > 0),
  max_order_notional_asset TEXT NOT NULL,
  max_position_notional NUMERIC(38, 18) NOT NULL CHECK (max_position_notional > 0),
  max_position_notional_asset TEXT NOT NULL,
  signal_only BOOLEAN NOT NULL DEFAULT false,
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX strategy_tasks_status_idx
  ON strategy_tasks (enabled, desired_status, mode);

CREATE TABLE task_runs (
  id TEXT PRIMARY KEY,
  task_id TEXT NOT NULL REFERENCES strategy_tasks(id),
  mode TEXT NOT NULL CHECK (mode IN ('paper', 'live')),
  observed_status TEXT NOT NULL
    CHECK (observed_status IN ('starting', 'running', 'stopping', 'stopped', 'failed', 'stopped_unclean')),
  started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  heartbeat_at TIMESTAMPTZ,
  stopped_at TIMESTAMPTZ,
  error TEXT
);

CREATE TABLE orders (
  id TEXT PRIMARY KEY,
  task_id TEXT NOT NULL REFERENCES strategy_tasks(id),
  strategy_id TEXT NOT NULL REFERENCES strategies(id),
  exchange_account_id TEXT NOT NULL REFERENCES exchange_accounts(id),
  exchange TEXT NOT NULL REFERENCES exchange_catalog(id),
  market TEXT NOT NULL DEFAULT 'spot' CHECK (market IN ('spot')),
  symbol TEXT NOT NULL,
  base_asset TEXT NOT NULL,
  quote_asset TEXT NOT NULL,
  client_order_id TEXT NOT NULL,
  exchange_order_id TEXT,
  side TEXT NOT NULL CHECK (side IN ('buy', 'sell')),
  type TEXT NOT NULL CHECK (type IN ('market', 'limit')),
  quantity NUMERIC(38, 18),
  quantity_asset TEXT,
  quote_amount NUMERIC(38, 18),
  quote_amount_asset TEXT,
  limit_price NUMERIC(38, 18),
  limit_price_base_asset TEXT,
  limit_price_quote_asset TEXT,
  status TEXT NOT NULL,
  reason TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (exchange_account_id, client_order_id),
  CHECK ((quantity IS NULL AND quantity_asset IS NULL) OR (quantity IS NOT NULL AND quantity_asset = base_asset)),
  CHECK ((quote_amount IS NULL AND quote_amount_asset IS NULL) OR (quote_amount IS NOT NULL AND quote_amount_asset = quote_asset)),
  CHECK ((limit_price IS NULL AND limit_price_base_asset IS NULL AND limit_price_quote_asset IS NULL)
    OR (limit_price IS NOT NULL AND limit_price_base_asset = base_asset AND limit_price_quote_asset = quote_asset)),
  CHECK (
    (type = 'market' AND ((quantity IS NOT NULL AND quote_amount IS NULL) OR (quantity IS NULL AND quote_amount IS NOT NULL))) OR
    (type = 'limit' AND quantity IS NOT NULL AND quote_amount IS NULL AND limit_price IS NOT NULL)
  )
);

CREATE TABLE fills (
  id TEXT PRIMARY KEY,
  order_id TEXT NOT NULL REFERENCES orders(id),
  task_id TEXT NOT NULL REFERENCES strategy_tasks(id),
  strategy_id TEXT NOT NULL REFERENCES strategies(id),
  exchange_account_id TEXT NOT NULL REFERENCES exchange_accounts(id),
  exchange_trade_id TEXT,
  symbol TEXT NOT NULL,
  base_asset TEXT NOT NULL,
  quote_asset TEXT NOT NULL,
  side TEXT NOT NULL CHECK (side IN ('buy', 'sell')),
  quantity NUMERIC(38, 18) NOT NULL CHECK (quantity > 0),
  quantity_asset TEXT NOT NULL,
  price NUMERIC(38, 18) NOT NULL CHECK (price > 0),
  price_base_asset TEXT NOT NULL,
  price_quote_asset TEXT NOT NULL,
  fee NUMERIC(38, 18) NOT NULL DEFAULT 0 CHECK (fee >= 0),
  fee_asset TEXT NOT NULL,
  filled_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (exchange_account_id, exchange_trade_id),
  CHECK (quantity_asset = base_asset),
  CHECK (price_base_asset = base_asset AND price_quote_asset = quote_asset)
);

CREATE TABLE data_sync_jobs (
  id TEXT PRIMARY KEY,
  exchange TEXT NOT NULL REFERENCES exchange_catalog(id),
  market TEXT NOT NULL DEFAULT 'spot' CHECK (market IN ('spot')),
  symbol TEXT NOT NULL,
  base_asset TEXT NOT NULL,
  quote_asset TEXT NOT NULL,
  interval TEXT NOT NULL,
  start_time TIMESTAMPTZ NOT NULL,
  end_time TIMESTAMPTZ NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('queued', 'running', 'succeeded', 'failed')),
  error TEXT,
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE data_quality_reports (
  id TEXT PRIMARY KEY,
  exchange TEXT NOT NULL REFERENCES exchange_catalog(id),
  market TEXT NOT NULL DEFAULT 'spot' CHECK (market IN ('spot')),
  symbol TEXT NOT NULL,
  interval TEXT NOT NULL,
  base_interval TEXT NOT NULL DEFAULT '1m',
  start_time TIMESTAMPTZ NOT NULL,
  end_time TIMESTAMPTZ NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('healthy', 'missing_gaps', 'invalid_candles')),
  expected_count BIGINT NOT NULL CHECK (expected_count >= 0),
  actual_count BIGINT NOT NULL CHECK (actual_count >= 0),
  missing_count BIGINT NOT NULL CHECK (missing_count >= 0),
  invalid_count BIGINT NOT NULL CHECK (invalid_count >= 0),
  summary_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  checked_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX data_quality_reports_lookup_idx
  ON data_quality_reports (exchange, market, symbol, interval, checked_at DESC);

CREATE TABLE backtest_runs (
  id TEXT PRIMARY KEY,
  strategy_id TEXT NOT NULL REFERENCES strategies(id),
  exchange TEXT NOT NULL REFERENCES exchange_catalog(id),
  market TEXT NOT NULL DEFAULT 'spot' CHECK (market IN ('spot')),
  symbol TEXT NOT NULL,
  interval TEXT NOT NULL,
  base_interval TEXT NOT NULL DEFAULT '1m',
  trigger_clock TEXT NOT NULL DEFAULT 'strategy_close'
    CHECK (trigger_clock IN ('strategy_close', 'base')),
  aggregation_method TEXT NOT NULL DEFAULT 'ohlcv_v1',
  start_time TIMESTAMPTZ NOT NULL,
  end_time TIMESTAMPTZ NOT NULL,
  candle_count BIGINT NOT NULL CHECK (candle_count >= 0),
  order_count BIGINT NOT NULL CHECK (order_count >= 0),
  fill_count BIGINT NOT NULL CHECK (fill_count >= 0),
  realized_pnl NUMERIC(38, 18) NOT NULL DEFAULT 0,
  realized_pnl_asset TEXT NOT NULL,
  fee NUMERIC(38, 18) NOT NULL DEFAULT 0 CHECK (fee >= 0),
  fee_asset TEXT NOT NULL,
  final_equity NUMERIC(38, 18) NOT NULL,
  final_equity_asset TEXT NOT NULL,
  max_drawdown NUMERIC(38, 18) NOT NULL DEFAULT 0,
  config_hash TEXT NOT NULL,
  summary_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE positions (
  id TEXT PRIMARY KEY,
  task_id TEXT NOT NULL REFERENCES strategy_tasks(id),
  strategy_id TEXT NOT NULL REFERENCES strategies(id),
  exchange_account_id TEXT NOT NULL REFERENCES exchange_accounts(id),
  exchange TEXT NOT NULL REFERENCES exchange_catalog(id),
  market TEXT NOT NULL DEFAULT 'spot' CHECK (market IN ('spot')),
  symbol TEXT NOT NULL,
  base_asset TEXT NOT NULL,
  quote_asset TEXT NOT NULL,
  quantity NUMERIC(38, 18) NOT NULL,
  quantity_asset TEXT NOT NULL,
  avg_entry_price NUMERIC(38, 18) NOT NULL DEFAULT 0 CHECK (avg_entry_price >= 0),
  avg_entry_price_base_asset TEXT NOT NULL,
  avg_entry_price_quote_asset TEXT NOT NULL,
  realized_pnl NUMERIC(38, 18) NOT NULL DEFAULT 0,
  realized_pnl_asset TEXT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (task_id, symbol)
);

CREATE TABLE portfolio_snapshots (
  id TEXT PRIMARY KEY,
  task_id TEXT NOT NULL REFERENCES strategy_tasks(id),
  strategy_id TEXT NOT NULL REFERENCES strategies(id),
  exchange_account_id TEXT NOT NULL REFERENCES exchange_accounts(id),
  equity NUMERIC(38, 18) NOT NULL,
  equity_asset TEXT NOT NULL,
  cash NUMERIC(38, 18) NOT NULL,
  cash_asset TEXT NOT NULL,
  position_notional NUMERIC(38, 18) NOT NULL DEFAULT 0,
  position_notional_asset TEXT NOT NULL,
  realized_pnl NUMERIC(38, 18) NOT NULL DEFAULT 0,
  realized_pnl_asset TEXT NOT NULL,
  unrealized_pnl NUMERIC(38, 18) NOT NULL DEFAULT 0,
  unrealized_pnl_asset TEXT NOT NULL,
  payload_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE events (
  id BIGSERIAL PRIMARY KEY,
  level TEXT NOT NULL CHECK (level IN ('debug', 'info', 'warn', 'error')),
  type TEXT NOT NULL,
  task_id TEXT REFERENCES strategy_tasks(id),
  strategy_id TEXT REFERENCES strategies(id),
  exchange_account_id TEXT REFERENCES exchange_accounts(id),
  actor_user_id TEXT REFERENCES web_users(id),
  actor_ip INET,
  actor_user_agent_hash TEXT,
  symbol TEXT,
  message TEXT NOT NULL,
  payload_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX events_created_at_idx ON events (created_at DESC);
CREATE INDEX events_task_idx ON events (task_id, created_at DESC);
CREATE INDEX events_actor_idx ON events (actor_user_id, created_at DESC);

CREATE TABLE settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE notification_channels (
  id TEXT PRIMARY KEY,
  type TEXT NOT NULL CHECK (type IN ('email', 'telegram', 'feishu', 'webhook')),
  name TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT true,
  config_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  secret_env TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE notification_routes (
  id TEXT PRIMARY KEY,
  scope TEXT NOT NULL CHECK (scope IN ('system', 'strategy', 'task')),
  strategy_id TEXT REFERENCES strategies(id),
  task_id TEXT REFERENCES strategy_tasks(id),
  min_severity TEXT NOT NULL DEFAULT 'info'
    CHECK (min_severity IN ('info', 'warn', 'error', 'critical')),
  channel_id TEXT NOT NULL REFERENCES notification_channels(id),
  cooldown_seconds INTEGER NOT NULL DEFAULT 300 CHECK (cooldown_seconds >= 0),
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (
    (scope = 'system' AND strategy_id IS NULL AND task_id IS NULL) OR
    (scope = 'strategy' AND strategy_id IS NOT NULL AND task_id IS NULL) OR
    (scope = 'task' AND task_id IS NOT NULL)
  )
);

CREATE INDEX notification_routes_strategy_idx
  ON notification_routes (strategy_id, enabled);

CREATE INDEX notification_routes_task_idx
  ON notification_routes (task_id, enabled);

CREATE TABLE notification_events (
  id TEXT PRIMARY KEY,
  source TEXT NOT NULL CHECK (source IN ('system', 'strategy', 'risk', 'execution')),
  severity TEXT NOT NULL CHECK (severity IN ('info', 'warn', 'error', 'critical')),
  category TEXT NOT NULL,
  title TEXT NOT NULL,
  message TEXT NOT NULL,
  dedupe_key TEXT NOT NULL,
  task_id TEXT REFERENCES strategy_tasks(id),
  strategy_id TEXT REFERENCES strategies(id),
  exchange_account_id TEXT REFERENCES exchange_accounts(id),
  symbol TEXT,
  payload_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX notification_events_created_at_idx
  ON notification_events (created_at DESC);

CREATE INDEX notification_events_dedupe_idx
  ON notification_events (dedupe_key, created_at DESC);

CREATE TABLE notification_deliveries (
  id TEXT PRIMARY KEY,
  notification_event_id TEXT NOT NULL REFERENCES notification_events(id),
  channel_id TEXT NOT NULL REFERENCES notification_channels(id),
  status TEXT NOT NULL CHECK (status IN ('pending', 'sending', 'sent', 'failed', 'skipped')),
  attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
  last_error TEXT,
  next_attempt_at TIMESTAMPTZ,
  sent_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (notification_event_id, channel_id)
);

CREATE INDEX notification_deliveries_pending_idx
  ON notification_deliveries (status, next_attempt_at);
```

Migration 规则：

- 使用顺序 migration，不使用 ORM auto-migrate。
- migration 必须可重复检测，不能静默跳过失败。
- 禁止 destructive migration 直接删除表或列；如确需删除，必须单独设计并保留备份步骤。
- migration runner 必须在事务中执行单个 migration。

## 22. K 线同步精确行为

### 22.1 时间区间

所有数据同步和回测使用半开区间：

```text
[from, to)
```

含义：

- 包含 `from` 对齐后的 K 线。
- 不包含 `to` 对齐点的 K 线。
- 数据同步的 `from` 和 `to` MUST 按 1m 对齐；CLI 可以自动向下/向上对齐，但必须在输出中说明。
- 高周期回测和任务的 `from` / `to` MUST 按目标周期对齐，同时也必须能映射到完整 1m 区间。
- 如果 `to` 为空，默认同步到最近一个已收盘 K 线，不同步正在形成的 K 线。

### 22.2 interval 白名单与数据来源

同步层第一版只支持：

```text
1m
```

策略、回测和任务 interval 第一版支持：

```text
1m, 5m, 15m, 30m, 1h, 4h, 1d
```

新增 interval 必须同时更新：

- aggregation interval mapping。
- open_time alignment。
- data check expected count。
- backtest query planning。
- tests。

禁止：

- 禁止 `hi data sync --interval 15m` 这类交易所高周期同步进入 MVP。
- 禁止 task runner 直接从交易所拉 15m/1h K 线绕过 1m 聚合。

### 22.3 sync 算法

```text
validate request
  -> normalize symbol per exchange
  -> require interval == 1m
  -> align [from, to) to 1m
  -> create data_sync_jobs row
  -> page fetch 1m candles from exchange
  -> validate each candle
  -> drop forming candle unless explicitly requested
  -> upsert market_candles
  -> update job status
  -> run data check on requested range
  -> save data_quality_reports
```

幂等要求：

- 同一请求重复执行 MUST 得到同一批 `market_candles`。
- upsert MUST 以 `(exchange, market, symbol, interval, open_time)` 为唯一键。
- 如果交易所返回同一 open_time 的不同 OHLC，默认以后拉结果覆盖旧值，并写 event 记录覆盖数量。
- 写入 `market_candles.interval` MUST 固定为 `1m`。

### 22.4 check 算法

给定 1m `[from, to)`：

```text
expected_count = (to - from) / 1m
actual rows ordered by open_time
  -> check first open_time == from
  -> check last open_time == to - 1m
  -> check each next == previous + 1m
  -> validate OHLC
```

输出：

```json
{
  "status": "healthy",
  "expected_count": 44640,
  "actual_count": 44640,
  "missing_count": 0,
  "invalid_count": 0,
  "first_open_time": "...",
  "last_open_time": "..."
}
```

### 22.5 repair 算法

repair 只修两类问题：

- 缺失 K 线。
- OHLC 非法 K 线。

repair 不做：

- 跨交易所价格一致性检查。
- 异常尖刺判断。
- 成交量异常判断。
- 自动删除历史数据。

流程：

```text
run check
  -> build repair windows
  -> merge adjacent windows
  -> fetch each window from exchange
  -> upsert
  -> run check again
```

repair window 合并规则：

- 连续缺口合并成一个窗口。
- 单根 invalid candle 按 `[open_time, open_time + 1m)` 修复。
- 相邻窗口距离小于等于 1m 时合并。

### 22.6 聚合算法

聚合器名称固定为 `ohlcv_v1`。

输入：

- 完整连续的 1m candles。
- target interval：`5m`、`15m`、`30m`、`1h`、`4h`、`1d`。
- 半开区间 `[from, to)`。

前置条件：

- `from` 和 `to` MUST 按 target interval 对齐。
- `[from, to)` 内所有 1m candle 必须完整存在。
- 未收盘 1m candle 不参与 closed target candle 聚合。

聚合公式：

```text
open   = first 1m open
high   = max(1m high)
low    = min(1m low)
close  = last 1m close
volume = sum(1m volume)
open_time = target bucket start
complete = all input 1m candles complete
```

bucket 对齐：

- `5m` / `15m` / `30m` / `1h` / `4h` 使用 UTC 时间边界。
- `1d` 使用 UTC 自然日。
- 第一版不支持交易所本地时区日线。

聚合输出默认不落库。消费者包括：

- backtest runner。
- paper runner。
- live runner 的策略输入。
- data aggregate-check。

如果性能需要缓存聚合结果，必须新增 `market_candle_aggregates` 表，并包含：

- `base_interval = 1m`
- `target_interval`
- `aggregation_method = ohlcv_v1`
- `source_range`
- `computed_at`

不能把聚合结果写回 `market_candles`。

## 22.7 时间钟模型

时间钟是 backtest、paper、live 共用的 runtime 推进模型。

时间钟的核心不是“1m K 线”，而是“系统当前拥有的最小市场数据事件”。

第一版实现：

```text
base_clock = closed_candle:1m
base_event = 已收盘 1m K 线
```

未来如果引入更细数据，可以扩展为：

```text
base_clock = trade
base_clock = tick
```

每个 task 可以配置：

```text
strategy_interval = 1m / 5m / 15m / 30m / 1h / 4h / 1d
trigger_clock = strategy_close / base
```

时间钟行为：

```text
for each base event ordered by event time:
  ingest base event
  update aggregator
  build current strategy candle
  if trigger_clock == base:
    call Strategy.OnTick
  if trigger_clock == strategy_close and strategy candle just closed:
    call Strategy.OnTick
```

防未来函数规则：

- `base_clock` 每次只推进一个已确认的基础事件。第一版基础事件是已收盘 1m candle；未来可以是 trade / tick。
- `trigger_clock=base` 时，策略可以看到当前 forming strategy candle，但这个 candle 只能由截至当前 base event 的数据构成。
- `trigger_clock=strategy_close` 时，策略只能看到已经收盘的 strategy candle。
- 策略产生的订单最早只能在下一个 base event 或其后续可成交事件执行。第一版等价于下一根 1m candle。
- 高周期策略不能用目标周期未来的 high / low / close。

`trigger_clock=base` 的用途：

- 允许用最小数据单位触发策略执行。
- 第一版表示每根 1m closed candle 后触发；未来 tick / trade 模式下可以比 1m 更细。
- 允许策略在 15m candle 尚未收盘时观察 forming candle。
- 适合信号监测、人工核对通知、提前预警。

`trigger_clock=strategy_close` 的用途：

- 更接近传统按 K 线收盘运行策略。
- 回测更容易解释。
- 默认适合大多数低频策略。

## 23. 回测精确语义

回测必须避免未来函数。第一版使用 `closed_candle:1m` base clock 推进：

```text
base_event[N] = 1m candle close
  -> TimeClock advances
  -> Strategy.OnTick may run depending on trigger_clock
  -> strategy emits intent
  -> order is eligible for fill on next executable base event; v1 uses base_candle[N+1]
```

第一版成交规则：

- Market order：在下一根 1m K 线 open price 成交，再应用滑点和手续费。
- Limit buy：如果下一根 1m K 线 low <= limit price，则成交价为 min(limit price, next open) 的保守实现。
- Limit sell：如果下一根 1m K 线 high >= limit price，则成交价为 max(limit price, next open) 的保守实现。
- 最后一根 K 线产生的 intent 不成交，只记录为 unfilled / ignored。

未来如果使用 tick / trade base clock，成交规则必须重新定义为基于下一批可成交事件，而不是沿用 1m OHLC 近似；该扩展必须单独更新 fill model 文档和测试。

Warmup：

- 策略 `Warmup()` 返回需要的 strategy interval closed candle 数。
- warmup 期间 runtime 不调用 `OnTick`。
- `trigger_clock=base` 时，只有 closed strategy candles 达到 warmup 数后，才允许每个 base event 调用 `OnTick`；第一版等价于每 1m 调用。

回测输出 MUST 包含：

- input range。
- base interval。
- strategy interval。
- trigger clock。
- aggregation method。
- candle count。
- strategy name/version/config hash。
- fee model。
- slippage model。
- fill model。
- order count。
- fill count。
- realized PnL。
- fee。
- final equity。
- max drawdown。

回测 run 入库必须保存 config hash，保证同一数据和同一策略配置可复验。

## 24. Daemon 与 TaskSupervisor 精确算法

daemon 启动流程：

```text
load config
  -> connect PostgreSQL
  -> run startup checks
  -> acquire daemon lock
  -> mark stale task_runs
  -> build exchange account runtimes
  -> start supervisor loop
```

MUST 使用 PostgreSQL advisory lock 或等价机制防止两个 daemon 同时管理同一数据库：

```sql
SELECT pg_try_advisory_lock(hashtext('tictick_hi_daemon'));
```

如果拿不到 lock，daemon MUST 拒绝启动。

supervisor poll 流程：

```text
every task_poll_interval:
  load enabled strategy_tasks
  for each task:
    if desired is running/dry_run and no local runner:
      validate task
      start runner goroutine
    if desired is paused/stopped and local runner exists:
      cancel runner context
    if task config changed while runner exists:
      stop old runner
      start new runner if desired still active
  for local runners whose task disappeared/disabled:
    cancel runner context
  write heartbeat for active task_runs
```

runner 停止要求：

- pause/stop MUST 使用 context cancellation。
- runner MUST 在合理时间内停止；默认 stop timeout 为 10 秒。
- 超过 timeout MUST 标记 `failed` 并写 event。

## 25. ExchangeAccountRuntime 模型

每个 `exchange_account` 在 daemon 内对应一个 `ExchangeAccountRuntime`：

```text
ExchangeAccountRuntime
  -> MarketDataClient
  -> TradingClient
  -> RateLimiter
  -> OrderIDSequence
  -> ReconcileLoop
```

多个 task 可以共享同一个 account runtime，但必须满足：

- 每个 task 有独立 runner。
- 下单前必须通过 task 级风控和 account 级风控。
- 同一 account 的 SubmitOrder MUST 经过统一 rate limiter。
- client order sequence MUST 由数据库事务生成，不能靠内存计数。
- reconcile 按 account 运行，再把订单和成交分发归因到 task。

第一版不做复杂 websocket 私有流依赖；可以先用 polling reconcile，后续再补 user stream。

## 26. CLI 合约

CLI 必须稳定、可脚本化。默认输出人类可读表格，所有查询命令 SHOULD 支持 `--json`。

### 26.1 交易所命令

```sh
hi exchange list
hi exchange show binance
hi exchange show okx
```

约束：

- `exchange list` 从 `exchange_catalog` 和 adapter registry 汇总可用交易所。
- 第一版只应显示 Binance 和 OKX 为 enabled。
- 如果 `exchange_catalog` 中存在但当前二进制没有注册 adapter，必须显示为 unavailable，不能在运行时 panic。

### 26.2 数据命令

```sh
hi data sync --exchange binance --symbol BTCUSDT --interval 1m --from ... --to ...
hi data sync --exchange okx --symbol BTC-USDT --interval 1m --from ... --to ...
hi data check --exchange binance --symbol BTCUSDT --interval 1m --from ... --to ...
hi data repair --exchange binance --symbol BTCUSDT --interval 1m --from ... --to ...
hi data aggregate-check --exchange binance --symbol BTCUSDT --target-interval 15m --from ... --to ...
hi data list
```

### 26.3 策略命令

```sh
hi strategy list
hi strategy create --id ema-btc --kind ema-cross --version v1 --config ./ema.json
hi strategy show ema-btc
```

### 26.4 回测命令

```sh
hi backtest run --strategy ema-btc --exchange binance --symbol BTCUSDT --interval 1m --trigger-clock strategy_close --from ... --to ...
hi backtest run --strategy ema-btc --exchange binance --symbol BTCUSDT --interval 15m --trigger-clock base --from ... --to ...
hi backtest list
hi backtest show RUN_ID
```

### 26.5 任务命令

```sh
hi task create --id btc-ema-paper --strategy ema-btc --account binance-main --symbol BTCUSDT --interval 1m --trigger-clock strategy_close --mode paper
hi task create --id btc-ema-15m-paper --strategy ema-btc --account binance-main --symbol BTCUSDT --interval 15m --trigger-clock base --mode paper
hi task start btc-ema-paper
hi task pause btc-ema-paper
hi task stop btc-ema-paper
hi task list
hi task show btc-ema-paper
```

`start/pause/stop` MUST 只写 desired status 和 event，不直接创建 goroutine。

### 26.6 鉴权命令

```sh
hi auth init-admin --username admin
hi auth change-password --username admin
hi auth session list --username admin
hi auth session revoke SESSION_ID
```

约束：

- `init-admin` MUST 交互式读取密码，禁止通过命令行参数传入明文密码。
- 如果已经存在管理员，`init-admin` MUST 拒绝覆盖；重置密码必须使用明确的 `change-password`。
- `change-password` 成功后 SHOULD 撤销该用户的所有旧 session。
- `session list` 不展示 token，只展示 session id、created_at、last_seen_at、expires_at、ip、user agent 摘要。
- `session revoke` 只标记 `revoked_at`，不删除审计事实。

### 26.7 daemon 命令

```sh
hi daemon --config ./config.yaml
hi health --live
hi health --ready
```

daemon MUST 打印：

- database target，不打印密码。
- bind address。
- loaded exchange accounts。
- acquired daemon lock。
- active task count。

`hi health --live` 只检查进程基础依赖和配置可解析。  
`hi health --ready` MUST 检查 PostgreSQL、migration 状态和必要运行时依赖，供 Docker healthcheck 使用。

### 26.8 通知命令

```sh
hi notification channel list
hi notification channel test ops-email
hi notification route list
hi notification event list
hi notification delivery retry DELIVERY_ID
```

约束：

- `channel test` MUST 发送一条明确标记为 test 的通知。
- `event list` 默认只展示最近 50 条。
- `delivery retry` 只修改 delivery 状态为 pending，不直接在 CLI 中执行 provider.Send。

## 27. 自动化代码质量检查清单

第 14 章的工程规则必须尽量落到脚本。第一版至少提供：

```sh
scripts/check-boundaries.sh
scripts/check-go-static.sh
scripts/check-frontend-static.sh
scripts/check-secrets.sh
scripts/smoke-test.sh
scripts/docker-smoke-test.sh
scripts/release-check.sh
```

`scripts/check-boundaries.sh` MUST 检查：

- `internal/model` 不导入其它 internal 包。
- `internal/strategy` 不导入 store、adapter、execution、notification provider、web。
- `internal/runtime` 不导入 store/postgres、任何 `internal/adapter/*`、web。
- `internal/auth` 不导入 store/postgres、adapter、runtime、web。
- `internal/store/postgres` 不导入 runtime、adapter、web。
- `cmd` 之外没有 package 同时导入 postgres 和 concrete adapter。

`scripts/check-go-static.sh` MUST 检查：

- `gofmt`。
- `go vet ./...`。
- 禁止新增 `internal/utils`、`internal/common`、`internal/helper`。
- 禁止 model、risk、execution、portfolio、runtime 中出现交易事实 `float64` 字段。
- 裸 `decimal.Decimal` 字段只允许出现在领域数值类型内部、DTO、数据库 scan struct 或指标临时计算中。
- 单文件超过 600 行时失败，测试 fixture 和 generated 文件除外。
- 单函数超过 100 行时至少输出 warning；后续可升级为失败。
- 禁止业务代码中的 `time.Sleep`，测试除外。
- 禁止业务代码中的裸 `panic`，测试除外。
- 禁止日志中出现 API key、secret、passphrase 字段值。

`scripts/check-frontend-static.sh` MUST 检查：

- TypeScript strict typecheck。
- Vue component test。
- 禁止页面组件直接 import 具体 chart library；只能 import ChartAdapter。
- 禁止前端实现 PnL、风控、订单状态转换和 K 线聚合业务逻辑。

`scripts/check-secrets.sh` MUST 检查：

- 禁止提交真实 `.env` 文件。
- 禁止提交私钥、API key、exchange secret、Telegram token、Feishu secret、SMTP password。
- 禁止提交数据库 dump、备份文件和本地日志。
- `.env.example` 只能包含占位符和变量说明。

`scripts/docker-smoke-test.sh` MUST 检查：

- `docker compose config`。
- Docker image build。
- PostgreSQL container healthcheck。
- migration 服务成功退出。
- app `/readyz` 成功。
- 默认端口只绑定 `127.0.0.1`。
- 未登录访问受保护 API 返回 401。
- image history / compose config 不包含明显 secret 字段值。

`scripts/release-check.sh` MUST 串联：

- Go static checks。
- frontend static checks。
- secret checks。
- import boundary check。
- unit tests。
- docker compose config。
- migration smoke。
- docker smoke。
- fake exchange data sync smoke。
- CandleAggregator / TimeClock smoke。
- auth smoke：init-admin、login、CSRF 写操作、logout revoke、session expired。
- backtest smoke。
- paper task start/pause/stop smoke。
- notification fake provider smoke。

任何 quality check 失败时，不能继续堆功能。

## 28. 阶段阻断条件

任何阶段如果出现以下情况，必须停止继续堆功能，先修结构：

- 出现第二套订单状态。
- 出现第二套 portfolio / ledger。
- 策略开始 import store 或 adapter。
- 策略开始 import notification provider 或直接发通知。
- CLI 命令直接执行 live 循环。
- Web 页面开始暴露内部复杂数据治理细节。
- runner 无法独立停止某一个 task。
- task 事件无法追溯到 exchange account。
- 一个 package 开始同时处理配置、数据库、交易所、策略和 HTTP。
- 测试只能靠真实交易所网络才能通过。

## 29. MVP 验收剧本

第一版真正可用前，必须能按下面剧本走通：

1. 初始化数据库：

```sh
hi migrate
```

2. 同步 Binance 和 OKX 历史 K 线：

```sh
hi data sync --exchange binance --symbol BTCUSDT --interval 1m --from 2026-01-01T00:00:00Z --to 2026-01-02T00:00:00Z
hi data sync --exchange okx --symbol BTC-USDT --interval 1m --from 2026-01-01T00:00:00Z --to 2026-01-02T00:00:00Z
```

3. 检查数据完整性：

```sh
hi data check --exchange binance --symbol BTCUSDT --interval 1m --from 2026-01-01T00:00:00Z --to 2026-01-02T00:00:00Z
hi data check --exchange okx --symbol BTC-USDT --interval 1m --from 2026-01-01T00:00:00Z --to 2026-01-02T00:00:00Z
hi data aggregate-check --exchange binance --symbol BTCUSDT --target-interval 15m --from 2026-01-01T00:00:00Z --to 2026-01-02T00:00:00Z
```

4. 基于同步数据跑回测：

```sh
hi strategy create --id ema-btc --kind ema-cross --version v1 --config ./configs/ema-btc.json
hi backtest run --strategy ema-btc --exchange binance --symbol BTCUSDT --interval 1m --trigger-clock strategy_close --from 2026-01-01T00:00:00Z --to 2026-01-02T00:00:00Z
hi backtest run --strategy ema-btc --exchange binance --symbol BTCUSDT --interval 15m --trigger-clock base --from 2026-01-01T00:00:00Z --to 2026-01-02T00:00:00Z
```

5. 创建两个 paper task 并同时运行：

```sh
hi task create --id btc-ema-paper --strategy ema-btc --account binance-main --symbol BTCUSDT --interval 1m --trigger-clock strategy_close --mode paper
hi task create --id eth-ema-paper --strategy ema-eth --account okx-main --symbol ETH-USDT --interval 15m --trigger-clock base --mode paper
hi task start btc-ema-paper
hi task start eth-ema-paper
hi daemon --config ./config.yaml
```

6. 暂停一个 task，另一个 task 必须继续运行：

```sh
hi task pause btc-ema-paper
hi task list
```

7. live dry-run 任务启动，不提交真实订单：

```sh
hi task create --id btc-ema-live-dry --strategy ema-btc --account binance-main --symbol BTCUSDT --interval 1m --trigger-clock strategy_close --mode live
hi task dry-run btc-ema-live-dry
```

8. 通知通道和策略信号通知可用：

```sh
hi notification channel test ops-email
hi notification route list
hi notification event list --json
```

验收要求：

- test 通知能产生 `notification_events`。
- fake provider 或真实测试通道能产生 `notification_deliveries.status = sent`。
- 策略信号能通过 `NotificationIntent` 进入通知链路。
- 相同 dedupe key 在 cooldown 内不会重复投递。

9. testnet / sandbox 小额 live gate 单独执行，默认不进入 release check。

只有这个剧本稳定通过，第一版才算具备使用价值。

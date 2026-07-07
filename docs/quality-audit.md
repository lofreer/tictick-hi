# Quality Audit

审计日期：2026-06-29

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
| Go 子命令 | scaffold | 保留后收敛 | 入口可用；API / sync / backtest / trading / notify 的关键 env 配置已收敛到严格解析函数，非法 duration / int / bool 和交易所限流配置会在启动前返回明确 env 错误，启动摘要会脱敏输出非敏感配置；`LOG_LEVEL` / `LOG_FORMAT` 已提供基础 text/json 结构化日志配置，`LOG_CORRELATION_ID` 已提供运行级日志关联 ID，`LOG_TRACEPARENT` 已提供运行级 W3C traceparent / `run_trace_id` 日志字段，API HTTP 边界已提供 `X-Request-ID` / W3C `traceparent` 响应头和带 `request_id` / `trace_id` 的 access log，API 创建的 data sync / backtest / trading task、data sync repair task 和 trading notification 已保存 `requestId` / `traceparent`，data sync / backtest / trading / notify worker 业务日志已在 task 或 delivery 带 request ID / traceparent 时输出 `request_id` / `trace_id`，notification provider 外发 HTTP 请求和 SMTP 邮件头已在 delivery 带 request ID / traceparent 时传播 `X-Request-ID` / `traceparent`，data sync 的 Binance / OKX market HTTP 请求已在 task 带 request ID / traceparent 时传播 `X-Request-ID` / `traceparent`，`DB_MAX_CONNS` / `DB_MIN_CONNS` / DB 连接生命周期配置已提供 PostgreSQL pool 上限，非法日志/资源配置不会回显；`SYNC_HEALTH_ADDR` / `BACKTEST_HEALTH_ADDR` / `TRADING_HEALTH_ADDR` / `NOTIFY_HEALTH_ADDR` 已提供可选 worker `/livez` 进程存活探针和带 PostgreSQL ping / 队列表可读 / 可配置 claim-ready backlog / stale-lease / sync exchange-backoff / sync catalog freshness / notify provider config 检查的 `/readyz` / `/healthz` readiness 探针，`docs/go-command-runbook.md` 已补基础子命令运行手册，`docs/production-runbook.md` 已补 Docker Compose 生产运行入口和备份/恢复操作边界，`scripts/stage8-backup.sh`、`scripts/stage8-backup-restore-drill.sh` 和 systemd timer 模板已补备份/恢复/调度入口，`scripts/stage8-command-config-smoke.sh`、`scripts/stage8-capacity-check.sh` 与 `scripts/stage8-backup-dry-run-smoke.sh` 已进入质量门禁并验证配置错误不泄露 DSN/密码/secret、基础连接/CPU/内存/磁盘/备份保留预算和备份 dry-run；仍缺 data sync market 请求和 notification provider 之外的更广外部系统 trace 传播、独立启动子命令之间的 W3C trace context 自动传播、claim 成功率 / live 外部依赖 readiness 语义、已通过的目标环境备份恢复演练、目标环境备份调度/外部存储监控证据、目标环境负载测试 / sizing 记录和更完整优雅停止证据 |
| Docker Compose | demo | 保留 | 运行形态对，Compose 已透传后台 worker health probe env，`docs/production-runbook.md` 已补启动、健康检查、备份、恢复演练、升级/回滚和事故处理清单，`scripts/stage8-backup.sh` 已补单次备份与保留清理入口，`scripts/stage8-backup-restore-drill.sh` 已补本地 restore drill 脚本，`deploy/systemd/tictick-hi-backup.{service,timer}` 已补目标主机调度模板，`scripts/stage8-smoke.sh` 已覆盖一键构建启动和全链路 smoke，`scripts/stage8-sigterm-smoke.sh` 已覆盖 data sync / backtest / trading / notify 容器 SIGTERM 收尾；仍缺目标环境备份调度安装和外部存储监控证据、已记录且通过的目标环境恢复演练、资源容量策略和外部依赖韧性验证 |
| PostgreSQL migrations | scaffold | 保留后加强 | `0011_domain_constraints.sql` 已补充核心 domain CHECK，`0012_referential_constraints.sql` 已补充核心事实表 FK / composite unique，`0016_worker_lease_constraints.sql` 已补充 worker lease 字段一致性 CHECK，`0017_strategy_intent_parent_constraints.sql` 已补充 `strategy_intents` 新增/更新时的多态父任务归属约束，`0018_strategy_intent_parent_delete_guards.sql` 已补充父任务删除防 orphan 保护，`0019_task_terminal_timestamp_constraints.sql` 已补充任务终态 `finished_at` 一致性约束，`0020_validate_worker_lease_constraints.sql` 已修补历史半截 lease 并 VALIDATE worker lease CHECK，`0021_task_status_transition_guards.sql` 已补充 data sync / backtest / trading 核心状态流转 trigger，`0024_data_sync_repair_source.sql` 已补充补同步任务源任务 FK / 非自引用约束，`0028_data_sync_restart_succeeded.sql` 已补充 data sync succeeded 任务重新启动为 pending/running 的状态约束，`0029_data_sync_soft_delete.sql` 已补充 data sync 任务软删除字段和 cancelled 状态流转，`0030_market_candle_positive_prices.sql` 已补充 `market_candles` 新写入 OHLC 正价格 CHECK（历史行暂不 VALIDATE），`0034_task_request_ids.sql` 已补充 task request ID 字段，`0036_task_traceparents.sql` 已补充 task traceparent 字段，`0037_notification_traceparents.sql` 已补充 notification traceparent 字段；`scripts/stage8-migration-audit.sh` 已进入 Stage 8 smoke 并校验状态流转 trigger 和 repair source 约束/孤儿行；仍缺完整统一状态机、数据迁移/回滚策略和全量历史数据验证 |
| API server | scaffold | 保留后加强 | 已按领域拆分，`/api/candles` 已返回 metadata，数据同步创建和 K 线查询已校验 Binance / OKX 交易对格式，`POST /api/data/tasks`、`POST /api/backtests` 和 `POST /api/trading/tasks` 已强制 exact active `market_instruments` catalog 命中，不命中返回 `market_instrument_not_active` 且不落库，`/api/data/tasks` 返回后端派生 `dataHealth`、任务窗口内（含 start/end 边界和整窗无数据）K 线 `gapSummary`、窗口内历史异常 OHLCV K 线 `dataHealth=invalid`、`invalidSummary`、`GET /api/data/tasks/{id}/invalid-issues` 异常详情列表和补同步来源 `repairSourceTaskId`，`GET /api/data/tasks/{id}/gaps` 可查看任务窗口内前 20 个缺口详情并返回总数/返回数量/修复上限 metadata，`POST /api/data/tasks/{id}/repair-gaps` 可为任务窗口内前 20 个缺口创建并启动带源任务 ID 的补同步任务、跳过同窗口重复任务且返回总数/上限 metadata，`POST /api/data/tasks/{id}/repair-gap` 可为图表单个缺口创建带源任务 ID 的补同步任务，`GET /api/market/candle-gaps` 可按 exchange/symbol/interval 扫描已落库 `market_candles` 全历史相邻缺口并返回扫描窗口、K 线数量、总缺口数、返回数量和 limited metadata，`POST /api/market/candle-gaps/repair` 会验证请求窗口是真实已落库相邻缺口后创建无源补同步任务并对同窗口重复请求返回 `skippedExisting`，`GET /api/market/instruments/status` 返回各交易所 instrument catalog 最近同步状态供研究页和运维上下文使用，回测 / 交易创建已复用策略 schema 校验，系统写请求已有 CSRF 检查，错误响应已统一为 `code/message/error` 且 500 响应不再泄露内部错误；数据同步 retry / command 状态冲突已映射为 `data_sync_retry_requires_failed` / `data_sync_command_invalid_state` 领域错误码；已知 API 资源路径的方法错误会返回 `405 method_not_allowed` 和 `Allow` header；`GET /api/system/api-contract` 已暴露基础 OpenAPI 3.1 request / response schema contract 和 `x-errorCodes` 错误码 catalog；`web/frontend/src/types/api.generated.ts` 已由后端 OpenAPI contract 生成，`scripts/quality-gate.sh` 已纳入前端 API route、核心 TypeScript DTO 字段、生成 DTO staleness、外部 OpenAPI validator 与后端 contract 漂移硬检查；登录和系统管理写操作已有基础操作审计日志；仍缺跨领域错误语义细分和生产级审计边界 |
| 登录会话 | demo | 保留后加强 | HttpOnly session cookie、CSRF double-submit 写保护、登录失败持久化节流、当前操作员 session 列表、基础来源地址 / User-Agent 上下文、非当前 session 撤销和当前操作员自禁用保护已进入 API / 系统管理边界；登录成功 / 失败、退出和会话撤销会进入基础操作审计；仍缺密码策略、RBAC / 更完整自保护规则和生产级设备上下文 |
| 数据同步 worker | demo | 保留后加强 | 能 claim、拉取、upsert 1m K 线并恢复游标，运行中会持续刷新 heartbeat / locked_until，heartbeat 丢失后会停止保存结果；批量拉取结果只按闭合且连续的 open_time 链推进 `last_synced_open_time`，不会把同步游标跨过批次内缺口或未闭合尾部 K 线；一次性有界同步在交易所返回空批次且没有 cursor 时会保存 completed 结果、进入 succeeded、释放 lease、保留任务窗口缺口健康且不伪造 K 线，只有未闭合 K 线的批次不会把有界或无 endTime 的一次性任务误判完成，succeeded 的 active catalog 任务可重新启动为 pending；删除 data sync task 会软删除任务行、置为 cancelled、停用 sync/realtime、释放 lease、从列表/claim/命令入口隐藏，但不删除 `market_candles` 事实数据且删除后不再接受同步结果写入；保存结果只接受 `running`、持有未过期 lease 且 `WorkerID` 匹配 `locked_by` 的任务，保存前会校验 fetched candle series 的任务目标、时间周期、排序、重复、OHLCV decimal / OHLC 正价格 / volume 非负 / 高低价边界，异常 payload 不写库、不推进游标并明确失败；`SaveDataSyncResult` 也会按 `task_id` 读取目标并拒绝 exchange / symbol / interval 不匹配的 candle，并校验 `LastOpenTime` 必须匹配本次 candles 对当前持久化游标可推进的连续链尖端，防止绕过 runner 的错标的写入或虚假游标推进；PostgreSQL + runner 集成测试已覆盖重启遗留过期 running realtime lease 后重新 claim、按持久化游标 overlap 拉取、upsert 去重、推进游标并回到研究页任务列表可观察；临时市场数据错误记录为 retry 并释放 lease，按任务持久化 `next_attempt_at` 退避窗口，并按交易所持久化 `data_sync_exchange_backoffs` 冷却，claim 会跳过未到期任务和 active 冷却交易所；运维健康和数据同步任务 API / 研究页任务表可观察 active exchange backoff 数量、最近重试时间、任务级 `exchangeBackoffUntil` 和脱敏错误；永久失败会停用 sync / realtime 期望；用户可从研究页 retry failed 任务，retry 只接受 failed 状态并清理错误、lease 和退避时间；用户 stop sync / realtime、runner 上下文取消和容器 SIGTERM 会释放 active lease；release / fail / pause 清锁语义已收敛到共享 helper；Binance / OKX public market 请求已有本地固定窗口限流，`hi sync` 中 K 线同步和 instrument catalog 同一进程共享 client 限流器，K 线拉取前会按交易所获取 PostgreSQL advisory lock，锁被占用时释放当前 data sync task lease、跳过交易所请求且不写入失败/退避状态，避免多实例同时拉取同一交易所 K 线；instrument catalog 临时错误会按 `SYNC_FETCH_RETRIES` / `SYNC_RETRY_DELAY` 短重试后写入 `market_instrument_sync_statuses` 并在运维健康中显示单交易所 warning，最近成功超过 24 小时会被运维健康标记为 stale warning；instrument catalog 同步开始前会按交易所获取 PostgreSQL advisory lock，锁被占用时跳过拉取和写入失败状态，避免多实例重复刷新同一 catalog；instrument catalog 变为 inactive / missing 时会保存原 sync/realtime 期望并暂停对应 data sync task，恢复 active 时只恢复这类自动 catalog pause 任务；已提供基于 `market_candles` 的全历史相邻缺口扫描入口，并可从研究页为单个真实缺口排补同步任务，但不会自动批量补全；本地外部临时失败恢复 smoke 已验证 Binance `Retry-After`、OKX `50011` 和 exchange backoff 隔离后可恢复落库，默认 Binance public exchange smoke 也已有 native/ok 证据；仍缺完整统一状态机、真正的分布式 token bucket / 动态交易所额度、OKX 真实外网恢复证据和长期多实例外部网络压测 |
| CandleProvider | demo | 保留后加强 | 已统一 native / 1m 聚合、来源和缺口 metadata，查询 limit 已有显式默认/上限，CandleProvider/repository 入口会先校验 exchange / symbol / interval 必填和 interval 合法，`from/to` 已在 API 和 CandleProvider/repository 入口校验顺序并按 interval 限制最大闭区间跨度，非法 target 或显式窗口会在读取 store 前失败；显式 `from/to` 窗口会把起点到首根 K 线、末根 K 线到终点和整窗无数据识别为缺口，聚合 fallback 会返回 coverage 并标记基础窗口受限，基础 `1m` 聚合窗口已改为最多 288 页 / 1440000 根的有界流式分页聚合，默认最新聚合窗口会按尾部裁剪保留最新 K 线，`scripts/stage1-candle-provider-perf-smoke.sh` 已用真实 PostgreSQL 验证 240000 根 `1m` 聚合成 1000 根 `4h` 的查询边界，PostgreSQL 集成测试已覆盖 6 个并发 `15m` 聚合查询的一致结果、coverage 和 pagination metadata，`/api/candles` 返回窗口级 pagination metadata、opaque `previousCursor/nextCursor` 和当前实际窗口 `from/to/count`，PostgreSQL 集成测试覆盖基础聚合、缺口、请求窗口边界缺口、默认最新窗口查询、latest-before 查询、上一/下一窗口 metadata、超大 limit clamp、repository 入口超大显式窗口拒绝、repository 入口缺 target 拒绝和 runner 侧闭合信号过滤；仍缺长期 soak / 冷缓存 / 真实生产数据分布压测、超过 1440000 根基础 K 线的缓存/分段策略和更多异常数据边界 |
| Binance / OKX K 线 adapter | demo | 保留后加强 | 能拉 K 线，Binance 支持多 base URL fallback，EOF/超时/429/5xx/OKX 50011 已分类为临时错误并由 sync runner 有限重试，临时错误会触发任务级和交易所级退避，错误摘要不泄露完整请求 URL；Binance 默认 client 会在成功读取 exchangeInfo 后解析 `REQUEST_WEIGHT` rateLimits 并切换为本地多窗口 fixed-window 限流，K 线请求仍按 weight=2、exchangeInfo 按 weight=20 计重，自定义 limiter 不被覆盖；OKX history-candles 和 public instruments 按 20 次/2s 本地限流；仍缺 OKX 动态额度、多实例共享额度、真实网络韧性和更完整交易所业务码分类 |
| 研究页 | scaffold | 保留后打磨 | 列表在上、图表在下，任务表格展示后端派生 `dataHealth`、`gapSummary`、`invalidSummary`、同步窗口和交易所退避窗口，可区分正常、同步中、有缺口、失败、暂停、重试中、数据不足和数据异常，并在质量摘要列显示任务窗口内缺口数量/首个缺口和异常数量/首个异常原因，异常任务可打开逐根异常详情弹窗并按异常类型/时间范围筛选和分页查看完整窗口异常；任务行可查看缺口详情弹窗，受限时显示已返回/总数/单次修复上限，也可调用后端 `repair-gaps` 为窗口内缺口批量排补同步任务，补同步任务在列表中可通过 `repairSourceTaskId` 与 `startTime/endTime` 窗口识别；图表 metadata 出现 CandleProvider 缺口时会在 K 线上标记缺口，并可为首个缺口创建并启动补同步任务；如果图表来自已选同步任务且基础周期匹配，修复会优先通过后端单缺口 repair API 并写入 `repairSourceTaskId`；删除任务弹窗已明确删除的是同步任务记录且不会删除已同步 K 线数据，确认后列表刷新并隐藏软删除任务；任务表格错误列、下次重试列、交易所退避列、failed retry 操作和图表高度已有前端约束，任务表外层改为可滚动视口且操作列固定在右侧，避免窄宽度裁掉关键操作；研究页图表面板不再继承全局 `.chart-panel` fixed height / size containment，图表槽改为 CSS 变量控制的固定 viewport 高度，`.research-chart-body` 使用固定 `flex-basis` / `height` / `max-height` 和 `contain: layout paint`，`.research-chart-panel` 覆盖为 `contain: layout paint` 避免 auto 高度被全局 size containment 折叠，研究页工具栏已收敛为紧凑 market strip + 单行可滚动状态摘要，并保留图表刷新 icon；symbol 输入收敛为桌面/窄桌面 `96px`、移动端 `92px`，`stage8-visual-smoke.mjs` 将最大宽度阈值收敛到 `100px`、工具栏控件阈值 `500px`、工具栏高度阈值 `72px`；研究页、回测详情和交易详情复用共享 K 线图表槽，plot 高度收敛为桌面 `clamp(680px, 72dvh, 820px)`、窄桌面 `700px`、移动端 `580px`，上下 padding 归零，左右 gutter 收敛为桌面 `14px/2px`、窄桌面 `12px/2px`、移动端 `10px/2px`，避免首屏图表过窄、左侧贴边和右侧外边距失控；`TradingViewChart` 只观察并读取最近带 `data-chart-viewport="fixed"` 的声明式固定图表槽，不观察传给 lightweight-charts 的 mount canvas，也不响应 `.trading-chart` root / canvas / 内部图表节点的 resize entry，固定槽高度不再信任 `ResizeObserver` content height 或被污染的 `clientHeight`，窗口尺寸不变时拒绝任何固定槽高度变化反馈，即使宽度变化也只更新宽度；lightweight-charts 外层受固定 viewport 尺寸约束，但内部 table / tbody / tr / td / canvas 不再被外部 CSS 强行写成整图宽高，`.tv-lightweight-charts` 根不再被外部 `width/height:100%!important` 强行缩放，避免价格轴、时间轴和主图 canvas 被外部布局规则裁切或拉伸；volume histogram 使用隐藏的 `volume` price scale，避免 overlay scale 撑出额外右侧坐标区；图表 root/canvas/lightweight-charts 外层使用明确 `top/left`，不再用 `inset: 0`，右侧价格轴按视口响应式使用桌面/窄桌面 `56px`、移动端 `54px` minimumWidth，开启 `entireTextOnly` 并把 `rightOffsetPixels` 收敛为 `0px`，价格轴保持完整数值显示，不允许 `k/K/m/M/b/B` 紧凑缩写，坐标轴字号收敛为桌面/窄桌面/移动端 `13px`，默认首屏按主绘图区宽度展示可读数量的最新 K 线，避免窄视口只剩网格或半截价格标签；headless Chrome 桌面、812x1320 窄桌面和移动连续采样会先验证主图 canvas、右侧价格轴和底部时间轴 canvas 均在固定图表槽内，且主图存在可见红/绿市场像素，正常首屏 tv 与固定槽等高且不留下人为缩图留白，外部高度污染后 document、panel、chart body、chart、canvas 高度不增长且不超过 viewport 上限，右侧价格轴超过 `72px`、坐标轴文字墨迹高度低于 `8px`、桌面/窄桌面/移动端高于 `15px`、canvas CSS scale 被明显拉伸/压缩、主图占比低于桌面 `90%` / 窄桌面 `82%` / 移动端 `70%`、最右侧 canvas 未贴住 viewport 右边或主图 canvas 未贴住右侧价格轴会失败，窄桌面还会验证初始首屏不截掉底部时间轴；显示 source / health / base interval / 当前窗口范围和当前数据源全历史缺口扫描摘要，摘要可打开详情弹窗并为单个或当前返回的多个全历史缺口排补同步任务，可通过最新 / 1H / 6H / 1D 时间范围按钮和上一/下一窗口按钮显式请求 K 线窗口，上一/下一优先保留 opaque cursor，旧 `from/to` URL 仍兼容；研究页、回测创建和交易创建的 symbol 输入已从 BTC/ETH 固定白名单收敛为交易所格式校验，并通过 `/api/market/instruments` 读取 PostgreSQL instrument catalog 建议项，前端可手动触发 Binance `/exchangeInfo` 和 OKX public instruments 同步，失败时回退本地建议；研究页会读取 `/api/market/instruments/status` 并在当前数据源和创建任务弹窗里显示所选交易所目录最近成功/失败状态；`/api/market/instruments` 支持按 `status=active/inactive/all` 查询，研究页、回测创建和交易创建在提交前会 exact 查询 catalog 并区分 active、inactive、missing，inactive 会给出明确不可用提示；`hi sync` 长运行模式会按配置后台定时同步 Binance / OKX instrument catalog 并写入 `market_instruments`；创建数据同步任务会先在前端校验 exact active catalog 命中，后端 `POST /api/data/tasks` 也会强制查询 PostgreSQL `market_instruments` active 记录，不命中返回 `market_instrument_not_active`；既有数据同步任务列表会返回并展示 `marketStatus=active/inactive/missing`，非 active 任务的 sync / realtime / retry 启动会被前后端阻止，`hi sync` claim 也只领取 active catalog 任务；catalog 失活时对应 data sync task 会带 market inactive 错误自动暂停并保留原同步期望，恢复 active 时只恢复这类自动暂停任务；但仍缺交易所业务状态细分、跨模块迁移和完整操作语义，图表研究能力仍薄 |
| 策略 registry / runtime | demo | 保留后加强 | 已有策略 schema 校验、默认参数规范化、order / notification intent 和边界门禁，仍缺策略沙箱、参数版本迁移和更多真实策略 |
| 回测 | demo | 保留后加强 | 已通过 CandleProvider 执行、`minute_replay` 以 `1m` 推进，策略输入前会丢弃未闭合 K 线，且 `gap/insufficient/limitedByBaseWindow` 不再进入策略输入；intent / order / result 落库，详情页展示 intent 和买卖点，并采用上方大图表、下方左窄摘要右宽列表的布局；runner 上下文取消和容器 SIGTERM 会释放 active lease 并复位为 pending；撮合模型、费用/滑点曲线、指标体系仍不可信 |
| 交易 runner | demo | 保留后加强 | 已通过 CandleProvider 取 K 线，策略输入前会丢弃未闭合 K 线，且 `gap/insufficient/limitedByBaseWindow` 不再进入策略输入；paper executor 落库 intent / order / execution / position / notification，交易详情页采用上方大图表、下方左窄摘要右宽列表的布局，running task claim 已按 `updated_at` 轮转避免旧任务长期占用队列，用户 pause、runner 上下文取消和容器 SIGTERM 会释放 active lease，live execute 已禁用；通知 intent 可经 local / webhook / email / Telegram / 飞书 provider 投递；仍缺可信风控、完整统一 worker lease 和实盘安全边界 |
| 实盘安全 | demo | 保留后加强 | 新建交易所账号凭据使用 `ENCRYPTION_KEY` + AES-GCM 加密保存，列表/API 不返回明文，live 任务创建校验账号启用和凭据状态；真实 testnet/sandbox live executor、幂等提交和生产密钥管理仍未完成 |
| 通知 | demo | 保留后加强 | NotificationIntent 已进入 notification outbox，`hi notify` 支持 local / webhook-demo / webhook / email / Telegram / 飞书 provider、失败重试和系统页 retry，delivered / failed / retry / runner 上下文取消会通过共享 lease helper 释放 outbox lock；真实 provider 采用 env-reference 凭据模型，密钥不进入 channel target；webhook / Telegram / 飞书支持真实 HTTP POST，email 支持 SMTP；notify 容器 SIGTERM 已由慢 webhook smoke 证明会释放 outbox lock；通道更新/删除、生产级模板/限流/回执、完整统一 worker lease 仍未完成 |
| 前端基础设施 | scaffold | 保留后加强 | Vue/Naive/Pinia/i18n/主题骨架存在，策略任务表单已由 schema 驱动并校验参数，路由页面已懒加载且生产入口 chunk 降到 500 kB 以下；概览页已改为真实聚合视图；研究页、回测详情、交易详情 K 线图表已收敛到共享 `klineChartLayout.css` 固定图表槽契约，复用高度、左右 gutter、内部 chart 填充规则，visual smoke 已新增右侧价格轴必须贴近图表视口边界、最右侧 canvas 必须贴住 viewport 右边界、主图占比、研究页工具栏高度最大 `72px` 的断言，并把 symbol 输入最大宽度阈值收敛到 `100px`、控件组最大宽度 `500px`、右侧价格轴最大宽度 `72px`、坐标轴文字墨迹高度范围收敛为桌面/窄桌面/移动端 `7px` 到 `13px`、图表高度收敛到桌面 `600px+` / 窄桌面 `620px+` / 移动端 `540px+`，防止右侧额外空白、工具栏过宽、坐标轴过小和图表过矮回归；`scripts/stage8-visual-smoke.mjs` 已覆盖当前全部登录后静态路由在 1440/812/390 视口、浅/深主题和 zh-CN/en-US 语言矩阵下的 runtime error、横向溢出、主内容存在性、html lang、顶部导航翻译和明显 i18n key 泄漏，并在存在任务数据时进入回测详情 / 交易详情检查上图表、下双栏布局；`scripts/stage8-state-visual-smoke.mjs` 已用 GET API 拦截覆盖研究、回测、交易、通知、系统和详情页可见空/错误状态在桌面/移动、浅/深主题、中英语言下的状态块可见性、横向溢出和 i18n 泄漏；`routes.test.ts` 会校验新增登录后静态路由必须同步进入 visual smoke；两类浏览器 smoke 已接入 `scripts/stage8-smoke.sh` 默认验收，可用 `STAGE8_BROWSER_SMOKE=0` 在无 Chrome 环境显式跳过；仍缺像素快照基线、动态详情全数据状态、多浏览器视觉回归和 CI 硬门禁，整体业务体验仍需继续打磨 |
| 概览页 | demo | 保留后加强 | 已从现有 API 读取系统健康、数据同步、回测、交易和通知记录，展示关键数量、异常提醒、worker 健康和最近活动；recent facts 和最近活动已有 24H / 7D / 30D 时间窗口筛选；汇总卡片已有到研究、回测、交易、通知和运维健康的操作入口；已有 7D 运行趋势条展示策略意图、订单、通知和失败信号；已有数据质量、自动化链路、执行面和通知投递深度指标；新增监控上下文展示快照时间、数据源降级、趋势覆盖和告警负载；通知、数据质量、自动化链路和回测/交易执行面入口已带状态筛选上下文跳转；仍缺 SLO、告警规则、实时订阅等生产级监控语义 |
| 系统管理 / 运维健康 | demo | 保留后加强 | 操作台账号可创建和启停，且会阻止当前操作员禁用自己；当前操作员 session 可查看来源地址 / User-Agent 并撤销非当前会话，基础操作审计页/API 可查看登录和系统管理写操作，运维健康页/API 展示数据库、api、worker count、heartbeat、locked_until 和 instrument catalog 同步状态；仍缺 RBAC、更完整自保护规则、不可篡改审计和生产监控 |
| 质量门禁 | demo | 保留后加强 | 阶段 0 硬门禁、策略边界检查、API contract route / field drift / generated TypeScript DTO staleness / external OpenAPI validator 检查、Go command config smoke、整体 scaffold 声明检查、完整本地质量门禁 `scripts/full-quality-gate.sh`、GitHub Actions 默认 full gate、独立 Stage 8 heavy smoke workflow、Stage 8 smoke gate（默认串联 full-chain 浏览器 visual / state visual smoke）和 data sync / backtest / trading / notify SIGTERM smoke 已通过；live executor/testnet、完整统一 worker lease、真实通知 provider 的生产启用边界和生产级登录安全作为后续风险审计保留 |

注：模块评级表用于保留主要风险摘要。研究页行中关于“退市/停牌后自动停用既有 data sync task”的旧风险，已在后续“instrument catalog 同步后自动停用非 active 数据同步任务补充”和“instrument catalog 自动暂停恢复语义补充”小节推进；原始交易所 instrument status 可观察已在后续“instrument catalog 交易所原始状态可观察补充”小节推进；仍未关闭的是跨模块迁移/删除处置和完整交易所业务状态语义。

补充：阶段 1 已新增 CandleProvider `invalid` 健康状态、CandleResult `issues` 摘要和任务列表窗口级 `dataHealth=invalid` 统计，用于把历史异常 K 线从 API 500 收敛为研究页列表和图表可观察的数据健康状态；历史行清洗和自动修复仍未关闭。

补充：阶段 1 全历史 invalid repair 已补 PostgreSQL 集成证据：通过全历史 invalid scan 找到 persisted 异常行，排补同步任务后由 `SaveDataSyncResult` 走正常 worker 写回路径覆盖为健康 K 线，随后 `ScanMarketCandleInvalidIssues` 回到无异常；该证据只证明“补同步成功写回时会收敛”，不代表自动清洗历史行或保证交易所一定返回健康数据。

补充：阶段 1 全历史 invalid repair 已补 HTTP API + PostgreSQL 集成证据：真实 API server 使用 PostgreSQL store 登录唯一测试操作员，经 CSRF 写请求排队 invalid repair task，再由 `SaveDataSyncResult` 写回健康 K 线，最后通过 `GET /api/market/candle-invalid-issues` 观察 `TotalCount=0`；该证据证明路由层、认证/CSRF、active instrument 校验和 store 收敛路径可以串起来，但仍不代表自动清洗或交易所数据一定可修复。

补充：阶段 1 全历史 gap repair 已补 HTTP API + PostgreSQL 集成证据：真实 API server 使用 PostgreSQL store 登录唯一测试操作员，经 CSRF 写请求对 `GET /api/market/candle-gaps` 发现的相邻缺口排队 repair task，再由 `SaveDataSyncResult` 写回缺失 K 线，最后通过同一路由观察 `TotalCount=0` 且窗口 K 线数量补齐；该证据证明全历史缺口路由、认证/CSRF、active instrument 校验和 worker 写回收敛路径可以串起来，但仍不代表自动批量补全或交易所一定返回缺失数据。

补充：阶段 1 data sync task 窗口 gap repair 已补 HTTP API + PostgreSQL 集成证据：真实 API server 使用 PostgreSQL store 登录唯一测试操作员，经 CSRF `POST /api/data/tasks` 创建带 start/end 窗口的源同步任务，由 `SaveDataSyncResult` 写入 0、1、5 分钟 K 线形成任务窗口内缺口，再通过 `GET /api/data/tasks/{id}/gaps` 观察缺口，`POST /api/data/tasks/{id}/repair-gap` 排队带 `repairSourceTaskId` 的补同步任务，最后由 `SaveDataSyncResult` 写回 2、3、4 分钟 K 线，并通过 `GET /api/data/tasks` 和 `/gaps` 观察源任务 `dataHealth=ok` 且缺口消失；该证据证明任务窗口缺口路由、认证/CSRF、active instrument 校验、源任务关联和 worker 写回收敛路径可以串起来，但仍不代表自动批量补全或交易所一定返回缺失数据。

补充：阶段 1 data sync task 窗口 invalid repair 已补 HTTP API + PostgreSQL 集成证据：真实 API server 使用 PostgreSQL store 登录唯一测试操作员，经 CSRF `POST /api/data/tasks` 创建带 start/end 窗口的源同步任务，由 `SaveDataSyncResult` 写入 0、1、3 分钟健康 K 线，再注入 2 分钟 legacy invalid K 线形成任务窗口内异常，通过 `GET /api/data/tasks/{id}/invalid-issues` 观察异常，`POST /api/data/tasks/{id}/repair-invalid-issues` 排队带 `repairSourceTaskId` 的补同步任务，最后由 `SaveDataSyncResult` 写回 2 分钟健康 K 线，并通过 `GET /api/data/tasks` 和 `/invalid-issues` 观察源任务 `dataHealth=ok` 且异常消失；该证据证明任务窗口异常路由、认证/CSRF、active instrument 源任务创建校验、源任务关联和 worker 写回收敛路径可以串起来，但仍不代表自动清洗历史异常行或交易所一定返回健康数据。

补充：阶段 1 data sync task repair active catalog 守卫已补证据：`RepairDataSyncTaskGaps`、`RepairDataSyncTaskGap` 和 `RepairDataSyncTaskInvalidIssues` 在锁定源任务并校验同步周期后，会在同一事务内要求源任务 exchange/symbol 仍存在 active `market_instruments` 记录；非 active / missing 时返回 `market_instrument_not_active`，不会扫描缺口/异常或插入补同步任务。HTTP API 测试覆盖三个 repair 入口的 inactive 源任务拒绝和不新增 repair task；PostgreSQL 集成测试覆盖直接调用 store 也返回同一领域错误且不落库。该证据只收紧任务窗口 repair 排队边界，不代表自动批量修复、交易所一定返回缺失数据或实盘交易能力已建立。

补充：阶段 1 CandleProvider repository 范围守卫已补证据：`CandleProvider.GetCandles` 会在读取 native store 前调用 `ValidateCandleQueryRange`，API、回测 runner、交易 runner 和任何直接 repository 调用都共享同一显式 `from/to` 顺序与最大跨度校验；单元测试覆盖超大窗口不会触碰 store，PostgreSQL 集成测试覆盖 `Store.GetCandles` 对超大显式窗口直接返回错误。该证据只证明查询入口自守，不代表长期冷缓存/真实生产分布压测已关闭。

补充：阶段 1 CandleProvider repository target 守卫已补证据：`CandleProvider.GetCandles` 会在读取 native store 前调用 `ValidateCandleQueryTarget`，直接 repository 调用缺少 exchange / symbol / interval 或 interval 非法时会直接返回错误，不再静默查成空结果；单元测试覆盖缺 symbol 不触碰 store，PostgreSQL 集成测试覆盖 `Store.GetCandles` 对缺 target 直接返回错误。该证据只证明查询入口 target 自守，不替代 API 层交易所 symbol 格式校验、active instrument catalog 校验或真实生产数据压测。

补充：阶段 1 研究页、回测详情和交易详情的 K 线图表布局在 2026-07-02 以 `tictickbot` / `tictick-lite` 的正常 K 线比例继续校准；当前有效布局约束以 `klineChartLayout.css`、`ResearchPage.css`、`detailChartLayout.css`、`scripts/stage8-visual-smoke.mjs` 和 `scripts/research-chart-height-smoke.mjs` 为准：研究页主工具栏 symbol 输入为桌面/窄桌面 `96px`、移动端 `92px`，桌面工具栏采用左侧 market strip + 右侧单行可滚动状态摘要的一行工作台布局，窄屏再堆叠；图表左/右 gutter 为桌面 `14px/2px`、窄桌面 `12px/2px`、移动端 `10px/2px`；plot 高度为桌面 `clamp(680px, 72dvh, 820px)`、窄桌面 `700px`、移动端 `580px`，上下 padding 归零；共享 `TradingViewChart` 坐标轴字号收敛为桌面/窄桌面/移动端 `13px`，右侧价格轴 minimumWidth 为桌面/窄桌面 `56px`、移动端 `54px`，全局 localization 和 K 线 series `priceFormat` 都使用完整数值 formatter，继续禁止 `k/K/m/M/b/B` 缩写；visual smoke 断言 symbol 最大宽度 `100px`、工具栏控件最大宽度 `500px`、工具栏高度最大 `72px`、右侧价格轴最大宽度 `72px`、坐标轴文字墨迹高度范围为桌面/窄桌面/移动端 `8px` 到 `15px`、canvas CSS scale 不能被明显拉伸/压缩、主图 canvas 右边界贴住右侧价格轴左边界、最右侧 canvas 贴住 viewport 右边界，详情页下方摘要列保持 `minmax(220px, 260px)`。

补充：阶段 1 K 线图表坐标轴正常范围在 2026-07-02 继续收敛；本段覆盖前述过小、缩写和巨型字号旧合同。参考 `tictickbot` / `tictick-lite` 的普通交易图表比例，当前有效合同为共享 `TradingViewChart` 坐标轴字号收敛为桌面/窄桌面/移动端 `13px`，右侧价格轴 minimumWidth 为桌面/窄桌面 `56px`、移动端 `54px`，价格 formatter 继续输出完整数值且禁止 `k/K/m/M/b/B` 缩写；visual smoke 仍要求右侧价格轴最大宽度 `72px`、主图宽度占比下限桌面 `0.90` / 窄桌面 `0.82` / 移动端 `0.70`，并把坐标轴文字墨迹高度范围为桌面/窄桌面/移动端 `8px` 到 `15px`，过小和过大的坐标轴都会失败，canvas CSS scale 被明显拉伸/压缩也会失败。该证据只修正 K 线图表视觉合同，不改变 CandleProvider、data sync、回测、交易或实盘能力等级。

补充：阶段 1 K 线图表内部缩放回归在 2026-07-02 继续收紧；共享 `TradingViewChart` 不再通过 `MutationObserver` 改写 lightweight-charts 生成的内部 table / canvas 宽高，避免自恢复逻辑把浏览器中的 price-axis canvas 放大或缩小成异常字号。组件现在只观察外部声明式 fixed viewport，并用该 viewport 的 `clientWidth/clientHeight` 或声明 CSS 高度调用 `createChart` / `resize`；`.tv-lightweight-charts` 根节点仍只保留最大尺寸约束和外层 fixed viewport 裁剪，不强制内部 table / canvas 几何。`scripts/stage8-visual-smoke.mjs` 和 `scripts/research-chart-height-smoke.mjs` 继续按运行态 canvas 识别右侧价格轴、主图和底部时间轴，并断言价格轴宽度、轴文字墨迹高度、canvas CSS scale 和主图/价格轴贴合关系。该证据只修正共享图表视觉与防回归合同，不改变 CandleProvider、data sync、回测、交易或实盘能力等级。

补充：阶段 1 K 线图表坐标轴与本地 8080 产物在 2026-07-02 完成回归验证；已执行 `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts src/pages/ResearchPage.layout.test.ts src/pages/DetailPages.layout.test.ts`、`pnpm --dir web/frontend run typecheck`、`pnpm --dir web/frontend run test`、`pnpm --dir web/frontend run build`、`go test ./...`、`go vet ./...`、`scripts/check-file-size.sh`、`scripts/check-research-chart-layout.sh`、`scripts/quality-gate.sh` 和 `git diff --check`。本地 8080 API 容器已通过 `docker compose up -d --build api` 重建，`curl -fsS http://127.0.0.1:8080/readyz` 返回 ok，`/research` 已加载新构建 `index-DJ1jHf_m.js`；`BASE_URL=http://127.0.0.1:8080 SMOKE_SAMPLES=4 SMOKE_INTERVAL_MS=120 SMOKE_SETTLE_MS=900 SMOKE_TOTAL_TIMEOUT_MS=180000 node scripts/research-chart-height-smoke.mjs` 通过 1440x900、2048x1152、812x1320、390x844 四视口稳定性检查，`BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=800 SMOKE_TOTAL_TIMEOUT_MS=300000 node scripts/stage8-visual-smoke.mjs` 通过桌面/窄桌面/移动、明暗主题、中英文共 12 组，每组 14 页。该证据只证明当前前端产物的图表布局和轴线没有复现过小、巨型、缩写或右侧额外空白问题，不代表实盘交易、私有交易所 API、订单幂等或长期生产运行边界已经完成。

补充：阶段 1 研究页 CandleProvider coverage 可观察性在 2026-07-01 继续收紧；研究页图表 metadata 不再只在 `limitedByBaseWindow=true` 时展示覆盖度，`health=gap/invalid/insufficient` 的非 OK `CandleResult` 也会显示 K 线返回/请求覆盖摘要；当后端返回 `requiredBaseCandles` 和 `returnedBaseCandles` 时，状态栏同步显示基础 K 线覆盖度，例如页面边界缺口可观察为请求 85 根返回 84 根、基础 5099/5100。该规则抽到 `researchCandleCoverage.ts` 并由单元测试覆盖健康完整结果隐藏、非受限 gap 显示、受限窗口显示、基础覆盖字段识别和 `research.coverageSummary` / `research.baseCoverage` 文案 key；`ResearchPage.layout.test.ts` 同步约束页面必须使用该 coverage helper。已执行 `pnpm --dir web/frontend exec vitest run src/pages/researchCandleCoverage.test.ts src/pages/ResearchPage.layout.test.ts`、`pnpm --dir web/frontend run typecheck`、`pnpm --dir web/frontend run test`、`pnpm --dir web/frontend run build`。该证据只补前端可观察性，不改变 CandleProvider 聚合算法、API contract、补同步调度或模块等级。

补充：阶段 1 研究页 CandleProvider invalid issue 可观察性在 2026-07-01 继续收紧；`/api/candles` 返回 `health=invalid` 和 `issues` 时，研究页图表状态栏不再只显示异常 K 线时间，会同步显示后端 issue 的可读原因。前端已为标准 OHLCV 异常和 CandleProvider 聚合链路使用的 `invalid_native_series` / `invalid_aggregation_base_series` 增加中英文文案；未知 issue code 回退后端 message，避免把原始 code 直接暴露给用户。`dataApi.getCandles` 单元测试覆盖 CandleProvider `issues` metadata 不丢失，`ResearchPage.layout.test.ts` 约束图表状态栏必须使用 `invalidIssueLabel`、`research.candleIssueNoTime` 和对应 i18n key。已执行 `pnpm --dir web/frontend exec vitest run src/services/api/data.test.ts src/pages/ResearchPage.layout.test.ts`、`pnpm --dir web/frontend run typecheck`、`pnpm --dir web/frontend run test`、`pnpm --dir web/frontend run build`、`go test ./...`、`go vet ./...`、`scripts/quality-gate.sh` 和 `git diff --check`。该证据只补图表异常原因可观察性，不改变 CandleProvider 校验算法、repair API、补同步调度或模块等级。

补充：阶段 1 研究页 CandleProvider invalid issue 修复入口在 2026-07-01 继续收紧；当 `/api/candles` 返回带 `openTime` 的首个异常 K 线 issue 时，研究页图表状态栏会显示“修复首个异常”操作，调用既有全历史异常 repair API `POST /api/market/candle-invalid-issues/repair`，由后端按 `exchange/symbol/baseInterval/openTime` 验证并创建补同步任务或返回重复排队结果；前端成功后刷新任务列表和 K 线窗口，并启动同一套有界 repair task polling，同时展示 `createdTasks` / `skippedExisting` repair result tags。没有 `openTime` 的异常只展示原因，不显示 repair 按钮，避免前端构造不可验证窗口。已执行 `pnpm --dir web/frontend exec vitest run src/composables/researchInvalidIssueRepairActions.test.ts src/pages/ResearchPage.layout.test.ts`、`pnpm --dir web/frontend run typecheck`、`pnpm --dir web/frontend run test`、`pnpm --dir web/frontend run build`、`go test ./...`、`go vet ./...`、`scripts/check-file-size.sh`、`scripts/quality-gate.sh`、`git diff --check`、`docker compose up -d --build api`、`curl -fsS http://127.0.0.1:8080/readyz`、`BASE_URL=http://127.0.0.1:8080 SMOKE_SAMPLES=4 SMOKE_INTERVAL_MS=120 SMOKE_SETTLE_MS=1000 SMOKE_TOTAL_TIMEOUT_MS=180000 node scripts/research-chart-height-smoke.mjs` 和 `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=800 SMOKE_TOTAL_TIMEOUT_MS=300000 node scripts/stage8-visual-smoke.mjs`。该证据只补图表异常排队补同步入口，不代表异常已被清洗，不改变 CandleProvider 校验算法、后端 repair API、data sync worker 调度、回测/交易能力或实盘能力。

补充：阶段 1 研究页 repair 后 K 线收敛观察在 2026-07-01 继续收紧；研究页图表首个缺口修复、图表首个异常修复、任务窗口缺口修复、任务窗口异常修复、全历史缺口修复和全历史异常修复会把本次 repair result 中的 `createdTasks` 交给统一有界 polling。polling 每轮刷新任务列表；如果被跟踪的补同步任务全部进入 `succeeded/failed/cancelled/paused`，会立即刷新一次 K 线并停止；如果一直没有进入终态，会在有界尝试耗尽后刷新一次 K 线作为兜底。`loadTasks` 返回最新任务列表供 polling 判断，普通命令入口仍只等待刷新完成，不消费返回值。已执行 `pnpm --dir web/frontend exec vitest run src/composables/useResearchRepairTaskPolling.test.ts src/pages/ResearchPage.layout.test.ts src/components/research/MarketCandleGapTag.test.ts src/components/research/MarketCandleInvalidIssueTag.test.ts src/components/research/ResearchTaskInvalidIssueModal.test.ts src/composables/researchInvalidIssueRepairActions.test.ts`、`pnpm --dir web/frontend run typecheck`、`pnpm --dir web/frontend run test`、`pnpm --dir web/frontend run build`、`go test ./...`、`go vet ./...`、`scripts/check-file-size.sh`、`scripts/quality-gate.sh`、`git diff --check`、`docker compose up -d --build api` 和 `curl -fsS http://127.0.0.1:8080/readyz`。该证据只补前端收敛观察，不保证补同步一定成功，不改变后端 repair API、data sync worker 调度、CandleProvider 算法、回测/交易能力或实盘能力。

补充：阶段 1 研究页 repair result 整体收敛状态在 2026-07-01 继续收紧；共享 `MarketRepairResultTags` 不再只展示 repair summary 和逐个补同步窗口，还会基于本次 `createdTasks` 的最新任务 `status` / `dataHealth` 显示整体状态：仍有补同步执行中、补同步窗口已恢复正常、补同步已结束但仍有失败或异常、补同步结束后仍需复查数据健康。所有复用该组件的图表缺口、图表异常、任务窗口缺口/异常、全历史缺口/异常 repair 结果会自动获得同一收敛状态标签。已执行 `pnpm --dir web/frontend exec vitest run src/components/research/MarketRepairResultTags.test.ts src/pages/ResearchPage.layout.test.ts`、`pnpm --dir web/frontend run typecheck`、`scripts/check-file-size.sh` 和 `git diff --check`。该证据只补 repair 结果可读性，不新增真实修复能力，不改变后端 repair API、data sync worker、CandleProvider 或模块等级。

补充：阶段 1 研究页图表 repair 当前窗口健康结果在 2026-07-01 继续收紧；图表首个缺口 repair 和图表首个异常 repair 的 `MarketRepairResultTags` 会接收最新 `CandleResult`，并在 repair result 区域直接显示当前图表窗口是否已经正常、仍有缺口、仍有异常或处于其它数据健康状态。该标签只在传入当前图表 `CandleResult` 的图表 repair 上展示，任务窗口和全历史 repair 仍保留任务级收敛语义，避免把局部图表窗口误写成全历史健康。已执行 `pnpm --dir web/frontend exec vitest run src/components/research/MarketRepairResultTags.test.ts src/pages/ResearchPage.layout.test.ts`、`pnpm --dir web/frontend run typecheck`、`pnpm --dir web/frontend run test`、`pnpm --dir web/frontend run build`、`go test ./...`、`go vet ./...`、`scripts/check-file-size.sh`、`scripts/quality-gate.sh`、`git diff --check`、`docker compose up -d --build api` 和 `curl -fsS http://127.0.0.1:8080/readyz`。该证据只把 `/api/candles` 返回的当前窗口健康结果接入 repair result 可观察性，不改变后端 repair API、CandleProvider 算法、data sync worker 或模块等级。

补充：阶段 1 任务窗口缺口 repair 详情收敛观察在 2026-07-01 继续收紧；研究页任务窗口缺口弹窗发起 `repair-gaps` 后，统一有界 repair task polling 在补同步任务进入终态或轮询耗尽时不再只刷新图表 K 线，也会在弹窗仍指向同一个源任务时重新拉取该任务的缺口详情，并保留本次 repair result 标签，避免用户只能看到“已排队”却看不到当前任务窗口缺口是否已经收敛。已执行 `pnpm --dir web/frontend exec vitest run src/pages/ResearchPage.layout.test.ts src/composables/useResearchRepairTaskPolling.test.ts src/composables/researchRepairPollingRefresh.test.ts`、`pnpm --dir web/frontend run typecheck`、`pnpm --dir web/frontend run test`、`pnpm --dir web/frontend run build`、`go test ./...`、`go vet ./...`、`scripts/check-file-size.sh`、`scripts/quality-gate.sh` 和 `git diff --check`。该证据只补任务窗口缺口 repair 的前端收敛观察，不保证补同步任务一定成功，不改变后端 repair API、data sync worker 调度、CandleProvider 算法、回测/交易能力或实盘能力。

补充：阶段 1 全历史缺口/异常 repair 终态重扫在 2026-07-01 继续收紧；研究页全历史缺口标签和全历史异常标签在本次 repair result 的 `createdTasks` 全部进入 `succeeded/failed/cancelled/paused` 后，会基于最新任务列表自动重扫对应全历史 gap / invalid scan，并保留本次 repair result 标签；运行中或缺失的补同步任务不会触发重扫，同一批 repair 只触发一次终态重扫。已执行 `pnpm --dir web/frontend exec vitest run src/components/research/MarketCandleGapTag.test.ts src/components/research/MarketCandleInvalidIssueTag.test.ts src/composables/researchRepairTaskSettle.test.ts src/pages/ResearchPage.layout.test.ts`、`pnpm --dir web/frontend run typecheck`、`pnpm --dir web/frontend run test`、`pnpm --dir web/frontend run build`、`go test ./...`、`go vet ./...`、`scripts/check-file-size.sh`、`scripts/quality-gate.sh` 和 `git diff --check`。该证据只补全历史扫描在补同步任务终态后的前端收敛观察，不保证补同步任务成功，不改变后端 repair API、data sync worker 调度、CandleProvider 算法、回测/交易能力或实盘能力。

补充：阶段 1 任务窗口异常 repair 详情终态重拉在 2026-07-02 继续收紧；研究页任务窗口异常详情弹窗在本次 `repair-invalid-issues` 返回的 `createdTasks` 全部进入 `succeeded/failed/cancelled/paused` 后，会基于当前任务、异常类型筛选、时间筛选和分页重新拉取异常详情，并保留本次 repair result 标签；补同步任务仍在运行或缺失时不会提前重拉，同一批 repair 只触发一次终态重拉。已执行 `pnpm --dir web/frontend exec vitest run src/components/research/ResearchTaskInvalidIssueModal.test.ts src/composables/researchRepairTaskSettle.test.ts`、`pnpm --dir web/frontend run typecheck`、`pnpm --dir web/frontend run test`、`pnpm --dir web/frontend run build`、`go test ./...`、`go vet ./...`、`scripts/check-file-size.sh`、`scripts/quality-gate.sh` 和 `git diff --check`。该证据只补任务窗口异常 repair 的前端收敛观察，不保证补同步任务成功，不改变后端 repair API、data sync worker 调度、CandleProvider 算法、回测/交易能力或实盘能力。

补充：阶段 1 研究页 inactive market 操作语义在 2026-07-01 继续收紧；数据同步任务表对 `marketStatus!=active` 的任务不再只禁用 start sync / start realtime，也会禁用会重新排队 worker 的 failed retry 和任务窗口 gap repair 操作，并用同一个“市场非 active”原因解释；查看图表、查看缺口/异常详情和停止类动作仍保留。API 层补充路由测试覆盖 inactive 任务的 sync/start、realtime/start 和 retry 均返回 `market_instrument_not_active` 且不变更任务状态，前端组件测试覆盖 inactive gap failed task 的 retry / repair / start realtime 不发事件但仍可查看缺口。该证据只收紧研究页操作语义和后端守卫回归，不改变 repair 算法、catalog 同步策略、CandleProvider 或模块等级。

补充：阶段 1 API active catalog 错误语义在 2026-07-01 继续收紧；`POST /api/data/tasks`、`POST /api/backtests`、`POST /api/trading/tasks`、全历史缺口/异常 repair API，以及已有 data sync task 的 sync / realtime / retry / repair 命令仍保持 `market_instrument_not_active` 错误码，但用户可见 message 会区分 `market instrument is inactive in catalog` 和 `market instrument is missing from catalog`，不再把 inactive 与 missing 都写成泛化不可用。创建和 repair 主动守卫会用 `ListMarketInstruments(status=all)` 做 exact symbol 诊断；已有任务命令会从任务列表的 `marketStatus` 派生错误文案；诊断失败时回退原泛化文案，不泄露内部 store 错误。测试覆盖数据同步创建、已有任务 inactive/missing 命令、任务 repair、回测创建、交易创建、全历史 gap batch/single repair 和 invalid repair 的错误文案与不落库/不变更状态语义；已执行 `go test ./...`、`go vet ./...`、`pnpm --dir web/frontend run typecheck`、`pnpm --dir web/frontend run test`、`pnpm --dir web/frontend run build`、`scripts/quality-gate.sh` 和 `git diff --check`。该证据只收紧 API 可诊断性，不改变 catalog 同步策略、data sync worker 状态机、repair 算法或模块等级。

补充：阶段 1 真实 public exchange smoke 在 2026-07-01 继续收紧；`real_exchange_data_sync_integration_test.go` 不再要求真实交易所 public K 线请求第一次运行 runner 必须成功，遇到 temporary EOF / retrying / exchange backoff 时会按任务 `nextAttemptAt` 和 `exchangeBackoffUntil` 做最多 6 次有界恢复尝试，最终仍必须通过 `/api/candles` 读回 3 根 healthy native `1m` K 线才算通过。默认 Binance public smoke 通过；OKX opt-in 在当前环境连续 EOF，直接 `curl https://www.okx.com/api/v5/market/history-candles?...` 也返回 TLS `SSL_ERROR_SYSCALL`，因此该项记录为外部连通性风险，不能作为 OKX 真实外网成功证据。该补充只增强真实 public market data smoke 的恢复语义，不代表私有交易 API、live executor、实盘下单或长期外网压测已完成。

补充：阶段 1 OKX public market 业务码退避传递在 2026-07-01 继续收紧；OKX candles / instruments 在 HTTP 200 但响应体 `code=50011` 时仍按临时限流错误处理，并会读取同一响应头中的 `Retry-After` 传给 `exchange.NewTemporaryErrorWithRetryAfter`，使 data sync runner 复用现有 `RetryAfter` 退避路径设置任务级 `next_attempt_at` 和交易所 backoff。单元测试覆盖 candles 与 instruments 的 `50011 + Retry-After` 均可通过 `exchange.RetryAfter` 读回等待时间，并回归 `TestRunnerUsesRetryAfterForTemporaryFetchError`。该证据只收紧 OKX public market data 恢复语义，不实现 OKX 动态额度、多实例共享额度、真实外网成功证据或任何实盘能力。

补充：阶段 1 API 级临时 public market 失败恢复在 2026-07-01 增加确定性证据；`TestIntegrationDataSyncRouteRecoversAfterTemporaryPublicMarketError` 通过真实 API handler 创建并启动 data sync task，使用本地 Binance-compatible K 线 endpoint 第一次返回 `429 Retry-After`，验证任务进入 `pending/retrying`、记录任务级 retry 和 exchange backoff、错误摘要不泄露请求路径/交易对 query，随后按 `nextAttemptAt` 恢复 runner，写入 PostgreSQL，任务回到 `succeeded/dataHealth=ok`，并通过 `/api/candles` 读回 3 根 `source=native`、`health=ok` 的 `1m` K 线。该测试只证明 public market data 临时失败恢复链路，不代表真实 OKX 外网、长期 soak、多实例共享额度或任何实盘交易能力。

补充：阶段 1 全历史缺口批量 repair API contract 覆盖在 2026-06-30 继续收紧；`POST /api/market/candle-gaps/repair-batch` 已由前端 `dataApi.repairMarketCandleGaps` 使用、后端 OpenAPI contract 声明并由 handler 测试覆盖，本轮把该路由补进 `TestAPIContractCoversCurrentFrontendRoutes` 的前端路由清单，并补进 `TestAPIMethodNotAllowedContracts`，确保未来不会出现前端可调用但 contract / 405 Allow gate 漏检的隐藏写路由。该证据只补契约覆盖，不改变批量 repair 行为、单次上限、active catalog 边界或 data sync worker 调度语义。

### 阶段 1 研究页图表缺口修复入口收敛补充

执行日期：2026-06-30

目标等级：scaffold。

范围内：

- 研究页图表“修复首个缺口”在没有选中源 data sync task 时，不再由前端手工创建普通同步任务并启动 sync。
- 该入口改为调用后端全历史 market gap repair API，由后端验证请求窗口是真实已落库相邻缺口，并复用 `skippedExisting` / `createdTasks` / `repairLimit` 的返回语义。
- 已选中且匹配当前 market/base interval 的源任务时，仍优先调用任务窗口 `repair-gap` API，保留 `repairSourceTaskId`。
- 前端单元测试覆盖无源任务图表缺口修复必须走 `repairMarketCandleGap`，且不再调用 `createTask` / `setSync`。

范围外：

- 不实现自动批量修复所有全历史缺口。
- 不新增后端 repair API 或改变补同步任务调度语义。
- 不推进实盘交易所私有 API、live executor 或订单提交。

剩余风险：

- 该入口只修复图表首个缺口，仍依赖用户手动触发和后续 data sync worker 成功写回。
- 真实交易所返回缺失窗口为空或异常时，仍需要现有失败/重试/数据健康路径暴露。

### 阶段 1 研究页全历史异常修复反馈补充

执行日期：2026-06-30

目标等级：scaffold。

范围内：

- 全历史异常 K 线标签在用户对当前返回的异常排队补同步后，会重新调用全历史 invalid scan，刷新当前 market 的异常扫描状态。
- 重新扫描时保留本次 repair result 展示，用户仍能看到匹配数量、创建数量、跳过数量、修复上限和创建出的补同步任务窗口。
- 前端组件测试覆盖 invalid repair 成功后会触发第二次 `scanMarketCandleInvalidIssues`，并保留 repair result tags。
- 前端组件测试进一步覆盖首次扫描有异常、repair 后第二次扫描返回健康时，弹窗切换为“全历史未检测到异常 K 线”，同时仍显示本次 repair result 和补同步任务窗口。

范围外：

- 不把“排队补同步”伪装成异常已经修复；最终收敛仍依赖 data sync worker 写回健康 K 线。
- 不改变后端 invalid repair API、扫描算法或 worker 调度。
- 不推进实盘交易所私有 API、live executor 或订单提交。

剩余风险：

- 如果补同步任务尚未执行，重新扫描仍可能显示原异常，这是正确但对用户仍需要进一步的运行状态解释。
- 当前仍缺浏览器像素快照和真实多浏览器矩阵验证。

### 阶段 1 instrument catalog 同步调度锁补充

执行日期：2026-06-30

目标等级：scaffold。

范围内：

- `hi sync` 后台 instrument catalog 同步在每个交易所刷新前获取 PostgreSQL advisory lock。
- 同一交易所锁已被其他实例持有时，本实例跳过拉取、跳过替换 `market_instruments`、也不写入失败状态。
- advisory lock 由持锁连接负责释放；释放失败时销毁持锁连接，避免锁残留在连接池。
- 单元测试覆盖 runner 锁占用跳过语义；PostgreSQL 集成测试覆盖同交易所互斥、不同交易所可并行、释放后可重获锁。

范围外：

- 不实现跨实例共享交易所请求额度。
- 不引入 sync history 表或生产调度器。
- 不推进实盘交易所私有 API。

剩余风险：

- 该锁只约束 instrument catalog 刷新，不约束 K 线同步 worker 的交易所请求额度。
- 仍缺真实外部交易所网络恢复压测和多实例长期运行证据。

### 阶段 1 K 线同步交易所 fetch 锁补充

执行日期：2026-06-30

目标等级：scaffold。

范围内：

- data sync runner 在调用交易所 K 线接口前按交易所获取 PostgreSQL advisory lock。
- 同一交易所 fetch 锁已被其他实例持有时，本实例释放当前 data sync task lease，跳过交易所请求、跳过保存结果、也不写入 failed / retry / exchange backoff 状态。
- fetch 锁获取出现基础设施错误时，释放当前 task lease 并让 runner 返回错误，不把该任务标记为永久失败。
- fetch 锁和 instrument catalog sync 锁使用不同 key 前缀，互不阻塞。
- 单元测试覆盖锁占用和锁获取错误语义；PostgreSQL 集成测试覆盖同交易所互斥、不同交易所可并行、与 catalog 锁互不阻塞、释放后可重获锁。

范围外：

- 不实现分布式 token bucket。
- 不动态读取 Binance `rateLimits` 或 OKX 额度。
- 不做真实外部交易所长期压测。

剩余风险：

- 当前只把同一交易所 K 线 fetch 串行化，降低多实例并发放大风险；它不是完整共享额度系统。
- 释放 task lease 后仍依赖下一轮 claim 调度，不提供公平排队或容量控制。

### 阶段 1 K 线同步 fetch 锁竞争可观察性补充

执行日期：2026-06-30

目标等级：scaffold。

范围内：

- data sync runner 在交易所 fetch advisory lock 被占用时记录按交易所聚合的跳过次数和最近跳过时间。
- 锁占用跳过统计落库到 `data_sync_exchange_fetch_lock_skips`，同 exchange 多次记录会递增 `skip_count`，`last_skipped_at` 只保留较新的时间。
- `GET /api/system/health` 的 `sync-worker` 服务返回 `fetchLockSkipCount` 和 `lastFetchLockSkippedAt`。
- 运维健康页展示 K 线 fetch 锁跳过次数和最近跳过时间。
- 单元测试覆盖 runner 锁占用记录、基础设施锁错误不记录锁占用跳过；PostgreSQL 集成测试覆盖统计递增、最近时间和系统健康字段；前端测试覆盖运维页展示。

范围外：

- 不实现分布式 token bucket。
- 不动态读取 Binance `rateLimits` 或 OKX 额度。
- 不做真实外部交易所长期压测。
- 不推进实盘交易所私有 API。

剩余风险：

- 该指标只证明锁竞争可观察，不提供共享交易所额度、调度公平性或容量自适应。
- 仍缺真实外部交易所网络恢复压测和多实例长期运行证据。

### 阶段 1 Binance exchangeInfo 动态 request weight 限流补充

执行日期：2026-06-30

目标等级：scaffold。

范围内：

- `internal/exchange` 增加本地多窗口 fixed-window rate limiter，单次请求会同时占用所有窗口；任一窗口容量不足时不会部分占用其它窗口。
- Binance market client 默认配置在成功读取 `/api/v3/exchangeInfo` 后解析 `rateLimits` 中的 `REQUEST_WEIGHT` 窗口，并把后续 public market 请求切换到交易所返回的本地多窗口限流；刚完成的 exchangeInfo 请求 weight=20 会计入新窗口初始用量。
- Binance K 线请求继续按 weight=2、exchangeInfo 按 weight=20 计重。
- 显式注入的自定义 `RateLimiter` 不会被 exchangeInfo 动态配置覆盖，保留测试和运维覆盖入口。
- 单元测试覆盖多窗口等待、超重拒绝、初始用量、阻塞时不部分预留、Binance 动态 limiter 生效和自定义 limiter 不被覆盖。

范围外：

- 不实现跨实例共享 token bucket。
- 不实现 OKX 动态额度。
- 不读取或提交任何交易所私有 API。
- 不推进 live executor、订单提交、幂等键或实盘安全边界。

剩余风险：

- 该能力只在单进程内使用交易所返回的 public request weight 窗口，不提供多实例共享额度或公平调度。
- exchangeInfo 成功前仍使用保守默认窗口；真实网络韧性和更完整交易所业务码分类仍未关闭。

### 阶段 1 HTTP Retry-After 退避传递补充

执行日期：2026-06-30

目标等级：scaffold。

范围内：

- 临时错误模型支持可选 `Retry-After` 等待时间，并可通过 `exchange.RetryAfter` 在错误链中读取。
- HTTP 状态错误会解析 `Retry-After` 秒数和 HTTP-date；无效、空值、零值或过期时间不会生成等待时间。
- Binance 多 base URL fallback 会在所有 endpoint 均为临时错误时保留最大的 `Retry-After`。
- OKX public candles / instruments 的 HTTP 429 / 5xx 临时错误会保留 `Retry-After`。
- data sync runner 遇到带 `Retry-After` 的临时 fetch error 时不做本地快速重试，释放任务并把任务级 `next_attempt_at` 和交易所 backoff 至少推迟到该等待时间。
- 单元测试覆盖 `Retry-After` 解析、临时错误承载、Binance / OKX adapter 传递和 runner 使用较大的退避。

范围外：

- 不新增数据库字段或迁移。
- 不实现跨实例共享 token bucket。
- 不实现私有交易 API 或 live executor。
- 不改变永久错误、数据校验错误或 fetch lock skip 的状态语义。

剩余风险：

- 当前只尊重 HTTP 标准 `Retry-After`，不解析交易所私有响应体中的全部限流元数据。
- 多实例额度、公平调度、真实外部网络长期恢复压测和实盘安全边界仍未关闭。

### 阶段 1 data sync 保存结果 active lease 守卫补充

执行日期：2026-06-30

目标等级：scaffold。

范围内：

- `SaveDataSyncResult` 在写入 K 线、推进 `last_synced_open_time` 或改变任务终态前，按 `task_id` 锁定任务行并要求任务仍为 `running`。
- 保存结果还要求任务持有未过期 lease；`pending`、`running` 但无 lease、`running` 但 lease 过期的任务会返回 `data_sync_command_invalid_state`，不会写入 `market_candles`、不会推进游标、不会改 retry/error 字段。
- 既有人工保存结果的 PostgreSQL 集成测试改为先显式建立 running active lease，使测试路径接近真实 worker claim 后保存结果。
- PostgreSQL 集成测试覆盖 invalid save state 的拒绝语义，并回归空批次完成、invalid repair 写回、错标的拒绝和删除后拒绝保存。

范围外：

- 不改变 `SaveDataSyncResult` 接口签名，不在本切片校验具体 `workerID`。
- 不实现完整统一状态机、不改变 claim 公平性、不实现跨实例 token bucket。
- 不推进实盘交易所私有 API、live executor、订单提交或幂等下单。

剩余风险：

- 当前只证明保存结果需要 running active lease；尚未把所有 worker 类型抽象成统一状态机接口。
- 具体 `workerID` 匹配已在后续 worker identity 守卫补充中推进；尚未把所有 worker 类型抽象成统一状态机接口。

### 阶段 1 data sync 保存结果 worker identity 守卫补充

执行日期：2026-06-30

目标等级：scaffold。

范围内：

- `DataSyncResult` 增加 `WorkerID` 字段，data sync runner 的所有保存结果路径都会传入当前 `WorkerID`。
- `SaveDataSyncResult` 在锁定任务行后，要求任务仍为 `running`、`locked_until` 未过期，且 `locked_by` 精确匹配 `DataSyncResult.WorkerID`。
- 保存结果的错误 worker、缺失 worker、无 active lease 或过期 lease 均返回 `data_sync_command_invalid_state`，不会写入 K 线或推进游标。
- 单元测试覆盖 runner 将 worker identity 传给 repository；PostgreSQL 集成测试覆盖错误 worker lease 被拒绝；API / store repair 集成测试改为传入与人工 lease 一致的 worker identity。

范围外：

- 不重构所有 worker 的统一 lease 状态机接口。
- 不改变 retry / fail / release 的现有状态语义。
- 不实现分布式 token bucket、真实外部交易所长期压测或实盘交易所私有 API。

剩余风险：

- 该切片只收紧 data sync 保存结果路径；retry / fail / release 已在后续 worker ownership 切片继续收紧，仍需抽象到统一 lease transition 语义。
- 仍缺跨 worker 类型的一致状态机封装和真实多实例长期运行证据。

### 阶段 1 data sync 保存结果游标链守卫补充

执行日期：2026-06-30

目标等级：scaffold。

范围内：

- `SaveDataSyncResult` 在写入 `market_candles` 或推进 `last_synced_open_time` 前，会读取任务当前 `last_synced_open_time`。
- 如果 `DataSyncResult.LastOpenTime` 非空，store 会用本次 `Candles` 和任务当前游标计算可推进的连续 open_time 链尖端。
- `LastOpenTime` 必须精确等于该连续链尖端；空 candles 携带游标、或本次 candles 存在缺口却把游标推进到缺口后面的保存结果会被拒绝。
- PostgreSQL 集成测试覆盖上述两类非法游标均不写入 `market_candles`、不推进 `last_synced_open_time`、不改变 retry/error 状态。
- 既有未来 exchange backoff 测试改为不再用“空 candles + 非空游标”的人工路径。

范围外：

- 不改变 data sync runner 的连续游标计算、claim、公平性、retry/backoff 或 fetch lock 语义。
- 不实现自动补缺口、分布式 token bucket、真实外部交易所长期压测或实盘交易所私有 API。
- 不推进 live executor、订单提交、撤单、查单或幂等实盘下单。

剩余风险：

- 该守卫只证明 store 不接受与本次保存 candles 不一致的游标推进；不代表自动修复历史缺口。
- 完整统一 worker 状态机、真实多实例长期 soak 和真实交易所网络恢复压测仍未关闭。

### 阶段 1 data sync 未闭合 K 线游标守卫补充

执行日期：2026-06-30

目标等级：scaffold。

范围内：

- data sync runner 会在保存结果前过滤交易所返回的未闭合 K 线。
- `last_synced_open_time` 只基于本批次闭合 K 线的连续 open_time 链推进。
- 如果批次只返回未闭合 K 线，runner 不写入 `market_candles`、不推进 cursor，也不会把有界同步或无 `endTime` 的一次性同步任务误判为 completed。
- 单元测试覆盖混合闭合/未闭合批次只保存闭合 K 线，以及有界和无 `endTime` 任务只有未闭合 K 线的批次不完成任务。
- PostgreSQL 集成测试覆盖真实 runner + store 路径下未闭合 K 线不会落库，cursor 只推进到最后一根闭合 K 线，有界任务和无 `endTime` 一次性任务都保持 pending 等待后续闭合数据。

范围外：

- 不改变交易所 adapter 对未闭合 K 线的标记方式。
- 不实现实时流式 K 线、partial candle 展示或自动补全历史缺口。
- 不推进实盘交易所私有 API、live executor、订单提交或幂等实盘下单。

剩余风险：

- 该守卫只保证 data sync 不用未闭合 K 线写库或推进游标；真实交易所返回乱序、缺失或长时间停留在未闭合状态时，仍依赖现有 retry、gap 和数据健康路径暴露。
- 仍缺真实外部交易所长期运行压测和完整统一 worker 状态机封装。

### 阶段 1 data sync retry/fail/release worker ownership 补充

执行日期：2026-06-30

目标等级：scaffold。

范围内：

- data sync runner 在记录 retry、标记 failed、shutdown release 和 fetch-lock skipped release 时传入当前 `WorkerID`。
- `RecordDataSyncRetry` 只允许 `running`、未过期 active lease 且 `locked_by` 匹配当前 worker 的任务释放 lease、写入 `last_error`、设置 `next_attempt_at` 并记录 exchange backoff。
- `MarkDataSyncFailed` 只允许持有未过期 active lease 的当前 worker 将任务标为 failed、停用 sync/realtime 并释放 lease。
- `ReleaseDataSyncTask` 和 `ReleaseDataSyncTaskAfterSkippedFetch` 不允许错误 worker 清理别人的 lease；fetch-lock skipped release 也不会让错误 worker 回滚 `attempt_count`。
- runner 单元测试覆盖 retry/fail/release/skipped release 的 worker identity 传递；PostgreSQL 集成测试覆盖错误 worker 被拒绝且不改状态、不写 backoff、不清 lease、不回滚 attempt。

范围外：

- 不重构所有 worker 的统一 lease 状态机接口。
- 不改变用户手动 retry failed 任务的 API 语义。
- 不实现分布式 token bucket、真实外部交易所长期压测或实盘交易所私有 API。
- 不推进 live executor、订单提交、撤单、查单或幂等实盘下单。

剩余风险：

- 该切片只收紧 data sync worker 的 retry/fail/release 状态变更所有权；backtest / trading / notification worker 仍未统一到同一状态机抽象。
- 仍缺多实例长期 soak、真实交易所网络恢复压测和跨进程共享交易所额度。

### 阶段 1 data sync lease race 收敛补充

执行日期：2026-06-30

目标等级：scaffold。

范围内：

- `HeartbeatDataSyncTask` 在当前 worker 不再持有 data sync active lease 时返回 `data_sync_command_invalid_state` 领域错误。
- data sync runner 遇到 heartbeat ownership lost、保存结果 ownership lost、retry 记录 ownership race、failed 记录 ownership race 或任务已删除时，只跳过当前任务并交还调度，不把旧 worker 的失效 lease 继续升级成任务 failed 或整个 worker 退出。
- 单元测试覆盖 heartbeat lease lost、retry 记录 invalid-state race 和 failed 记录 invalid-state race；PostgreSQL 集成测试覆盖错误 worker / 过期 lease heartbeat 返回领域 invalid-state 且不改锁。

范围外：

- 不改变任务 claim、公平性、退避算法、fetch lock 或 candle 写入语义。
- 不重构 backtest / trading / notification worker。
- 不实现跨进程共享交易所额度、真实外部交易所长期压测或实盘交易所私有 API。

剩余风险：

- 该切片只避免 data sync 旧 worker 在 lease race 后错误退出或错误标失败；完整统一 worker 状态机仍未实现。
- 仍缺多实例长期 soak 和真实交易所网络恢复压测。

### 阶段 1 data sync release race 收敛补充

执行日期：2026-06-30

目标等级：scaffold。

范围内：

- data sync runner 在 shutdown release 和 exchange fetch lock held skipped release 时，如果任务已删除或当前 worker 已不再持有 lease，会记录为 ownership race 并跳过当前任务，不让整个 sync worker 退出。
- exchange fetch lock 基础设施错误仍保留原错误上抛；如果释放 lease 同时遇到 ownership race，不会用 release race 掩盖基础设施错误。
- 单元测试覆盖 shutdown release invalid-state race、fetch-lock held release invalid-state race，以及 fetch-lock 基础设施错误叠加 release not-found race 时仍返回基础设施错误。

范围外：

- 不改变 fetch lock 获取、skip 统计、任务 retry/backoff、claim 公平性或 K 线写入语义。
- 不重构 backtest / trading / notification worker。
- 不实现真实交易所网络长期压测、跨进程共享额度或实盘交易所私有 API。

剩余风险：

- release race 处理仍是 data sync runner 局部修补；完整统一 worker 状态机和跨 worker 类型一致 release 语义仍未实现。
- 仍缺真实多实例长期运行验证。

### 阶段 1 data sync claim 并发互斥证据补充

执行日期：2026-06-30

目标等级：scaffold。

范围内：

- 新增 PostgreSQL 集成测试 `TestIntegrationClaimDataSyncTaskIsExclusiveUnderConcurrentWorkers`。
- 测试构造单个 active catalog 的 pending data sync task，并用两个 worker goroutine 同时调用 `ClaimDataSyncTask`。
- 断言并发 claim 中只有一个 worker 能领取该任务，第三个 worker 在 lease 未过期时不能再次领取同一任务。
- 测试回查 `data_sync_tasks`，确认 `locked_by` 只属于实际领取 worker，`attempt_count` 只增加一次。

范围外：

- 不改变 claim SQL、任务优先级、lease TTL、retry/backoff、fetch lock 或 K 线写入语义。
- 不把该测试扩展为 Docker Compose 多 sync 容器长期运行压测。
- 不实现分布式 token bucket、跨实例公平调度或实盘交易所私有 API。

剩余风险：

- 该证据只证明单个 PostgreSQL claim 入口在并发调用时不会让同一任务被双 worker 同时领取。
- 仍缺真实多实例长期 soak、真实交易所网络恢复压测和跨进程共享交易所额度。

### 阶段 8 browser smoke 全局超时补充

执行日期：2026-06-30

目标等级：scaffold。

范围内：

- `scripts/stage8-visual-smoke.mjs`、`scripts/stage8-state-visual-smoke.mjs` 和 `scripts/research-chart-height-smoke.mjs` 增加 `SMOKE_TOTAL_TIMEOUT_MS` 总运行时限。
- 总超时时会打印明确失败原因，关闭已建立的 CDP WebSocket，停止本脚本拉起的 headless Chrome，并清理临时 Chrome profile。
- 保留原有 per-selector / per-sample 等细粒度等待超时，不改变视觉断言语义。
- 超时清理后仍在 unwind 的浏览器流程 reject 会被抑制，避免 Node 进程出现未处理拒绝。

范围外：

- 不新增像素快照基线。
- 不把 browser smoke 升级为多浏览器视觉回归。
- 不把视觉 smoke 改成 CI 独立硬门禁。
- 不修正业务页面布局或产品语义。

剩余风险：

- browser smoke 仍依赖本地 API、数据状态和 headless Chrome 环境；正常全量通过不等于多浏览器像素级无回归。
- 总超时只能防止脚本无限挂起，不能替代后续视觉基线、真实数据状态覆盖和长期运行验证。

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

### 阶段 0 质量底座当前复核补充

复核时间：2026-06-30

目标等级：scaffold。

范围内：

- 把阶段 0 的基础工程要求固化为可重复脚本检查，而不是只依赖人工读文档。
- 复核 `internal/web/api/server.go` 和 `web/frontend/src/i18n/messages.ts` 继续保持拆分后的入口规模。
- 复核前端路由和页面不再引用 `PageStub`。
- 复核 README 继续声明整体项目是 `scaffold`，并指向交付协议、质量审计和实施计划。
- 复核轻量质量门禁继续保留文件规模、scaffold marker 和阶段 0 foundation 检查。

范围外：

- 不推进 CandleProvider、数据同步、策略、回测、模拟盘、通知或实盘能力。
- 不升级任何模块到 demo、usable 或 production-safe。

新增检查：

- `scripts/check-stage0-foundation.sh`。
- `scripts/quality-gate.sh` 已把 `stage 0 foundation` 纳入阻断性检查。

验收结果：

- `scripts/check-stage0-foundation.sh` 通过。
- `scripts/quality-gate.sh` 通过，新增 `stage 0 foundation` 阻断项已进入轻量门禁。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `cd web/frontend && pnpm run typecheck` 通过。
- `cd web/frontend && pnpm run test` 通过。
- `cd web/frontend && pnpm run build` 通过。

阶段 0 结论：

- 本轮只加强质量底座检查能力。
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
- CandleProvider 返回窗口级 `pagination` 元数据和 opaque `previousCursor/nextCursor`；研究页可显式请求上一/下一窗口并把 cursor 保留在 URL，旧 `from/to` URL 仍兼容。
- 已有 `scripts/stage1-candle-provider-perf-smoke.sh` 覆盖 240000 根基础 `1m` 到 1000 根 `4h` 的 PostgreSQL 查询边界；单测覆盖默认 `1d limit=1000` 从 1440000 根基础 `1m` 流式聚合；PostgreSQL 集成测试覆盖 6 个并发 `15m` 聚合查询的一致结果、coverage 和 pagination metadata；仍缺长期 soak / 冷缓存 / 真实生产数据分布压测、超过 1440000 根基础 K 线的缓存/分段策略和更多异常数据边界；闭合周期信号已有 runner 侧基础过滤，未闭合 K 线不再进入策略输入。

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

### 阶段 1 研究页 K 线高度稳定性九次加固

执行时间：2026-06-28

触发问题：

- 用户侧再次反馈前端 K 线图表界面会无限拉高，直到页面崩掉。
- 复查提交发现 `f891f46` 将研究页图表面板从 flex 剩余空间改回 grid，并把 `.research-chart-body` 恢复为 `height: 100%`，与前一轮“切断百分比高度反馈”的目标冲突。

修复范围：

- `ResearchPage` 图表面板恢复为固定高度 flex 列布局，工具栏占自然高度，`.research-chart-body` 用 `flex: 1 1 0` 承载剩余视口。
- `.research-chart-body` 不再声明 `height: 100%`，避免 grid track / 百分比高度解算重新参与 chart DOM 高度反馈。
- 新增 `ResearchPage.layout.test.ts`，用 SFC raw source 检查研究页图表布局契约，防止再次回退到 grid 或 `height: 100%`。
- 保持 `TradingViewChart` 不向 root / canvas 写 inline 高度，图表库内部 DOM 仍被固定 viewport CSS 和 `contain: strict` 限制。

验证：

- `pnpm --dir web/frontend run test -- src/components/chart/TradingViewChart.test.ts src/pages/ResearchPage.layout.test.ts`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `docker compose up -d --build api`
- `curl -fsS http://127.0.0.1:8080/readyz`
- Headless Chrome 桌面 `1440x900` 登录并打开 `/research`，80 次采样 first/last 均为 `documentHeight=1238`、`panel=680`、`chartBody=603`、`chart=603`、`canvasHost=603`、`tv=603`，`uniqueDocumentHeights=1`、`uniqueChartBodyHeights=1`、`grew=false`，无 runtime/log error。
- Headless Chrome 移动 `390x844` 登录并打开 `/research`，80 次采样 first/last 均为 `documentHeight=1256`、`panel=624`、`chartBody=457`、`chart=457`、`canvasHost=457`、`tv=457`，`uniqueDocumentHeights=1`、`uniqueChartBodyHeights=1`、`grew=false`，无 runtime/log error。
- `go test ./...`
- `go vet ./...`
- `git diff --check`
- `scripts/quality-gate.sh`

失败：

- 新增布局测试首次失败两次：第一次使用 Node `fs/url`，不符合前端 src TS 环境；第二次把 `max-height: 100%` 误判为 `height: 100%`。已改为 Vite `?raw` 导入和 CSS 声明级匹配后通过。

剩余风险：

- 本轮在 headless Chrome 桌面/移动均未复现持续增长；用户可见 Chrome 会话仍需要人工确认，但已关闭当前代码中实际存在的 grid 百分比高度回归入口。

### 阶段 1 研究页 K 线高度稳定性十次加固

执行时间：2026-06-28

触发问题：

- 用户继续反馈前端 K 线图表界面会无限拉高，直到页面崩掉。
- 本地 Vite 页面、`127.0.0.1:8080` 当前静态页和真实 API 登录后的 headless Chrome 采样均未复现持续增长，但九次加固后的 `.research-chart-body` 仍保留 `height: auto`，在 flex 固有尺寸计算上仍不够硬。

修复范围：

- `.research-chart-body` 从 `height: auto` 改为 `height: 0`，保持 `flex: 1 1 0` 和 `contain: strict`，让图表槽只接受父面板分配的剩余空间，不再让子内容参与 flex item 的基础高度计算。
- `ResearchPage.layout.test.ts` 增加对 `height: 0` 和 `contain: strict` 的布局契约检查，防止再次回退到 `auto` / `100%` 高度。
- 重新执行生产构建，确认构建产物包含 `.research-chart-body{height:0;contain:strict}`。

验证：

- `pnpm --dir web/frontend exec vitest run src/pages/ResearchPage.layout.test.ts src/components/chart/TradingViewChart.test.ts`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- Headless Chrome + Vite mock API 桌面 `2048x997` 打开 `/research`，40 次采样 first/last 均为 `scrollHeight=1099`、`panel=760`、`body=683`、`chart=683`、`tv=683`，`stable=true`。
- 修复前已对 `127.0.0.1:8080/research` 当前静态页做 60 次 mock API 采样和 80 次真实 API 登录采样，均未复现持续增长；本轮改动进一步收紧 CSS flex 基准。
- `docker compose up -d --build api`
- `curl -fsS http://127.0.0.1:8080/assets/ResearchPage-DDrIJRVm.css` 确认构建产物包含 `.research-chart-body{height:0;contain:strict}`。
- Headless Chrome + 重建后的 `127.0.0.1:8080` 真实 API 桌面 `2048x997` 登录并打开 `/research`，100 次采样 first/last 均为 `scrollHeight=1318`、`panel=760`、`body=683`、`chart=683`、`tv=683`，`stable=true`、`uniqueDocHeights=[1318]`、`uniqueChartHeights=[683]`。
- `go test ./...`
- `go vet ./...`

失败：

- 无硬失败。

剩余风险：

- 本轮仍未在用户可见 Chrome 会话中复现原始无限增长；当前 8080 服务已重建，需要用户刷新可见 Chrome 页面确认。但当前源码和构建产物已经把研究页图表 body 的 flex 基准收敛为固定 0，不再允许子图表内容放大父级基础高度。

### 阶段 1 研究页 K 线高度稳定性十一次加固

执行时间：2026-06-28

触发问题：

- 用户继续反馈前端 K 线图表界面会无限拉高，直到页面崩掉。
- 复查发现当前 `TradingViewChart` 仍会自己向上查找 `.research-chart-body` / `.chart-panel` 并推导高度；这让真实浏览器里任何祖先高度污染都有机会继续进入 `chart.resize`。
- 本轮第一次改为 `height:auto` 后，headless Chrome 采样发现 `.research-chart-panel` 继承全局 `.chart-panel { contain: size layout paint; }` 时会折叠到 `2px`，必须同时覆盖 containment。

修复范围：

- `TradingViewChart` 的 ResizeObserver 目标从页面布局祖先收敛到自己的 `.trading-chart__canvas`。
- `TradingViewChart` 初始化和 resize 只读取 canvas viewport 的 `clientWidth/clientHeight`、observer content box、computed height 或自身 bounds；不再向上读取 `.research-chart-body` / `.chart-panel`，也不再计算 panel 可用高度。
- 研究页图表槽改为 `--research-chart-viewport-height` 控制的固定高度；`.research-chart-body` 使用 `flex: 0 0 var(...)`、`height: var(...)`、`max-height: var(...)` 和 `contain: strict`。
- `.research-chart-panel` 保持 auto 高度但覆盖 `contain: layout paint`，避免全局 size containment 把 panel auto 高度折叠。
- `ResearchPage.layout.test.ts` 增加固定 viewport、`contain: layout paint`、不回退到 grid / `height:100%` 的布局契约检查。
- `TradingViewChart.test.ts` 改为验证组件只观察 canvas viewport、祖先膨胀不影响初始化、viewport 变化才 resize、异常直接 viewport 高度仍被安全上限截断，且不写 inline 宽高。

验证：

- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts src/pages/ResearchPage.layout.test.ts`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `go vet ./...`
- `go test ./...`（首次因本机磁盘空间不足失败，`go clean -cache -testcache` 后重试通过）
- `scripts/quality-gate.sh`
- `git diff --check`
- 本地生产构建产物 `web/frontend/dist/assets/ResearchPage-B18wn4Jg.css` 包含 `--research-chart-viewport-height`、`.research-chart-body` 固定高度和 `.research-chart-panel{...contain:layout paint}`。
- Headless Chrome + 本轮 mock preview 桌面 `1440x900` 打开 `/research`，100 次采样 first/last 均为 `documentHeight=1020`、`panelHeight=680`、`chartBodyHeight=603`、`chartHeight=603`、`tvHeight=603`，`uniqueDocs=[1020]`、`uniquePanels=[680]`、`uniqueBodies=[603]`、`uniqueCharts=[603]`、`grew=false`。
- Headless Chrome + 本轮 mock preview 移动 `390x844` 打开 `/research`，80 次采样 first/last 均为 `documentHeight=1058`、`panelHeight=624`、`chartBodyHeight=457`、`chartHeight=457`、`tvHeight=457`，`uniqueDocs=[1058]`、`uniquePanels=[624]`、`uniqueBodies=[457]`、`uniqueCharts=[457]`、`grew=false`。

失败：

- 第一次浏览器采样发现 `.research-chart-panel` 因全局 `contain:size` 在 auto 高度下折叠到 `2px`；已通过研究页覆盖 `contain: layout paint` 修正，并由后续桌面/移动采样验证。
- `docker compose up -d --build api` 第二次替换 8080 时 Docker Desktop 返回 `metadata_v2.db: input/output error`；重试又返回 `postgres:16-alpine` blob `input/output error`。当前 8080 容器随后出现 `/`、`/research`、assets 均 404，`/api/auth/login` 返回 500，不能作为本轮真实运行验收依据。
- `go test ./...` 首次失败于 okx 测试链接阶段：`mapping output file failed: no space left on device`；`df -h` 显示可用空间约 `146MiB`。执行 `go clean -cache -testcache` 后可用空间约 `1.4GiB`，重试 `go test ./...` 通过。

剩余风险：

- 本轮通过 mock preview 验证了最新前端产物的浏览器布局稳定性，但由于本机 Docker Desktop 存储层 I/O 异常，未能把最终修正版可靠替换到 `127.0.0.1:8080` 真实 API 容器。
- 用户可见 Chrome 会话仍需在 Docker 恢复后用真实 8080 再做一次确认；当前代码已经移除图表组件向上读取祖先高度的反馈入口。

### 阶段 1 研究页 K 线高度稳定性 8080 复验

执行时间：2026-06-28

触发问题：

- 用户继续反馈前端 K 线图表界面会无限拉高，直到页面崩掉。
- 上一轮剩余风险中记录 Docker Desktop 存储层 I/O 异常导致真实 `127.0.0.1:8080` 未能作为最终运行验收依据。

修复范围：

- 新增 `scripts/research-chart-height-smoke.mjs`，使用系统 Chrome DevTools Protocol 启动隔离 headless Chrome、登录本地 8080、打开 `/research`，并对桌面与移动 viewport 连续采样。
- smoke 检查 `document`、`.research-chart-panel`、`.research-chart-body`、`.trading-chart` 和 `.tv-lightweight-charts` 高度；任一高度在采样窗口内增长超过容差即失败。
- README 增加本地栈启动后的研究页图表高度 smoke 入口。

验证：

- `docker compose ps --format json` 显示 `api` 与 `postgres` healthy，`sync`、`backtest`、`trading`、`notify` running。
- `curl -I http://127.0.0.1:8080/research` 返回 `HTTP/1.1 200 OK`。
- `curl http://127.0.0.1:8080/assets/ResearchPage-B18wn4Jg.css` 确认真实 8080 产物包含固定 `--research-chart-viewport-height`、`.research-chart-body` 固定高度和 `.research-chart-panel{...contain:layout paint}`。
- 临时 headless Chrome 手工采样桌面 `1440x900` 登录并打开 `/research`，30 次 first/last 均为 `documentHeight=1238`、`panel=680`、`chartBody=603`、`chart=603`、`canvas=603`、`tv=603`，无增长。
- 临时 headless Chrome 手工采样移动 `390x844` 登录并打开 `/research`，30 次 first/last 均为 `documentHeight=1256`、`panel=624`、`chartBody=457`、`chart=457`、`canvas=457`、`tv=457`，无增长。
- `node scripts/research-chart-height-smoke.mjs` 在真实 8080 上通过：桌面 `doc 1238->1238, panel 680->680, body 603->603, chart 603->603, tv 603->603`；移动 `doc 1256->1256, panel 624->624, body 457->457, chart 457->457, tv 457->457`。

失败：

- 当前真实 8080 未复现高度增长。

剩余风险：

- 该 smoke 依赖本机可用 Chrome；无 Chrome 的 CI/主机需要设置 `CHROME_PATH` 或跳过该本地运行检查。
- 这次只证明当前真实 8080 构建的研究页图表高度稳定，不关闭研究页交易对硬编码、图表工具薄和外部交易所网络韧性风险。

### 阶段 1 研究页 K 线高度稳定性十二次加固

执行时间：2026-06-28

触发问题：

- 用户再次反馈前端 K 线图表界面会无限拉高，直到页面崩掉。
- 现有 8080 headless Chrome 长采样未复现增长，但代码层仍把 ResizeObserver 绑定在传给 lightweight-charts 的 `.trading-chart__canvas` mount 节点上；该节点内部 DOM 会被图表库持续改写，不应作为组件自己的稳定 resize 输入。

修复范围：

- `TradingViewChart` 的 ResizeObserver 目标从 `.trading-chart__canvas` 改为稳定的 `.trading-chart` root viewport。
- ResizeObserver 回调只在 entry target 等于当前观察宿主时才调度 `chart.resize`，忽略 chart mount 或其他子节点的异常 resize entry。
- 初始化和 resize 仍只读取组件自身 viewport，不向上读取 `.research-chart-body` / `.chart-panel`，也不向 root/canvas 写 inline 尺寸。
- `TradingViewChart.test.ts` 覆盖 root viewport 观察、chart mount 异常 entry 被忽略、祖先/内部 chart DOM 膨胀不影响初始化和 resize。

验证：

- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts src/pages/ResearchPage.layout.test.ts`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `go test ./...`
- `go vet ./...`
- `scripts/quality-gate.sh`
- `git diff --check`
- `docker compose up -d --build api` 已重建本地 8080 API 容器，`docker inspect` 显示 `tictick-hi-api-1` healthy。
- `curl -I http://127.0.0.1:8080/research` 返回 `HTTP/1.1 200 OK`。
- 重建前真实 8080 长采样 `SMOKE_SAMPLES=180 SMOKE_INTERVAL_MS=250 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 仍稳定：桌面 `doc 1238->1238, panel 680->680, body 603->603, chart 603->603, tv 603->603`；移动 `doc 1256->1256, panel 624->624, body 457->457, chart 457->457, tv 457->457`。
- 重建后真实 8080 采样 `SMOKE_SAMPLES=80 SMOKE_INTERVAL_MS=150 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：桌面 `doc 1238->1238, panel 680->680, body 603->603, chart 603->603, tv 603->603`；移动 `doc 1256->1256, panel 624->624, body 457->457, chart 457->457, tv 457->457`。

剩余风险：

- 真实 8080 headless Chrome 未复现用户可见 Chrome 中的原始无限增长；本轮代码层已切断 chart mount DOM 到 `chart.resize` 的反馈入口，并由本地 8080 smoke 覆盖当前构建。

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

### 阶段 1 Candle 窗口分页 metadata 补充

执行时间：2026-06-28

目标等级：demo

触发问题：

- `/api/candles` 已支持 `limit/from/to`，但响应没有说明当前窗口前后是否还有 K 线。
- 研究页只能加载默认窗口，用户无法显式切换上一段或下一段窗口，也无法把当前窗口游标保留到 URL。

修复范围：

- `CandleResult` 新增 `pagination` 元数据，包含 `hasPrevious/hasNext` 和相邻窗口 `previousFrom/previousTo/nextFrom/nextTo`。
- CandleProvider 对 native 查询使用同周期数据探测前后是否存在 K 线；对 aggregated 查询使用 `1m` 基础 K 线探测，但暴露的窗口游标仍按请求周期计算。
- `/api/system/api-contract` 和 `web/frontend/src/types/api.generated.ts` 增加 `CandlePagination`。
- 前端 API wrapper 保留 pagination；研究页从 URL 读取 `from/to`，上一/下一窗口按钮会请求相邻窗口并更新 URL。

验证：

- `go test ./internal/data ./internal/web/api ./internal/store/postgres`
- `pnpm --dir web/frontend exec vitest run src/services/api/data.test.ts src/composables/useResearchWorkspace.test.ts src/pages/ResearchPage.layout.test.ts`
- `pnpm --dir web/frontend run typecheck`

失败：

- 无硬失败。

后续风险：

- 这只是窗口级 metadata 和显式翻页，不是完整 cursor pagination；还没有大范围历史查询性能压测、虚拟化、预取或聚合缓存。
- aggregated 的前后探测基于基础 `1m` 是否存在，不能证明下一整个目标周期窗口完整健康，仍需结合 health/gap/coverage 观察。

### 阶段 1 Candle 当前窗口可观察补充

执行时间：2026-06-28

目标等级：demo

触发问题：

- `/api/candles` 已返回上一/下一窗口游标，但没有返回当前实际窗口范围。
- 研究页用户点击上一/下一窗口后，只能从 URL 推断窗口，图表 metadata 没有直接显示当前返回数据覆盖的 `from/to/count`。

修复范围：

- `CandleResult` 新增 `window` 元数据，包含当前响应实际 K 线窗口 `from/to/count`。
- CandleProvider 对 native 和 aggregated 结果统一从返回 K 线的首尾 `openTime` 生成窗口 metadata；空窗口返回 `count=0`。
- `/api/system/api-contract` 和 `web/frontend/src/types/api.generated.ts` 增加 `CandleWindow`。
- 前端 API wrapper 保留 `window`；研究页元信息显示当前窗口范围和 K 线数量。
- 研究页上一/下一窗口按钮提取为 `ResearchWindowControls`，避免页面文件再次超过质量门禁硬上限。

验证：

- `go test ./internal/data ./internal/web/api`
- `pnpm --dir web/frontend exec vitest run src/services/api/data.test.ts src/composables/useResearchWorkspace.test.ts src/pages/ResearchPage.layout.test.ts`
- `pnpm --dir web/frontend run typecheck`

失败：

- 无硬失败。

后续风险：

- 当前窗口 metadata 只说明本次响应实际覆盖范围，不证明窗口内无缺口；仍需结合 `health/gaps/coverage` 判断数据质量。
- 这仍不是完整 cursor pagination 或大范围性能证明。

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

### 阶段 1 数据同步错误展示脱敏补充

执行时间：2026-06-28

目标等级：demo

触发问题：

- 研究页同步任务历史 `last_error` 中可能已经存在完整 Binance / OKX 请求 URL。
- 旧错误虽然不再由新 adapter 产生，但 `/api/data/tasks` 直接返回存量 `last_error`，前端表格 tooltip / title 也会保留原文，仍可能泄露请求 path、query 参数并撑坏错误列阅读体验。

修复范围：

- API server 在返回 `DataSyncTask` 前统一清理 `lastError`，覆盖列表、创建、retry、sync/realtime command 和 repair result 中的 created tasks。
- 清理规则将 `http/https` 外部 URL 替换为 host，保留交易所域名和错误原因，但移除 `/api/v3/klines`、`symbol`、时间窗口、limit 等 query。
- 前端 API wrapper 和 `DataSyncTaskTable` 使用同一类脱敏规则作为保底，避免测试注入或未来绕过 service 的原始错误进入 title / tooltip。
- 研究页错误列仍保持单行省略，tooltip 展示脱敏后的完整错误摘要。

验证：

- `go test ./internal/web/api -run 'TestDataSyncTaskRoutes|TestAPIContract|TestFrontendAPI'`
- `pnpm --dir web/frontend exec vitest run src/services/api/data.test.ts src/components/tables/DataSyncTaskTable.test.ts`
- `go test ./...`
- `go vet ./...`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `scripts/quality-gate.sh`
- `git diff --check`

失败：

- 无硬失败。

剩余风险：

- 本轮不修改数据库历史 `last_error` 原文，只在 API 和前端展示边界脱敏；后续若增加导出或审计页面，需要复用同类脱敏边界。
- 这不是交易所精确限流或真实网络长期压测。

### 阶段 1 数据同步完成边界补充

执行时间：2026-06-28

触发问题：

- 本地真实 8080 的研究页数据同步列表里保留了多条 Stage 8 smoke 生成的 `S8...USDT` 任务，这些任务已经有 `last_synced_open_time=2026-01-01T01:59:00Z` 和 `end_time=2026-01-01T02:00:00Z`，但仍是 `running + sync_enabled=true`。
- 常驻 sync worker 重启后会按 overlap 窗口继续领取这些一次性任务，并对测试 symbol 访问真实 Binance，造成 `api.binance.com: EOF; data-api.binance.vision: status 400 Bad Request` 之类的研究页噪音。
- 该问题不属于 UI 展示问题，而是一次性同步任务已覆盖 `endTime` 后仍留在 claim 队列的状态边界问题。

修复范围：

- `datasync.Runner` 在获取 exchange client 之前先判断非实时任务是否已经通过 `latestSyncedOpenTime + interval >= endTime` 覆盖目标窗口；已覆盖时直接 `SaveDataSyncResult(... Completed=true)`，不再因为 overlap 重打外部 API。
- `scripts/stage8-smoke.sh` 的研究数据 seed SQL 改为按状态机先 `pending -> running`，再落到 `succeeded + sync_enabled=false + finished_at`，避免以后 Stage 8 smoke 在本地 volume 留下可 claim 的假 symbol 同步任务。

验证：

- `go test ./internal/datasync ./internal/store/postgres`
- `go test ./...`
- `go vet ./...`
- `scripts/quality-gate.sh`
- `scripts/stage8-smoke.sh`
- `git diff --check`
- `TestRunnerCompletesOneShotTaskAlreadySyncedThroughEndWithoutExchangeClient` 覆盖已同步到 `endTime` 的一次性任务，即使没有注册 exchange client 也会直接完成，不会发起外部 fetch。
- Stage 8 smoke 本轮创建的 `dataTask=dst_6f4d9aac2a0d58838a69960b` 在 PostgreSQL 中为 `status=succeeded`、`sync_enabled=false`、`realtime_enabled=false`、锁字段为空、`last_error=''`。
- 使用新镜像执行 `docker compose run --rm sync sync --once` 后，旧 S8 running 残留数从 15 降到 14，新增一条 `succeeded`，且没有新增 `last_error`，证明常驻 worker 会按新完成边界逐步消化旧残留。

失败：

- 无硬失败。

剩余风险：

- 本轮没有删除历史本地测试残留；旧 `S8...` running 任务会被新 sync worker 逐条转为 `succeeded`，不是一次性批量清库。
- 真实交易所网络限流和全局退避仍未达到 usable；研究页 / 数据同步入口的 symbol 合法性预校验已在后续补充中收敛，但交易对仍是固定白名单。

### 阶段 1 数据同步持久化退避补充

执行时间：2026-06-28

触发问题：

- 临时 market data 错误此前只释放 lease 并等待下一轮 worker 轮询；如果外部交易所持续 EOF / 429 / 5xx，任务会被频繁重新领取，缺少跨 worker / 重启可见的退避时间。
- 研究页只能看到最近错误，不能看到临时错误下一次会何时重试。

修复范围：

- `data_sync_tasks` 新增 `next_attempt_at`，用于持久化单任务退避时间，并增加按 `status,next_attempt_at,locked_until` 的 claim 辅助索引。
- `hi sync` 增加 `SYNC_RETRY_BACKOFF` 和 `SYNC_MAX_RETRY_BACKOFF`，默认 `30s` / `5m`；临时错误耗尽短重试后按 attempt count 做有上限指数退避。
- `ClaimDataSyncTask` 跳过 `next_attempt_at > now()` 的任务，重新 claim 到期任务时清理 `next_attempt_at`。
- `SaveDataSyncResult`、`MarkDataSyncFailed`、手动 retry、重新启用 sync / realtime 均清理过期退避时间。
- `/api/data/tasks` 返回 `nextAttemptAt`，前端研究页任务表新增“下次重试 / Next retry”列。
- Docker Compose 和 `.env.example` 暴露同步退避配置。

验证：

- `go test ./internal/datasync ./internal/store/postgres ./internal/web/api`
- `go test ./...`
- `go vet ./...`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `scripts/quality-gate.sh`
- `git diff --check`
- `docker compose up -d --build api sync`
- `docker inspect` 显示 `tictick-hi-api-1` healthy，`docker compose ps` 显示 `api` / `sync` running。
- 本地 PostgreSQL `information_schema.columns` 确认 `data_sync_tasks.next_attempt_at` 为 `timestamp with time zone`。
- 本地 PostgreSQL `pg_indexes` 确认 `idx_data_sync_tasks_next_attempt` 已存在。
- `docker compose exec -T sync ...` 确认 `SYNC_RETRY_BACKOFF=30s`、`SYNC_MAX_RETRY_BACKOFF=5m`。
- `curl -I http://127.0.0.1:8080/research` 返回 `HTTP/1.1 200 OK`。
- `SMOKE_SAMPLES=30 SMOKE_INTERVAL_MS=150 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：桌面 `doc 1238->1238, panel 680->680, body 603->603, chart 603->603, tv 603->603`；移动 `doc 1256->1256, panel 624->624, body 457->457, chart 457->457, tv 457->457`。

剩余风险：

- 这是单任务持久化退避，不是交易所级 / 集群级全局限流器；真实网络长期恢复压测仍未关闭。

### 阶段 1 数据同步交易所级冷却补充

执行时间：2026-06-28

触发问题：

- 单任务 `next_attempt_at` 退避只能阻止同一个任务被立刻重新领取；同一交易所下的其他任务仍可能在交易所 EOF / 429 / 5xx 期间被其他 worker 继续领取。
- 阶段 1 仍需向真实交易所网络恢复边界推进，但不能冒充完整生产级权重限流。

修复范围：

- 新增 `data_sync_exchange_backoffs` 表，按 exchange 持久化 `next_attempt_at` 和最近错误摘要。
- 临时 market data 错误记录 retry 时，同一事务内更新任务级 `next_attempt_at` 并 upsert 交易所级冷却；同交易所已有更晚冷却时间时保留更晚时间。
- `ClaimDataSyncTask` 增加 active exchange backoff 排除条件，跳过同交易所未到冷却期的任务。
- 系统健康 `sync-worker` 服务增加 `exchangeBackoffCount` 和 `nextExchangeAttemptAt`，前端运维健康页展示交易所冷却数量和下次交易所重试时间。

验证：

- `go test ./internal/store/postgres ./internal/web/api`
- `pnpm --dir web/frontend exec vitest run src/services/api/system.test.ts`
- `pnpm --dir web/frontend run typecheck`
- `go test ./...`
- `go vet ./...`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `scripts/quality-gate.sh`
- `git diff --check`
- `docker compose up -d --build api sync`
- `curl -fsS --max-time 10 http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`。
- 本地 PostgreSQL `to_regclass('public.data_sync_exchange_backoffs')` 确认交易所冷却表存在。
- 本地 PostgreSQL `to_regclass('public.idx_data_sync_exchange_backoffs_next_attempt')` 确认交易所冷却索引存在。
- 临时插入 `codex-smoke` active exchange backoff 后，`GET /api/system/health` 返回 `sync-worker` 为 `warning`，`exchangeBackoffCount=1`，`nextExchangeAttemptAt` 有值，`detail` 包含 `exchange_backoff=1`；验证后已删除临时行。

剩余风险：

- 这是按交易所维度的冷却门禁，不是 Binance / OKX request weight 级别的限流器；仍未做真实网络长期压测、分 endpoint 权重模型和多实例压力证明。

### 阶段 1 数据同步连续游标补充

执行时间：2026-06-28

触发问题：

- 数据同步 runner 此前用本批次最大 `open_time` 推进 `last_synced_open_time`。
- 如果交易所批量返回 `00:00, 00:01, 00:03` 这类带内部缺口的数据，游标会跨过 `00:02`，后续恢复窗口可能不再稳定修复该缺口。
- 一次性同步任务还可能因为 `len(candles) < batchLimit` 在未覆盖 `endTime` 的情况下标记完成。

修复范围：

- `datasync.Runner` 新增连续 open_time 链计算，先排序去重，再只把游标推进到按 interval 连续的链尾。
- 已有 `last_synced_open_time` 时，批次必须从当前游标或下一根 K 线连续延伸；如果 overlap 窗口内缺口仍存在，则保存返回 K 线但不推进游标。
- 一次性任务只有在连续游标的下一根 open_time 覆盖 `endTime` 时才标记完成，不再仅凭返回数量小于 batch limit 完成。
- 删除旧的“最大 open_time 即游标” helper，避免后续误用。

验证：

- `go test ./internal/datasync`
- `go test ./...`
- `go vet ./...`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `scripts/quality-gate.sh`
- `git diff --check`
- `docker compose up -d --build sync`
- `docker compose ps sync postgres migrate` 显示 `postgres` healthy，`sync` running。
- `TestRunnerDoesNotAdvanceCursorPastFetchedGap` 覆盖初始批次存在内部缺口时，游标停在缺口前且任务不完成。
- `TestRunnerDoesNotAdvanceCursorWhenOverlapGapRemains` 覆盖重启 overlap 窗口内缺口未修复时，不推进已有游标。
- `TestRunnerAdvancesCursorAfterOverlapGapIsFilled` 覆盖 overlap 缺口补齐后，游标推进并按 `endTime` 完成一次性任务。

剩余风险：

- 本轮只保证同步游标不跨过批次内缺口，不做全历史缺口扫描、自动补全队列或 UI 一键修复。
- 如果交易所长期不返回缺失 K 线，任务会继续停留在可重试状态；真实恢复压测和告警策略仍未关闭。

### 阶段 1 数据源 symbol 前门校验补充

执行时间：2026-06-28

触发问题：

- 研究页此前把 Binance compact symbol 和 OKX hyphen instrument 混在同一个下拉里，用户可以创建 `binance / BTC-USDT` 或 `okx / BTCUSDT` 这类不一致数据源。
- API 只校验 `exchange/symbol/interval` 非空，非法组合会落库并进入 sync worker，最后表现为外部交易所 API 报错，不利于恢复和排障。

修复范围：

- `POST /api/data/tasks` 增加 exchange-specific symbol 校验：Binance 只接受 `BTCUSDT` 这类大写紧凑格式，OKX 只接受 `BTC-USDT` 这类大写 instrument 格式；未知 exchange 返回 `400 invalid_request`。
- `GET /api/candles` 复用同一校验，非法 URL query 不再进入 CandleProvider。
- 研究页交易对选项按当前 exchange 过滤；切换 exchange 或从 URL 初始化时自动把 symbol 收敛到对应交易所默认值。
- 创建同步任务弹窗使用表单 exchange 对应的交易对选项，并移除自由 tag 输入，避免前端直接创建非法组合。

验证：

- `go test ./internal/web/api`
- `pnpm --dir web/frontend exec vitest run src/utils/marketSymbols.test.ts src/pages/ResearchPage.layout.test.ts`
- `go test ./...`
- `go vet ./...`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `scripts/quality-gate.sh`（首次因新增测试让 `server_test.go` 超过 700 行失败；已拆分到 `data_task_validation_test.go` 后重跑通过）
- `git diff --check`
- `docker compose up -d --build api`
- `docker inspect` 显示 `tictick-hi-api-1` healthy。
- `curl -I http://127.0.0.1:8080/research` 返回 `HTTP/1.1 200 OK`。
- 真实 8080 `POST /api/data/tasks` 验证 `binance / BTC-USDT` 与 `okx / BTCUSDT` 均返回 `400 invalid_request`，错误消息分别指向 Binance compact 和 OKX instrument 格式。
- 真实 8080 `GET /api/candles` 验证同样的非法组合均返回 `400 invalid_request`，不进入 CandleProvider。
- `SMOKE_SAMPLES=30 SMOKE_INTERVAL_MS=150 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：桌面 `doc 1238->1238, panel 680->680, body 603->603, chart 603->603, tv 603->603`；移动 `doc 1256->1256, panel 624->624, body 457->457, chart 457->457, tv 457->457`。

剩余风险：

- 研究页交易对仍是固定白名单，不是生产级交易所 instrument 搜索；回测 / 交易任务表单的交易对选项仍需后续阶段统一收敛。

### 阶段 1 研究页 symbol 输入白名单收敛补充

执行时间：2026-06-28

触发问题：

- 后端已经对 Binance / OKX symbol 做交易所格式校验，但研究页前端仍把图表上下文和创建同步任务的 symbol 限制在 BTC/ETH 两个建议项。
- 这会让用户无法从研究页同步或观察其它真实标的，Stage 1 数据同步和 K 线研究仍停留在玩具级入口。

修复范围：

- `marketSymbols` 从白名单判断改为格式判断：Binance 接受大写 compact symbol，OKX 接受大写 hyphen instrument，并保留 BTC/ETH 作为建议项和默认值。
- 研究页 symbol 控件从固定 `NSelect` 改为 `NAutoComplete`，允许输入任意符合当前交易所格式的 symbol。
- 图表查询和创建同步任务前会 trim/uppercase 并做前端格式校验；非法格式不会发起 candles 请求或创建任务。
- 切换 exchange 时仍会在当前 symbol 不匹配目标交易所格式时收敛到目标交易所默认值。

验证：

- `pnpm --dir web/frontend exec vitest run src/utils/marketSymbols.test.ts src/composables/useResearchWorkspace.test.ts src/pages/ResearchPage.layout.test.ts`
- `go test ./...`
- `go vet ./...`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `scripts/quality-gate.sh`
- `git diff --check`
- `docker compose build api`
- `docker compose up -d --no-deps api`
- `docker inspect --format '{{.State.Health.Status}}' tictick-hi-api-1` 返回 `healthy`。
- `curl -fsSI http://127.0.0.1:8080/research` 返回 `HTTP/1.1 200 OK`。
- `SMOKE_SAMPLES=40 SMOKE_INTERVAL_MS=150 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：桌面 `doc 1238->1238, panel 680->680, body 603->603, chart 603->603, tv 603->603`；移动 `doc 1256->1256, panel 624->624, body 457->457, chart 457->457, tv 457->457`。

剩余风险：

- 这不是生产级 instrument 搜索或在线校验：前端只校验格式，不证明交易所存在该标的；回测 / 交易任务表单已在后续补充中收敛到同一格式边界，但仍缺共享市场选择器和真实交易所 instrument 元数据。

### 阶段 1 策略任务数据源 symbol 边界补充

执行时间：2026-06-28

触发问题：

- 研究页已把 symbol 输入从固定白名单收敛为交易所格式校验，但回测 / 交易创建页仍保留混合 `BTCUSDT` / `BTC-USDT` 的 tag select。
- 后端创建回测和交易任务时只校验必填字段，没有复用 `/api/candles` 和数据同步任务的 exchange-specific symbol 校验，后续 runner 仍可能接收错格式数据源。

修复范围：

- `validateCreateBacktest` 和 `validateCreateTradingTask` 复用 `validateExchangeSymbol`，Binance hyphen / OKX compact 等错误组合会在 API 创建阶段返回 `400 invalid_request`。
- 策略任务表单复用 `marketSymbols` 的建议项、格式校验、trim/uppercase 和 exchange 切换收敛逻辑。
- 回测/交易创建 payload 使用 normalize 后的 symbol；非法格式时前端不提交，并显示 `research.invalidSymbolFormat`。
- `StrategyTaskFormPage` 的 symbol 控件从混合 `NSelect tag` 改为 `NAutoComplete`，BTC/ETH 只作为当前交易所的建议项。

验证：

- `go test ./internal/web/api -count=1`
- `pnpm --dir web/frontend exec vitest run src/composables/useStrategyTaskForm.test.ts src/pages/StrategyTaskFormPage.layout.test.ts src/utils/marketSymbols.test.ts`
- `pnpm --dir web/frontend run typecheck`
- `go test ./...`
- `go vet ./...`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build` 首次在并行检查中于产物输出后出现 `Segmentation fault: 11`；单独重跑通过。
- `scripts/quality-gate.sh`
- `git diff --check`
- `docker compose build api`
- `docker compose up -d --no-deps api`
- `docker inspect --format '{{.State.Health.Status}}' tictick-hi-api-1` 返回 `healthy`。
- `curl -fsSI http://127.0.0.1:8080/research` 返回 `HTTP/1.1 200 OK`。
- `SMOKE_SAMPLES=40 SMOKE_INTERVAL_MS=150 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：桌面 `doc 1238->1238, panel 680->680, body 603->603, chart 603->603, tv 603->603`；移动 `doc 1256->1256, panel 624->624, body 457->457, chart 457->457, tv 457->457`。

剩余风险：

- 这仍不是生产级 instrument registry：格式合法不代表交易所存在该标的，也没有交易所元数据缓存、搜索分页、退市/停牌状态或跨页面共享市场选择器。

### 阶段 1 K 线图表外部固定槽观察补充

执行时间：2026-06-28

触发问题：

- 用户再次反馈前端 K 线图表界面会无限拉高，直到页面崩掉。
- 真实 8080 初始和长采样未复现增长，但组件仍以自己的 `.trading-chart` root 作为 ResizeObserver 输入，图表库内部 resize 写入与组件 resize 读入之间仍应进一步隔离。

修复范围：

- `TradingViewChart` 的 ResizeObserver 输入改为最近的声明式固定图表槽：研究页为 `.research-chart-body`，详情页为 `.chart-panel`。
- 组件不观察传给 lightweight-charts 的 mount canvas，也不响应 `.trading-chart` root、canvas 或图表库内部节点的 resize entry。
- 单测改为模拟内部 root / canvas 异常膨胀，验证初始化和 resize 只信任固定图表槽尺寸。

验证：

- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts src/pages/ResearchPage.layout.test.ts`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `docker compose up -d --build api`
- `docker inspect` 显示 `tictick-hi-api-1` healthy。
- `curl -I http://127.0.0.1:8080/research` 返回 `HTTP/1.1 200 OK`。
- 重建后真实 8080 采样 `SMOKE_SAMPLES=80 SMOKE_INTERVAL_MS=150 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：桌面 `doc 1238->1238, panel 680->680, body 603->603, chart 603->603, tv 603->603`；移动 `doc 1256->1256, panel 624->624, body 457->457, chart 457->457, tv 457->457`。

剩余风险：

- 本轮没有在用户的可视 Chrome 会话中复现原始无限增长；当前本地 8080 构建已通过 headless Chrome 长采样。

### 阶段 1 K 线图表固定槽高度反馈补充

执行时间：2026-06-28

触发问题：

- 用户再次反馈前端 K 线图表界面会无限拉高，直到页面崩掉。
- 真实 8080 构建在 headless Chrome 中未复现持续增长，但旧实现仍允许固定图表槽在 `clientHeight` 暂不可用时使用 `ResizeObserver` content height 作为高度输入，保留了被内部图表布局反馈污染的入口。

修复范围：

- `TradingViewChart` 对 `data-chart-viewport="fixed"` 宿主的高度读取收敛为 `clientHeight -> computed height -> bounds height`。
- 固定图表槽不再把 `ResizeObserver` content height 作为 fallback；observer height 只保留给非固定宿主兼容路径。
- 单测覆盖固定槽 `clientHeight=0`、observer height 异常变大时，图表初始化和 resize 仍只使用 CSS 声明高度。

验证：

- `pnpm --dir web/frontend run test -- TradingViewChart ResearchPage.layout`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run build`
- `docker compose build api`
- `docker compose up -d --no-deps api`
- `docker inspect` 显示 `tictick-hi-api-1` healthy。
- `node scripts/research-chart-height-smoke.mjs` 通过：桌面 `doc 1238->1238, panel 680->680, body 603->603, chart 603->603, tv 603->603`；移动 `doc 1256->1256, panel 624->624, body 457->457, chart 457->457, tv 457->457`。
- `go test ./...`
- `go vet ./...`
- `pnpm --dir web/frontend run test`
- `scripts/quality-gate.sh`

剩余风险：

- 本轮未在用户可视 Chrome 会话中捕获原始崩溃栈；当前本地 8080 已更新为本轮镜像并通过桌面/移动连续高度采样。

### 阶段 1 K 线图表固定槽 CSS 高度优先补充

执行时间：2026-06-28

触发问题：

- 用户继续反馈前端 K 线图表界面会无限拉高，直到页面崩掉。
- 真实 8080 的 headless Chrome 采样未复现增长，但代码复查发现固定图表槽高度仍优先读取 `clientHeight`，再读取 CSS computed height；如果真实浏览器中的 `clientHeight` 被图表内部布局污染，仍可能把异常高度送入 `chart.resize()`。

修复范围：

- `TradingViewChart` 对 `data-chart-viewport="fixed"` 宿主的高度读取改为 `computed height -> clientHeight -> bounds height`，固定槽优先信任页面声明的 CSS 高度。
- `ResizeObserver` 不再保存或回放 observer content box 宽高，只作为“重新测量”信号，避免 observer 回报的污染高度成为图表 resize 输入。
- 单测新增固定槽 `clientHeight=5000px`、observer height `9000px`，但 CSS height 为 `603px` 的污染场景，验证图表初始化和 resize 均以 CSS 声明高度为准。

验证：

- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts src/pages/ResearchPage.layout.test.ts`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `go test ./...`
- `go vet ./...`
- `scripts/quality-gate.sh`
- `docker compose build api`
- `docker compose up -d --no-deps api`
- `docker inspect` 显示 `tictick-hi-api-1` healthy。
- `curl -I http://127.0.0.1:8080/research` 返回 `HTTP/1.1 200 OK`。
- 真实 8080 采样 `SMOKE_SAMPLES=100 SMOKE_INTERVAL_MS=150 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：桌面 `doc 1238->1238, panel 680->680, body 603->603, chart 603->603, tv 603->603`；移动 `doc 1256->1256, panel 624->624, body 457->457, chart 457->457, tv 457->457`。

剩余风险：

- 本轮仍未在用户的可视 Chrome 会话中捕获原始无限增长栈；当前代码已切断固定槽 observer content height 和污染 `clientHeight` 优先级两条反馈入口。

### 阶段 1 K 线图表固定槽增长反馈补充

执行时间：2026-06-28

触发问题：

- 用户继续反馈前端 K 线图表界面会无限拉高，直到页面崩掉。
- 本地真实 8080 研究页 headless Chrome 采样未复现持续增长，但代码复查发现固定图表槽即使已优先读取 CSS 高度，仍会在窗口尺寸和图表宽度不变时接受更大的宿主高度，并把它送入 `chart.resize()`。
- 这种“同一 viewport、同一宽度下只增高”的变化更像图表 DOM / ResizeObserver 反馈，而不是用户可见布局变化。

修复范围：

- `TradingViewChart` 记录上一次窗口尺寸和图表尺寸。
- 对 `data-chart-viewport="fixed"` 宿主，如果窗口宽高和图表宽度都没有变化，则拒绝高度增长，只保留上一次图表高度。
- 固定槽高度收缩、宽度变化或真实窗口变化仍允许重新计算，避免完全锁死响应式布局。
- `scripts/research-chart-height-smoke.mjs` 不再只检查高度稳定，还检查 panel / body / chart / canvas / lightweight-charts 根节点没有稳定在超过 viewport 的异常高度。

验证：

- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts src/pages/ResearchPage.layout.test.ts`
- `go test ./...`
- `go vet ./...`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `scripts/quality-gate.sh`
- `git diff --check`
- `docker compose build api`
- `docker compose up -d --no-deps api`
- `docker inspect --format '{{.State.Health.Status}}' tictick-hi-api-1` 返回 `healthy`。
- `curl -fsSI http://127.0.0.1:8080/research` 返回 `HTTP/1.1 200 OK`。
- `SMOKE_SAMPLES=100 SMOKE_INTERVAL_MS=150 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：桌面 `doc 1238->1238, panel 680->680, body 603->603, chart 603->603, tv 603->603`；移动 `doc 1256->1256, panel 624->624, body 457->457, chart 457->457, tv 457->457`。

剩余风险：

- 本轮仍未在用户的可视 Chrome 会话中捕获原始无限增长栈；当前修复关闭的是固定图表槽高度增长反馈入口，不是完整桌面/移动/主题视觉回归体系。

### 阶段 1 K 线图表内部高度污染补充

执行时间：2026-06-28

触发问题：

- 用户继续反馈前端 K 线图表界面会无限拉高直到页面崩掉。
- 现有 8080 headless Chrome 连续采样仍未自然复现增长，但旧实现对固定图表槽仍会在没有声明式 CSS 高度时回退信任 `clientHeight`，这会给内部图表 DOM 高度污染留下入口。
- 原 smoke 只验证自然采样稳定，没有主动模拟 lightweight-charts 内部节点被写成异常高度的情况。

修复范围：

- `TradingViewChart` 对 `data-chart-viewport="fixed"` 宿主的高度读取改为只信任声明式 CSS `height` / `max-height`，或不超过 viewport cap 的 bounds；不再把固定槽 `clientHeight` 作为高度来源。
- `.tv-lightweight-charts` 内部 table 和 canvas 增加 `max-height: 100%` / 固定高度裁剪，内部节点即使被写入超大高度也不能撑开固定 viewport。
- `scripts/research-chart-height-smoke.mjs` 在每次采样前主动把 `.tv-lightweight-charts`、内部 table 和 canvas 写成 `9000px`，并断言 chart / canvas / tv 高度必须贴合 `.research-chart-body`。

验证：

- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts src/pages/ResearchPage.layout.test.ts`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `SMOKE_SAMPLES=40 SMOKE_INTERVAL_MS=150 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：桌面 `doc 1238->1238, panel 680->680, body 603->603, chart 603->603, tv 603->603`；移动 `doc 1256->1256, panel 624->624, body 457->457, chart 457->457, tv 457->457`。

剩余风险：

- 本轮仍未捕获用户可视 Chrome 会话里的原始无限增长栈；当前修复和 smoke 证明内部图表高度污染不会再突破固定 viewport，但仍不是完整视觉回归体系。

### 阶段 1 K 线图表固定槽高度零信任补充

执行时间：2026-06-28

触发问题：

- 用户继续反馈前端 K 线图表界面会无限拉高直到页面崩掉。
- 真实 8080 研究页长采样没有自然复现持续增长，但代码复查发现旧 guard 只在窗口和图表宽度都不变时拒绝高度增长；如果 scrollbar 或布局抖动引起宽度变化，固定槽观测到的污染高度仍可能进入 `chart.resize()`。

修复范围：

- `TradingViewChart` 对 `data-chart-viewport="fixed"` 宿主改为窗口尺寸不变时拒绝任何固定槽高度变化反馈。
- 宽度变化但窗口未变化时，只把新宽度送入 `chart.resize()`，高度继续沿用上一次可信高度。
- 固定槽高度变化只在真实 `window.innerWidth/innerHeight` 变化后接受，保留桌面/移动响应式能力。
- 单测覆盖窗口不变时高度增减都被拒绝、仅宽度变化时不接受污染高度，以及窗口变化后才允许新 CSS 高度生效。

验证：

- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts src/pages/ResearchPage.layout.test.ts`
- `go test ./...`
- `go vet ./...`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `scripts/quality-gate.sh`
- `git diff --check`
- `docker compose build api`
- `docker compose up -d --no-deps api`
- `docker inspect --format '{{.State.Health.Status}}' tictick-hi-api-1` 返回 `healthy`。
- `curl -fsSI http://127.0.0.1:8080/research` 返回 `HTTP/1.1 200 OK`。
- `SMOKE_SAMPLES=100 SMOKE_INTERVAL_MS=150 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：桌面 `doc 1238->1238, panel 680->680, body 603->603, chart 603->603, tv 603->603`；移动 `doc 1284->1284, panel 652->652, body 457->457, chart 457->457, tv 457->457`。

剩余风险：

- 本轮仍未捕获用户可视 Chrome 会话里的原始无限增长栈；当前修复关闭的是固定槽在窗口不变时接受任何高度反馈的入口，不是完整视觉回归体系。

### 阶段 1 K 线图表固定槽 inline 高度污染补充

执行时间：2026-06-28

触发问题：

- 用户继续反馈前端 K 线图表界面会无限拉高，直到页面崩掉。
- 本地 8080 当前构建的 headless Chrome 高度 smoke 未自然复现持续增长，但代码复查发现固定槽高度读取仍优先读取 computed `height`，如果运行态把固定槽本身写入异常 inline height，首次初始化或真实窗口 resize 后仍可能把污染高度送入 `chart.resize()`。

修复范围：

- `TradingViewChart` 对 `data-chart-viewport="fixed"` 宿主改为优先读取 CSS `max-height`，再回退 `height` / bounds，避免运行态 inline `height` 覆盖固定槽声明式高度。
- 固定槽高度按 `window.innerWidth/innerHeight` 快照缓存；窗口尺寸不变时，后续 ResizeObserver 反馈不会重新读取或接受宿主高度变化。
- `TradingViewChart.css` 进一步约束 lightweight-charts 外层、table、table cell 和 canvas 的 `block-size/max-block-size/overflow`，防止内部布局高度突破固定 viewport。
- `scripts/research-chart-height-smoke.mjs` 将 `.research-chart-body` 本身加入污染对象，验证固定槽被写入 `9000px` height 后 document / panel / body / chart / tv 高度仍稳定。

验证：

- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts src/pages/ResearchPage.layout.test.ts` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：18 个测试文件、74 个测试通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- `docker compose build api` 通过。
- `docker compose up -d --no-deps api` 后 `docker inspect --format '{{.State.Health.Status}}' tictick-hi-api-1` 返回 `healthy`。
- `curl -fsSI http://127.0.0.1:8080/research` 返回 `HTTP/1.1 200 OK`，且页面入口已更新为 `/assets/index-5FiAzABM.js`。
- `SMOKE_SAMPLES=40 SMOKE_INTERVAL_MS=150 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：桌面 `doc 1238->1238, panel 680->680, body 603->603, chart 603->603, tv 603->603`；移动 `doc 1284->1284, panel 652->652, body 457->457, chart 457->457, tv 457->457`。
- `go test ./...` 通过。
- `go vet ./...` 通过。

剩余风险：

- 仍未捕获用户可视 Chrome 会话里的原始无限增长栈；本轮修复关闭的是固定槽 inline height 污染和内部 table/canvas 污染入口，不代表完整视觉回归体系。

### 阶段 1 K 线图表固定槽 observer 零信任补充

执行时间：2026-06-28

触发问题：

- 用户继续反馈前端 K 线图表界面会无限拉高，直到页面崩掉。
- 本地 8080 真实构建的 headless Chrome 长采样未自然复现持续增长，但既有实现仍让固定槽 `ResizeObserver` 触发完整尺寸重读；在真实 Chrome 会话中如果固定槽或图表内部 DOM 持续发出高度变化，仍可能形成高频测量 / 写回路径。

修复范围：

- `TradingViewChart` 对 `data-chart-viewport="fixed"` 宿主改为只把 `ResizeObserver` 当作宽度变化通知。
- 固定槽高度只在初始化和真实 `window.resize` 时从声明式 CSS 高度读取；observer 触发时不会重新读取或接受任何宿主高度。
- lightweight-charts 创建时显式设置 `autoSize: false`，避免库内部 ResizeObserver 与外部手动 resize 形成双通道尺寸控制。
- 单测调整为验证固定槽高度刷新只能来自 `window.resize`，而不是 ResizeObserver 的 height entry。

验证：

- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts src/pages/ResearchPage.layout.test.ts` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：18 个测试文件、77 个测试通过。
- `pnpm --dir web/frontend run build` 通过，生产入口为 `/assets/index-ZKMZqIFC.js`。
- `docker compose build api` 通过。
- `docker compose up -d --no-deps api` 后 `docker inspect --format '{{.State.Health.Status}}' tictick-hi-api-1` 返回 `healthy`。
- `curl -fsSI http://127.0.0.1:8080/research` 返回 `HTTP/1.1 200 OK`。
- `SMOKE_SAMPLES=120 SMOKE_INTERVAL_MS=120 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：桌面 `doc 1238->1238, panel 680->680, body 603->603, chart 603->603, tv 603->603`；移动 `doc 1284->1284, panel 652->652, body 457->457, chart 457->457, tv 457->457`。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。

剩余风险：

- 本轮仍未捕获用户可视 Chrome 会话中的原始无限增长栈；当前修复关闭的是固定槽 ResizeObserver 高度反馈入口和 lightweight-charts autoSize 双通道风险，不代表完整桌面 / 移动 / 主题视觉回归体系。

### 阶段 1 K 线图表污染高度拒绝补充

执行时间：2026-06-28

目标等级：demo

触发问题：

- 用户继续反馈前端 K 线图表界面会无限拉高，直到页面崩掉。
- 现有固定槽逻辑已忽略 `ResizeObserver` 的高度 entry，但在 `window.resize` 触发完整重读时，仍可能把被污染成超大值的 host `height` / bounds 当作候选高度再喂给 `chart.resize`。

修复范围：

- `TradingViewChart` 固定 viewport 高度读取新增上限过滤：CSS `max-height`、CSS `height` 和 bounds 只有在大于 0 且不超过当前 viewport 高度上限时才会被接受。
- 如果固定槽候选高度已被污染到 `9000px` 这类值，组件会忽略该值并沿用上一次固定高度快照；没有快照时回退到安全默认高度。
- 新增回归测试覆盖 `window.resize` + 宿主宽度变化 + 宿主高度污染为 `9000px` 的场景，要求图表只更新宽度，不吸收污染高度。
- 本地 Docker API 已重建并重启，`/research` 当前由新前端 dist 提供。

验证：

- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts` 通过：12 个测试通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：20 个测试文件、82 个测试通过。
- `pnpm --dir web/frontend run build` 通过，生产入口为 `/assets/index-Dr9QVqKa.js`。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- `docker compose build api` 通过。
- `docker compose up -d --no-deps api` 后 `docker inspect --format '{{.State.Health.Status}}' tictick-hi-api-1` 返回 `healthy`。
- `curl -fsSI http://127.0.0.1:8080/research` 返回 `HTTP/1.1 200 OK`，`Last-Modified: Sun, 28 Jun 2026 09:02:06 GMT`。
- `SMOKE_SAMPLES=120 SMOKE_INTERVAL_MS=120 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：桌面 `doc 1238->1238, panel 680->680, body 603->603, chart 603->603, tv 603->603`；移动 `doc 1284->1284, panel 652->652, body 457->457, chart 457->457, tv 457->457`。
- 重新应用新镜像内 migration 后，`0025_market_instruments.sql` 已进入 `schema_migrations`，`GET /api/market/instruments?exchange=binance&q=SOL&limit=5` 返回 `SOLUSDT`。
- `SMOKE_SAMPLES=60 SMOKE_INTERVAL_MS=120 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 再次通过：桌面 `doc 1238->1238, panel 680->680, body 603->603, chart 603->603, tv 603->603`；移动 `doc 1284->1284, panel 652->652, body 457->457, chart 457->457, tv 457->457`。

剩余风险：

- 本轮仍未拿到用户可视 Chrome 会话中的原始增长堆栈；当前关闭的是固定槽超大高度污染被 `window.resize` 重读接受的入口，以及本地 headless Chrome 桌面/移动高度漂移风险，不等于完整视觉回归体系。

### 阶段 1 K 线图表固定槽 CSS 硬边界补充

执行时间：2026-06-28

目标等级：demo

触发问题：

- 用户继续反馈前端 K 线图表界面会无限拉高，直到页面崩掉。
- 现有 headless Chrome 自然采样和高度污染 smoke 都未复现持续增长，但固定图表槽仍只防住普通 `height` 污染；如果运行态或浏览器异常把 `max-height` / `block-size` / `max-block-size` 一起污染，CSS 边界仍不够硬。

修复范围：

- `ResearchPage` 的 `.research-chart-body` 固定槽对 `height`、`max-height`、`block-size`、`max-block-size` 使用声明式固定高度并加 `!important`，阻断运行态 inline 高度污染继续撑开页面。
- 研究页深层 `.trading-chart`、通用 `TradingViewChart` root 和 canvas host 均改为 `height/max-height/block-size/max-block-size: 100% !important`，继续限制 lightweight-charts 内部 DOM 只能铺满固定宿主。
- `scripts/research-chart-height-smoke.mjs` 的污染场景从只写 `height=9000px` 升级为同时写 `height/max-height/block-size/max-block-size=9000px`。
- `ResearchPage.layout.test.ts` 增加固定槽硬边界契约断言。

验证：

- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts src/pages/ResearchPage.layout.test.ts` 通过：17 个测试通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：20 个测试文件、84 个测试通过。
- `pnpm --dir web/frontend run build` 通过，生产入口为 `/assets/index-BOT9mGld.js`。
- `docker compose build api` 通过。
- `docker compose up -d --no-deps api` 后 `docker inspect --format '{{.State.Health.Status}}' tictick-hi-api-1` 返回 `healthy`。
- `curl -fsSI http://127.0.0.1:8080/research` 返回 `HTTP/1.1 200 OK`，`Last-Modified: Sun, 28 Jun 2026 09:56:41 GMT`。
- `curl http://127.0.0.1:8080/assets/ResearchPage-_0E2Tmyj.css` 确认产物包含固定槽 `height/max-height/block-size/max-block-size` 的 `!important` 边界。
- `curl http://127.0.0.1:8080/assets/TradingViewChart-CAKikaGH.css` 确认产物包含 chart root / canvas host `100% !important` 高度边界。
- `SMOKE_SAMPLES=80 SMOKE_INTERVAL_MS=100 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：桌面 `doc 1238->1238, panel 680->680, body 603->603, chart 603->603, tv 603->603`；移动 `doc 1284->1284, panel 652->652, body 457->457, chart 457->457, tv 457->457`。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

剩余风险：

- 本轮仍未拿到用户可视 Chrome 会话中的原始增长堆栈；当前修复关闭的是固定槽多属性 inline 高度污染和 chart root/canvas host 高度外溢入口，不等于完整桌面 / 移动 / 主题视觉回归体系。

### 阶段 1 K 线图表受控尺寸锁补充

执行时间：2026-06-28

目标等级：demo

触发问题：

- 用户再次反馈前端 K 线图表界面会无限拉高，直到页面崩掉。
- 既有 smoke 在本地 8080 通过，说明旧验证能证明固定槽污染被裁剪，但没有把组件自身 root / canvas / lightweight-charts 内部节点统一锁到同一份受控像素尺寸。

修复范围：

- `TradingViewChart` 初始化和 resize 时把最近 `data-chart-viewport="fixed"` 固定槽测得的宽高写入 `--tt-chart-render-width` / `--tt-chart-render-height`，并对 root 与 canvas host 设置同尺寸 inline hard lock。
- `TradingViewChart.css` 将 root、canvas host、`.tv-lightweight-charts`、内部 table 和 canvas 的 width / height / max-size 统一改为受控 CSS 变量，阻断内部 DOM 或运行态 inline height 反向撑开父级。
- resize 后即使测得尺寸没有变化，也会重新应用上一次受控尺寸，覆盖运行态节点污染后外层尺寸不变的恢复场景。
- `TradingViewChart.test.ts` 新增受控尺寸锁和运行态污染恢复断言。

验证：

- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts src/pages/ResearchPage.layout.test.ts src/components/tables/DataSyncTaskTable.test.ts` 通过：3 个测试文件、26 个测试通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：20 个测试文件、87 个测试通过。
- `pnpm --dir web/frontend run build` 通过，生产入口为 `/assets/index-Djw7oz3o.js`，图表 CSS 为 `/assets/TradingViewChart-D6RFVE4h.css`。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- `docker compose build api` 通过。
- `docker compose up -d --no-deps api` 后 `docker inspect --format '{{.State.Health.Status}}' tictick-hi-api-1` 返回 `healthy`。
- `curl http://127.0.0.1:8080/research` 确认页面引用 `/assets/index-Djw7oz3o.js`。
- `curl http://127.0.0.1:8080/assets/TradingViewChart-D6RFVE4h.css` 确认产物包含 `--tt-chart-render-height`、`.tv-lightweight-charts` 和 canvas 的受控尺寸变量。
- `SMOKE_SAMPLES=120 SMOKE_INTERVAL_MS=100 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：桌面 `doc 1238->1238, panel 680->680, body 603->603, chart 603->603, tv 603->603`；移动 `doc 1284->1284, panel 652->652, body 457->457, chart 457->457, tv 457->457`。

剩余风险：

- 本轮仍未拿到用户可视 Chrome 会话中的原始增长堆栈；当前修复关闭的是 chart root / canvas / lightweight-charts 内部节点与固定槽尺寸不一致导致的高度反馈入口，不等于完整视觉回归体系。

### 阶段 1 instrument catalog 搜索基础补充

执行时间：2026-06-28

目标等级：demo

触发问题：

- 阶段 1 研究核心仍缺生产级 instrument 搜索 / 在线校验。
- 研究页、回测创建和交易创建虽然已有 exchange-specific symbol 格式校验，但前端建议项仍主要来自本地静态 fallback，不能从后端市场元数据演进。

修复范围：

- 新增 `market_instruments` PostgreSQL catalog，保存 exchange / symbol / base / quote / instrument type / status / search priority / synced_at / 时间戳。
- 新增 migration `0025_market_instruments.sql`，幂等 seed Binance / OKX 常见 USDT spot instrument。
- 新增 `GET /api/market/instruments?exchange=&q=&limit=`，受登录认证保护，limit clamp 到 50，OpenAPI contract 和生成 TypeScript DTO 已覆盖。
- 新增 PostgreSQL `ListMarketInstruments`，只返回 active instrument，并按匹配度、priority、symbol 排序。
- 研究页、数据同步创建弹窗、回测创建和交易创建页的 symbol 输入改为共用 `MarketSymbolAutoComplete`，优先读取后端 catalog，失败时回退本地格式建议。

验证：

- `go test ./internal/web/api -run 'TestMarketInstrumentRoutes|TestAPIContract|TestAPIMethodNotAllowedContracts|TestFrontendAPI|TestWriteGeneratedFrontendAPITypes|TestFrontendAPIGeneratedTypesAreCurrent' -count=1` 通过。
- `go test ./internal/store/postgres -run 'TestIntegrationListMarketInstruments' -count=1` 通过。
- `pnpm --dir web/frontend exec vitest run src/services/api/market.test.ts src/components/market/MarketSymbolAutoComplete.test.ts src/pages/ResearchPage.layout.test.ts src/pages/StrategyTaskFormPage.layout.test.ts` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：20 个测试文件、81 个测试通过。
- `pnpm --dir web/frontend run build` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。

剩余风险：

- 这不是生产级在线 instrument 校验：当前 catalog 只 seed 常见 spot instrument，尚未从 Binance `/exchangeInfo` 或 OKX public instruments 定时同步，不能证明任意输入标的真实存在、可交易或未退市。
- 后端创建 data sync / backtest / trading task 仍按格式校验放行，不强制 catalog 命中，避免 seed catalog 不全时误阻断用户。

### 阶段 1 instrument catalog 真实同步补充

执行时间：2026-06-28

目标等级：demo

触发问题：

- 上一轮 instrument catalog 仍依赖 seed 常见交易对，无法从真实交易所公开元数据更新。
- 研究页 symbol 建议项虽然接入 PostgreSQL catalog，但没有用户可触发的同步入口，catalog 过期后只能回退本地静态建议。

修复范围：

- Binance adapter 新增 `/api/v3/exchangeInfo` spot instrument 拉取，解析 `symbol/baseAsset/quoteAsset/status/isSpotTradingAllowed`，只保留 spot instrument，`TRADING` 映射为 active，其它 spot 状态映射为 inactive。
- OKX adapter 新增 `/api/v5/public/instruments?instType=SPOT` 拉取，解析 `instId/baseCcy/quoteCcy/state`，`live` 映射为 active，其它状态映射为 inactive。
- 新增 `POST /api/market/instruments/sync?exchange=`，受 session + CSRF 保护，`hi api` 启动时注入 Binance / OKX instrument client。
- PostgreSQL `ReplaceMarketInstruments` 在事务中 upsert 本次返回 instrument，并把同交易所本次未返回的旧 active instrument 标记为 inactive；搜索 API 仍只返回 active。
- `MarketSymbolAutoComplete` 右侧新增刷新按钮，触发真实 catalog 同步后重新加载建议项；同步失败时仍回退本地建议。
- OpenAPI contract 新增 sync 路径，生成 TypeScript DTO 维持在 399 行硬上限内。

验证：

- `go test ./internal/adapter/binance ./internal/adapter/okx ./internal/web/api ./internal/store/postgres -run 'TestFetchInstruments|TestMarketInstrument|TestAPIContract|TestAPIMethodNotAllowedContracts|TestFrontendAPI|TestWriteGeneratedFrontendAPITypes|TestFrontendAPIGeneratedTypesAreCurrent|TestIntegrationReplaceMarketInstruments|TestIntegrationListMarketInstruments' -count=1` 通过。
- `pnpm --dir web/frontend exec vitest run src/services/api/market.test.ts src/components/market/MarketSymbolAutoComplete.test.ts src/pages/ResearchPage.layout.test.ts src/pages/StrategyTaskFormPage.layout.test.ts` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：20 个测试文件、84 个测试通过。
- `pnpm --dir web/frontend run build` 通过，生产入口为 `/assets/index-_im0Y4Jb.js`。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- `docker compose build api` 通过。
- `docker compose up -d --no-deps api` 后 `docker inspect --format '{{.State.Health.Status}}' tictick-hi-api-1` 返回 `healthy`。
- `curl -fsSI http://127.0.0.1:8080/research` 返回 `HTTP/1.1 200 OK`。
- 本地 8080 `POST /api/market/instruments/sync?exchange=binance` 通过，返回 `activeCount=1359`；随后 `GET /api/market/instruments?exchange=binance&q=SOL&limit=3` 返回 `SOLUSDT` 以及同步后的 `SOLBNB` / `SOLBRL`。
- 本地 8080 `POST /api/market/instruments/sync?exchange=okx` 连续 3 次返回 `request_failed`，错误摘要为 `www.okx.com: EOF`；宿主侧 `curl https://www.okx.com/api/v5/public/instruments?instType=SPOT` 返回 `SSL_ERROR_SYSCALL`，因此当前环境未证明 OKX 实网同步可用。

剩余风险：

- 后台定时同步 worker 已在后续补充中收敛；仍需继续补跨实例调度锁、同步历史和真实交易所状态语义。
- 创建 backtest / trading task 的 API 层 catalog 强制命中已在后续补充中收敛；仍需继续补真实交易所状态语义和后台自动同步证据。
- 当前环境到 OKX 公共 API 的 TLS/EOF 失败未关闭；代码路径由 httptest 覆盖，但 OKX 实网同步未通过本地验证。
- 未实现交易所权重级限流、增量同步、水位观测、同步失败重试队列和退市/停牌状态在创建任务时的阻断策略，因此不能升级为 usable。

### 阶段 1 data sync task catalog 强制命中补充

执行时间：2026-06-28

目标等级：demo

触发问题：

- 阶段 1 研究页已能从 PostgreSQL instrument catalog 搜索和手动同步交易对，但数据同步任务创建仍可绕过 catalog，只凭交易所格式校验落库。
- 这会让用户创建本地 catalog 不存在或已 inactive 的交易对，后续 worker 才在交易所 adapter 层报错，不符合“研究页 + 数据同步 + PostgreSQL 可观察”的垂直切片边界。

修复范围：

- `Repository` 增加 `GetActiveMarketInstrument`，PostgreSQL 实现按 `exchange + symbol + status='active'` exact lookup，不命中返回 `data.ErrNotFound`。
- `POST /api/data/tasks` 在创建任务前强制调用 active instrument lookup，不命中返回 HTTP 400 和领域错误码 `market_instrument_not_active`，且不落库。
- API error catalog 和生成的前端 TypeScript `APIErrorCode` 同步加入 `market_instrument_not_active`。
- 研究页创建同步任务前调用 `/api/market/instruments?exchange=&q=&limit=1` 做 exact active catalog 预校验；无法校验或无 exact active 命中时不调用创建 API，并显示明确错误。
- Stage 8 smoke 和 SIGTERM smoke 在创建合成 data sync task 前 seed 对应 active `market_instruments`，避免 smoke 因测试 symbol 不在 catalog 中误失败。

验证：

- `go test ./internal/web/api ./internal/store/postgres -run 'TestDataSyncTaskRoutes|TestAPIError|TestAPIContract|TestAPIMethodNotAllowedContracts|TestFrontendAPI|TestWriteGeneratedFrontendAPITypes|TestFrontendAPIGeneratedTypesAreCurrent|TestIntegration.*MarketInstrument' -count=1` 通过。
- `pnpm --dir web/frontend exec vitest run src/composables/useResearchWorkspace.test.ts src/services/api/market.test.ts src/components/market/MarketSymbolAutoComplete.test.ts` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：20 个测试文件、87 个测试通过。
- `pnpm --dir web/frontend run build` 通过，生产入口为 `/assets/index-Djw7oz3o.js`。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- `docker compose build api` 通过。
- `docker compose up -d --no-deps api` 后 `docker inspect --format '{{.State.Health.Status}}' tictick-hi-api-1` 返回 `healthy`。
- 本地 8080 登录后 `POST /api/data/tasks` 创建 `NOTREALUSDT` 返回 HTTP 400，响应 `code=market_instrument_not_active`。
- 本地 8080 登录后 `POST /api/data/tasks` 创建 active `BTCUSDT` 返回 HTTP 201，并已删除该临时验证任务。
- `scripts/stage8-smoke.sh` 顺序重跑通过，证明 Stage 8 合成 data sync task seed active catalog 后全链路 smoke 仍可走通。
- `scripts/stage8-sigterm-smoke.sh` 顺序重跑通过，证明 Stage 8 SIGTERM smoke 的合成 data sync task seed active catalog 后仍可走通。

剩余风险：

- backtest / trading task 创建 API 层 active catalog 强制命中已在后续补充中收敛；仍需继续补真实交易所状态语义和后台自动同步证据。
- active catalog 已接入后台定时同步和 stale 健康判断；仍没有跨实例调度锁、交易所级权重限流、退市/停牌完整操作语义和自动重试队列，因此不能升级为 usable。

### 阶段 1 策略任务 catalog 强制命中补充

执行时间：2026-06-30

目标等级：scaffold

触发问题：

- 回测 / 交易创建页前端已经在提交前做 `market_instruments` exact active 预校验，但 API 层仍只做 exchange-specific symbol 格式校验。
- 这意味着绕过前端即可创建本地 catalog 不存在或已 inactive 的回测 / 交易任务，后续 worker 才在 CandleProvider 或交易 runner 阶段失败，不符合阶段 1 数据源前门边界。

修复范围：

- `POST /api/backtests` 在策略 schema 校验通过后、落库前调用 `GetActiveMarketInstrument`；缺失或 inactive 返回 HTTP 400 和 `market_instrument_not_active`，且不落库。
- `POST /api/trading/tasks` 在策略 schema 校验通过后、账号校验和落库前调用同一 active catalog 检查；缺失或 inactive 返回 HTTP 400 和 `market_instrument_not_active`，且不落库。
- 复用既有 `requireActiveMarketInstrument` 和 `market_instrument_not_active` 错误码，不新增 migration，不改变 data sync task 已有语义。

验证：

- `go test ./internal/web/api` 通过，新增覆盖 inactive backtest 和 missing trading task 均返回 `market_instrument_not_active` 且不落库。
- `pnpm --dir web/frontend exec vitest run src/composables/useStrategyTaskForm.test.ts src/pages/StrategyTaskFormPage.layout.test.ts src/services/api/market.test.ts` 通过，确认前端创建页仍在提交前做 exact active catalog 预检。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：31 个测试文件、155 个测试通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- `docker compose build api && docker compose up -d --no-deps api` 后 `curl -fsS http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`。
- 本地 8080 登录后直接 `POST /api/backtests` 创建不存在 catalog 的 `MISSINGUSDT` 返回 HTTP 400，响应 `code=market_instrument_not_active`。

剩余风险：

- 这只关闭任务创建 API 可绕过 catalog 的边界，不证明 catalog 自动保持最新。
- 仍缺交易所业务状态细分、后台自动同步/重试队列、真实外部交易所恢复压测和跨模块迁移语义，因此阶段 1 不能升级为 usable。

### 阶段 8 SIGTERM smoke 状态机兼容补充

执行时间：2026-06-28

目标等级：demo

触发问题：

- 顺序重跑 `scripts/stage8-sigterm-smoke.sh` 时，脚本在证明 backtest worker SIGTERM 释放 lease 后，直接把 backtest 任务从 `pending` 更新为 `cancelled`，触发既有 `backtest_tasks_status_transition_check`。
- notify 段 seed 的 `notifications.task_id` 使用不存在的 synthetic notify task id，触发 `notifications_trading_task_fk`。

修复范围：

- backtest proof 后清理改为 `pending -> running -> failed`，符合现有 backtest 状态机，不放宽数据库 trigger。
- 历史 `S8TERM%` backtest 清理按状态分段处理：pending 先转 running，running 转 failed，terminal 状态只清锁。
- notify seed 改为复用本轮已创建的 trading task id，满足 notifications / outbox 对 trading task 的 FK 约束。

验证：

- `bash -n scripts/stage8-sigterm-smoke.sh` 通过。
- `scripts/stage8-sigterm-smoke.sh` 顺序重跑通过，输出 `Stage 8 SIGTERM smoke passed`。
- `scripts/stage8-smoke.sh` 顺序重跑通过，输出 `Stage 8 smoke passed`。

剩余风险：

- 本轮没有把 SIGTERM smoke 并发执行变成受支持场景；这两个 Stage 8 脚本仍应顺序运行，避免共享 compose project 的临时容器名冲突。

### 阶段 1 数据同步任务健康可观察补充

执行时间：2026-06-28

目标等级：demo

范围内：

- `DataSyncTask` API 增加派生字段 `dataHealth`，取值为 `ok / syncing / gap / failed / paused / retrying / insufficient / invalid`。
- `ListDataSyncTasks` 使用真实 PostgreSQL `market_candles` 相邻 `open_time` 窗口函数检测任务同交易所、同交易对、同周期窗口内缺口。
- 写操作返回的 `DataSyncTask` 使用任务状态、`next_attempt_at` 和 `last_synced_open_time` 派生健康状态，避免写路径引入额外重查询。
- 研究页任务表新增“数据健康”列。
- 概览页数据同步缺口告警改为按 `dataHealth=gap` 统计，不再把 `TaskStatus` 当成缺口状态。
- API contract 和生成的前端 TypeScript DTO 同步更新。

范围外：

- 不做全历史缺口扫描。
- 不做交易所权重限流。
- 不重构完整 worker 状态机。
- 不改变同步游标推进语义。

验证：

- `go test ./internal/web/api ./internal/store/postgres`
- `TICTICK_WRITE_GENERATED_API_TYPES=1 go test ./internal/web/api -run TestWriteGeneratedFrontendAPITypes`
- `docker run --rm --network tictick-hi_default ... go test ./internal/store/postgres -run TestIntegrationListDataSyncTasksReportsDataHealth -count=1 -v` 通过，真实 PostgreSQL 验证 `gap / ok / syncing / paused / retrying / failed` 派生结果。
- `cd web/frontend && pnpm run test -- DataSyncTaskTable useOverviewWorkspace data`
- `cd web/frontend && pnpm run typecheck`
- `go test ./...`
- `go vet ./...`
- `cd web/frontend && pnpm run test`
- `cd web/frontend && pnpm run build`
- `scripts/quality-gate.sh`

剩余风险：

- 当前缺口检测限定在任务自身交易所、交易对、周期和同步窗口内；仍不是生产级全历史缺口扫描，也没有缺口修复工作流。

### 阶段 1 数据同步缺口摘要可观察补充

执行时间：2026-06-28

目标等级：demo

范围内：

- `DataSyncTask` API 增加可选 `gapSummary`，在任务窗口内存在相邻 K 线缺口时返回缺口数量和首个缺口 `from/to/missingCandles`。
- `ListDataSyncTasks` 继续使用 PostgreSQL `market_candles` 窗口函数推导任务同交易所、同交易对、同周期窗口内的相邻缺口；写操作返回路径不做额外重查询。
- API contract 注册 `DataSyncGapSummary`，并重新生成 `web/frontend/src/types/api.generated.ts`。
- 研究页任务表新增“缺口摘要”列，展示缺口数量和首个缺口范围，长时间戳通过 tooltip 展示，避免撑爆表格。
- 前端 API wrapper 保留 `gapSummary` 字段。

范围外：

- 不做全历史缺口扫描。
- 不新增批量修复 API。
- 不自动排队修复所有缺口。
- 不改变同步游标推进语义。
- 不证明真实交易所网络下长期恢复能力。

验证：

- `go test ./internal/store/postgres -run TestIntegrationListDataSyncTasksReportsDataHealth -count=1`
- `docker run --rm --network tictick-hi_default -v "$PWD":/src -w /src ... golang:1.26-bookworm go test ./internal/store/postgres -run TestIntegrationListDataSyncTasksReportsDataHealth -count=1 -v`
- `go test ./internal/web/api -run 'TestFrontendAPIGeneratedTypesAreCurrent|TestFrontendAPIResponseTypesMatchContractFields' -count=1`
- `cd web/frontend && pnpm run test -- DataSyncTaskTable data`
- `cd web/frontend && pnpm run typecheck`
- `go test ./...`
- `go vet ./...`
- `cd web/frontend && pnpm run test`
- `cd web/frontend && pnpm run build`
- `scripts/quality-gate.sh`

剩余风险：

- 当前 `gapSummary` 只覆盖任务窗口内已入库 K 线之间的相邻缺口；不覆盖任务起点前缺失、尾部未同步、跨任务全历史扫描和批量修复进度。

### 阶段 1 图表单缺口来源修复补充

执行时间：2026-06-28

目标等级：demo

范围内：

- 新增 `POST /api/data/tasks/{id}/repair-gap`，请求体为单个缺口 `from/to`，由后端按源同步任务创建带 `repair_source_task_id` 的补同步任务。
- 单缺口 repair 复用补同步任务窗口去重；同一 source 任务、同一 exchange / symbol / interval / start / end 已存在时返回 `skippedExisting`，不重复创建。
- `RepairDataSyncTaskGapRequest` 进入 OpenAPI contract 和生成的前端 TypeScript DTO。
- 前端 `dataApi.repairTaskGap` 接入新 endpoint。
- 研究页从任务列表“查看图表”后记录当前图表来源任务；图表首个缺口修复在来源任务和基础周期匹配时优先调用后端单缺口 repair API。
- 无匹配来源任务时保留旧 `createTask + setSync` fallback，避免 URL 直接打开图表时完全失去修复入口。

范围外：

- 不做全历史缺口扫描。
- 不改变批量 `repair-gaps` 上限。
- 不保证外部交易所拉取必然成功。
- 不给历史手工补同步任务回填 `repair_source_task_id`。

验证：

- `TICTICK_WRITE_GENERATED_API_TYPES=1 go test ./internal/web/api -run TestWriteGeneratedFrontendAPITypes -count=1`
- `go test ./internal/web/api -run 'TestDataSyncTaskRoutes|TestAPIMethodNotAllowedContracts|TestWriteGeneratedFrontendAPITypes|TestFrontendAPIGeneratedTypesAreCurrent|TestAPIContract' -count=1`
- `go test ./internal/store/postgres -run 'TestIntegrationRepairDataSyncTaskGapCreatesSyncTask|TestIntegrationRepairDataSyncTaskGapsCreatesSyncTasks' -count=1`
- `pnpm --dir web/frontend exec vitest run src/composables/useResearchWorkspace.test.ts src/services/api/data.test.ts`
- `go test ./...`
- `go vet ./...`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test` 通过：18 个测试文件、77 个测试通过。
- `pnpm --dir web/frontend run build`
- `scripts/quality-gate.sh`
- `git diff --check`
- `docker compose build api`
- `docker compose up -d --no-deps api` 后 `docker inspect --format '{{.State.Health.Status}}' tictick-hi-api-1` 返回 `healthy`。
- `curl -fsSI http://127.0.0.1:8080/research` 返回 `HTTP/1.1 200 OK`，且页面入口已更新为 `/assets/index-AN-Cff53.js`。

剩余风险：

- 从 URL 直接进入图表或基础周期不匹配时仍会走旧 fallback，创建的任务没有 source 关系；后续应让 URL 也能携带 source task id 或在后端按上下文解析来源任务。
- 该补充只证明补同步任务被正确排队和可观察，不证明外部交易所一定能补回缺失 K 线。

### 阶段 1 数据同步任务缺口详情可观察补充

执行时间：2026-06-28

目标等级：demo

范围内：

- 新增 `GET /api/data/tasks/{id}/gaps`，返回源任务窗口内前 20 个相邻 K 线缺口和 `limited` 标志。
- 后端详情查询复用 `repair-gaps` 的缺口窗口检测 SQL，但详情路径不锁源任务行，不创建补同步任务。
- API contract 注册 `DataSyncGapList`，并重新生成 `web/frontend/src/types/api.generated.ts`。
- 研究页任务表在 `gapSummary.count > 0` 时新增“查看缺口”操作，打开缺口详情弹窗，展示 `from/to/missingCandles`。
- 保留现有“修复缺口”操作，用户可以先查看详情再决定是否排修复任务。

范围外：

- 不做全历史分页扫描。
- 不推断任务起点前缺失或尾部尚未同步的 K 线。
- 不做修复进度视图。
- 不保证外部交易所拉取成功。

验证：

- `go test ./internal/web/api`
- `go test ./internal/store/postgres -run 'TestIntegration(ListDataSyncTaskGapsReportsWindows|RepairDataSyncTaskGapsCreatesSyncTasks|ListDataSyncTasksReportsDataHealth)' -count=1`
- `pnpm --dir web/frontend exec vitest run src/services/api/data.test.ts src/components/tables/DataSyncTaskTable.test.ts src/composables/useResearchWorkspace.test.ts`
- `pnpm --dir web/frontend run typecheck`
- `go test ./...`
- `go vet ./...`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `scripts/quality-gate.sh`
- `docker compose build api`
- `docker compose up -d --no-deps api`
- `docker inspect` 显示 `tictick-hi-api-1` healthy。
- `curl -I http://127.0.0.1:8080/research` 返回 `HTTP/1.1 200 OK`。
- 登录后读取 `GET /api/system/api-contract`，包含 `listDataSyncTaskGaps`、`DataSyncGapList` 和 `/api/data/tasks/{id}/gaps`。
- `SMOKE_SAMPLES=30 SMOKE_INTERVAL_MS=150 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：桌面 `doc 1238->1238, panel 680->680, body 603->603, chart 603->603, tv 603->603`；移动 `doc 1256->1256, panel 624->624, body 457->457, chart 457->457, tv 457->457`。

剩余风险：

- 当前缺口详情仍只覆盖任务窗口内已入库 K 线之间的相邻缺口；不代表生产级全历史数据完整性证明。

### 阶段 1 数据同步任务缺口修复排队补充

执行时间：2026-06-28

目标等级：demo

范围内：

- 新增 `POST /api/data/tasks/{id}/repair-gaps`，由后端根据源任务窗口内 `market_candles` 相邻 K 线缺口创建补同步任务。
- 修复仍走 `data_sync_tasks` 和现有 data sync worker：补任务使用源任务 `exchange/symbol/interval`，`start_time/end_time` 来自缺口 `from/to`，创建时 `sync_enabled=true`、`status=pending`。
- PostgreSQL 实现会在事务内锁定源任务行，最多处理前 20 个相邻缺口，并跳过已存在的同交易所、同交易对、同周期、同 `start_time/end_time` 同步任务，避免重复点击造成重复排队。
- API 返回 `DataSyncGapRepairResult`，包含源任务 id、创建的补同步任务、跳过的已存在数量和是否受批次上限限制。
- 研究页任务表在 `gapSummary.count > 0` 时显示“修复缺口”操作，点击后调用后端 repair API，并刷新任务列表。
- API contract、生成的前端 TypeScript DTO、前端 API wrapper 和研究页测试同步更新。

范围外：

- 不扫描任务窗口外的全历史缺口。
- 不推断起点前缺失或尾部尚未同步的 K 线。
- 不保证外部交易所拉取一定成功。
- 不做修复进度视图。
- 不做交易所精确权重限流。

验证：

- `go test ./internal/web/api -run 'TestDataSyncTaskRoutes|TestAPIMethodNotAllowedContracts|TestFrontendAPI|TestAPIContract' -count=1`
- `go test ./internal/store/postgres -run 'TestIntegrationRepairDataSyncTaskGapsCreatesSyncTasks|TestIntegrationListDataSyncTasksReportsDataHealth' -count=1`
- `docker run --rm --network tictick-hi_default -v "$PWD":/src -w /src ... golang:1.26-bookworm go test ./internal/store/postgres -run 'TestIntegrationRepairDataSyncTaskGapsCreatesSyncTasks|TestIntegrationListDataSyncTasksReportsDataHealth' -count=1 -v`
- `cd web/frontend && pnpm run test -- DataSyncTaskTable useResearchWorkspace data`
- `cd web/frontend && pnpm run typecheck`
- `go test ./...`
- `go vet ./...`
- `cd web/frontend && pnpm run test`
- `cd web/frontend && pnpm run build`
- `scripts/quality-gate.sh`

剩余风险：

- 当前 repair API 只排队前 20 个任务窗口内相邻缺口；真实修复结果仍依赖 data sync worker、交易所可用性、退避重试和后续 `dataHealth/gapSummary` 观察。

### 阶段 1 数据同步缺口总量可观察补充

执行时间：2026-06-28

目标等级：demo

范围内：

- `GET /api/data/tasks/{id}/gaps` 返回 `totalCount`、`returnedCount` 和 `repairLimit`，避免用户把受限结果误判为全部缺口。
- `POST /api/data/tasks/{id}/repair-gaps` 返回 `totalCount` 和 `repairLimit`，让修复排队结果明确表达本次只处理了批次上限内窗口。
- PostgreSQL 缺口窗口查询改为在同一 SQL 中统计全量任务窗口缺口，再按修复上限返回可处理窗口。
- 研究页缺口详情弹窗在 `limited=true` 时显示已返回数量、总数和单次修复上限。
- API contract、生成的前端 TypeScript DTO、前端 API wrapper、composable 测试和页面布局契约同步更新。

范围外：

- 不做无限批量修复。
- 不新增后台全历史扫描任务。
- 不改变现有 repair API 单次最多 20 个窗口的写入范围。
- 不证明真实交易所网络下长期恢复能力。

验证：

- `go test ./internal/web/api -run 'TestDataSyncTaskRoutes|TestAPIMethodNotAllowedContracts|TestFrontendAPI|TestAPIContract' -count=1`
- `go test ./internal/store/postgres -run 'TestIntegrationListDataSyncTaskGapsReportsWindows|TestIntegrationListDataSyncTaskGapsReportsLimitedTotal|TestIntegrationRepairDataSyncTaskGapsCreatesSyncTasks' -count=1`
- `pnpm --dir web/frontend exec vitest run src/services/api/data.test.ts src/composables/useResearchWorkspace.test.ts src/pages/ResearchPage.layout.test.ts`
- `pnpm --dir web/frontend run typecheck`
- `go test ./...`
- `go vet ./...`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `scripts/quality-gate.sh`
- `git diff --check`
- `docker compose build api`
- `docker compose up -d --no-deps api`
- `docker inspect --format '{{.State.Health.Status}}' tictick-hi-api-1` 返回 `healthy`。
- `curl -fsSI http://127.0.0.1:8080/research` 返回 `HTTP/1.1 200 OK`。
- `SMOKE_SAMPLES=30 SMOKE_INTERVAL_MS=150 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：桌面 `doc 1238->1238, panel 680->680, body 603->603, chart 603->603, tv 603->603`；移动 `doc 1284->1284, panel 652->652, body 457->457, chart 457->457, tv 457->457`。

剩余风险：

- 当前总量仍限定在源任务窗口内已入库 K 线之间的相邻缺口；不覆盖任务起点前缺失、尾部尚未同步和跨任务全历史数据完整性证明。

### 阶段 1 数据同步任务窗口边界缺口补强

执行时间：2026-06-29

目标等级：scaffold

触发问题：

- `CandleProvider` 已能识别显式 `from/to` 请求窗口的头部、尾部和整窗无数据缺口，但数据同步任务列表、缺口详情和批量修复仍只检测已入库 K 线之间的相邻断点。
- 这会让任务 `start_time/end_time` 边界缺数据时，研究页仍可能显示误导性的健康状态，`repair-gaps` 也不会为头尾窗口排补同步任务。

修复范围：

- PostgreSQL 数据同步任务缺口 SQL 收敛为共享 CTE，统一服务 `ListDataSyncTasks`、`GET /api/data/tasks/{id}/gaps` 和 `POST /api/data/tasks/{id}/repair-gaps`。
- 缺口定义从“已入库相邻 K 线断点”扩展为任务请求窗口内的 `start_time -> first_open`、中间相邻断点、`last_open -> end_time` 和整窗无数据。
- 缺口计算使用任务 interval 的 UTC 网格和 `date_bin`，不伪造 K 线，不改 `market_candles` 事实表 schema。
- 补同步任务 repair 路径复用同一缺口定义，因此头部、尾部和整窗无数据窗口都能排队为 bounded sync task。
- 既有只验证中间断点的 PostgreSQL 集成测试改为显式写入任务 `start_time`，避免测试夹具自身制造头部缺口。

范围外：

- 不做自动后台批量修复。
- 不做完整 cursor pagination。
- 不做真实交易所长时间压测。
- 不改变外部交易所拉取、退避、限流或同步游标推进策略。

验证：

- `go test ./internal/store/postgres -run 'TestIntegration(DataSyncTaskGapsReportRequestedWindowBoundaries|DataSyncTaskGapsReportWholeRequestedWindow|ListDataSyncTasksReportsDataHealth|ListDataSyncTaskGapsReportsWindows|RepairDataSyncTaskGapsCreatesSyncTasks)' -count=1 -v`：本机未设置 `TICTICK_TEST_DATABASE_URL`，集成用例按设计 skip。
- `docker run --rm --network tictick-hi_default -v "$PWD":/src -w /src -e TICTICK_TEST_DATABASE_URL='postgresql://tictick:tictick-local-postgres-password@postgres:5432/tictick_hi?sslmode=disable' golang:1.26-bookworm go test ./internal/store/postgres -run 'TestIntegration(DataSyncTaskGapsReportRequestedWindowBoundaries|DataSyncTaskGapsReportWholeRequestedWindow|ListDataSyncTasksReportsDataHealth|ListDataSyncTaskGapsReportsWindows|RepairDataSyncTaskGapsCreatesSyncTasks)' -count=1 -v` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：21 个测试文件、100 个测试通过。
- `pnpm --dir web/frontend run build` 通过。
- `git diff --check` 通过。

失败：

- 第一次 Docker PostgreSQL 定向测试失败 2 项，原因是旧测试 helper 固定写入 `start_time=2026-06-27T03:00:00Z`，但用例 K 线从后续时间开始；新窗口边界语义正确识别了测试夹具制造的头部缺口。已将相关旧用例改为显式任务窗口后重跑通过。

剩余风险：

- 当前 repair API 仍按单次最多 20 个窗口排队；任务窗口外的全历史数据完整性仍需要 `/api/market/candle-gaps` 或后续后台扫描能力。
- 真实补数结果仍依赖 data sync worker、交易所可用性、退避重试和后续 `dataHealth/gapSummary` 观察。
- 本轮只增强 PostgreSQL 缺口语义，没有提升整体项目等级；整体仍为 `scaffold`。

### 阶段 1 数据同步任务窗口可观察补充

执行时间：2026-06-28

目标等级：demo

范围内：

- 研究页数据同步任务表新增“同步窗口”列，展示 `startTime/endTime`。
- `repair-gaps` 创建的补同步任务可在任务列表中通过 bounded 窗口识别，不再只依赖一次性 toast。
- 持续同步、仅开始时间、仅结束时间和闭合窗口都有明确文案。
- 表格窗口单元格使用 tooltip 和单行省略，避免长 ISO 时间撑破列表布局。
- 中英文 i18n 和组件测试同步更新。

范围外：

- 不新增数据库字段或 migration。
- 不伪造 repair parent / child 关系。
- 不改变缺口修复排队、worker claim 或同步游标语义。
- 不新增全历史扫描或修复进度 worker。

验证：

- `pnpm --dir web/frontend exec vitest run src/components/tables/DataSyncTaskTable.test.ts`
- `pnpm --dir web/frontend run typecheck`
- `go test ./internal/store/postgres -run 'TestIntegrationRepairDataSyncTaskGapsCreatesSyncTasks|TestIntegrationListDataSyncTasksReportsDataHealth' -count=1`
- `go test ./...`
- `go vet ./...`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `scripts/quality-gate.sh`
- `git diff --check`
- `docker compose build api`
- `docker compose up -d --no-deps api`
- `docker inspect --format '{{.State.Health.Status}}' tictick-hi-api-1` 返回 `healthy`。
- `curl -fsSI http://127.0.0.1:8080/research` 返回 `HTTP/1.1 200 OK`。
- `SMOKE_SAMPLES=20 SMOKE_INTERVAL_MS=150 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：桌面 `doc 1238->1238, panel 680->680, body 603->603, chart 603->603, tv 603->603`；移动 `doc 1284->1284, panel 652->652, body 457->457, chart 457->457, tv 457->457`。

剩余风险：

- 当前列表只能展示修复任务自身窗口，不能表达源任务父子关系；真实修复成功与否仍需要观察任务状态、`dataHealth` 和后续缺口摘要。

### 阶段 1 数据同步补任务来源可追踪补充

执行时间：2026-06-28

目标等级：demo

范围内：

- `data_sync_tasks` 新增 nullable `repair_source_task_id`，用于记录由 `repair-gaps` 创建的补同步任务来源。
- migration 增加源任务 FK、非自引用 CHECK 和索引；源任务删除时子任务来源清空，避免 orphan。
- `POST /api/data/tasks/{id}/repair-gaps` 创建补任务时写入源任务 ID；`GET /api/data/tasks` 和 repair result 返回 `repairSourceTaskId`。
- `web/frontend/src/types/api.generated.ts` 由后端 OpenAPI contract 重新生成。
- 研究页任务表同步窗口文案显示修复来源，用户不再只能靠窗口猜测补任务来源。
- `scripts/stage8-migration-audit.sh` 增加 repair source 约束和 orphan 检查。

范围外：

- 不做完整任务父子树。
- 不给手工创建的同窗口任务补写来源。
- 不改变 repair 去重逻辑、worker claim、同步游标推进或交易所拉取逻辑。
- 不新增全历史扫描或修复进度 worker。

验证：

- `TICTICK_WRITE_GENERATED_API_TYPES=1 go test ./internal/web/api -run TestWriteGeneratedFrontendAPITypes -count=1`
- `go test ./internal/store/postgres -run 'TestIntegrationDataSyncRepairSourceConstraints|TestIntegrationRepairDataSyncTaskGapsCreatesSyncTasks' -count=1`
- `go test ./internal/web/api -run 'TestDataSyncTaskRoutes|TestAPIContract|TestFrontendAPI|TestFrontendAPIGeneratedTypesAreCurrent' -count=1`
- `pnpm --dir web/frontend exec vitest run src/components/tables/DataSyncTaskTable.test.ts src/services/api/data.test.ts`
- `pnpm --dir web/frontend run typecheck`
- `go test ./...`
- `go vet ./...`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `scripts/quality-gate.sh`
- `git diff --check`
- `docker compose build api`
- `docker compose run --rm migrate`
- `docker compose up -d --no-deps api`
- `docker inspect --format '{{.State.Health.Status}}' tictick-hi-api-1` 返回 `healthy`。
- `scripts/stage8-migration-audit.sh`
- `curl -fsSI http://127.0.0.1:8080/research` 返回 `HTTP/1.1 200 OK`。
- `SMOKE_SAMPLES=20 SMOKE_INTERVAL_MS=150 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：桌面 `doc 1238->1238, panel 680->680, body 603->603, chart 603->603, tv 603->603`；移动 `doc 1284->1284, panel 652->652, body 457->457, chart 457->457, tv 457->457`。

剩余风险：

- `repairSourceTaskId` 覆盖由 `repair-gaps` 和当前图表单缺口 repair API 新建的补同步任务；历史同窗口任务和旧版本图表“修复首个缺口”创建的任务仍没有源任务关系。

### 阶段 1 研究页缺口修复入口补充

执行时间：2026-06-28

目标等级：demo

范围内：

- 研究页当前图表 `CandleResult.gaps` 非空时显示“修复首个缺口”操作。
- 点击后复用现有 `POST /api/data/tasks` 和 `POST /api/data/tasks/{id}/sync/start`，用当前 `exchange/symbol`、`baseInterval || interval`、首个 gap 的 `from/to` 创建并启动补同步任务。
- 聚合周期缺口使用 `baseInterval` 修复，例如 `5m` 图表由 `1m` 聚合时，补任务周期为 `1m`。
- 成功后刷新同步任务列表，用户能在研究页继续观察任务状态。
- `useResearchWorkspace` 测试覆盖有 gap 的请求顺序和无 gap 的禁止创建。

范围外：

- 不批量修复所有缺口。
- 不新增后端 repair endpoint。
- 不保证外部交易所同步必然成功。
- 不做全历史缺口扫描和修复进度视图。

验证：

- `cd web/frontend && pnpm run test -- useResearchWorkspace ResearchPage.layout`
- `cd web/frontend && pnpm run typecheck`
- `go test ./...`
- `go vet ./...`
- `cd web/frontend && pnpm run test`
- `cd web/frontend && pnpm run build`
- `scripts/quality-gate.sh`

剩余风险：

- 当前仅排队首个缺口的补同步任务；真实修复结果仍依赖 data sync worker、交易所可用性和后续任务健康观察。

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
- 登录会话基础审计、持久化限流、基础来源上下文和当前操作员自禁用保护已在后续补充关闭到 demo 边界；密码策略、RBAC / 更完整自保护规则和生产级设备上下文仍未完成。

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
- 全系统 worker lease、登录会话密码策略 / RBAC / 更完整自保护规则 / 更完整生产级设备上下文仍未完成。

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
- 当时登录失败节流为 API 进程内存态，多实例、重启后持久化和全局限流未实现；后续“阶段 7 登录失败持久化限流补充”已关闭该项到 demo 边界。
- 当时操作台账号启停没有 RBAC、自保护规则或强密码策略；基础操作审计和当前操作员自禁用保护已在后续阶段 7 补充中覆盖到 `demo` 边界。
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

- RBAC、更完整自保护规则、生产级会话审计、密码策略、设备指纹、IP / UA 变更告警。

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

- 这只是基础 session 管理，不是生产级登录安全；持久化限流、基础来源上下文和当前操作员自禁用保护已在后续补充关闭到 demo 边界，仍缺密码策略、RBAC / 更完整自保护、设备指纹 / 变更告警和生产级审计。
- 本轮未运行完整 `scripts/stage8-smoke.sh`；session 路由用本地 HTTP / DOM smoke 覆盖。

### 阶段 7 登录失败持久化限流补充

执行时间：2026-07-07

目标等级：demo 增量。

范围内：

- 新增 `operator_login_rate_limits` PostgreSQL 表，按 `key_hash` 保存失败次数、首次失败时间、锁定到期时间和更新时间。
- 登录限流 key 继续使用归一化 username + 远端 host，但进入仓储前转换为 SHA-256 hex；数据库不保存用户名或 IP 原文。
- `POST /api/auth/login` 在认证前通过 repository 检查持久化限流；认证失败记录失败次数，认证成功清理失败状态，达到阈值后返回 `429`。
- `data.Repository`、PostgreSQL store 和 API 测试 fake 均提供登录限流检查、失败记录和清理方法。
- 限流触发的登录请求写入匿名 `auth.login` 失败审计，reason 为 `rate_limited`。

范围外：

- 密码策略、MFA、RBAC、更完整自保护规则、设备指纹、IP / UA 变更告警、边缘层全局限流和生产级审计留存。

验证：

- `go test ./internal/web/api ./internal/store/postgres -count=1`
- `go test ./...`
- `go vet ./...`
- `scripts/quality-gate.sh`
- `scripts/check-file-size.sh`
- `git diff --check`

未执行：

- `scripts/stage8-migration-audit.sh`：本地 Docker daemon 未运行，返回无法连接 `/Users/xiaobai/.docker/run/docker.sock`；新增 PostgreSQL 集成测试在无 `TICTICK_TEST_DATABASE_URL` 时按既有约定跳过，需在目标测试库或 Compose 环境补跑。

剩余风险：

- 当前补充只把 API 层登录失败节流从进程内存态推进到 PostgreSQL demo 边界；生产还需要更完整的账号安全策略、设备上下文和边缘/网关层防护。

### 阶段 7 操作员自禁用保护补充

执行时间：2026-07-07

目标等级：demo 增量。

范围内：

- `POST /api/system/operators/:id/disable` 在目标 id 等于当前登录操作员 id 时返回 `409 invalid_state`，不调用 repository、不改变账号状态。
- 系统操作员页面读取当前 auth store，当前操作员的停用按钮置为 disabled，并提供“不能停用当前操作员”提示。
- API 单测覆盖自禁用请求不会禁用当前账号，也不会写入成功的 `operator.disable` 审计事件。
- Vue 单测覆盖当前账号停用按钮禁用，其他操作员仍可触发停用 API。

范围外：

- RBAC、角色权限、最后一个启用管理员保护、多人审批、强密码策略、MFA、不可篡改审计。

验证：

- `go test ./internal/web/api -run TestOperatorCannotDisableSelf -count=1`
- `pnpm --dir web/frontend exec vitest run src/pages/SystemOperatorsPage.test.ts`
- `go test ./internal/web/api -count=1`
- `pnpm --dir web/frontend exec vitest run src/pages/SystemOperatorsPage.test.ts src/services/api/system.test.ts`
- `go test ./...`
- `go vet ./...`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `scripts/quality-gate.sh`
- `scripts/check-file-size.sh`
- `git diff --check`

剩余风险：

- 本补充只防止当前操作员误禁用自己；仍未实现角色模型、最后管理员保护和完整账号生命周期策略。

### 阶段 7 登录会话来源上下文补充

执行时间：2026-07-07

目标等级：demo 增量。

范围内：

- `operator_sessions` 新增可空 `remote_addr` 和 `user_agent` 字段，旧 session 兼容为空。
- 登录创建 session 时写入 `clientAddress(r)` 和 User-Agent；`X-Forwarded-For` 仍按审计同一规则取首个地址，User-Agent 做 trim 和 255 rune 裁剪。
- `GET /api/auth/sessions` 返回 `remoteAddr` / `userAgent`，仍不返回 `tokenHash`。
- 系统登录会话页展示来源地址和 User-Agent，长 User-Agent 单行省略并保留 title。

范围外：

- 设备指纹、可信设备、IP / UA 变更告警、地理位置识别、session 风险评分、强制下线策略。

验证：

- `go test ./internal/web/api -run 'Test(LoginStoresSessionClientContext|AuthSessionManagementRoutes|FrontendAPI|APIContract|WriteGeneratedFrontendAPITypes)' -count=1`
- `go test ./internal/store/postgres -run TestOperatorSessionStorePersistsClientContext -count=1 -v`（当前本地无 `TICTICK_TEST_DATABASE_URL`，按既有规则跳过）
- `pnpm --dir web/frontend exec vitest run src/services/api/auth.test.ts`
- `scripts/generate-api-types.sh`
- `pnpm --dir web/frontend exec vitest run src/pages/SystemSessionsPage.test.ts src/services/api/auth.test.ts`
- `go test ./internal/web/api -count=1`
- `go test ./...`
- `go vet ./...`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `scripts/check-file-size.sh`
- `scripts/quality-gate.sh`
- `git diff --check`

未执行：

- `scripts/stage8-migration-audit.sh`：本地 Docker daemon 未运行，返回无法连接 `/Users/xiaobai/.docker/run/docker.sock`；新增 migration 需在 Compose 或目标测试库环境补跑。

剩余风险：

- 本补充只提供基础可见上下文，不做设备可信度判断或异常变更告警。

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
| Go 子命令 | scaffold | `hi api/sync/backtest/trading/notify/migrate` 可由 compose 和 smoke 调用；API / sync / backtest / trading / notify 的 env 配置严格解析和脱敏启动摘要已有 `cmd/hi` 单测覆盖；基础子命令运行手册和命令配置 smoke 已覆盖配置错误脱敏边界 | 仍缺生产部署运行手册、结构化日志/trace、子命令级健康探针和更完整优雅停止证据 |
| Docker Compose | demo | `scripts/stage8-smoke.sh` 从 compose build/up 进入并完成全链路 smoke；`scripts/stage8-sigterm-smoke.sh` 从 compose stop 进入并验证 data sync / backtest / trading / notify 收尾 | 缺备份/恢复、资源限制、外部依赖失败策略和共享环境部署说明 |
| PostgreSQL migrations | scaffold | 当前 smoke 可从 migrations 建库并运行；`0011_domain_constraints.sql` 已补充核心状态、类型、数值和时间范围 CHECK，`0012_referential_constraints.sql` 已补充 orders / executions / positions / notifications / outbox / backtest_orders 的核心 FK 和同 task composite FK，`0016_worker_lease_constraints.sql` 已补充 task/outbox lease 字段一致性 CHECK，`0017_strategy_intent_parent_constraints.sql` 已补充 `strategy_intents` 新增/更新父任务归属约束，`0018_strategy_intent_parent_delete_guards.sql` 已补充父任务删除防 orphan 保护，`0019_task_terminal_timestamp_constraints.sql` 已补充任务终态 `finished_at` 一致性约束，`0020_validate_worker_lease_constraints.sql` 已修补历史半截 lease 并 VALIDATE worker lease CHECK，`0021_task_status_transition_guards.sql` 已补充 data sync / backtest / trading 核心状态流转 trigger，`0024_data_sync_repair_source.sql` 已补充 data sync repair source FK / 非自引用 CHECK；`scripts/stage8-migration-audit.sh` 已校验迁移全量应用、worker lease CHECK validated、状态流转 trigger、repair source、终态 finished_at、lease、intent parent 和核心事实 orphan | 完整统一状态机、全量历史数据验证、数据迁移/回滚策略不足 |
| API server | scaffold | 核心路由已拆分，CSRF 写保护、策略参数校验、retry API、结构化错误响应和基础操作审计可测；前端 API client 会读取服务端 `message/error` 并保留 `code`；数据同步 retry / command 状态冲突已有领域错误码；`DataSyncTask` 暴露 `repairSourceTaskId` 用于补同步任务来源追踪；已知 API 路径的方法错误返回 405 和 `Allow` header；`GET /api/system/api-contract` 返回基础 OpenAPI 3.1 contract，覆盖当前前端路由、request body、success schema、错误 schema、错误码 catalog、session cookie 和 CSRF header；`web/frontend/src/types/api.generated.ts` 已从该 contract 生成；`TestFrontendAPI*` 和 `scripts/check-api-contract-drift.sh` 会阻止前端 service route、request DTO、核心 response DTO、adapter response 字段、generated DTO staleness、external OpenAPI validator 和 candle query 参数漂移 | 跨领域错误语义细分和生产级审计边界不足 |
| 登录会话 | demo | HttpOnly session、CSRF double-submit、登录失败持久化节流、session 列表 / 来源上下文和撤销有 route / smoke 覆盖；登录成功 / 失败、退出、session 撤销已进入基础操作审计；当前操作员自禁用保护已覆盖 | 无密码策略/RBAC、更完整自保护规则和生产级设备上下文 |
| 数据同步 worker | demo | claim/heartbeat/upsert/retry/release、批次内连续 open_time 游标推进、临时错误任务级 `next_attempt_at` 持久化退避、交易所级 `data_sync_exchange_backoffs` 冷却、失败后 UI retry、公开 market 请求本地固定窗口限流、全历史相邻缺口扫描和单缺口补同步排队、Stage 8 smoke 和容器 SIGTERM smoke 有覆盖 | 未证明真实交易所网络下长期恢复、分布式多实例限流、完整状态机和批量自动修复 |
| CandleProvider | demo | native/aggregated/gap/coverage/pagination/window metadata、opaque adjacent-window cursor、最多 1440000 根基础 `1m` 的有界流式聚合读取、240000 根 PostgreSQL 性能 smoke、6 并发聚合读取 smoke、runner 健康门禁和集成测试已覆盖 | 长期 soak / 冷缓存 / 真实生产数据分布压测、超过 1440000 根基础 K 线的缓存/分段策略、异常数据修复策略不足 |
| Binance / OKX adapter | demo | 临时错误分类、Binance fallback、OKX rate-limit 码、URL 脱敏、交易所级冷却和本地固定窗口限流有测试 | 缺动态读取交易所限流元数据、真实网络压测、代理/地域策略和完整业务码审计 |
| 研究页 | scaffold | 数据源 metadata、当前窗口范围、全历史缺口摘要与单缺口修复入口、列表在上图表在下、图表高度稳定且轴线 canvas 完整落在固定槽内、opaque cursor 上一/下一窗口、失败任务 retry、缺口详情、补同步窗口和 repair source 展示已覆盖 | 生产级 instrument 搜索、图表工具薄、缺指标和完整批量缺口修复工作流 |
| 策略 registry / runtime | demo | schema 驱动参数、intent 输出和策略边界门禁已覆盖 | 缺策略沙箱、版本迁移、权限隔离和真实策略库 |
| 回测 | demo | CandleProvider、closed/minute replay、intent/order/result、买卖点展示和容器 SIGTERM release 已走通 | 撮合、费用/滑点曲线、指标体系和结果可信度不足 |
| 交易 runner | demo | paper execute/notification、position/order/execution/outbox、claim 公平性和容器 SIGTERM release 已走通；通知 intent 可进入 email / Telegram / 飞书 provider 基础发送路径 | 风控、PnL 可信度、通知 provider 生产启用边界、统一状态机和实盘隔离不足 |
| 实盘安全 | demo | 凭据 AES-GCM、本地 live 任务创建护栏和 live execute 禁用已验证 | testnet/sandbox executor、订单先落库再提交、幂等 retry、KMS/轮换未完成 |
| 通知 | demo | outbox、local/webhook-demo/webhook/email/Telegram/飞书 provider、失败重试、系统页 retry 和 notify 容器 SIGTERM release 已覆盖 | 真实第三方账号联网验收、模板、限流、审计和通道管理不足 |
| 前端基础设施 | scaffold | Vue/Naive/Pinia/i18n/主题/API wrapper/图表封装已存在并通过测试；路由级 code split 已让生产入口 chunk 降至 437.44 kB，构建不再出现 Vite 大 chunk 警告；概览页已接入真实 API 聚合；Stage 8 visual smoke 已覆盖核心页面 1440/812/390 视口、浅/深主题和 zh-CN/en-US 语言矩阵 | 缺像素快照基线、真正全路由覆盖和 CI 硬门禁 |
| 概览页 | demo | 从现有 API 聚合系统健康、data sync、backtest、trading 和 notification，展示关键计数、异常提醒、worker 状态和最近活动；`useOverviewWorkspace` 单测覆盖聚合契约 | 缺时间窗口筛选、趋势图、关键操作入口和生产级监控语义 |
| 系统管理 / 运维健康 | demo | 操作台账号启停、当前操作员自禁用保护、当前操作员 session 来源上下文 / 撤销、基础操作审计页、健康页 worker 统计和通知/账号管理可用 | 无 RBAC、更完整自保护、不可篡改审计和生产监控 |
| 质量门禁 | demo | 通用门禁、API contract route / field drift / generated TypeScript DTO staleness / external OpenAPI validator gate、Go command config smoke、stage8 smoke、data sync/backtest/trading/notify SIGTERM smoke、scaffold 声明检查可重复运行 | 尚未把真实网络压测、像素视觉回归和安全审计纳入硬门禁 |

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

### 阶段 8 前端视觉 smoke gate 补充

执行时间：2026-06-29

目标等级：scaffold

触发问题：

- Stage 8 readiness 重审计将前端基础设施列为 `scaffold`，其中一个明确 blocker 是缺系统性桌面 / 移动 / 主题视觉回归。
- 既有 `research-chart-height-smoke.mjs` 只覆盖研究页图表高度，不覆盖概览、列表页、系统健康页、浅色 / 深色主题和整体横向溢出。

修复范围：

- 新增 `scripts/stage8-visual-smoke.mjs`，复用本地 8080、登录态和 Chrome DevTools Protocol，不引入新浏览器依赖。
- smoke 覆盖 `/overview`、`/research`、`/backtests`、`/trading`、`/system/health` 五个核心页面。
- 每个页面在 desktop `1440x900`、mobile `390x844` 和 light / dark 主题下验证 app shell、page title、主内容节点存在且有非零尺寸。
- 每个页面验证无 JS runtime error / console error，`documentWidth` 不超过 viewport 容差，主页面容器不逃出 viewport。
- 研究页额外验证 `.research-chart-body`、`.trading-chart` 和 `.tv-lightweight-charts` 存在且高度不超过 viewport，避免把通用视觉 smoke 与图表专项 smoke 脱节。
- README 增加 Stage 8 visual smoke 本地运行入口。

范围外：

- 不做像素快照 diff 或截图基线。
- 不覆盖所有系统管理子页面、详情页、中文/英文双语言矩阵。
- 不把该脚本加入轻量 `scripts/quality-gate.sh`，因为它依赖本地 8080 和可用 Chrome。
- 不把前端基础设施升级为 `demo` 或 `usable`。

验证：

- `node scripts/stage8-visual-smoke.mjs` 通过：desktop light/dark 与 mobile light/dark 各覆盖 5 个页面，最大 document width 分别为 `1440 / 1440 / 390 / 390`。
- `node scripts/research-chart-height-smoke.mjs` 通过：desktop、narrow desktop、mobile 图表高度稳定，内部高度污染后不增长。

失败：

- 无当前硬失败。

剩余风险：

- 前端基础设施仍为 `scaffold`；当前只是核心页面的 DOM/layout smoke，不是完整像素级视觉回归体系。
- 该 smoke 依赖本机 Chrome 和已启动本地 8080；无 Chrome 的 CI/主机需要设置 `CHROME_PATH` 或跳过本地视觉检查。

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

- API server 仍为 `scaffold`；本轮只补基础 OpenAPI/schema contract；字段级 diff gate、错误 taxonomy、前端 TypeScript 类型生成和外部 OpenAPI validator 已在后续补充中覆盖，仍缺生产级审计边界或 RBAC。

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

- API server 仍为 `scaffold`；本轮只防止前端 API route 与后端 contract 漂移，字段级 contract drift、TypeScript 类型生成和外部 OpenAPI schema validator 已在后续补充中覆盖。

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

- API server 仍为 `scaffold`；本轮字段级 drift gate 依赖当前项目简单 TypeScript type 语法解析，不是通用 TypeScript compiler AST；前端类型自动生成和外部 OpenAPI validator 已在后续补充中覆盖。

### 阶段 8 API TypeScript DTO 生成补充

执行时间：2026-06-28

触发问题：

- API server 仍被 Stage 8 readiness 标为 `scaffold`，其中一个明确 blocker 是前端 TypeScript DTO 仍手写维护。
- 已有字段级 drift gate 可以发现部分漂移，但不能提供从后端 OpenAPI contract 到前端 DTO 的生成路径。

修复范围：

- 新增 `internal/web/api/typescript_generator.go`，从 `apiContractDocument()` 生成前端 TypeScript DTO。
- 新增 `scripts/generate-api-types.sh`，通过 `TICTICK_WRITE_GENERATED_API_TYPES=1 go test ./internal/web/api -run '^TestWriteGeneratedFrontendAPITypes$' -count=1` 写入生成文件。
- 新增 `web/frontend/src/types/api.generated.ts`，作为 contract 生成产物，保持 343 行，低于 TypeScript 文件硬上限。
- `web/frontend/src/types/app.ts` 改为复用生成 DTO，并只保留前端 UI 专用类型和更严格的表单类型。
- `web/frontend/src/services/api/data.ts`、`web/frontend/src/services/api/strategies.ts` 改为使用生成的 API response 类型作为原始边界。
- `api_schema_drift_test.go` 改为读取 `api.generated.ts`，并新增 `TestFrontendAPIAppTypesReferenceGeneratedContract`，防止核心 app 类型绕开生成 contract。
- `TestFrontendAPIGeneratedTypesAreCurrent` 纳入 `scripts/check-api-contract-drift.sh` 和 `scripts/quality-gate.sh`，生成文件 stale 会使门禁失败。

验证：

- `scripts/generate-api-types.sh`
- `go test ./internal/web/api -count=1`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `scripts/check-api-contract-drift.sh`
- `go test ./...`
- `go vet ./...`
- `pnpm --dir web/frontend run build`
- `scripts/quality-gate.sh`
- `git diff --check`

失败：

- 首次 `scripts/quality-gate.sh` 失败：生成文件 414 行超过 TypeScript 400 行硬上限。已压缩 generator 输出空行，重新生成后 `api.generated.ts` 为 343 行并通过门禁。

剩余风险：

- API server 仍为 `scaffold`；本轮关闭的是 in-repo OpenAPI contract 到前端 DTO 的生成和 staleness gate；外部 OpenAPI validator 已在后续补充中覆盖，仍不是完整 SDK/client 生成、运行时 JSON schema validator、领域级错误语义或生产级审计边界。

### 阶段 8 API 外部 OpenAPI validator 补充

执行时间：2026-06-28

触发问题：

- API server 仍被 Stage 8 readiness 标为 `scaffold`，其中一个明确 blocker 是 `/api/system/api-contract` 只有项目内手写断言，没有第三方 OpenAPI 解析/校验证据。
- 前端 DTO 生成和 drift gate 能证明项目内契约一致，但不能证明 HTTP route 实际输出的 JSON 是合法 OpenAPI 3.1 文档。

修复范围：

- 新增 `github.com/getkin/kin-openapi/openapi3` 作为测试侧外部 OpenAPI validator。
- 新增 `TestAPIContractValidatesWithExternalOpenAPIValidator`，通过已登录测试 server 请求真实 `/api/system/api-contract` HTTP 响应，再用 `openapi3.NewLoader().LoadFromData` 解析并 `Validate`。
- validator 禁用 external refs，避免 contract 测试读取外部文件或网络。
- `scripts/check-api-contract-drift.sh` 扩大为同时运行 `TestFrontendAPI*` 和外部 validator 测试，因此 `scripts/quality-gate.sh` 会硬性执行规范校验。

验证：

- `scripts/check-api-contract-drift.sh`
- 本轮通用门禁见最终回复。

失败：

- 无当前硬失败；当前 `/api/system/api-contract` 已能通过 `kin-openapi` OpenAPI 3.1 解析和验证。

剩余风险：

- API server 仍为 `scaffold`；本轮关闭的是 contract 文档的外部规范校验，不是完整 SDK/client 生成、运行时 JSON schema validator、领域级错误语义、RBAC 或生产级审计边界。

### 阶段 8 API error taxonomy contract 补充

执行时间：2026-06-28

触发问题：

- API server 已有结构化错误响应和基础 OpenAPI contract，但错误码仍散落在 `defaultError`、CSRF、auth/store error 写入点中。
- OpenAPI 只声明 `APIErrorResponse`，没有可枚举的错误码 schema，也没有说明每个错误 HTTP status 可能返回哪些 `code`。
- 新增错误码时缺少硬测试，容易绕过现有 `code/message/error` 形状，导致前端和 API 使用者拿到未登记的错误 code。

修复范围：

- 新增 `internal/web/api/error_catalog.go`，集中定义 `apiErrorCode`、错误码 catalog、HTTP status、描述和 retryable 标记。
- `writeAPIError` 改为接收 typed `apiErrorCode`；未知错误码会被降级为 `500 internal_error` 和安全文案 `internal server error`。
- `defaultError`、`writeAuthError`、`writeStoreError`、CSRF 检查全部改为使用 catalog 常量，不再直接写错误码字符串。
- `GET /api/system/api-contract` 的 components 新增 `APIErrorCode` enum schema 和 `x-errorCodes` catalog；`APIErrorResponse.code` 引用 `APIErrorCode`。
- OpenAPI error response 增加 `x-errorCodes`，按 HTTP status 声明该 response 可能返回的错误码。
- 新增 `api_error_taxonomy_test.go`，覆盖 catalog 唯一性/状态码合法性、contract enum、error response `x-errorCodes`、未知错误码兜底和源码 callsite 禁止直接写字符串错误码。

验证：

- `go test ./internal/web/api -count=1`
- `go test ./...`
- `go vet ./...`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `scripts/quality-gate.sh`
- `git diff --check`
- `scripts/stage8-smoke.sh`
- 本轮 Stage 8 smoke 证据：symbol `S81782596411USDT`、data task `dst_8aad8bbd835de1e3215b1741`、backtest `bt_152477ba71b59ae63355f9d7`、paper execute `tt_fc8ce26166d2eecbed5e9bd5`、paper notify `tt_2d76f91b9eed943eb15dc392`、notification channel `stage8-smoke-1782596411`。

失败：

- 无当前硬失败。

剩余风险：

- API server 仍为 `scaffold`；本轮关闭的是基础错误码 catalog 和 contract 暴露；TS 类型生成和外部 OpenAPI validator 已在前一补充中覆盖，仍不是完整领域错误语义、RBAC 或生产级审计。

### 阶段 8 API data sync domain error 补充

执行时间：2026-06-28

触发问题：

- 数据同步 retry 和 start/stop/sync/realtime command 的状态冲突此前都返回泛化 `409 invalid_state`，前端和 API 调用方无法区分“任务必须 failed 才能 retry”和“当前任务状态不允许该 command”。
- PostgreSQL store 虽然在错误文本里带有业务原因，但 API error code 没有稳定领域语义，OpenAPI enum 和前端生成类型也无法表达该差异。

修复范围：

- `internal/data` 新增可 `errors.Is(err, ErrInvalidState)` 的 `DomainError`，并定义 `data_sync_retry_requires_failed`、`data_sync_command_invalid_state` 两个数据同步错误码。
- PostgreSQL `RetryDataSyncTask` 和 data sync command 状态冲突返回领域错误，同时保留 `ErrInvalidState` unwrap 兼容现有状态机测试。
- API error catalog 新增两个 409 领域错误码，`writeStoreError` 优先映射领域错误，再回退泛化 `invalid_state`。
- fake repository 与 API route 测试同步模拟真实 store 的状态冲突语义。
- 重新生成 `web/frontend/src/types/api.generated.ts`，让 `APIErrorCode` union 包含新增领域错误码。

验证：

- `scripts/generate-api-types.sh`
- `go test ./internal/data ./internal/web/api ./internal/store/postgres`
- `scripts/check-api-contract-drift.sh`

失败：

- 首次窄范围测试失败于 `TestFrontendAPIGeneratedTypesAreCurrent`，原因是 APIErrorCode enum 新增后前端生成 DTO 过期；已运行 `scripts/generate-api-types.sh` 并重跑通过。

剩余风险：

- API server 仍为 `scaffold`；本轮只细分了数据同步状态冲突错误。交易任务、通知、auth/session、系统管理等其他领域还需要继续建立稳定领域错误码和生产级审计边界。

### 阶段 1 K 线图表固定槽高度读取规则补充

执行时间：2026-06-28

目标等级：demo

触发问题：

- 用户继续反馈前端 K 线图表界面会无限拉高，直到页面崩掉。
- 当前真实 8080 长采样未复现持续增长，但代码复查发现固定图表槽在同时存在 `height` 和较宽松 `max-height` 时优先读取 `max-height`，会让详情页这类 `chart-panel[data-chart-viewport="fixed"]` 的渲染高度偏离真实容器高度。
- 现有研究页高度 smoke 只污染 `.tv-lightweight-charts` 根、内部 table 和 canvas，未覆盖 table 布局中间层 `tbody/tr/td`。

修复范围：

- `TradingViewChart` 固定槽高度读取规则调整为优先使用有效 `height`；只有 `height` 缺失、无效或超过有效 `max-height` 时，才用 `max-height` 兜底。
- `TradingViewChart.test.ts` 新增固定 `chart-panel` 场景，验证 `height: 720px` 不会被误读成 `max-height: 820px`。
- `scripts/research-chart-height-smoke.mjs` 的高度污染对象扩展到 `.tv-lightweight-charts tbody/tr/td`，覆盖 table 布局中间层被写入超大高度的情况。

验证：

- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts src/pages/ResearchPage.layout.test.ts` 通过：2 个测试文件、19 个测试通过。
- `node --check scripts/research-chart-height-smoke.mjs` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：20 个测试文件、88 个测试通过。
- `pnpm --dir web/frontend run build` 通过，生产入口为 `/assets/index-DW1G5fki.js`。
- `docker compose build api` 通过。
- `docker compose up -d --no-deps api` 后 `docker inspect -f '{{.State.Health.Status}}' tictick-hi-api-1` 返回 `healthy`。
- `curl http://127.0.0.1:8080/research` 确认真实 8080 已服务新入口 `/assets/index-DW1G5fki.js`。
- `SMOKE_SAMPLES=120 SMOKE_INTERVAL_MS=100 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：桌面 `doc 1238->1238, panel 680->680, body 603->603, chart 603->603, tv 603->603`；移动 `doc 1284->1284, panel 652->652, body 457->457, chart 457->457, tv 457->457`。

失败：

- 当前真实 8080 仍未复现用户可视 Chrome 会话中的原始无限增长堆栈。

剩余风险：

- 本轮关闭的是固定槽高度读取顺序和 table 中间层高度污染 smoke 覆盖缺口，不等于完整桌面 / 移动 / 主题视觉回归体系。
- 研究页图表研究能力仍薄，项目整体仍为 `scaffold`。

### 阶段 1 instrument catalog 定时同步补充

执行时间：2026-06-28

目标等级：demo

触发问题：

- 阶段 1 研究核心已支持手动 instrument 同步、catalog 搜索和数据同步任务 active catalog 强校验，但长期运行时 catalog 不会自动刷新。
- 这会让交易所新增、下架或状态变化的交易对只能靠人工点击刷新进入 PostgreSQL，无法支撑研究页创建同步任务时的 active catalog 校验。

修复范围：

- 新增 `internal/marketsync.Runner`，按 exchange 拉取公开 instrument 元数据并调用 `ReplaceMarketInstruments` 写入 `market_instruments`。
- `hi sync` 长运行模式复用 Binance / OKX market client，并行启动 instrument catalog 后台同步；默认启动时同步一次，之后按 `MARKET_INSTRUMENT_SYNC_INTERVAL` 定时同步。
- `hi sync --once` 保持原有一次性 K 线任务 claim 语义，不触发 instrument 网络同步，避免破坏 smoke / 运维一次性任务。
- 单个交易所 instrument 同步失败只记录 warning 并继续其它交易所，不终止 K 线 data sync runner。
- `docker-compose.yml` 和 `.env.example` 增加 `MARKET_INSTRUMENT_SYNC_ENABLED`、`MARKET_INSTRUMENT_SYNC_ON_START`、`MARKET_INSTRUMENT_SYNC_INTERVAL` 配置。

验证：

- `go test ./internal/marketsync ./cmd/hi -count=1` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：20 个测试文件、88 个测试通过。
- `pnpm --dir web/frontend run build` 通过，生产入口为 `/assets/index-DW1G5fki.js`。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- `docker compose build sync` 通过。
- `docker compose up -d --no-deps sync` 后 `tictick-hi-sync-1` 运行中。
- `docker compose logs --tail=80 sync` 显示启动后自动执行 instrument 同步：Binance `active=1359 inactive=5`，OKX 在当前网络环境返回 `www.okx.com: EOF` 并记录 warning，sync 容器未退出。
- PostgreSQL 查询 `market_instruments` 显示 Binance `active=1359`，最新 `synced_at=2026-06-28 11:34:26.300367+00`。
- 登录后真实 API `GET /api/market/instruments?exchange=binance&q=SOL&limit=3` 返回 `SOLUSDT` 等结果，`syncedAt=2026-06-28T11:34:26.300367Z`。
- `curl -I http://127.0.0.1:8080/research` 返回 `HTTP/1.1 200 OK`。

失败：

- OKX instrument 同步在当前宿主网络环境仍返回 `www.okx.com: EOF`；本轮实现把该错误限定为单交易所 warning，不影响 Binance catalog 同步或 K 线 data sync worker 继续运行。

剩余风险：

- 本轮只是单实例 `hi sync` 内的周期刷新；没有做跨实例分布式调度锁、catalog 同步失败的系统健康告警，交易所公开 market 请求限流仍只是本地固定窗口。
- 退市/停牌后的任务操作语义仍未完整定义；项目整体仍为 `scaffold`。

### 阶段 1 instrument catalog 临时错误重试补充

执行时间：2026-06-29

目标等级：scaffold 增量

触发问题：

- `hi sync` 的 K 线同步已对交易所临时错误做有限重试，但后台 instrument catalog 同步只请求一次。
- OKX public instruments 在当前本地网络环境出现过 `www.okx.com: EOF`，单次临时失败会直接记录 warning 并等到下一次 `MARKET_INSTRUMENT_SYNC_INTERVAL`，默认间隔较长。

修复范围：

- `internal/marketsync.Runner` 增加 `FetchRetries` / `RetryDelay` 配置。
- instrument catalog fetch 遇到 `exchange.TemporaryError` 时按线性短延迟有限重试；永久错误、context 取消和重试耗尽仍返回错误，不伪装成功。
- `hi sync` 将既有 `SYNC_FETCH_RETRIES` / `SYNC_RETRY_DELAY` 传给 instrument catalog runner，使 K 线拉取和后台 catalog 拉取使用一致的短重试配置。
- `docs/go-command-runbook.md` 明确 `SYNC_FETCH_RETRIES` / `SYNC_RETRY_DELAY` 同时作用于 K 线 fetch 和后台 instrument catalog fetch。

验证：

- `go test ./internal/marketsync ./cmd/hi -count=1`
- `go test ./...`
- `go vet ./...`
- `cd web/frontend && pnpm run typecheck`
- `cd web/frontend && pnpm run test`
- `cd web/frontend && pnpm run build`
- `scripts/quality-gate.sh`
- 本地重建并重启 `sync`，确认容器运行且 `api` readyz 正常。

失败：

- 无硬失败。

剩余风险：

- 本轮只是短重试，不是跨实例 distributed backoff，也没有把 catalog 同步失败持久化为系统健康事件。
- 真实 OKX/Binance 网络恢复压测仍未完成；项目整体仍为 `scaffold`。

### 阶段 1 instrument catalog 同步状态可观察补充

执行时间：2026-06-29

目标等级：scaffold 增量

触发问题：

- 上一轮已让 instrument catalog fetch 对临时错误做短重试，但重试耗尽后的失败只在 `sync` 日志里可见。
- 研究页创建数据同步任务依赖 active `market_instruments` catalog；如果后台 catalog 同步持续失败，用户无法从系统健康中判断 catalog 是否 stale 或失败。

修复范围：

- 新增 `market_instrument_sync_statuses`，按 exchange 持久化 `last_attempt_at`、`last_success_at`、`last_error` 和 `updated_at`。
- `ReplaceMarketInstruments` 成功时同事务写入成功状态并清空 `last_error`。
- `hi sync` 后台 runner 和手动 `/api/market/instruments/sync` 失败时写入最近失败状态；错误文本复用 500 rune 规范化，避免长错误撑爆健康页。
- `/api/system/health` 新增 `market-instrument-catalog` service；存在最近失败、从未同步或无状态行时返回 warning，并让总体健康降级为 `degraded`。
- 运维健康页复用现有 service 列表展示 catalog service 名称、状态和 detail，不新增独立页面。

验证：

- `go test ./internal/marketsync ./internal/web/api ./internal/store/postgres -count=1`
- `go test ./...`
- `go vet ./...`
- `cd web/frontend && pnpm run typecheck`
- `cd web/frontend && pnpm run test`
- `cd web/frontend && pnpm run build`
- `scripts/quality-gate.sh`
- 本地 migrate + 重启 `api` / `sync` 后，`GET /api/system/health` 能看到 `market-instrument-catalog` service。

失败：

- 无硬失败。

剩余风险：

- 本轮只保留每个 exchange 的最新状态，不保存 catalog 同步历史事件。
- 仍未做跨实例 catalog 同步调度锁、分布式退避或真实交易所恢复压测；项目整体仍为 `scaffold`。

### 阶段 1 instrument catalog 状态进入研究页补充

执行时间：2026-06-29

目标等级：scaffold 增量

触发问题：

- instrument catalog 最近同步状态已经落库并进入运维健康页，但研究页创建同步任务时仍只能看到 symbol 建议和 active/missing 校验结果。
- 当 OKX public instruments 持续 EOF 时，用户在研究页不知道当前交易所目录本身最近同步失败，只能到运维健康页排查。

修复范围：

- 新增 `GET /api/market/instruments/status`，返回各交易所 instrument catalog 的 `lastAttemptAt`、`lastSuccessAt`、`lastError` 和 `updatedAt`。
- OpenAPI contract 和 `web/frontend/src/types/api.generated.ts` 同步新增 `MarketInstrumentSyncStatus`。
- 研究页 `useResearchWorkspace` 会随任务/K 线一起加载 catalog 状态。
- 研究页当前数据源 metadata 和创建同步任务弹窗会展示所选交易所目录最近成功/失败状态；手动刷新交易对后会重新读取状态。
- `MarketSymbolAutoComplete` 刷新按钮新增 `synced` 事件，父页面据此刷新 catalog 状态。

验证：

- `scripts/generate-api-types.sh`
- `go test ./internal/web/api ./internal/store/postgres -count=1`
- `go test ./...`
- `go vet ./...`
- `cd web/frontend && pnpm run typecheck`
- `cd web/frontend && pnpm run test`
- `cd web/frontend && pnpm run build`
- `scripts/quality-gate.sh`
- 本地重启 API 后，`GET /api/market/instruments/status` 可返回 OKX 最近 EOF 状态，研究页可见目录 warning。

失败：

- 无硬失败。

剩余风险：

- 研究页只展示最近状态，不提供 catalog 同步历史、手动强制后台重试队列或跨实例调度。
- OKX 当前宿主网络 EOF 仍是外部依赖风险；项目整体仍为 `scaffold`。

### 阶段 1 instrument catalog stale 健康补充

执行时间：2026-06-30

目标等级：scaffold

触发问题：

- `market_instrument_sync_statuses` 已记录最近成功和失败，但 `/api/system/health` 之前只在最近失败、从未同步或无状态行时降级。
- 如果某个交易所 catalog 最后一次同步成功已经很久以前，且之后没有失败记录，系统健康仍可能显示 ok，研究页创建任务所依赖的 active catalog 会被误看成新鲜数据。

修复范围：

- `market-instrument-catalog` health 增加 stale 判断：单个交易所 `last_success_at` 距当前时间超过 24 小时即进入 warning。
- stale warning 会让整体系统健康降级为 `degraded`，detail 中显示对应交易所和 `stale_since=<last_success_at>`。
- 保留原有优先级：最近失败仍优先展示 `last_error`，从未成功仍展示 `never_synced`。

验证：

- `go test ./internal/store/postgres -run 'TestIntegrationSystemHealthReports.*MarketInstrumentCatalog' -count=1` 通过，覆盖最近失败和 25 小时前成功两类 catalog warning。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：31 个测试文件、155 个测试通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

失败：

- 首次 `scripts/quality-gate.sh` 失败：新增测试使 `internal/store/postgres/market_store_integration_test.go` 达到 706 行，超过 700 行硬上限；已拆出 `system_health_market_catalog_integration_test.go`，复跑通过。

剩余风险：

- 当前 stale 阈值是本地健康判断的固定 24 小时，不等同于按每个环境的 `MARKET_INSTRUMENT_SYNC_INTERVAL` 动态计算。
- 仍未做跨实例 catalog 同步调度锁、同步历史事件、分布式退避或真实交易所恢复压测，因此阶段 1 不能升级为 usable。

### 阶段 1 交易所公开 market 请求本地限流补充

执行时间：2026-06-28

目标等级：demo

触发问题：

- Binance / OKX adapter 此前只在收到 HTTP 429 或 OKX `50011` 后把错误分类为临时错误并进入任务级 / 交易所级退避。
- 这属于事后保护，不能在请求发出前控制本进程对交易所公开 market API 的请求速率。

修复范围：

- 新增 `internal/exchange.FixedWindowRateLimiter`，支持加权请求、context 取消和 overweight 配置错误。
- Binance market client 默认启用本地 request-weight 固定窗口限流；K 线 `/api/v3/klines` 消耗 weight=2，`/api/v3/exchangeInfo` 消耗 weight=20。
- OKX market client 默认启用 20 次 / 2 秒固定窗口限流；history-candles 和 public instruments 均按 1 次请求消耗。
- `hi sync` 复用同一组 Binance / OKX client 运行 K 线同步和 instrument catalog 同步，因此同一进程内共享限流器。
- API 手动 instrument 同步入口也使用同样的默认限流配置。
- `.env.example` 和 `docker-compose.yml` 暴露 `BINANCE_REQUEST_WEIGHT_LIMIT`、`BINANCE_REQUEST_WEIGHT_WINDOW`、`OKX_MARKET_REQUEST_LIMIT`、`OKX_MARKET_REQUEST_WINDOW`。
- Binance adapter 在 rate limiter 或 HTTP 请求返回 `context.Canceled` 时直接透传，避免 shutdown 被 base URL fallback 摘要包装后误判为普通失败。

验证：

- `go test ./internal/exchange ./internal/adapter/binance ./internal/adapter/okx ./cmd/hi -count=1` 通过。
- Binance adapter 单元测试证明 K 线请求在 HTTP 前消耗 weight=2，instrument 请求在 HTTP 前消耗 weight=20，限流等待失败时不会发出 HTTP 请求。
- OKX adapter 单元测试证明 K 线和 instruments 请求在 HTTP 前进入 limiter，限流等待失败时不会发出 HTTP 请求。
- `internal/exchange` 单元测试覆盖固定窗口等待、context deadline 和 overweight 配置错误。

失败：

- 无。

剩余风险：

- 本轮是单进程固定窗口限流，不是跨副本 / 多进程共享额度。
- Binance request-weight 总额度仍由配置给定，尚未动态读取并应用 `exchangeInfo.rateLimits` 或响应头中的已用权重。
- 没有执行真实 Binance / OKX 长时间公网压测；不能据此宣称数据同步稳定或 production-safe。

### 阶段 1 全历史 K 线缺口扫描可观察

目标等级：demo

触发问题：

- 研究页此前只能看到当前 CandleProvider 窗口缺口和单个 data sync task 窗口内缺口。
- 数据同步 worker 审计项仍标记缺少跨任务、跨窗口的已落库全历史相邻 K 线缺口扫描。

修复范围：

- 新增 `MarketCandleGapScan` / `MarketCandleGapScanQuery` 数据模型。
- 新增 PostgreSQL 只读扫描，按 exchange / symbol / interval 对 `market_candles` 全历史相邻 `open_time` 做窗口函数检测，返回扫描窗口、K 线数量、总缺口数、返回数量、limited 和前 N 个缺口。
- 新增 `GET /api/market/candle-gaps?exchange=&symbol=&interval=&limit=`，受登录会话保护，limit 最大 100，OpenAPI contract 和生成 TypeScript DTO 已覆盖。
- 研究页当前数据源 metadata 新增全历史缺口扫描 tag，可显示扫描中、失败、无缺口和缺口总数，首个缺口信息进入 tag title。

验证：

- `go test ./internal/web/api ./internal/store/postgres -run 'TestMarketCandleGap|TestIntegrationScanMarketCandleGaps|TestAPIContract|TestFrontendAPI|TestAPIMethod' -count=1` 通过。
- `pnpm --dir web/frontend run test -- data.test.ts MarketCandleGapTag.test.ts ResearchPage.layout.test.ts` 通过。

失败：

- 无。

剩余风险：

- 本轮只做已落库数据的相邻缺口只读扫描，不自动创建补同步任务。
- 不覆盖任务起点前交易所实际历史是否应存在的数据，也不证明尾部实时同步已经追平交易所。
- 未做大规模历史表性能压测，不能据此宣称数据同步 usable 或 production-safe。

### 阶段 1 全历史 K 线单缺口补同步入口

目标等级：demo

触发问题：

- 全历史 K 线缺口扫描只能观察，不能从研究页直接为真实缺口排补同步任务。
- 既有 `repair-gap(s)` API 绑定到已有 data sync task 窗口，不能表达跨任务、全历史已落库相邻缺口。

修复范围：

- 新增 `RepairMarketCandleGapRequest` 和 repository/store/API 边界。
- 新增 `POST /api/market/candle-gaps/repair`，写请求受 session + CSRF 保护，后端校验 exchange / symbol / interval / from / to。
- PostgreSQL repair 会重新基于 `market_candles` 的 `LAG(open_time)` 验证请求窗口确实是相邻缺口；不是真实缺口返回 `404`。
- 对真实缺口创建 `data_sync_tasks` 补同步任务，`sync_enabled=true`、`realtime_enabled=false`、`status=pending`，无任务来源时 `repair_source_task_id` 为空；同 exchange/symbol/interval/start/end 重复请求返回 `skippedExisting=1`。
- 研究页全历史缺口 tag 可打开详情弹窗，展示返回缺口并对单个缺口排补同步任务，成功后刷新任务列表。
- OpenAPI contract、生成 TypeScript DTO、前端 API wrapper 和组件测试已覆盖。

验证：

- `go test ./internal/web/api ./internal/store/postgres -run 'TestMarketCandleGap|TestIntegrationRepairMarketCandleGap|TestAPIContract|TestAPIMethod|TestFrontendAPI' -count=1` 通过。
- `pnpm --dir web/frontend run test -- src/services/api/data.test.ts src/components/research/MarketCandleGapTag.test.ts src/pages/ResearchPage.layout.test.ts` 通过，21 个测试文件 / 98 个测试。
- 本地 8080 HTTP smoke 通过：登录后写入临时 `market_candles` 缺口，`GET /api/market/candle-gaps` 扫出 1 个缺口，`POST /api/market/candle-gaps/repair` 创建 1 个补同步任务，重复请求返回 `skippedExisting=1`；临时数据已清理。

失败：

- 无。

剩余风险：

- 本轮只支持用户选择单个已返回全历史缺口进行 repair，不做全历史批量自动修复。
- 不证明交易所实际历史数据完整，也不修复公网 EOF / 临时网络错误根因。
- 未做大规模历史表 repair 查询性能压测，不能据此把数据同步提升到 usable。

### 阶段 1 研究页 K 线可视边界加固

目标等级：demo

触发问题：

- 用户在本地 127.0.0.1:8080 观察到 K 线图表容器裁掉右侧价格轴、底部时间轴和图表内容。
- 前一轮固定高度只证明内部 DOM 高度不会反向撑爆页面，但没有覆盖首尾轴标签贴边裁切。

修复范围：

- 761-980px 窄桌面断点把研究页固定图表 viewport 上限从 560px 降到 500px，给新增 metadata 和浏览器可视高度留出余量。
- `rightPriceScale.minimumWidth` 提高到 132，避免价格轴和最新价标签在窄容器或浏览器缩放场景下被截断。
- `.research-chart-body` 声明 `--tt-chart-fixed-inline-end-gutter: 12px` 和 `--tt-chart-fixed-block-end-gutter: 12px`；`TradingViewChart` 读取固定槽时从渲染宽高扣除该安全留白，使价格轴和时间轴不再贴着 `overflow: clip` 边界。
- `TradingViewChart` 在 `fitContent()` 后按当前渲染宽度折算约 64px 的左右 logical padding，并用 `data.length` 显式设置完整可视范围，避免首尾时间标签裁切，同时不退化成只显示尾部窗口。

验证：

- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过，21 个文件 / 98 个测试。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `docker compose up -d --build && curl -fsS http://127.0.0.1:8080/readyz` 通过。
- `node scripts/research-chart-height-smoke.mjs http://127.0.0.1:8080/research` 通过，desktop / 812x1320 / mobile 三组采样高度稳定。
- headless Chrome 812x1320 实际截图确认：固定图表体 500px，chart/tv 渲染高度 488px，右侧价格轴 132px，右/底各保留 12px 安全留白；右侧价格标签完整，底部时间轴不再贴边裁字。

失败：

- 无。

剩余风险：

- 本轮是研究页 K 线可视边界加固，不是完整桌面/移动/暗色主题视觉回归体系。
- lightweight-charts 交互工具仍薄，不能据此把研究页提升到 usable。

### 阶段 1 研究页图表窄视口溢出约束

目标等级：demo

触发问题：

- 用户在本地 127.0.0.1:8080 观察到研究页 K 线图表区域在窄视口下仍有内容被裁掉的观感。
- 排查确认 lightweight-charts 固定槽本身稳定，但研究页工具栏 metadata 在移动宽度下可横向溢出，且横向布局的 `flex-basis` 在纵向断点里会把图表工具栏撑高。

修复范围：

- 将研究页样式拆到 `ResearchPage.css`，避免 `ResearchPage.vue` 为修布局突破 Vue 文件 450 行硬上限。
- 研究页图表面板、toolbar、context、metadata 增加 `min-width: 0` / `max-width: 100%` 约束，长 metadata tag 使用单行 ellipsis，不再把图表面板撑出横向滚动。
- 移动断点下把 `.research-context` 从横向布局的 `flex: 1 1 320px` 收敛为 `flex: 0 1 auto`，避免纵向 flex-basis 把工具栏高度撑到 320px。
- `scripts/research-chart-height-smoke.mjs` 增加横向溢出断言，要求研究页图表 panel/body/chart/canvas/tv 的 `scrollWidth <= clientWidth`。

验证：

- `pnpm --dir web/frontend run test -- src/pages/ResearchPage.layout.test.ts src/components/chart/TradingViewChart.test.ts` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `docker compose build api && docker compose up -d api && curl -fsS http://127.0.0.1:8080/readyz` 通过。
- `node scripts/research-chart-height-smoke.mjs` 通过，desktop / 812x1320 / mobile 三组采样高度稳定且无横向溢出；mobile 图表 panel 730px、body/chart/tv 457px。
- headless Chrome 390x844 截图确认：document 宽度 390，图表 panel / body / tv 的 `scrollWidth` 均等于 `clientWidth`，窗口 metadata 以 ellipsis 收敛。

失败：

- 无。

剩余风险：

- 本轮只修研究页图表卡片及其 metadata 的窄视口溢出，不处理宽数据表和顶部导航在移动宽度下的完整产品级重排。
- 仍缺系统性视觉回归矩阵，不能据此把研究页或前端基础设施提升到 usable。

### 阶段 1 数据同步交易所退避可观察补充

目标等级：demo

触发问题：

- 数据同步 worker 已按交易所持久化 `data_sync_exchange_backoffs`，claim 也会跳过 active 冷却交易所。
- 研究页任务表此前只能看到任务级 retry，无法解释“任务看起来运行中但因为交易所级冷却没有继续推进”的状态。

修复范围：

- `DataSyncTask` 增加 `exchangeBackoffUntil` 和脱敏前 `exchangeBackoffLastError` 字段，API 响应阶段统一脱敏。
- PostgreSQL 任务列表把 active `data_sync_exchange_backoffs` 投影到每条同交易所任务，并把 dataHealth 标为 `retrying`。
- 研究页任务表新增“交易所退避 / Exchange backoff”列，用 tooltip 展示脱敏错误摘要，不泄露请求 path 和 query。
- 前端 API wrapper、生成 DTO、后端 API sanitization 和 PostgreSQL 集成测试同步更新。

验证：

- `go test ./internal/store/postgres ./internal/web/api -run 'TestIntegrationListDataSyncTasksReports|TestDataSyncTaskRoutesSanitizeLastError|TestAPIContract|TestFrontendAPI|TestAPISchemaDrift' -count=1` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过，21 个测试文件 / 99 个测试。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run build` 通过。

失败：

- 无。

剩余风险：

- 本轮只补可观察性，不实现分布式共享限流或动态读取交易所 `rateLimits`。
- 没做真实交易所网络压力测试，不能据此把数据同步提升到 usable。

### 阶段 1 研究页 K 线安全区二次加固

目标等级：demo

触发问题：

- 用户在本地 127.0.0.1:8080 继续观察到研究页 K 线图表容器有内容被截掉的观感。
- 既有 smoke 已能证明高度稳定和横向不溢出，但右侧价格轴、底部时间轴只是最小贴边通过，安全边界不够硬。

修复范围：

- `.research-chart-body` 固定槽右侧安全留白从 12px 提高到 24px，底部安全留白从 12px 提高到 20px。
- `rightPriceScale.minimumWidth` 从 132 提高到 156，给价格轴和最新价标签保留更宽显示空间。
- `TradingViewChart` 的时间轴 logical padding 从约 64px 提高到约 72px，减少首尾时间标签贴边裁切风险。
- `scripts/research-chart-height-smoke.mjs` 把轴线 inset 下限从 6px 提高到 16px，后续回归会直接失败。

验证：

- `pnpm --dir web/frontend run test -- src/components/chart/TradingViewChart.test.ts src/pages/ResearchPage.layout.test.ts` 通过，21 个测试文件 / 99 个测试。
- `pnpm --dir web/frontend run test` 通过，21 个测试文件 / 99 个测试。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `docker compose up -d --build` 成功，`curl -fsS http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`。
- `curl http://127.0.0.1:8080/assets/ResearchPage-DMJPPtvt.css` 确认产物包含 `--tt-chart-fixed-inline-end-gutter: 24px` 和 `--tt-chart-fixed-block-end-gutter: 20px`。
- `node scripts/research-chart-height-smoke.mjs http://127.0.0.1:8080/research` 通过；desktop / 812x1320 / mobile 三组高度稳定，812x1320 下 chart/tv 为 480px。
- headless Chrome 812x1320 实际截图确认：固定图表体 500px，右轴 canvas 156px，右侧 inset 24px，底部时间轴 inset 20px，document 宽度 812 且无横向滚动。

失败：

- 无。

剩余风险：

- 本轮继续只修研究页 K 线固定容器，不构成完整视觉回归体系。
- 宽数据同步表在窄宽度下仍主要依赖横向滚动，不在本轮图表提交范围内。

### 阶段 1 数据同步重启恢复验收补强

目标等级：demo

触发问题：

- 计划要求实时同步任务在系统重启后能根据持久化游标继续同步。
- 既有测试分别覆盖 runner overlap 计算、lease 释放和 store claim，但缺少 PostgreSQL store + data sync runner 组合验收，无法证明过期 running lease 的 realtime 任务能完整恢复到研究页可观察状态。

修复范围：

- 新增 `integration_data_sync_resume_test.go`，用真实 PostgreSQL store 和 `datasync.Runner` 组合跑恢复场景。
- 测试模拟旧 worker 崩溃后遗留 `status=running`、`realtime_enabled=true`、`locked_until` 已过期的任务。
- 测试断言新 runner 会重新 claim 任务，并从 `last_synced_open_time - overlap` 发起拉取。
- 测试断言 overlap 内既有 K 线通过 upsert 更新且不产生重复 open_time，`last_synced_open_time` 推进到新游标。
- 测试断言任务保存结果后仍保持 `running + realtime_enabled`，释放当前 lease，`ListDataSyncTasks` 能返回推进后的 `latestSyncedAt` 和 `dataHealth=syncing`。

验证：

- `go test ./internal/store/postgres -run TestIntegrationDataSyncRunnerResumesRealtimeTaskFromExpiredLease -count=1` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过，21 个测试文件 / 99 个测试。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

失败：

- 无。

剩余风险：

- 本轮验证的是受控 fake exchange client 下的 PostgreSQL + runner 恢复语义，不是 Docker 级进程崩溃恢复压测。
- 未覆盖真实 Binance / OKX 网络抖动、交易所限流和多实例并发恢复，不能据此把数据同步提升到 usable。

### 阶段 1 数据同步 Docker 重启恢复 smoke 补强

目标等级：demo

触发问题：

- 上一轮只证明 PostgreSQL store + runner 组合恢复语义，仍缺真实 Docker Compose 运行形态下的 `hi sync` 容器重启恢复证据。
- 计划要求实时同步任务在系统重启后立即恢复，且恢复时应从持久化游标向前 overlap 拉取并通过 upsert 去重。

修复范围：

- 新增 `scripts/stage1-data-sync-restart-smoke.sh`。
- 脚本临时生成 Docker Compose override，注入 Docker 网络内的 Binance-compatible `restart-market` mock，不依赖公网 Binance。
- 脚本先停止并移除当前 `sync` 容器，再 seed 一个 `status=running`、`realtime_enabled=true`、`locked_until` 已过期、`last_synced_open_time=2026-01-01T00:02:00Z` 的任务，模拟旧 sync 进程崩溃遗留 lease。
- 脚本重启真实 `sync` service，断言新 worker 会 claim 该任务、请求 mock `/api/v3/klines`，并把 overlap 区间内旧 K 线 upsert 为新值。
- 脚本断言 `last_synced_open_time` 推进到 `2026-01-01T00:04:00Z`，任务保持 `running + realtime_enabled=true`，当前 lease 已释放，且 `/api/data/tasks` 可观察到 `latestSyncedAt` 和 `dataHealth=syncing`。
- 脚本运行前保存 `binance` 交易所退避行，测试期间临时清空以避免 claim 被冷却窗口挡住，退出时恢复；只暂停 `S1RESTART...` smoke 命名空间任务。

验证：

- `bash -n scripts/stage1-data-sync-restart-smoke.sh` 通过。
- `scripts/stage1-data-sync-restart-smoke.sh` 通过；证据：`symbol=S1RESTART1782665970USDT`、`task=dst_s1restart_1782665970`、`syncWorker=stage1-restart-1782665970`、`cursor=2026-01-01T00:04:00Z`、`klinesHits=1`。
- smoke 退出后 `docker inspect` 确认 `tictick-hi-api-1` 和 `tictick-hi-sync-1` 的 `com.docker.compose.project.config_files` 均恢复为 `/Users/xiaobai/Work/TicTick/tictick-hi/docker-compose.yml`，`docker compose ps -a` 未残留 `restart-market`。

失败：

- 无。

剩余风险：

- 本轮是 Docker Compose + 本地 mock exchange 的容器重启 smoke，不是 Docker daemon kill、宿主机断电或 Kubernetes 编排恢复测试。
- 未覆盖真实 Binance / OKX 网络抖动、长时间限流、多 sync 实例并发 claim 和跨交易所共享限流，不能据此把数据同步提升到 usable 或 production-safe。

### 阶段 1 CandleProvider 请求窗口边界缺口补强

目标等级：demo

触发问题：

- CandleProvider 之前只检测返回 K 线之间的 open_time 断点；显式查询 `from/to` 时，如果窗口起点或终点缺 K 线但返回结果内部连续，`health` 仍可能是 `ok`。
- 这会让研究页、回测和交易 runner 在请求指定窗口时误把不完整窗口当作健康数据。

修复范围：

- 新增 `DetectCandleGapsInRange`，在 UTC 周期网格上计算显式窗口边界缺口。
- `DetectCandleGaps` 保持兼容，默认只检测返回 K 线之间的缺口。
- CandleProvider 的 native 和 1m aggregation fallback 路径改为传入 `query.From/query.To`，因此窗口起点到首根 K 线、末根 K 线到窗口终点、整窗无数据都会进入 `gaps`。
- 没有显式边界的最新窗口查询不额外制造首尾缺口。

验证：

- `go test ./internal/data -run 'TestCandleProvider' -count=1` 通过。
- Docker network PostgreSQL 定向集成测试通过：`docker run --rm --network tictick-hi_default -v "$PWD":/src -w /src -e TICTICK_TEST_DATABASE_URL='postgresql://tictick:tictick-local-postgres-password@postgres:5432/tictick_hi?sslmode=disable' golang:1.26-bookworm go test ./internal/store/postgres -run TestIntegrationCandleProviderReportsRequestedRangeBoundaryGaps -count=1 -v`。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过，21 个测试文件 / 99 个测试。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

失败：

- 无。

剩余风险：

- 本轮只补显式窗口边界缺口，不补完整 cursor token pagination。
- 未做大范围历史表性能压测，不能据此把 CandleProvider 或阶段 1 升级为 usable。

### 阶段 1 研究页图表固定容器与任务表裁切修复

目标等级：scaffold

触发问题：

- 用户在本地 127.0.0.1:8080 继续观察到 K 线图表内容被截掉，以及研究页可视质量不符合生产级要求。
- 现有研究页图表面板仍继承全局 `.chart-panel` class；该全局 class 带固定高度和 `contain: size layout paint`，需要通过覆盖规则抵消，生产构建或 CSS 顺序变化时不够稳。
- 图表 root / canvas / lightweight-charts 外层使用 `inset: 0`，同时由 JS 写入固定宽高，属于过约束定位，真实浏览器里容易出现右/下边界解算不一致。
- 同步任务表外层使用 `overflow: hidden`，且操作列不固定；在窄桌面宽度下，用户需要横向滚到最右才能找到关键操作，观感上等同于操作列被裁掉。

修复范围：

- 研究页图表 panel 从 `surface chart-panel research-chart-panel` 改为 `surface research-chart-panel`，不再继承全局 `.chart-panel` 的固定高度和 size containment。
- `TradingViewChart.css` 中 `.trading-chart`、`.trading-chart__canvas` 和 `.tv-lightweight-charts` 外层定位改为显式 `top/right/bottom/left`，不再使用 `inset: 0` 与 JS 写入尺寸混用。
- `ResearchPage.layout.test.ts` 增加静态断言，防止研究页重新继承全局 `.chart-panel`，并防止图表根节点回退到 `inset: 0`。
- `scripts/research-chart-height-smoke.mjs` 增加运行态 classList 断言，确保真实页面图表 panel 不带全局 `.chart-panel`；同时断言任务表外层不再隐藏溢出。
- `.research-tasks-panel` 从隐藏溢出改为可滚动视口，数据同步任务表 actions 列固定在右侧，避免窄宽度裁掉关键操作。

验证：

- `pnpm --dir web/frontend exec vitest run src/pages/ResearchPage.layout.test.ts src/components/chart/TradingViewChart.test.ts src/components/tables/DataSyncTaskTable.test.ts` 通过，3 个测试文件 / 34 个测试。
- `node --check scripts/research-chart-height-smoke.mjs` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `docker compose up -d --build api` 通过，8080 返回新入口 `/assets/index-BpxxOPBE.js`。
- `SMOKE_SAMPLES=30 SMOKE_INTERVAL_MS=100 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过；desktop / 812x1320 / mobile 三组高度稳定，812x1320 下 panel 为 `surface research-chart-panel`，chart/tv 为 480px。
- Headless Chrome 812x1320 截图 `/tmp/tictick-hi-research-812x1320-final.png`：图表右侧和底部有安全留白，时间轴和价格轴可见；任务表 panel 为 `overflow:auto`，操作列固定在右侧。

失败：

- 无。

剩余风险：

- 本轮修复的是研究页固定容器和任务表裁切，不是完整桌面 / 移动 / 暗色主题视觉回归体系。
- 图表工具能力仍薄，不能据此把研究页或项目整体升级为 usable。

### 阶段 1 研究页 K 线固定槽完整渲染修复

目标等级：scaffold

触发问题：

- 用户在本地 `127.0.0.1:8080/research` 截图反馈 K 线图表容器内容被截掉。
- 上一轮实现为了避免价格轴/时间轴贴裁切边，把固定槽宽高扣除右侧 24px / 底部 20px 后再传给 lightweight-charts。
- 真实 8080 旧构建 smoke 仍显示 `body 603 / chart 583 / tv 583`，说明图表实际渲染高度比固定槽少 20px，属于人为缩图，不是生产级布局。

修复范围：

- `TradingViewChart` 固定槽尺寸读取不再扣减 `--tt-chart-fixed-inline-end-gutter` / `--tt-chart-fixed-block-end-gutter`。
- `.research-chart-body` 删除旧 gutter CSS 变量，图表 root / canvas / lightweight-charts 外层使用固定槽完整 width / height 渲染。
- 保留固定槽高度快照、内部 resize entry 过滤和污染后恢复锁定，继续阻止 K 线图表无限拉高。
- `TradingViewChart.test.ts` 改为断言 chart/root/canvas 使用完整固定槽尺寸，并覆盖污染后仍恢复到固定槽尺寸。
- `ResearchPage.layout.test.ts` 明确禁止研究页重新声明 fixed gutter 变量。
- `scripts/research-chart-height-smoke.mjs` 从“轴线必须离裁切边至少 16px”改为“轴线 canvas 不越界，chart/canvas/tv 不得比固定槽小出人为留白”。

验证：

- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts src/pages/ResearchPage.layout.test.ts` 通过，2 个测试文件 / 25 个测试。
- `node --check scripts/research-chart-height-smoke.mjs` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过，21 个测试文件 / 100 个测试。
- `pnpm --dir web/frontend run build` 通过。
- `docker compose up -d --build api` 通过，已替换本地 8080 API 容器。
- `curl -fsS http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`。
- `SMOKE_SAMPLES=20 SMOKE_INTERVAL_MS=100 SMOKE_SETTLE_MS=1000 BASE_URL=http://127.0.0.1:8080 node scripts/research-chart-height-smoke.mjs` 通过：desktop `body 603->603, chart 603->603, tv 603->603`；812x1320 `body 500->500, chart 500->500, tv 500->500`；mobile `body 457->457, chart 457->457, tv 457->457`。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

失败：

- 无。

剩余风险：

- 本轮修复的是研究页 K 线固定槽裁切/缩图问题，不是完整像素回归体系。
- 图表交互工具、指标层、移动端信息密度和全路由视觉基线仍不足，研究页和整体项目仍不能升级为 usable。

### 阶段 8 Go 子命令配置错误边界补充

目标等级：scaffold

触发问题：

- Stage 8 readiness 重审计将 Go 子命令维持为 `scaffold`，其中一个明确 blocker 是配置错误边界和启动日志粗糙。
- `hi sync` / `hi backtest` / `hi trading` / `hi notify` / `hi api` 之前直接在入口内读取 env，`durationEnv` / `intEnv` / `boolEnv` 对非法值静默回退，容易让容器以非预期配置运行。
- 交易所 public client 限流配置也由同一类静默回退函数读取，非法配置不能在启动前暴露。

修复范围：

- 新增 `cmd/hi/config.go`，把 API / sync / backtest / trading / notify 的 env 和 `--once` flag 配置构建收敛为可单测函数。
- 对关键 duration / int / bool 配置执行严格解析；非法值返回包含 env 名的错误，非正 duration、低于下限的 int、非法 bool 都不再静默回退。
- `SYNC_HEARTBEAT_INTERVAL` 默认由 `SYNC_LEASE_TTL / 3` 推导，并拒绝大于 lease TTL 的配置。
- Binance / OKX public market client 限流配置改为复用严格解析后的 `exchangeClientConfig`。
- API / sync / backtest / trading / notify 启动时输出非敏感配置摘要，摘要会过滤 `database_url`、password、secret、token、API key、private key、`ENCRYPTION_KEY`、credential 和 DSN 类 key。
- 移除 `cmd/hi/main.go` 中旧的静默回退 `durationEnv` / `intEnv` / `boolEnv`。
- 新增 `cmd/hi/config_test.go` 覆盖缺失 `DATABASE_URL`、非法 duration / int / bool、sync heartbeat 默认值、heartbeat 大于 lease、交易所限流非法值、脱敏摘要和未知 flag 错误。

验证：

- `go test ./cmd/hi` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过，21 个测试文件 / 100 个测试。
- `pnpm --dir web/frontend run build` 通过。
- `git diff --check` 通过。

失败：

- 无。

剩余风险：

- 本轮只补配置解析、脱敏启动摘要和单测证据，不补完整运行手册、结构化日志/trace、子命令级健康探针或容器级优雅停止新 smoke。
- Go 子命令仍保持 `scaffold`；整体项目仍不能升级为 usable 或 production-safe。

### 阶段 8 Go 子命令运行手册与配置 smoke 补充

目标等级：scaffold

触发问题：

- Stage 8 readiness 中 Go 子命令仍缺运行手册和可重复启动配置边界证据。
- 上一轮只有 `cmd/hi` 单元测试覆盖配置解析，没有真实编译后二进制的 smoke gate。
- 新增 smoke 首次运行时发现 `AUTH_COOKIE_SECURE must be a boolean, got "stage8_config_secret"` 会回显非法 env 原始值；如果误填 secret，会泄露到命令错误日志。

修复范围：

- 新增 `docs/go-command-runbook.md`，记录 `hi api/sync/backtest/trading/notify/migrate` 的用途、必要 env、敏感值边界、启动/停止方式、配置 smoke 和排障流程。
- 新增 `scripts/stage8-command-config-smoke.sh`，构建真实 `hi` 二进制并验证缺失 `DATABASE_URL`、非法 duration/int/bool、heartbeat 大于 lease、交易所限流非法值和未知 flag 都会启动前失败。
- smoke 断言错误输出包含具体 env 名，同时不包含测试 DSN、密码或 secret marker。
- `durationEnvStrict` / `intEnvStrict` / `boolEnvStrict` 不再回显非法原始值，只返回 env 名和类型要求。
- `scripts/quality-gate.sh` 将 command config smoke 纳入硬门禁。
- README 增加 Go 子命令运行手册和 command config smoke 入口。

验证：

- `bash -n scripts/stage8-command-config-smoke.sh` 通过。
- `scripts/stage8-command-config-smoke.sh` 通过。
- `go test ./cmd/hi` 通过。
- `scripts/quality-gate.sh` 通过，并执行 `command config smoke`。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过，21 个测试文件 / 100 个测试。
- `pnpm --dir web/frontend run build` 通过。
- `git diff --check` 通过。

失败：

- 初次 `scripts/stage8-command-config-smoke.sh` 失败，证明非法 bool 错误会回显 `stage8_config_secret`；已改为不回显原始 env 值并重跑通过。

剩余风险：

- 本轮补的是基础子命令运行手册和配置错误 smoke，不是生产部署运行手册。
- 仍缺结构化日志/trace、子命令级健康探针、生产资源限制和更完整优雅停止证据；Go 子命令仍保持 `scaffold`。

### 阶段 1 研究页图表布局静态门禁补充

目标等级：scaffold

触发问题：

- 研究页 K 线图表此前多次暴露高度反馈、容器裁切和人为缩图问题，虽然已有浏览器 smoke，但轻量 `scripts/quality-gate.sh` 不能在不启动 8080/Chrome 的情况下阻止高频源码回退。
- 当前本地运行态 smoke 已能证明 `/research` 三组 viewport 高度稳定，但运行态 smoke 仍依赖本机 Chrome 和已启动的本地服务，不适合直接放入轻量门禁。

修复范围：

- 新增 `scripts/check-research-chart-layout.sh`，用源码契约检查研究页任务列表在上、图表在下、图表固定槽带 `data-chart-viewport="fixed"`、研究页不重新继承全局 `.chart-panel`、固定槽不声明旧 gutter 变量。
- 检查 `TradingViewChart.css` 中 `.trading-chart`、`.trading-chart__canvas` 和 `.trading-chart__canvas > .tv-lightweight-charts` 三个具体 block 必须使用显式 `top/right/bottom/left` 与 `contain: strict`，且不能回退到 `inset: 0`。
- 检查 `scripts/research-chart-height-smoke.mjs` 仍覆盖 `narrow-desktop-812x1320`、初始首屏图表 fit、内部 table/tbody/tr/td/canvas 高度污染和固定图表槽完整渲染断言。
- `scripts/quality-gate.sh` 将 `research chart layout` 纳入硬门禁；浏览器级 `research-chart-height-smoke.mjs` 保持为本地 8080 后的运行态检查。
- README 明确轻量门禁与运行态浏览器 smoke 的边界。

验证：

- `bash -n scripts/check-research-chart-layout.sh` 通过。
- `scripts/check-research-chart-layout.sh` 通过。
- `node --check scripts/research-chart-height-smoke.mjs` 通过。
- `SMOKE_SAMPLES=8 SMOKE_INTERVAL_MS=100 SMOKE_SETTLE_MS=1000 BASE_URL=http://127.0.0.1:8080 node scripts/research-chart-height-smoke.mjs` 通过：desktop `doc 1310->1310, panel 752->752, body 603->603, chart 603->603, tv 603->603`；812x1320 `doc 1320->1320, panel 669->669, body 500->500, chart 500->500, tv 500->500`；mobile `doc 1362->1362, panel 730->730, body 457->457, chart 457->457, tv 457->457`。
- `pnpm --dir web/frontend exec vitest run src/pages/ResearchPage.layout.test.ts src/components/chart/TradingViewChart.test.ts` 通过，2 个测试文件 / 25 个测试。
- `scripts/quality-gate.sh` 通过，并执行 `research chart layout`。
- `git diff --check` 通过。

失败：

- 首次 `scripts/check-research-chart-layout.sh` 失败：CSS 变量 `--research-chart-viewport-height` 被 `grep` 当成参数解析；已改为 `grep --`。
- 第二次定向检查失败：全文件禁止 `inset: 0` 误伤 `.trading-chart__empty` 空状态 overlay；已收敛为只检查三个图表布局 block。

剩余风险：

- 本轮补的是源码级布局契约门禁和本地短采样浏览器证据，不是完整像素快照系统。
- 研究页图表交互工具、指标层、全语言/全主题视觉矩阵和长期浏览器采样仍不足，研究页和整体项目仍不能升级为 usable。

### 阶段 1 CandleProvider opaque cursor 分页补充

目标等级：scaffold

触发问题：

- CandleProvider 虽然返回 `previousFrom/previousTo/nextFrom/nextTo`，但前端上一/下一窗口仍直接拼时间窗口，不是稳定分页协议。
- Stage 1 审计仍记录缺少完整 cursor pagination；这会让 URL、API contract 和前端调用链容易漂移。

修复范围：

- 新增 `internal/data.CandleCursor`，使用 base64url JSON opaque token 绑定 `exchange/symbol/interval/from/to/limit`，不包含 secret。
- `CandlePagination` 增加 `previousCursor/nextCursor`，同时保留旧 `previousFrom/previousTo/nextFrom/nextTo` 兼容字段。
- `/api/candles?cursor=...` 解码 cursor 后加载相邻窗口；cursor 不能与 `from/to/limit` 混用，且必须匹配当前 `exchange/symbol/interval`。
- `/api/system/api-contract` 增加 `cursor` query 参数，`web/frontend/src/types/api.generated.ts` 已重新生成。
- 研究页 URL 优先保留 `cursor`，上一/下一窗口按钮优先使用 cursor；旧 `from/to` URL 仍可打开。
- `useResearchWorkspace.ts` 因新增逻辑触发文件行数门禁，窗口选择逻辑已下沉到 `researchWorkspaceHelpers.ts`，主 composable 保持在 400 行硬上限内。

验证：

- `go test ./internal/data ./internal/web/api -run 'TestCandle|TestCandles|TestFrontendAPICandle|TestFrontendAPIGeneratedTypesAreCurrent' -count=1` 通过。
- `pnpm --dir web/frontend exec vitest run src/composables/useResearchWorkspace.test.ts src/services/api/data.test.ts` 通过，2 个测试文件 / 24 个测试。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过，21 个测试文件 / 102 个测试。
- `pnpm --dir web/frontend run build` 通过。
- `git diff --check` 通过。

失败：

- 首次完整 `scripts/quality-gate.sh` 失败：`web/frontend/src/composables/useResearchWorkspace.ts` 增至 422 行，超过 400 行硬上限。
- 修复方式：将上一/下一窗口选择逻辑下沉到 `researchWorkspaceHelpers.ts`，并保持 `useResearchWorkspace.ts` 为 399 行；重跑 `scripts/quality-gate.sh` 通过。

剩余风险：

- 本轮只补 adjacent-window opaque cursor，不做聚合缓存、预取、虚拟化或大范围历史性能压测。
- aggregated 前后探测仍基于基础 `1m` 是否存在，下一页完整健康仍需要结合 `health/gaps/coverage` 判断。
- CandleProvider 仍保留后续风险；研究页和项目整体仍是 `scaffold`。

### 阶段 1 CandleProvider 聚合基础窗口分页补充

目标等级：scaffold

触发问题：

- 高周期聚合 fallback 之前只读取单页基础 `1m` K 线，`1h limit=1000` 需要 60000 根基础 K 线时会被 5000 根单页上限截断。
- 默认最新窗口多页读取后，聚合结果如果仍按头部裁剪，会丢掉最新 K 线。

修复范围：

- `NativeCandleStore` 增加 `ListLatestNativeCandles`，PostgreSQL 实现可在可选 `from/to` 范围内按最新页读取，再按升序返回。
- CandleProvider 聚合 fallback 改为最多 12 页 / 60000 根基础 `1m` 的有界分页读取；显式 `from` 窗口正向翻页，默认最新或仅 `to` 窗口后向翻页。
- 聚合基础查询会按目标周期把 `from/to` 投影到需要的 `1m` 基础窗口；聚合结果会按请求范围过滤。
- 默认最新聚合结果超过 limit 时取尾部，显式 `from` 请求仍取头部，避免丢掉最新 K 线。
- coverage 的 `requiredBaseCandles/baseLimit/returnedBaseCandles/limitedByBaseWindow` 继续暴露基础读取需求、有效上限和实际返回。

验证：

- `go test ./internal/data -run 'TestCandleProvider' -count=1` 通过。
- `go test ./internal/web/api -run 'TestCandles|TestFrontendAPI' -count=1` 通过。
- `go test ./internal/store/postgres -run 'TestIntegrationCandleProvider|TestIntegrationListNativeCandles' -count=1` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过，21 个测试文件 / 102 个测试。
- `pnpm --dir web/frontend run build` 通过。
- `git diff --check` 通过。

失败：

- 本轮未出现失败检查。

剩余风险：

- 60000 根基础 K 线是有界读取上限，不是无限历史查询方案；`4h/1d` 大窗口仍可能因为超过上限而返回 `limitedByBaseWindow=true`。
- 已有 60000 根基础 K 线的单次 PostgreSQL 性能 smoke；仍未做长期/并发性能压测、聚合缓存、预取、虚拟化或批量异常数据修复策略。
- CandleProvider 仍保留后续风险；研究页和项目整体仍是 `scaffold`。

### 阶段 1 CandleProvider 异常 K 线边界补充

目标等级：scaffold

触发问题：

- CandleProvider 之前会对 store 返回的 K 线直接做缺口检测或聚合；如果底层出现 open_time 未按 interval 对齐、重复 open_time、close_time 不匹配 interval 等异常数据，可能被排序、缺口检测或聚合流程掩盖。
- 这些异常会影响研究页图表、回测 runner 和交易 runner 的共同数据入口，不能静默返回健康数据。

修复范围：

- 新增 `validateCandleSeries`，校验 K 线序列的 open_time interval 对齐、close_time = open_time + interval、升序和重复 open_time。
- CandleProvider 在 native 路径和 aggregation base `1m` 路径都先校验 K 线序列，发现异常直接返回错误，不继续生成 health=ok 的结果。
- 单元测试覆盖校验器本身，以及 `GetCandles` 的 native/base 两条异常路径。

验证：

- `go test ./internal/data -run 'TestCandleProvider|TestValidateCandleSeries|TestAggregateCandles|TestDetect' -count=1` 通过。
- `go test ./internal/web/api -run 'TestCandles|TestFrontendAPI' -count=1` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过，21 个测试文件 / 102 个测试。
- `pnpm --dir web/frontend run build` 通过。
- `git diff --check` 通过。

失败：

- 本轮未出现失败检查。

剩余风险：

- 本轮没有新增 API 级错误码；底层异常仍会作为服务端错误被 API 层统一处理。
- 未覆盖 OHLCV 业务合法性、异常尖刺、交易所修正数据审计或坏数据自动修复策略。
- 已有 60000 根基础 K 线的单次 PostgreSQL 性能 smoke；仍未做长期/并发性能压测、聚合缓存或超过 60000 根基础 K 线的分段策略。

### 阶段 1 CandleProvider 大窗口性能 smoke 补充

目标等级：scaffold

触发问题：

- CandleProvider 已支持最多 60000 根基础 `1m` 的聚合分页读取，但之前缺少真实 PostgreSQL 大窗口查询证据。
- 阶段 1 审计仍保留“大范围性能压测”风险，无法区分单次 60000 根查询边界和后续长期/并发压测风险。

修复范围：

- 新增 `TestIntegrationCandleProviderLargeAggregationWindowPerformance`，在真实 PostgreSQL 中写入 60000 根 `1m` K 线，通过 `Store.GetCandles` 请求 `1h limit=1000`，验证 `source=aggregated`、`health=ok`、coverage 完整且不受限。
- 新增 `scripts/stage1-candle-provider-perf-smoke.sh`，启动 compose PostgreSQL 后在 Docker 网络内运行单个集成测试；无 `.env` 或 `.env` 变量为空时可回退 `.env.example` 的本地默认值。
- smoke 默认阈值为 `TICTICK_CANDLE_PERF_MAX_MS=10000`，可通过环境变量调整。

验证：

- `go test ./internal/store/postgres -run TestIntegrationCandleProviderLargeAggregationWindowPerformance -count=1 -v` 在未设置 `TICTICK_TEST_DATABASE_URL` 时按集成测试约定跳过。
- `scripts/stage1-candle-provider-perf-smoke.sh` 通过；实测写入 60000 根基础 K 线后，聚合查询读取 60000 根并返回 1000 根 `1h` K 线，耗时 `412.778167ms`。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过，21 个测试文件 / 102 个测试。
- `pnpm --dir web/frontend run build` 通过。

失败：

- 首次 `scripts/stage1-candle-provider-perf-smoke.sh` 失败：脚本只读取 `.env`，当前本地缺少可用 `ENCRYPTION_KEY`。
- 修复方式：脚本先读取 `.env.example`，再读取 `.env` 覆盖，并对必需变量逐项回退到 `.env.example` 默认值；重跑通过。

剩余风险：

- 该 smoke 是单次本地 PostgreSQL 查询证据，不是长期、并发、冷缓存或真实生产数据分布压测。
- 超过 60000 根基础 K 线仍需要聚合缓存、分段查询或预取策略。
- CandleProvider 仍保留后续风险；研究页和项目整体仍是 `scaffold`。

### 阶段 1 研究页 K 线内部布局裁切修复

目标等级：scaffold

触发问题：

- 研究页 K 线图表仍有用户可见裁切反馈，前几轮高度防护把 lightweight-charts 内部 `table/tbody/tr/td/canvas` 也纳入外部固定宽高约束，存在干预图表库自行分配主图、右侧价格轴和底部时间轴的风险。
- 既有运行态 smoke 只验证右侧价格轴和底部时间轴 canvas，未显式验证主图 canvas 左/上/右/下边界是否完整落在固定图表槽内。

修复范围：

- `.trading-chart__canvas > .tv-lightweight-charts` 外层继续使用固定槽完整宽高和 `contain: strict`，但删除对内部 `table/tbody/tr/td/canvas` 的整图尺寸强制，让 lightweight-charts 自行分配主图、价格轴和时间轴区域。
- `ResearchPage.layout.test.ts` 和 `scripts/check-research-chart-layout.sh` 增加源码契约，防止再次用外部 CSS 覆盖 lightweight-charts 内部 table/canvas 几何。
- `scripts/research-chart-height-smoke.mjs` 增加 `mainPaneCanvas` 采样和边界断言，运行态同时验证主图 canvas、右侧价格轴 canvas 和底部时间轴 canvas 都在固定槽内。
- 本地 `api` 容器已通过 `docker compose up -d --build api` 重建，`http://127.0.0.1:8080/research` 当前加载的是本轮构建产物。

验证：

- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts src/pages/ResearchPage.layout.test.ts` 通过，2 个测试文件 / 26 个测试。
- `scripts/check-research-chart-layout.sh` 通过。
- `docker compose up -d --build api` 通过并重启本地 8080 API。
- `scripts/research-chart-height-smoke.mjs` 在更新后的 8080 上通过：desktop `body/chart/tv 603->603`，narrow desktop `500->500`，mobile `457->457`。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过，21 个测试文件 / 103 个测试。
- `scripts/quality-gate.sh` 通过。
- `pnpm --dir web/frontend run build` 通过，入口 chunk `449.09 kB`。
- `git diff --check` 通过。

剩余风险：

- 本轮关闭的是研究页 K 线固定槽内主图/轴 canvas 裁切风险和内部 table/canvas 外部几何覆盖风险，不是完整像素快照回归体系。
- 研究页仍缺指标层、十字线工具、图表设置、全主题/全语言/真实浏览器矩阵视觉基线和长期采样；项目整体仍为 `scaffold`。

### 阶段 1 研究页成交量图层补充

目标等级：scaffold

触发问题：

- 后端 `Candle` contract 已包含 `volume`，CandleProvider 聚合规则也要求高周期 `volume` 由基础 K 线求和，但前端 `ChartCandle` 和 `normalizeCandleResult()` 丢弃了 volume。
- 研究页实施计划把成交量列为图表逐步增强项，当前 K 线图只有 OHLC 主图，无法支撑基础行情研究。

修复范围：

- `ChartCandle` 增加必需 `volume` 字段，前端 API 映射从 `/api/candles` 的 decimal string 解析为 number；任何 OHLCV 字段不是有限数字时丢弃整根 candle，避免 K 线层和成交量层时间错位。
- `TradingViewChart` 增加 lightweight-charts `HistogramSeries` 成交量图层，绑定 overlay price scale，隐藏成交量 last value / price line，并按 K 线涨跌使用绿色/红色半透明柱。
- 主 K 线价格轴通过 `scaleMargins` 给底部成交量区域留出空间，成交量图层使用同一固定图表槽和同一时间轴，不引入额外 DOM 容器。

验证：

- `pnpm --dir web/frontend exec vitest run src/services/api/data.test.ts src/components/chart/TradingViewChart.test.ts` 通过，2 个测试文件 / 29 个测试。
- `pnpm --dir web/frontend run typecheck` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过，21 个测试文件 / 105 个测试。
- `pnpm --dir web/frontend run build` 通过，入口 chunk `449.09 kB`。
- `scripts/quality-gate.sh` 通过。
- `docker compose up -d --build api` 通过并重启本地 8080 API。
- `scripts/research-chart-height-smoke.mjs` 在更新后的 8080 上通过：desktop `body/chart/tv 603->603`，narrow desktop `500->500`，mobile `457->457`。
- Headless Chrome 截图核验底部成交量柱已渲染在固定图表槽内，右侧价格轴和底部时间轴未被裁切。
- `git diff --check` 通过。

剩余风险：

- 本轮只把后端 volume 数据接入研究页图表，不包含成交量均线、指标层、图表设置或完整视觉回归。
- 研究页和项目整体仍是 `scaffold`，不能升级。

### 阶段 1 研究页图表缺口标记补充

目标等级：scaffold

触发问题：

- 研究页 metadata 已显示当前 K 线窗口缺口数量，也能修复首个缺口，但图表本体没有对应可视化提示。
- 实施计划仍保留“数据缺口提示”作为研究页图表逐步增强项。

修复范围：

- `useResearchWorkspace` 将 CandleProvider 返回的 `candleResult.gaps` 转成 `TradingViewChart` markers。
- 中间缺口锚定到缺口后的第一根可见 K 线；窗口头尾边界缺口锚定到最近可见 K 线，避免切换上一 / 下一窗口时缺口提示丢失。
- 研究页将 `chartMarkers` 显式传给 `TradingViewChart`，并新增中英文 marker 短文案。
- 静态布局测试增加 markers 传递契约，避免后续只保留计算但漏接图表组件。

验证：

- `pnpm --dir web/frontend exec vitest run src/composables/useResearchWorkspace.test.ts src/pages/ResearchPage.layout.test.ts` 通过，2 个测试文件 / 26 个测试。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过，21 个测试文件 / 108 个测试。
- `scripts/quality-gate.sh` 通过。
- `pnpm --dir web/frontend run build` 通过，入口 chunk `449.18 kB`。
- `docker compose up -d --build api` 通过并重建本地 8080。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SAMPLES=20 SMOKE_INTERVAL_MS=100 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：desktop `body/chart/tv 603->603`，narrow desktop `500->500`，mobile `457->457`。
- `git diff --check` 通过。

失败：

- 首次 `scripts/quality-gate.sh` 失败：`useResearchWorkspace.ts` 被新增 helper 推到 463 行，超过 400 行硬限制；已拆出 `researchChartGapMarkers.ts` 并将主 composable 降到 396 行。
- 拆分后首次 `pnpm --dir web/frontend run typecheck` 失败：`ChartCandle` 类型 import 漏保留；已补回 type-only import，重跑通过。

剩余风险：

- 本轮只标记 CandleProvider 当前窗口返回的缺口，不新增跨窗口批量修复、图表 tooltip 详情、缺口区间背景带或完整视觉快照回归。
- 研究页和项目整体仍是 `scaffold`，不能升级。

### 阶段 1 研究页时间范围切换补充

目标等级：scaffold

触发问题：

- 实施计划仍保留“时间范围切换”作为研究页图表逐步增强项。
- 研究页此前只能通过上一 / 下一窗口翻页或手写 URL `from/to`，缺少明确的当前数据窗口 preset 入口。

修复范围：

- `ResearchWindowControls` 增加最新 / 1H / 6H / 1D 时间范围按钮组。
- `useResearchWorkspace` 新增 `applyTimeRange`，将 preset 转换成 `/api/candles` 的 `from/to` 查询参数；选择“最新”会清空 `cursor/from/to` 并回到默认最新窗口查询。
- 上一 / 下一窗口仍保留 opaque cursor 优先语义；时间范围 preset 会显式清掉 cursor，避免 cursor 和 `from/to` 混用。
- 新增中英文时间范围文案，并补静态布局契约与 workspace 查询参数测试。

验证：

- `pnpm --dir web/frontend exec vitest run src/composables/useResearchWorkspace.test.ts src/pages/ResearchPage.layout.test.ts` 通过，2 个测试文件 / 28 个测试。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过，21 个测试文件 / 110 个测试。
- `pnpm --dir web/frontend run build` 通过，入口 chunk `449.44 kB`。
- `scripts/quality-gate.sh` 通过。
- `docker compose up -d --build api` 通过并重建本地 8080。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SAMPLES=20 SMOKE_INTERVAL_MS=100 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：desktop `body/chart/tv 603->603`，narrow desktop `500->500`，mobile `457->457`。
- `git diff --check` 通过。

失败：

- 本轮未出现失败检查；前置行数检查发现 `useResearchWorkspace.ts` 一度达到 401 行，提交前已整理回 400 行，质量门禁通过。

剩余风险：

- 本轮只提供固定 preset，不包含日期选择器、自定义时间范围、根据不同 interval 动态调整 preset、图表拖拽选区或完整视觉快照回归。
- 研究页和项目整体仍是 `scaffold`，不能升级。

### 阶段 1 全历史 K 线返回缺口批量补同步入口

目标等级：scaffold

触发问题：

- 全历史 K 线缺口详情弹窗此前只能对单个返回缺口排补同步任务。
- 用户需要从研究页把当前扫描返回的多个真实相邻缺口一次性排入 `data_sync_tasks`，但不能绕过后端缺口复核或创建重复窗口任务。

修复范围：

- 新增 `POST /api/market/candle-gaps/repair-batch`，请求包含 exchange / symbol / interval 和当前扫描返回的缺口窗口数组。
- PostgreSQL store 在同一事务内逐个复核每个请求窗口仍是已落库相邻 K 线缺口；任一窗口不是缺口则返回 not found，不创建半截结果。
- 对已存在的同交易所 / 交易对 / 周期 / startTime / endTime 补同步任务跳过并计入 `skippedExisting`，只创建缺失窗口的无源补同步任务。
- OpenAPI contract、生成 TypeScript DTO、前端 API wrapper 和 schema drift 测试已更新。
- 研究页全历史缺口详情弹窗新增“修复当前 N 个”入口，成功后刷新扫描结果和数据同步任务列表。

验证：

- `scripts/generate-api-types.sh` 通过。
- `go test ./internal/web/api -run 'TestMarketCandleGap|TestFrontendAPI|TestAPIContract' -count=1` 通过。
- `go test ./internal/store/postgres -run 'TestIntegrationRepairMarketCandleGap|TestIntegrationRepairMarketCandleGaps|TestIntegrationScanMarketCandleGaps' -count=1` 通过。
- `pnpm --dir web/frontend exec vitest run src/components/research/MarketCandleGapTag.test.ts src/services/api/data.test.ts` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

失败：

- 本轮该批量补同步入口未出现失败检查。

剩余风险：

- 本轮只支持修复当前扫描 API 返回的有界缺口集合，不做自动后台修复、不做无限批量、不直接修改或删除 K 线事实数据。
- 研究页和项目整体仍是 `scaffold`，不能升级。

### 阶段 1 研究页 K 线可见绘制与裁切修复

目标等级：scaffold

触发问题：

- 用户在本地 `127.0.0.1:8080/research` 继续观察到 K 线图表容器裁切，窄视口下容易只剩网格、价格轴或半截价格标签。
- 既有 smoke 只证明图表高度稳定和 canvas 不越界，没有证明主绘图区实际有可见 K 线/成交量绘制。

修复范围：

- `TradingViewChart` 的右侧价格轴宽度改为按渲染宽度响应式选择 96 / 112 / 128px，避免窄视口被固定 156px 轴区挤压主图。
- 价格标签按价格量级格式化，BTC 这类大价格不再强制显示 `.00`，降低最新价标签被边界裁掉的风险。
- 默认初始可视范围不再把 1000 根 K 线全部硬塞进首屏，而是按主绘图区宽度显示可读数量的最新 K 线；用户仍可通过图表交互查看已加载窗口内其它 K 线。
- resize 后会重新应用响应式图表 options 并按新宽度重算初始可视范围。
- `scripts/research-chart-height-smoke.mjs` 新增主绘图区红/绿市场像素检查，要求可见绘制不是只有网格或一条价格线；`scripts/check-research-chart-layout.sh` 将该运行态断言纳入静态保留契约。

验证：

- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts src/pages/ResearchPage.layout.test.ts` 通过，2 个测试文件 / 30 个测试。
- `node --check scripts/research-chart-height-smoke.mjs` 通过。
- `bash -n scripts/check-research-chart-layout.sh` 通过。
- `scripts/check-research-chart-layout.sh` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `docker compose up -d --build api` 通过并重启本地 8080。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SAMPLES=20 SMOKE_INTERVAL_MS=100 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：desktop、812x1320 窄桌面、mobile 三组高度稳定，并通过主图可见市场像素断言。
- Headless Chrome 812x1320 截图确认：固定图表体 500px，主图 canvas 666px，右轴 canvas 112px，价格标签完整，K 线和成交量可见。

失败：

- 本轮该图表修复未出现失败检查。

剩余风险：

- 本轮关闭的是研究页 K 线首屏可见绘制、价格轴裁切和 smoke 漏检，不是完整像素快照回归体系。
- 图表仍缺指标层、绘图工具、完整缩放/拖拽 UX、全主题/全语言视觉基线和长期真实浏览器矩阵采样；研究页和项目整体仍是 `scaffold`，不能升级。

### 阶段 1 instrument catalog inactive 状态可见语义补充

目标等级：scaffold

触发问题：

- instrument catalog 已能从交易所真实元数据写入 active / inactive 状态，但搜索 API 默认只返回 active。
- 研究页、回测创建和交易创建提交前只能把 exact active 未命中统一视为“目录不存在”，无法区分“交易对存在但当前不是 active”。
- 这会让用户在退市、暂停或交易所状态变化时得到误导性错误，也让阶段 1 研究核心的 catalog 边界不清。

修复范围：

- `GET /api/market/instruments` 新增 `status=active|inactive|all` 查询参数，默认保持 `active`，非法值返回 400。
- PostgreSQL instrument catalog 查询支持 active / inactive / all 过滤，`all` 结果仍优先展示 active。
- OpenAPI contract、API handler 测试和 PostgreSQL 集成测试覆盖 status 查询语义。
- 前端 market API wrapper 支持 status 参数；研究页数据同步创建、回测创建和交易创建提交前统一执行 exact `status=all` catalog 校验。
- 前端可区分 active、inactive、missing：active 继续提交，inactive 显示明确不可用提示，missing 保持目录未命中提示，catalog 查询失败保持校验失败提示。

验证：

- `go test ./internal/web/api -run 'TestMarketInstrument|TestAPIContract|TestFrontendAPI' -count=1` 通过。
- `go test ./internal/store/postgres -run 'TestIntegrationListMarketInstruments|TestIntegrationGetActiveMarketInstrument|TestIntegrationReplaceMarketInstruments' -count=1` 通过。
- `pnpm --dir web/frontend exec vitest run src/services/api/market.test.ts src/composables/useResearchWorkspace.test.ts src/composables/useResearchWorkspace.instrumentCatalog.test.ts src/composables/useStrategyTaskForm.test.ts` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

失败：

- 本轮该 catalog 状态语义补充未出现失败检查。

剩余风险：

- 本轮不自动停用、迁移或修复已经存在的同步 / 回测 / 交易任务；既有任务遇到退市或暂停仍依赖后续状态机和运维语义补强。
- 本轮只保留 active / inactive 两档内部状态，没有完整建模各交易所的业务状态、暂停原因、只撤单、只减仓、清算或迁移窗口。
- 研究页和项目整体仍是 `scaffold`，不能升级。

### 阶段 1 数据同步既有任务 marketStatus 可见与启动阻断补充

目标等级：scaffold

触发问题：

- instrument catalog 已能区分 active / inactive / missing，但既有数据同步任务列表此前不暴露该状态。
- 已经创建的任务如果对应交易对后来变为 inactive 或从 catalog 消失，用户仍可能从研究页继续点击 sync / realtime / retry，worker 也可能继续 claim。
- 这会让退市/停牌后的任务操作语义继续不清，和阶段 1 研究核心的“数据源健康可观察”目标冲突。

修复范围：

- `DataSyncTask` 新增 `marketStatus=active|inactive|missing`，由 PostgreSQL 基于 `market_instruments` 派生，不新增 migration。
- `GET /api/data/tasks`、data sync task command 返回、repair 返回任务均携带 `marketStatus`；OpenAPI contract 和前端 generated types 已更新。
- `SetSyncEnabled`、`SetRealtimeEnabled` 和 failed retry 在启用任务时要求 exact active catalog 命中；不命中返回 `market_instrument_not_active`。
- `ClaimDataSyncTask` 增加 active catalog 条件，`hi sync` 不再领取 inactive / missing market 的同步任务。
- 研究页任务表新增市场状态列；非 active 任务禁用启动 sync / realtime，并在 workspace action 层给出明确错误提示。

验证：

- `scripts/generate-api-types.sh` 通过。
- `go test ./internal/data ./internal/web/api -run 'TestDataSync|TestFrontendAPI|TestAPIContract|TestWriteGeneratedFrontendAPITypes' -count=1` 通过。
- `go test ./internal/store/postgres -run 'TestDataSyncTaskScanColumnsPlaceMarketStatusBeforeHealth|TestIntegrationListDataSyncTasksReportsMarketStatus|TestIntegrationDataSyncCommandsRequireActiveMarketInstrument|TestIntegrationClaimDataSyncTaskSkipsInactiveMarketInstrument|TestIntegrationDataSyncRetryReleasesAndReclaimsTask|TestIntegrationRetryDataSyncTask' -count=1` 通过。
- `pnpm --dir web/frontend exec vitest run src/components/tables/DataSyncTaskTable.test.ts src/composables/useResearchWorkspace.test.ts src/composables/useResearchWorkspace.instrumentCatalog.test.ts src/services/api/data.test.ts` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- `docker compose up -d --build api sync` 后，`api` healthy、`sync` stable，`/readyz` 返回 ok，`/research` 返回 200。

失败：

- 首次 `scripts/quality-gate.sh` 失败：`useResearchWorkspace.test.ts`、`internal/web/api/fake_repository_test.go`、`internal/store/postgres/integration_test.go` 超过硬行数；已拆出 `useResearchWorkspace.instrumentCatalog.test.ts` 补充用例、`fake_repository_data_sync_commands_test.go` 和 `integration_data_sync_market_status_test.go`，复验通过。
- 本地重建后首次 `sync` 容器启动即重启：`dataSyncTaskScanColumns` 把 `marketStatus` 表达式放进原 data health alias 位置，生成 `AS market_status AS data_health`；已修正列顺序并新增 `TestDataSyncTaskScanColumnsPlaceMarketStatusBeforeHealth`，重建后 worker 稳定。

剩余风险：

- 该小节当时只做到“可见 + 阻止继续启动 + worker 不领取”；后续已补充 instrument catalog 同步后的 data sync task 自动停用，但仍不自动迁移、删除或修复历史任务。
- 本轮仍没有完整建模交易所退市、停牌、只撤单、只减仓或迁移窗口等业务状态；`inactive` 仍是粗粒度内部状态。
- 研究页和项目整体仍是 `scaffold`，不能升级。

### 阶段 1 研究页 K 线图表边缘裁切补充

目标等级：scaffold

触发问题：

- 用户在本地 `127.0.0.1:8080/research` 继续观察到 K 线图表容器内容被截掉。
- 运行态复查确认 fixed viewport、主图 canvas、右侧价格轴和底部时间轴均没有横向溢出，但时间轴文字仍可能贴近 canvas 边缘，形成视觉裁切。

修复范围：

- `TradingViewChart` 将时间轴逻辑边缘留白提高到最少 12 根、48-96px、12% 绘图区宽度，避免首尾时间标签贴边。
- 右侧价格轴响应式最小宽度从 96/112/128px 提高到 104/128/144px，给价格标签和最后价格标记留出更稳的绘制区域。
- `scripts/research-chart-height-smoke.mjs` 新增底部时间轴边缘深色文字像素检查，防止“canvas 没溢出但文字贴边被裁”的回归。
- `scripts/check-research-chart-layout.sh` 纳入该运行态断言文本，轻量门禁会检查 smoke 仍保留边缘裁切检测。

验证：

- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts src/pages/ResearchPage.layout.test.ts` 通过。
- `docker compose up -d --build api sync` 通过，本地 `http://127.0.0.1:8080/research` 返回 200。
- `node scripts/research-chart-height-smoke.mjs` 通过：desktop、812x1320 窄桌面、mobile 三组高度稳定，并通过主图可见像素、右侧价格轴、底部时间轴和时间轴边缘文字检查。

失败：

- 本轮先前 smoke 只覆盖 canvas 边界和高度稳定，未覆盖时间轴文字贴边；已补运行态检查。

剩余风险：

- 仍未建立人工视觉基线截图审批、全主题/全语言矩阵或长期浏览器 soak；研究页图表能力仍是阶段 1 scaffold 增量。
- 图表仍缺指标层、十字线增强、绘图工具、完整缩放/拖拽 UX 和自定义时间范围。

### 阶段 1 研究页 K 线图表边缘裁切二次修正

目标等级：scaffold

触发问题：

- 用户在本地继续观察到研究页 K 线固定容器内的内容被截掉，且前一版运行态 smoke 把时间轴边界线 / 刻度线也计入文字贴边像素，容易产生误判。

修复范围：

- `TradingViewChart` 为 lightweight-charts 时间轴增加短 UTC `tickMarkFormatter`，年 / 月 / 日 / 分钟刻度均控制在 8 字符以内。
- `TradingViewChart` 的初始可见逻辑范围增加半整数 K 线内边距，避免首个时间刻度坐标落在 canvas 物理 0 像素边界。
- `scripts/research-chart-height-smoke.mjs` 补齐所有布局节点的 `top/bottom` 几何采样，并把时间轴边缘深色像素检查改为阈值判断，允许轴线 / 刻度线但仍拦截明显文字贴边。

验证：

- `pnpm --dir web/frontend exec vitest run src/pages/ResearchPage.layout.test.ts src/components/chart/TradingViewChart.test.ts` 通过。
- `scripts/check-research-chart-layout.sh` 通过。
- `pnpm --dir web/frontend run build` 通过。
- 使用当前源码构建的本地预览服务 `BASE_URL=http://127.0.0.1:5174 node scripts/research-chart-height-smoke.mjs` 通过：desktop `body/chart/tv 603->603`，812x1320 窄桌面 `500->500`，mobile `457->457`，内部高度污染后无增长。

剩余风险：

- 本轮修复的是图表固定容器、轴标签边界和 smoke 误判，不是完整像素快照基线。
- 当前 8080 Docker API 容器尚未重建为本轮前端产物；本轮先用临时预览服务验证当前源码构建。
- 研究页和项目整体仍是 `scaffold`，不能升级。

### 阶段 1 instrument catalog 同步后自动停用非 active 数据同步任务补充

目标等级：scaffold

触发问题：

- 已有 `marketStatus=inactive/missing` 可见和启动阻断，但已经启用的既有数据同步任务仍可能保留 `sync_enabled` / `realtime_enabled` 期望状态。
- `hi sync` claim 虽然跳过非 active 任务，但用户在研究页仍会看到“期望继续同步/实时”的状态残留，不利于判断退市、停牌或交易对迁移后的处置边界。

修复范围：

- `ReplaceMarketInstruments` 在同一 PostgreSQL 事务中完成 instrument upsert / stale active 标记 inactive 后，自动停用当前 exchange 下不再命中 active catalog 的数据同步任务。
- 自动停用只影响 `pending/running/paused` 且 `sync_enabled` 或 `realtime_enabled` 为 true 的 data sync task：设置 `sync_enabled=false`、`realtime_enabled=false`、`status=paused`，并清理 `locked_by/locked_until/heartbeat_at`。
- `MarketInstrumentSyncResult` 新增 `pausedDataSyncTaskCount`，API contract、fake repository、前端类型和测试样本同步更新。
- `hi sync` instrument catalog 同步日志输出 `paused_data_sync_tasks`。

验证：

- `go test ./internal/store/postgres ./internal/web/api ./internal/marketsync -run 'TestIntegrationReplaceMarketInstruments|TestMarketInstrument|TestMarketInstrumentSync|TestRunner' -count=1` 通过。
- `pnpm --dir web/frontend exec vitest run src/services/api/market.test.ts src/components/market/MarketSymbolAutoComplete.test.ts` 通过。

失败：

- 本轮定向检查未出现失败。

剩余风险：

- 本轮只自动停用数据同步任务，不自动删除任务、不删除 K 线、不为退市/迁移生成修复任务，也不处理回测 / 交易任务。
- `inactive` 仍是粗粒度内部状态，没有区分停牌、退市、只撤单、只减仓、迁移窗口等交易所业务语义。
- 研究页和项目整体仍是 `scaffold`，不能升级。

### 阶段 1 instrument catalog 交易所原始状态可观察补充

目标等级：scaffold

触发问题：

- 研究页 data sync task 只能看到 `active/inactive/missing`，无法区分 Binance `BREAK`、OKX `suspend`、catalog 中未返回等具体来源状态。
- `market_instruments` 只保存内部归一化状态，后续排查交易所停牌 / 退市 / catalog 漂移时缺少原始状态证据。

修复范围：

- `market_instruments` 新增 `exchange_status`，迁移会为既有行填充非空状态，后续 catalog upsert 保留 Binance / OKX 返回的原始 instrument 状态。
- catalog 同步时，新返回的 inactive 交易对保留原始状态；本次 catalog 中不再返回的既有 active 交易对标记为 `exchange_status='not_returned'`。
- `/api/market/instruments` 返回 `exchangeStatus`；`/api/data/tasks` 返回 `marketStatusDetail`，由后端从 `market_instruments.exchange_status` 派生，缺 catalog 时为 `missing`。
- 研究页任务表市场状态列展示非 active 细节，例如 `Inactive · BREAK`，让用户能直接分辨粗状态背后的交易所原因。

验证：

- `go test ./internal/adapter/binance ./internal/adapter/okx ./internal/store/postgres ./internal/web/api -run 'TestFetchInstruments|TestDataSyncTaskScanColumns|TestIntegrationListMarketInstruments|TestIntegrationGetActiveMarketInstrument|TestIntegrationReplaceMarketInstruments|TestIntegrationListDataSyncTasksReportsMarketStatus|TestAPIContract|TestFrontendAPI|TestWriteGeneratedFrontendAPITypes' -count=1` 通过。
- `pnpm --dir web/frontend exec vitest run src/services/api/market.test.ts src/services/api/data.test.ts src/components/tables/DataSyncTaskTable.test.ts` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。

失败：

- 本轮定向检查未出现失败。

剩余风险：

- 本轮只保存和展示交易所原始 instrument 状态，不建立停牌 / 退市 / 只撤单 / 迁移窗口的完整状态机。
- 不自动删除 K 线、不迁移任务、不处理回测 / 交易任务的历史语义。
- 研究页和项目整体仍是 `scaffold`，不能升级。

### 阶段 1 研究页图表容器横向溢出与短视口补充

目标等级：scaffold

触发问题：

- 用户在本地研究页继续观察到 K 线图表固定容器内容被截掉，且此前截图显示 data sync 表格横向内容可能把页面整体撑宽，导致只看到表格或图表右侧区域。
- 任务表最近错误列虽然已有摘要和 tooltip，但表格外层缺少足够的 grid 子项宽度边界，长表格仍可能影响页面级横向滚动。

修复范围：

- 研究页工作区、任务面板和任务表根节点补充 `width/max-width/min-width` 边界，把横向滚动限制在任务表自身，不让 2210px 表格宽度反向撑大页面。
- data sync 任务表长文本单元格统一 `width:100%`、`min-width:0`、单行省略，避免最近错误、缺口摘要和同步窗口在窄列里失控换行。
- 研究页图表固定视口的短视口最小高度下调，避免浏览器高度不足时图表轴线更容易落到可见区域外。
- `research-chart-height-smoke.mjs` 新增 document/body 横向溢出和 `scrollX` 检查，并修复 Chrome profile 清理偶发 `ENOTEMPTY` 导致的误失败。

验证：

- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- 使用当前源码重建的本地 Docker API `http://127.0.0.1:8080/research` 通过 `scripts/research-chart-height-smoke.mjs`：desktop、812x1320 窄桌面、mobile 三组高度稳定，主图 canvas、右侧价格轴、底部时间轴和页面横向溢出检查均通过。

剩余风险：

- 本轮只修复研究页容器、表格横向边界和运行态 smoke 漏检，不代表完整图表交互达到 production-safe。
- 仍未建立人工像素快照基线、全语言/全主题视觉矩阵、真实浏览器长时间 soak 和完整图表工具能力。
- 研究页和项目整体仍是 `scaffold`，不能升级。

### 阶段 1 研究页 K 线图表安全边距与内部裁剪补充

目标等级：scaffold

触发问题：

- 用户在本地研究页继续观察到 K 线图表容器内容贴边和被截断，尤其是右侧价格轴、底部时间轴在窄桌面或浏览器缩放下缺少安全空间。
- 既有修复为了防止 lightweight-charts 内部节点高度污染，把 chart root、canvas host 和 `.tv-lightweight-charts` 都设置为 `contain: strict` / `overflow: clip`，这会把第三方图表内部标签也变成潜在裁剪对象。

修复范围：

- `.research-chart-body` 保持固定高度和外层裁剪，但新增 16px 右侧安全边距、12px 底部安全边距。
- `TradingViewChart` 从固定 chart slot 读取 CSS 安全边距，传给 lightweight-charts 的实际 render width / height 会扣除边距，避免价格轴和时间轴贴到裁剪边。
- chart root、canvas host 和 `.tv-lightweight-charts` 不再使用 paint containment；overflow 改为 visible，并用 `!important` 覆盖 lightweight-charts 根节点 inline overflow，外层固定槽仍负责阻断异常溢出。
- `research-chart-height-smoke.mjs` 改为验证运行态 chart / tv / 右价轴 / 底部时间轴 inset 必须匹配 CSS 配置，而不是要求图表贴满固定槽。
- 抽出 `chartSizing.ts` 放置 DOM 尺寸读取工具，避免主 chart 组件超过行数硬限制。

验证：

- `scripts/check-research-chart-layout.sh` 通过。
- `node --check scripts/research-chart-height-smoke.mjs` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- 使用当前源码重建本地 Docker API 后，`BASE_URL=http://127.0.0.1:8080 SMOKE_SAMPLES=20 SMOKE_INTERVAL_MS=100 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：desktop `body 603, chart/tv 591`，812x1320 `body 500, chart/tv 488`，mobile `body 457, chart/tv 445`。

剩余风险：

- 本轮只修复研究页图表容器裁剪和运行态回归检查，不补齐完整图表工具、像素快照基线或全浏览器矩阵。
- 任务表在极窄视口下仍依赖自身横向滚动和 sticky 操作列，未重做成响应式列管理。
- 研究页和项目整体仍是 `scaffold`，不能升级。

### 阶段 1 数据同步交易所退避成功恢复清理补充

目标等级：scaffold

触发问题：

- 临时交易所错误会写入 `data_sync_exchange_backoffs`，claim 会跳过 active 冷却交易所，但成功同步后此前只清理任务级 `last_error/next_attempt_at`。
- 过期 exchange backoff 虽不会继续阻断 claim，却会在数据库里残留，恢复闭环不清晰，也不利于后续健康统计和运维排查。

修复范围：

- `SaveDataSyncResult` 在同一 PostgreSQL 事务中成功 upsert K 线并更新任务结果后，删除该任务交易所下 `next_attempt_at <= now()` 的过期 exchange backoff。
- 清理条件只覆盖已过期 backoff；未来时间的 exchange backoff 保留，避免并发任务刚记录的新冷却被成功任务误删。
- runner 恢复集成测试插入过期 exchange backoff，验证成功恢复后数据库行被清理。
- store 直接保存结果的集成测试插入未来 exchange backoff，验证成功保存不会误删未来冷却。

验证：

- `go test ./internal/store/postgres -run 'TestIntegrationDataSyncRunnerResumesRealtimeTaskFromExpiredLease|TestIntegrationSaveDataSyncResultKeepsFutureExchangeBackoff' -count=1` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
剩余风险：

- 本轮只补 exchange backoff 成功恢复清理，不证明真实交易所网络长期恢复、分布式限流或完整 data sync 状态机。
- 不处理历史失败错误迁移、不删除 K 线、不自动批量修复缺口。
- 研究页和项目整体仍是 `scaffold`，不能升级。

### 阶段 1 数据同步错误 API 出口脱敏补充

目标等级：scaffold

触发问题：

- data sync adapter 已能生成不含 path/query 的 endpoint 摘要，前端也会兜底清洗外部 URL，但历史库里已有的 `Get "https://.../klines?...": EOF` 形态仍可能在 API 出口被简化为 `Get "host": EOF`，对用户仍是开发者噪声。
- 外部交易所错误不能依赖单一前端展示层清洗；API 返回给浏览器前应已经剥离完整 URL、path、query 和潜在参数。

修复范围：

- `/api/data/tasks`、data sync command 和 gap repair 返回的 `lastError` / `exchangeBackoffLastError` 在 API 出口识别 `Get "https://...": reason` 这类 transport error，并压缩为 `host: reason`。
- 前端 `sanitizeExternalError` 保留同样兜底规则，即使后端或测试 fixture 传入历史完整 URL，也只展示 `host: reason`。
- 后端 API 测试和前端 API 归一化测试收紧断言，禁止回退到完整 URL、query 参数或 `Get "host"` 形态。

验证：

- `go test ./internal/web/api -run 'TestDataSyncTaskRoutesSanitizeLastError|TestDataSyncTaskRoutes' -count=1` 通过。
- `pnpm --dir web/frontend exec vitest run src/services/api/data.test.ts src/components/tables/DataSyncTaskTable.test.ts` 通过。

剩余风险：

- 本轮只加强 data sync 错误 API 出口和前端兜底展示，不处理真实交易所恢复压测、分布式限流或完整错误码分类。
- 已入库的历史 `last_error` 文本仍不做迁移清洗；清洗发生在 API/前端出口。
- 研究页和项目整体仍是 `scaffold`，不能升级。

### 阶段 1 研究页图表容器和任务表窄屏遮挡修复

目标等级：scaffold

触发问题：

- 研究页 K 线图表组件在前几轮修复中叠加了内部高度快照、右/底 gutter 扣减和内联 `!important` 锁宽高，真实浏览器窄窗口下仍容易出现容器裁切、尺寸判断复杂且不可维护。
- 数据同步任务表的操作列固定在右侧，窄窗口下覆盖同步窗口、最新同步时间等中间列，刷新本地页面后仍会给用户造成“列表被盖住”的质量问题。

修复范围：

- `TradingViewChart` 改为只测量外部 `data-chart-viewport="fixed"` 宿主，图表 root / canvas 由 CSS 填满父容器，不再写入 `--tt-chart-render-*` 和内联 `!important` 宽高。
- 图表高度读取优先使用宿主 CSS height / max-height，再回退到 client / bounds，保留对内部节点高度污染的防线，但不再冻结真实宿主 resize。
- 研究页图表 body 去掉 JS gutter 扣减变量，保留固定 viewport 高度、`overflow: hidden` 和 `contain: layout paint`。
- 数据同步任务表移除 `actions` 列固定右侧，保留横向滚动和操作列宽度，避免窄窗口遮挡中间列。

验证：

- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts src/pages/ResearchPage.layout.test.ts` 通过。
- `pnpm --dir web/frontend run test` 通过，22 个前端测试文件、118 条测试通过。
- `pnpm --dir web/frontend run build` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- `docker compose up -d --build api` 通过，`tictick-hi-api-1` 健康。
- Headless Chrome 811x1320 登录 `/research` 复验：`chartBody` / `.trading-chart` / `.trading-chart__canvas` 均为 `777x500`，`.trading-chart` 无内联 style，固定右侧表格单元数量为 `0`，无页面横向溢出。

剩余风险：

- 本轮只修研究页图表容器和任务表遮挡，不补完整交互设计、任务操作语义、数据同步状态机或真实交易所稳定性。
- 表格仍是横向滚动型工作区，不是最终生产级密度/列管理方案。
- 研究页和项目整体仍是 `scaffold`，不能升级。

### 阶段 1 研究页 K 线图表内部根节点裁剪与高度反馈修正

目标等级：scaffold

触发问题：

- 用户在本地 `127.0.0.1:8080/research` 继续观察到 K 线图表固定容器内容被截掉。
- 首轮去掉 `.tv-lightweight-charts` 外层 `max-* / overflow / contain` 后，运行态 smoke 复现内部根节点高度反馈：desktop 采样失败为 `tv=9000`，说明仅移除裁剪会重新打开无限拉高风险。

修复范围：

- `.tv-lightweight-charts` 外层不再设置 `max-width/max-inline-size`、`overflow:hidden` 或 `contain`，避免外部 CSS 干预 lightweight-charts 的价格轴、时间轴和内部 table 布局。
- 只对 `.tv-lightweight-charts` 外层增加纵向 `block-size/height/max-block-size/max-height: 100% !important`，用固定图表槽锁住第三方根节点高度，阻断内部 inline 高度污染继续撑高页面。
- `ResearchPage.layout.test.ts` 和 `scripts/check-research-chart-layout.sh` 更新为“外层固定、内部不裁剪、第三方根节点只锁高度”的契约。
- 本地 `api` 已通过 `docker compose up -d --build api` 使用本轮前端构建产物重建，`http://127.0.0.1:8080/research` 当前加载新资源。

验证：

- 首次 `node scripts/research-chart-height-smoke.mjs` 失败：`desktop-1440x900 tv height exceeded viewport cap`，`tv=9000`；已作为本轮复现证据。
- `scripts/check-research-chart-layout.sh` 通过。
- `pnpm --dir web/frontend exec vitest run src/pages/ResearchPage.layout.test.ts src/components/chart/TradingViewChart.test.ts` 通过，2 个测试文件 / 30 个测试。
- `pnpm --dir web/frontend run build` 通过。
- `docker compose up -d --build api` 通过，`/readyz` 返回 `{"status":"ok"}`。
- 修复后 `node scripts/research-chart-height-smoke.mjs` 在本地 8080 通过：desktop `body/chart/tv 603->603`，812x1320 窄桌面 `500->500`，mobile `457->457`；主图可见像素、右侧价格轴、底部时间轴、时间轴边缘和内部高度污染稳定性检查均通过。
- `pnpm --dir web/frontend run test` 通过，22 个测试文件 / 118 条测试。
- `scripts/quality-gate.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。

剩余风险：

- 本轮关闭的是研究页 K 线容器裁剪和 `.tv-lightweight-charts` 内部根节点高度反馈，不代表完整图表工具链达到 production-safe。
- 仍未建立人工像素快照基线、全主题/全语言视觉矩阵、长期浏览器 soak 或完整图表交互能力。
- 研究页和项目整体仍是 `scaffold`，不能升级。

### 阶段 1 数据同步任务单缺口修复真实性校验补充

目标等级：scaffold

触发问题：

- `POST /api/data/tasks/{id}/repair-gap` 原先只校验 `from/to` 格式和顺序，store 层会直接用前端传入窗口创建补同步任务。
- 这允许前端或调用方绕过后端缺口检测，为任意时间窗口排补同步任务，和“缺口修复必须走同一套同步任务逻辑、不能手工拼接补同步语义”的 Stage 1 要求不一致。

修复范围：

- `RepairDataSyncTaskGap` 在事务内锁定源任务后，使用和 `ListDataSyncTaskGaps` 相同的任务窗口 gap CTE 校验请求窗口。
- 只有 `gap_from/gap_to` 与后端当前检测出的真实缺口精确匹配时，才继续跳过重复任务或创建带 `repairSourceTaskId` 的补同步任务。
- 非真实缺口返回 `data.ErrNotFound`，API 层映射为 `404 not_found`，不会写入 `data_sync_tasks`。
- API fake repository 按 `taskGapDetails` / `GapSummary.FirstGap` 校验单缺口修复，避免路由测试继续接受伪造窗口。

验证：

- `go test ./internal/web/api -run TestDataSyncTaskRoutes -count=1` 通过。
- 本机 `go test ./internal/store/postgres -run TestIntegrationRepairDataSyncTaskGapCreatesSyncTask -count=1 -v` 因未设置 `TICTICK_TEST_DATABASE_URL` 跳过。
- Docker Compose PostgreSQL 集成测试通过：`docker run --rm --network tictick-hi_default -v "$PWD":/src -w /src -e TICTICK_TEST_DATABASE_URL='postgresql://tictick:tictick-local-postgres-password@postgres:5432/tictick_hi?sslmode=disable' golang:1.26-bookworm go test ./internal/store/postgres -run TestIntegrationRepairDataSyncTaskGapCreatesSyncTask -count=1 -v`。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过，22 个前端测试文件、118 条测试通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

剩余风险：

- 本轮只关闭单缺口修复的“任意窗口可创建”问题，不自动批量补齐全历史缺口。
- 真实交易所补数成功率、分布式限流和完整 data sync 状态机仍未证明。
- 研究页和项目整体仍是 `scaffold`，不能升级。

### 阶段 1 补同步任务结果驱动源任务健康收敛补证

目标等级：scaffold

触发问题：

- 单缺口 repair API 已能验证真实缺口并创建带 `repairSourceTaskId` 的补同步任务，但此前只证明“任务可创建”，没有证明补同步任务写入缺失 K 线后，源任务的 `dataHealth/gapSummary` 会通过同一套后端派生逻辑自然回落。
- 研究页数据健康依赖 `ListDataSyncTasks` 的动态 SQL，如果补同步结果只更新 repair task 而源任务派生查询没有被验证，用户仍可能看到源任务长期保持 `gap`。

修复范围：

- 新增 PostgreSQL 集成测试 `TestIntegrationRepairTaskExecutionConvergesSourceDataHealth`。
- 测试创建一个只有单个真实缺口的源任务，调用 `RepairDataSyncTaskGap` 创建补同步任务，并模拟该 repair task 进入 running 后通过 `SaveDataSyncResult` upsert 缺失的 `market_candles`。
- 测试随后重新调用 `ListDataSyncTasks`，断言 repair task 进入 `succeeded` 且 `syncEnabled=false`，源任务 `dataHealth=ok`、`gapSummary=nil`。
- 新测试拆到独立 `integration_data_sync_repair_convergence_test.go`，避免把既有 `integration_data_sync_health_test.go` 推过 700 行质量门禁。

验证：

- 本机 `go test ./internal/store/postgres -run 'TestIntegrationRepair(DataSyncTaskGapCreatesSyncTask|TaskExecutionConvergesSourceDataHealth)' -count=1 -v` 因未设置 `TICTICK_TEST_DATABASE_URL` 跳过，编译通过。
- Docker Compose PostgreSQL 集成测试通过：`docker run --rm --network tictick-hi_default -v "$PWD":/src -w /src -e TICTICK_TEST_DATABASE_URL='postgresql://tictick:tictick-local-postgres-password@postgres:5432/tictick_hi?sslmode=disable' golang:1.26-bookworm go test ./internal/store/postgres -run 'TestIntegrationRepair(DataSyncTaskGapCreatesSyncTask|TaskExecutionConvergesSourceDataHealth)' -count=1 -v`。
- `go test ./internal/datasync -count=1` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过，22 个前端测试文件、118 条测试通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 首次因 `integration_data_sync_health_test.go` 超过 700 行失败；测试拆分后通过。

剩余风险：

- 本轮证明的是 PostgreSQL store / `SaveDataSyncResult` / `ListDataSyncTasks` 的缺口健康收敛，不代表真实交易所补数一定成功。
- 本轮未做分布式调度隔离或真实交易所恢复压测。
- 研究页和项目整体仍是 `scaffold`，不能升级。

### 阶段 1 补同步任务 claim 顺序防饥饿补充

目标等级：scaffold

触发问题：

- `ClaimDataSyncTask` 原先按 `realtime_enabled DESC, created_at ASC` 领取任务。
- 长期 realtime 任务每次保存结果后会释放 lease 并保持 `running/realtime_enabled=true`，在共享开发库或单 worker 场景中可能反复排在 pending 补同步任务前面，导致缺口修复任务长期拿不到执行机会。

修复范围：

- 数据同步 claim 排序改为 pending 任务优先，其次 `sync_enabled` 历史/补同步任务，再到 realtime 轮询任务，同级保持 `created_at ASC`。
- 不改变 active market、exchange backoff、next attempt、lease 过期和状态候选条件。
- 新增 PostgreSQL 集成测试 `TestIntegrationClaimDataSyncTaskPrioritizesPendingRepairOverRealtimePoll`：构造一个更早的 running realtime 任务和一个 pending repair 任务，断言 claim 领取 repair，realtime 任务保持未领取。

验证：

- 本机 `go test ./internal/store/postgres -run TestIntegrationClaimDataSyncTaskPrioritizesPendingRepairOverRealtimePoll -count=1 -v` 因未设置 `TICTICK_TEST_DATABASE_URL` 跳过，编译通过。
- Docker Compose PostgreSQL 集成测试通过：`docker run --rm --network tictick-hi_default -v "$PWD":/src -w /src -e TICTICK_TEST_DATABASE_URL='postgresql://tictick:tictick-local-postgres-password@postgres:5432/tictick_hi?sslmode=disable' golang:1.26-bookworm go test ./internal/store/postgres -run 'TestIntegrationClaimDataSyncTask(PrioritizesPendingRepairOverRealtimePoll|SkipsInactiveMarketInstrument)' -count=1 -v`。
- `go test ./internal/datasync -count=1` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过，22 个前端测试文件、118 条测试通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。

剩余风险：

- 本轮只修单 worker claim 顺序，未实现跨多实例的全局公平调度或交易所级共享速率预算。
- 真实交易所补数成功率、完整 data sync 状态机和长期恢复压测仍未证明。
- 研究页和项目整体仍是 `scaffold`，不能升级。

### 阶段 1 数据同步保存前 K 线 payload 校验补充

目标等级：scaffold

触发问题：

- `datasync.Runner` 之前会把 exchange adapter 返回的 `data.Candle` 直接交给 `SaveDataSyncResult`。
- 如果 fetched candle 的 OHLCV 不是 decimal、为负数、OHLC 高低价边界不成立，或 open/close time 不符合周期，错误会延迟到 PostgreSQL numeric cast / CHECK constraint 才暴露。
- 这会把数据质量问题伪装成底层写库错误，不利于研究页识别真实失败原因，也不利于阶段 1 数据同步可信度推进。

Definition of Done：

- data sync worker 保存 K 线前必须校验 fetched candle series。
- 校验必须覆盖 open/close time 周期对齐、排序、重复 open_time、OHLCV decimal、非负值和 OHLC bounds。
- 异常 payload 不能写入 `market_candles`，不能推进 `last_synced_open_time`。
- 异常 payload 不作为 temporary exchange error 重试，任务进入 failed，保留明确 validation error。
- 不引入 migration，不改变已落库 candle schema。

修复范围：

- `internal/data.ValidateCandleSeries` 导出并扩展 OHLCV decimal / 非负 / OHLC bounds 校验。
- `datasync.Runner` 在计算 cursor 和调用 `SaveDataSyncResult` 前执行 `ValidateCandleSeries`。
- `TestRunnerRejectsInvalidFetchedCandleBeforeSaving` 覆盖 invalid fetched candle 不会调用保存结果、不进入 temporary retry、会标记 failed。
- 新增 runner validation 测试拆到 `runner_validation_test.go`，避免把既有 `runner_test.go` 推过 700 行质量门禁。
- `TestIntegrationDataSyncRunnerResumesRealtimeTaskFromExpiredLease` 的 fixture 改为真实 numeric candle，避免测试数据绕过事实表约束。

验证：

- `go test ./internal/data ./internal/datasync -count=1` 通过。
- 本机 `go test ./internal/store/postgres -run 'TestIntegrationDataSyncRunnerResumesRealtimeTaskFromExpiredLease|TestIntegrationSaveDataSyncResultKeepsFutureExchangeBackoff|TestIntegrationClaimDataSyncTaskPrioritizesPendingRepairOverRealtimePoll' -count=1 -v` 因未设置 `TICTICK_TEST_DATABASE_URL` 跳过，编译通过。
- Docker Compose PostgreSQL 集成测试通过：`docker run --rm --network tictick-hi_default -v "$PWD":/src -w /src -e TICTICK_TEST_DATABASE_URL='postgresql://tictick:tictick-local-postgres-password@postgres:5432/tictick_hi?sslmode=disable' golang:1.26-bookworm go test ./internal/store/postgres -run 'TestIntegrationDataSyncRunnerResumesRealtimeTaskFromExpiredLease|TestIntegrationSaveDataSyncResultKeepsFutureExchangeBackoff|TestIntegrationClaimDataSyncTaskPrioritizesPendingRepairOverRealtimePoll' -count=1 -v`。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。

剩余风险：

- 本轮只增加保存前数据质量防线，不处理真实交易所长期网络抖动、交易所业务异常码、跨实例共享限流或完整状态机。
- 异常 payload 的修复策略仍是失败后人工 retry / 后续代码修正，不是自动隔离单根脏 K 线继续同步。
- 数据同步 worker 仍不能升级到 usable；研究页和项目整体仍是 `scaffold`。

### 阶段 1 数据同步结果目标绑定校验补充

目标等级：scaffold

触发问题：

- `SaveDataSyncResult` 原先按 `result.Candles` 自带的 exchange / symbol / interval 写入 `market_candles`。
- 如果 adapter 或测试 fake client 返回了错标的 candle，可能出现“BTCUSDT 任务推进游标，但 ETHUSDT K 线被写库”的跨标的污染风险。
- 上一节保存前 payload 校验覆盖了时间和 OHLCV 形态，但还没有证明 candle 目标必须绑定当前 data sync task。

Definition of Done：

- data sync worker 在保存前必须校验每根 fetched candle 的 exchange / symbol / interval 等于被领取任务的目标。
- PostgreSQL store 层 `SaveDataSyncResult` 也必须按 `task_id` 读取目标并拒绝不匹配 candle，防止绕过 runner 的直接写库路径。
- 错目标 payload 不能写入 `market_candles`，不能推进 `last_synced_open_time`，不能转换任务状态。
- 不引入 migration，不改变已落库 candle schema。

修复范围：

- 新增 `ValidateCandleSeriesForTarget`，先校验 exchange / symbol / interval，再复用既有 candle series 结构和值校验。
- `datasync.Runner` 改为调用 `ValidateCandleSeriesForTarget`。
- `SaveDataSyncResult` 在事务内读取 data sync task 目标，并在 upsert 前执行同一目标绑定校验。
- `TestRunnerRejectsMismatchedFetchedCandleTargetBeforeSaving` 覆盖 runner 层错 symbol 不保存、不 temporary retry、标记 failed。
- `TestIntegrationSaveDataSyncResultRejectsMismatchedCandleTarget` 覆盖直接调用 store 保存错 symbol 时不写入任何 candle、不推进游标、不转换任务状态。

验证：

- `go test ./internal/data ./internal/datasync -count=1` 通过。
- 本机 `go test ./internal/store/postgres -run 'TestIntegrationSaveDataSyncResultRejectsMismatchedCandleTarget|TestIntegrationSaveDataSyncResultKeepsFutureExchangeBackoff|TestIntegrationDataSyncRunnerResumesRealtimeTaskFromExpiredLease' -count=1 -v` 因未设置 `TICTICK_TEST_DATABASE_URL` 跳过，编译通过。
- Docker Compose PostgreSQL 集成测试通过：`docker run --rm --network tictick-hi_default -v "$PWD":/src -w /src -e TICTICK_TEST_DATABASE_URL='postgresql://tictick:tictick-local-postgres-password@postgres:5432/tictick_hi?sslmode=disable' golang:1.26-bookworm go test ./internal/store/postgres -run 'TestIntegrationSaveDataSyncResultRejectsMismatchedCandleTarget|TestIntegrationSaveDataSyncResultKeepsFutureExchangeBackoff|TestIntegrationDataSyncRunnerResumesRealtimeTaskFromExpiredLease' -count=1 -v`。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。

剩余风险：

- 本轮只关闭同步结果目标绑定，不证明真实交易所长期恢复、多实例共享限流或完整统一状态机。
- 错目标 payload 仍会让任务失败，需要后续人工 retry 或 adapter 修正；本轮不做自动跳过单根异常 K 线。
- 数据同步 worker 仍不能升级到 usable；研究页和项目整体仍是 `scaffold`。

### 阶段 1 数据同步空批次一次性任务终态补充

目标等级：scaffold

触发问题：

- 一次性有界 data sync 任务在交易所 adapter 返回空 K 线批次且没有 cursor 时，runner 之前会保存一个未完成结果。
- 该路径不会写入 K 线，也不会推进 `last_synced_open_time`，但任务仍可能保持可继续运行状态，导致研究页长期显示同一个无进展同步任务。
- 直接把空数据伪装成成功 K 线不可接受；正确行为是让任务进入可观察终态，同时通过后端派生健康继续显示任务窗口存在缺口。

Definition of Done：

- 非 realtime、带 `end_time` 的一次性任务，如果本轮 fetch 返回空批次且没有 cursor，必须保存 completed 结果并停止本轮任务循环。
- 该结果不能写入假 K 线，不能推进 `last_synced_open_time`。
- 任务列表必须能看到终态，并继续通过 `dataHealth/gapSummary` 表示窗口缺口或数据不足。
- active catalog 下的 succeeded 一次性任务必须可重新启动为 pending，便于用户在外部数据恢复后重新同步。
- failed 任务仍不能绕过 retry 直接 start sync。

修复范围：

- `datasync.Runner.isCompleted` 增加空批次输入，只对非 realtime、带结束时间、无 cursor 的空批次一次性任务返回 completed。
- `SetSyncEnabled` 启动任务时允许 `succeeded -> pending`，同时清空 `finished_at`；停止任务仍写入 `finished_at`。
- 新增 `0028_data_sync_restart_succeeded.sql`，让 PostgreSQL 状态流转 trigger 接受 `succeeded -> pending/running`，但不放开 failed 任务直接启动。
- 前端表格补充 succeeded 一次性任务点击“同步”的回归测试，确认 UI 不把该路径当成 failed retry。

验证：

- `go test ./internal/datasync -run 'TestRunner(CompletesBoundedOneShotTaskOnEmptyFetch|SyncsClaimedTask|DoesNotAdvanceCursor)' -count=1` 通过。
- `go test ./internal/web/api -run TestDataSyncTaskRoutes -count=1` 通过。
- `pnpm --dir web/frontend exec vitest run src/components/tables/DataSyncTaskTable.test.ts` 通过，1 个测试文件、11 条测试通过。
- Docker Compose PostgreSQL 集成测试通过：`docker run --rm --network tictick-hi_default -v "$PWD":/src -w /src -e TICTICK_TEST_DATABASE_URL='postgresql://tictick:tictick-local-postgres-password@postgres:5432/tictick_hi?sslmode=disable' golang:1.26-bookworm go test ./internal/store/postgres -run 'TestIntegrationEmptyCompletedDataSyncResultStopsOneShotLoop|TestIntegrationTaskCommandsRejectInvalidStatusTransitions' -count=1 -v`。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过，22 个前端测试文件、119 条测试通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

剩余风险：

- 本轮不证明真实 Binance / OKX 在空数据、延迟上市、限流和网络抖动下的长期恢复表现。
- 本轮不做自动补数、不删除任务、不迁移历史数据，也不改变 CandleProvider 缺口判定。
- data sync 仍缺完整统一状态机、分布式多实例限流和真实外部交易所恢复压测；研究页和项目整体仍是 `scaffold`。

### 阶段 1 数据同步任务软删除语义补充

目标等级：scaffold

触发问题：

- 实施计划明确“删除同步任务不等于删除 K 线数据”，但此前 `DeleteDataSyncTask` 直接硬删除 `data_sync_tasks` 行。
- 硬删除会丢掉任务操作上下文，也会让补同步来源关系只能依赖 FK `ON DELETE SET NULL` 被动断开，不利于研究页排障和后续审计。
- 用户在研究页点击删除时，也没有明确看到“不会删除已同步 K 线数据”的边界。

Definition of Done：

- 删除 data sync task 必须改为软删除：任务行保留，`deleted_at` 非空，状态进入 `cancelled`，`sync_enabled/realtime_enabled=false`，lease 清理。
- 软删除任务不能再出现在研究页任务列表，不能再被 `hi sync` claim，不能再接收 start/retry/repair/save result 等操作。
- 删除不能删除 `market_candles` 事实数据。
- 研究页删除确认文案必须明确删除的是同步任务记录，不删除已同步 K 线数据。
- 不实现恢复已删除任务，不实现 K 线数据删除，不改变 repair task 的数据修复语义。

修复范围：

- 新增 `0029_data_sync_soft_delete.sql`，为 `data_sync_tasks` 增加 `deleted_at`，并允许状态流转到 `cancelled`。
- `DeleteDataSyncTask` 改为 cancelled + `deleted_at` 软删除，并释放 sync/realtime/lease。
- `ListDataSyncTasks`、`ClaimDataSyncTask`、`SetSyncEnabled`、`RetryDataSyncTask`、`SaveDataSyncResult`、任务缺口查看和补同步查询都排除软删除任务。
- 补同步重复任务检查排除软删除任务，避免历史已删除 repair task 永久挡住同窗口重新修复。
- 研究页删除确认文案补充“已同步的 K 线数据不会被删除”。

验证：

- `go test ./internal/web/api -run 'TestDataSyncTask(DeleteRouteHidesTask|Routes)$' -count=1` 通过。
- `pnpm --dir web/frontend exec vitest run src/composables/useResearchWorkspace.test.ts src/composables/useResearchWorkspace.delete.test.ts` 通过，2 个测试文件、19 条测试通过。
- 本机 `go test ./internal/store/postgres -run TestIntegrationDeleteDataSyncTaskSoftDeletesAndKeepsCandles -count=1 -v` 因未设置 `TICTICK_TEST_DATABASE_URL` 跳过，编译通过。
- Docker Compose PostgreSQL 集成测试通过：`docker run --rm --network tictick-hi_default -v "$PWD":/src -w /src -e TICTICK_TEST_DATABASE_URL='postgresql://tictick:tictick-local-postgres-password@postgres:5432/tictick_hi?sslmode=disable' golang:1.26-bookworm go test ./internal/store/postgres -run TestIntegrationDeleteDataSyncTaskSoftDeletesAndKeepsCandles -count=1 -v`。
- `go test ./internal/store/postgres -count=1` 通过。
- `go test ./internal/web/api -count=1` 通过。
- `pnpm --dir web/frontend exec vitest run src/composables/useResearchWorkspace.test.ts src/composables/useResearchWorkspace.delete.test.ts src/services/api/data.test.ts` 通过，3 个测试文件、31 条测试通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过，23 个前端测试文件、120 条测试通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `scripts/stage8-migration-audit.sh` 通过。
- `git diff --check` 通过。

剩余风险：

- 本轮只定义 data sync task 删除语义，不提供行情数据删除、恢复已删除任务、删除审计事件或批量删除能力。
- 本轮没有做真实浏览器点击删除的端到端 smoke；前端行为由组合逻辑单测覆盖。
- data sync 仍缺完整统一状态机、分布式多实例限流和真实外部交易所恢复压测；研究页和项目整体仍是 `scaffold`。

### 阶段 1 研究页图表首屏容器补充

目标等级：scaffold

触发问题：

- 研究页虽然已把数据同步列表放在图表上方，但桌面 toolbar 左侧控件会被右侧状态区挤压换成多行，导致图表固定槽从首屏下半部开始。
- 2048x1152 桌面视口下，图表固定槽高度为 683px、底部时间轴落在首屏外，用户看到的是被窗口底部截掉的 K 线图。
- 既有研究页图表 smoke 只验证高度不会无限增长，没有把普通桌面和大屏桌面的“初始底轴必须在视口内”作为硬断言。

Definition of Done：

- 研究页保持“数据同步任务列表在上、K 线图表在下”的结构。
- 数据同步任务列表必须使用滚动上限，不允许表格内容把图表推到首屏外。
- 桌面 toolbar 必须有明确 flex 预算，不能让状态区把图表控件压成多行；移动端 column 布局不能继承桌面 flex-basis 造成面板虚高。
- K 线图表固定槽必须按剩余视口高度收敛，桌面、宽桌面、窄桌面下底部时间轴和右侧价格轴必须在首屏内。
- 研究页图表 smoke 必须覆盖 1440x900、2048x1152、812x1320 和 390x844，并继续污染内部 lightweight-charts DOM 高度验证不反向撑高。

修复范围：

- `ResearchPage.css` 将任务列表高度收敛为 `min(260px, 28dvh)`，保留表格内部滚动。
- 桌面 `.research-toolbar` 左右两侧改为明确 flex 预算，symbol 输入改为响应式宽度；移动端覆盖 toolbar row 为自然高度。
- 图表固定槽高度从大幅固定上限改为按剩余视口高度计算：桌面 `clamp(280px, calc(100dvh - 620px), 560px)`，窄桌面 `clamp(240px, calc(100dvh - 680px), 480px)`，移动 `clamp(260px, calc(100dvh - 520px), 420px)`。
- `research-chart-height-smoke.mjs` 增加 1440x900 和 2048x1152 首屏图表适配断言。
- `check-research-chart-layout.sh` 和 `ResearchPage.layout.test.ts` 更新为新的列表、toolbar 和图表高度契约。

验证：

- `pnpm --dir web/frontend run test -- TradingViewChart ResearchPage.layout` 通过，23 个测试文件、120 条测试通过。
- `scripts/check-research-chart-layout.sh` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `docker compose build api` 通过，Docker 镜像内前端 build 和 Go build 均通过。
- `docker compose up -d api` 后 `http://127.0.0.1:8080/research` 返回 200，API 容器 healthy。
- `scripts/research-chart-height-smoke.mjs` 通过：1440x900、2048x1152、812x1320、390x844 均稳定，桌面 document 高度不再超过视口。
- Headless Chrome 2048x1152 截图验证：document 高度 1152，图表面板底部 1126，chart/tv 底部 1113，底部时间轴和右侧价格轴可见。
- `pnpm --dir web/frontend run test` 通过，23 个测试文件、120 条测试通过。
- `scripts/quality-gate.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/stage8-visual-smoke.mjs` 通过。
- `git diff --check` 通过。

剩余风险：

- 本轮只关闭研究页图表首屏容器和高度稳定问题，不升级图表交互能力、盘口/指标研究能力或数据修复语义。
- 移动端仍需要滚动查看完整研究页上下文；本轮保证图表自身不再无限拉高和首屏桌面不裁切，不把移动端完整工作流升级为 usable。
- 研究页仍缺完整操作语义和真实交易所长期恢复压测；项目整体仍是 `scaffold`。

### 阶段 1 研究页 K 线固定槽填充修正

目标等级：scaffold

触发问题：

- 用户在本地 `127.0.0.1:8080/research` 继续观察到 K 线图表固定容器内容被截掉。
- 现有实现通过 `--tt-chart-inline-end-gutter` / `--tt-chart-block-end-gutter` 同时在研究页 CSS 和 `TradingViewChart` 尺寸读取中扣减右侧、底部空间；这种人为缩图会让真实图表槽、图表 root 和 lightweight-charts 渲染尺寸不一致，容易形成右侧价格轴、底部时间轴的裁切观感。

修复范围：

- 研究页固定图表槽仍由 `--research-chart-viewport-height` 控制高度，保持数据同步任务列表在上、K 线图表在下。
- 移除研究页固定槽的右/底 gutter 变量，`.research-chart-body .trading-chart` 改为 `width/height: 100%` 填满固定槽。
- `TradingViewChart` 尺寸读取不再读取或扣减 gutter，只使用外部 `data-chart-viewport="fixed"` 宿主的真实宽高。
- `TradingViewChart.css` 中图表 root 改为 `width/height: 100%`，避免父层和组件自身重复缩小绘图区。
- `ResearchPage.layout.test.ts` 和 `scripts/check-research-chart-layout.sh` 禁止研究页重新声明旧 gutter 变量，并断言图表 root 填满固定槽。

验证：

- `pnpm --dir web/frontend run test -- TradingViewChart ResearchPage.layout` 通过，23 个测试文件、120 条测试通过。
- `scripts/check-research-chart-layout.sh` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `pnpm --dir web/frontend run test` 通过，23 个测试文件、120 条测试通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- `docker compose build api` 通过，并已重启本地 `api` 容器。
- `http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`，`http://127.0.0.1:8080/research` 返回 200。
- 新 8080 构建下 `node scripts/research-chart-height-smoke.mjs http://127.0.0.1:8080/research` 通过：1440x900、2048x1152、812x1320、390x844 均稳定，且 `body/chart/tv` 等高。
- Headless Chrome 812x1320 截图复验：`.research-chart-body`、`.trading-chart`、`.trading-chart__canvas`、`.tv-lightweight-charts` 均为 `778x480`，右侧价格轴和底部时间轴 canvas 均在固定槽内。

剩余风险：

- 本轮修复的是研究页 K 线固定槽裁切/缩图问题，不补齐完整图表工具、指标层、全主题/全语言视觉矩阵或人工像素快照基线。
- 移动端仍需要滚动查看完整研究页上下文；研究页和项目整体仍是 `scaffold`，不能升级。

### 阶段 1 数据同步删除与临时重试竞态补充

目标等级：scaffold

触发问题：

- 研究页删除 data sync task 后，任务行会软删除并对前端隐藏，但已领取该任务的 worker 仍可能在删除后收到交易所临时错误。
- `RecordDataSyncRetry` 原先更新任务行时会过滤 `deleted_at IS NULL`，但随后读取 exchange 时没有过滤软删除任务。
- 该竞态可能让已删除任务写入 `data_sync_exchange_backoffs`，导致同交易所其它任务被全局退避阻塞，研究页也会看到与已删除任务相关的退避状态。

Definition of Done：

- 已软删除或不存在的 data sync task 不能再记录 retry 状态。
- 已软删除或不存在的 data sync task 不能再写入 exchange-level backoff。
- sync runner 遇到“任务已删除后才尝试记录 retry”的竞态时不能退出长运行 worker。
- 同交易所其它 active catalog 同步任务仍可被 claim。
- 不引入 migration，不改变删除恢复语义，不改变真实交易所限流策略。

修复范围：

- `RecordDataSyncRetry` 检查 retry update 的 `RowsAffected`，未命中时返回 `data.ErrNotFound`。
- `RecordDataSyncRetry` 读取 exchange 时同步过滤 `deleted_at IS NULL`，避免软删除任务进入 backoff 写入路径。
- `datasync.Runner` 在临时错误 retry 记录阶段遇到 `data.ErrNotFound` 时将其视为用户已删除任务的竞态并继续运行。
- `TestRunnerIgnoresRetryRecordForDeletedTask` 覆盖 runner 不因该竞态退出、不保存结果、不标记失败。
- `TestIntegrationDeletedDataSyncTaskRetryDoesNotCreateExchangeBackoff` 覆盖软删除任务 retry 返回 `ErrNotFound`、不写 `data_sync_exchange_backoffs`、同交易所兄弟任务仍可 claim。

验证：

- `go test ./internal/datasync -run 'TestRunner(RecordsTemporaryFetchErrorForRetry|IgnoresRetryRecordForDeletedTask)' -count=1` 通过。
- 本机 `go test ./internal/store/postgres -run TestIntegrationDeletedDataSyncTaskRetryDoesNotCreateExchangeBackoff -count=1 -v` 因未设置 `TICTICK_TEST_DATABASE_URL` 跳过，编译通过。
- Docker Compose PostgreSQL 集成测试通过：`docker run --rm --network tictick-hi_default -v "$PWD":/src -w /src -e TICTICK_TEST_DATABASE_URL='postgresql://tictick:tictick-local-postgres-password@postgres:5432/tictick_hi?sslmode=disable' golang:1.26-bookworm go test ./internal/store/postgres -run TestIntegrationDeletedDataSyncTaskRetryDoesNotCreateExchangeBackoff -count=1 -v`。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过，23 个测试文件、120 条测试通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

剩余风险：

- 本轮只关闭删除与 retry 记录之间的竞态，不改变已经存在的有效 exchange backoff 清理策略。
- 本轮不证明真实 Binance / OKX 长时间抖动、多实例共享限流或完整统一状态机。
- 数据同步 worker 仍不能升级到 usable；研究页和项目整体仍是 `scaffold`。

### 阶段 1 全历史缺口修复软删除与批量原子性补充

目标等级：scaffold

触发问题：

- 研究页全历史缺口修复会创建无源 `data_sync_tasks` 补同步任务；用户删除错误或失败的补同步任务后，同一真实缺口必须能重新排修复任务，不能被软删除历史行永久挡住。
- 批量修复当前返回的多个全历史缺口时，如果前面的窗口有效、后面的窗口不是已落库真实相邻缺口，不能留下半截创建的补同步任务。
- 既有代码从 SQL 看过滤了 `deleted_at IS NULL` 且批量修复在事务内执行，但缺少真实 PostgreSQL 集成测试证明。

Definition of Done：

- 软删除的全历史缺口补同步任务保留审计行且 `deleted_at` 非空。
- 同一真实缺口在旧补同步任务软删除后再次 repair 时创建新的 active pending 任务，而不是返回 `skippedExisting`。
- 批量全历史缺口 repair 中任一窗口无效时返回 `data.ErrNotFound`，事务回滚，已经尝试创建的任务不落库。
- 回滚后同一真实缺口仍能单独 repair，证明没有残留 duplicate 或半截状态。
- 不引入 migration，不改变 `market_candles` 事实数据，不改变前端 API 契约。

修复范围：

- `market_candle_gap_store_integration_test.go` 增加 `TestIntegrationRepairMarketCandleGapIgnoresSoftDeletedRepairTask`，覆盖软删除后同窗口重建补同步任务。
- `market_candle_gap_store_integration_test.go` 增加 `TestIntegrationRepairMarketCandleGapsRollsBackWhenAnyGapIsInvalid`，覆盖批量 repair 事务回滚。
- 生产代码无需改动；现有 `marketCandleRepairTaskExists` 的 `deleted_at IS NULL` 和事务边界由新增测试锁定。

验证：

- 本机 `go test ./internal/store/postgres -run 'TestIntegrationRepairMarketCandleGap(IgnoresSoftDeletedRepairTask|sRollsBackWhenAnyGapIsInvalid)' -count=1 -v` 因未设置 `TICTICK_TEST_DATABASE_URL` 跳过，编译通过。
- Docker Compose PostgreSQL 集成测试通过：`docker run --rm --network tictick-hi_default -v "$PWD":/src -w /src -e TICTICK_TEST_DATABASE_URL='postgresql://tictick:tictick-local-postgres-password@postgres:5432/tictick_hi?sslmode=disable' golang:1.26-bookworm go test ./internal/store/postgres -run 'TestIntegrationRepairMarketCandleGap(IgnoresSoftDeletedRepairTask|sRollsBackWhenAnyGapIsInvalid)' -count=1 -v`。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过，23 个测试文件、120 条测试通过。
- `pnpm --dir web/frontend run build` 通过。

剩余风险：

- 本轮只补强全历史缺口 repair 的软删除去重和批量事务原子性证据，不实现自动补齐或重试调度策略。
- 数据同步 worker 仍缺完整统一状态机、分布式多实例限流和真实外部交易所长期恢复压测；研究页和项目整体仍是 `scaffold`。

### 阶段 1 全历史缺口修复 active catalog 边界补充

目标等级：scaffold

触发问题：

- 研究页全历史缺口修复入口已经会创建无源 `data_sync_tasks` 补同步任务，但该入口只验证缺口窗口真实性，没有在 API 层验证 exchange / symbol 仍是 active catalog。
- inactive 或 missing market 的补同步任务创建后会被 data sync claim 边界跳过，用户看到的是“已排队但不会被执行”的半截状态。

Definition of Done：

- `POST /api/market/candle-gaps/repair` 在创建补同步任务前校验 exact active `market_instruments` catalog 命中。
- `POST /api/market/candle-gaps/repair-batch` 在批量创建前校验同一 active catalog 边界。
- inactive / missing market 返回 HTTP 400 和领域错误码 `market_instrument_not_active`，且不写入 `data_sync_tasks`。
- 已有真实缺口 repair 成功路径不回退。
- 不引入 migration，不改变 `market_candles` 事实数据，不改变前端 API 契约。

修复范围：

- `internal/web/api/market_handlers.go` 增加全历史缺口 repair 的 active catalog 前置校验。
- `internal/web/api/market_handlers_test.go` 覆盖单缺口 repair inactive catalog 拦截和批量 repair missing catalog 拦截。

验证：

- `go test ./internal/web/api -run 'TestMarketCandleGap(RepairRouteQueuesSyncTask|RepairRouteRequiresActiveMarketInstrument|BatchRepairRouteQueuesReturnedGaps|BatchRepairRouteRequiresActiveMarketInstrument)' -count=1 -v` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过，23 个测试文件、120 条测试通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

剩余风险：

- 本轮只阻止 inactive / missing market 继续创建全历史缺口补同步任务，不实现退市/停牌后的跨模块迁移策略。
- active catalog 仍依赖后台同步或人工刷新，不等于生产级交易所状态治理。
- 数据同步 worker 仍缺完整统一状态机、分布式多实例限流和真实外部交易所长期恢复压测；研究页和项目整体仍是 `scaffold`。

### 阶段 1/3/4 图表可读高度与详情页布局补充

目标等级：scaffold

触发问题：

- 研究页虽然已经改成列表在上、图表在下，但图表高度继续按剩余视口折算，任务列表稍高时 K 线主图被压成窄条，无法承载研究页“图表是主体”的产品要求。
- 图表内部为避免边缘裁切预留的价格轴和时间轴逻辑 padding 过大，在宽屏下形成明显右侧空白；图表左侧也缺少页面级安全边距。
- 交易详情和回测详情仍沿用左侧图表、右侧信息栏布局，和当前要求的“上方图表、下方左窄概要右宽列表 Tab”不一致。

Definition of Done：

- 研究页数据同步任务列表继续在上方，但高度上限收敛为轻量工作区，内部滚动，不继续挤压 K 线图表。
- 研究页 K 线图表使用明确的可读固定高度区间，不再用 `viewport - 大常量` 平分或挤剩余高度。
- 研究页图表增加内层固定视口，左右和底部有受控安全边距，`TradingViewChart` 只观察该固定视口。
- `TradingViewChart` 缩小桌面/移动价格轴最小宽度，并收紧时间轴左右逻辑 padding，减少右侧大块空白，同时保留边缘标签防裁切。
- 交易详情页和回测详情页统一为上方全宽图表、下方两列布局；左列为概要，右列为 Tab 汇总列表信息；窄屏自动堆叠。
- 不改变 API、数据模型、K 线查询语义、回测和交易 runner 行为。

修复范围：

- `web/frontend/src/pages/ResearchPage.vue` / `ResearchPage.css` 调整研究页图表固定视口、任务列表高度上限和图表安全边距。
- `web/frontend/src/components/chart/TradingViewChart.vue` 收紧价格轴宽度和时间轴逻辑 padding。
- `web/frontend/src/pages/TradingDetailPage.vue` 调整交易详情页为上图表、下概要 + Tab 列表。
- `web/frontend/src/pages/BacktestDetailPage.vue` 调整回测详情页为同款布局，并将参数、意图、订单纳入 Tab 区。
- `web/frontend/src/pages/ResearchPage.layout.test.ts`、`web/frontend/src/pages/DetailPages.layout.test.ts` 和 `web/frontend/src/components/chart/TradingViewChart.test.ts` 锁定布局契约。
- `scripts/check-research-chart-layout.sh` 和 `scripts/research-chart-height-smoke.mjs` 同步新的内层固定视口、可读高度和滚动语义。

验证：

- `pnpm --dir web/frontend run test -- src/pages/ResearchPage.layout.test.ts src/pages/DetailPages.layout.test.ts src/components/chart/TradingViewChart.test.ts` 通过，实际执行 24 个测试文件、122 条测试通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- `docker compose build api && docker compose up -d api` 通过，本地 `http://127.0.0.1:8080/readyz` 返回 ready。
- `node scripts/stage8-visual-smoke.mjs` 通过，桌面/移动、浅色/深色核心页面无横向溢出和运行时错误。
- `node scripts/research-chart-height-smoke.mjs` 通过，桌面 1440 图表体 522px、桌面 2048 图表体 668px、812x1320 图表体 640px、移动图表体 523px，内部高度污染后保持稳定。
- 本地 headless Chrome 实际采样 `/trading/tt_a9a9801f53152b7fcf74f78e` 和 `/backtests/bt_8c9a0535e2a3f8a60a7a6918` 通过：图表在上且高度 522px，下方两列为 427px / 949px 左窄右宽，无横向溢出。

剩余风险：

- 本轮只修复研究页、交易详情页和回测详情页的布局可读性，不建立人工视觉基线截图审批或全语言/全主题像素基线。
- 详情页图表仍复用当前 lightweight-charts 封装，未新增指标、画线、缩放预设或交易分析工具。
- 项目整体仍是 `scaffold`，不能升级为 demo、usable 或 production-safe。

### 阶段 1 K 线正价格边界补充

目标等级：scaffold

触发问题：

- 共享 K 线校验原先只拒绝负数，允许 `open/high/low/close=0` 进入 CandleProvider、数据同步结果和后续研究/回测/交易输入链路。
- PostgreSQL 旧约束 `market_candles_non_negative_values_check` 只要求 OHLCV 非负，直接绕过 Go 层写入时也允许零价格 K 线成为“健康”事实数据。
- 真实交易所 K 线价格为 0 应视为异常 payload；成交量为 0 仍是合法场景。

Definition of Done：

- `internal/data` 共享 K 线校验要求 `open/high/low/close > 0`，且 `volume >= 0`。
- data sync runner 抓取到零价格 K 线时任务失败，不保存、不推进游标、不作为临时错误重试。
- PostgreSQL 对 `market_candles` 新写入增加 OHLC 正价格约束，历史行暂不强制扫描。
- 现有集成测试 fixture 不再生成零价格假数据。
- 不改变 API 契约、图表布局、交易所 adapter、历史数据清理策略、订单/成交价格约束。

修复范围：

- `internal/data/candle_validation.go` 将价格字段和成交量字段分开校验，价格必须为正，成交量只拒绝负数。
- `internal/data/candle_validation_test.go` 增加零价格负向用例，并保持零成交量场景可用于该用例。
- `internal/datasync/runner_validation_test.go` 覆盖零价格 fetched candle 不入库、不 retry、任务失败。
- `internal/store/postgres/migrations/0030_market_candle_positive_prices.sql` 新增 `market_candles_positive_price_values_check`，使用 `NOT VALID` 避免阻断历史行迁移。
- `internal/store/postgres/integration_constraints_test.go` 覆盖直接 insert 零价格命中新约束。
- `internal/store/postgres/integration_test.go` 和 `internal/store/postgres/candle_pagination_integration_test.go` 将与价格无关的 fixture 改为正价格数据。

验证：

- `go test ./internal/data -run TestValidateCandleSeriesRejectsInvalidCandles -count=1` 通过。
- `go test ./internal/datasync -run 'TestRunnerRejects(InvalidFetchedCandleBeforeSaving|ZeroPriceFetchedCandleBeforeSaving|MismatchedFetchedCandleTargetBeforeSaving)' -count=1` 通过。
- 本机 `go test ./internal/store/postgres -run 'TestIntegrationDatabaseConstraintsRejectInvalidDomainValues|TestIntegrationListNativeCandlesUsesLatestWindowWithoutRange|TestIntegrationListNativeCandlesClampsOversizedLimit' -count=1 -v` 因未设置 `TICTICK_TEST_DATABASE_URL` 跳过，编译通过。
- Docker Compose PostgreSQL 集成测试通过：`docker run --rm --network tictick-hi_default -v "$PWD":/src -w /src -e TICTICK_TEST_DATABASE_URL='postgresql://tictick:tictick-local-postgres-password@postgres:5432/tictick_hi?sslmode=disable' golang:1.26-bookworm go test ./internal/store/postgres -run 'TestIntegrationDatabaseConstraintsRejectInvalidDomainValues|TestIntegrationListNativeCandlesUsesLatestWindowWithoutRange|TestIntegrationListNativeCandlesClampsOversizedLimit' -count=1 -v`。
- Docker Compose PostgreSQL 扩展目标集通过：`TestIntegrationCandleProviderReportsPaginationWindows`、`TestIntegrationListNativeCandlesUsesLatestWindowBeforeTo`、`TestIntegrationCandleProviderReportsRequestedRangeBoundaryGaps`、`TestIntegrationDatabaseConstraintsRejectInvalidDomainValues`、`TestIntegrationListNativeCandlesUsesLatestWindowWithoutRange`、`TestIntegrationListNativeCandlesClampsOversizedLimit`。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过，24 个测试文件、122 条测试通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

剩余风险：

- `0030` 使用 `NOT VALID`，只保证新写入/更新被约束，不证明历史 `market_candles` 已全量清洗。
- 本轮不实现自动隔离已存在零价格 K 线、重算缺口、图表异常标记或补同步修复策略。
- 数据同步 worker、CandleProvider 和研究页仍不能升级到 usable；项目整体仍是 `scaffold`。

### 阶段 1 历史异常 K 线可观察补充

目标等级：scaffold

触发问题：

- `0030_market_candle_positive_prices.sql` 使用 `NOT VALID`，可以保护新写入，但历史 `market_candles` 里如果已存在零价格或其它异常 K 线，不会被迁移自动扫描出来。
- 上一轮收紧共享校验后，CandleProvider 读取到这类历史异常行会返回 Go error，`/api/candles` 可能变成 500；这对研究页用户不可观察，也容易被误判为系统故障而不是数据质量问题。
- 策略入口已经拒绝非 `ok` 健康状态，因此历史异常 K 线应进入统一健康状态，而不是绕过健康语义。

Definition of Done：

- CandleProvider 遇到 native 或 aggregation base 历史异常 K 线时返回 `health=invalid`，不把该问题冒泡为 API 500。
- CandleResult 返回 `issues` 摘要，至少包含问题 code、message，并在能定位时包含异常 K 线 `openTime`。
- `/api/candles` 对历史异常 K 线返回 HTTP 200 和 `health=invalid`。
- 研究页显示 `invalid` 数据健康标签和首个异常摘要；前端 API generated types、app types 和 i18n 同步更新。
- 策略侧 `ValidateStrategyCandleResult` 继续拒绝 `invalid`。
- 不清洗历史数据，不自动补同步，不改变新写入正价格约束，不扩大到任务列表窗口级 invalid 统计。

修复范围：

- `internal/data/model.go` 新增 `CandleHealthInvalid` 和 `CandleIssue`，并将 `issues` 加入 `CandleResult`。
- `internal/data/candle_provider.go` 将 native / aggregation base 校验失败转换为 invalid CandleResult。
- `internal/data/candle_provider_test.go` 覆盖 duplicate、零价格和聚合基础 K 线异常的 invalid 返回。
- `internal/data/candle_result_test.go` 覆盖策略入口拒绝 `invalid`。
- `internal/web/api/candles_test.go` 覆盖 `/api/candles` 历史异常数据返回 200 + `health=invalid`。
- `internal/store/postgres/candle_invalid_integration_test.go` 通过临时模拟 legacy 行验证真实 PostgreSQL 下 `NOT VALID` 约束后的历史异常行可被 CandleProvider 标为 invalid。
- `internal/web/api/contract_schema.go`、`web/frontend/src/types/api.generated.ts`、`web/frontend/src/types/app.ts`、`web/frontend/src/services/api/data.ts`、`web/frontend/src/pages/ResearchPage.vue` 和 i18n 文案同步前端可观察状态。

验证：

- `go test ./internal/data -run 'TestCandleProviderReportsInvalid|TestValidateStrategyCandleResult' -count=1` 通过。
- `go test ./internal/web/api -run 'TestCandlesRouteReturnsInvalidHealthForHistoricalBadCandles|TestFrontendAPI(ResponseTypesMatchContractFields|AdapterResponseFieldsExistInContract|GeneratedTypesAreCurrent)' -count=1` 通过。
- 本机 `go test ./internal/store/postgres -run TestIntegrationCandleProviderReportsLegacyInvalidCandle -count=1 -v` 因未设置 `TICTICK_TEST_DATABASE_URL` 跳过，编译通过。
- Docker Compose PostgreSQL 集成测试通过：`docker run --rm --network tictick-hi_default -v "$PWD":/src -w /src -e TICTICK_TEST_DATABASE_URL='postgresql://tictick:tictick-local-postgres-password@postgres:5432/tictick_hi?sslmode=disable' golang:1.26-bookworm go test ./internal/store/postgres -run TestIntegrationCandleProviderReportsLegacyInvalidCandle -count=1 -v`。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test -- src/pages/ResearchPage.layout.test.ts` 通过，实际执行 24 个测试文件、122 条测试通过。
- `go test ./internal/web/api -run 'TestFrontendAPI' -count=1` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过，24 个测试文件、122 条测试通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- 本地 PostgreSQL `market_candles_positive_price_values_check` 约束仍存在且 `convalidated=false`。

剩余风险：

- 任务列表窗口级 `dataHealth=invalid` 统计已在后续补充关闭；用户仍需要进入图表请求窗口才能看到具体 `issues` 摘要。
- 本轮不清洗已有异常行，不自动排补同步任务，也不把 invalid 行纳入全历史缺口扫描。
- 项目整体仍是 `scaffold`，CandleProvider 和研究页仍不能升级为 usable。

### 阶段 1 数据同步任务窗口级 invalid 健康补充

目标等级：scaffold

触发问题：

- CandleProvider 已能把历史异常 K 线转成 `health=invalid`，但研究页任务列表只统计缺口、失败、重试、同步中等状态，用户必须进入图表窗口才会看到异常数据。
- `0030_market_candle_positive_prices.sql` 使用 `NOT VALID`，历史 `market_candles` 可以继续保留零价格等 legacy 脏行；任务列表需要在窗口级别提前暴露这些风险。

Definition of Done：

- `DataSyncHealth` 保持 `invalid` 枚举，并进入后端 OpenAPI contract 和前端 generated DTO。
- `ListDataSyncTasks` 在任务窗口内统计历史异常 OHLCV K 线：价格非正、成交量为负、高低价边界错误都派生为 `dataHealth=invalid`。
- 任务列表返回 `invalidSummary`，包含窗口内异常数量和首个异常 `openTime/code/message`，不再只给一个状态标签。
- 健康优先级保持执行状态优先：failed / retrying 仍优先于 invalid，invalid 优先于 gap。
- 研究页任务表质量摘要列优先展示异常摘要，其次展示缺口摘要；异常摘要通过 tooltip 保留完整原因。
- 不清洗历史行，不自动修复，不新增补同步策略，不改变 CandleProvider 查询语义。

修复范围：

- `internal/data/data_sync_model.go` 新增 `DataSyncInvalidSummary` 并挂到 `DataSyncTask.InvalidSummary`。
- `internal/store/postgres/data_sync_task_scan.go` 在任务窗口 lateral 查询中增加异常数量和首个异常详情，覆盖正价格、成交量、高低价边界。
- `internal/web/api/contract_schema.go` 和 `web/frontend/src/types/api.generated.ts` 同步 `DataSyncInvalidSummary` contract。
- `web/frontend/src/services/api/data.ts` 保留 `invalidSummary` DTO 字段。
- `web/frontend/src/components/tables/DataSyncTaskTable.vue` 将质量列拆为 `DataSyncQualitySummary`，避免表格组件继续膨胀。
- `web/frontend/src/components/tables/DataSyncQualitySummary.vue` 统一渲染异常/缺口质量摘要。
- `internal/store/postgres/integration_data_sync_invalid_health_test.go` 覆盖真实 PostgreSQL legacy 零价格 K 线下的任务列表 `dataHealth=invalid` 与 `invalidSummary`。
- `web/frontend/src/components/tables/DataSyncTaskTable.test.ts` 覆盖任务表显示异常数量和首个异常原因。

验证：

- `scripts/generate-api-types.sh` 通过。
- `go test ./internal/store/postgres -run '^TestIntegrationListDataSyncTasksReportsInvalidCandleHealth$' -count=1` 通过。
- `pnpm --dir web/frontend exec vitest run src/components/tables/DataSyncTaskTable.test.ts src/services/api/data.test.ts` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过，24 个测试文件 / 124 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

剩余风险：

- 任务列表质量摘要仍只展示首个异常；逐根异常详情入口已在后续补充关闭，但本轮不清洗或自动修复异常行。
- 本轮不清洗 legacy 脏数据，不自动排补同步任务，也不把 invalid 行纳入全历史缺口扫描。
- 项目整体仍是 `scaffold`，阶段 1 研究核心仍未达到 usable。

### 阶段 1/3/4 图表布局按 tictickbot 模式返工

目标等级：scaffold

触发问题：

- 用户在本地 `127.0.0.1:8080/research` 继续反馈 K 线图表高度、左右边距和右侧空白不符合生产级要求。
- 用户明确指出交易详情、回测详情也存在同类排版问题，不能只修研究页。
- 参考 `tictickbot` 后确认其 K 线图表组件自身只负责 `width: 100%; height: 100%`，由页面外层提供清晰固定高度和少量容器内边距。

Definition of Done：

- 研究页仍保持同步任务列表在上、图表在下，但任务列表作为轻量工作区滚动，K 线图表拥有可读高度。
- 研究页、交易详情、回测详情都使用外层卡片控制边框 / 内边距，内层 `data-chart-viewport="fixed"` 作为真实图表测量节点。
- 交易详情和回测详情不再继承全局 `.chart-panel` 的旧高度 / size containment，图表在上，下方两列保持左窄右宽。
- `TradingViewChart` 收紧价格轴宽度和时间轴逻辑 padding，减少右侧空白，同时保留边缘标签防裁切。
- 浏览器级检查不再强迫图表完整塞进当前首屏，改为验证固定图表槽内部不裁切、不横向溢出、不被内部节点污染到无限增高。

修复范围：

- `web/frontend/src/pages/ResearchPage.css` 调整研究页任务表高度上限、图表高度区间和图表容器安全边距。
- `web/frontend/src/pages/ResearchPage.vue` 将图表工具栏拆成主控件行和状态行，避免 symbol 输入、刷新按钮、窗口按钮和状态标签互相挤压。
- `web/frontend/src/components/market/MarketSymbolAutoComplete.vue` 支持控件尺寸参数，研究页工具栏使用小尺寸输入和刷新按钮。
- `web/frontend/src/components/research/ResearchWindowControls.vue` 收紧窗口按钮间距和最小宽度。
- `web/frontend/src/pages/detailChartLayout.css` 抽出交易详情和回测详情共用图表高度 / viewport 样式，避免详情页文件超过质量门禁硬限制。
- `web/frontend/src/pages/TradingDetailPage.vue`、`web/frontend/src/pages/BacktestDetailPage.vue` 改为外层图表卡片 + 内层固定 viewport，去除全局 `.chart-panel` 继承。
- `web/frontend/src/components/chart/TradingViewChart.vue` 收紧价格轴宽度和时间轴边缘 padding。
- `web/frontend/src/pages/ResearchPage.layout.test.ts`、`web/frontend/src/pages/DetailPages.layout.test.ts`、`web/frontend/src/components/chart/TradingViewChart.test.ts` 更新布局契约。
- `scripts/check-research-chart-layout.sh`、`scripts/research-chart-height-smoke.mjs` 同步运行态检查语义。

验证：

- `pnpm --dir web/frontend exec vitest run src/pages/ResearchPage.layout.test.ts src/pages/DetailPages.layout.test.ts src/components/chart/TradingViewChart.test.ts` 通过，3 个测试文件 / 32 条测试。
- `scripts/check-research-chart-layout.sh` 通过。
- `node --check scripts/research-chart-height-smoke.mjs` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `git diff --check` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过，24 个测试文件 / 122 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过，包含 file size、research chart layout 和 Stage 8 command config smoke；`TradingDetailPage.vue` 拆分后低于 450 行硬限制。
- `docker compose up -d --build api` 通过，本地 `http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：1440 桌面图表渲染高 690px，2048 桌面 799px，812 窄桌面 732px，390 移动 550px，高度稳定，812 窄桌面不再横向溢出。
- Headless Chrome 几何采样通过：研究页 1440 桌面 panel `1392x802`、toolbar `1390x80`、symbol 输入 `520px` 封顶；812 窄桌面 panel `780x897`、toolbar `778x135`、source controls `754x28`、symbol 输入 `314px`，刷新按钮在 symbol 控件内部。交易详情 `/trading/tt_a9a9801f53152b7fcf74f78e` 和回测详情 `/backtests/bt_8c9a0535e2a3f8a60a7a6918` 两页图表 panel 均为 `1392x630`，实际图表 viewport 均为 `1366x604`，下方两列为 `427/949` 左窄右宽，无横向溢出。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=1000 node scripts/stage8-visual-smoke.mjs` 通过，桌面 / 移动、浅色 / 深色核心页面 max document width 均未超过 viewport。

剩余风险：

- 本轮只修复 K 线图表布局、高度和边距，不新增指标工具、绘图工具、成交点联动或完整交易分析能力。
- 研究页、交易详情、回测详情仍未建立人工截图基线审批；本轮以 DOM 几何、canvas 像素、无横向溢出和高度稳定性作为自动验收。
- 项目整体仍是 `scaffold`，不能升级为 production-safe。

### 阶段 1 数据同步错误摘要和 retry/backoff 可观察性补充

目标等级：scaffold

触发问题：

- 数据同步任务遇到外部交易所 EOF / 超时等临时错误时，运行时 adapter 通常已经生成 host 级错误摘要，但 store 层仍只做空白归一和 500 字截断；如果未来其他 fetcher 或测试直接传入 `Get "https://..."` 错误，`data_sync_tasks.last_error` 和 `data_sync_exchange_backoffs.last_error` 仍可能保存完整请求路径和 query。
- 研究页任务表虽然已有错误列省略和 tooltip，但 retry / exchange backoff 时间仍直接显示原始 ISO 字符串，影响任务表密度。

Definition of Done：

- Go 侧外部错误清洗规则抽成共享边界，API 响应和 PostgreSQL 写入都复用同一逻辑。
- `RecordDataSyncRetry` 和 `MarkDataSyncFailed` 写入 `last_error` 前移除外部请求 URL path / query，只保留 host 和原因摘要。
- exchange-level backoff 表保存的 `last_error` 同样不包含完整外部请求 URL。
- 研究页任务表的最新同步时间、下次重试时间和交易所退避时间显示为紧凑本地时间，完整原始时间保留在 title / tooltip。
- 不改变 retry/backoff 状态机、不改变退避时长、不新增分布式限流、不自动重试所有历史失败任务。

修复范围：

- 新增 `internal/errtext.ExternalError`，统一清洗外部错误 URL、空白和最大长度。
- `internal/store/postgres/sync_store.go` 的 data sync 错误持久化改为调用共享清洗函数。
- `internal/web/api/data_handlers.go` 的 data sync API 响应清洗改为复用共享函数，避免 API / store 规则分叉。
- `internal/store/postgres/integration_test.go` 的 data sync retry 集成路径改用原始 Binance URL 错误输入，并断言任务表与 exchange backoff 表都只保存 host 摘要。
- `web/frontend/src/components/tables/DataSyncTaskTable.vue` 将 retry/backoff/最新同步时间渲染为紧凑本地时间。
- 新增 `web/frontend/src/utils/displayText.ts` 承载前端短时间和文本摘要工具，避免表格组件超过文件大小硬限制。

验证：

- `go test ./internal/errtext ./internal/store/postgres ./internal/web/api -run 'TestExternalError|TestNormalizeTaskError|TestIntegrationDataSyncRetryReleasesAndReclaimsTask|TestDataSyncTaskRoutesSanitizeLastError' -count=1` 通过。
- `pnpm --dir web/frontend exec vitest run src/components/tables/DataSyncTaskTable.test.ts src/services/api/data.test.ts` 通过，2 个测试文件 / 24 条测试。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过，24 个测试文件 / 123 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过，文件大小硬门禁恢复：`DataSyncTaskTable.vue` 448 行，`internal/store/postgres/integration_test.go` 695 行。

剩余风险：

- 本轮只保证 data sync 任务和 exchange backoff 错误摘要不会保存完整外部请求 URL；其它领域的历史 `last_error` 文本没有迁移清洗。
- 本轮不证明真实 Binance / OKX 网络恢复压测，也不实现多实例共享限流。
- retry/backoff 仍是基础可观察和可恢复边界，尚未升级为 production-safe。

### 阶段 1 研究页工具栏与价格轴二次收口补充

目标等级：scaffold

触发问题：

- 本地 `127.0.0.1:8080/research` 中 symbol 输入仍显得过宽，压缩了工具栏密度。
- K 线图表右侧价格轴区域仍出现明显宽 gutter，影响主图可读面积。
- 这类问题必须有可重复的 DOM / canvas 尺寸检查，不能只靠截图和主观判断。

Definition of Done：

- 研究页桌面工具栏 symbol 输入收敛到 `clamp(180px, 18vw, 300px)`，窄桌面最多 240px。
- `TradingViewChart` 右侧价格轴最小宽度收敛到 56 / 60 / 64px，宽屏不再保留 88px 以上的价格轴预算。
- `scripts/research-chart-height-smoke.mjs` 增加右侧价格轴宽度上限，超过 96px 直接失败。
- 本地生产构建后的 1440 桌面 DOM 采样证明 symbol 输入约 259px、右侧价格轴 canvas 64px，且 document 不横向溢出。

修复范围：

- `web/frontend/src/pages/ResearchPage.css` 收紧 `.research-source-controls` 桌面和窄桌面 grid 宽度。
- `web/frontend/src/pages/ResearchPage.layout.test.ts` 和 `scripts/check-research-chart-layout.sh` 锁定新的工具栏宽度契约。
- `web/frontend/src/components/chart/TradingViewChart.vue` 收紧响应式右侧价格轴宽度。
- `web/frontend/src/components/chart/TradingViewChart.test.ts` 锁定桌面 64px、窄屏 56px 的价格轴配置。
- `scripts/research-chart-height-smoke.mjs` 增加 `SMOKE_MAX_RIGHT_PRICE_AXIS_WIDTH` 检查。

验证：

- `pnpm --dir web/frontend exec vitest run src/pages/ResearchPage.layout.test.ts src/components/chart/TradingViewChart.test.ts` 通过，2 个测试文件 / 30 条测试。
- `scripts/check-research-chart-layout.sh` 通过。
- `node --check scripts/research-chart-height-smoke.mjs` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `docker compose up -d --build api` 通过，本地 `http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`。
- Headless Chrome 1440 桌面 DOM 采样：source controls `723px`，symbol 输入 `259px`，right price-axis canvas `64px`，document width `1440px`。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：1440 桌面 chart/tv `690px`，2048 桌面 `799px`，812 窄桌面 `732px`，390 移动 `550px`，连续采样稳定。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=1000 node scripts/stage8-visual-smoke.mjs` 通过，桌面 / 移动、浅色 / 深色核心页面无横向溢出。
- `go test ./...`、`go vet ./...`、`pnpm --dir web/frontend run typecheck`、`pnpm --dir web/frontend run test`、`scripts/quality-gate.sh`、`git diff --check` 均通过。

剩余风险：

- 本轮只收紧研究页工具栏和通用 K 线价格轴 gutter，不新增指标、绘图工具或交易分析交互。
- 交易详情 / 回测详情沿用同一个 `TradingViewChart` 收紧后的价格轴，但本轮没有重新做人工截图基线审批。
- 项目整体仍是 `scaffold`，阶段 1 研究核心仍未达到 usable。

### 阶段 1 数据同步任务 invalid 逐根详情入口补充

目标等级：scaffold

触发问题：

- 任务列表已经返回 `invalidSummary`，但只能看到异常数量和首个异常摘要。
- 用户仍需要一个从研究页任务表直接进入的详情入口，查看任务窗口内逐根异常 K 线，不应必须切到图表窗口碰运气。
- 该入口必须是 PostgreSQL 真实窗口查询，不允许前端只展开 summary。

Definition of Done：

- 后端提供 `GET /api/data/tasks/{id}/invalid-issues`，按数据同步任务窗口返回异常 K 线 `issues`。
- 返回 metadata 包含 `totalCount`、`returnedCount`、`limited`、`issueLimit`，超过单次上限时前端能看见。
- PostgreSQL 查询复用任务窗口边界和 OHLCV 异常判定：价格非正、成交量为负、高低价边界错误。
- 研究页任务表对有 `invalidSummary` 的任务显示异常详情按钮，打开弹窗展示逐根异常时间、类型和原始原因。
- 不清洗历史行，不自动修复，不新增补同步策略，不改变 CandleProvider 或 data sync 状态机。

修复范围：

- `internal/data/data_sync_model.go` 新增 `DataSyncInvalidIssueList`。
- `internal/data/model.go` 新增 repository 方法 `ListDataSyncTaskInvalidIssues`。
- `internal/store/postgres/sync_invalid_issue_store.go` 新增窗口级 invalid issues 查询。
- `internal/web/api/data_handlers.go`、`contract_routes.go`、`contract_schema.go`、相关 route / method / schema drift 测试接入新 endpoint。
- `web/frontend/src/services/api/data.ts` 和 generated/app types 接入 `DataSyncInvalidIssueList`。
- `web/frontend/src/components/tables/DataSyncTaskTable.vue` 增加 `view-invalid` 操作。
- `web/frontend/src/components/research/ResearchTaskInvalidIssueModal.vue` 新增异常详情弹窗。
- `web/frontend/src/pages/ResearchPage.vue` 接入弹窗和表格事件。
- 中英文 i18n 新增异常详情文案。

验证：

- `scripts/generate-api-types.sh` 通过。
- `go test ./internal/store/postgres ./internal/web/api -run 'TestIntegrationListDataSyncTasksReportsInvalidCandleHealth|TestDataSyncTaskRoutes|TestAPIMethodNotAllowedContracts|TestAPIContractRoutes|TestFrontendAPIResponseTypesMatchContractFields|TestFrontendAPIAppTypesReferenceGeneratedContract' -count=1` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend exec vitest run src/services/api/data.test.ts src/services/api/data.invalid.test.ts src/components/tables/DataSyncTaskTable.test.ts src/components/research/ResearchTaskInvalidIssueModal.test.ts src/pages/ResearchPage.layout.test.ts` 通过，5 个测试文件 / 39 条测试。
- `pnpm --dir web/frontend exec vitest run src/pages/DetailPages.layout.test.ts src/pages/ResearchPage.layout.test.ts src/components/chart/TradingViewChart.test.ts` 通过，3 个测试文件 / 33 条测试。
- `pnpm --dir web/frontend run test` 通过，26 个测试文件 / 128 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/check-file-size.sh` 通过。
- `scripts/check-research-chart-layout.sh` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- `docker compose up -d --build api` 通过，本地 `http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：1440 桌面 chart/tv `690px`，2048 桌面 `799px`，812 窄桌面 `732px`，390 移动 `550px`，高度稳定。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=1000 node scripts/stage8-visual-smoke.mjs` 通过，桌面 / 移动、浅色 / 深色核心页面无横向溢出。
- 文件行数仍低于硬门禁：`ResearchPage.vue` 416 行，`useResearchWorkspace.ts` 382 行，`DataSyncTaskTable.vue` 411 行。

剩余风险：

- invalid 详情是只读观察入口，不提供自动清洗、自动补同步或批量处置。
- 单次最多返回 50 条异常；更大窗口分页查看已在后续补充关闭，但仍缺筛选能力。
- 项目整体仍是 `scaffold`，阶段 1 研究核心仍未达到 usable。

### 阶段 1 数据同步任务 invalid 详情分页补充

目标等级：scaffold

触发问题：

- `GET /api/data/tasks/{id}/invalid-issues` 只能返回前 50 条异常。
- 大窗口内存在超过 50 条历史异常 K 线时，用户只能知道受限，不能继续追踪后续异常。
- 该补充必须保持单次返回上限，避免为了展示完整异常列表而扩大 API 响应。

Definition of Done：

- `GET /api/data/tasks/{id}/invalid-issues` 支持 `limit` 和 `offset` query，`limit` 最大仍为 50。
- 响应返回 `offset`、`totalCount`、`returnedCount`、`limited` 和当前页 `issueLimit`。
- PostgreSQL 查询在 offset 越界时仍返回正确 `totalCount` 和空页，不把总数误判为 0。
- 研究页异常详情弹窗可以分页请求后续异常，并显示当前已查看到第几条。
- 不清洗历史行，不自动修复，不新增补同步策略，不改变 CandleProvider 或 data sync 状态机。

修复范围：

- `internal/data/data_sync_model.go` 新增 invalid issue query limit/offset 和响应 `offset`。
- `internal/store/postgres/sync_invalid_issue_store.go` 拆分 total count 和 page 查询，支持 offset 分页。
- `internal/web/api/data_handlers.go` 解析并校验 `limit/offset`。
- `internal/web/api/contract_routes.go` 和 generated frontend API types 同步 query/response contract。
- `web/frontend/src/services/api/data.ts` 支持传入 invalid issue `limit/offset`。
- `web/frontend/src/components/research/ResearchTaskInvalidIssueModal.vue` 使用分页控件请求后续页。

验证：

- `scripts/generate-api-types.sh` 通过。
- `go test ./internal/store/postgres ./internal/web/api -run 'TestIntegrationListDataSyncTasksReportsInvalidCandleHealth|TestDataSyncTaskRoutes|TestAPIMethodNotAllowedContracts|TestAPIContractRoutes|TestFrontendAPIResponseTypesMatchContractFields|TestFrontendAPIAppTypesReferenceGeneratedContract' -count=1` 通过。
- `pnpm --dir web/frontend exec vitest run src/services/api/data.invalid.test.ts src/components/research/ResearchTaskInvalidIssueModal.test.ts src/pages/ResearchPage.layout.test.ts` 通过，3 个测试文件 / 14 条测试。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过，26 个测试文件 / 129 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/check-file-size.sh` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- `docker compose up -d --build api` 通过，本地 `http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`。

剩余风险：

- invalid 详情仍是只读观察入口，不提供自动清洗、自动补同步或批量处置。
- 已支持按异常类型和时间范围过滤，但没有异常严重程度分级。
- 项目整体仍是 `scaffold`，阶段 1 研究核心仍未达到 usable。

### 阶段 1 数据同步任务 invalid 详情筛选补充

目标等级：scaffold

触发问题：

- invalid 详情分页后仍需要人工翻页定位具体异常类型或时间段。
- 任务窗口内异常数量较多时，缺少 code/time 过滤会降低排障效率。

Definition of Done：

- `GET /api/data/tasks/{id}/invalid-issues` 支持 `code`、`from`、`to` query，并保留 `limit/offset` 分页上限。
- `code` 只接受已知 OHLCV invalid code，未知值返回 400。
- `from/to` 使用 RFC3339 时间并校验 `from <= to`。
- PostgreSQL total count 与返回页均基于相同过滤条件。
- 前端异常详情弹窗支持按异常类型和时间范围筛选，筛选时重置到第一页。

修复范围：

- `DataSyncInvalidIssueQuery` 增加 code/from/to，API handler、contract 和 fake repository 同步过滤语义。
- PostgreSQL invalid issue count/page 查询增加参数化过滤。
- 研究页 invalid 详情弹窗增加异常类型下拉、时间范围选择和重置按钮。
- 前端 data API、i18n 和单测覆盖筛选请求。

验证：

- `scripts/generate-api-types.sh` 通过。
- `go test ./internal/store/postgres ./internal/web/api -run 'TestIntegrationListDataSyncTasksReportsInvalidCandleHealth|TestDataSyncTaskRoutes|TestAPIContractRoutes|TestFrontendAPIResponseTypesMatchContractFields|TestFrontendAPIAppTypesReferenceGeneratedContract|TestFrontendAPIRoutesAreCoveredByContract' -count=1` 通过。
- `pnpm --dir web/frontend exec vitest run src/pages/ResearchPage.layout.test.ts src/pages/DetailPages.layout.test.ts src/components/chart/TradingViewChart.test.ts src/services/api/data.invalid.test.ts src/components/research/ResearchTaskInvalidIssueModal.test.ts` 通过，5 个测试文件 / 37 条测试。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过，26 个测试文件 / 130 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/check-file-size.sh` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

剩余风险：

- invalid 详情仍是只读观察入口，不提供自动清洗、自动补同步或批量处置。
- 仍没有异常严重程度分级和批量处置流程。
- 项目整体仍是 `scaffold`，阶段 1 研究核心仍未达到 usable。

### 阶段 1 K 线图表布局收紧补充

目标等级：scaffold

触发问题：

- 研究页工具栏控件过宽，symbol 输入占据过多横向空间。
- K 线图表左右边距和右侧价格轴留白不够精细。
- 交易详情和回测详情需要统一为上方大图表、下方左窄摘要右宽列表的布局。

Definition of Done：

- 研究页工具栏使用紧凑控件组，symbol 输入桌面宽度收敛到 `clamp(180px, 15vw, 240px)`，移动端保持全宽输入。
- 研究页图表槽用真实 plot height 计算，移动和窄桌面有独立高度/边距约束，不再把上下 padding 从可读绘图区中扣掉。
- `TradingViewChart` 右侧价格轴按视口收敛为 40/44/48px，减少非绘图区空白。
- 交易详情和回测详情均采用上图表、下双栏；摘要栏固定窄宽，列表栏占主要宽度。
- visual smoke 会检查 symbol 输入桌面最大宽度、图表 viewport 高度下限、右侧价格轴最大宽度和 chart renderer 是否填满固定 viewport。

修复范围：

- `ResearchPage.css` 收紧工具栏、symbol 输入宽度、图表槽高度和左右内边距。
- `detailChartLayout.css` 统一详情页图表 plot height 和内边距。
- `BacktestDetailPage.vue`、`TradingDetailPage.vue` 下半区改为 `minmax(280px, 340px) minmax(0, 1fr)`。
- `TradingViewChart.vue` 和主题 token 收敛右侧价格轴与时间轴边缘 padding。
- `scripts/stage8-visual-smoke.mjs` 增加研究页 toolbar / chart viewport / right price-axis 断言，并复用到详情页固定图表 viewport。
- `scripts/research-chart-height-smoke.mjs` 固定 light theme 采样，避免继承上一次浏览器 dark theme 后把时间轴深色背景误判为贴边文字，并将右侧价格轴上限收紧到 68px。

验证：

- `pnpm --dir web/frontend run test -- src/pages/ResearchPage.layout.test.ts src/pages/DetailPages.layout.test.ts src/components/chart/TradingViewChart.test.ts` 通过，26 个测试文件 / 130 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/check-research-chart-layout.sh` 通过。
- `docker compose up -d --build api` 通过，本地 `http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`。
- `BASE_URL=http://127.0.0.1:8080 CDP_ENDPOINT=http://127.0.0.1:9222 node scripts/stage8-visual-smoke.mjs` 通过，桌面 / 移动、浅色 / 深色当前本地数据均覆盖 7 个页面。
- `BASE_URL=http://127.0.0.1:8080 CDP_ENDPOINT=http://127.0.0.1:9222 SMOKE_SAMPLES=8 SMOKE_INTERVAL_MS=120 node scripts/research-chart-height-smoke.mjs` 通过：1440 桌面 chart/tv `680px`，2048 桌面 `864px`，812 窄桌面 `760px`，390 移动 `608px`，污染内部 DOM 高度后 document/panel/body/chart/tv 均稳定。
- Headless Chrome 1440 抽样：研究页工具栏 `79px`，symbol 输入 `216px`，图表 viewport / tv `1342x680`，右侧价格轴 canvas `60px`，viewport 与 tv 左右边界一致。

剩余风险：

- 仍缺像素快照基线和多浏览器视觉回归门禁。
- 详情页图表 marker、列表密度和真实长数据下的可用性还需要继续打磨。
- 项目整体仍是 `scaffold`，不能按 usable 或 production-safe 声明。

### 阶段 8 visual smoke 详情页覆盖补充

目标等级：scaffold

触发问题：

- 回测详情 / 交易详情的布局修复上一轮只有临时 Chrome DevTools 抽样，没有进入固定 smoke 脚本。
- 后续改 CSS 或详情页结构时，仍可能把上图表、下双栏布局回退而不触发质量门禁。

Definition of Done：

- `scripts/stage8-visual-smoke.mjs` 在登录后只读 `/api/backtests` 和 `/api/trading/tasks`。
- 当存在任务数据时，脚本自动进入首个回测详情和交易详情。
- 详情页 smoke 校验图表面板、固定图表 viewport、下方 grid、摘要栏和 tab 列表可见。
- 桌面视口校验摘要栏窄于列表栏且宽度落在预期范围；移动视口校验下方两块堆叠后等宽。
- 不创建任务、不写数据库、不改变业务状态。

修复范围：

- `scripts/stage8-visual-smoke.mjs` 增加详情页动态路由发现、详情页 DOM 采样和布局断言。
- `docs/quality-audit.md` 更新前端基础设施风险摘要。

验证：

- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=1000 node scripts/stage8-visual-smoke.mjs` 通过；当前本地数据下桌面 / 移动、浅色 / 深色均覆盖 7 个页面，最大 document width 分别为 `1440 / 1440 / 390 / 390`。
- `scripts/check-file-size.sh` 通过。

剩余风险：

- 详情页覆盖依赖当前环境存在至少一条回测任务和交易任务；无任务数据时脚本仍只覆盖列表和核心页面。
- 仍缺像素截图基线、全路由全语言覆盖和 CI 硬门禁。
- 项目整体仍是 `scaffold`，不能按 usable 或 production-safe 声明。

### 阶段 8 K 线图表边距和窄桌面 smoke 补充

目标等级：scaffold

触发问题：

- 用户实际查看研究页、交易详情和回测详情时反馈 K 线图表右侧空白、工具项和图表边距不够精细。
- 旧 visual smoke 只覆盖 1440 桌面和 390 移动，缺少 812px 这类窄桌面 / 半屏窗口断点。
- 旧 smoke 只检查右侧价格轴不超过 72px，没有把图表左右 gutter 作为可回归的前端质量指标。

Definition of Done：

- 研究页和详情页图表槽改为非对称 gutter：左侧保留读图空间，右侧收紧，避免价格轴外再叠加明显空白。
- `TradingViewChart` 使用更紧凑的 chart 字体，右侧价格轴响应式最小宽度收敛为 `36/40/44px`。
- `stage8-visual-smoke.mjs` 增加 `narrow-desktop-812x1320`，并断言图表左/右 gutter 与 CSS 配置一致，右侧 gutter 必须保持紧凑。
- `research-chart-height-smoke.mjs` 和 visual smoke 的右侧价格轴上限统一收紧为 `60px`。
- 本地 `8080` 必须重建 API 镜像后用真实生产静态产物验证。

修复范围：

- `web/frontend/src/pages/ResearchPage.css` 将图表 body padding 拆为 left/right 变量，桌面右侧收紧到 `4px`，窄桌面收紧到 `3px`，移动端收紧到 `2px`。
- `web/frontend/src/pages/detailChartLayout.css` 将交易详情和回测详情图表 padding 同步改为左保留、右收紧。
- `web/frontend/src/theme/tokens.ts` 给 chart layout 明确 `fontFamily/fontSize`，并收紧默认 `rightPriceScale.minimumWidth`。
- `web/frontend/src/components/chart/TradingViewChart.vue` 将响应式右价格轴最小宽度改为 `36/40/44px`。
- `scripts/stage8-visual-smoke.mjs` 增加 812 窄桌面断点和 chart gutter 断言。
- `scripts/research-chart-height-smoke.mjs` 增加 gutter 范围断言并收紧右轴上限。

验证：

- `pnpm --dir web/frontend exec vitest run src/pages/ResearchPage.layout.test.ts src/pages/DetailPages.layout.test.ts src/components/chart/TradingViewChart.test.ts` 通过：3 个测试文件、34 条测试。
- `scripts/check-research-chart-layout.sh` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run build` 通过，生产入口为 `/assets/index-cgHYQmgA.js`，研究页 CSS 为 `/assets/ResearchPage-DR58cRmi.css`。
- `docker compose up -d --build api` 通过；本地 `http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`，`tictick-hi-api-1` 为 healthy。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=1000 node scripts/stage8-visual-smoke.mjs` 通过：`1440x900`、`812x1320`、`390x844` 三个断点的 light/dark 均覆盖 7 个页面，最大 document width 分别等于 `1440 / 812 / 390`。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SAMPLES=8 SMOKE_INTERVAL_MS=120 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：1440 桌面 chart/tv `680px`，2048 桌面 `864px`，812 窄桌面 `760px`，390 移动 `608px`，污染内部 DOM 高度后均稳定。

剩余风险：

- 仍缺像素快照基线和多浏览器视觉回归门禁；当前只用 Headless Chrome 几何 smoke 覆盖。
- 详情页列表内容密度、marker 可读性和真实长数据滚动体验还需要继续打磨。
- 项目整体仍是 `scaffold`，不能按 usable、done 或 production-safe 声明。

### 阶段 8 visual smoke 中英语言矩阵补充

目标等级：scaffold

触发问题：

- 前端基础设施审计仍标记缺全语言覆盖；既有 visual smoke 只覆盖视口和主题，没有验证中英 locale 初始化链路。
- 顶部导航、页面标题、详情页和系统页若出现 i18n key 泄漏，旧 smoke 不会失败。

Definition of Done：

- `scripts/stage8-visual-smoke.mjs` 覆盖 `zh-CN` / `en-US` 与既有视口、主题矩阵组合。
- smoke 使用应用真实存储键 `tictick-hi.locale`，触发 `LocaleStore`、`vue-i18n` 和 Naive UI provider 的真实初始化路径。
- 每页采样校验 `document.documentElement.lang` 等于目标语言。
- 中文矩阵下顶部导航必须包含 `概览` 且不能残留 `Overview`；英文矩阵下反向校验。
- 页面可见文本不得泄漏明显 i18n key，例如 `page.research.title`、`common.create`。
- 既有横向溢出、主题、研究页图表、详情页图表/双栏布局断言保持不变。

修复范围：

- `scripts/stage8-visual-smoke.mjs` 增加 `locales` 矩阵、`setLocale`、locale/nav/i18n leak 采样和断言。
- `docs/quality-audit.md` 更新前端基础设施当前风险：核心 smoke 页已有中英矩阵覆盖，但仍不等于真正全路由覆盖。

验证：

- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=800 node scripts/stage8-visual-smoke.mjs` 通过：`1440x900`、`812x1320`、`390x844` 三视口 × light/dark × `zh-CN/en-US`，当前本地数据下每组 7 页，最大 document width 分别等于 `1440 / 812 / 390`。

剩余风险：

- 语言矩阵当前覆盖 visual smoke 的核心页面和存在数据时的首个详情页，不等于所有系统管理子路由全覆盖。
- 仍缺像素快照基线、多浏览器视觉回归和 CI 硬门禁。
- 项目整体仍是 `scaffold`，不能按 usable、done 或 production-safe 声明。

### 阶段 8 K 线图表工具栏和详情页布局二次收敛

目标等级：scaffold

触发问题：

- 用户反馈研究页 K 线工具栏仍粗糙、symbol 输入过宽、图表右侧价格轴区域留白明显。
- 交易详情和回测详情虽然已改为上方图表、下方摘要/列表，但仍需要和研究页共用更紧凑的图表 gutter / 高度策略。
- 旧静态布局门禁仍锁定上一轮 CSS 数值，无法约束当前真实体验。

Definition of Done：

- 研究页工具栏在桌面尽量单行，窄桌面两行内，移动端使用横向可滚动工具条，不再竖向堆成表单。
- 研究页 symbol 输入收敛为桌面 `clamp(168px, 12vw, 210px)`、窄桌面最多 `210px`、移动端 `180px`。
- 研究页、交易详情、回测详情图表槽使用明确可读高度：桌面约 `620-720px`，窄桌面约 `600-720px`，移动端约 `500-620px`。
- 图表 gutter 收敛为左侧 `8-18px`、右侧 `2-6px`，右侧价格轴上限 `60px`。
- `stage8-visual-smoke.mjs`、`research-chart-height-smoke.mjs`、`check-research-chart-layout.sh` 同步新阈值。

修复范围：

- `web/frontend/src/pages/ResearchPage.css` 收敛任务列表高度、工具栏布局、symbol 宽度、图表高度和左右 gutter。
- `web/frontend/src/pages/detailChartLayout.css` 同步回测详情和交易详情的图表高度与 gutter。
- `web/frontend/src/components/chart/TradingViewChart.vue` 和 `web/frontend/src/theme/tokens.ts` 将右侧价格轴最小宽度收敛为 `36/40/44px`，chart 字体收敛到 `10px`。
- `scripts/stage8-visual-smoke.mjs`、`scripts/research-chart-height-smoke.mjs`、`scripts/check-research-chart-layout.sh` 更新运行态和静态断言。

验证：

- `pnpm --dir web/frontend run test -- ResearchPage.layout DetailPages.layout TradingViewChart` 通过：26 个测试文件、131 条测试。
- `pnpm --dir web/frontend run typecheck` 通过。
- `docker compose up -d --build api` 后 `http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`。
- 真实 8080 Headless Chrome 几何采样通过：
  - 1440 research：toolbar `45px`，symbol `173px`，viewport `1370x650`，gutter `16/4`，price axis `52px`。
  - 812 research：toolbar `76px`，symbol `210px`，viewport `763x720`，gutter `12/3`，price axis `52px`。
  - 390 research：toolbar `76px`，symbol `180px`，viewport `346x604`，gutter `8/2`，price axis `52px`。
  - backtest / trading detail：1440 viewport `1372x640`、812 viewport `768x620`、390 viewport `346x574`，均无横向溢出。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=500 node scripts/stage8-visual-smoke.mjs` 通过：三视口 × light/dark × `zh-CN/en-US`，当前本地数据下每组 7 页，最大 document width 分别等于 `1440 / 812 / 390`。

剩余风险：

- 当前仍是几何和 DOM 级视觉 smoke，不是像素快照基线。
- 移动端工具条采用横向滚动保留密度，后续若要做移动端专门交互，还需要单独设计。
- 项目整体仍是 `scaffold`，不能按 usable、done 或 production-safe 声明。

### 阶段 8 visual smoke 静态路由覆盖补充

目标等级：scaffold

触发问题：

- 前端基础设施仍标记缺真正全路由覆盖；旧 visual smoke 只覆盖核心页面、系统健康页和有数据时的详情页。
- 系统管理子路由、回测创建页、交易创建页未进入真实浏览器矩阵，容易在中英文、暗色、窄屏下退化但不触发门禁。

Definition of Done：

- `scripts/stage8-visual-smoke.mjs` 覆盖当前所有登录后静态路由：overview、research、backtests、backtests/new、trading、trading/new、system/notifications、system/exchange-accounts、system/operators、system/sessions、system/audit-events、system/health。
- 继续在当前环境存在数据时覆盖首个回测详情和交易详情。
- 继续保留 1440 / 812 / 390 三视口、light / dark、zh-CN / en-US 矩阵。
- 每页校验主内容 selector、主题、html lang、顶部导航翻译、可见 i18n key 泄漏、横向溢出和 runtime error。
- 不提交表单、不创建任务、不写数据库。

修复范围：

- `scripts/stage8-visual-smoke.mjs` 增加创建页和系统管理页静态路由列表。
- i18n key 泄漏检测跳过 `.audit-code` / `.session-id`，避免把审计事件码如 `auth.login` 误判为未翻译文案。
- `web/frontend/src/router/routes.test.ts` 增加 visual smoke 与登录后静态路由同步检查，后续新增静态路由未纳入 smoke 会失败。
- `docs/quality-audit.md` 更新前端基础设施风险摘要。

验证：

- `pnpm --dir web/frontend run test -- routes` 通过：26 个测试文件、132 条测试。
- `node --check scripts/stage8-visual-smoke.mjs` 通过。
- 首次新增系统审计页覆盖时复现并修正误报：`system-audit-events leaked i18n keys into visible text: auth.login`。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=500 node scripts/stage8-visual-smoke.mjs` 通过：三视口 × light/dark × `zh-CN/en-US`，当前本地数据下每组 14 页，最大 document width 分别等于 `1440 / 812 / 390`。

剩余风险：

- 当前覆盖的是登录后静态路由和有数据时的首个动态详情页，不等于所有动态详情 ID、所有空/错误/加载状态的像素基线。
- 仍缺像素快照基线、多浏览器视觉回归和 CI 硬门禁。
- 项目整体仍是 `scaffold`，不能按 usable、done 或 production-safe 声明。

### 阶段 8 K 线图表共享布局契约补充

目标等级：scaffold

触发问题：

- 研究页、回测详情、交易详情已分别修过图表高度和 gutter，但 CSS 仍分散在页面私有规则里，容易后续漂移。
- 用户反馈 K 线图表右侧仍可能出现明显空白；旧 visual smoke 只约束价格轴宽度和 renderer 宽度，没有明确约束价格轴贴近图表视口右边界。
- `tictickbot` 的图表体验采用稳定主图高度、紧凑工具栏和下方信息面板，`tictick-hi` 需要形成相同方向的可维护布局契约。

Definition of Done：

- 新增共享 `klineChartLayout.css`，研究页、回测详情、交易详情复用同一个固定图表槽、图表 viewport 和内部 chart 填充规则。
- 页面私有 CSS 只负责各自的图表高度、左右 gutter 和断点变量，不再重复声明内部 TradingView chart 填充规则。
- 详情页继续保持上方 K 线图表、下方左窄摘要、右宽 tab 列表；研究页继续保持列表在上、图表在下。
- visual smoke 对右侧价格轴增加贴近图表视口右边界断言，避免右侧大空白回归。
- 保持当前不创建任务、不写数据、不声明 production-safe。

修复范围：

- `web/frontend/src/pages/klineChartLayout.css` 抽出共享 K 线图表槽规则。
- `web/frontend/src/pages/ResearchPage.vue`、`BacktestDetailPage.vue`、`TradingDetailPage.vue` 引入共享图表槽类。
- `web/frontend/src/pages/ResearchPage.css` 和 `detailChartLayout.css` 收敛为页面变量配置。
- `ResearchPage.layout.test.ts` 和 `DetailPages.layout.test.ts` 改为检查共享契约。
- `scripts/stage8-visual-smoke.mjs` 增加右侧价格轴贴近图表 viewport 右边界断言。

验证：

- `pnpm --dir web/frontend run test -- ResearchPage.layout DetailPages.layout TradingViewChart` 通过：26 个测试文件、132 条测试。
- `node --check scripts/stage8-visual-smoke.mjs` 通过。
- `pnpm --dir web/frontend run build` 通过，产物包含 `klineChartLayout` CSS/JS chunk。
- `docker compose up -d --build api` 通过，API 使用新镜像启动；`http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=500 node scripts/stage8-visual-smoke.mjs` 通过：三视口 × light/dark × `zh-CN/en-US`，当前本地数据下每组 14 页，最大 document width 分别等于 `1440 / 812 / 390`。
- 首次完整 `scripts/quality-gate.sh` 失败于旧 `check-research-chart-layout.sh` 仍查找 `class="research-chart-body"`；同步脚本到共享 `klineChartLayout.css` 契约后，`scripts/check-research-chart-layout.sh` 和 `scripts/quality-gate.sh` 均通过。
- 通用检查通过：`go test ./...`、`go vet ./...`、`pnpm --dir web/frontend run typecheck`、`pnpm --dir web/frontend run test`、`pnpm --dir web/frontend run build`、`scripts/quality-gate.sh`。

剩余风险：

- 当前仍是几何和 DOM 级视觉 smoke，不是像素快照基线，也不是多浏览器视觉回归。
- 详情页只覆盖当前本地数据中的首个回测/交易详情，不等于所有动态详情状态、空状态和错误状态。
- 项目整体仍是 `scaffold`，不能按 usable、done 或 production-safe 声明。

### 阶段 8 空状态 / 错误状态视觉 smoke 补充

目标等级：scaffold

触发问题：

- 常规 visual smoke 主要覆盖真实数据下的路由布局，空状态和错误状态依赖当前数据库/外部接口，容易没有被稳定触发。
- 研究页、回测详情、交易详情、系统页需要证明在无数据或 API 错误时不会出现空白大面板、横向溢出、i18n key 泄漏或状态块不可见。
- 直接做像素 PNG 基线会受本地动态数据影响，本轮先建立无写操作、可重复的状态视觉 smoke。

Definition of Done：

- 新增 `scripts/stage8-state-visual-smoke.mjs`，通过 CDP Fetch 拦截 GET API 响应强制页面进入空状态和错误状态。
- 覆盖研究页任务空态/任务错误/图表错误、回测列表空态/错误、交易列表空态/错误、通知空态/错误、系统账号空态、回测详情图表空态/错误、交易详情图表空态/错误和交易详情默认列表空态。
- 覆盖桌面 `1440x900` 和移动 `390x844`，浅/深主题，`zh-CN/en-US`。
- 每个用例校验主容器、状态块可见、状态文案非空、无横向溢出、html lang、顶部导航翻译和明显 i18n key 泄漏。
- 不提交表单、不调用写接口、不创建任务、不修改数据库。

修复范围：

- `scripts/stage8-state-visual-smoke.mjs` 新增状态视觉 smoke。
- `docs/quality-audit.md` 更新前端基础设施风险摘要和本小节。

验证：

- `node --check scripts/stage8-state-visual-smoke.mjs` 通过。
- 首次脚本运行发现回测详情默认 tab 是参数快照，等待 `.backtest-detail-tabs .state-block` 不成立；已把回测详情用例收敛为稳定可见的图表空态/错误态，交易详情继续覆盖默认 tab 列表空态。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=300 node scripts/stage8-state-visual-smoke.mjs` 通过：桌面/移动 × light/dark × `zh-CN/en-US`，每组 14 个状态用例，最大 document width 分别等于 `1440 / 390`。

剩余风险：

- 这仍是 DOM/几何级状态 smoke，不是像素 PNG 快照基线。
- 只覆盖当前列出的可见空/错误状态，不覆盖所有隐藏 tab、所有表单校验状态和所有动态详情 ID。
- 仍缺多浏览器视觉回归和 CI 硬门禁。
- 项目整体仍是 `scaffold`，不能按 usable、done 或 production-safe 声明。

### 阶段 8 一键 smoke 浏览器视觉门禁接入

执行时间：2026-06-29

目标等级：scaffold

触发问题：

- 已有 `scripts/stage8-visual-smoke.mjs` 和 `scripts/stage8-state-visual-smoke.mjs` 能发现研究页、回测详情、交易详情和空/错误状态的布局回归，但原先需要人工单独运行。
- `scripts/stage8-smoke.sh` 是 Stage 8 全链路验收入口，如果不默认串联浏览器视觉 smoke，页面布局问题容易在全链路 smoke 通过后继续漏出。
- 首次重跑 Stage 8 smoke 暴露脚本自身窗口语义不严谨：同步 seed 窗口闭合到 `2026-01-01T02:00:00Z`，但 5m 回测直接把 `endTime=02:00` 传给 CandleProvider，provider 按 inclusive `open_time` 闭区间正确报告缺少 `02:00` open 的尾部 K 线。

Definition of Done：

- `scripts/stage8-smoke.sh` 在研究、回测、模拟盘、通知和系统健康全部通过后，默认执行常规视觉 smoke 和状态视觉 smoke。
- 无 Chrome 环境必须显式设置 `STAGE8_BROWSER_SMOKE=0` 才能跳过，且跳过会降低验收强度。
- 常规视觉 smoke 支持接收本轮 Stage 8 创建的 backtest / trading task ID，详情页检查不再依赖数据库里的首条旧记录。
- Stage 8 smoke 的回测任务窗口和 CandleProvider inclusive `open_time` 语义一致，不用错误的 `endTime` 制造假缺口。
- 项目整体等级不升级，仍为 `scaffold`。

修复范围：

- `scripts/stage8-smoke.sh` 增加 `run_browser_visual_smokes`，默认串联 `stage8-visual-smoke.mjs` 和 `stage8-state-visual-smoke.mjs`。
- `scripts/stage8-smoke.sh` 增加 `STRATEGY_END_TIME=2026-01-01T01:55:00Z`，保留 data sync seed close boundary `END_TIME=2026-01-01T02:00:00Z`。
- `scripts/stage8-visual-smoke.mjs` 支持 `SMOKE_BACKTEST_ID` / `SMOKE_TRADING_TASK_ID`。
- README 标明 Stage 8 smoke 默认运行浏览器视觉 smoke 和显式跳过方式。

验证：

- `bash -n scripts/stage8-smoke.sh` 通过。
- `node --check scripts/stage8-visual-smoke.mjs` 通过。
- `node --check scripts/stage8-state-visual-smoke.mjs` 通过。
- 首次 `scripts/stage8-smoke.sh` 失败于 backtest `bt_bc72c740376be5e74156f13a`，错误为 `candle data health is gap`；已确认原因是 smoke 回测窗口使用了最后一根 5m K 线 close boundary 而不是 open boundary。
- 修正后 `scripts/stage8-smoke.sh` 通过，证据：symbol `S81782748166USDT`、data task `dst_ac88c47f5085ec975b7e639e`、backtest `bt_3ce8fd3ad92f729d11c094c8`、paper execute `tt_c92f4b546e925c9b9840caa8`、paper notify `tt_fbc8ae35a33790fa48932800`、notification channel `stage8-smoke-1782748166`。
- 串联的 visual smoke 覆盖 1440 / 812 / 390 视口、浅/深主题、`zh-CN/en-US`，每组 14 页面，通过。
- 串联的 state visual smoke 覆盖 1440 / 390 视口、浅/深主题、`zh-CN/en-US`，每组 14 个状态用例，通过。

失败：

- 首次 Stage 8 smoke 的假缺口失败已修复并复跑通过。

剩余风险：

- 浏览器 smoke 仍是 DOM / 几何级断言，不是人工审批的像素快照基线。
- 仍只覆盖 Chrome/CDP，不覆盖 WebKit/Firefox 或真实用户浏览器长时间 soak。
- `STAGE8_BROWSER_SMOKE=0` 允许无 Chrome 环境跳过浏览器验收；跳过时不能把页面布局风险视为关闭。
- 项目整体仍是 `scaffold`，不能按 usable、done 或 production-safe 声明。

### 阶段 0 完整本地质量门禁入口补充

执行时间：2026-06-29

目标等级：scaffold

触发问题：

- `scripts/quality-gate.sh` 是轻量静态门禁，只覆盖文件大小、交易事实浮点边界、策略边界、API contract、图表静态布局契约和若干 scaffold marker。
- 协议要求的完整验证命令仍需要人工逐条执行，容易出现漏跑 `go vet`、前端测试或生产构建的情况。
- Stage 8 smoke 和 SIGTERM smoke 已经存在，但属于 Docker / Chrome 重型验收，不能无条件塞进所有本地快速门禁。

Definition of Done：

- 新增一个完整本地质量门禁入口，默认按固定顺序执行协议要求的通用检查。
- 脚本在某项失败后继续跑后续检查，最后统一返回失败，便于一次看到多个问题。
- Stage 8 full-chain smoke 和 SIGTERM smoke 通过显式环境变量开启，不改变轻量门禁职责。
- README 说明轻量门禁、完整门禁和重型 Stage 8 smoke 的边界。
- 不改变业务功能、不新增 migration、不升级任何模块等级。

修复范围：

- 新增 `scripts/full-quality-gate.sh`。
- README 增加完整本地质量门禁入口和 `FULL_QUALITY_STAGE8=1 FULL_QUALITY_SIGTERM=1` 重型验收开关说明。
- `docs/quality-audit.md` 更新质量门禁评级摘要和本小节。

验证：

- `bash -n scripts/full-quality-gate.sh` 通过。
- `scripts/full-quality-gate.sh` 通过，覆盖 `go test ./...`、`go vet ./...`、`pnpm --dir web/frontend run typecheck`、`pnpm --dir web/frontend run test`、`pnpm --dir web/frontend run build`、`scripts/quality-gate.sh`。

剩余风险：

- 默认完整门禁仍不包含 Docker / Chrome 重型 Stage 8 smoke；需要发布前或 Stage 8 变更时显式设置 `FULL_QUALITY_STAGE8=1` 和按需 `FULL_QUALITY_SIGTERM=1`。
- 仍缺远程 CI 硬门禁、像素快照基线和多浏览器视觉回归。
- 项目整体仍是 `scaffold`，不能按 usable、done 或 production-safe 声明。

### 阶段 0 远程 CI 质量门禁补充

执行时间：2026-06-30

目标等级：scaffold

触发问题：

- 上一轮已新增 `scripts/full-quality-gate.sh`，但仍只靠本地人工执行。
- 阶段 0 质量底座要求建立可持续质量门禁；没有远程 CI 时，推送和 PR 仍可能绕过通用检查。

Definition of Done：

- 新增 GitHub Actions workflow，在 pull request、`main` push 和手动触发时运行默认完整质量门禁。
- CI 使用 `go.mod` 的 Go 版本和前端 `packageManager` 指定的 pnpm 版本，安装 `ripgrep` 和前端依赖，再运行 `scripts/full-quality-gate.sh`。
- CI 默认不执行 Docker / Chrome 重型 Stage 8 smoke，避免把本地完整门禁和环境依赖更强的验收混在一起。
- README 说明远程 CI 与本地完整门禁的关系。
- 不改业务功能、不新增 migration、不升级模块等级。

修复范围：

- 新增 `.github/workflows/quality.yml`。
- README 增加 GitHub Actions 默认 full gate 说明。
- `docs/quality-audit.md` 更新质量门禁评级摘要和本小节。

验证：

- `scripts/full-quality-gate.sh` 本地通过，覆盖 `go test ./...`、`go vet ./...`、前端 typecheck、前端 test、前端 build 和 `scripts/quality-gate.sh`。
- 首次远程 run `28385821859` 失败，原因是 GitHub runner 缺少 `rg`，导致 `check-trading-floats.sh`、`check-strategy-boundary.sh` 和 scaffold marker 检查无法运行；已在 workflow 中安装 `ripgrep`。
- 修正后远程 run `28386293502` 通过：`Quality Gate / Full local quality gate`，耗时约 1m30s。
- `git diff --check` 通过。

剩余风险：

- GitHub Actions 日志有 Node 20 deprecation annotation，原因是 `actions/checkout@v4`、`actions/setup-go@v5`、`actions/setup-node@v4` 被 runner 强制以 Node 24 执行；当前不阻断门禁，但后续需要关注 actions 主版本升级。
- CI 默认仍不跑 `FULL_QUALITY_STAGE8=1` / `FULL_QUALITY_SIGTERM=1` 重型 Docker smoke；Stage 8 和 worker shutdown 相关变更仍需显式运行。
- 仍缺像素快照基线和多浏览器视觉回归。
- 项目整体仍是 `scaffold`，不能按 usable、done 或 production-safe 声明。

### 阶段 0 Stage 8 heavy smoke workflow 补充

执行时间：2026-06-30

目标等级：scaffold

触发问题：

- 默认 GitHub Actions full gate 已覆盖通用检查，但不会运行 Docker / Chrome 重型 Stage 8 smoke。
- `scripts/stage8-smoke.sh` 和 `scripts/stage8-sigterm-smoke.sh` 仍靠本地人工执行，发布前或全链路相关变更有漏跑风险。
- Stage 8 smoke 依赖 Docker Compose、Chrome 和本地 `.env`，需要与普通 PR gate 分离。

Definition of Done：

- 新增独立 GitHub Actions workflow，支持手动触发 full-chain smoke、SIGTERM smoke，并按固定周计划定时跑两者。
- workflow 使用 `.env.example` 生成 CI `.env`，并关闭 instrument catalog 自动同步，降低对真实交易所网络的依赖。
- workflow 显式检测 runner 上的 Chrome 可执行文件，避免 browser visual smoke 在缺 Chrome 时进入不明确失败。
- workflow 结束后清理 Docker Compose volume 和 orphan 容器。
- README 说明手动触发入口。
- 不把重型 smoke 默认接入普通 PR / push full gate，不改业务功能、不新增 migration、不升级模块等级。

修复范围：

- 新增 `.github/workflows/stage8-heavy-smoke.yml`。
- README 增加 `gh workflow run "Stage 8 Heavy Smoke"` 入口。
- `docs/quality-audit.md` 更新质量门禁评级摘要和本小节。

验证：

- 待执行：YAML 解析、本地 `scripts/full-quality-gate.sh`、远程手动触发 Stage 8 heavy smoke。

剩余风险：

- 该 workflow 仍只覆盖 GitHub Ubuntu + Chrome 环境，不等于多浏览器视觉回归或生产部署验收。
- 定时运行会占用较多 CI 分钟；失败需要按 smoke 日志回溯具体全链路问题。
- 项目整体仍是 `scaffold`，不能按 usable、done 或 production-safe 声明。

### 阶段 8 K 线图表生产布局第三次收口

目标等级：scaffold

触发问题：

- 用户继续反馈研究页 K 线图表工具栏不够精细、symbol 输入仍显得过宽、右侧价格轴区域空白明显。
- 用户要求交易详情、回测详情同样按生产级图表布局收敛，不能只修研究页。
- 本地 `8080` 由 Docker API 镜像服务静态资源，只执行前端 build 不会更新用户可见页面；必须重建并替换 API 镜像后再做真实采样。

Definition of Done：

- 研究页同步任务列表仍在上方，K 线图表在下方，图表成为主要阅读区域，不再按当前首屏硬平分高度。
- 研究页工具栏改为紧凑 market strip + 状态行，symbol 输入桌面实际宽度不超过 `200px`。
- 研究页、回测详情、交易详情复用共享 K 线图表槽；研究页真实 plot height 为 `700px`，详情页真实 plot height 为 `680px`。
- 图表左侧保留读图 gutter，右侧外层 gutter 收敛到 `1-2px`；右侧价格轴保持贴近图表视口边界，宽度不超过 `56px`。
- 回测详情和交易详情保持上方大图表、下方左窄摘要、右宽 tab 列表。
- 仍不新增业务功能、不改 API 契约、不声明 production-safe。

修复范围：

- `web/frontend/src/pages/ResearchPage.css` 改为单列 toolbar 布局、紧凑 source controls、状态行、桌面/窄桌面/移动独立图表高度和 gutter。
- `web/frontend/src/components/market/MarketSymbolAutoComplete.vue` 增加 root 宽度边界，避免 scoped 样式把页面级 symbol 宽度约束冲掉。
- `web/frontend/src/pages/detailChartLayout.css` 同步详情页大图高度和 gutter。
- `web/frontend/src/components/chart/TradingViewChart.vue` 收紧 right price scale minimumWidth 为 `34/38/42px`，启用 `entireTextOnly`，并把 `rightOffsetPixels` 收敛到 `8px`。
- `web/frontend/src/components/chart/TradingViewChart.css` 使用显式 `top/left` 定位 canvas 宿主，不再用 `inset: 0`。
- `scripts/stage8-visual-smoke.mjs` 收紧 symbol / price-axis / 图表高度阈值，并在每个 Chrome target 完成后关闭页面，减少矩阵 smoke 资源累积。
- `ResearchPage.layout.test.ts`、`DetailPages.layout.test.ts`、`TradingViewChart.test.ts` 同步当前布局契约。

验证：

- `pnpm --dir web/frontend exec vitest run src/pages/ResearchPage.layout.test.ts src/pages/DetailPages.layout.test.ts src/components/chart/TradingViewChart.test.ts src/components/market/MarketSymbolAutoComplete.test.ts` 通过：4 个测试文件、38 条测试。
- `node --check scripts/stage8-visual-smoke.mjs` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `docker compose up -d --no-deps --build api` 通过，并确认 `tictick-hi-api-1` health 为 `healthy`。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：26 个测试文件、132 条测试。
- `scripts/quality-gate.sh` 通过。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SAMPLES=8 SMOKE_INTERVAL_MS=120 SMOKE_SETTLE_MS=800 node scripts/research-chart-height-smoke.mjs` 通过：1440 / 2048 / 812 / 390 四视口连续采样均稳定，document、panel、body、chart、tv 高度不增长。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=500 node scripts/stage8-visual-smoke.mjs` 通过：三视口 × light/dark × `zh-CN/en-US`，每组 14 页，最大 document width 分别等于 `1440 / 812 / 390`。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=300 node scripts/stage8-state-visual-smoke.mjs` 通过：桌面/移动 × light/dark × `zh-CN/en-US`，每组 14 个状态用例，最大 document width 分别等于 `1440 / 390`。
- 真实 `http://127.0.0.1:8080/research` Headless Chrome 1440x900 几何采样：toolbar `80px`，symbol `173px`，chart viewport `1366x700`，frame gutter `22/2`，price axis `52px`，document width `1440`。
- 真实回测详情采样：chart panel `1392x706`，viewport `1364x680`，summary `340px`，tabs `1036px`，price axis `46px`，document width `1440`。
- 真实交易详情采样：chart panel `1392x706`，viewport `1364x680`，summary `340px`，tabs `1036px`，price axis `46px`，document width `1440`。
- 截图产物用于人工核对：`/tmp/tictick-hi-research-layout.png`、`/tmp/tictick-hi-trading-detail-layout.png`。

剩余风险：

- 本轮仍是 DOM 几何和人工截图核对，不是像素快照基线。
- 只核对了当前本地数据下首个回测详情和交易详情，不等于所有动态详情状态。
- 移动端和窄桌面仍需完整矩阵 smoke 继续守护；本轮不能把前端整体升级为 usable。
- 项目整体仍是 `scaffold`，不能按 usable、done 或 production-safe 声明。

### 阶段 8 K 线图表生产布局第四次收口

目标等级：scaffold

触发问题：

- 研究页图表高度变量曾声明在 `.research-chart-panel` 父级，但共享 `.kline-chart-frame` 在真实 frame 元素上声明了默认高度，导致运行态仍读到默认 `640px`。
- 用户继续反馈图表区域过矮、工具栏 symbol 输入过宽、右侧留白明显；这些问题必须通过真实浏览器 smoke 验证，而不是只看源码。
- 旧验收只证明“不会无限拉高”，没有把研究页、回测详情、交易详情的真实右侧价格轴贴合和工具栏宽度作为本地 8080 几何指标。

Definition of Done：

- 研究页图表高度变量声明在真实 frame 元素 `.research-chart-body` 上，避免被共享 frame 默认变量覆盖。
- 研究页 plot height：1440 桌面 `700px`、2048 宽桌面 `760px`、812px 窄桌面 `680px`、390px 移动 `560px`，且连续采样不增长。
- symbol 输入宽度收敛为桌面 `174px`、窄桌面 `166px`、移动 `158px`。
- 研究页、回测详情、交易详情右侧外层 gutter 为 `0px`；右侧价格轴贴近图表 viewport 边界且宽度不超过 `54px`。
- 主图 canvas 右边界必须贴住右侧价格轴左边界，防止右侧大空白回归。
- 回测详情和交易详情继续保持上方大图表、下方左窄摘要右宽 tab 列表。

修复范围：

- `web/frontend/src/pages/ResearchPage.css` 把 K 线高度和 gutter 变量迁移到 `.research-chart-body`，并收敛 symbol 输入宽度。
- `web/frontend/src/pages/detailChartLayout.css` 同步详情页图表高度到桌面 `700-760px`、窄桌面/移动 `680px`。
- `web/frontend/src/pages/klineChartLayout.css` 保持共享 frame 固定高度、左侧读图 gutter 和右侧 `0px` 紧凑 gutter。
- `web/frontend/src/components/chart/TradingViewChart.vue` 增加渲染高度上限并收紧右侧 price scale / time scale 偏移到 `0px`。
- `scripts/check-research-chart-layout.sh` 和 `TradingViewChart.test.ts` 同步当前契约。
- `scripts/research-chart-height-smoke.mjs` 增加主图 canvas 与右侧价格轴贴合断言。
- `scripts/stage8-visual-smoke.mjs` 继续收紧 symbol 宽度、右侧价格轴宽度和图表高度阈值。

验证：

- `scripts/check-research-chart-layout.sh` 通过。
- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts` 通过：1 个测试文件、21 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `docker compose up -d --build migrate api` 通过，`http://127.0.0.1:8080` 已服务当前镜像内前端资产。
- `SMOKE_SAMPLES=3 SMOKE_INTERVAL_MS=120 SMOKE_SETTLE_MS=900 node scripts/research-chart-height-smoke.mjs` 通过：1440x900 图表 `700px`、2048x1152 图表 `760px`、812x1320 图表 `680px`、390x844 图表 `560px`，连续采样高度稳定。
- 定向 CDP layout smoke 通过：
  - 1440 research：viewport `1372x700`、symbol `174px`、price axis `52px`。
  - 1440 backtest detail / trading detail：viewport `1372x700`、summary `320px`、tabs `1056px`。
  - 812 research：viewport `764x680`、symbol `166px`、price axis `52px`。
  - 812 backtest detail / trading detail：viewport `764x680`，下方 summary / tabs 堆叠同宽。
  - 390 research：viewport `344x560`、symbol `158px`、price axis `52px`。
  - 390 backtest detail / trading detail：viewport `342x680`，下方 summary / tabs 堆叠同宽。
- `SMOKE_SETTLE_MS=300 node scripts/stage8-visual-smoke.mjs` 通过：1440 / 812 / 390 三视口 × light/dark × `zh-CN/en-US`，每组 14 页，最大 document width 分别等于 `1440 / 812 / 390`。

剩余风险：

- 本轮仍是 DOM 几何和运行态 smoke，不是像素快照基线。
- 浏览器 smoke 使用当前本地数据库中的详情页样本；不能代表所有未来长文本、长列表和极端数据状态。
- 全矩阵 visual smoke 本轮使用较短 settle 通过；后续仍需引入像素快照基线和更多动态详情状态。
- 项目整体仍是 `scaffold`，不能按 usable、done 或 production-safe 声明。

### 阶段 8 K 线图表工具栏密度第五次收口

目标等级：scaffold

触发问题：

- 用户继续反馈研究页 K 线图表工具栏不够精细，symbol 输入仍显得过宽。
- 上一轮虽然证明图表高度稳定、右侧价格轴贴边，但 `scripts/check-research-chart-layout.sh` 和 visual smoke 对工具栏宽度的约束仍偏松。

Definition of Done：

- 保持研究页“同步任务列表在上、K 线图表在下”的结构。
- 保持研究页图表 plot height：桌面 `700-760px`、窄桌面 `680px`、移动端 `560px`。
- 研究页工具栏 source controls 收敛为 `112px 156px 68px max-content`，symbol 总宽桌面 `156px`、窄桌面 `148px`、移动端 `142px`。
- `ResearchWindowControls` 按钮从 `30px / 10px` 收敛到 `28px / 8px`，避免窗口按钮组撑宽工具条。
- visual smoke 的 symbol 宽度阈值从 `180px` 收紧为 `160px`。
- 不改数据同步、CandleProvider、回测、交易 runner 或 API 契约，不声明 usable / production-safe。

修复范围：

- `web/frontend/src/pages/ResearchPage.css` 收紧 exchange / symbol / interval 工具栏网格和响应式宽度。
- `web/frontend/src/components/research/ResearchWindowControls.vue` 收紧窗口按钮尺寸。
- `web/frontend/src/pages/ResearchPage.layout.test.ts` 同步源码布局契约。
- `scripts/check-research-chart-layout.sh` 同步轻量质量门禁。
- `scripts/stage8-visual-smoke.mjs` 收紧 symbol 输入最大宽度阈值。

验证：

- `pnpm --dir web/frontend exec vitest run src/pages/ResearchPage.layout.test.ts src/pages/DetailPages.layout.test.ts src/components/chart/TradingViewChart.test.ts` 通过：3 个测试文件、34 条测试。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `docker compose up -d --build migrate api` 通过，`http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`。
- `CHROME_REMOTE_DEBUGGING_PORT=9243 SMOKE_SETTLE_MS=800 SMOKE_SAMPLES=2 scripts/research-chart-height-smoke.mjs` 通过：1440/2048/812/390 四视口高度稳定。
- `BASE_URL=http://127.0.0.1:8080 CHROME_REMOTE_DEBUGGING_PORT=9242 SMOKE_SETTLE_MS=800 scripts/stage8-visual-smoke.mjs` 通过：1440/812/390 三视口 × light/dark × `zh-CN/en-US`，每组 14 页。
- `pnpm --dir web/frontend run test` 通过：26 个测试文件、134 条测试。
- `scripts/quality-gate.sh` 通过。
- 真实 8080 Headless Chrome 1440x900 几何采样：source controls `568px`、symbol `156px`、chart viewport `1372x700`、right price axis `52px`、document width `1440px`。

剩余风险：

- 本轮仍是 DOM 几何、运行态 smoke 和人工截图核对，不是像素快照基线。
- 图表指标、绘图工具、成交点联动和完整交易分析能力仍未补齐。
- 项目整体仍是 `scaffold`，不能按 usable、done 或 production-safe 声明。

### 阶段 1 CandleProvider 流式大窗口聚合补充

目标等级：demo

触发问题：

- 阶段 1 审计仍记录 CandleProvider 聚合基础窗口最多 12 页 / 60000 根 `1m`，`4h` / `1d` 默认研究窗口会被过早标记 `limitedByBaseWindow`。
- 旧实现会先收集全部基础 `1m` K 线，再做缺口检测和聚合；继续增大上限会放大内存压力，不符合后续 usable 方向。

Definition of Done：

- 高周期聚合 fallback 改为分页流式读取基础 `1m`：每页校验、流式追踪缺口、流式聚合，不再为了生成高周期窗口一次性持有全部基础 K 线。
- 默认 `1d limit=1000` 需要的 1440000 根基础 `1m` 不再因旧 60000 上限被截断。
- coverage 继续返回 `requiredBaseCandles`、`baseLimit`、`returnedBaseCandles` 和 `limitedByBaseWindow`。
- native / aggregated / gap / invalid / pagination 行为保持兼容。
- 不新增 migration，不改变 `market_candles` 事实语义，不升级模块到 usable。

修复范围：

- `internal/data/candle_provider_aggregation.go` 将聚合 fallback 收敛为有界流式分页聚合，默认上限为 288 页 / 1440000 根基础 `1m`。
- `internal/data/candle_provider.go` 改为消费流式聚合结果，沿用原有 invalid、gap、coverage 和 fallback 语义。
- `internal/data/candle_provider_test.go` 新增 generated store，覆盖 60000 根以上分页聚合、默认 `1d limit=1000` 流式聚合和受限 coverage。
- `internal/store/postgres/candle_provider_perf_integration_test.go` 将真实性能 smoke 从 60000 根 `1m -> 1h` 提升为 240000 根 `1m -> 4h`。

验证：

- `go test ./internal/data -run TestCandleProvider -count=1` 通过。
- `go test ./internal/data -run 'TestCandleProvider|TestValidateCandleSeries' -count=1` 通过。
- `go test ./internal/data ./internal/store/postgres -run 'TestCandleProvider|TestIntegrationCandleProvider|TestIntegrationListNativeCandles' -count=1` 通过。
- `scripts/check-file-size.sh` 通过。
- `scripts/stage1-candle-provider-perf-smoke.sh` 通过；真实 PostgreSQL 中读取 240000 根基础 `1m` 并聚合成 1000 根 `4h`，查询耗时 `2.170233251s`。

剩余风险：

- 1440000 根仍是有界读取上限，不是无限历史查询方案。
- `1d limit=5000` 等更大窗口仍需要聚合缓存、分段查询或预取策略。
- 本轮没有做长期/并发压测，也没有新增持久化聚合缓存。
- CandleProvider 和阶段 1 仍是 `demo`，项目整体仍是 `scaffold`。

### 阶段 1 CandleProvider 并发聚合读取 smoke 补充

执行日期：2026-06-30

目标等级：scaffold。

范围内：

- 新增 PostgreSQL 集成测试 `TestIntegrationCandleProviderConcurrentAggregationQueries`。
- 测试在真实 `market_candles` 表写入 2880 根连续 `1m` K 线。
- 6 个 goroutine 并发调用 `Store.GetCandles` 请求 `15m limit=96` 聚合窗口。
- 断言每个并发请求都返回 `source=aggregated`、`health=ok`、`baseInterval=1m`、96 根 K 线、1440 根基础 K 线 coverage、一致的窗口边界和上一页 pagination metadata。

范围外：

- 不改变 CandleProvider 查询语义、分页语义、聚合算法、API contract 或数据库 schema。
- 不新增聚合缓存、预取策略、虚拟化或分段历史查询方案。
- 不做长时间 soak、冷缓存压测、多租户/多 symbol 混合负载压测或真实生产数据分布压测。

验证：

- 本地 `go test ./internal/store/postgres -run TestIntegrationCandleProviderConcurrentAggregationQueries -count=1 -v` 因未设置 `TICTICK_TEST_DATABASE_URL` 跳过，编译通过。
- Docker Compose PostgreSQL 集成测试通过：`docker run --rm --network tictick-hi_default -v "$PWD":/src -w /src -e TICTICK_TEST_DATABASE_URL='postgresql://tictick:tictick-local-postgres-password@postgres:5432/tictick_hi?sslmode=disable' golang:1.26-bookworm go test ./internal/store/postgres -run TestIntegrationCandleProviderConcurrentAggregationQueries -count=1 -v`。

剩余风险：

- 该测试只是短时并发 smoke，用于证明当前聚合读取路径在真实 PostgreSQL 下可以并发返回一致结果。
- CandleProvider 仍缺长期 soak、冷缓存、真实生产数据分布、超过 1440000 根基础 K 线窗口和缓存/分段策略证据；项目整体仍是 `scaffold`。

### 阶段 1 instrument catalog 自动暂停恢复语义补充

目标等级：demo

触发问题：

- 既有 `ReplaceMarketInstruments` 已能在交易所 catalog 变为 inactive / missing 时自动暂停 data sync task，但没有保存暂停前的 sync / realtime 期望。
- 当交易所 catalog 后续恢复 active 时，系统无法区分用户手动暂停和 catalog 自动暂停，也无法自动恢复原任务期望。
- 这会让阶段 1 的“研究页数据源健康可观察”停在单向暂停，仍需要用户新建任务或手动猜测恢复动作。

Definition of Done：

- catalog 自动暂停时保存原 `sync_enabled` / `realtime_enabled` 期望，并把 `last_error` 写为 `market instrument is not active in catalog`。
- catalog 恢复 active 时，只恢复带 catalog pause 标记的 paused data sync task；用户手动暂停、failed、cancelled、succeeded 不自动恢复。
- 用户手动 start / stop data sync task 会清除 catalog pause 标记，用户操作优先于自动恢复。
- `POST /api/market/instruments/sync` 返回 `restoredDataSyncTaskCount`，和已有 `pausedDataSyncTaskCount` 一起可观察。
- 不删除任务、不删除 K 线、不处理回测 / 交易任务，不把阶段 1 升级为 usable。

修复范围：

- 新增 migration `0031_data_sync_catalog_pause_restore.sql`，为 `data_sync_tasks` 增加 `market_pause_sync_enabled` 和 `market_pause_realtime_enabled`。
- `internal/store/postgres/market_store.go` 在同一事务中完成 catalog upsert、非 active 自动暂停和 active 恢复。
- `internal/store/postgres/store.go` 在用户手动 sync / realtime start-stop 时清除 catalog pause 标记。
- `MarketInstrumentSyncResult`、API contract、前端 app 类型和 market API 测试增加 `restoredDataSyncTaskCount`。
- fake repository 增加测试用 catalog pause state，覆盖 API route 的 paused / restored 计数。

验证：

- `go test ./internal/store/postgres -run 'TestIntegrationReplaceMarketInstruments|TestIntegrationClaimDataSyncTaskSkipsInactiveMarketInstrument' -count=1` 通过。
- `go test ./internal/web/api -run 'TestMarketInstrumentSyncRoute|TestAPIContract|TestFrontendAPI|TestWriteGeneratedFrontendAPITypes|TestFrontendAPIGeneratedTypesAreCurrent' -count=1` 通过。
- `pnpm --dir web/frontend exec vitest run src/services/api/market.test.ts src/components/market/MarketSymbolAutoComplete.test.ts` 通过：2 个测试文件、6 条测试。

剩余风险：

- `inactive` 仍是粗粒度内部状态，没有区分停牌、退市、只撤单、只减仓或迁移窗口。
- 本轮只处理 data sync task，不处理既有 backtest / trading task 的 catalog 状态处置。
- 恢复仍依赖下一次 instrument catalog sync，不是交易所事件驱动。
- 阶段 1 仍是 `demo`，项目整体仍是 `scaffold`。

### 阶段 1 数据同步任务 invalid 补同步入口补充

复核时间：2026-06-30

目标等级：scaffold 增量。

背景：

- 研究页已经能在任务表和异常详情弹窗中展示历史异常 K 线，但用户只能查看，不能从同一研究工作流排补同步任务。
- 历史异常行不能由前端直接清洗，也不能绕过后端检测任意创建修复窗口；补数必须继续走 `data_sync_tasks` 和 worker 同一套同步逻辑。

Definition of Done：

- `GET /api/data/tasks/{id}/invalid-issues` 保持只列出后端检测出的 invalid candle issue。
- 新增 `POST /api/data/tasks/{id}/repair-invalid-issues`，只按异常类型和时间范围过滤当前真实 invalid issues，并为每个 issue 的 `open_time` 创建 `[open_time, open_time + interval]` 补同步任务。
- 补同步任务必须写入 `repair_source_task_id`，`sync_enabled=true`，`realtime_enabled=false`，不直接修改或删除 `market_candles`。
- 已存在同窗口 active 补同步任务时返回 `skippedExisting`，不重复创建。
- 研究页异常详情弹窗能按当前筛选排队修复异常，并刷新任务列表。
- 不新增 migration，不清洗历史异常行，不做自动后台修复，不把阶段 1 或项目整体升级为 usable / production-safe。

改动范围：

- `internal/data/data_sync_model.go` / `internal/data/model.go` 增加 invalid repair request 和 repository 方法。
- `internal/store/postgres/sync_invalid_issue_store.go` 增加 invalid issue repair 的事务逻辑。
- `internal/web/api/data_handlers.go`、API contract 和 fake repository 增加新路由。
- `web/frontend/src/components/research/ResearchTaskInvalidIssueModal.vue` 增加“排队修复当前异常”入口。
- `web/frontend/src/services/api/data.ts`、`web/frontend/src/types/api.generated.ts`、`web/frontend/src/types/app.ts` 同步新 API 类型。
- `web/frontend/src/i18n/messages.research.zh.ts` / `messages.research.en.ts` 拆出研究页文案，避免中英文主 messages 文件超过阶段 0 文件规模门禁。

验证：

- `go test ./internal/web/api -run 'TestDataSyncTaskRoutes|TestAPIContract|TestAPIMethodNotAllowedContracts|TestFrontendAPI' -count=1` 通过。
- `go test ./internal/store/postgres -run 'TestIntegrationListDataSyncTasksReportsInvalidCandleHealth|TestIntegrationRepairDataSyncTaskInvalidIssuesConvergesSourceHealth' -count=1` 通过。
- `pnpm --dir web/frontend exec vitest run src/services/api/data.invalid.test.ts src/components/research/ResearchTaskInvalidIssueModal.test.ts src/pages/ResearchPage.layout.test.ts` 通过：3 个测试文件、17 条测试。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `cd web/frontend && pnpm run typecheck` 通过。
- `cd web/frontend && pnpm run test` 通过：26 个测试文件、134 条测试。
- `cd web/frontend && pnpm run build` 通过。
- `scripts/quality-gate.sh` 通过。

失败项：

- 首次 `scripts/quality-gate.sh` 暴露 `messages.zh.ts` / `messages.en.ts` 超过 400 行硬上限；已通过拆分研究页文案修复，并复跑通过。

剩余风险：

- 本轮只把 invalid issue 变成可排补同步任务；真实补数是否成功仍依赖 exchange adapter、worker 和外部网络。
- 不自动清洗历史异常行，不自动重试 invalid repair，也不把 invalid 行纳入全历史缺口扫描。
- 阶段 1 研究核心仍不能升级为 usable；项目整体仍是 `scaffold`。

### 阶段 1 K 线图表布局与全历史异常扫描复核补充

执行时间：2026-06-30

触发问题：

- 用户反馈研究页工具栏控件粗糙、symbol 输入过宽、图表高度过窄、右侧价格轴区域过宽，且交易详情和回测详情存在同类布局问题。
- 用户继续反馈同步任务列表虽可见缺口，但全历史异常 K 线没有从研究页直接可见。
- 既有前端契约曾把研究页右侧图表 gutter 写死为 `0px` 或过宽值，并把图表高度和工具栏布局散落在页面样式中，容易在不同视口下回归。

修复范围：

- 研究页图表工具栏改为两段式紧凑工作条：上方只放交易所、交易对、刷新、周期、窗口控制，下方放当前数据源和数据状态标签。
- 研究页 symbol 输入收敛为桌面 `120px`、窄桌面 `112px`、移动端 `108px`，`stage8-visual-smoke.mjs` 将桌面最大宽度阈值收紧到 `124px`。
- 研究页、交易详情、回测详情共享 `klineChartLayout.css` 固定图表槽，plot 高度收敛为桌面 `clamp(660px, 66vh, 800px)`、窄桌面 `680px`、移动端 `540px`。
- 图表左右 gutter 收敛为桌面 `14px/8px`、窄桌面 `12px/8px`、移动端 `10px/10px`，不再使用右侧 `0px` 或过宽右侧留白旧契约。
- `TradingViewChart` 的成交量柱改用隐藏的 `volume` price scale，避免 overlay scale 参与右侧坐标区布局；右侧价格轴 minimumWidth 收敛为 34/38/40px，并继续由 browser smoke 限制不超过 54px。
- `TradingViewChart.css` 只让 `.tv-lightweight-charts` 外层填满固定 viewport，继续禁止覆写 lightweight-charts 内部 table / tbody / tr / td / canvas 几何。
- 新增 `GET /api/market/candle-invalid-issues`，按 exchange / symbol / interval 全历史扫描已落库 `market_candles` 的真实 OHLCV 异常，返回窗口、总数、返回数和 limited metadata。
- 研究页新增“全历史无异常 / 全历史异常 N 根”状态标签，可打开详情弹窗查看异常时间、类型和原因；该入口只观察，不自动清洗历史数据。
- `ResearchPage.layout.test.ts`、`DetailPages.layout.test.ts`、`check-research-chart-layout.sh`、`research-chart-height-smoke.mjs`、`stage8-visual-smoke.mjs` 和前端 API / 组件测试同步为新的布局与异常扫描 contract。

验证：

- `scripts/generate-api-types.sh` 通过。
- `go test ./internal/data ./internal/web/api -run 'TestMarketCandleInvalidIssue|TestAPIContract|TestAPIMethodNotAllowedContracts|TestFrontendAPI' -count=1` 通过。
- `go test ./internal/store/postgres -run 'TestIntegrationScanMarketCandleInvalidIssues' -count=1` 通过。
- `pnpm --dir web/frontend exec vitest run src/services/api/data.test.ts src/services/api/marketCandle.test.ts` 通过：2 个测试文件、13 条测试。
- `pnpm --dir web/frontend exec vitest run src/components/research/MarketCandleInvalidIssueTag.test.ts src/pages/ResearchPage.layout.test.ts src/pages/DetailPages.layout.test.ts src/components/chart/TradingViewChart.test.ts` 通过。
- `scripts/check-research-chart-layout.sh` 通过。
- `scripts/research-chart-height-smoke.mjs` 通过：`1440x900` 图表 `660px`、`2048x1152` 图表 `760px`、`812x1320` 图表 `680px`、`390x844` 图表 `540px`，污染内部高度后 document/panel/body/chart/tv 高度均稳定。
- 本地 CDP 几何复核通过：研究页 symbol 输入 `120px`，图表 viewport / tv `1368x660`，价格轴约 `52px`；回测详情和交易详情图表 viewport / tv `1368x660`，下方双栏为左侧约 `320px`、右侧约 `1056px`，图表在上、详情在下。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：28 个测试文件、140 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过；此前暴露的 `data.test.ts` 文件规模失败已通过拆分 `marketCandle.test.ts` 关闭。
- `docker compose up --build -d api sync backtest trading notify` 后，`curl -fsS http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`，本地 `http://127.0.0.1:8080/research` 可访问。

失败项：

- `node scripts/stage8-visual-smoke.mjs` 本轮曾因运行时间过长被 SIGTERM 中断，未计为通过；已用更小的 CDP 几何复核覆盖研究页、回测详情和交易详情本轮布局风险。
- 首次 `scripts/quality-gate.sh` 暴露 `web/frontend/src/services/api/data.test.ts` 超过 650 行硬上限；已拆出 `web/frontend/src/services/api/marketCandle.test.ts` 并复跑质量门禁通过。

剩余风险：

- 本轮只修复 K 线图表布局、toolbar 密度、右侧坐标区、详情页共享布局，并补全全历史异常 K 线观察入口；未新增交易/回测业务能力。
- 全历史异常扫描只读观察，不自动修复或清洗历史异常行；真实修复仍依赖后续补同步和外部交易所可用性。
- 阶段 1 研究核心仍为 `scaffold`，项目整体仍不能声明 usable 或 production-safe。

### 阶段 1 K 线图表布局精细化复核补充

执行时间：2026-06-30

目标等级：scaffold 增量。

触发问题：

- 用户继续反馈研究页 K 线工具栏粗糙、symbol 输入仍显得过宽、右侧留白明显，并要求交易详情和回测详情不要逐页提醒才修。
- 首次收紧后的 `node scripts/stage8-visual-smoke.mjs` 暴露 812px 窄桌面下 `.research-source-controls` 被媒体查询拉成整行 `758px`，虽然内部控件为固定列宽，但容器视觉上仍不是紧凑 market strip。

修复范围：

- 研究页 symbol 输入进一步收敛为桌面 `112px`、窄桌面 `108px`、移动端 `104px`；内部 instrument refresh 按钮收敛为 `26px`。
- 研究页工具栏去掉窄桌面 / 移动端 `.research-source-controls { width: 100%; }`，控件组按内容宽度呈现，超窄时只在控件组内部滚动。
- 图表固定槽左/右 gutter 收敛为桌面 `16px/6px`、窄桌面 `12px/6px`、移动端 `10px/10px`，降低右侧空白并保留左侧可读边距。
- 交易详情和回测详情下方布局继续沿用上图表、下双栏，摘要列从 `minmax(260px, 320px)` 收窄为 `minmax(240px, 300px)`。
- `TradingViewChart` 初始时间轴逻辑 padding 下调，减少最新 K 线右侧无数据留白。
- `stage8-visual-smoke.mjs` 新增/收紧工具栏约束：桌面 symbol 输入最大 `116px`，桌面/窄桌面 controls 最大 `590px`，详情摘要列期望范围改为 `260px-330px`。

验证：

- `scripts/check-research-chart-layout.sh` 通过。
- `pnpm --dir web/frontend exec vitest run src/pages/ResearchPage.layout.test.ts src/pages/DetailPages.layout.test.ts src/components/chart/TradingViewChart.test.ts` 通过：3 个测试文件、35 条测试。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：28 个测试文件、142 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `docker compose up --build -d api` 后，`curl -fsS http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`。
- `node scripts/stage8-visual-smoke.mjs` 通过：1440 / 812 / 390 视口、浅/深主题、`zh-CN/en-US`，每组 14 页面，最大 document width 分别不超过对应 viewport。
- `node scripts/research-chart-height-smoke.mjs` 通过：`1440x900` 图表 `660px`、`2048x1152` 图表 `760px`、`812x1320` 图表 `680px`、`390x844` 图表 `540px`，污染内部 chart/table/canvas 高度后 document/panel/body/chart/tv 高度均稳定。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

失败项：

- 首次 `node scripts/stage8-visual-smoke.mjs` 因窄桌面 `.research-source-controls` 容器宽度 `758px` 超出新阈值失败；已修复并复跑通过。

剩余风险：

- 本轮只修复 K 线图表布局、工具栏密度、左右 gutter 和详情页双栏比例；未新增指标、绘图、成交联动或策略分析能力。
- 视觉验证仍是几何 smoke，不是像素快照基线；前端基础设施仍保持 `scaffold`。
- 阶段 1 研究核心仍为 `scaffold`，项目整体仍不能声明 usable 或 production-safe。

### 阶段 1 K 线图表布局最终几何约束补充

执行时间：2026-06-30

目标等级：scaffold 增量。

背景：

- 用户继续反馈研究页 K 线工具栏不够精细、symbol 输入仍过宽、右侧价格轴区域仍有明显空白。
- 旧 visual smoke 能通过，但只约束价格轴自身宽度和价格轴贴 chart viewport 右边界，没有直接约束主图 canvas 与价格轴之间是否存在空白。
- 本轮只处理研究页、回测详情和交易详情复用的图表几何约束，不新增指标、绘图或交易分析能力。

Definition of Done：

- 研究页主工具栏按紧凑 market strip 处理，symbol 输入不再携带内置 instrument sync 按钮。
- symbol 输入实际宽度收敛为桌面 `104px`、窄桌面 `102px`、移动端 `100px`，visual smoke 最大宽度阈值收紧到 `108px`。
- 图表槽右侧外 gutter 收敛为桌面 / 窄桌面 `4px`、移动端 `8px`；左侧保留 `16px / 12px / 10px` 读图边距。
- `TradingViewChart` 右侧价格轴 minimumWidth 收敛为 `32/34/36px`，时间轴边缘逻辑 padding 下调，减少最新 K 线右侧空白。
- visual smoke 必须同时检查主图 canvas 右边界贴住右侧价格轴左边界，防止“价格轴不宽但主图与价格轴之间有空白”的回归。
- 回测详情和交易详情继续复用同一 K 线图表槽，保持上图表、下方左窄摘要右宽 tab 列表。

改动范围：

- `web/frontend/src/components/market/MarketSymbolAutoComplete.vue` 增加 `showSyncButton`，默认保留，研究页主工具栏关闭。
- `web/frontend/src/pages/ResearchPage.vue`、`web/frontend/src/pages/ResearchPage.css` 收紧 toolbar、symbol 宽度和图表 gutter。
- `web/frontend/src/pages/klineChartLayout.css`、`web/frontend/src/pages/detailChartLayout.css` 收紧共享图表槽右 gutter。
- `web/frontend/src/components/chart/TradingViewChart.vue` 收紧价格轴和时间轴边缘 padding。
- `scripts/stage8-visual-smoke.mjs` 增加主图 canvas 与右侧价格轴贴合断言，并收紧 toolbar 宽度阈值。
- `scripts/check-research-chart-layout.sh`、相关 layout / chart 单测同步新几何契约。

验证：

- `pnpm --dir web/frontend run test -- src/pages/ResearchPage.layout.test.ts src/pages/DetailPages.layout.test.ts src/components/chart/TradingViewChart.test.ts src/components/market/MarketSymbolAutoComplete.test.ts` 通过：实际执行 28 个测试文件、142 条测试。
- `scripts/check-research-chart-layout.sh` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `docker compose up -d --build api` 通过，`curl -fsS http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`，`http://127.0.0.1:8080/` 已指向新 bundle `index-CebMP6ZI.js`。
- `node scripts/stage8-visual-smoke.mjs` 通过：1440 / 812 / 390 视口、浅/深主题、`zh-CN/en-US`，每组 14 页面，最大 document width 不超过对应 viewport。
- `node scripts/research-chart-height-smoke.mjs` 通过：`1440x900` 图表 `660px`、`2048x1152` 图表 `760px`、`812x1320` 图表 `680px`、`390x844` 图表 `540px`，污染内部 chart/table/canvas 高度后 document/panel/body/chart/tv 高度均稳定。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：28 个测试文件、142 条测试。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

失败项：

- 首次在 `127.0.0.1:8080` 运行 visual smoke 时命中旧 Docker API 容器内置的旧前端资产，symbol 仍为 `112px`，按新阈值失败；已通过 `docker compose up -d --build api` 重建 API 容器并复跑通过。

剩余风险：

- 当前仍是几何 smoke，不是像素快照基线；细粒度视觉回归仍需后续引入截图基线。
- 本轮未新增指标工具、绘图工具、成交点交互或策略分析能力。
- 阶段 1 研究核心仍为 `scaffold`，项目整体仍不能声明 usable 或 production-safe。

### 阶段 1 K 线图表生产高度复核补充

执行时间：2026-06-30

目标等级：scaffold 增量。

背景：

- 用户继续反馈研究页图表可视高度不足、左侧贴边、右侧价格轴区域留白明显，且回测详情和交易详情需要同一布局质量。
- 前一轮已经解决无限拉高，但研究页和详情页的实际 plot 高度仍偏保守，文档中的当前约束也残留旧几何数字。

Definition of Done：

- 研究页、回测详情、交易详情复用同一 K 线固定图表槽；桌面 plot height 收敛为 `clamp(640px, 72vh, 820px)`，窄桌面为 `720px`，移动端为 `600px`，避免首屏被硬撑满但保留可读高度。
- 图表左/右 gutter 保持可读且紧凑：桌面 `18px/4px`、窄桌面 `16px/4px`、移动端 `12px/8px`，上下 padding 为 `0px`。
- 研究页主工具栏从上下两段式改为桌面一行工作台条：左侧 market strip、右侧状态摘要；窄屏再堆叠。symbol 输入固定 `92px`，整组 controls 不超过 `500px`。
- 右侧价格轴减少非绘图区宽度：chart 字体桌面 `9px`、移动 `8px`，right price scale minimumWidth 为 `24/26/28px`，真实右侧价格轴 canvas 不超过 `48px`。
- 回测详情和交易详情继续保持上图表、下方左窄概要右宽 tab 列表，概要列保持 `minmax(240px, 280px)`。
- 本地 `127.0.0.1:8080` 必须运行新 bundle，并通过真实浏览器 smoke。

改动范围：

- `web/frontend/src/pages/klineChartLayout.css` 收敛共享图表槽高度。
- `web/frontend/src/pages/ResearchPage.css` 同步研究页任务列表高度、图表高度、紧凑工具栏布局和控件宽度。
- `web/frontend/src/pages/detailChartLayout.css` 同步回测详情和交易详情图表高度。
- `web/frontend/src/components/chart/TradingViewChart.vue`、`web/frontend/src/theme/tokens.ts` 收紧 chart 字体和右侧价格轴 minimumWidth。
- `web/frontend/src/pages/ResearchPage.layout.test.ts`、`web/frontend/src/pages/DetailPages.layout.test.ts`、`web/frontend/src/components/chart/TradingViewChart.test.ts`、`scripts/research-chart-height-smoke.mjs`、`scripts/stage8-visual-smoke.mjs` 同步新布局契约。

验证：

- `pnpm --dir web/frontend run test -- --run src/pages/ResearchPage.layout.test.ts src/pages/DetailPages.layout.test.ts src/components/chart/TradingViewChart.test.ts` 通过：实际执行 28 个测试文件、142 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `docker compose build api && docker compose up -d api` 通过，`http://127.0.0.1:8080/research` 已运行新 bundle。
- `node scripts/research-chart-height-smoke.mjs` 通过：`1440x900` 图表 `648px`、`2048x1152` 图表 `820px`、`812x1320` 图表 `720px`、`390x844` 图表 `600px`，污染内部 chart/table/canvas 高度后 document/panel/body/chart/tv 高度稳定。
- `node scripts/stage8-visual-smoke.mjs` 通过：1440 / 812 / 390 视口、浅/深主题、`zh-CN/en-US`，每组 14 页面，最大 document width 不超过对应 viewport。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

失败项：

- 首次 `node scripts/research-chart-height-smoke.mjs` 失败：桌面右侧价格轴已从 `52px` 降至 `48px`，但临时 smoke 上限误设为 `46px`；已按真实稳定宽度把上限收敛为 `48px`，并补充移动端 `8px` chart 字体后复跑通过。
- 首次 `node scripts/stage8-visual-smoke.mjs` 失败：详情页仍使用旧桌面最小高度 `700px` 断言；已同步详情页桌面下限为 `620px` 并复跑通过。
- 首次 `scripts/quality-gate.sh` 失败：`scripts/check-research-chart-layout.sh` 仍要求旧任务列表高度和旧工具栏/plot 数字；已同步硬门禁脚本并复跑通过。

剩余风险：

- 本轮仍是几何和浏览器 smoke，不是像素快照基线。
- 未新增绘图工具、指标工具、成交点交互或策略分析能力。
- 研究页和前端基础设施仍为 `scaffold`，项目整体仍不能声明 usable 或 production-safe。

### 阶段 1 K 线图表紧凑轴和大图工作区复核补充

执行时间：2026-06-30

目标等级：scaffold 增量。

背景：

- 用户继续指出研究页工具栏输入过宽、右侧价格轴区域留白明显，并要求回测详情、交易详情沿用同一质量标准。
- 上一轮图表高度和 gutter 数字仍偏保守，且右侧 price scale minimumWidth 会给非绘图区增加额外宽度。

Definition of Done：

- 研究页、回测详情、交易详情继续复用共享 K 线固定图表槽；桌面 plot height 为 `clamp(680px, 72dvh, 820px)`，窄桌面为 `700px`，移动端为 `580px`。
- 研究页主工具栏 symbol 输入收敛为桌面 `112px`、窄桌面 `108px`、移动端 `104px`，market strip 按内容宽布局，状态摘要在剩余空间内横向滚动。
- 图表 gutter 收敛为桌面 `14px/0px`、窄桌面 `12px/0px`、移动端 `10px/0px`，右侧不再人为留 padding。
- `TradingViewChart` 价格轴 `minimumWidth` 收敛为 `0`，chart 字体为桌面 `8px`、移动 `7px`，latest window 默认可见 K 线数减少到更可读的密度。
- 不覆盖 lightweight-charts 内部 table / tbody / tr / td；只约束 lightweight root 不超过外层固定 viewport。

改动范围：

- `web/frontend/src/pages/klineChartLayout.css`、`web/frontend/src/pages/ResearchPage.css`、`web/frontend/src/pages/detailChartLayout.css` 更新共享图表槽高度、gutter 和工具栏几何。
- `web/frontend/src/components/chart/TradingViewChart.vue`、`web/frontend/src/components/chart/TradingViewChart.css` 更新价格轴、字体、首屏可见密度和 root overflow 约束。
- `web/frontend/src/components/chart/TradingViewChart.test.ts`、`web/frontend/src/pages/ResearchPage.layout.test.ts`、`web/frontend/src/pages/DetailPages.layout.test.ts`、`scripts/check-research-chart-layout.sh`、`scripts/research-chart-height-smoke.mjs` 同步布局契约。

验证：

- `scripts/check-research-chart-layout.sh` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：28 个测试文件、142 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `BASE_URL=http://127.0.0.1:8081 SMOKE_SAMPLES=8 SMOKE_SETTLE_MS=1000 scripts/research-chart-height-smoke.mjs` 通过：`1440x900` 图表 `680px`、`2048x1152` 图表 `820px`、`812x1320` 图表 `700px`、`390x844` 图表 `580px`；反复污染内部 chart/table/canvas 高度后 document/panel/body/chart/tv 高度稳定。

未完成 / 风险：

- 该切片只收敛 K 线工作区几何和布局回归检查，不代表研究页、回测或交易业务流程达到 production-safe。
- 仍缺像素快照基线、多浏览器视觉回归和动态详情全数据状态覆盖。

### 阶段 1 全历史 invalid 补同步入口补充

执行时间：2026-06-30

目标等级：scaffold 增量。

背景：

- 研究页已能扫描并展示当前数据源全历史 invalid K 线，但详情弹窗仍只能观察。
- 阶段 1 的研究核心需要从“发现数据健康问题”推进到“可在同一研究工作流排补同步任务”，但仍不能由前端直接清洗或改写 `market_candles`。

Definition of Done：

- 新增 `POST /api/market/candle-invalid-issues/repair`，请求包含 exchange / symbol / interval / openTimes。
- API 写请求继续要求登录、CSRF 和 active market instrument。
- PostgreSQL store 必须逐个 `open_time` 重查 persisted `market_candles` 的真实 OHLCV invalid 条件；不是当前真实 invalid 的 openTime 返回 `ErrNotFound`，不创建任务。
- 补同步任务窗口为 `[open_time, open_time + interval]`，`sync_enabled=true`，`realtime_enabled=false`，`status=pending`，不写 `repair_source_task_id`，因为来源是全历史市场数据而非某个 data sync task。
- 已存在同 exchange / symbol / interval / start_time / end_time 的 active 补同步任务时返回 `skippedExisting`，不重复创建。
- 研究页全历史 invalid 详情弹窗可排队补同步当前返回的异常 openTime，并刷新任务列表。
- 不自动清洗历史行，不自动后台修复，不升级阶段 1 或项目整体等级。

改动范围：

- `internal/data/market_candle_gap.go` 和 repository interface 增加 `RepairMarketCandleInvalidIssuesRequest` / 方法。
- `internal/store/postgres/market_candle_invalid_issue_store.go` 增加全历史 invalid repair 事务逻辑。
- `internal/web/api/market_handlers.go`、API contract、schema drift 和 method contract 增加新路由。
- `web/frontend/src/components/research/MarketCandleInvalidIssueTag.vue` 增加“排队补同步当前异常”入口和结果反馈。
- `web/frontend/src/services/api/data.ts`、generated API types、app types 和 i18n 同步新请求类型和文案。

当前验证：

- `scripts/generate-api-types.sh` 通过。
- `go test ./internal/web/api -run 'TestMarketCandleInvalidIssueRepair|TestAPIContract|TestAPIMethodNotAllowedContracts|TestFrontendAPI' -count=1` 通过。
- `go test ./internal/store/postgres -run 'TestIntegrationRepairMarketCandleInvalidIssues|TestIntegrationScanMarketCandleInvalidIssues' -count=1` 通过。
- `pnpm --dir web/frontend exec vitest run src/services/api/marketCandle.test.ts src/components/research/MarketCandleInvalidIssueTag.test.ts` 通过：2 个测试文件、10 条测试。
- `pnpm --dir web/frontend run typecheck` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过：28 个测试文件、142 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `docker compose up --build -d api sync backtest trading notify` 通过，`curl -fsS http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`。
- `node scripts/research-chart-height-smoke.mjs` 通过，当前 8080 在 1440 / 2048 / 812 / 390 视口下图表高度稳定。

失败项：

- 首次前端组件测试没有点到 Naive UI modal teleport 中的按钮；已改为从真实 `document` 查询按钮并重跑通过。
- 首次 typecheck 暴露 `openTime` 可选字段未被 `filter(Boolean)` 收窄；已加显式 type guard 并重跑通过。

剩余风险：

- 本轮只创建补同步任务，不保证交易所一定能返回替换历史异常行；结果仍依赖 sync worker、adapter 和外部网络。
- 全历史 invalid repair 按当前返回的 openTime 列表执行，不做自动全量修复。
- 阶段 1 研究核心仍不能标记 usable；项目整体仍为 `scaffold`。

### 阶段 1 全历史 invalid 补同步执行收敛证据补充

执行时间：2026-06-30

目标等级：scaffold 增量。

背景：

- 上一轮已经提供全历史 invalid K 线扫描和补同步任务排队入口，但验收重点停留在“能创建 repair task”。
- 阶段 1 研究核心要继续接近真实可用，必须证明当补同步 worker 成功拉回健康 K 线后，历史异常行会通过正常写回路径被替换，并且研究页依赖的全历史 invalid scan 能观察到健康回流。

Definition of Done：

- 使用真实 PostgreSQL 集成测试制造 legacy invalid `market_candles` 行。
- 通过 `RepairMarketCandleInvalidIssues` 创建全历史 invalid repair task，不直接改写市场数据。
- 通过 `SaveDataSyncResult` 写入同一 open_time 的健康 K 线，覆盖历史 invalid 行。
- 验证 repair task 收敛到 `succeeded`，`sync_enabled=false`，`latest_synced_open_time` 推进到修复 open_time。
- 验证 `ScanMarketCandleInvalidIssues` 在相同 exchange / symbol / interval 下回到 `TotalCount=0`、`Issues=[]`，且窗口 K 线数量保持完整。
- 不新增自动清洗、不绕过 positive price constraint、不把阶段 1 升级为 usable。

改动范围：

- `internal/store/postgres/market_candle_gap_store_integration_test.go` 新增 `TestIntegrationRepairMarketCandleInvalidIssueConvergesFullHistoryScan`。

当前验证：

- `go test ./internal/store/postgres -run 'TestIntegrationRepairMarketCandleInvalidIssueConvergesFullHistoryScan' -count=1` 通过。
- `go test ./internal/store/postgres -run 'TestIntegration(RepairMarketCandleInvalidIssue|RepairMarketCandleInvalidIssues|ScanMarketCandleInvalidIssues|RepairDataSyncTaskInvalidIssuesConvergesSourceHealth|RepairTaskExecutionConvergesSourceDataHealth)' -count=1` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：28 个测试文件、142 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

失败项：

- 无。

剩余风险：

- 该测试证明的是“worker 成功保存健康 K 线时会收敛”，不保证交易所一定能返回对应历史数据。
- 当前仍没有自动全量修复或自动清洗历史 invalid 行；用户仍需要从研究页排队补同步并观察任务结果。
- 阶段 1 研究核心仍为 `scaffold`，项目整体仍不能声明 usable 或 production-safe。

### 阶段 1 全历史 invalid HTTP API 收敛证据补充

执行时间：2026-06-30

目标等级：scaffold 增量。

背景：

- store 层已有全历史 invalid repair 收敛证据，但 API 路由层仍主要依赖 fake repository 测试。
- 阶段 1 研究核心需要证明用户可见工作流对应的 HTTP API 能串起真实 PostgreSQL store、登录会话、CSRF、active market instrument 校验、repair task 创建和 worker 写回后的 scan 归零。

Definition of Done：

- 在 `internal/web/api` 包新增 PostgreSQL 集成测试，使用真实 `postgres.Store` 启动 `NewServer`。
- 测试创建唯一 operator，通过 `/api/auth/login` 取得 session + CSRF。
- 测试创建唯一 active `market_instruments`，并种入 3 根 `market_candles`，其中 1 根为 legacy invalid OHLCV。
- 通过 `GET /api/market/candle-invalid-issues` 证明 HTTP scan 能看到 1 条 persisted invalid issue。
- 通过 `POST /api/market/candle-invalid-issues/repair` 证明 CSRF 写请求能创建 1 个 full-history invalid repair task。
- 通过 `SaveDataSyncResult` 写回同 open_time 健康 K 线，再通过 HTTP scan 证明 `TotalCount=0`、`Issues=[]`。
- 测试 cleanup 必须删除测试 operator/session/task/candle/instrument，并恢复 `market_candles_positive_price_values_check`。
- 没有 `TICTICK_TEST_DATABASE_URL` 时允许 skip；必须额外在 Docker Compose PostgreSQL 上运行一次真实集成测试。

改动范围：

- `internal/web/api/market_invalid_repair_integration_test.go` 新增 API + PostgreSQL 集成测试和测试辅助函数。

当前验证：

- `go test ./internal/web/api -run 'TestIntegrationMarketCandleInvalidIssueRepairRouteConvergesPostgresScan|TestMarketCandleInvalidIssueRepairRoute' -count=1` 通过；本机无 `TICTICK_TEST_DATABASE_URL` 时集成测试 skip，fake repository 路由测试执行。
- Docker Compose PostgreSQL 真实执行通过：
  `docker run --rm --network tictick-hi_default -v "$PWD":/src -w /src -e TICTICK_TEST_DATABASE_URL=postgresql://...@postgres:5432/... golang:1.26-bookworm go test ./internal/web/api -run TestIntegrationMarketCandleInvalidIssueRepairRouteConvergesPostgresScan -count=1 -v`
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：28 个测试文件、142 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

失败项：

- 首次 Docker PostgreSQL 集成测试失败：seed 数据 `open=0, low=99` 违反 `market_candles_ohlc_bounds_check`；已改为 `open=0, low=0, close=100, high=101`，只违反 positive price invalid 条件并复跑通过。

剩余风险：

- 该测试证明 API 到 PostgreSQL 的单条 invalid repair 收敛路径，不代表自动全量清洗历史 invalid 行。
- 仍不保证交易所一定返回可覆盖异常 open_time 的健康历史数据。
- 阶段 1 研究核心仍为 `scaffold`，项目整体仍不能声明 usable 或 production-safe。

### 阶段 1 K 线图表右侧 gutter 合同统一补充

执行时间：2026-06-30

目标等级：scaffold 增量。

背景：

- 用户继续指出研究页、回测详情、交易详情的 K 线图表右侧视觉留白和工具栏密度仍不稳定，不能只修单页。
- 本地 `stage8-visual-smoke.mjs` 暴露研究页右侧图表 gutter 合同冲突：`research-chart-height-smoke` 允许 `0px`，但 Stage 8 visual 要求生产级右侧呼吸空间。
- 窄桌面 `812px` 视口下研究页 toolbar 实测高度 `77px`，超过 `76px` 合同，说明工具栏仍有 1px 级别的密度回归。

Definition of Done：

- 研究页、回测详情、交易详情继续复用同一固定 K 线图表槽。
- 图表右侧外层 gutter 统一为 `4px`，左侧保持桌面 `14px`、窄桌面 `12px`、移动端 `10px`。
- 右侧价格轴 canvas 默认上限从 `48px` 收紧到 `44px`，主图 canvas 必须贴住右侧价格轴左边界。
- 研究页窄桌面 toolbar 压缩到 Stage 8 visual 合同内，不通过放宽阈值解决。
- 不覆盖 lightweight-charts 内部 table / tbody / tr / td；不新增指标、绘图工具或交易分析功能。

改动范围：

- `web/frontend/src/pages/klineChartLayout.css` 统一共享图表槽右侧 gutter。
- `web/frontend/src/pages/ResearchPage.css` 同步研究页 gutter，并收紧窄桌面 toolbar `gap` / `padding`。
- `web/frontend/src/pages/detailChartLayout.css` 同步回测详情和交易详情右侧 gutter。
- `web/frontend/src/pages/ResearchPage.layout.test.ts`、`web/frontend/src/pages/DetailPages.layout.test.ts`、`scripts/check-research-chart-layout.sh` 同步静态布局合同。
- `scripts/research-chart-height-smoke.mjs`、`scripts/stage8-visual-smoke.mjs` 统一右侧 gutter 范围并把价格轴宽度上限收紧到 `44px`。

当前验证：

- `scripts/check-research-chart-layout.sh` 通过。
- `pnpm --dir web/frontend exec vitest run src/pages/ResearchPage.layout.test.ts src/pages/DetailPages.layout.test.ts src/components/chart/TradingViewChart.test.ts` 通过：3 个测试文件、35 条测试。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：28 个测试文件、142 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `docker compose build api && docker compose up -d api` 通过，`curl -fsS http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SAMPLES=8 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：`1440x900` 图表 `680px`、`2048x1152` 图表 `820px`、`812x1320` 图表 `700px`、`390x844` 图表 `580px`，连续采样稳定。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=800 node scripts/stage8-visual-smoke.mjs` 通过：1440 / 812 / 390 视口、浅 / 深主题、`zh-CN/en-US`，每组 14 页，最大 document width 不超过对应 viewport。

失败项：

- 首次 `stage8-visual-smoke.mjs` 失败：研究页右侧 gutter 为 `0px`，不满足 Stage 8 visual 的 `2-8px` 合同；已统一为 `4px` 并复跑通过。
- 第二次 `stage8-visual-smoke.mjs` 失败：`812px` 窄桌面研究页 toolbar 实测 `77px`，超过 `76px`；已收紧窄桌面 toolbar gap / padding 并复跑通过。

剩余风险：

- 该切片只修复 K 线布局几何合同，不代表研究页、回测或交易业务流程达到 production-safe。
- 仍缺多浏览器视觉回归、图表指标/绘图工具、成交点交互和完整交易分析能力。

### 阶段 1 任务窗口异常修复结果可观察性补充

执行时间：2026-06-30

目标等级：scaffold 增量。

背景：

- 任务窗口 invalid repair 已有 HTTP API + PostgreSQL 收敛证据，但研究页弹窗只显示“已排队 / 已存在 / 无可修复”，没有把后端返回的 total / limit / limited / created / skipped 信息回显给用户。
- 用户需要能判断一次修复请求到底匹配了多少异常、是否被单次上限截断、创建了哪些补同步窗口，而不是只能等待任务列表刷新后猜测。

Definition of Done：

- `ResearchTaskInvalidIssueModal` 在 `POST /api/data/tasks/{id}/repair-invalid-issues` 成功后显示本次匹配总数、创建数量、跳过已存在数量、单次修复上限和受限标记。
- 弹窗列出最多 3 个新创建补同步任务的 ID 和 start/end 窗口，超出时显示剩余数量。
- 切换筛选、重置筛选或重新打开任务时清空旧 repair result，避免 stale 结果误导。
- repair 失败只显示泛化失败文案，不展示底层交易所 URL 或原始错误。
- 不改变后端 repair 语义，不新增 API，不自动清洗历史 K 线。

改动范围：

- `web/frontend/src/components/research/ResearchTaskInvalidIssueModal.vue` 增加 repair result 状态、summary tags 和任务窗口 tags。
- `web/frontend/src/components/research/ResearchTaskInvalidIssueModal.test.ts` 覆盖 created、skipped、limited、no repair、筛选清空 stale result 和失败不泄漏 URL。
- `web/frontend/src/i18n/messages.research.zh.ts` / `messages.research.en.ts` 增加中英文 repair result 文案。
- `docs/quality-audit.md` 同步本轮 DoD、验证和剩余风险，并修正旧 K 线 gutter / price axis 合同残留。

当前验证：

- `pnpm --dir web/frontend exec vitest run src/components/research/ResearchTaskInvalidIssueModal.test.ts src/services/api/data.invalid.test.ts` 通过：2 个测试文件、10 条测试。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：28 个测试文件、146 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

失败项：

- 首次补丁因尾部上下文不匹配未写入文件；已拆成小块补丁重新应用。
- 测试最初不能假设 `toLocaleString()` 精确日期格式；已将断言收敛到补同步任务 ID 和 summary。

剩余风险：

- 该切片只增强研究页修复结果可观察性，不证明交易所一定能返回健康历史数据。
- 自动批量修复、历史异常自动清洗、多任务修复调度和生产级数据修复策略仍未关闭。

### 阶段 1 K 线图表生产级布局返工补充

执行时间：2026-06-30

目标等级：scaffold 增量。

背景：

- 研究页、回测详情和交易详情虽然已接入共享 K 线固定图表槽，但仍需要用真实运行态几何验证工具栏密度、图表高度、左右边距、右侧价格轴贴合和内部 canvas 裁切。
- 本轮不能只改静态 CSS contract；必须用本地浏览器检查 `lightweight-charts` 实际生成的主图 canvas、右侧 price scale 和时间轴位置。

Definition of Done：

- 研究页工具栏保持一行紧凑 market strip；exchange / symbol / refresh / interval / window controls 不出现超长输入框，状态标签单独滚动且不挤压图表。
- 研究页、回测详情和交易详情复用同一固定图表槽：图表在上，下方为概要窄列 + tab 宽列；移动和窄桌面退化为单列。
- K 线图表在 1440、812、390 视口下有可读高度，不按首屏剩余空间机械平分；内部 canvas、右侧价格轴、时间轴必须都落在固定 viewport 内。
- 左侧保留少量读图 gutter，右侧只保留必要 breathing room；不得出现价格轴右侧大块空白，主图 canvas 必须贴近 price scale。
- `TradingViewChart` 不再因内部 table/canvas resize feedback 造成图表无限拉高。
- 更新静态 layout test、浏览器几何 smoke 和质量审计；不新增业务功能，不改变后端 API。

改动范围：

- `web/frontend/src/pages/ResearchPage.css`、`detailChartLayout.css`、`klineChartLayout.css`、`TradingViewChart.vue/css` 按真实几何回收布局。
- `web/frontend/src/pages/ResearchPage.layout.test.ts`、`DetailPages.layout.test.ts`、`TradingViewChart.test.ts` 和 `scripts/*chart*smoke*` 同步生产级布局合同。
- `docs/quality-audit.md` 记录本轮验证、失败项和剩余风险。

当前验证：

- `scripts/check-research-chart-layout.sh` 通过。
- `pnpm --dir web/frontend exec vitest run src/pages/ResearchPage.layout.test.ts src/pages/DetailPages.layout.test.ts src/components/chart/TradingViewChart.test.ts` 通过：3 个测试文件、35 条测试。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：28 个测试文件、146 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- `docker compose build api && docker compose up -d api` 通过，`curl -fsS http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SAMPLES=4 SMOKE_INTERVAL_MS=120 SMOKE_SETTLE_MS=900 node scripts/research-chart-height-smoke.mjs` 通过：1440 / 2048 / 812 / 390 视口高度稳定，图表 root / TV canvas 没有增长。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=800 node scripts/stage8-visual-smoke.mjs` 通过：1440 / 812 / 390 视口、浅 / 深主题、`zh-CN/en-US`，每组 14 页，最大 document width 不超过对应 viewport。
- 真实 8080 Headless Chrome 几何采样：1440 下 toolbar `47px`、controls `528px`、symbol `112px`、body `680px`、main `1332px`、price axis `42px`、right blank `2px`；812 下 toolbar `74px`、symbol `108px`、body `700px`、main `722px`、price axis `42px`、right blank `2px`；390 下 toolbar `80px`、symbol `104px`、body `580px`、main `302px`、price axis `42px`、right blank `2px`。

失败项：

- 未出现验证失败；本轮先用运行态几何采样确认旧问题后再收紧合同。

剩余风险：

- 本轮只处理图表与详情页布局，不代表研究、回测、交易业务流程达到 production-safe。
- 仍缺像素快照基线和真实用户浏览器矩阵，当前浏览器证据来自 Headless Chrome。

### 阶段 1 任务窗口缺口修复结果可观察性补充

执行时间：2026-06-30

目标等级：scaffold 增量。

背景：

- 任务窗口 gap repair 已有 HTTP API + PostgreSQL 收敛证据，但研究页行操作只弹出“已排队/已存在/无可修复”的短提示。
- 后端已经返回 `totalCount`、`repairLimit`、`limited`、`createdTasks` 和 `skippedExisting`，前端必须把这些结果留在当前工作流中，用户才能判断本次修复到底处理了多少缺口、是否被上限截断、排了哪些补同步窗口。

Definition of Done：

- 研究页缺口详情弹窗提供“修复缺口”入口，复用 `POST /api/data/tasks/{id}/repair-gaps`，不让用户只能回到表格行操作。
- repair 成功后在缺口详情弹窗展示本次匹配总数、创建数量、跳过已存在数量、单次修复上限和受限标记。
- 弹窗列出最多 3 个新创建补同步任务的 ID 与 start/end 窗口，超出时显示剩余数量。
- 从任务行直接点“修复缺口”也会打开/刷新缺口详情弹窗并展示 repair result，避免只有 toast 的短暂反馈。
- 重新打开其他任务缺口详情时清空旧 repair result，避免 stale 结果误导。
- repair 失败只显示泛化失败文案，不展示底层交易所 URL 或原始错误。
- 不改变后端 repair 语义，不新增 API，不直接写入或清洗 `market_candles`。

改动范围：

- `web/frontend/src/components/research/ResearchTaskGapDetailsModal.vue`：把任务缺口详情弹窗拆成独立组件，弹窗内展示 repair 入口、结果摘要、受限标记和最多 3 个补同步任务窗口。
- `web/frontend/src/composables/useResearchWorkspace.ts`、`researchWorkspaceHelpers.ts`：保留行操作 repair 入口，并把 repair 结果写入弹窗状态；切换任务时清空旧结果；失败时只暴露泛化失败文案。
- `web/frontend/src/i18n/messages.research.zh.ts`、`messages.research.en.ts`：补齐 repair 结果摘要、受限和补同步任务窗口文案。
- `web/frontend/src/composables/useResearchWorkspace.taskGapRepair.test.ts`、`useResearchWorkspace.test.ts`、`ResearchPage.layout.test.ts`：拆分任务缺口 repair 用例，并更新页面到弹窗组件的静态布局合同。
- 不改后端 API、repair 语义、数据库 schema 或 K 线数据写入规则。

当前验证：

- `pnpm --dir web/frontend exec vitest run src/composables/useResearchWorkspace.test.ts src/composables/useResearchWorkspace.taskGapRepair.test.ts src/pages/ResearchPage.layout.test.ts` 通过：3 个测试文件、34 条测试。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：29 个测试文件、150 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- `curl -fsS http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SAMPLES=4 SMOKE_INTERVAL_MS=120 SMOKE_SETTLE_MS=900 node scripts/research-chart-height-smoke.mjs` 通过：1440 / 2048 / 812 / 390 视口图表高度稳定，`body/chart/tv` 高度分别为 680 / 820 / 700 / 580。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=800 node scripts/stage8-visual-smoke.mjs` 通过：1440 / 812 / 390 视口、浅 / 深主题、`zh-CN/en-US`，每组 14 页，最大 document width 不超过对应 viewport。

失败项：

- 首次完整质量门禁失败：`useResearchWorkspace.ts` 415 行超过 400、`useResearchWorkspace.test.ts` 701 行超过 650、`ResearchPage.vue` 493 行超过 450；已通过抽出 `ResearchTaskGapDetailsModal.vue`、拆分 `useResearchWorkspace.taskGapRepair.test.ts`、抽出 `taskGapRepairFeedback` 修复，复跑 `scripts/quality-gate.sh` 通过。
- 重构后 `pnpm --dir web/frontend run typecheck` 曾因 `t(feedback.messageKey, feedback.values)` 的 `undefined` 参数类型失败；已改成 `feedback.values ?? {}`，复跑 typecheck 通过。

剩余风险：

- 本轮只增强任务窗口缺口修复结果可观察性，不保证交易所一定返回缺失历史数据。
- 弹窗可以观察补同步任务创建结果，但本轮不证明这些补同步任务后续一定收敛为健康 K 线。
- 当前浏览器 UI 证据来自 Headless Chrome smoke，不等同于完整真实浏览器矩阵或像素快照基线。

### 阶段 1 K 线图表 OHLCV 可读性补充

执行时间：2026-06-30

目标等级：scaffold 增量。

背景：

- 研究页、回测详情和交易详情已经共用固定 K 线图表槽，但图表只呈现 K 线形状、成交量和 marker。
- 研究工作台需要让用户直接读到当前 K 线的 open/high/low/close/volume、涨跌和时间；否则用户只能靠视觉估算，图表研究能力仍薄。

Definition of Done：

- `TradingViewChart` 在有数据时显示紧凑 OHLCV readout，默认展示最新一根 K 线。
- 鼠标十字光标移动到某根 K 线时，readout 切换为该 K 线；鼠标离开或无命中时恢复最新一根。
- readout 展示 UTC 时间、O/H/L/C/V、涨跌绝对值和涨跌百分比，并按涨跌方向使用成功/危险/中性色，不遮挡图表交互。
- 研究页、回测详情、交易详情通过共享 `TradingViewChart` 自动获得该能力，不新增页面级重复实现。
- 不新增指标系统、不改后端 API、不改 CandleProvider 语义、不改变 K 线渲染数据。

改动范围：

- `web/frontend/src/components/chart/TradingViewChart.vue`：新增共享 OHLCV readout，默认显示最新 K 线，十字光标命中时切换到对应 K 线，离开或未命中时恢复最新 K 线。
- `web/frontend/src/components/chart/chartReadout.ts`：封装 UTC 时间、价格、成交量、涨跌和涨跌百分比格式化，避免继续膨胀图表组件。
- `web/frontend/src/components/chart/TradingViewChart.css`：新增轻量 readout overlay，`pointer-events: none`，不拦截图表交互。
- `web/frontend/src/components/chart/TradingViewChart.readout.test.ts`、`TradingViewChart.test.ts`：覆盖最新 K 线读数、十字光标切换、卸载退订和既有 chart mock API。
- `scripts/stage8-visual-smoke.mjs`：有数据图表必须出现 OHLCV readout，且 readout 必须在固定图表 viewport 内、保持紧凑高度；空图表允许 empty state。
- 不改后端 API、CandleProvider、数据库 schema、K 线数据语义或页面级重复实现。

当前验证：

- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts src/components/chart/TradingViewChart.readout.test.ts` 通过：2 个测试文件、23 条测试。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：30 个测试文件、152 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `node --check scripts/stage8-visual-smoke.mjs` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- `docker compose build api && docker compose up -d api && curl -fsS http://127.0.0.1:8080/readyz` 通过，readyz 返回 `{"status":"ok"}`。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SAMPLES=4 SMOKE_INTERVAL_MS=120 SMOKE_SETTLE_MS=900 node scripts/research-chart-height-smoke.mjs` 通过：1440 / 2048 / 812 / 390 视口图表高度稳定，`body/chart/tv` 高度分别为 680 / 820 / 700 / 580。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=800 node scripts/stage8-visual-smoke.mjs` 通过：1440 / 812 / 390 视口、浅 / 深主题、`zh-CN/en-US`，每组 14 页，最大 document width 不超过对应 viewport。

失败项：

- 首次 `TradingViewChart` 目标测试失败：既有测试 mock 未提供 `subscribeCrosshairMove` / `unsubscribeCrosshairMove`，新增 readout 测试也未等待 Vue DOM 刷新；已补 mock API 和 `nextTick`，复跑目标测试通过。

剩余风险：

- 本轮只增强图表读数，不代表策略指标、画线工具、成交点联动或完整交易分析能力已具备。
- readout 只展示当前/悬停 K 线的基础 OHLCV，不包含 MA/EMA、指标模板、区间统计或订单成交联动。

### 阶段 1 K 线图表工作台布局合同复核

执行时间：2026-06-30

目标等级：scaffold 增量。

背景：

- 用户指出研究页、回测详情和交易详情的 K 线图表曾出现工具栏粗糙、symbol 输入过宽、图表高度过窄、左侧贴边、右侧大空白和内容裁切等问题。
- 这些问题必须固化为共享布局合同和运行态 smoke，而不是只靠单次截图人工纠错。

Definition of Done：

- 研究页图表工具栏保持工作台密度：exchange / symbol / refresh / interval / window controls 在桌面为单行紧凑 market strip，symbol 输入不超过生产阈值，状态摘要独立滚动且不挤压主图。
- 研究页、回测详情和交易详情共用同一个固定 K 线图表槽合同：图表是上方主体，下方详情页摘要左窄、列表右宽；窄屏可堆叠但不得水平溢出。
- 图表 plot 高度在桌面、窄桌面和移动视口都有明确下限，不能为了平分首屏导致图表不可读，也不能因 ResizeObserver 或图表内部 DOM 反馈无限增高。
- 图表左侧 gutter 只保留视觉呼吸空间，右侧 gutter 只允许价格轴贴边所需空间；右侧价格轴必须贴近 viewport 右边界，不允许出现肉眼可见的大空白。
- visual smoke 必须同时验证工具栏宽高、symbol 控件宽度、左右 gutter、右侧价格轴宽度、主图 canvas 和价格轴/viewport 的贴合关系、详情页上下两栏和下方左右列比例。
- 不改 CandleProvider、后端 API、数据库 schema、交易/回测业务语义或新增空壳功能。

改动范围：

- `web/frontend/src/pages/klineChartLayout.css`：把 K 线图表 plot 高度、左右 gutter 和响应式断点集中到共享图表槽，作为研究页、回测详情和交易详情的单一尺寸来源。
- `web/frontend/src/pages/ResearchPage.css`：删除研究页重复声明的图表高度和 gutter 变量，只保留研究页工具栏、任务列表和 frame flex 承载职责。
- `web/frontend/src/pages/detailChartLayout.css`：删除回测详情 / 交易详情重复声明的图表高度和 gutter 变量，只保留详情页 viewport isolation。
- `web/frontend/src/pages/ResearchPage.layout.test.ts`、`DetailPages.layout.test.ts`：静态合同改为断言共享 `klineChartLayout.css` 承担尺寸来源，并禁止研究页 / 详情页重新私有声明图表尺寸。
- `scripts/check-research-chart-layout.sh`：轻量门禁同步单一尺寸来源合同，并检查详情页 CSS 不再重复声明图表尺寸变量。
- `scripts/stage8-visual-smoke.mjs`：新增 `SMOKE_MAX_CHART_EDGE_GAP`，把右侧 canvas / 价格轴贴边容差收紧为默认 `3px`，避免右侧大空白回归。
- 不改后端 API、CandleProvider、数据库 schema、交易 / 回测业务语义或 K 线数据语义。

当前验证：

- `scripts/check-research-chart-layout.sh` 通过。
- `node --check scripts/stage8-visual-smoke.mjs` 通过。
- `pnpm --dir web/frontend exec vitest run src/pages/ResearchPage.layout.test.ts src/pages/DetailPages.layout.test.ts` 通过：2 个测试文件、15 条测试。
- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts src/components/chart/TradingViewChart.readout.test.ts src/pages/ResearchPage.layout.test.ts src/pages/DetailPages.layout.test.ts` 通过：4 个测试文件、38 条测试。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：30 个测试文件、152 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- `docker compose build api && docker compose up -d api && curl -fsS http://127.0.0.1:8080/readyz` 通过，readyz 返回 `{"status":"ok"}`。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SAMPLES=4 SMOKE_INTERVAL_MS=120 SMOKE_SETTLE_MS=900 node scripts/research-chart-height-smoke.mjs` 通过：1440 / 2048 / 812 / 390 视口图表高度稳定，`body/chart/tv` 高度分别为 680 / 820 / 700 / 580。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=800 node scripts/stage8-visual-smoke.mjs` 通过：1440 / 812 / 390 视口、浅 / 深主题、`zh-CN/en-US`，每组 14 页，最大 document width 不超过对应 viewport，右侧图表边缘合同使用默认 `3px` 上限。

失败项：

- 无。

剩余风险：

- 本轮是布局合同复核和维护性收敛，不代表研究、回测、交易流程达到 `usable` 或 `production-safe`。
- 当前浏览器证据仍来自 Headless Chrome smoke，缺真实浏览器矩阵和像素快照基线。
- 图表工具栏、OHLCV readout 和详情页双栏布局仍是基础工作台能力，策略指标、画线工具、成交点联动和完整交易分析能力未关闭。

### 阶段 1 全历史缺口和异常修复结果可观察性补充

执行时间：2026-06-30

目标等级：scaffold 增量。

背景：

- 任务窗口 gap / invalid repair 已经能在弹窗里显示 total / limit / created / skipped 和补同步任务窗口。
- 全历史 gap / invalid repair 仍主要依赖 toast 或简单数量标签，用户不能在当前全历史详情弹窗里持续看到本次修复创建了哪些补同步任务、是否跳过已有任务、是否受单次上限截断。

Definition of Done：

- `MarketCandleGapTag` 在修复首个缺口或当前返回缺口后，在全历史缺口详情弹窗内展示本次匹配总数、创建数量、跳过已存在数量、单次修复上限和受限标记。
- `MarketCandleInvalidIssueTag` 在修复当前返回异常后，在全历史异常详情弹窗内展示同样的修复结果摘要。
- 两个全历史详情弹窗都列出最多 3 个新创建补同步任务的 ID 与 start/end 窗口，超出时显示剩余数量。
- 切换 exchange / symbol / interval 或重新扫描时清空旧 repair result，避免 stale 结果误导当前数据源。
- repair 失败只显示泛化失败文案，不展示底层交易所 URL 或原始错误。
- 不新增 API、不改变后端 repair 语义、不直接修改 `market_candles`、不做自动批量清洗。

改动范围：

- `web/frontend/src/components/research/MarketRepairResultTags.vue`：新增全历史修复结果展示组件，统一 summary、limited 标记、最多 3 个补同步任务窗口和剩余数量。
- `web/frontend/src/components/research/MarketCandleGapTag.vue`：修复首个缺口或当前返回缺口后，在全历史缺口详情弹窗中保留本次 repair result；修复后刷新扫描但不清空本次结果；repair 失败只显示泛化失败文案。
- `web/frontend/src/components/research/MarketCandleInvalidIssueTag.vue`：全历史异常修复后保留完整 repair result；repair 失败不再把原始 provider URL 写入 `title`。
- `web/frontend/src/components/research/MarketRepairResultTags.test.ts`、`MarketCandleGapTag.test.ts`、`MarketCandleInvalidIssueTag.test.ts`：覆盖 summary、limited、任务窗口、隐藏数量和失败不泄漏 URL。
- `web/frontend/src/i18n/messages.research.zh.ts`、`messages.research.en.ts`：补全全历史 repair result 中英文文案。
- 不改后端 API、数据库 schema、repair 创建语义或 `market_candles` 写入规则。

当前验证：

- `pnpm --dir web/frontend exec vitest run src/components/research/MarketRepairResultTags.test.ts src/components/research/MarketCandleGapTag.test.ts src/components/research/MarketCandleInvalidIssueTag.test.ts` 通过：3 个测试文件、12 条测试。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：31 个测试文件、155 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- `docker compose build api && docker compose up -d api && curl -fsS http://127.0.0.1:8080/readyz` 通过，readyz 返回 `{"status":"ok"}`。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=800 node scripts/stage8-visual-smoke.mjs` 通过：1440 / 812 / 390 视口、浅 / 深主题、`zh-CN/en-US`，每组 14 页，最大 document width 不超过对应 viewport。

失败项：

- 首次目标组件测试失败：测试写死 UTC 字符串，但新共享组件复用 `formatCompactDateTime` 的本地紧凑时间格式；已改为测试侧复用同一格式化函数，复跑通过。

剩余风险：

- 该切片只增强全历史 gap / invalid repair 的前端可观察性，不证明交易所一定返回缺失或健康 K 线。
- 全历史修复仍是用户触发的有限批次排队，不是自动全量修复、历史清洗或生产级数据修复调度。
- 当前浏览器回归来自 Headless Chrome smoke，弹窗细节主要由组件测试覆盖，仍缺像素快照基线和真实浏览器矩阵。

### 阶段 1 K 线图表布局生产化补充

目标等级：scaffold。

范围内：

- 研究页、回测详情、交易详情继续复用同一套 K 线图表固定 viewport 契约。
- 研究页数据同步列表在上，K 线图表在下，图表高度必须是首屏主体，不再平分当前可视窗口。
- 回测详情和交易详情采用上方大图表、下方左窄摘要右宽 tab 列表布局。
- 研究页图表工具栏收敛为紧凑 market strip：交易所、交易对、刷新、周期、时间窗口控件均有明确宽度边界，交易对输入不再占用大段横向空间。
- 图表左侧保留可读 gutter，右侧只保留必要价格轴和 2px 级外边距；主图 canvas 必须贴住右侧价格轴，价格轴必须贴近 viewport 右边。
- `TradingViewChart` 只按固定外部 viewport 尺寸渲染，图表库内部 DOM / canvas 高度污染不能反向撑高页面。
- visual smoke 必须覆盖研究页、回测详情、交易详情的图表高度、左右 gutter、右侧价格轴宽度、主图占比、工具栏宽度和高度。

范围外：

- 不新增指标、画线、盘口、策略叠加等研究功能。
- 不改变 CandleProvider、数据同步、回测、交易 API 语义。
- 不新增后端字段、migration 或 worker 行为。
- 不做像素快照基线和跨浏览器矩阵。

用户可见行为：

- 研究页打开后先看到任务列表，下面是足够高、左右边距正常的 K 线图表。
- 图表工具项在桌面视口保持单行紧凑；窄屏可横向滚动但不撑宽页面。
- 回测详情和交易详情均以上方大图表为主体，下方再展示摘要和列表信息。
- 右侧价格轴不再形成明显空白带，图表内容不被截掉、不无限拉高。

前端验收：

- `klineChartLayout.css` 继续作为图表固定 viewport 单一契约来源。
- `ResearchPage.css` 不重复定义图表高度和左右 gutter，只定义工具栏和研究页布局。
- `detailChartLayout.css` 不重复定义图表高度和左右 gutter。
- `TradingViewChart.vue` 的 resize 逻辑不信任内部图表 DOM 尺寸。

测试验收：

- `ResearchPage.layout.test.ts`
- `DetailPages.layout.test.ts`
- `TradingViewChart.test.ts`
- `scripts/research-chart-height-smoke.mjs`
- `scripts/stage8-visual-smoke.mjs`

质量门禁：

- `pnpm --dir web/frontend exec vitest run src/pages/ResearchPage.layout.test.ts src/pages/DetailPages.layout.test.ts src/components/chart/TradingViewChart.test.ts`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `go test ./...`
- `go vet ./...`
- `scripts/quality-gate.sh`
- `git diff --check`
- 本地 `api` 服务 readyz
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=800 node scripts/research-chart-height-smoke.mjs`
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=800 node scripts/stage8-visual-smoke.mjs`

当前验证：

- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts src/pages/ResearchPage.layout.test.ts src/pages/DetailPages.layout.test.ts` 通过：3 个测试文件、36 条测试。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：31 个测试文件、155 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- `docker compose build api && docker compose up -d api && curl -fsS http://127.0.0.1:8080/readyz` 通过，`readyz` 返回 `{"status":"ok"}`。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=800 SMOKE_SAMPLES=4 SMOKE_INTERVAL_MS=120 node scripts/research-chart-height-smoke.mjs` 通过：1440x900、2048x1152、812x1320、390x844 连续采样高度稳定。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=800 node scripts/stage8-visual-smoke.mjs` 通过：1440 / 812 / 390 视口、浅 / 深主题、`zh-CN/en-US`，每组 14 页，最大 document width 不超过对应 viewport。
- Headless Chrome 几何采样：1440 视口 toolbar `43px`、controls `488px`、symbol `96px`、price axis `42px`、main share `96.9%`、right edge gap `0px`；812 视口 toolbar `72px`、controls `482px`、symbol `96px`、price axis `42px`、main share `94.5%`、right edge gap `0px`；390 视口 toolbar `78px`、symbol `92px`、price axis `44px`、main share `87.2%`、right edge gap `0px`。

失败项：

- 首次严格 visual smoke 失败：移动端详情页完整价格轴下主图占比低于 `90%`，说明用固定过高占比压缩价格轴会牺牲可读性；已改为完整价格优先，移动端主图占比下限调整为 `87%`，并继续断言价格轴贴边、无额外右侧空白。
- 首次质量门禁失败：新增测试让 `TradingViewChart.test.ts` 达到 665 行，超过 650 行硬上限；已合并窄屏价格格式测试并把文件降回 650 行。

剩余风险：

- 本轮只收敛 K 线图表布局、工具栏密度和浏览器几何回归，不新增图表交互能力。
- 价格轴最大宽度现在按完整价格标签允许到 `48px`；后续如果支持更多位数资产或法币价格，需要按真实数据重新校准标签格式和轴宽。
- 当前浏览器验证基于 Headless Chrome，仍缺像素快照基线和多浏览器视觉回归。

### 阶段 1 data sync 创建窗口校验补充

执行日期：2026-06-30

目标等级：scaffold。

范围内：

- `POST /api/data/tasks` 和 `Store.CreateDataSyncTask` 共享 `ValidateDataSyncTaskWindow`。
- 创建 data sync task 时校验 interval 必须是当前 data sync 入口明确支持的 `1m/5m/15m/1h/4h/1d`。
- 创建 data sync task 支持无窗口或单边窗口；当 `startTime/endTime` 同时存在时必须满足 `startTime < endTime`。
- 研究页创建弹窗在反向或相等窗口时禁用创建，并展示明确错误。
- 单元测试、API 测试、PostgreSQL 集成测试和前端组合逻辑测试覆盖非法 interval、相等窗口和反向窗口。

范围外：

- 不要求 data sync 任务必须有 `startTime/endTime`。
- 不限制最大同步跨度、不做自动拆分。
- 不推进实盘交易所私有 API、live executor、订单提交、撤单、查单或幂等实盘下单。

剩余风险：

- data sync 仍缺完整统一状态机、自动批量补全、真实交易所长期恢复压测和多实例共享限流。
- 项目整体仍是 `scaffold`，不能升级。

当前验证：

- `go test ./internal/data -run TestValidateDataSyncTaskWindow` 通过。
- `go test ./internal/web/api -run TestDataSyncTaskRoutesRejectInvalidIntervalAndWindow` 通过。
- `go test ./internal/store/postgres -run TestIntegrationCreateDataSyncTaskRejectsInvalidIntervalAndWindow` 通过。
- `pnpm --dir web/frontend exec vitest run src/composables/useResearchWorkspace.test.ts` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：32 个测试文件、158 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `git diff --check` 通过。

失败项：

- 首次 targeted 测试暴露 `IntervalDuration("2m")` 会被当作可换算时长；已改为 data sync 创建入口显式支持 `1m/5m/15m/1h/4h/1d`，`2m` 会返回 `unsupported data sync interval "2m"`。
- 首次 `scripts/quality-gate.sh` 失败于前端文件行数上限；已把 `useResearchWorkspace.ts` 压回 400 行、`useResearchWorkspace.test.ts` 压回 649 行并复跑通过。

### 阶段 1 data sync repair 创建校验补充

执行日期：2026-06-30

目标等级：scaffold。

范围内：

- market 全历史缺口单个 repair、批量 repair 和全历史异常 repair 的 API 校验复用 `ValidateDataSyncTaskWindow`，不再只用 `IntervalDuration` 判断周期。
- market repair store 入口和最终 `insertMarketCandleRepairTask` 插入点都复用 data sync 支持周期和窗口顺序校验。
- task 窗口缺口 repair、单缺口 repair、异常 repair 在读取源任务后校验源任务 interval，最终 `insertDataSyncRepairTask` 也校验 repair window。
- 直接调用 store 或绕过 API 时，`2m` 等非 data sync 支持周期不能创建补同步任务。

范围外：

- 不新增 data sync 支持周期。
- 不改变 CandleProvider 聚合周期或图表周期选择。
- 不推进实盘交易所私有 API、live executor、订单提交、撤单、查单或幂等实盘下单。

当前验证：

- `go test ./internal/web/api -run TestMarketCandleRepairRoutesRejectUnsupportedDataSyncInterval` 通过。
- `go test ./internal/store/postgres -run 'TestIntegration(DataSyncTaskRepairsRejectUnsupportedInterval|MarketCandleRepairsRejectUnsupportedDataSyncInterval)'` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：32 个测试文件、158 条测试。
- `pnpm --dir web/frontend run build` 通过。

失败项：

- 首次 PostgreSQL targeted 测试编译失败：新增测试用了 `strings` 但所在文件没有 import；已改为精确比较错误文本且保持文件 650 行上限内。

剩余风险：

- data sync 仍缺完整统一状态机、自动批量补全、真实交易所长期恢复压测和多实例共享限流。
- 项目整体仍是 `scaffold`，不能升级。

### 阶段 1 全历史 market repair active catalog store 守卫补充

执行日期：2026-06-30

目标等级：scaffold。

范围内：

- 全历史 `RepairMarketCandleGap`、`RepairMarketCandleGaps` 和 `RepairMarketCandleInvalidIssues` 在同一 PostgreSQL 事务内先校验 `market_instruments` exact active 记录。
- 该校验下沉到 store 层共享 helper，直接调用 store 时也不能绕过 API 层 active market 守卫。
- inactive 或 missing market 会返回 `market_instrument_not_active` 领域错误，且不会插入 `data_sync_tasks` 补同步任务。
- 成功 repair 的 PostgreSQL fixture 明确插入 active instrument，避免测试依赖无 catalog 市场。
- legacy invalid close 测试 fixture 临时放开并恢复 OHLC bounds 约束，使真实 PostgreSQL 中历史异常 close 行可被扫描和分页测试覆盖。

范围外：

- 不实现自动批量修复。
- 不改变 worker 调度、repair 上限或已存在的 API 请求/响应语义。
- 不推进实盘交易所私有 API、live executor、订单提交、撤单、查单或幂等实盘下单。

当前验证：

- 真实 PostgreSQL targeted 测试通过：`TestIntegrationRepairMarketCandleInvalidIssuesCreatesSyncTasks`、`TestIntegrationRepairMarketCandleInvalidIssueConvergesFullHistoryScan`、`TestIntegrationRepairMarketCandleGapCreatesSyncTask`、`TestIntegrationMarketCandleRepairsRequireActiveMarketInstrument`、`TestIntegrationRepairMarketCandleGapIgnoresSoftDeletedRepairTask`、`TestIntegrationRepairMarketCandleGapsCreatesSyncTasks`、`TestIntegrationRepairMarketCandleGapsRollsBackWhenAnyGapIsInvalid` 和 `TestIntegrationListDataSyncTasksReportsInvalidCandleHealth`。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：32 个测试文件、158 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。

失败项：

- 首次宿主 targeted 集成测试因未设置 `TICTICK_TEST_DATABASE_URL` 被跳过；已改用当前 Docker Compose PostgreSQL 容器所在 network 运行真实 PostgreSQL targeted 测试。
- 首次真实 PostgreSQL targeted 测试暴露 legacy invalid close fixture 会违反 `market_candles_ohlc_bounds_check`；已为该测试 helper 临时放开并恢复 OHLC bounds 约束。
- 随后真实 PostgreSQL targeted 测试暴露 invalid issue 列表旧断言只期望 2 条；已修正为覆盖 3 条历史异常和分页 `Limited=true` 语义。
- 首次 `scripts/quality-gate.sh` 失败于 `market_candle_gap_store_integration_test.go` 超过 700 行；已把新增 active catalog 守卫测试拆到独立文件并复跑通过。

剩余风险：

- 该守卫只证明补同步任务排队边界收紧，不代表自动修复、真实交易所一定返回缺失数据或长期多实例同步已验证。
- data sync 仍缺完整统一状态机、自动批量补全、真实交易所长期恢复压测和多实例共享限流。
- 项目整体仍是 `scaffold`，不能升级。

### 阶段 1 K 线图表坐标轴字号恢复补充

执行日期：2026-07-01

目标等级：scaffold。

范围内：

- `TradingViewChart` 坐标轴字体恢复为统一 `14px`，该约束覆盖前文仍保留的桌面 `12px` / 移动 `10px` 旧记录。
- 右侧价格轴最小宽度调整为桌面 `76px`、窄桌面 `74px`、移动端 `72px`，继续保持完整数值显示，不恢复 `k/m/b` 缩写。
- browser visual smoke 的右侧价格轴上限调整到 `84px`，主图占比下限调整为桌面 `94%`、窄桌面 `90%`、移动端 `78%`；价格轴贴住 viewport 右边、主图贴住价格轴左边和无额外右侧空白断言继续保留。
- 研究页、回测详情和交易详情继续复用同一图表组件，字号恢复一次覆盖三处图表。

范围外：

- 不改 K 线数据、同步、回测、交易或实盘链路。
- 不恢复价格缩写。
- 不改变研究页工具栏和详情页上下双栏布局。

当前验证：

- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts` 通过：21 条测试。
- `scripts/check-research-chart-layout.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：32 个测试文件、158 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `docker compose up -d --build api` 通过，`curl -fsS http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`。
- `scripts/quality-gate.sh` 通过。
- 最新前端产物经本地 API `http://127.0.0.1:8080` 验证，`BASE_URL=http://127.0.0.1:8080 SMOKE_SAMPLES=4 SMOKE_INTERVAL_MS=120 SMOKE_SETTLE_MS=800 node scripts/research-chart-height-smoke.mjs` 通过。
- 最新前端产物经本地 API `http://127.0.0.1:8080` 验证，`BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=800 node scripts/stage8-visual-smoke.mjs` 通过：桌面、窄桌面、移动端，浅/深主题，中/英文共 12 组矩阵，每组 14 页。

失败项：

- 首次 chart 单元测试失败于 dense candle window 的 logical range 精确预期；原因是右侧价格轴恢复正常宽度后主图 plot width 变化，已更新预期并复跑通过。

剩余风险：

- 该补充只恢复坐标轴基础可读字号，不代表图表研究能力达到 usable，也不替代像素快照基线或多浏览器视觉回归。
- 价格轴可读性与主图宽度仍是取舍；后续支持更多价格位数或不同资产时，需要按真实数据重新校准轴宽。
- 项目整体仍是 `scaffold`，不能升级。

### 阶段 1 data sync restart smoke 门禁入口补充

执行日期：2026-06-30

目标等级：scaffold。

范围内：

- `scripts/full-quality-gate.sh` 增加 `FULL_QUALITY_STAGE1_RESTART=1` 可选门禁入口。
- 该入口运行 `scripts/stage1-data-sync-restart-smoke.sh`，用 Docker Compose、API、PostgreSQL、`hi sync` worker 和本地 Binance K 线 mock 串起恢复验收。
- smoke 覆盖过期 realtime lease 被新 sync worker 重新 claim、从持久化 cursor overlap 继续拉取、upsert 复写 overlap K 线、推进 `last_synced_open_time`、释放 lease，并通过 `/api/data/tasks` 观察恢复后的任务状态和数据健康。
- 默认 full gate 仍保持轻量；需要阶段 1 重型恢复验收时显式设置 `FULL_QUALITY_STAGE1_RESTART=1`。

范围外：

- 不改变 data sync runner / store 行为。
- 不把 stage1 restart smoke 加入默认 CI 必跑矩阵。
- 不实现真实外部交易所长期压测、多实例 soak、分布式 token bucket 或实盘交易。

当前验证：

- `bash -n scripts/full-quality-gate.sh scripts/stage1-data-sync-restart-smoke.sh` 通过。
- `FULL_QUALITY_STAGE1_RESTART=1 scripts/full-quality-gate.sh` 通过；其中 `stage1 data sync restart smoke` 通过，验证样本为 `task=dst_s1restart_1782834280`、`cursor=2026-01-01T00:04:00Z`、`klinesHits=1`。

剩余风险：

- 该入口只让既有恢复 smoke 进入统一门禁调度，不新增恢复语义。
- smoke 仍依赖 Docker Compose、本地 PostgreSQL、headless-free API 环境和本地 market mock；它不是长期真实交易所恢复压测。
- 项目整体仍是 `scaffold`，不能升级。

### 阶段 1 重型数据门禁总入口补充

执行日期：2026-06-30

目标等级：scaffold。

范围内：

- `scripts/full-quality-gate.sh` 增加 `FULL_QUALITY_STAGE1=1` 阶段 1 重型数据验收总开关。
- `FULL_QUALITY_STAGE1=1` 会在通用完整门禁后连续运行 `scripts/stage1-data-sync-restart-smoke.sh` 和 `scripts/stage1-candle-provider-perf-smoke.sh`。
- 保留单项开关：`FULL_QUALITY_STAGE1_RESTART=1` 只运行 data sync restart smoke，`FULL_QUALITY_STAGE1_CANDLE_PERF=1` 只运行 CandleProvider 大窗口聚合性能 smoke。
- README 记录 Stage 1 总开关和两个单项开关，避免阶段 1 重型验收入口只存在于脚本实现里。

范围外：

- 不改变 CandleProvider、data sync runner、PostgreSQL schema 或 API 行为。
- 不把 Stage 1 重型 smoke 加入默认 CI 必跑矩阵。
- 不关闭长期 soak、冷缓存、真实生产数据分布、多实例共享额度或真实交易所恢复压测风险。
- 不推进实盘交易所私有 API、live executor 或订单提交。

当前验证：

- `bash -n scripts/full-quality-gate.sh scripts/stage1-data-sync-restart-smoke.sh scripts/stage1-candle-provider-perf-smoke.sh` 通过。
- `FULL_QUALITY_STAGE1=1 scripts/full-quality-gate.sh` 通过；其中 `stage1 data sync restart smoke` 验证样本为 `task=dst_s1restart_1782834897`、`cursor=2026-01-01T00:04:00Z`、`klinesHits=1`。
- 同一次 Stage 1 总门禁中的 `stage1 candle provider perf smoke` 通过；真实 PostgreSQL 中读取 `240000` 根基础 `1m` K 线并聚合成 `1000` 根 `4h` K 线，查询耗时 `2.535631876s`。

剩余风险：

- 该补充只把已有 Stage 1 重型 smoke 收敛到统一入口，不新增业务语义。
- `stage1-candle-provider-perf-smoke.sh` 仍是短时真实 PostgreSQL 大窗口聚合 smoke，不代表长期冷缓存或真实生产分布压测。
- `stage1-data-sync-restart-smoke.sh` 仍依赖本地 Docker Compose 和 mock exchange，不代表真实外部交易所长期恢复压测。
- 项目整体仍是 `scaffold`，不能升级。

### 阶段 1 研究页全历史缺口修复反馈补充

执行日期：2026-07-01

目标等级：scaffold。

范围内：

- 全历史缺口标签补齐键盘 Space 打开详情能力，与 Enter / 点击交互一致。
- 前端组件测试覆盖全历史缺口 repair 后会触发第二次 `scanMarketCandleGaps`。
- 前端组件测试进一步覆盖首次扫描有缺口、repair 后第二次扫描返回健康时，弹窗切换为“全历史未检测到缺口”，同时仍显示本次 repair result 和创建出的补同步任务窗口。

范围外：

- 不把“排队补同步”伪装成缺口已经被交易所数据修复。
- 不改变后端 gap scan / repair API、补同步任务调度或 worker 写回语义。
- 不推进实盘交易所私有 API、live executor 或订单提交。

当前验证：

- `pnpm --dir web/frontend exec vitest run src/components/research/MarketCandleGapTag.test.ts` 通过：7 条测试。

剩余风险：

- repair 后重新扫描仍只代表当前 API 返回的扫描状态；真实修复结果依赖 data sync worker 成功写回和后续 scan 观察。
- 当前仍缺长期真实交易所恢复压测、像素快照基线和多浏览器视觉回归，研究页不能升级。
- 项目整体仍是 `scaffold`，不能升级。

### 阶段 1 研究页图表缺口修复后刷新补充

执行日期：2026-07-01

目标等级：scaffold。

范围内：

- 研究页图表上的“修复首个缺口”在 repair API 成功返回后，不再只刷新 data sync 任务列表。
- `repairChartGap` 会同时刷新任务列表和当前 CandleProvider 查询结果，使图表 metadata / gaps 能在 repair 排队后重新读取最新 API 状态。
- 前端 composable 测试覆盖无源任务的全历史 market gap repair 路径和选中源任务的 task gap repair 路径，均会在 repair 成功后重新调用 `getCandles`。

范围外：

- 不把“repair API 成功返回”解释为缺口已经实际修复；真实修复仍依赖 data sync worker 后续写回。
- 不改变后端 gap repair API、CandleProvider 查询语义或 worker 调度。
- 不推进实盘交易所私有 API、live executor 或订单提交。

当前验证：

- `git diff --check` 通过。
- `pnpm --dir web/frontend exec vitest run src/composables/useResearchWorkspace.test.ts` 通过：18 条测试。
- `pnpm --dir web/frontend run test` 通过：160 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- 本地 `docker compose up -d --build api` 通过，`http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`，`/research` 返回 HTTP 200。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=800 node scripts/stage8-visual-smoke.mjs` 通过：1440x900、812x1320、390x844 三个视口 × light/dark × zh-CN/en-US，每组 14 页，最大 document width 均等于视口宽度。

剩余风险：

- repair 后立即刷新可能仍看到原缺口，这是正确状态，表示补同步任务尚未写回。
- 当前仍缺真实交易所长期恢复压测、自动完成后的前端轮询/通知机制和像素视觉回归，研究页不能升级。
- 项目整体仍是 `scaffold`，不能升级。

### 阶段 1 研究页图表缺口修复结果可观察性补充

执行日期：2026-07-01

目标等级：scaffold。

范围内：

- 研究页图表“修复首个缺口”不再只通过 toast 表达结果。
- `repairChartGap` 返回后端 repair metadata，研究页状态条复用 `MarketRepairResultTags` 展示本次匹配数量、创建数量、跳过数量、单次上限、limited 状态和创建出的补同步窗口。
- 切换交易所、交易对、周期或重新加载 K 线后清除旧的图表缺口修复结果，避免把上一个数据源的 repair 结果留在当前图表上下文。
- 前端测试覆盖 repair action 返回 metadata、刷新任务/K线、页面挂载 repair result tags。

范围外：

- 不改变后端 gap repair API、去重、单次修复上限或 data sync worker 调度语义。
- 不实现 repair 任务完成后的自动轮询或通知。
- 不推进实盘交易所私有 API、live executor 或订单提交。

当前验证：

- `git diff --check` 通过。
- `pnpm --dir web/frontend exec vitest run src/composables/researchGapRepairActions.test.ts src/pages/ResearchPage.layout.test.ts src/components/research/MarketRepairResultTags.test.ts` 通过：17 条测试。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：163 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。

剩余风险：

- repair result 仍只是“已排队/已跳过”的返回结果，不代表缺口已经被实际数据写回。
- repair 任务完成后仍缺自动完成状态反馈、轮询/通知机制和浏览器像素快照基线，研究页不能升级。
- 项目整体仍是 `scaffold`，不能升级。

### 阶段 1 研究页补同步任务状态可观察性补充

执行日期：2026-07-01

目标等级：scaffold。

范围内：

- `MarketRepairResultTags` 展示 repair result 中创建出的补同步任务时，不再只显示任务 ID 和窗口。
- 每个补同步任务 tag 会同时展示任务 `status` 和 `dataHealth`，并按数据健康状态映射 tag 类型。
- 如果研究页已有最新 `data sync tasks` 列表，repair result tag 会优先按任务 ID 使用列表中的最新状态覆盖 repair API 返回时的任务快照。
- 研究页图表缺口 repair、全历史缺口 repair 和全历史异常 repair 的结果展示均接收当前任务列表，用户刷新任务列表后能在同一 repair result 区域观察补同步任务状态变化。

范围外：

- 不实现后台自动轮询、任务完成通知或 worker 推送。
- 不改变后端 repair API、任务状态机、data sync worker 调度或 K 线写回语义。
- 不推进实盘交易所私有 API、live executor 或订单提交。

当前验证：

- `git diff --check` 通过。
- `pnpm --dir web/frontend exec vitest run src/components/research/MarketRepairResultTags.test.ts src/pages/ResearchPage.layout.test.ts src/components/research/MarketCandleGapTag.test.ts src/components/research/MarketCandleInvalidIssueTag.test.ts` 通过：30 条测试。
- `pnpm --dir web/frontend run typecheck` 通过。
- `scripts/check-file-size.sh` 通过。
- `pnpm --dir web/frontend run test` 通过：164 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。

剩余风险：

- 该能力只显示已有任务列表里的最新状态；没有自动轮询时，用户仍需要刷新任务列表或触发现有刷新路径。
- repair 任务完成后仍缺自动完成状态反馈、通知机制和真实外部交易所恢复压测，研究页不能升级。
- 项目整体仍是 `scaffold`，不能升级。

### 阶段 1 K 线图表坐标轴可读性二次恢复补充

执行日期：2026-07-01

目标等级：scaffold。

范围内：

- `TradingViewChart` 坐标轴字体从 `14px` 恢复到统一 `16px`，覆盖研究页、回测详情和交易详情共用 K 线图表。
- 右侧价格轴继续显示完整数值，不恢复 `k/K/m/M/b/B` 紧凑缩写。
- 右侧价格轴 minimumWidth 调整为桌面 `82px`、窄桌面 `80px`、移动端 `76px`，为 16px 完整价格标签保留绘制空间，同时避免移动端主图被价格轴过度挤压。
- `scripts/stage8-visual-smoke.mjs` 和 `scripts/research-chart-height-smoke.mjs` 的右侧价格轴上限调整为 `96px`，仍继续断言主图贴住价格轴、价格轴贴住图表 viewport 右边界，避免额外右侧空白。
- 两条浏览器 smoke 新增坐标轴 canvas 文字墨迹高度检查，低于 `10px` CSS 像素会失败，避免只检查配置对象而真实渲染仍偏小。
- `scripts/check-research-chart-layout.sh` 增加轴字号、轴宽和真实像素 smoke guard 的源码检查。

范围外：

- 不新增技术指标、绘图工具、K 线交互分析或图表快捷交易。
- 不改变 CandleProvider、data sync worker、repair API 或后端数据语义。
- 不推进实盘交易所私有 API、live executor 或订单提交。

当前验证：

- `git diff --check` 通过。
- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts` 通过：21 条测试。
- `scripts/check-research-chart-layout.sh` 通过。
- `node --check scripts/stage8-visual-smoke.mjs` 通过。
- `node --check scripts/research-chart-height-smoke.mjs` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：164 条测试。
- `pnpm --dir web/frontend run build` 通过。
- 本地 `docker compose up -d --build api` 通过，`http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`，`/research` 返回 HTTP 200。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SAMPLES=4 SMOKE_INTERVAL_MS=120 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：1440x900、2048x1152、812x1320、390x844 四个视口高度稳定，移动端 body/chart/tv 均为 `580px`。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=800 node scripts/stage8-visual-smoke.mjs` 通过：1440x900、812x1320、390x844 三个视口 × light/dark × zh-CN/en-US，每组 14 页，最大 document width 均等于视口宽度。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。

修正过程：

- 首次 `research-chart-height-smoke` 在移动端因 `80px` 价格轴导致主图占比降到 `0.767` 失败；已把移动端价格轴收敛到 `76px`。
- 第二次 `research-chart-height-smoke` 在移动端主图占比 `0.779` 与旧 `0.780` 阈值仅差 `0.001`；已把移动端主图占比阈值校准到 `0.775`，同时保留价格轴贴边、主图贴价格轴和无额外右侧空白断言。

剩余风险：

- 该补充只恢复坐标轴可读性和几何 guard；仍没有像素快照基线、多浏览器视觉回归或设计评审关闭。
- 价格轴可读性与主图宽度仍是取舍；后续支持更长价格文本或更多资产报价精度时，需要重新校准轴宽。
- 项目整体仍是 `scaffold`，不能升级。

### 阶段 1 研究页补同步任务状态自动刷新补充

执行日期：2026-07-01

目标等级：scaffold。

范围内：

- 研究页新增有界 repair task 状态轮询 composable：默认每 4 秒刷新一次 data sync task 列表，最多 6 次，重复启动会取消上一轮，组件卸载会清理定时器。
- 图表首个缺口 repair、任务窗口缺口 repair、全历史缺口 repair、全历史异常 repair 和任务窗口异常 repair 成功排队后，都会启动同一轮有界任务列表刷新。
- 如果 repair 入口本身已经同步刷新过任务列表，轮询第一轮延后执行，避免立刻重复打同一个任务列表请求。
- 任务窗口缺口详情弹窗和任务窗口异常详情弹窗复用 `MarketRepairResultTags`，并接收研究页最新 tasks；repair result 中创建出的补同步任务会随任务列表刷新显示最新 `status` 和 `dataHealth`。
- 前端测试覆盖轮询 immediate / delayed / restart / unmount 清理行为，以及研究页所有 repair 入口接线。

范围外：

- 不实现 WebSocket / SSE / worker push。
- 不把“补同步任务已排队”解释为缺口或异常已经实际修复；最终收敛仍依赖 data sync worker 写回健康 K 线。
- 不改变后端 repair API、任务状态机、data sync worker 调度或 K 线写回语义。
- 不推进实盘交易所私有 API、live executor 或订单提交。

当前验证：

- `git diff --check` 通过。
- `pnpm --dir web/frontend exec vitest run src/composables/useResearchRepairTaskPolling.test.ts src/pages/ResearchPage.layout.test.ts src/components/research/MarketRepairResultTags.test.ts src/components/research/ResearchTaskInvalidIssueModal.test.ts` 通过：29 条测试。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：34 个测试文件、169 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- 本地 `docker compose up -d --build api` 通过，`http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`，`/research` 返回 HTTP 200。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=800 node scripts/stage8-visual-smoke.mjs` 通过：1440x900、812x1320、390x844 三个视口 × light/dark × zh-CN/en-US，每组 14 页，最大 document width 均等于视口宽度。

剩余风险：

- 轮询是有界兜底，不是实时订阅；长时间运行或卡住的补同步任务仍可能在轮询结束后需要用户手动刷新。
- 真实交易所返回空窗口、持续异常或限流时，仍依赖现有 failed/retry/backoff/数据健康路径暴露。
- 当前仍缺真实外部交易所恢复压测、像素快照和多浏览器视觉回归，研究页不能升级。
- 项目整体仍是 `scaffold`，不能升级。

### 阶段 1 K 线图表坐标轴字号恢复补充

执行日期：2026-07-01

目标等级：scaffold。

范围内：

- 研究页、回测详情和交易详情共享 K 线图表轴字号从 `16px` 提升到 `18px`。
- 右侧价格轴 minimumWidth 调整为桌面 `92px`、窄桌面 `90px`、移动端 `86px`，继续保持完整价格数值显示，不恢复 `k/K/m/M/b/B` 紧凑缩写。
- 研究页图表布局硬门禁同步锁定 `18px` 轴字号和新的价格轴宽度，避免后续回退到小字号。
- 浏览器 visual smoke 把坐标轴文字墨迹高度下限提升到 `13px`，并同步校准价格轴最大宽度与主图占比阈值。

范围外：

- 不改变 K 线数据查询、同步、回测、交易或订单执行语义。
- 不引入 TradingView 高级图表库或新的图表依赖。
- 不推进实盘交易所私有 API、live executor 或订单提交。

当前验证：

- `git diff --check` 通过。
- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts src/pages/ResearchPage.layout.test.ts src/pages/DetailPages.layout.test.ts` 通过：38 条测试。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：34 个测试文件、169 条测试。
- `pnpm --dir web/frontend run build` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/check-research-chart-layout.sh` 通过。
- `scripts/quality-gate.sh` 通过。
- 本地 `docker compose up -d --build api` 通过，`http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`，`/research` 返回 HTTP 200。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=800 node scripts/stage8-visual-smoke.mjs` 通过：1440x900、812x1320、390x844 三个视口 × light/dark × zh-CN/en-US，每组 14 页，最大 document width 均等于视口宽度。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SAMPLES=4 SMOKE_INTERVAL_MS=120 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：1440x900、2048x1152、812x1320、390x844 四个视口高度稳定。
- 真实 8080 Headless Chrome 聚焦采样：1440x900 研究页固定图表 viewport `1374x680`，右侧价格轴 canvas `92x640`，价格轴单行文字最大墨迹高度 `13px`，时间轴单行文字最大墨迹高度 `13px`。

剩余风险：

- 坐标轴可读性和主图宽度存在取舍；更长价格、更多小数位或超窄屏仍需要重新校准。
- 当前仍缺像素快照基线和多浏览器视觉回归，浏览器 smoke 只能覆盖当前 Chrome 矩阵。
- 项目整体仍是 `scaffold`，不能升级。

### 阶段 1 K 线图表坐标轴字号二次恢复补充

执行日期：2026-07-01

目标等级：scaffold。

范围内：

- 研究页、回测详情和交易详情共享 `TradingViewChart` 坐标轴字体从 `18px` 提升到 `20px`。
- 右侧价格轴 minimumWidth 调整为桌面 `100px`、窄桌面 `94px`、移动端 `88px`，避免大字号后右侧轴区继续挤压主图。
- 价格 formatter 继续输出完整数值，不恢复 `k/K/m/M/b/B` 紧凑缩写。
- 研究页图表布局硬门禁同步锁定 `20px` 轴字号和新的价格轴宽度。
- 浏览器 visual smoke 坐标轴文字墨迹高度下限从 `13px` 提高到 `14px`，并继续检查右侧价格轴贴住 viewport、主图贴住价格轴且没有额外右侧空白。

范围外：

- 不改变 K 线数据查询、同步、回测、交易或订单执行语义。
- 不引入新的图表库，不改工具栏布局。
- 不推进实盘交易所私有 API、live executor 或订单提交。

当前验证：

- `git diff --check` 通过。
- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts src/pages/ResearchPage.layout.test.ts src/pages/DetailPages.layout.test.ts` 通过：38 条测试。
- `scripts/check-research-chart-layout.sh` 通过。
- `docker compose up -d --build api` 通过，容器内前端 typecheck / build 通过，`http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`，`/research` 返回 HTTP 200。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SAMPLES=4 SMOKE_INTERVAL_MS=120 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：1440x900、2048x1152、812x1320、390x844 四个视口高度稳定。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=800 node scripts/stage8-visual-smoke.mjs` 通过：1440x900、812x1320、390x844 三个视口 × light/dark × zh-CN/en-US，每组 14 页，最大 document width 均等于视口宽度。
- `scripts/full-quality-gate.sh` 通过：`go test ./...`、`go vet ./...`、前端 typecheck / test / build 和轻量质量门禁均通过。

剩余风险：

- 当前仍是 Chrome headless 几何/像素 smoke，不是跨浏览器像素快照基线。
- 更长价格、小数位更多的品种或浏览器缩放比例仍可能需要继续校准价格轴宽度。
- 项目整体仍是 `scaffold`，不能升级。

### 阶段 1 K 线图表坐标轴字号恢复到正常尺寸补充

执行日期：2026-07-01

目标等级：scaffold。

范围内：

- 研究页、回测详情和交易详情共享 `TradingViewChart` 坐标轴字体从 `20px` 提升到 `26px`。
- 右侧价格轴 minimumWidth 调整为桌面 `122px`、窄桌面 `116px`、移动端 `102px`，继续显示完整价格数值，不恢复 `k/K/m/M/b/B` 紧凑缩写。
- 时间轴左右逻辑边缘 padding 提升到 `12px` 到 `18px`，避免 26px 时间轴标签贴住固定图表槽边缘。
- 研究页图表布局硬门禁同步锁定 26px 轴字号、新价格轴宽度和时间轴 padding。
- `scripts/stage8-visual-smoke.mjs` 与 `scripts/research-chart-height-smoke.mjs` 把坐标轴文字墨迹高度下限提高到 `18px`，并继续检查右侧价格轴贴住 viewport、主图贴住价格轴且没有额外右侧空白。

范围外：

- 不改变 K 线数据查询、同步、回测、交易或订单执行语义。
- 不引入新的图表库，不改研究页工具栏和详情页上下栏布局。
- 不推进实盘交易所私有 API、live executor 或订单提交。

当前验证：

- `git diff --check` 通过。
- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.test.ts src/pages/ResearchPage.layout.test.ts src/pages/DetailPages.layout.test.ts` 通过：38 条测试。
- `scripts/check-research-chart-layout.sh` 通过。
- `docker compose up -d --build api` 通过，容器内前端 typecheck / build 通过，`http://127.0.0.1:8080/readyz` 返回 `{"status":"ok"}`。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SAMPLES=4 SMOKE_INTERVAL_MS=120 SMOKE_SETTLE_MS=1000 node scripts/research-chart-height-smoke.mjs` 通过：1440x900、2048x1152、812x1320、390x844 四个视口高度稳定，且正式 `18px` 轴文字墨迹下限通过。
- 临时高阈值采样 `SMOKE_MIN_AXIS_LABEL_INK_HEIGHT=20` 在 1440x900 研究页按预期失败并暴露真实最大墨迹高度：右侧价格轴 canvas `122x626`，最大单行文字墨迹高度 `19px`。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=800 node scripts/stage8-visual-smoke.mjs` 通过：1440x900、812x1320、390x844 三个视口 × light/dark × zh-CN/en-US，每组 14 页，最大 document width 均等于视口宽度。
- `scripts/full-quality-gate.sh` 通过：`go test ./...`、`go vet ./...`、前端 typecheck / test / build 和轻量质量门禁均通过。

剩余风险：

- 当前仍是 Chrome headless 几何/像素 smoke，不是跨浏览器像素快照基线。
- 26px 坐标轴会占用更多右侧价格轴宽度；移动端为了完整价格可读，主图宽度占比阈值按实际渲染校准到 `0.665`，仍需后续按更多标的价格位数复核。
- 项目整体仍是 `scaffold`，不能升级。

### 阶段 1 真实 public exchange 数据同步 smoke 补充

执行日期：2026-07-01

目标等级：scaffold。

范围内：

- 默认跳过的真实 public exchange API 集成测试收敛为 Binance / OKX 共用路径：通过认证/CSRF API 创建 data sync task，调用 `/api/data/tasks/{id}/sync/start` 启动同步，再由真实 public market adapter 拉取 3 根已闭合 `1m` K 线。
- Binance 路径显式设置 `TICTICK_REAL_EXCHANGE_SMOKE=1` 后运行，默认 base URL 为 `https://data-api.binance.vision`，并支持 `TICTICK_REAL_BINANCE_SYMBOL` / legacy `TICTICK_REAL_EXCHANGE_SYMBOL` 覆盖交易对。
- OKX 路径需要同时设置 `TICTICK_REAL_EXCHANGE_SMOKE=1` 和 `TICTICK_REAL_OKX_SMOKE=1` 才运行，默认 base URL 为 `https://www.okx.com`，默认 symbol 为 `BTC-USDT`。
- 两条路径都验证 data sync runner 写入 PostgreSQL 后，任务列表可观察到 `succeeded`、`dataHealth=ok`、游标推进，并通过 `/api/candles` 观察到 `source=native`、`health=ok`、3 根真实 K 线。
- `scripts/stage1-real-exchange-data-sync-smoke.sh` 启动 compose PostgreSQL，创建临时 smoke database，默认运行 Binance 非跳过测试且 OKX 跳过；设置 `STAGE1_REAL_EXCHANGE_SMOKE_OKX=1` 时同时启用 OKX 非跳过测试；退出时 drop 临时 database，避免污染本地默认库。
- `scripts/full-quality-gate.sh` 增加 `FULL_QUALITY_STAGE1_REAL_EXCHANGE=1` / `FULL_QUALITY_STAGE1=1` 可选入口。

范围外：

- 不访问交易所私有 API，不读取或提交订单，不推进 live executor。
- 不把真实外网 smoke 放进默认 CI；该检查受网络、区域、交易所可用性影响，只能显式启用。
- 不声明 OKX 真实网络已通过；当前只增加 opt-in 测试入口并记录本地外网失败证据。
- 不声明长时间 soak、多实例共享额度或动态全交易所业务码分类已关闭。

当前验证：

- `go test ./internal/web/api -run 'TestIntegrationReal(Binance|OKX)DataSyncRouteServesNativeCandles' -count=1 -v` 通过编译并按预期跳过：未设置真实外网 env 时不访问外网。
- `scripts/stage1-real-exchange-data-sync-smoke.sh` 通过：临时 PostgreSQL database 中 Binance 非跳过执行，真实 `data-api.binance.vision` public K 线被 data sync runner 写入 PostgreSQL，并由 `/api/candles` 返回 native/ok；OKX 在未设置 `STAGE1_REAL_EXCHANGE_SMOKE_OKX=1` 时跳过。
- `STAGE1_REAL_EXCHANGE_SMOKE_OKX=1 scripts/stage1-real-exchange-data-sync-smoke.sh` 失败于当前环境访问 OKX public endpoint：Binance 部分通过，OKX `/api/v5/market/history-candles` 连续返回 `okx candles temporary unavailable: www.okx.com: EOF`，runner 按预期把任务置回 `pending`、记录 retry/backoff，并按 `nextAttemptAt` / `exchangeBackoffUntil` 做 6 次有界恢复尝试后仍未达到 `succeeded`；直接 `curl https://www.okx.com/api/v5/market/history-candles?...` 同样返回 TLS `SSL_ERROR_SYSCALL`。
- `docker run --rm --network tictick-hi_default ... go test ./internal/web/api -run TestIntegrationDataSyncRouteRecoversAfterTemporaryPublicMarketError -count=1 -v` 通过：真实 API handler 创建/启动 data sync task，本地 Binance-compatible endpoint 首次 `429 Retry-After` 后，任务进入 `pending/retrying` 且错误不泄露请求 URL，随后按 retry/backoff 恢复，写入 PostgreSQL，并由 `/api/candles` 返回 3 根 healthy native `1m` K 线。
- `FULL_QUALITY_STAGE1_REAL_EXCHANGE=1 scripts/full-quality-gate.sh` 通过：包含 `go test ./...`、`go vet ./...`、前端 typecheck / test / build、轻量质量门禁，并串联默认真实 public exchange data sync smoke。

剩余风险：

- 当前只实证通过 Binance public data API 的小窗口真实拉取；OKX 非跳过路径已接入但受当前网络/TLS EOF 阻塞，没有通过真实外网验收。
- 该证据证明真实 public K 线可进入 Stage 1 主路径，不代表多实例共享 token bucket、交易所动态额度或生产级外部依赖韧性已完成。
- 项目整体仍是 `scaffold`，不能升级。

### 阶段 1 数据同步外部临时失败恢复补充

执行日期：2026-07-01

目标等级：scaffold。

范围内：

- 新增 PostgreSQL 集成测试，串起 `data sync runner`、真实 PostgreSQL store、Binance / OKX public market adapter 和本地 `httptest` 交易所。
- 本地交易所第一次 `/api/v3/klines` 返回 `429` 与 `Retry-After`，验证 runner 不在同一轮快速重试，而是记录 task retry 和 exchange backoff，释放 lease，且不写入 K 线。
- 本地 OKX `/api/v5/market/history-candles` 连续返回两次业务码 `50011`，验证 runner 只做有限快速重试，耗尽后记录 task retry 和 exchange backoff，释放 lease，且不写入 K 线。
- 将 task / exchange backoff 推进到期后第二次运行 runner，验证同一任务成功拉取 5 根闭合 `1m` K 线、写入 `market_candles`、推进 `last_synced_open_time`、清理 task 错误与 exchange backoff，并在 `ListDataSyncTasks` 中显示 `dataHealth=ok`。
- OKX 恢复路径同样验证成功拉取 5 根闭合 `1m` K 线、写入 OKX `market_candles`、推进游标、清理 task / exchange retry 状态，并在 `ListDataSyncTasks` 中显示 `dataHealth=ok`。
- PostgreSQL 集成测试覆盖 exchange backoff 隔离：Binance 进入交易所级冷却后，同交易所 retry / sibling 任务不会被 claim，但 OKX pending task 仍可被 claim，避免单个交易所故障全局阻塞其它交易所同步。
- `scripts/stage1-data-sync-external-recovery-smoke.sh` 通过 Docker Compose postgres + 临时 Go 容器运行 Binance `Retry-After`、OKX `50011` 和 exchange backoff 隔离三条非跳过集成测试。
- `scripts/full-quality-gate.sh` 增加 `FULL_QUALITY_STAGE1_EXTERNAL_RECOVERY=1` / `FULL_QUALITY_STAGE1=1` 可选入口。

范围外：

- 不访问真实外网交易所，不声明真实外部网络长期压测已关闭。
- 不实现跨实例共享 token bucket、OKX 动态额度或交易所私有 API。
- 不推进 live executor、实盘下单、订单幂等或生产密钥管理。

当前验证：

- `scripts/stage1-data-sync-external-recovery-smoke.sh` 通过：目标测试在 compose PostgreSQL 上非跳过执行，Binance 第一次 `429 Retry-After` 记录 retry/backoff、第二次恢复并落库；OKX 两次 `50011` 后记录 retry/backoff、强制到期后第三次恢复并落库；Binance exchange backoff 激活期间 OKX sibling 任务仍可 claim。
- `FULL_QUALITY_STAGE1_EXTERNAL_RECOVERY=1 scripts/full-quality-gate.sh` 通过：包含 `go test ./...`、`go vet ./...`、前端 typecheck / test / build、轻量质量门禁，并串联上述 Stage 1 外部临时失败恢复 smoke。
- 2026-07-01 复核：`FULL_QUALITY_STAGE1_EXTERNAL_RECOVERY=1 scripts/full-quality-gate.sh` 再次通过；目标测试 `TestIntegrationDataSyncExchangeBackoffDoesNotBlockOtherExchanges`、`TestIntegrationDataSyncRunnerRecoversAfterBinanceRetryAfter` 和 `TestIntegrationDataSyncRunnerRecoversAfterOKXRateLimitCode` 均非跳过执行。

剩余风险：

- 该验证是确定性的本地交易所故障注入，不等同于真实 Binance / OKX 长时间网络恢复压测；OKX 当前真实外网 EOF 风险仍按上一段真实 public exchange smoke 记录保留。
- 该脚本是 Stage 1 可选重型门禁，默认 GitHub `Quality Gate` 仍不会强制拉 Docker Go 容器执行它。
- 项目整体仍是 `scaffold`，不能升级。

### 阶段 1 CandleProvider 受限聚合分页合同补充

执行日期：2026-07-01

目标等级：scaffold。

范围内：

- 补强 `CandleProvider` 单元测试：当聚合请求超过基础 `1m` 窗口上限并触发 `limitedByBaseWindow` 时，返回结果必须同时暴露 `previousCursor`，让调用方可以继续向前分页观察更早窗口。
- 验证受限聚合窗口不能伪装成完整结果：`health=insufficient`、`returnedCandles < requestedLimit`、`returnedBaseCandles == baseLimit` 和 `limitedByBaseWindow=true` 继续作为硬断言。
- 验证受限 latest 聚合窗口只暴露 previous pagination，不暴露 next cursor，避免把最新受限窗口误导为仍有后续数据。
- 验证 previous cursor 保留原始 `exchange/symbol/interval/limit` 查询上下文，前端和 API 可以继续使用 opaque cursor，而不是手工拼接相邻窗口语义。

范围外：

- 不实现超过 1440000 根基础 K 线的无限历史查询、持久化聚合缓存或分段聚合策略。
- 不改变 `CandleProvider` 运行语义、API contract、前端图表、回测或交易 runner。
- 不关闭长期 soak、冷缓存、真实生产数据分布和超过基础窗口上限的产品级查询策略风险。

当前验证：

- `go test ./internal/data -run TestCandleProviderReportsLimitedAggregationCoverage -count=1 -v` 通过。
- `go test ./internal/data -run TestCandleProvider -count=1` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过；首轮因 `internal/data/candle_provider_test.go` 超过 700 行硬上限失败，已拆出 `internal/data/candle_provider_limited_test.go` 后复跑通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过：34 个测试文件、169 条测试。
- `pnpm --dir web/frontend run build` 通过。

剩余风险：

- 该补充只是把受限窗口的 pagination 语义锁进测试，仍不是超过 1440000 根基础 K 线的完整查询方案。
- 仍缺真实 PostgreSQL 冷缓存、长时间 soak、真实生产数据分布和持久化聚合/分段策略。
- 项目整体仍是 `scaffold`，不能升级。

### 阶段 1 CandleProvider 聚合分页边界缺口补充

执行日期：2026-07-01

目标等级：scaffold。

范围内：

- 补强 `CandleProvider` 单元测试：当 `1m` 基础 K 线聚合更高周期时，如果缺失的基础 K 线刚好跨过内部 `MaxCandleLimit` 分页边界，聚合结果必须返回 `health=gap`。
- 测试同时断言 `requiredBaseCandles`、`baseLimit`、`returnedBaseCandles` 和 `returnedCandles`，确保分页边界缺口不会被误当成完整覆盖或基础窗口受限。
- 测试断言包含缺口的目标聚合窗口不会被返回，避免用不完整基础 K 线伪造高周期 K 线。
- 继续补强真实 PostgreSQL repository 集成测试：通过 `market_candles` seed 真实 `1m` 基础数据，跳过第 `MaxCandleLimit` 根基础 K 线，再通过 `store.GetCandles` 查询 `1h` 聚合，验证跨内部分页边界的缺口不会在 SQL 分页路径中被吞掉。
- 继续补强真实 `/api/candles` HTTP handler + PostgreSQL 集成测试：使用测试 operator session 访问 authenticated API，验证 query parser、handler、repository、JSON response 共同暴露同一分页边界缺口 metadata。

范围外：

- 不改变 `CandleProvider` 运行语义。
- 不新增聚合缓存、分段持久化或自动补同步策略。
- 不推进实盘交易、私有交易所 API 或 live executor。

当前验证：

- `go test ./internal/data -run TestCandleProviderReportsAggregationGapAcrossBasePageBoundary -count=1 -v` 通过。
- `go test ./internal/data -count=1` 通过。
- 主机未设置 `TICTICK_TEST_DATABASE_URL` 时，`go test ./internal/store/postgres -run TestIntegrationCandleProviderReportsAggregationGapAcrossBasePageBoundary -count=1 -v` 按预期跳过 PostgreSQL 集成用例。
- `docker run --rm --network tictick-hi_default ... go test ./internal/store/postgres -run TestIntegrationCandleProviderReportsAggregationGapAcrossBasePageBoundary -count=1 -v` 通过：真实 PostgreSQL 中 seed `5099/5100` 根 `1m` K 线，`store.GetCandles` 返回 `source=aggregated`、`health=gap`、`requiredBaseCandles=5100`、`returnedBaseCandles=5099`、`returnedCandles=84`，且不返回包含缺口的 `1h` 聚合 K 线。
- 主机未设置 `TICTICK_TEST_DATABASE_URL` 时，`go test ./internal/web/api -run TestIntegrationCandlesRouteReportsAggregationGapAcrossBasePageBoundary -count=1 -v` 按预期跳过 PostgreSQL API 集成用例。
- `docker run --rm --network tictick-hi_default ... go test ./internal/web/api -run TestIntegrationCandlesRouteReportsAggregationGapAcrossBasePageBoundary -count=1 -v` 通过：authenticated `/api/candles` response 返回 `source=aggregated`、`health=gap`、`requiredBaseCandles=5100`、`returnedBaseCandles=5099`、`returnedCandles=84`，且不返回包含缺口的 `1h` 聚合 K 线。

剩余风险：

- 该证据覆盖内存 store、真实 PostgreSQL repository 和 authenticated `/api/candles` handler 的分页边界缺口，但仍不替代真实 PostgreSQL 冷缓存、真实生产数据分布、长期 soak 或超过基础窗口上限后的持久化聚合/分段策略。
- 项目整体仍是 `scaffold`，不能升级。

### 阶段 1 回测 / 交易创建页 market catalog 可观察性补充

执行日期：2026-07-01

目标等级：scaffold。

范围内：

- 回测创建页和交易创建页复用的策略任务表单会持续校验当前 `exchange + symbol` 是否 exact 命中 `market_instruments` catalog。
- 表单顶部直接展示交易对目录状态：校验中、active、inactive、missing 或校验失败，并在摘要区显示当前市场状态。
- `inactive` / `missing` / 校验失败时创建按钮不可提交；直接调用 submit 时仍会在前端二次校验并给出明确错误，不调用 `POST /api/backtests` 或 `POST /api/trading/tasks`。
- catalog helper 保留 status-only API，同时新增返回 exact instrument 的 lookup API，用于展示交易所原始状态。
- 前端单元测试覆盖 inactive 回测创建阻断、missing 交易创建阻断、catalog 校验失败阻断；布局测试约束创建页必须保留 backend-backed symbol autocomplete 和 market status 展示。

范围外：

- 不改变后端 `POST /api/backtests` / `POST /api/trading/tasks` 的 active catalog 强校验。
- 不改变策略执行、回测撮合模型、paper executor 或任何 live executor 行为。
- 不实现实盘交易、私有交易所 API、订单幂等或账号安全边界。

当前验证：

- `pnpm --dir web/frontend exec vitest run src/composables/useStrategyTaskForm.test.ts src/pages/StrategyTaskFormPage.layout.test.ts` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过；新增逻辑拆分到独立 composable / 组件后，`useStrategyTaskForm.ts` 和 `StrategyTaskFormPage.vue` 均低于文件行数硬上限。
- `git diff --check` 通过。
- 临时启动 `hi api` 容器挂载当前 `web/frontend/dist` 到 `http://127.0.0.1:18080` 后，`BASE_URL=http://127.0.0.1:18080 SMOKE_TOTAL_TIMEOUT_MS=600000 node scripts/stage8-visual-smoke.mjs` 通过，覆盖 3 个 viewport、浅/深色、中/英文，每组 14 页。

剩余风险：

- 该证据只把创建页的 market catalog 状态从“提交后弹错”提升为“提交前可观察和可阻断”；不代表回测结果可信、交易任务可用于真实工作或实盘能力已经建立。
- 当前创建页仍依赖 `market_instruments` catalog 的同步新鲜度；catalog 外部同步失败时只能阻断创建并提示，不能自动恢复交易所目录。

### 阶段 1 market catalog 业务状态展示收敛补充

执行日期：2026-07-01

目标等级：scaffold。

范围内：

- 前端新增统一 market status formatter，把 `marketStatus`、`marketStatusDetail` 和 `exchangeStatus` 组合成用户可读文案。
- 研究页数据同步任务表不再直接显示 `BREAK`、`TRADING`、`not_returned`、`active`、`inactive`、`missing` 等原始状态；中文界面显示“可用 / 不可用 / 未入库”和“交易中 / 暂停交易 / 交易所未返回”等业务语义。
- 回测 / 交易创建页的 market catalog 状态组件复用同一 formatter，active / inactive 的交易所详情不再泄露 raw 状态码。
- 对未知交易所状态保留原值展示，避免为了美化隐藏真实外部状态。
- 前端单元测试覆盖 formatter 映射、创建页状态组件和研究页数据同步任务表 inactive 状态展示及操作阻断文案。

范围外：

- 不改变 `market_instruments` 数据模型、catalog 同步策略或后端 API contract。
- 不改变 data sync task 的 active catalog 守卫、claim 逻辑或 repair 调度。
- 不推进实盘交易、私有交易所 API、订单提交或风控边界。

当前验证：

- `pnpm --dir web/frontend exec vitest run src/utils/marketStatusDisplay.test.ts src/components/strategy/StrategyMarketCatalogStatus.test.ts src/components/tables/DataSyncTaskTable.test.ts` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- 本地 `http://127.0.0.1:18080` 当前构建服务上，`BASE_URL=http://127.0.0.1:18080 SMOKE_TOTAL_TIMEOUT_MS=600000 node scripts/stage8-visual-smoke.mjs` 通过，覆盖 3 个 viewport、浅/深色、中/英文，每组 14 页。

剩余风险：

- 该证据只收敛用户可见状态语义，不证明 catalog 新鲜度、真实交易所业务状态分类完整性或自动恢复能力。
- 未知状态仍原样展示；后续需要结合真实 Binance / OKX instrument 状态枚举继续细化。

### 阶段 1 CandleProvider 序列异常定位补充

执行日期：2026-07-01

目标等级：scaffold。

范围内：

- `validateCandleSeries` 对重复 open time 和乱序 open time 返回结构化 series issue error，保留原有错误文案，同时携带异常 K 线 `openTime`。
- `CandleProvider` 在返回 `invalid_native_series` 或 `invalid_aggregation_base_series` 时，会把该结构化 `openTime` 写入 `CandleIssue`。
- 聚合基础 K 线分页校验的跨页重复 / 乱序也返回结构化异常时间，避免只给出不可修复的泛化 message。
- 后端单元测试覆盖 native 重复、native 乱序、native 数值异常、aggregation base close time 异常和 aggregation base 跨页乱序的 issue `openTime`。
- 为保持质量门禁，CandleProvider invalid 场景测试拆到独立文件，避免 `candle_provider_test.go` 超过 700 行硬上限。

范围外：

- 不改变 CandleProvider 聚合算法、缺口检测、repair API、data sync worker 调度或补同步执行语义。
- 不新增自动清洗异常 K 线能力。
- 不推进回测可信度、交易 runner、live executor、私有交易所 API 或实盘能力。

当前验证：

- `go test ./internal/data` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

剩余风险：

- 该补充只让已有异常修复入口能够拿到具体异常 K 线时间；是否能修复仍依赖 repair API、data sync worker 和交易所返回健康数据。
- 当前仍缺长期冷缓存 / 真实生产数据分布压测、超过 1440000 根基础 K 线的缓存/分段策略，以及更完整的异常数据边界。

### 阶段 1 时间边界异常扫描补充

执行日期：2026-07-01

目标等级：scaffold。

范围内：

- 新增标准 K 线异常 code：`invalid_open_time` 和 `invalid_close_time`。
- `DetectCandleIssue`、任务窗口 invalid issue SQL、全历史 invalid issue SQL 都会检测 open time 是否按 interval UTC 对齐、close time 是否等于 `open_time + interval`。
- CandleProvider 对单根时间边界异常返回标准 `invalid_open_time` / `invalid_close_time`，不再把这类问题只落成泛化 series code。
- data sync 任务窗口 invalid issue 列表支持按新增 code 过滤；summary 能展示首个时间边界异常。
- 全历史 invalid scan 能展示时间边界异常；全历史 invalid repair 和任务 invalid repair 会跳过 `invalid_open_time`，避免为无法通过普通补同步删除的 misaligned 历史坏行创建假修复任务；`invalid_close_time` 仍可排补同步。
- 研究页图表、全历史异常标签和任务异常弹窗保留 `invalid_open_time` 展示，但不会为它显示普通补同步修复按钮；新增中英文文案避免暴露原始 code。
- 测试覆盖 Go 领域检测、PostgreSQL 任务窗口 scan/filter/repair、全历史 scan/repair、前端 repairable 判断和两个异常 repair 入口。

范围外：

- 不实现 misaligned open-time 历史坏行的删除、隔离或迁移清理策略。
- 不改变 K 线聚合算法、缺口检测、data sync worker 调度、交易所 adapter 或补同步执行语义。
- 不推进回测可信度、交易 runner、live executor、私有交易所 API 或实盘能力。

当前验证：

- `go test ./internal/data ./internal/store/postgres` 通过。
- `pnpm --dir web/frontend exec vitest run src/utils/candleIssues.test.ts src/components/research/MarketCandleInvalidIssueTag.test.ts src/components/research/ResearchTaskInvalidIssueModal.test.ts src/pages/ResearchPage.layout.test.ts` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

剩余风险：

- `invalid_open_time` 目前只可观察、可筛选、不可普通补同步修复；后续需要设计历史坏行清理/隔离边界，不能用补同步任务伪装为已修复。
- 时间边界异常扫描增强不代表长期冷缓存、真实生产数据分布压测、超过 1440000 根基础 K 线的缓存/分段策略已经关闭。

### 阶段 1 缺口扫描时间边界补强

执行日期：2026-07-01

目标等级：scaffold。

范围内：

- 任务窗口 gap CTE 保留全量 K 线用于 invalid summary，但新增只包含按 interval UTC 对齐 `open_time` 的 `gap_candles`，后续首段、相邻、尾段和整窗缺口都只用该集合计算。
- 全历史 market gap scan 的扫描窗口、相邻缺口列表和单个 / 批量 repair 校验都只使用按周期对齐的 K 线作为连续性边界。
- 错位 `open_time` 行继续通过 `invalid_open_time` 暴露，不再生成 `00:01 -> 00:01:30`、`00:02:30 -> 00:03` 这类无法由普通同步任务表达的假缺口窗口。
- PostgreSQL 集成测试覆盖任务窗口列表 / 单缺口拒绝 / 批量修复、全历史 scan / 单缺口拒绝 / repair，均验证错位 open time 不参与 gap 边界。

范围外：

- 不删除、不隔离、不迁移 misaligned 历史坏行。
- 不改变 CandleProvider 聚合算法、data sync worker 拉取策略、交易所 adapter 或补同步执行语义。
- 不推进回测可信度、交易 runner、live executor、私有交易所 API 或实盘能力。

当前验证：

- `go test ./internal/store/postgres -run 'TestIntegration(DataSyncTaskGapsIgnoreMisalignedOpenTimeCandles|MarketCandleGapsIgnoreMisalignedOpenTimeCandles)$'` 通过。
- `git diff --check` 通过。

剩余风险：

- `invalid_open_time` 仍然只是可观察、可筛选和在普通补同步中跳过；历史坏行的删除 / 隔离策略仍未设计。
- 本轮只证明错位 open time 不再污染 gap/repair 窗口，不代表长期冷缓存、真实生产数据分布压测或外部交易所返回质量已经关闭。

### 阶段 1 数据同步窗口 UTC 对齐校验补强

执行日期：2026-07-01

目标等级：scaffold。

范围内：

- `ValidateDataSyncTaskWindow` 不再只校验 interval 和 start/end 先后顺序；当 `startTime` 或 `endTime` 存在时，必须按对应 interval 的 UTC epoch 边界对齐。
- 创建 data sync task、任务单缺口 repair、全历史单缺口 repair 和全历史批量 gap repair 都复用同一窗口对齐规则。
- 任务单缺口 repair 的 API handler 会先读取源任务 interval，对存在的源任务提前返回 400，避免错位窗口绕过 HTTP 层进入 store。
- 全历史批量 gap repair 的 API handler 改为逐个 gap 校验对齐，不再只完整校验第一个 gap。
- 单元测试、API 测试和 PostgreSQL 集成测试覆盖错位 start/end 创建、错位任务 repair、错位全历史单个和批量 repair。

范围外：

- 不清理、不隔离、不迁移历史 misaligned K 线。
- 不改变 task window 的 end boundary 语义、CandleProvider 聚合算法、data sync worker 拉取策略或交易所 adapter。
- 不推进实盘交易、私有交易所 API、live executor 或订单幂等。

当前验证：

- `go test ./internal/data -run TestValidateDataSyncTaskWindow -count=1` 通过。
- `go test ./internal/web/api -run 'Test(DataSyncTaskRoutesRejectInvalidIntervalAndWindow|DataSyncTaskRoutes|MarketCandleGapRepairRouteQueuesSyncTask|MarketCandleGapBatchRepairRouteQueuesSyncTasks|MarketCandleRepairRoutesRejectUnsupportedDataSyncInterval)' -count=1` 通过。
- `go test ./internal/store/postgres -run 'TestIntegration(CreateDataSyncTaskRejectsInvalidIntervalAndWindow|DataSyncTaskGapsIgnoreMisalignedOpenTimeCandles|MarketCandleGapsIgnoreMisalignedOpenTimeCandles)$' -count=1` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/check-file-size.sh` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

剩余风险：

- 该补强只阻止新错位任务窗口进入系统，不处理已存在历史错位 K 线或可能已存在的旧错位任务窗口。
- HTTP handler 为了校验任务单缺口 repair 的 interval 会读取当前任务列表；后续如果补 `GetDataSyncTask` repository 方法，可以进一步收敛该路径。
- 项目整体仍为 `scaffold`，不能升级为 usable。

### 阶段 1 Definition of Done：研究核心

目标等级：usable。

范围内：

- CandleProvider 统一提供 native / aggregated K 线，返回 source、health、coverage、window、pagination、gap 和 invalid issue metadata。
- 数据同步任务能持久化 1m 原始 K 线，按闭合且连续的 `open_time` 推进游标，重启后可恢复。
- 内部周期聚合只基于健康、闭合、连续的 1m 基础 K 线；缺口、错位时间和异常 OHLCV 不能进入策略、回测或交易 runner 的输入。
- 研究页能创建/管理数据同步任务，展示 K 线、数据健康、缺口、异常、数据来源和补同步/隔离动作结果。
- 缺口修复和可修复异常走 data sync repair task；不可用普通补同步表达的 `invalid_open_time` 必须可观察并可隔离，不能被伪装成普通 repair。

范围外：

- 策略沉淀、回测可信撮合、模拟盘和实盘安全边界不在阶段 1 内升级。
- 聚合 K 线持久化缓存、tick 数据、长期生产数据分布压测和跨实例共享交易所额度不在阶段 1 内关闭。

质量门禁：

- `go test ./...`
- `go vet ./...`
- `pnpm --dir web/frontend run typecheck`
- `pnpm --dir web/frontend run test`
- `pnpm --dir web/frontend run build`
- `scripts/quality-gate.sh`
- 阶段 1 重型 smoke 按子能力启用：data sync restart、external recovery、real public exchange、CandleProvider perf。

当前判定：

- 阶段 1 尚未达到 usable；数据同步 worker、CandleProvider 和研究页多个子能力已有 demo 证据，但仍缺长期 soak、冷缓存、真实生产数据分布压测、完整自动收敛和更强操作语义。
- 项目整体仍为 `scaffold`。

### 阶段 1 invalid_open_time 隔离入口补充

执行日期：2026-07-01

目标等级：scaffold。

范围内：

- 新增 `market_candle_quarantines` 归档表，保存被隔离 K 线的原始 OHLCV、时间、原因、消息和隔离时间。
- 新增 `POST /api/market/candle-invalid-issues/quarantine`，只允许隔离后端重新判定为 `invalid_open_time` 的持久化 K 线。
- 隔离在同一事务内先归档原始行，再从 `market_candles` active 集合删除；其它异常类型会被跳过，不会被该接口删除。
- 研究页全历史异常详情中，`invalid_open_time` 不显示普通补同步 repair，而显示“隔离错位 K 线”；隔离成功后刷新全历史异常扫描、任务列表和当前 K 线窗口。
- 为新增前端 API route 同步 OpenAPI contract 和生成的 TypeScript DTO。
- 为避免阶段 0 文件规模回退，`internal/data/model.go` 中的 `Repository` 接口拆到 `internal/data/repository.go`。

范围外：

- 不自动隔离所有历史错位 K 线。
- 不为任务窗口异常弹窗单独增加隔离入口。
- 不改变交易所拉取、补同步调度、CandleProvider 聚合算法、回测/交易 runner 或实盘能力。

当前验证：

- `go test ./internal/web/api ./internal/store/postgres` 通过。
- `pnpm --dir web/frontend exec vitest run src/components/research/MarketCandleInvalidIssueTag.test.ts src/pages/ResearchPage.layout.test.ts src/services/api/data.test.ts` 通过。
- `scripts/check-file-size.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

剩余风险：

- 当前隔离入口需要用户在研究页手动触发，不能证明历史错位数据已被自动清理。
- 被隔离后形成的真实缺口仍需要用户或后续流程通过缺口 repair 排补同步任务。
- 该能力只关闭 `invalid_open_time` 的可审计隔离入口，不代表阶段 1 usable。

### 阶段 1 任务窗口 invalid_open_time 隔离入口补充

执行日期：2026-07-01

目标等级：scaffold。

范围内：

- 数据同步任务异常详情弹窗在当前返回页包含 `invalid_open_time` 时，显示“隔离错位 K 线”操作。
- 该操作复用 `POST /api/market/candle-invalid-issues/quarantine`，按当前任务的 exchange / symbol / interval 和当前返回页中的错位 `openTime` 归档并移除 active K 线。
- 隔离成功后关闭任务异常详情弹窗，通知研究页刷新任务列表和当前图表 K 线窗口，并从同一任务上下文打开既有缺口详情弹窗。
- 缺口详情弹窗只展示隔离后形成的真实缺口和既有“修复任务窗口缺口”入口，不自动创建补同步任务，也不把隔离伪装成修复完成。
- 普通异常补同步按钮仍只用于可修复异常，`invalid_open_time` 不会被普通 repair 伪装成已修复。

范围外：

- 不新增任务专用 quarantine API；本轮复用全历史 market quarantine API。
- 不自动隔离所有历史错位 K 线，也不自动排隔离后形成的缺口 repair；用户仍需要在缺口详情里显式触发 repair。
- 不改变 data sync worker、CandleProvider 聚合算法、交易所 adapter、回测/交易 runner 或实盘能力。

当前验证：

- `pnpm --dir web/frontend exec vitest run src/components/research/ResearchTaskInvalidIssueModal.test.ts src/pages/ResearchPage.layout.test.ts src/services/api/data.test.ts` 通过。
- `pnpm --dir web/frontend exec vitest run src/components/research/ResearchTaskInvalidIssueModal.test.ts src/pages/ResearchPage.layout.test.ts` 通过。
- `scripts/check-file-size.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

剩余风险：

- 任务窗口隔离仍基于当前弹窗返回页，不是全量自动清洗。
- 被隔离后形成的缺口会自动打开缺口详情作为接续观察入口，但仍需要用户显式排补同步任务。
- 该能力只补前端任务窗口操作闭环，不代表阶段 1 usable。

### 阶段 1 K 线图表 canvas 尺寸污染修复补充

执行日期：2026-07-01

目标等级：scaffold。

范围内：

- 共用 `TradingViewChart` 在检测到 lightweight-charts 内部 DOM / canvas inline 尺寸污染后，不再用可能已经污染的 canvas bitmap 反推出 CSS 宽高。
- `chartCanvasRepair` 改为只清理内部 `.tv-lightweight-charts`、table、tbody、tr、td 和 canvas 的异常 inline size 锁，再由 `TradingViewChart` 触发 `chart.resize(lastSize.width, lastSize.height)` 让图表库按固定 viewport 重算。
- 保留研究页、回测详情和交易详情共用的固定 K 线 viewport、价格轴最大宽度、完整价格显示和坐标轴文字墨迹高度门禁。
- 单元测试覆盖首次修复时 canvas bitmap 已被污染到 9000px 的场景，防止再次把右侧价格轴撑成大块空白或把坐标轴文字缩放到异常尺寸。

范围外：

- 不新增图表指标、绘图工具或交易操作能力。
- 不改变 CandleProvider、data sync worker、交易所 adapter、回测 / 交易 runner 或实盘能力。
- 不把当前前端体验升级为 usable；它仍需要继续按阶段 1 收敛研究工作台质量。

当前验证：

- `pnpm --dir web/frontend exec vitest run src/components/chart/chartCanvasRepair.test.ts src/components/chart/TradingViewChart.test.ts src/components/chart/TradingViewChart.readout.test.ts src/pages/ResearchPage.layout.test.ts src/pages/DetailPages.layout.test.ts` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/check-research-chart-layout.sh` 通过。
- `scripts/check-file-size.sh` 通过。
- `scripts/quality-gate.sh` 通过。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SAMPLES=4 SMOKE_INTERVAL_MS=120 SMOKE_SETTLE_MS=1000 SMOKE_TOTAL_TIMEOUT_MS=180000 node scripts/research-chart-height-smoke.mjs` 通过。
- `BASE_URL=http://127.0.0.1:8080 SMOKE_SETTLE_MS=800 SMOKE_TOTAL_TIMEOUT_MS=300000 node scripts/stage8-visual-smoke.mjs` 通过。
- `git diff --check` 通过。

剩余风险：

- 该修复只约束当前 lightweight-charts DOM / canvas 污染路径；仍需要更多真实浏览器与真实长时间交互样本。
- 当前图表工具条和研究交互仍处于阶段 1 打磨中，不代表研究页已达到 usable。

### 阶段 1 数据同步 worker 可观察和 DPR 图表重绘补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- `/api/data/tasks` 返回数据同步任务的 `lockedBy`、`lockedUntil`、`heartbeatAt`、`startedAt` 和 `finishedAt`，来源仍是既有 worker lease / task 状态字段。
- 研究页数据同步任务表新增 worker 列，展示当前持锁 worker、最近心跳、租约截止、启动或结束时间，并纳入中英 i18n。
- 前端 typed API wrapper 和生成 DTO 同步新增字段，避免前后端 contract 漂移。
- `TradingViewChart` 在窗口尺寸没有变化但 DPR 变化或内部 canvas 几何比例异常时，按固定 viewport 尺寸强制重绘。
- canvas 几何异常检测改为相对当前 device pixel ratio 判断，避免正常高 DPR 画布被误判为尺寸污染。

范围外：

- 不改变 data sync worker 的 claim、heartbeat、release、retry、repair 或状态机语义。
- 不新增 worker 调度、公平队列、分布式 token bucket 或长期压测能力。
- 不改变 CandleProvider、交易所 adapter、回测 / 交易 runner 或实盘能力。
- 不把研究页、数据同步 worker 或整体项目升级为 usable。

当前验证：

- `pnpm --dir web/frontend exec vitest run src/components/chart/TradingViewChart.geometry.test.ts src/components/tables/DataSyncTaskTable.test.ts src/services/api/data.test.ts` 通过。
- `go test ./internal/store/postgres -run 'TestDataSyncTaskScanColumnsPlaceMarketStatusBeforeHealth|TestIntegrationListDataSyncTasksReportsWorkerLease' -count=1` 通过。
- `scripts/generate-api-types.sh` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/check-file-size.sh` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

未执行：

- `node scripts/research-chart-height-smoke.mjs` 未执行；本机 Docker daemon 不可用，`http://127.0.0.1:8080/readyz` 返回 502，没有可登录 API 服务。
- `node scripts/stage8-visual-smoke.mjs` 未执行；原因同上。
- 内置浏览器人工验证未执行；浏览器控制运行时连接失败，且本地 API 目标不可用。

剩余风险：

- worker 列只增强研究页可观察性，不证明 worker 状态机在长期运行或多实例环境下已经 production-safe。
- DPR / canvas 重绘只覆盖当前已知的窗口缩放和 canvas 几何异常路径，仍缺真实浏览器长时间交互与跨屏拖拽视觉 smoke。
- 本轮没有补充 OKX 真实外网恢复证据、长期 soak、多实例共享额度或实盘能力。

### 阶段 1 data sync task 单任务读取收敛补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- `data.Repository` 新增 `GetDataSyncTask(ctx, id)`，PostgreSQL 实现复用既有单任务查询语义，并继续过滤软删除任务。
- `/api/data/tasks/{id}/repair-gap` 的 HTTP 层窗口对齐预校验不再通过 `ListDataSyncTasks` 扫描全量任务，而是直接读取当前源任务 interval。
- data sync task inactive / missing catalog 错误文案诊断不再依赖全量任务列表，改为读取当前任务的 `marketStatus`。
- API 回归测试覆盖 `ListDataSyncTasks` 故意失败时，单缺口 repair 仍能通过直接任务读取完成排队。
- PostgreSQL 集成测试覆盖 `GetDataSyncTask` 与列表读取一样返回 worker lease / heartbeat 字段。

范围外：

- 不改变 gap repair 判定、去重、创建补同步任务或 data sync worker 调度语义。
- 不新增 `GET /api/data/tasks/{id}` HTTP 路由。
- 不改变 CandleProvider、交易所 adapter、研究页 UI、回测 / 交易 runner 或实盘能力。
- 不把研究页、API server 或整体项目升级为 usable。

当前验证：

- `go test ./internal/web/api -run 'TestDataSyncTaskSingleGapRepairUsesDirectTaskLookup|TestDataSyncTaskRoutes|TestDataSyncTaskRepairRoutesRequireActiveMarketInstrument' -count=1` 通过。
- `go test ./internal/store/postgres -run 'TestIntegrationListDataSyncTasksReportsWorkerLease' -count=1` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/check-file-size.sh` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

剩余风险：

- 该补充只收敛 API handler 的读取路径，不扩大 repair 能力，也不证明补同步最终成功。
- data sync task 仍没有独立 HTTP 详情接口；当前单任务读取只服务后端内部命令语义。
- 项目整体仍为 `scaffold`，不能升级为 usable。

### 阶段 1 概览页 invalid 数据健康可观察性补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 概览页数据同步摘要新增 `invalid` 数据健康计数，继续复用既有 `/api/data/tasks` 返回的 `dataHealth` 字段。
- 概览页告警列表新增 `sync-invalid` 告警，和 `failed` 一样按错误级别展示，避免只把缺口 `gap` 暴露出来而漏掉异常 K 线状态。
- 中英文 i18n 同步补充数据同步摘要的 invalid 文案。
- 组合逻辑测试覆盖 failed / gap / invalid 同时出现时的摘要计数与告警顺序。

范围外：

- 不新增后端 API 字段、任务详情路由或 data sync worker 行为。
- 不改变 CandleProvider、交易所 adapter、repair、回测 / 交易 runner 或实盘能力。
- 不把概览页或整体项目升级为 usable；本轮只补缺口和异常数据源的入口可见性。

当前验证：

- `pnpm --dir web/frontend exec vitest run src/composables/useOverviewWorkspace.test.ts` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/check-file-size.sh` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

未执行：

- 浏览器 / 视觉 smoke 未执行；本轮是概览页组合逻辑与 i18n 小切片，没有启动本地 API 或浏览器环境。

剩余风险：

- 概览页只展示已有任务返回的 `dataHealth=invalid`，不证明异常 K 线已经被隔离或修复。
- 告警入口只能提示数据源异常，不能替代研究页任务窗口的诊断、隔离和缺口 repair 流程。
- 项目整体仍为 `scaffold`，不能升级为 usable。

### 阶段 1 概览页策略意图和订单近期活动补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 概览页在加载回测任务和交易任务后，复用既有 detail API 读取最近任务的 strategy intents 和 orders。
- 最近活动列表新增策略意图和订单条目，展示来源模块、任务名、市场、intent 类型 / policy 或订单 side / quantity / price，并跳转到对应回测或交易详情页。
- 为避免概览页对全部历史任务做无界 N+1 请求，本轮只读取最近 4 个回测任务和最近 4 个交易任务的 detail facts。
- 中英文 i18n 补充概览页“策略意图”和“订单”标题。
- 组合逻辑测试覆盖回测 / 交易 intents、回测 / 交易 orders 的加载、排序和路由目标。

范围外：

- 不新增后端全局 recent facts API，也不改变现有 detail API contract。
- 不改变 PostgreSQL schema、runner、worker、CandleProvider、订单执行或通知投递语义。
- 不把概览页近期活动解释为全局完整事实流；它仍是基于最近任务的轻量概览。
- 不把概览页或整体项目升级为 usable。

当前验证：

- `pnpm --dir web/frontend exec vitest run src/composables/useOverviewWorkspace.test.ts` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `scripts/check-file-size.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

未执行：

- 浏览器 / 视觉 smoke 未执行；本轮未改变页面结构，只让既有最近活动列表展示更多事实来源，没有启动本地 API 或浏览器环境。

剩余风险：

- 当前实现基于最近任务读取 detail facts，不是 PostgreSQL 级别的全局最近 intent / order 查询；旧任务若后来写入新事实，可能不会出现在概览页。
- detail facts 任一端点失败时概览页仍按现有 all-or-nothing 加载模型进入错误状态，后续可以补局部降级。
- 项目整体仍为 `scaffold`，不能升级为 usable。

### 阶段 1 概览页全局 recent facts API 补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 新增只读 `GET /api/overview/recent-facts`，返回全局最近 strategy intents 和 orders，而不是由前端按最近任务 fan-out 调 detail API。
- PostgreSQL 查询直接从 `strategy_intents`、`backtest_orders` 和 `orders` 读取事实，并 join `backtest_tasks` / `trading_tasks` 补齐任务名、交易所、交易对和周期。
- `limit` 默认有界，最大 50，超限返回 400；每类 facts 分别限制数量。
- OpenAPI contract 和生成的 TypeScript DTO 同步新增 `OverviewRecentFacts`、`OverviewStrategyIntentFact`、`OverviewOrderFact`。
- 概览页 `overviewFacts` 适配层改为调用新 endpoint，并继续映射市场标签、来源和详情页路由。
- 测试覆盖 HTTP route 全局排序 / 超限拒绝、真实 PostgreSQL 查询、前端 API wrapper 和概览组合逻辑。

范围外：

- 不新增写操作、迁移或新的事实表。
- 不改变 backtest / trading runner、worker、CandleProvider、订单执行、通知投递或实盘安全边界。
- 不新增趋势图、时间窗口筛选、生产级监控语义或局部降级加载。
- 不把概览页或整体项目升级为 usable。

当前验证：

- `go test ./internal/web/api -run 'TestOverviewRecentFactsRoute|TestAPIContractCoversCurrentFrontendRoutes' -count=1` 通过。
- `go test ./internal/store/postgres -run TestIntegrationListOverviewRecentFactsReturnsGlobalFacts -count=1` 通过。
- `go test ./internal/web/api -count=1` 通过。
- `go test ./internal/store/postgres -count=1` 通过。
- `scripts/generate-api-types.sh` 通过。
- `scripts/check-api-contract-drift.sh` 通过。
- `pnpm --dir web/frontend exec vitest run src/services/api/overview.test.ts src/composables/useOverviewWorkspace.test.ts` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `scripts/check-file-size.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

未执行：

- 浏览器 / 视觉 smoke 未执行；本轮变更是只读 API 与概览活动数据源替换，没有改变页面结构或图表布局。

剩余风险：

- recent facts 仍只是轻量概览聚合，不包含时间窗口筛选、趋势图、完整监控语义或局部降级加载。
- API 当前分别返回最近 intents 和最近 orders，不提供跨类型统一游标分页。
- 项目整体仍为 `scaffold`，不能升级为 usable。

### 阶段 1 概览页 recent facts 局部降级补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 概览页基础加载仍然并行读取系统健康、数据同步任务、回测任务、交易任务和通知。
- `/api/overview/recent-facts` 失败时不再把整个概览页置为全局错误；页面会保留已成功读取的基础数据并把 `hasLoaded` 置为 true。
- recent facts 失败时清空当前策略意图和订单事实，避免把上一轮 stale facts 混作最新活动。
- 概览页异常提醒新增 `recent-facts-degraded`，最近活动标题显示“局部降级 / Degraded”标签并用 title 保留错误信息。
- 中英文 i18n 补充局部降级和最近事实加载失败文案。
- 前端组合逻辑测试覆盖 recent facts 失败时基础概览照常加载、错误不进入全局 `error`、策略意图和订单活动不展示。

范围外：

- 不改变 `GET /api/overview/recent-facts` 后端 contract、PostgreSQL 查询或 OpenAPI schema。
- 不新增重试队列、自动恢复通知、趋势图、时间窗口筛选或统一游标分页。
- 不改变 backtest / trading runner、worker、CandleProvider、订单执行、通知投递或实盘安全边界。
- 不把概览页或整体项目升级为 usable。

当前验证：

- `pnpm --dir web/frontend exec vitest run src/composables/useOverviewWorkspace.test.ts` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `scripts/check-file-size.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

未执行：

- 浏览器 / 视觉 smoke 未执行；本轮只改变概览页非阻断错误处理和标题标签，没有启动本地 API 或浏览器环境。

剩余风险：

- 局部降级只覆盖概览页 recent facts；系统健康、任务列表和通知等基础端点失败时仍按既有全局错误模型处理。
- 降级后没有自动重试 recent facts，只能通过概览页刷新重新加载。
- recent facts 仍只是轻量概览聚合，不包含时间窗口筛选、趋势图、完整监控语义或统一游标分页。
- 项目整体仍为 `scaffold`，不能升级为 usable。

### 阶段 1 data sync task HTTP 详情接口补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 新增只读 `GET /api/data/tasks/{id}`，复用既有 `GetDataSyncTask` repository 读取路径返回单个数据同步任务。
- 单任务详情返回与 `/api/data/tasks` 列表一致的 `DataSyncTask` DTO，并复用 `sanitizeDataSyncTask` 脱敏 `lastError` 和 `exchangeBackoffLastError`。
- `/api/data/tasks/{id}` 资源路径保留 `DELETE`，错误方法现在返回 `Allow: GET, DELETE`。
- OpenAPI contract 新增 `getDataSyncTask` operation，并纳入 API contract route 覆盖测试。
- 前端 `dataApi` 新增 `getTask(id)`，复用列表同一套 `normalizeTask` 字段归一化和外部错误脱敏。
- 测试覆盖详情读取不依赖 `ListDataSyncTasks` 扫描全量任务、详情错误脱敏、前端 wrapper URL 编码和默认 `marketStatus` 归一化。

范围外：

- 不新增或修改数据同步任务详情页面；本轮只补 HTTP/API 能力和前端 typed wrapper。
- 不改变 data sync worker 的 claim、heartbeat、release、retry、repair 或状态机语义。
- 不改变 gap / invalid repair 判定、补同步任务创建、CandleProvider、交易所 adapter、回测 / 交易 runner 或实盘能力。
- 不把研究页、API server 或整体项目升级为 usable。

当前验证：

- `go test ./internal/web/api -run 'TestDataSyncTaskRoutes|TestDataSyncTaskDetailUsesDirectTaskLookup|TestDataSyncTaskSingleGapRepairUsesDirectTaskLookup|TestAPIMethodNotAllowedContracts|TestAPIContractCoversCurrentFrontendRoutes' -count=1` 通过。
- `pnpm --dir web/frontend exec vitest run src/services/api/data.test.ts` 通过。
- `scripts/check-api-contract-drift.sh` 通过。
- `scripts/generate-api-types.sh` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `scripts/check-file-size.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

未执行：

- 浏览器 / 视觉 smoke 未执行；本轮是只读 API、typed wrapper 和 contract 切片，没有改变页面结构或图表布局。

剩余风险：

- 新接口只提供单任务当前快照，不包含独立详情页、事件流、状态变更订阅或 repair 自动收敛证明。
- 该能力关闭“已有 repository 单任务读取但无 HTTP 详情路由”的 API 边界，不扩大数据同步 worker 或 repair 能力。
- 项目整体仍为 `scaffold`，不能升级为 usable。

### 阶段 1 研究页 repair task 单任务轮询补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 研究页 repair task 状态轮询不再每轮调用 `/api/data/tasks` 刷新全量任务列表。
- 新增 `loadRepairTaskSnapshots(ids)`，按 repair task id 调用 `GET /api/data/tasks/{id}` 读取单任务快照，并合并回当前任务列表。
- 单任务快照会更新任务表中对应 repair task 的状态和数据健康，也会同步更新当前选中图表任务和已打开缺口详情弹窗中的任务引用。
- repair task 单次读取失败或 404 时吞掉该次快照，保持有界重试，不覆盖研究页全局任务列表错误状态。
- `useResearchRepairTaskPolling` 改为把 watched repair task ids 传给 targeted loader，并继续用返回快照判断 created repair tasks 是否进入终态。
- 前端测试覆盖轮询传递 repair task ids、研究页接线改为 `loadRepairTaskSnapshots`、任务快照合并和缺口弹窗任务引用更新。

范围外：

- 不新增独立数据同步任务详情页、事件流、WebSocket 或实时订阅。
- 不改变 repair API、data sync worker 调度、补同步任务创建、CandleProvider、交易所 adapter、回测 / 交易 runner 或实盘能力。
- 不证明 repair task 一定成功，也不证明数据同步长期自动收敛。
- 不把研究页或整体项目升级为 usable。

当前验证：

- `pnpm --dir web/frontend exec vitest run src/composables/useResearchRepairTaskPolling.test.ts src/composables/useResearchWorkspace.taskGapRepair.test.ts src/pages/ResearchPage.layout.test.ts` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `scripts/check-file-size.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

未执行：

- 浏览器 / 视觉 smoke 未执行；本轮只改变研究页 repair polling 数据读取路径，没有改变页面布局或图表渲染结构。

剩余风险：

- 轮询仍是有界前端轮询，不是后台自动收敛证明或实时订阅。
- 单任务详情快照只更新 watched repair tasks；非 watched 任务仍依赖用户刷新或其它操作加载。
- 项目整体仍为 `scaffold`，不能升级为 usable。

### 阶段 1 研究页 repair source task 快照轮询补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- `useResearchRepairTaskPolling` 新增 snapshot-only task ids，把 watched repair task ids 与额外快照任务 ids 去重后交给 targeted loader。
- repair 轮询完成判定仍只看 created repair tasks；source task 快照即使仍是 running，也不会阻塞 `onSettled` 回调。
- 研究页所有 repair 入口统一把 repair result 的 `sourceTaskId` 和当前缺口详情任务 id 传入 snapshot-only ids，修复任务轮询时同步刷新源任务健康状态。
- 继续复用 `loadRepairTaskSnapshots` 的单任务详情读取和本地任务列表合并路径，不恢复全量 `/api/data/tasks` 轮询。
- 前端测试覆盖额外 source snapshot id 会随 repair task 一起请求、不会参与终态判定，以及研究页接线包含 source task snapshot ids。

范围外：

- 不新增独立数据同步任务详情页、事件流、WebSocket 或实时订阅。
- 不改变 repair API、data sync worker 调度、补同步任务创建、CandleProvider、交易所 adapter、回测 / 交易 runner 或实盘能力。
- 不证明 repair task 一定成功，也不证明数据同步长期自动收敛。
- 不把研究页或整体项目升级为 usable。

当前验证：

- `pnpm --dir web/frontend exec vitest run src/composables/useResearchRepairTaskPolling.test.ts src/composables/useResearchWorkspace.taskGapRepair.test.ts src/pages/ResearchPage.layout.test.ts` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `scripts/check-file-size.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

未执行：

- 浏览器 / 视觉 smoke 未执行；本轮只改变研究页 repair polling 数据读取集合，没有改变页面布局或图表渲染结构。

剩余风险：

- 轮询仍是有界前端轮询，不是后台自动收敛证明或实时订阅。
- source task 健康状态只在 repair polling 窗口内按快照刷新；非相关任务仍依赖用户刷新或其它操作加载。
- 项目整体仍为 `scaffold`，不能升级为 usable。

### 阶段 1 概览页 recent facts 时间窗口补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- `GET /api/overview/recent-facts` 新增可选 `since` query 参数，按 RFC3339 date-time 解析，非法时间返回 400。
- PostgreSQL recent facts 查询对 strategy intents 使用 `created_at >= since`，对 backtest / trading orders 使用统一的 `occurred_at >= since` 过滤，并继续保留每类 facts 的有界 `limit`。
- OpenAPI contract 声明 `since` date-time query 参数，并增加 contract 回归测试避免漏参。
- 前端 `overviewApi.recentFacts` 改为 options 参数，支持同时传入 `limit` 和 `since`。
- 概览页默认只请求最近 24 小时的 recent facts，基础系统健康、数据同步任务、回测任务、交易任务和通知列表加载模型不变。
- 测试覆盖 HTTP route 时间过滤 / 非法 since、PostgreSQL 查询过滤、前端 URL 编码和概览页默认 24 小时窗口调用。

范围外：

- 不新增跨类型统一游标分页、趋势图、监控查询语言、实时订阅或自动重试队列。
- 不改变 backtest / trading runner、strategy intent 写入、订单写入、通知投递、CandleProvider、data sync worker 或实盘安全边界。
- 不把概览页或整体项目升级为 usable。

当前验证：

- `go test ./internal/web/api -run 'TestOverviewRecentFactsRoute|TestAPIContractDeclaresOverviewRecentFactFilters|TestAPIContractCoversCurrentFrontendRoutes' -count=1` 通过。
- `go test ./internal/store/postgres -run TestIntegrationListOverviewRecentFactsReturnsGlobalFacts -count=1` 通过。
- `pnpm --dir web/frontend exec vitest run src/services/api/overview.test.ts src/composables/useOverviewWorkspace.test.ts` 通过。
- `scripts/generate-api-types.sh` 通过。
- `scripts/check-api-contract-drift.sh` 通过。
- `scripts/check-file-size.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

未执行：

- 浏览器 / 视觉 smoke 未执行；本轮只改变概览页 recent facts 的数据读取过滤和 API 参数，没有改变页面布局或图表渲染结构。

剩余风险：

- recent facts 仍分别返回 strategy intents 和 orders，不提供跨类型统一游标分页。
- 默认 24 小时窗口只约束概览页 recent facts 请求；基础任务列表、通知列表和系统健康仍按既有全量/当前快照模型加载。
- 时间窗口筛选不等同于完整监控语义、趋势图或生产级观测平台。
- 项目整体仍为 `scaffold`，不能升级为 usable。

### 阶段 1 概览页 recent activity 时间窗口控件补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 概览页最近活动标题区新增 24H / 7D / 30D 紧凑时间窗口控制。
- 默认窗口保持 24H；切换窗口时只重拉 `/api/overview/recent-facts`，不重新加载系统健康、数据同步任务、回测任务、交易任务或通知列表。
- recent facts 继续通过 `since` 参数按窗口过滤 strategy intents / orders。
- 最近活动列表中的本地数据同步、回测、交易和通知条目也按同一个窗口在前端过滤，避免窗口控制只影响部分 activity 来源。
- recent facts 局部降级语义保持不变；切换窗口失败时清空 strategy intents / orders 并显示既有降级提醒。
- 中英文 i18n 增加窗口控件 aria label 和 24H / 7D / 30D 文案。
- 测试覆盖默认 24H、切换 7D 只重拉 recent facts、本地 activity 条目按窗口过滤、刷新概览时重新计算 recent facts `since` 锚点，以及页面 raw-source 接线。

范围外：

- 不新增趋势图、监控查询语言、跨类型统一游标分页、实时订阅或自动重试队列。
- 不改变 `/api/overview/recent-facts` contract、PostgreSQL 查询、runner、通知投递、CandleProvider、data sync worker 或实盘安全边界。
- 不把概览页或整体项目升级为 usable。

当前验证：

- `pnpm --dir web/frontend exec vitest run src/composables/useOverviewWorkspace.test.ts src/pages/OverviewPage.layout.test.ts` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `scripts/check-file-size.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

未执行：

- 浏览器 / 视觉 smoke 未执行；本轮是概览页控件和数据过滤语义的小切片，尚未启动本地 API 或浏览器环境。

剩余风险：

- 时间窗口控件只过滤最近活动列表，不提供趋势图、聚合统计时间序列或生产级监控语义。
- recent facts 仍分别返回 strategy intents 和 orders，不提供跨类型统一游标分页。
- 项目整体仍为 `scaffold`，不能升级为 usable。

### 阶段 1 概览页汇总卡片操作入口补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 概览页五个汇总卡片从纯展示块改为可点击导航入口。
- 数据同步卡片跳转研究页，回测卡片跳转回测列表，交易卡片跳转交易列表，通知卡片跳转通知管理，后台服务卡片跳转运维健康。
- 卡片保留既有数量和摘要文案，新增箭头图标、hover/focus 状态和 aria label，避免只靠异常列表或最近活动进入业务页面。
- 中英文 i18n 增加汇总卡片打开入口的无障碍文案。
- 测试覆盖卡片目标路由、页面 RouterLink 接线、箭头图标和 i18n key。

范围外：

- 不新增趋势图、监控查询语言、生产级告警语义、过滤参数透传或深度指标。
- 不改变概览页加载 API、recent facts 查询、runner、CandleProvider、data sync worker 或实盘安全边界。
- 不把概览页或整体项目升级为 usable。

当前验证：

- `pnpm --dir web/frontend exec vitest run src/composables/useOverviewWorkspace.test.ts src/pages/OverviewPage.layout.test.ts` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `scripts/check-file-size.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- 前端 dev server HTTP smoke 通过：`http://127.0.0.1:5173/overview` 返回 200 和 Vite HTML。

未执行：

- 内置浏览器 / 视觉 smoke 未完成；浏览器控制工具返回 `sandboxCwd must be an absolute file URI` 元数据错误，未取得渲染截图或登录后概览页可视证据。

剩余风险：

- 汇总卡片只提供模块级入口，不提供带筛选条件的深链、趋势图或生产级监控语义。
- 概览页仍缺时间序列趋势、深度指标和更完整的观测闭环。
- 项目整体仍为 `scaffold`，不能升级为 usable。

### 阶段 1 概览页 7D 运行趋势补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 新增只读 `GET /api/overview/trends`，默认返回最近 7 个 UTC 日 bucket，`days` 最大 30。
- PostgreSQL 趋势查询使用 `generate_series` 保留空 bucket，并分别统计 strategy intents、backtest/trading orders、notifications 和失败信号。
- 失败信号只来自现有可审计字段：data sync / backtest / trading failed 任务的 `updated_at`，以及 failed notification 的 `created_at`。
- OpenAPI contract 和生成 TypeScript DTO 同步新增 `OverviewTrends` / `OverviewTrendBucket`。
- 前端概览页新增运行趋势面板，展示 7D 日柱、事实总数、失败信号和局部降级标签；趋势加载失败不阻断基础概览。
- 测试覆盖 HTTP route、days 上限、PostgreSQL bucket 统计、前端 API wrapper、趋势 composable、页面接线和 i18n key。

范围外：

- 不新增 Prometheus / 指标平台、实时订阅、趋势预测、跨类型统一游标分页、深度指标或生产级告警语义。
- 不改变任务 runner、通知投递、CandleProvider、data sync worker、订单执行或实盘安全边界。
- 不把概览页或整体项目升级为 usable。

当前验证：

- `go test ./internal/web/api -run 'TestOverview(RecentFacts|Trends)Route|TestAPIContract(DeclaresOverview|CoversCurrentFrontendRoutes)' -count=1` 通过。
- `go test ./internal/store/postgres -run 'TestIntegrationListOverview(RecentFacts|Trends)Returns' -count=1` 通过。
- `scripts/generate-api-types.sh` 通过。
- `scripts/check-api-contract-drift.sh` 通过。
- `pnpm --dir web/frontend exec vitest run src/services/api/overview.test.ts src/composables/useOverviewTrends.test.ts src/composables/useOverviewWorkspace.test.ts src/pages/OverviewPage.layout.test.ts` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `scripts/check-file-size.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- 前端 dev server HTTP smoke 通过：`http://127.0.0.1:5173/overview` 返回 200 和 Vite HTML。

未执行：

- 内置浏览器 / 视觉 smoke 未完成；浏览器控制工具继续返回 `sandboxCwd must be an absolute file URI` 元数据错误，未取得渲染截图或登录后概览页可视证据。

剩余风险：

- 当前趋势是有界日 bucket 概览，不是生产监控平台，也不提供实时订阅、SLO、告警规则或历史长期保留策略。
- 失败信号基于现有表字段粗粒度统计，不能替代任务详情、通知详情或审计日志中的精确诊断。
- 后续深度指标补充见下一节；生产级监控语义仍未关闭，项目整体仍为 `scaffold`，不能升级为 usable。

### 阶段 1 概览页深度指标补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 概览页新增深度指标面板，指标只从本页已经加载的系统健康、数据同步、回测、交易和通知数据推导，不新增 API。
- 数据质量指标展示健康同步任务占比，并把缺口、无效 K 线和失败同步任务纳入风险状态。
- 自动化链路指标展示运行同步任务、实时任务、过期锁和交易所冷却，链接到运维健康。
- 执行面指标汇总回测和交易任务运行/失败信号，并保留实盘任务数量提示。
- 通知投递指标展示已发送、失败和待处理通知，链接到通知管理。
- 中英文 i18n 增加深度指标标题、四类指标文案和状态标签。
- 测试覆盖概览数据源到深度指标的推导结果，以及页面组件接线和 i18n key。

范围外：

- 不新增生产级监控平台、Prometheus 指标、告警规则、SLO、实时订阅、深链筛选参数或新的后端聚合 API。
- 不改变任务 runner、通知投递、CandleProvider、data sync worker、订单执行或实盘安全边界。
- 不把概览页或整体项目升级为 usable。

当前验证：

- `pnpm --dir web/frontend exec vitest run src/composables/useOverviewWorkspace.test.ts src/pages/OverviewPage.layout.test.ts` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `scripts/check-file-size.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- 前端 dev server HTTP smoke 通过：`http://127.0.0.1:5173/overview` 返回 200 和 Vite HTML。

未执行：

- 内置浏览器 / 视觉 smoke 未完成；浏览器控制工具继续返回 `sandboxCwd must be an absolute file URI` 元数据错误，未取得渲染截图或登录后概览页可视证据。

剩余风险：

- 深度指标来自现有列表和健康快照的前端派生，不是生产级时序监控或可配置告警。
- 通知投递指标入口的状态上下文在后续小节补充；其它指标入口仍只跳到模块页面，不携带过滤参数或诊断上下文。
- 项目整体仍为 `scaffold`，不能升级为 usable。

### 阶段 1 概览页监控上下文补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 概览页新增监控上下文面板，明确展示当前概览数据只是一次快照，不是实时监控平台。
- 快照项展示系统健康检查时间、健康状态和服务数量，继续复用既有 `/api/system/health` 数据。
- 数据源降级项汇总 recent facts 和 7D trends 两类概览补充数据源的降级数量。
- 趋势覆盖项展示当前 7D trend bucket 数量，趋势为空或加载失败时标为关注状态。
- 告警负载项展示当前异常提醒数量，帮助区分“无告警”和“有告警但非生产告警规则”。
- 中英文 i18n 增加监控上下文、四类上下文项和状态标签文案。
- 测试覆盖组件推导输出、趋势空覆盖关注状态、页面接线和 i18n key。

范围外：

- 不新增 Prometheus / 指标平台、SLO、告警规则、实时订阅、长期保留策略或新的后端聚合 API。
- 不改变任务 runner、通知投递、CandleProvider、data sync worker、订单执行或实盘安全边界。
- 不把概览页或整体项目升级为 usable。

当前验证：

- `pnpm --dir web/frontend exec vitest run src/components/overview/OverviewMonitoringContextPanel.test.ts src/pages/OverviewPage.layout.test.ts` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `scripts/check-file-size.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- 前端 dev server HTTP smoke 通过：`http://127.0.0.1:5173/overview` 返回 200 和 Vite HTML。

未执行：

- 内置浏览器 / 视觉 smoke 未完成；浏览器控制工具继续返回 `sandboxCwd must be an absolute file URI` 元数据错误，未取得渲染截图或登录后概览页可视证据。

剩余风险：

- 监控上下文只说明快照边界和局部降级，不提供生产级指标采集、SLO、告警规则或实时订阅。
- 告警负载来自概览页现有异常列表，不等同于可配置告警系统。
- 项目整体仍为 `scaffold`，不能升级为 usable。

### 阶段 1 概览页通知诊断上下文补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 通知管理页新增通知状态筛选，支持全部、失败、待处理和已发送四类视图。
- 通知状态筛选读取并同步 URL `status` query，允许概览页入口带诊断上下文跳转。
- 待处理筛选包含 `pending` 和 `retry_scheduled`，已发送筛选包含 `sent` 和 `delivered`。
- 概览页通知汇总卡片、通知失败告警、最近通知活动和通知投递深度指标带 `status` query 跳转通知管理。
- 通知管理页中英文 i18n 增加筛选项、筛选 aria label 和筛选空状态文案。
- 测试覆盖状态 query 归一化、状态分组匹配、通知管理页接线、概览页通知入口路由上下文和 i18n key。

范围外：

- 不新增后端通知过滤 API；本轮只在前端对已加载通知列表筛选。
- 不改变通知 outbox、provider、retry、runner、投递幂等或真实第三方 provider 生产启用边界。
- 不为数据质量、自动化链路等其它概览指标新增深链筛选；执行面上下文由后续独立切片补充。
- 不把通知管理、概览页或整体项目升级为 usable。

当前验证：

- `pnpm --dir web/frontend exec vitest run src/pages/systemNotificationsFilters.test.ts src/pages/SystemNotificationsPage.layout.test.ts src/composables/useOverviewWorkspace.test.ts src/pages/OverviewPage.layout.test.ts` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `scripts/check-file-size.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- 前端 dev server HTTP smoke 通过：`http://127.0.0.1:5173/overview` 和 `http://127.0.0.1:5173/system/notifications?status=failed` 均返回 200 和 Vite HTML。

未执行：

- 内置浏览器 / 视觉 smoke 未完成；浏览器控制工具继续返回 `sandboxCwd must be an absolute file URI` 元数据错误，未取得渲染截图或登录后概览页/通知管理页可视证据。

剩余风险：

- 通知筛选是前端列表筛选，不是后端查询、分页或生产级告警规则。
- 数据质量和自动化链路上下文由后续独立切片补充；其它概览指标入口仍没有模块内筛选上下文或诊断深链。
- 项目整体仍为 `scaffold`，不能升级为 usable。

### 阶段 1 概览页执行面诊断上下文补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 回测列表页和交易列表页新增任务状态筛选，读取并同步 URL `status` query。
- 两个列表页支持全部、运行中、失败和已成功四类显式筛选；共享 helper 仍归一化 `pending` / `cancelled` 等任务状态，避免旧 URL 或手写 URL 泄漏成异常状态。
- 概览页回测/交易汇总卡片、失败告警和执行面深度指标带 `status` query 跳转对应列表，失败优先进入失败筛选，运行中任务可进入运行中筛选。
- 回测/交易列表页中英文 i18n 增加筛选项、筛选 aria label 和筛选空状态文案。
- 测试覆盖状态 query 归一化、状态匹配、回测/交易页接线、概览页执行面入口路由上下文和 i18n key。

范围外：

- 不新增后端回测或交易任务过滤 API；本轮只在前端对已加载任务列表筛选。
- 不改变回测 runner、交易 runner、worker lease、撮合模型、paper/live executor、订单幂等或实盘安全边界。
- 不为数据质量、自动化链路等其它概览指标新增深链筛选。
- 不把回测、交易、概览页或整体项目升级为 usable。

当前验证：

- `pnpm --dir web/frontend exec vitest run src/pages/taskStatusFilters.test.ts src/pages/BacktestsPage.layout.test.ts src/pages/TradingPage.layout.test.ts src/composables/useOverviewWorkspace.test.ts` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `scripts/check-file-size.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- 前端 dev server HTTP smoke 通过：`http://127.0.0.1:5173/overview`、`http://127.0.0.1:5173/backtests?status=failed` 和 `http://127.0.0.1:5173/trading?status=failed` 均返回 200 和 Vite HTML。

未执行：

- 内置浏览器 / 视觉 smoke 未完成；浏览器控制工具继续返回 `sandboxCwd must be an absolute file URI` 元数据错误，未取得渲染截图或登录后概览页/回测页/交易页可视证据。

剩余风险：

- 回测/交易筛选是前端列表筛选，不是后端查询、分页、任务队列诊断或生产级执行监控。
- 执行面深链只覆盖任务状态，不提供失败原因聚合、runner lease 诊断、订单执行链路或实盘安全审计。
- 数据质量和自动化链路上下文由后续独立切片补充；其它概览指标入口仍没有模块内筛选上下文或诊断深链。
- 项目整体仍为 `scaffold`，不能升级为 usable。

### 阶段 1 概览页数据质量诊断上下文补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 研究页同步任务列表读取 URL `dataHealth` query，并在前端对已加载任务列表做数据健康筛选。
- 支持全部、正常、同步中、有缺口、失败、已暂停、重试中、数据不足和数据异常等后端返回的 data health 值归一化。
- 概览页数据同步汇总卡片、失败 / 缺口 / 异常告警和数据质量深度指标带 `dataHealth` query 跳转研究页。
- 研究页在筛选后无匹配任务时显示筛选空状态文案；表格仍保留原任务操作语义，repair 相关组件继续接收完整任务列表用于收敛状态判断。
- 测试覆盖 `dataHealth` query 归一化、任务匹配、研究页接线、研究 workspace 筛选和 query 保留、概览页数据质量入口路由上下文和 i18n key。

范围外：

- 不新增后端 data sync 任务过滤 API；本轮只在前端对已加载任务列表筛选。
- 不改变 CandleProvider、data sync worker、repair API、补同步调度、图表缺口/异常检测或历史数据清洗语义。
- 不为自动化链路等其它概览指标新增深链筛选。
- 不把研究页、概览页或整体项目升级为 usable。

当前验证：

- `pnpm --dir web/frontend exec vitest run src/composables/researchTaskFilters.test.ts src/composables/useResearchWorkspace.dataHealthFilter.test.ts src/pages/ResearchPage.layout.test.ts src/composables/useOverviewWorkspace.test.ts` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `scripts/check-file-size.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- 前端 dev server HTTP smoke 通过：`http://127.0.0.1:5173/overview`、`http://127.0.0.1:5173/research?dataHealth=gap` 和 `http://127.0.0.1:5173/research?dataHealth=invalid` 均返回 200 和 Vite HTML。

未执行：

- 内置浏览器 / 视觉 smoke 未完成；浏览器控制工具继续返回 `sandboxCwd must be an absolute file URI` 元数据错误，未取得渲染截图或登录后概览页/研究页可视证据。

剩余风险：

- 数据健康筛选是前端列表筛选，不是后端查询、分页、全局数据质量索引或生产级数据质量监控。
- 数据质量深链只覆盖任务级 `dataHealth`，不直接定位到具体 K 线缺口、异常详情页筛选或 repair 收敛结果。
- 自动化链路上下文由后续独立切片补充；其它概览指标入口仍没有模块内筛选上下文或诊断深链。
- 项目整体仍为 `scaffold`，不能升级为 usable。

### 阶段 1 概览页自动化链路诊断上下文补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 运维健康页服务列表新增 `focus` query 筛选，支持全部、异常、过期锁和交易所冷却四类视图。
- 运维健康页筛选会同步 URL query，便于概览页带诊断上下文跳转。
- 概览页后台服务汇总卡片、系统健康告警和自动化链路深度指标带 `focus` query 跳转运维健康页；优先聚焦过期锁，其次交易所冷却，再其次异常服务。
- 运维健康页中英文 i18n 增加筛选项、筛选 aria label 和筛选空状态文案。
- 测试覆盖 `focus` query 归一化、服务匹配、运维健康页筛选、概览页自动化入口路由上下文和 i18n key。

范围外：

- 不新增后端 system health 过滤 API；本轮只在前端对已加载 service 列表筛选。
- 不改变 worker lease、exchange backoff、fetch lock、data sync worker 调度、健康状态计算或生产监控语义。
- 不新增自动化链路 SLO、告警规则、实时订阅、trace 或结构化日志。
- 不把运维健康、概览页或整体项目升级为 usable。

当前验证：

- `pnpm --dir web/frontend exec vitest run src/pages/systemHealthFilters.test.ts src/pages/SystemHealthPage.test.ts src/pages/SystemHealthPage.layout.test.ts src/composables/useOverviewWorkspace.test.ts` 通过。
- `pnpm --dir web/frontend run typecheck` 通过。
- `scripts/check-file-size.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。
- 前端 dev server HTTP smoke 通过：`http://127.0.0.1:5173/overview`、`http://127.0.0.1:5173/system/health?focus=stale` 和 `http://127.0.0.1:5173/system/health?focus=backoff` 均返回 200 和 Vite HTML。

未执行：

- 内置浏览器 / 视觉 smoke 未完成；浏览器控制工具继续返回 `sandboxCwd must be an absolute file URI` 元数据错误，未取得渲染截图或登录后概览页/运维健康页可视证据。

剩余风险：

- 运维健康筛选是前端列表筛选，不是后端查询、分页、worker 事件流或生产级监控。
- 自动化链路深链只覆盖服务级 focus，不直接定位具体 lease、具体冷却交易所、fetch lock 竞争历史或任务级排队原因。
- 项目整体仍为 `scaffold`，不能升级为 usable。

### 阶段 8 Docker Compose 生产运行手册补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 新增 `docs/production-runbook.md`，作为当前 Docker Compose 运行形态的生产操作入口。
- 手册覆盖服务拓扑、共享环境启用前决策、首次启动、健康检查、备份命令、恢复演练流程、升级/回滚边界、停止/重启、事故检查清单和验证矩阵。
- README 增加生产运行手册入口；`docs/go-command-runbook.md` 链接到生产运行手册，避免子命令手册继续承担 Compose 级运维细节。
- 质量审计表中 Go 子命令和 Docker Compose 的缺口从“缺生产运行手册”校准为“已补手册入口，但缺自动化备份、已记录恢复演练、资源容量策略、结构化日志/trace、子命令级健康探针和外部依赖韧性证据”。

范围外：

- 不修改 Docker Compose、Dockerfile、Go 子命令、migration、worker、API 或前端运行时代码。
- 不新增自动备份任务、restore automation、secret manager / KMS 集成、资源限制配置、外部监控或告警规则。
- 不执行真实备份恢复演练，不声明 Docker Compose、Go 子命令或项目整体 production-safe。

当前验证：

- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过，55 个 test file / 240 个测试。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `scripts/check-file-size.sh` 通过。
- `git diff --check` 通过。

未执行：

- 未执行 `scripts/stage8-smoke.sh` 或 `scripts/stage8-sigterm-smoke.sh`；本轮是文档切片，没有改 Compose 运行时、容器 SIGTERM 或前端页面。
- 未执行 PostgreSQL 备份 / 恢复 drill；本轮只记录操作流程，恢复证据仍需后续独立切片在隔离数据库上完成。
- 未执行浏览器 / 视觉 smoke；本轮没有前端渲染变更。

剩余风险：

- 生产运行手册只提供可执行操作边界，不等于自动化备份、恢复演练、容量规划、外部监控或告警已完成。
- `hi api` 之外仍缺子命令级健康探针，后台 worker 长期运行仍依赖现有 system health 快照和日志排查。
- 项目整体仍为 `scaffold`，不能升级为 usable 或 production-safe。

### 阶段 8 PostgreSQL 备份恢复演练入口补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 新增 `scripts/stage8-backup-restore-drill.sh`，作为可重复执行的本地 PostgreSQL backup / restore drill 入口。
- 脚本会启动 Compose PostgreSQL、对源库运行 `hi migrate`、生成压缩 `pg_dump -Fc`、恢复到独立 drill database、在 drill database 上复跑 `hi migrate` 验证幂等，并检查 restored public table 列表和 `schema_migrations` 记录。
- drill database 默认退出时删除；可用 `STAGE8_BACKUP_RESTORE_KEEP_DB=1` 保留本地演练库用于人工检查。
- `scripts/full-quality-gate.sh` 增加显式 `FULL_QUALITY_STAGE8_BACKUP_RESTORE=1` 开关，不放入默认门禁，避免普通 PR gate 隐式依赖 Docker daemon 和本地数据库状态。
- README 和 `docs/production-runbook.md` 增加 backup / restore drill 入口说明。

范围外：

- 不新增自动备份调度、外部备份存储、恢复审批流程、容量/保留策略、KMS / secret manager 或生产告警。
- 不修改 Compose service 定义、Go migration 逻辑、PostgreSQL schema、worker、API 或前端。
- 不把脚本存在等同于恢复演练已经通过；目标环境仍必须实际运行并记录证据。

当前验证：

- `bash -n scripts/stage8-backup-restore-drill.sh scripts/full-quality-gate.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过，55 个 test file / 240 个测试。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `scripts/check-file-size.sh` 通过。
- `git diff --check` 通过。

未执行：

- `scripts/stage8-backup-restore-drill.sh` 未通过运行；脚本停在 `start postgres`，Docker 返回 `unable to get image 'postgres:16-alpine': Cannot connect to the Docker daemon at unix:///Users/xiaobai/.docker/run/docker.sock. Is the docker daemon running?`。同一环境下 `docker compose ps` / `docker version` 也无法连接 daemon。
- 未执行真实 `pg_dump` / `pg_restore` / drill database 迁移幂等检查；恢复演练通过证据仍需 Docker 可用后补跑。

剩余风险：

- 脚本只覆盖本地 Compose PostgreSQL drill，不是自动定时备份、异地恢复、生产容量规划或长期数据保留策略。
- 当前没有目标环境恢复演练成功记录，因此 Docker Compose 和 Go 子命令仍不能升级为 production-safe。
- 项目整体仍为 `scaffold`，不能升级为 usable 或 production-safe。

### 阶段 8 worker 子命令进程级健康探针补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- `hi sync`、`hi backtest`、`hi trading`、`hi notify` 支持可选 `*_HEALTH_ADDR`，分别为 `SYNC_HEALTH_ADDR`、`BACKTEST_HEALTH_ADDR`、`TRADING_HEALTH_ADDR`、`NOTIFY_HEALTH_ADDR`。
- 当对应 env 设置为合法 TCP `host:port` 且命令不是 `--once` 时，worker 在成功打开 PostgreSQL 后、进入长运行 runner 前启动进程级 HTTP 探针。
- 探针提供 `GET /livez`、`GET /readyz`、`GET /healthz`，返回 `status=ok`、`command` 和 `workerId`；`HEAD` 返回 200，无 body；写方法返回 405 和 `Allow`。
- health addr 在打开 PostgreSQL 前校验，非法配置会由 command config smoke 捕获，且错误输出继续不泄露测试 DSN / password / secret marker。
- Docker Compose 透传可选 health addr env；`.env.example`、Go 子命令手册和生产运行手册同步说明默认关闭和探针边界。

范围外：

- 不新增 Docker Compose 默认 healthcheck 或默认端口映射，避免空配置下把 worker 容器伪装成完整业务健康。
- 不新增 Prometheus 指标、结构化 trace、外部监控、SLO、告警规则或 worker 任务队列级 readiness。
- 不改变 worker lease、heartbeat、claim、retry/backoff、SIGTERM 收尾、system health API 或运维健康页逻辑。
- 不把进程级探针等同于生产级监控或业务可用性证明。

当前验证：

- `go test ./cmd/hi -run 'Test(LoadWorkerHealthProbeAddr|WorkerHealthProbe|StartWorkerHealthProbe|LoadSyncCommandConfig|SafeConfigSummary)' -count=1 -v` 通过。
- `go test ./cmd/hi -count=1` 通过。
- `scripts/stage8-command-config-smoke.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过，55 个 test file / 240 个测试。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `scripts/check-file-size.sh` 通过。
- `git diff --check` 通过。

未执行：

- 未执行 Docker Compose worker health probe smoke；当前本机 Docker daemon 仍不可用。该能力已通过本地 Go HTTP listener 单元测试和 env 配置 smoke 覆盖，Compose 运行态证据仍需 Docker 可用后补跑。
- 未执行浏览器 / 视觉 smoke；本轮没有前端渲染变更。

剩余风险：

- 这些探针只证明进程已打开 PostgreSQL 并启动本地 HTTP 探针；任务锁、队列积压、stale worker、exchange backoff、fetch-lock skip 和 catalog 状态仍要通过 `hi api` system health 观察。
- 子命令仍缺结构化日志 / trace、业务 readiness、资源容量策略和完整生产监控接入。
- 项目整体仍为 `scaffold`，不能升级为 usable 或 production-safe。

### 阶段 8 Go 子命令基础结构化日志配置补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 所有 `hi` 子命令在执行具体命令前读取 `LOG_LEVEL` 和 `LOG_FORMAT`。
- `LOG_LEVEL` 支持 `debug/info/warn/error`，默认 `info`；`LOG_FORMAT` 支持 `text/json`，默认 `text`。
- JSON 格式使用 Go `slog.JSONHandler`，text 格式使用 `slog.TextHandler`；现有启动摘要、worker runner 和 API 日志继续走统一默认 `slog`。
- 非法 `LOG_LEVEL` / `LOG_FORMAT` 会在打开 PostgreSQL 前失败，错误只命名 env key 和允许值，不回显非法值，避免误填 secret 后泄露到启动日志。
- `.env.example`、Docker Compose、执行计划配置清单、Go 子命令手册和生产运行手册同步日志配置入口；command config smoke 增加非法 `LOG_LEVEL` 且不泄露 secret marker 的覆盖。

范围外：

- 不新增 trace ID / correlation ID 传播，不改 HTTP request logging，不引入外部日志 sink、OpenTelemetry、Prometheus、SLO 或告警规则。
- 不改变业务错误码、审计日志、worker lease、system health API 或前端可观察性。
- 不把可选 JSON 日志配置等同于生产级集中日志或可追踪链路。

当前验证：

- `go test ./cmd/hi -run 'TestNewLoggerFromEnv|TestLoadSyncCommandConfig|TestSafeConfigSummary' -count=1 -v` 通过。
- `go test ./cmd/hi -count=1` 通过。
- `scripts/stage8-command-config-smoke.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过，55 个 test file / 240 个测试。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `scripts/check-file-size.sh` 通过。
- `git diff --check` 通过。

未执行：

- 未执行 Docker Compose 日志格式 smoke；本轮配置解析、text/json handler 输出和 secret 不回显已由 Go 单元测试与 command config smoke 覆盖，Compose 运行态证据仍需 Docker 可用后补跑。
- 未执行浏览器 / 视觉 smoke；本轮没有前端渲染变更。

剩余风险：

- 仍缺跨 API / worker / task 的 trace ID 或 correlation ID 传播。
- 仍缺外部日志 sink、日志保留策略、告警规则和生产排障 runbook。
- 项目整体仍为 `scaffold`，不能升级为 usable 或 production-safe。

### 阶段 8 Go 子命令运行级 correlation ID 补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 所有 `hi` 子命令的默认 `slog` logger 会带 `correlation_id` 字段。
- `LOG_CORRELATION_ID` 允许外部部署注入运行级关联 ID；未设置时命令进程启动时生成随机 ID。
- `LOG_CORRELATION_ID` 只接受 8 到 128 位的 ASCII 字母、数字、`.`、`_`、`:`、`-`；非法值会在打开 PostgreSQL 前失败，错误只命名 env key 和允许值，不回显非法值。
- JSON / text 两种日志格式都输出同一个 `correlation_id` 字段；command config smoke 增加非法 `LOG_FORMAT` 和非法 correlation ID 且不泄露 secret marker 的覆盖。
- `.env.example`、Docker Compose、执行计划配置清单、Go 子命令手册和生产运行手册同步 correlation ID 配置入口。

范围外：

- 不新增 HTTP request ID 中间件、OpenTelemetry、跨 worker task 的 trace 传播或外部日志 sink；运行级 `LOG_TRACEPARENT` 见后续补充。
- 不改变审计日志、业务错误码、worker lease、health probe、system health API 或前端可观察性。
- 不把运行级 correlation ID 等同于分布式 trace 或生产级链路追踪。

当前验证：

- `go test ./cmd/hi -run 'Test(NewLoggerFromEnv|LogCorrelationIDFromEnv|LoadSyncCommandConfig|SafeConfigSummary)' -count=1 -v` 通过。
- `go test ./cmd/hi -count=1` 通过。
- `scripts/stage8-command-config-smoke.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过，55 个 test file / 240 个测试。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `scripts/check-file-size.sh` 通过。
- `git diff --check` 通过。

未执行：

- 未执行 Docker Compose correlation ID smoke；当前本机 Docker daemon 仍不可用。该能力已由 Go 单元测试和 command config smoke 覆盖，Compose 运行态证据仍需 Docker 可用后补跑。
- 未执行浏览器 / 视觉 smoke；本轮没有前端渲染变更。

剩余风险：

- 运行级 traceparent 见后续补充；仍缺 worker task / 外部系统之间的 trace ID 传播。
- 仍缺外部日志 sink、日志保留策略、告警规则和生产排障 runbook。
- 项目整体仍为 `scaffold`，不能升级为 usable 或 production-safe。

### 阶段 8 Go 子命令运行级 traceparent 补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 新增 `LOG_TRACEPARENT`，允许部署注入运行级 W3C v00 traceparent；空值时每个 command process 生成合法非零 traceparent。
- 默认 `slog` logger 在所有子命令日志中输出 `run_traceparent` 和 `run_trace_id`，并继续输出 `correlation_id`。
- 非法 `LOG_TRACEPARENT` 在打开 PostgreSQL 前失败，错误只命名 env key，不回显非法值。
- `.env.example`、Docker Compose、执行计划配置清单、Go 子命令手册、生产运行手册和 command config smoke 同步配置入口。
- 单元测试覆盖 injected uppercase normalize、默认生成、非法值不回显以及默认日志字段。

范围外：

- 不引入 OpenTelemetry SDK、span lifecycle、采样策略、外部 tracing backend、日志 sink 或 retention policy。
- 不自动把一个子命令生成的 traceparent 分发给其他 independently started subcommands；需要部署显式设置同一个 `LOG_TRACEPARENT`。
- 不改变 API request traceparent、task traceparent、notification provider 或 data sync exchange request metadata propagation。

当前验证：

- `docker compose --env-file .env.example config --quiet` 通过。
- `bash -n scripts/stage8-command-config-smoke.sh` 通过。
- `go test ./cmd/hi -count=1` 通过。
- `scripts/stage8-command-config-smoke.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `scripts/check-file-size.sh` 通过。
- `git diff --check` 通过。

未执行：

- 未执行 Docker Compose 运行态 smoke；当前本机 Docker daemon 仍不可用。配置渲染、Go 单元测试和 command config smoke 已覆盖本轮配置路径。
- 未执行浏览器 / 视觉 smoke；本轮没有前端渲染变更。

剩余风险：

- Run-level traceparent 只是进程级日志上下文，不是 OpenTelemetry trace，也不会自动传播到 independently started subcommands。
- data sync market request 和 notification provider 之外的更广外部系统 trace propagation 仍未闭环。
- 项目整体仍为 `scaffold`，不能升级为 usable 或 production-safe。

### 阶段 8 API HTTP request ID 和 access log 补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- API server 在 `ServeHTTP` 最外层为每个 HTTP 请求设置 `X-Request-ID` 响应头。
- 如果请求带合法 `X-Request-ID`，服务端复用该值；缺失或非法时生成随机 request ID，非法输入不会回显到响应头。
- request ID 写入 `request.Context()`，后续 handler / repository 调用可通过 `RequestIDFromContext` 读取。
- `/readyz`、认证失败、认证成功 API 路由和静态前端响应共享同一 request ID 注入入口。
- OpenAPI contract 的所有响应声明 `X-Request-ID` header。
- API access log 输出 `request_id`、method、path、status、bytes 和 duration；path 不包含 query string，避免把 token 类参数写入日志。

范围外：

- 不把 request ID 自动写入所有业务日志，不引入 W3C traceparent、OpenTelemetry 或外部日志 sink。
- 不把 API request ID 传播到 worker task、交易所请求、通知 provider 或审计事件。
- 不改变认证、CSRF、错误响应 body、请求/响应 JSON schema 或前端 API client 行为。

当前验证：

- `go test ./internal/web/api -run 'Test(Server(AssignsRequestID|ReusesValidRequestID|ReplacesInvalidRequestIDWithoutEchoingValue|AccessLogIncludesRequestIDWithoutQuery)|APIContractDeclaresRequestIDResponseHeader)' -count=1 -v` 通过。
- `go test ./internal/web/api -count=1` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过，55 个 test file / 240 个测试。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `scripts/check-file-size.sh` 通过。
- `git diff --check` 通过。

未执行：

- 未执行 Docker Compose / 真实 8080 API request ID smoke；当前本机 Docker daemon 仍不可用。该能力已由 API handler 单元测试覆盖，Compose 运行态证据仍需 Docker 可用后补跑。
- 未执行浏览器 / 视觉 smoke；本轮没有前端渲染变更。

剩余风险：

- 仍缺 worker task / 外部系统之间的 trace ID 传播。
- 仍缺外部日志 sink、日志保留策略、告警规则和生产排障 runbook。
- 项目整体仍为 `scaffold`，不能升级为 usable 或 production-safe。

### 阶段 8 Go 子命令 PostgreSQL pool 资源边界补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- `postgres.OpenWithOptions` 使用 `pgxpool.ParseConfig` / `NewWithConfig` 设置 PostgreSQL pool 上限。
- 所有 `hi` 子命令在打开 PostgreSQL 前解析 `DB_MAX_CONNS`、`DB_MIN_CONNS`、`DB_MAX_CONN_LIFETIME`、`DB_MAX_CONN_IDLE_TIME`。
- 默认值为 `DB_MAX_CONNS=10`、`DB_MIN_CONNS=0`、`DB_MAX_CONN_LIFETIME=1h`、`DB_MAX_CONN_IDLE_TIME=30m`；`DB_MAX_CONNS` 限制在 1..1000，`DB_MIN_CONNS` 限制在 0..1000，且 `DB_MIN_CONNS <= DB_MAX_CONNS`。
- 启动摘要输出生效的 DB pool 配置，但不会输出 `DATABASE_URL`。
- `.env.example`、Docker Compose、执行计划配置清单、Go 子命令手册和生产运行手册同步 DB pool 配置入口；command config smoke 增加非法 DB pool 配置覆盖。

范围外：

- 不新增 PostgreSQL server 参数调优、连接池外置代理、CPU / memory / disk 配额、容量压测、保留策略或自动伸缩策略。
- 不改变 SQL 查询、事务语义、worker lease、health probe、OpenAPI contract 或前端行为。
- 不把连接池上限等同于完整生产容量规划。

当前验证：

- `go test ./cmd/hi -run 'Test(LoadAPICommandConfigLoadsDatabasePoolOptions|LoadAPICommandConfigRejectsInvalidDatabasePool|LoadSyncCommandConfig|SafeConfigSummary)' -count=1 -v` 通过。
- `go test ./internal/store/postgres -run 'TestNewPoolConfig' -count=1 -v` 通过。
- `scripts/stage8-command-config-smoke.sh` 通过。
- `go test ./cmd/hi -count=1` 通过。
- `go test ./internal/store/postgres -count=1` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过，55 个 test file / 240 个测试。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `scripts/check-file-size.sh` 通过。
- `git diff --check` 通过。

未执行：

- 未执行 Docker Compose DB pool smoke；当前本机 Docker daemon 仍不可用。该能力由 Go 单元测试和 command config smoke 覆盖，Compose 运行态证据仍需 Docker 可用后补跑。
- 未执行浏览器 / 视觉 smoke；本轮没有前端渲染变更。

剩余风险：

- 仍缺 CPU / memory / disk 容量测试、保留策略和目标环境 sizing 证据。
- 仍缺外部日志 sink、日志保留策略、告警规则和生产排障 runbook。
- 项目整体仍为 `scaffold`，不能升级为 usable 或 production-safe。

### 阶段 8 API 创建任务 request ID 关联补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- `POST /api/data/tasks`、`POST /api/backtests` 和 `POST /api/trading/tasks` 从当前请求 context 读取 `RequestIDFromContext`，并在创建任务时保存为 nullable `request_id`。
- `DataSyncTask`、`BacktestTask` 和 `TradingTask` 响应暴露只读 `requestId`；`CreateDataSyncTask`、`CreateBacktestTask` 和 `CreateTradingTask` 保持 `json:"-"`，客户端不能在创建 payload 中伪造 request ID。
- `0034_task_request_ids.sql` 为 `data_sync_tasks`、`backtest_tasks` 和 `trading_tasks` 增加 nullable `request_id` 字段及非空 partial index，便于按 API 请求关联任务。
- `web/frontend/src/types/api.generated.ts` 已由后端 OpenAPI contract 重新生成，三类任务响应类型包含可选 `requestId`。

范围外：

- 本小节不覆盖 repair 派生 data sync task；repair 派生任务 request ID 关联见下一小节。
- 不覆盖 notification delivery、交易所 / provider 请求、worker 业务日志、W3C `traceparent`、OpenTelemetry 或外部日志 sink。
- 不改变任务状态机、worker lease、任务 claim / retry / stop 语义、认证 / CSRF、前端运行时行为或创建请求 JSON schema。
- 不把 API 创建任务 request ID 关联等同于完整分布式 trace。

当前验证：

- `scripts/generate-api-types.sh` 通过。
- `go test ./internal/web/api -run 'Test(TaskCreateRoutesPropagateRequestID|APIContract|FrontendAPIGeneratedTypesAreCurrent)' -count=1 -v` 通过。
- `go test ./internal/web/api -count=1` 通过。
- `go test ./internal/store/postgres -count=1` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过，55 个 test file / 240 个测试。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `scripts/check-file-size.sh` 通过。
- `git diff --check` 通过。

未执行：

- 未执行 Docker Compose / 真实 8080 API task request ID smoke；当前本机 Docker daemon 仍不可用。该能力已由 API handler / contract 单元测试覆盖，Compose 运行态证据仍需 Docker 可用后补跑。
- 未执行浏览器 / 视觉 smoke；本轮没有前端渲染变更。

剩余风险：

- notification delivery 和外部系统仍没有 request ID / trace 传播。
- notification delivery 日志还没有完整 request ID 端到端关联。
- 仍缺外部日志 sink、日志保留策略、告警规则和生产排障 runbook。
- 项目整体仍为 `scaffold`，不能升级为 usable 或 production-safe。

### 阶段 8 repair 派生 data sync task request ID 补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- `POST /api/data/tasks/{id}/repair-gaps`、`/repair-gap`、`/repair-invalid-issues` 从当前请求 context 读取 `RequestIDFromContext`，并传入 repository repair request。
- `POST /api/market/candle-gaps/repair`、`/repair-batch` 和 `/api/market/candle-invalid-issues/repair` 同步传入当前 request ID。
- `insertDataSyncRepairTask` 和 `insertMarketCandleRepairTask` 对新建补同步任务写入 nullable `request_id`；重复窗口命中已有任务时仍返回 `skippedExisting`，不回填旧任务。
- repair request 结构中的 `RequestID` 均保持 `json:"-"`，OpenAPI / 前端 request DTO 不暴露该字段。
- response 中新创建的 repair task 继续通过 `DataSyncTask.requestId` 暴露只读关联 ID。

范围外：

- 本小节不覆盖 notification delivery；notification request ID 关联见后续小节。
- 不覆盖交易所 / provider 请求、worker 业务日志、W3C `traceparent`、OpenTelemetry 或外部日志 sink。
- 不改变 repair 去重规则、任务状态机、worker lease、claim / retry / stop 语义、认证 / CSRF、前端运行时行为或创建请求 JSON schema。
- 不为已有历史 repair task 回填 request ID。

当前验证：

- `scripts/generate-api-types.sh` 通过。
- `go test ./internal/web/api -run 'Test(TaskCreateRoutesPropagateRequestID|RepairRoutesPropagateRequestID|APIContract|FrontendAPIGeneratedTypesAreCurrent)' -count=1 -v` 通过。
- `go test ./internal/data -count=1` 通过。
- `go test ./internal/store/postgres -count=1` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过，55 个 test file / 240 个测试。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `scripts/check-file-size.sh` 通过。
- `git diff --check` 通过。

未执行：

- 未执行 Docker Compose / 真实 8080 repair task request ID smoke；当前本机 Docker daemon 仍不可用。该能力已由 API handler / contract 单元测试覆盖，Compose 运行态证据仍需 Docker 可用后补跑。
- 未执行浏览器 / 视觉 smoke；本轮没有前端渲染变更。

剩余风险：

- notification provider request ID / traceparent 和 data sync market request metadata 传播已由后续补充覆盖；更广外部系统仍没有 request ID / trace 闭环。
- 仍缺外部日志 sink、日志保留策略、告警规则和生产排障 runbook。
- 项目整体仍为 `scaffold`，不能升级为 usable 或 production-safe。

### 阶段 8 worker task 日志 request ID 补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 新增 `internal/workerlog.TaskAttrs`，统一生成 `task_id` 和可选 `request_id` 日志字段；task 没有 request ID 时不输出空 `request_id`。
- data sync runner 的任务失败、临时市场数据错误重试、重试记录/失败记录前 lease race、exchange fetch lock skip / error / shutdown release 相关业务日志已带 task request ID。
- data sync runner 的单次市场数据 fetch 重试日志已带 task request ID、exchange、symbol、interval、attempt 和 max attempts。
- backtest runner 和 trading runner 的任务失败日志已带 task request ID。

范围外：

- 本小节不覆盖 notification delivery 和 notification outbox；notification request ID 关联见下一小节。
- 不覆盖交易所 / provider HTTP 请求、W3C `traceparent`、OpenTelemetry 或外部日志 sink。
- 不改变 task claim、heartbeat、release、retry、failure 状态语义或日志级别。
- 不为没有 request ID 的历史任务伪造关联 ID。

当前验证：

- `go test ./internal/workerlog ./internal/datasync ./internal/backtest ./internal/trading -count=1` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `scripts/check-file-size.sh` 通过。
- `git diff --check` 通过。
- `go vet ./internal/workerlog ./internal/datasync ./internal/backtest ./internal/trading` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过，55 个 test file / 240 个测试。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `scripts/check-file-size.sh` 通过。
- `git diff --check` 通过。

未执行：

- 未执行 Docker Compose / 真实 worker 日志 request ID smoke；当前本机 Docker daemon 仍不可用。该能力已由 helper 单元测试和受影响 runner 包编译/测试覆盖，Compose 运行态证据仍需 Docker 可用后补跑。
- 未执行浏览器 / 视觉 smoke；本轮没有前端渲染变更。

剩余风险：

- notification provider request ID / traceparent 和 data sync market request metadata 传播已由后续补充覆盖；更广外部系统仍没有 request ID / trace 闭环。
- 仍缺外部日志 sink、日志保留策略、告警规则和生产排障 runbook。
- 项目整体仍为 `scaffold`，不能升级为 usable 或 production-safe。

### 阶段 8 notification request ID 关联补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 新增 `0035_notification_request_ids.sql`，为 `notifications` 和 `notification_outbox` 增加 nullable `request_id` 字段及非空 partial index。
- trading runner 生成 notification 时继承 `TradingTask.RequestID`；`SaveTradingRunResult` 同时写入 `notifications.request_id` 和 `notification_outbox.request_id`。
- `Notification` API response 暴露可选只读 `requestId`；`NotificationDelivery` claim 结果带出 request ID。
- notify runner 的 delivery failure 日志通过 `workerlog.TaskAttrs` 输出 `task_id`、`request_id`、`delivery_id` 和 error。
- `web/frontend/src/types/api.generated.ts` 已由后端 OpenAPI contract 重新生成，`Notification` 类型包含可选 `requestId`。

范围外：

- 不向 webhook / Telegram / 飞书 / SMTP / 交易所等外部 provider 请求注入 request ID header 或 trace context。
- 不引入 W3C `traceparent`、OpenTelemetry、外部日志 sink、日志保留策略或告警规则。
- 不为已有历史 notification / outbox 行回填 request ID。

当前验证：

- `scripts/generate-api-types.sh` 通过。
- `go test ./internal/trading ./internal/notification ./internal/store/postgres -count=1` 通过。
- `go test ./internal/web/api -run 'Test(APIContract|FrontendAPIGeneratedTypesAreCurrent|FrontendAPIResponseTypesMatchContractFields)' -count=1 -v` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `pnpm --dir web/frontend run test` 通过（55 个 test files / 240 个 tests）。
- `pnpm --dir web/frontend run build` 通过。
- `scripts/quality-gate.sh` 通过。
- `scripts/check-file-size.sh` 通过。
- `git diff --check` 通过。

未执行：

- 未执行 Docker Compose / 真实 notification delivery request ID smoke；当前本机 Docker daemon 仍不可用。该能力已由 trading runner 单元测试、store 编译测试和 API contract/generated DTO 检查覆盖，Compose 运行态证据仍需 Docker 可用后补跑。
- 未执行浏览器 / 视觉 smoke；本轮没有前端渲染变更。

剩余风险：

- notification provider HTTP request ID / traceparent 和 data sync market request metadata 传播已由后续补充覆盖；更广外部系统仍没有 request ID / trace 闭环。
- 仍缺外部日志 sink、日志保留策略、告警规则和生产排障 runbook。
- 项目整体仍为 `scaffold`，不能升级为 usable 或 production-safe。

### 阶段 8 notification provider request ID 传播补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- webhook / Telegram / 飞书 HTTP provider 在 `NotificationDelivery.RequestID` 非空且不含 CR/LF 时向外发请求写入 `X-Request-ID` header。
- webhook provider 的 JSON payload 新增可选 `requestId`，用于 webhook 接收端直接记录业务关联 ID。
- email provider 在 `MailMessage` 和 SMTP 邮件正文 header 中写入同一个 `X-Request-ID`，便于邮件链路按 request ID 搜索。
- notification provider 单元测试覆盖 webhook header/payload、非法 request ID header 跳过、Telegram / 飞书外发 header 和 email header 输出。

范围外：

- 不向 Binance / OKX / 其他交易所请求或非 notification provider 的外部系统请求注入 request ID。
- 不引入 W3C `traceparent`、OpenTelemetry、外部日志 sink、日志保留策略或告警规则。
- 不保证 Telegram / 飞书 / SMTP 中继或 webhook 接收端一定保存、回显或按该 header 建立回执关联。

当前验证：

- `go test ./internal/notification -count=1` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `scripts/check-file-size.sh` 通过。
- `git diff --check` 通过。

未执行：

- 未执行 Docker Compose / 真实 notification provider 外网投递 smoke；当前本机 Docker daemon 仍不可用，且本轮不调用真实 Telegram / 飞书 / SMTP / webhook 服务。
- 未执行浏览器 / 视觉 smoke；本轮没有前端渲染变更。

剩余风险：

- data sync market request metadata 和 notification provider traceparent 传播已由后续补充覆盖；其他交易所请求和更广外部系统仍没有 request ID / trace 闭环。
- 仍缺 OpenTelemetry、外部日志 sink、日志保留策略、告警规则和生产排障 runbook。
- 项目整体仍为 `scaffold`，不能升级为 usable 或 production-safe。

### 阶段 8 worker PostgreSQL readiness 探针补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- `hi sync`、`hi backtest`、`hi trading`、`hi notify` 的可选 worker probe 在启动时接入当前 PostgreSQL store 的 `Ping` 检查。
- `/livez` 保持进程存活语义，不执行数据库检查。
- `/readyz` 和 `/healthz` 在每次请求时执行 PostgreSQL ping；成功返回 `status=ok` 和 `checks.postgres=ok`，失败返回 HTTP 503、`status=unavailable` 和 `checks.postgres=unavailable`，不把底层错误写入响应体。
- worker health probe JSON 响应新增 `uptimeSeconds`，便于区分刚启动与长时间运行的 worker。
- `cmd/hi` 单元测试覆盖 readiness 成功、PostgreSQL readiness 失败、`/livez` 不触发 readiness check、写方法 405 和本地 listener readyz。
- `docs/go-command-runbook.md` 与 `docs/production-runbook.md` 已同步新 probe 语义和剩余边界。

范围外：

- 本小节不新增队列 backlog、claim 可用性、stale worker、exchange backoff、fetch-lock skip、catalog freshness 或 notification provider 可达性的 worker probe 检查；可配置 claim-ready backlog readiness 见后续补充。
- 不新增 Prometheus 指标、SLO、告警规则、外部 uptime monitor 或 Docker Compose 默认 healthcheck。
- 不改变 worker lease、heartbeat、claim、retry/backoff、SIGTERM 收尾、system health API 或运维健康页逻辑。

当前验证：

- `go test ./cmd/hi -count=1` 通过。
- `go test ./internal/store/postgres -count=1` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `scripts/check-file-size.sh` 通过。
- `git diff --check` 通过。

未执行：

- 未执行 Docker Compose worker health probe smoke；当前本机 Docker daemon 仍不可用。该能力已由本地 Go HTTP listener 单元测试覆盖，Compose 运行态证据仍需 Docker 可用后补跑。
- 未执行浏览器 / 视觉 smoke；本轮没有前端渲染变更。

剩余风险：

- worker probe 只覆盖进程存活和 PostgreSQL ping；可配置 claim-ready backlog readiness 见后续补充，仍不覆盖 claim 成功率、stale worker、交易所退避、fetch-lock skip、catalog freshness 或 provider 可达性。
- 仍缺生产级监控、SLO、告警规则、外部 uptime monitor、资源容量策略和目标环境恢复演练证据。
- 项目整体仍为 `scaffold`，不能升级为 usable 或 production-safe。

### 阶段 8 worker 队列表 readiness 补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- worker readiness check 从单个函数扩展为多个命名检查，`/readyz` 和 `/healthz` 响应的 `checks` 可同时返回 `postgres` 与 `queue` 状态。
- `hi sync`、`hi backtest`、`hi trading`、`hi notify` 的 probe 启动时分别注册 PostgreSQL ping 和对应队列表可读检查。
- `CheckWorkerQueue` 按 worker 类型轻量读取对应表：`data_sync_tasks`、`backtest_tasks`、`trading_tasks`、`notification_outbox`，验证 pending/running 状态列和表结构可访问；查询成功即为 ready，不要求当前一定有可 claim 任务。
- 任一 readiness check 失败时 `/readyz` / `/healthz` 返回 HTTP 503、`status=unavailable`，失败 check 标记为 `unavailable`，已通过的 check 保持 `ok`；`/livez` 仍不触发 readiness check。
- `cmd/hi` 单元测试覆盖多 check 成功、queue check 失败、`/livez` 不触发 readiness check。

范围外：

- 不把队列为空、任务积压、claim 饥饿、stale worker、exchange backoff、fetch-lock skip、catalog freshness 或 notification provider 可达性作为 readiness 失败条件；可配置 claim-ready backlog readiness 见后续补充。
- 不新增 Prometheus 指标、SLO、告警规则、外部 uptime monitor 或 Docker Compose 默认 healthcheck。
- 不改变 worker lease、heartbeat、claim、retry/backoff、SIGTERM 收尾、system health API 或运维健康页逻辑。

当前验证：

- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `scripts/check-file-size.sh` 通过。
- `git diff --check` 通过。

未执行：

- 未执行 Docker Compose worker health probe smoke；当前本机 Docker daemon 仍不可用。该能力已由本地 Go HTTP listener 单元测试和 store 包编译/测试覆盖，Compose 运行态证据仍需 Docker 可用后补跑。
- 未执行浏览器 / 视觉 smoke；本轮没有前端渲染变更。

剩余风险：

- worker probe 只证明对应队列表可读；可配置 claim-ready backlog readiness 见后续补充，仍不衡量 claim 成功率、处理时延、stale worker、交易所退避、fetch-lock skip、catalog freshness 或 provider 可达性。
- 仍缺生产级监控、SLO、告警规则、外部 uptime monitor、资源容量策略和目标环境恢复演练证据。
- 项目整体仍为 `scaffold`，不能升级为 usable 或 production-safe。

### 阶段 8 worker backlog readiness 补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 新增 `<COMMAND>_READY_MAX_BACKLOG` 和 `<COMMAND>_READY_MAX_AGE`，支持 `SYNC` / `BACKTEST` / `TRADING` / `NOTIFY` 四类 worker；空值或 0 表示关闭。
- 配置任一正阈值后，worker `/readyz` / `/healthz` 注册 `queue_backlog` check；超过 claim-ready backlog 数量或最老 ready 任务 age 时返回 HTTP 503 和 `checks.queue_backlog=unavailable`。
- backlog 查询按 worker claim 语义统计可领取工作：sync pending 且启用、到期、active catalog、无 active exchange backoff；backtest pending；trading running 且可领取；notify pending / retry_scheduled 且到期。
- Docker Compose 和 `.env.example` 已透传所有 backlog readiness env，`docs/go-command-runbook.md` 与 `docs/production-runbook.md` 已记录启用方式和边界。
- 单元测试覆盖 env 解析、阈值错误、backlog 阈值判断和各 worker backlog 查询语义。

范围外：

- 不把 backlog readiness 默认启用，不改变现有 worker claim、lease、heartbeat、retry/backoff 或 system health API。
- 不新增 claim 成功率、处理耗时、exchange backoff、fetch-lock skip、catalog freshness 或 notification provider 可达性 readiness；stale-lease readiness 见后续补充。
- 不新增 Prometheus 指标、SLO、告警规则、外部 uptime monitor 或 Docker Compose 默认 healthcheck。

当前验证：

- `docker compose --env-file .env.example config --quiet` 通过。
- `bash -n scripts/stage8-command-config-smoke.sh` 通过。
- `go test ./cmd/hi ./internal/store/postgres -count=1` 通过。
- `scripts/stage8-command-config-smoke.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `scripts/check-file-size.sh` 通过。
- `git diff --check` 通过。

未执行：

- 未执行 Docker Compose worker backlog readiness smoke；当前本机 Docker daemon 仍不可用。

剩余风险：

- backlog readiness 只在显式配置阈值后生效，不等同于容量规划、吞吐压测、claim 成功率监控或外部依赖可达性检查。
- 项目整体仍为 `scaffold`，不能升级为 usable 或 production-safe。

### 阶段 8 worker stale lease readiness 补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 新增 `<COMMAND>_READY_MAX_STALE_LEASES`，支持 `SYNC` / `BACKTEST` / `TRADING` / `NOTIFY` 四类 worker；空值表示关闭，`0` 表示任何 stale lease 都会让 readiness 失败。
- 配置阈值后，worker `/readyz` / `/healthz` 注册 `stale_leases` check；超过阈值时返回 HTTP 503 和 `checks.stale_leases=unavailable`。
- stale lease 查询按 worker 表读取 `locked_until IS NOT NULL AND locked_until < now()`；sync 额外排除 soft-deleted task。
- Docker Compose 和 `.env.example` 已透传所有 stale lease readiness env，`docs/go-command-runbook.md` 与 `docs/production-runbook.md` 已记录启用方式和边界。
- 单元测试覆盖 env 解析、阈值错误、stale lease 阈值判断和各 worker stale lease 查询语义。

范围外：

- 不把 stale lease readiness 默认启用，不改变现有 worker claim、lease、heartbeat、retry/backoff 或 system health API。
- 不新增 claim 成功率、处理耗时、fetch-lock skip、catalog freshness 或 notification provider 可达性 readiness；sync exchange-backoff readiness 见后续补充。
- 不新增 Prometheus 指标、SLO、告警规则、外部 uptime monitor 或 Docker Compose 默认 healthcheck。

当前验证：

- `docker compose --env-file .env.example config --quiet` 通过。
- `bash -n scripts/stage8-command-config-smoke.sh` 通过。
- `go test ./cmd/hi ./internal/store/postgres -count=1` 通过。
- `scripts/stage8-command-config-smoke.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `scripts/check-file-size.sh` 通过。
- `git diff --check` 通过。

未执行：

- 未执行 Docker Compose worker stale lease readiness smoke；当前本机 Docker daemon 仍不可用。

剩余风险：

- stale lease readiness 只在显式配置阈值后生效，不等同于 claim 成功率监控、处理时延监控或外部依赖可达性检查。
- 项目整体仍为 `scaffold`，不能升级为 usable 或 production-safe。

### 阶段 8 sync exchange backoff readiness 补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 新增 `SYNC_READY_MAX_EXCHANGE_BACKOFFS`；空值表示关闭，`0` 表示任何 active exchange backoff 都会让 `hi sync` readiness 失败。
- 配置阈值后，`hi sync` `/readyz` / `/healthz` 注册 `exchange_backoff` check；超过阈值时返回 HTTP 503 和 `checks.exchange_backoff=unavailable`。
- readiness check 只读取 PostgreSQL `data_sync_exchange_backoffs.next_attempt_at > now()` 的持久化 backoff 状态，不向 Binance / OKX 发实时探测请求。
- Docker Compose 和 `.env.example` 已透传该 env，`docs/go-command-runbook.md` 与 `docs/production-runbook.md` 已记录启用方式和边界。
- 单元测试覆盖 env 解析、阈值错误和 active exchange backoff 阈值判断。

范围外：

- 不把 exchange-backoff readiness 默认启用，不改变现有 sync retry/backoff、claim、lease、heartbeat 或 system health API。
- 不新增真实 Binance / OKX availability probe、live provider delivery probe、claim 成功率、处理耗时或 fetch-lock skip readiness；catalog freshness 和 notify provider config readiness 见后续补充。
- 不新增 Prometheus 指标、SLO、告警规则、外部 uptime monitor 或 Docker Compose 默认 healthcheck。

当前验证：

- `docker compose --env-file .env.example config --quiet` 通过。
- `bash -n scripts/stage8-command-config-smoke.sh` 通过。
- `go test ./cmd/hi ./internal/store/postgres -count=1` 通过。
- `scripts/stage8-command-config-smoke.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `scripts/check-file-size.sh` 通过。
- `git diff --check` 通过。

未执行：

- 未执行 Docker Compose sync exchange-backoff readiness smoke；当前本机 Docker daemon 仍不可用。
- 未执行真实 Binance / OKX 可达性探测；本补充刻意不对真实交易所发送 readiness 探测请求。

剩余风险：

- exchange-backoff readiness 只反映已经持久化的 sync exchange backoff，不等同于实时外部依赖可达性监控。
- 项目整体仍为 `scaffold`，不能升级为 usable 或 production-safe。

### 阶段 8 sync catalog freshness readiness 补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 新增 `SYNC_READY_MAX_CATALOG_STALENESS`；空值表示关闭，配置正 duration 后为 `hi sync` `/readyz` / `/healthz` 注册 `catalog_freshness` check。
- readiness 只读取 PostgreSQL `market_instrument_sync_statuses`，不向 Binance / OKX 发送实时探测请求。
- 配置阈值后，无 catalog sync status、任一交易所从未成功同步、最近一次 catalog attempt 失败且没有更新的 success、或 last success 超过配置 duration 时，`catalog_freshness` 返回 unavailable。
- Docker Compose、`.env.example`、Go 子命令手册、生产运行手册和 command config smoke 同步该配置入口。
- 单元测试覆盖 env 解析、阈值错误、check 注册、fresh/stale/never-synced/failed-without-newer-success 判断和不泄露 raw last_error。

范围外：

- 不把 catalog freshness readiness 默认启用，不改变现有 marketsync 拉取、catalog 写入、data sync claim、lease、heartbeat 或 system health API。
- 不新增真实 Binance / OKX live availability probe、claim 成功率、处理耗时、fetch-lock skip 或 notification provider delivery readiness。
- 不新增 Prometheus 指标、SLO、告警规则、外部 uptime monitor 或 Docker Compose 默认 healthcheck。

当前验证：

- `docker compose --env-file .env.example config --quiet` 通过。
- `bash -n scripts/stage8-command-config-smoke.sh` 通过。
- `go test ./cmd/hi ./internal/store/postgres -count=1` 通过。
- `scripts/stage8-command-config-smoke.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `scripts/check-file-size.sh` 通过。
- `git diff --check` 通过。

未执行：

- 未执行 Docker Compose sync catalog freshness readiness smoke；当前本机 Docker daemon 仍不可用。
- 未执行真实 Binance / OKX 可达性探测；本补充刻意不对真实交易所发送 readiness 探测请求。

剩余风险：

- catalog freshness readiness 只反映已持久化的 instrument catalog sync status，不等同于实时交易所可达性监控。
- 仍缺 claim 成功率 / 处理耗时 / live provider delivery readiness、生产级监控、SLO 和告警接入。
- 项目整体仍为 `scaffold`，不能升级为 usable 或 production-safe。

### 阶段 8 notify provider config readiness 补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 新增 `NOTIFY_READY_VALIDATE_PROVIDER_CONFIG`；默认 `false`，设置为 `true` 后 `hi notify` readiness 注册 `notification_providers` check。
- readiness check 读取 enabled notification channels，并复用 provider target parser 本地验证 provider 名称、target 格式和必需 env reference 是否存在。
- 覆盖 local / webhook-demo / webhook / email / Telegram / 飞书 provider target validation；不会发送真实 notification delivery。
- Docker Compose 和 `.env.example` 已透传该 env，`docs/go-command-runbook.md` 与 `docs/production-runbook.md` 已记录启用方式和边界。
- 单元测试覆盖 env 解析、非法 bool、readiness check 注册和 provider target validation。

范围外：

- 不把 provider config readiness 默认启用，不改变现有 notification outbox claim、lease、retry 或 provider delivery 语义。
- 不新增真实 Telegram / 飞书 / SMTP / webhook delivery probe、外部 uptime monitor、claim 成功率、处理耗时或 provider latency readiness。
- 不新增 Prometheus 指标、SLO、告警规则或 Docker Compose 默认 healthcheck。

当前验证：

- `docker compose --env-file .env.example config --quiet` 通过。
- `bash -n scripts/stage8-command-config-smoke.sh` 通过。
- `go test ./cmd/hi ./internal/notification -count=1` 通过。
- `scripts/stage8-command-config-smoke.sh` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `scripts/check-file-size.sh` 通过。
- `git diff --check` 通过。

未执行：

- 未执行 Docker Compose notify provider config readiness smoke；当前本机 Docker daemon 仍不可用。
- 未执行真实 Telegram / 飞书 / SMTP / webhook 投递探测；本补充刻意不对第三方 provider 发送 readiness 探测请求。

剩余风险：

- provider config readiness 只证明已启用 channel 的 provider target 和 env reference 可本地解析，不等同于第三方 provider 实时可达性或投递成功率监控。
- 项目整体仍为 `scaffold`，不能升级为 usable 或 production-safe。

### 阶段 8 capacity preflight 补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 新增 `scripts/stage8-capacity-check.sh`，作为离线容量预算 preflight。
- 默认按 5 个长运行 `hi` 进程估算，检查 `DB_MAX_CONNS * STAGE8_HI_PROCESS_COUNT` 是否落在 PostgreSQL 为应用保留的连接预算内。
- 检查 CPU millicores、内存 MB、目标路径可用磁盘、每日备份估算大小和备份保留天数，所有预算参数都可通过 `STAGE8_*` env 覆盖。
- 脚本会在预算值非法、连接预算超限、CPU / memory 预算超限、容量路径不存在或磁盘无法覆盖最小空闲空间加备份保留估算时失败。
- `scripts/quality-gate.sh` 已加入 capacity check，默认配置必须在本地质量门禁中通过。
- `docs/production-runbook.md` 记录 capacity preflight 命令、覆盖参数和“预算检查不等于负载测试”的边界。

范围外：

- 不执行真实压测、长期 soak、冷缓存测试、生产数据分布测试或自动扩缩容。
- 不自动读取目标 PostgreSQL `max_connections`、容器 CPU/memory limit、云盘容量、备份实际增量或对象存储策略。
- 不新增自动备份调度、备份删除任务、外部监控、SLO 或告警规则。

当前验证：

- `scripts/stage8-capacity-check.sh` 通过。
- 故意超出 PostgreSQL 连接预算的负向用例按预期失败。
- `bash -n scripts/stage8-capacity-check.sh scripts/quality-gate.sh` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

未执行：

- 未执行目标环境容量测试、负载测试、恢复演练或长期 soak；当前本机 Docker daemon 仍不可用，无法补 Compose 运行态容量证据。
- 未执行浏览器 / 视觉 smoke；本轮没有前端渲染变更。

剩余风险：

- capacity preflight 只验证声明式预算，不证明真实吞吐、延迟、冷缓存、磁盘增长、备份大小、恢复耗时或交易所外部依赖韧性。
- 仍缺目标环境 sizing 记录、自动化备份调度、备份保留 enforcement、SLO、告警规则和外部 uptime monitor。
- 项目整体仍为 `scaffold`，不能升级为 usable 或 production-safe。

### 阶段 8 backup automation scaffold 补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- 新增 `scripts/stage8-backup.sh`，封装单次 PostgreSQL `pg_dump -Fc` 备份、非空 dump 校验、原子落盘和按 `STAGE8_BACKUP_RETENTION_DAYS` 清理旧 `tictick-hi-*.dump`。
- 备份脚本支持 `--dry-run`，可在不调用 Docker 的情况下校验 `POSTGRES_USER`、`POSTGRES_DB`、备份 stamp、保留天数和目标文件路径。
- 新增 `scripts/stage8-backup-dry-run-smoke.sh`，覆盖备份 dry-run 正向路径和非法 retention 负向路径，并接入 `scripts/quality-gate.sh`。
- 新增 `deploy/systemd/tictick-hi-backup.service` 和 `deploy/systemd/tictick-hi-backup.timer`，作为目标主机每日备份调度模板。
- `docs/production-runbook.md` 已补备份脚本、dry-run、systemd timer 安装步骤和剩余边界。

范围外：

- 不在本仓库启用目标主机 systemd timer，不写入真实外部备份存储，不执行真实 `pg_dump`。
- 不新增对象存储上传、备份加密、备份完整性签名、保留策略审计、外部监控或告警规则。
- 不把 timer 模板等同于目标环境已完成自动备份或恢复能力。

当前验证：

- `POSTGRES_USER=stage8 POSTGRES_DB=tictick_hi scripts/stage8-backup.sh --dry-run` 通过。
- 非法 `STAGE8_BACKUP_RETENTION_DAYS=0` dry-run 负向用例按预期失败。
- `bash -n scripts/stage8-backup.sh scripts/stage8-backup-restore-drill.sh` 通过。
- `scripts/stage8-backup-dry-run-smoke.sh` 通过。
- `bash -n scripts/stage8-backup.sh scripts/stage8-backup-dry-run-smoke.sh scripts/quality-gate.sh` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

未执行：

- 未执行真实备份、真实旧备份清理、systemd timer 安装或目标环境外部存储写入；当前本机 Docker daemon 仍不可用，无法补 Compose 运行态备份证据。
- 未执行浏览器 / 视觉 smoke；本轮没有前端渲染变更。

剩余风险：

- 仍缺目标主机 scheduler 安装证据、外部备份存储监控、备份加密/签名、保留策略审计和恢复演练通过记录。
- 仍缺 SLO、告警规则、外部 uptime monitor、目标环境容量测试和长期运行证据。
- 项目整体仍为 `scaffold`，不能升级为 usable 或 production-safe。

### 阶段 8 API traceparent 边界补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- API HTTP 边界支持 W3C `traceparent`：合法 v00 traceparent 会被复用并规范为小写，缺失、非法、全零 trace ID / span ID 或不支持版本会生成新的 v00 traceparent。
- 所有 API / frontend / readyz 响应都会返回 `traceparent` header；非法输入不会回显到响应头。
- HTTP access log 新增 `trace_id` 字段，并继续只记录 path 不记录 query string。
- OpenAPI response headers 统一声明 `traceparent`，与现有 `X-Request-ID` header 一起覆盖成功和错误响应。
- 单元测试覆盖生成 traceparent、复用合法 traceparent、替换非法 traceparent 且不回显、access log 输出 `trace_id` 且不泄露 query string。

范围外：

- 本小节自身不把 W3C trace context 持久化到 worker task 或 notification delivery；这些持久化路径见后续补充。
- 本小节自身不向 worker 日志、notification provider、Binance / OKX 或其他外部系统传播 `traceparent`；worker 日志、notification provider 和 data sync market request 传播见后续补充。
- 不引入 OpenTelemetry SDK、span lifecycle、采样策略、外部 tracing backend、日志 sink 或 retention policy。

当前验证：

- `scripts/generate-api-types.sh` 通过。
- `go test ./internal/web/api -count=1` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `git diff --check` 通过。

未执行：

- 未执行 Docker Compose / 真实 8080 traceparent smoke；当前本机 Docker daemon 仍不可用。该能力已由 HTTP handler 单元测试和 OpenAPI contract 包测试覆盖，Compose 运行态证据仍需 Docker 可用后补跑。
- 未执行浏览器 / 视觉 smoke；本轮没有前端渲染变更。

剩余风险：

- traceparent 已进入 API HTTP 边界、access log、API-created task 行、data sync / backtest / trading / notify worker task 日志、notification delivery、provider 外发、data sync market HTTP 请求和 command run-level 日志上下文；更广外部系统和独立启动子命令之间的自动传播仍未闭环。
- 仍缺 OpenTelemetry、外部 trace backend、外部日志 sink、日志保留策略、SLO、告警规则和生产排障 runbook。
- 项目整体仍为 `scaffold`，不能升级为 usable 或 production-safe。

### 阶段 8 API-created task traceparent 关联补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- `0036_task_traceparents.sql` 为 data sync / backtest / trading task 增加 nullable `traceparent` 字段和部分索引。
- `POST /api/data/tasks`、`POST /api/backtests`、`POST /api/trading/tasks` 会把 API context 中规范化后的 W3C `traceparent` 写入创建请求并随响应返回。
- data sync task gap / invalid issue repair、market candle gap / batch gap / invalid issue repair 创建的补同步任务会保存同一个 `traceparent`。
- Postgres insert / returning / scan 路径使用 `NULLIF(..., '')` 与 `COALESCE(..., '')` 保持缺失值语义，并更新生成的前端 TypeScript task DTO。
- 单元测试覆盖 task create routes 和 repair routes 对 `traceparent` 的传播。

范围外：

- 本小节自身不把 `traceparent` 注入 worker 业务日志、notification delivery、provider 请求、交易所请求或其他子命令；worker 日志、notification delivery / provider 和 data sync market request 传播见后续补充。
- 不引入 OpenTelemetry SDK、span lifecycle、采样策略、外部 tracing backend、日志 sink 或 retention policy。

当前验证：

- `scripts/generate-api-types.sh` 通过。
- `go test ./internal/web/api -run 'TestTaskCreateRoutesPropagateTraceParent|TestRepairRoutesPropagateTraceParent|TestTaskCreateRoutesPropagateRequestID|TestRepairRoutesPropagateRequestID|TestServerReusesValidTraceParent' -count=1` 通过。
- `go test ./internal/store/postgres -run '^$' -count=1` 通过。
- `go test ./internal/web/api ./internal/store/postgres -count=1` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `scripts/check-file-size.sh` 通过。
- `git diff --check` 通过。

未执行：

- 未执行 Docker Compose / 真实 8080 traceparent smoke；当前本机 Docker daemon 仍不可用。

剩余风险：

- traceparent 已保存到 API-created task 行，并在后续补充进入 data sync / backtest / trading / notify worker task 日志、notification delivery、provider 外发、data sync market HTTP 请求和 command run-level 日志上下文；其他 exchange 调用和独立启动子命令之间的自动 W3C trace context 传播仍未闭环。
- 项目整体仍为 `scaffold`，不能升级为 usable 或 production-safe。

### 阶段 8 worker task 日志 trace ID 补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- `internal/workerlog` 增加 trace-aware task log attrs helper，从合法 W3C `traceparent` 提取小写 `trace_id`。
- data sync / backtest / trading worker 业务日志在 task 带合法 `traceparent` 时输出 `trace_id`，继续在 task 带 request ID 时输出 `request_id`。
- 非法、缺失或全零 traceparent 不写入 `trace_id` 日志字段，避免坏输入污染结构化日志。
- 单元测试覆盖合法 traceparent、大小写规范化、非法输入和全零 traceparent。

范围外：

- 本小节不把 `traceparent` 注入 notification delivery / notify worker 日志、notification provider 请求、交易所请求或其他子命令；notification delivery / provider 和 data sync market request traceparent 见后续补充。
- 不引入 OpenTelemetry SDK、span lifecycle、采样策略、外部 tracing backend、日志 sink 或 retention policy。

当前验证：

- `go test ./internal/workerlog ./internal/datasync ./internal/backtest ./internal/trading -count=1` 通过。

未执行：

- 未执行 Docker Compose / 真实 worker 日志 smoke；当前本机 Docker daemon 仍不可用。

剩余风险：

- trace ID 已进入 data sync / backtest / trading worker 业务日志；notification delivery / provider、data sync market request traceparent 和 command run-level traceparent 已由后续补充覆盖，其他 exchange 调用和独立启动子命令之间的自动 trace context 传播仍未闭环。
- 项目整体仍为 `scaffold`，不能升级为 usable 或 production-safe。

### 阶段 8 notification traceparent 传播补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- `0037_notification_traceparents.sql` 为 `notifications` / `notification_outbox` 增加 nullable `traceparent` 字段和部分索引。
- trading runner 创建 notification 时从 trading task 复制 `TraceParent`。
- notifications / notification_outbox insert、list、claim、retry scan 路径返回 `traceparent`，notify worker delivery 日志通过 `TaskTraceAttrs` 输出 `trace_id`。
- webhook payload 和 HTTP provider 请求头（webhook / Telegram / 飞书）传播合法小写 `traceparent`；SMTP message header 传播合法 `traceparent`；非法 / 全零 traceparent 不外发。
- 生成的前端 `Notification` DTO 包含 `traceparent?: string`。

范围外：

- 本小节不向 Binance / OKX / exchange market requests 或其他子命令传播 `traceparent`；data sync market request 传播见后续补充。
- 不引入 OpenTelemetry SDK、span lifecycle、采样策略、外部 tracing backend、日志 sink 或 retention policy。

当前验证：

- `scripts/generate-api-types.sh` 通过。
- `go test ./internal/notification ./internal/trading ./internal/store/postgres ./internal/web/api -count=1` 通过。
- `pnpm --dir web/frontend run test` 通过。
- `pnpm --dir web/frontend run build` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `scripts/check-file-size.sh` 通过。
- `git diff --check` 通过。

未执行：

- 未执行 Docker Compose / 真实 notify worker/provider smoke；当前本机 Docker daemon 仍不可用。

剩余风险：

- traceparent 已覆盖 API HTTP 边界、API-created task、trading notification、notify worker 日志、notification provider 外发、data sync market HTTP 请求和 command run-level 日志上下文；其他 exchange / broader external systems / 独立启动子命令之间的自动传播仍未覆盖。
- 项目整体仍为 `scaffold`，不能升级为 usable 或 production-safe。

### 阶段 8 data sync exchange request metadata 传播补充

执行日期：2026-07-07

目标等级：scaffold。

范围内：

- `internal/exchange` 新增 request metadata context helper，安全传播合法 `X-Request-ID` 和 W3C `traceparent`。
- data sync runner 拉取 K 线时把 claimed task 的 `RequestID` / `TraceParent` 写入 exchange request context。
- Binance `/api/v3/klines` 和 OKX `/api/v5/market/history-candles` HTTP 请求会应用这些 headers；adapter 在 context metadata 存在时也会把同一 helper 应用于 instrument 请求。
- 单元测试覆盖 context helper、Binance / OKX header 输出和 data sync runner propagation。

范围外：

- 不引入 OpenTelemetry SDK、span lifecycle、采样策略、外部 tracing backend、日志 sink 或 retention policy。
- 不覆盖 trading live / private order requests、marketsync scheduled catalog context、更广外部系统或跨子命令 trace propagation。
- 不把该能力声明为生产级分布式 tracing，只是把已有 task request metadata 带到 data sync market HTTP 边界。

当前验证：

- `go test ./internal/exchange ./internal/adapter/binance ./internal/adapter/okx ./internal/datasync -count=1` 通过。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `scripts/quality-gate.sh` 通过。
- `scripts/check-file-size.sh` 通过。
- `git diff --check` 通过。

未执行：

- 未执行真实 Binance / OKX 外网 header smoke；当前不对真实交易所发送探测请求。
- 未执行 Docker Compose smoke；当前本机 Docker daemon 仍不可用。

剩余风险：

- 只有 data sync market HTTP 请求、notification providers 和 command run-level 日志上下文会携带 request / trace metadata；其他 exchange 调用、更广外部系统和独立启动子命令之间的自动 trace context 仍未闭环。
- 项目整体仍为 `scaffold`，不能升级为 usable 或 production-safe。

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
- 概览页生产级监控语义。
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

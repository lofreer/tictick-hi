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
| API server | scaffold | 保留后加强 | 已按领域拆分，仍缺统一 request / response mapping 和更强错误边界 |
| 登录会话 | scaffold | 返工加强 | 有 cookie session，但 CSRF、防暴力破解、会话审计不足 |
| 数据同步 worker | scaffold | 返工加强 | 有 claim 和 upsert，但没有真正 heartbeat loop、优雅停止状态机 |
| CandleProvider | scaffold | 重点返工 | 有 native + 1m 聚合雏形，但没有健康判断、缺口检测、来源标记 |
| Binance / OKX K 线 adapter | scaffold | 保留后加强 | 能拉 K 线，但 symbol 规范、限流、错误分类不完整 |
| 研究页 | scaffold | 保留后打磨 | 图表和同步表有骨架，但交易对硬编码、图表研究能力很薄 |
| 回测 | scaffold | 返工 | 交易事实已移出 `float64`，但费用、滑点、触发语义仍不可信 |
| 交易 runner | scaffold | 返工 / 延后 live | paper/live executor 未真正分离，live 只是本地 pending 订单 |
| 实盘安全 | below-scaffold | 延后 | 密钥字段名叫 encrypted，实际是 digest，不是真加密也不能解密 |
| 通知 | scaffold | 返工 | 有通知记录雏形，但 provider/outbox/retry 不完整 |
| 前端基础设施 | scaffold | 保留后加强 | Vue/Naive/Pinia/i18n/主题骨架存在，业务体验仍粗糙 |
| 概览页 | scaffold | 保留后加强 | 有 scaffold 状态面板和基础健康信息，不是完整概览 |
| 质量门禁 | scaffold | 保留后加强 | 阶段 0 硬门禁已通过，实盘安全和 live executor 作为后续风险审计保留 |

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
- live trading 仍停留在本地 `pending_submission`，不是实盘 executor。

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
- `internal/trading/runner.go` 仍存在 `pending_submission`，live executor 未建立。

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

现状问题：

- `ListCandles` 在 store 中直接做 native 查询和聚合。
- 只要同周期查到任意数据就返回，不判断缺口。
- 聚合结果没有返回 `native / aggregated` 来源。
- 图表、回测、交易虽然调用 `ListCandles`，但接口语义不够强。

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

## 4. 第一条可推进切片

下一步只推进：

```text
CandleProvider + 数据同步 + 研究页
```

目标等级：`demo`，不是 usable。

Definition of Done：

- 数据同步任务能写入 `1m` K 线。
- 请求 `1m` 返回 native。
- 请求 `5m / 15m / 1h` 时没有 native 就由 `1m` 聚合。
- 返回数据来源和缺口状态。
- 研究页显示数据来源：native / aggregated。
- 研究页显示数据健康：正常 / 缺口 / 数据不足。
- 回测和交易 runner 不直接绕过 CandleProvider。
- `go test ./...` 通过。
- `scripts/quality-gate.sh` 通过或明确列出历史债失败项。

范围外：

- 实盘下单。
- tick 数据。
- 指标系统。
- 通知 provider。
- 复杂回测指标。

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

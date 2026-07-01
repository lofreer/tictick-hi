# tictick-hi 小而美交易系统实施计划

## 0. 文档状态

本文档是 `tictick-hi` 重新设计后的主计划文档。

本文档只记录已经确认的产品理解、前端交互、核心模块、实现边界和分阶段验收。后续实现如果发现边界不清，先更新本文档，再写代码。

所有实现工作必须先遵守：

- `docs/ai-delivery-protocol.md`
- `docs/quality-audit.md`

在质量审计明确关闭前，当前项目只能称为 `scaffold`，不能称为 demo、usable、production-safe 或完成。

当前已经确认：

- `tictick-hi` 是多交易所、多账号交易系统。
- 它不是单脚本策略执行器。
- 它也不是大而杂的机构化系统。
- 前端展示必须先被定义清楚。
- 顶部导航平铺，不做传统左侧管理后台。
- 数据同步和 K 线图表研究是高权重能力，且可以在同一个研究页面中呈现。
- 数据实时同步必须在系统重启后立即恢复，继续同步对应交易所、交易对、周期的数据。
- 如果第一版只同步 `1m` K 线，系统内部必须提供 K 线聚合能力，用 `1m` 生成更高周期供图表、回测、模拟盘 / 实盘策略使用。
- 策略代码沉淀在后端目录中，前端只映射选择和配置参数。
- 模拟盘 / 实盘是系统的重要任务类型，不要求先回测才能使用。
- 策略运行后产生意图，意图可以是订单，也可以是通知。
- 信号通知不是并列任务类型，而是策略意图的一种。
- 交易所账号在操作台的系统管理中配置。
- 前端整体风格参考 `tictickbot` 项目，再向 `tictick-lite` 的卡片式体验靠拢。
- 前端必须支持双色主题：浅色主题和深色主题。
- 主题色参考 `tictick-lite`。
- Logo 使用 `tictick-lite` 的 logo。
- 前端必须支持中英切换，不只是按钮展示，而是完整 i18n 能力。
- 前端要像素级抠细节，该用第三方 UI 库就用第三方 UI 库，避免手写堆出难维护 UI 代码。
- 前端构建方案确定为 Vue 3 + Vite + TypeScript + Naive UI + Pinia + vue-i18n + lightweight-charts + 原生 fetch typed wrapper + pnpm。
- 后端除数据库驱动等必要依赖外，尽量少用第三方库，主打安全、清晰、可审计实现。
- 数据库使用 PostgreSQL。
- 交易所架构必须可扩展，但当前需要支持 Binance 和 OKX。
- 系统按同一 Go 项目的单二进制多子命令模式拆分：`hi api`、`hi sync`、`hi trading`、`hi backtest`、`hi notify`、`hi migrate`。
- 第一版运行部署使用 Docker / Docker Compose；同一镜像通过不同 command 启动不同子命令。
- worker 必须使用统一 lease 状态机。
- 数据同步必须幂等，并支持重启后的断点恢复和尾部缺口修复。
- 实盘必须有明确安全边界：密钥加密、live executor 隔离、订单意图幂等、账号禁用。
- 前端必须先建立应用壳、主题、i18n、路由、API client 和图表封装，再实现业务页面。

当前待确认：

- 第一版具体支持哪些 K 线周期。
- TradingView 开源图表库的具体接入包和授权边界。
- 真实邮件、Telegram、飞书 provider 已采用 env-reference 凭据模型；生产启用边界仍需继续确认。

## 1. 产品定位

`tictick-hi` 是一个小而美的多交易所、多账号交易系统。

它支持：

- 多交易所。
- 同一交易所下多个账号。
- 单账号下运行不同标的、不同策略。
- 在操作台中配置交易所账号。
- 使用 PostgreSQL。
- 当前支持 Binance 和 OKX。
- 交易所 adapter 边界保持可扩展。
- 目标交易所、目标交易对、目标周期的 K 线数据同步。
- 完整 K 线图表研究。
- 后端策略沉淀。
- 回测验证。
- 模拟盘任务。
- 实盘任务。
- 策略意图处理：下单或通知。
- 基础操作台登录。
- 系统辅助管理能力。

它不应该变成：

- 单文件脚本。
- 只跑一个账号的策略 demo。
- 传统 CRUD 管理后台。
- `tictick-pro` 那样重治理、重证据、重平台化的系统。
- 复杂机构化交易平台。

## 2. 产品权重

以下不是线性流程，而是产品能力权重。

从高到低：

1. 数据同步 + K 线图表研究。
3. 策略沉淀。
4. 回测验证。
5. 模拟盘 / 实盘。
6. 策略运行后产生意图。
7. 意图被执行或通知。
8. 观察结果。

这些能力不是严格前后置关系。

特别说明：

- 回测用于验证策略，但系统不能表达为“必须先回测验证，才能创建模拟盘 / 实盘任务”。
- 模拟盘和实盘是重要模块，本身就可以创建和运行。
- 信号通知不是一种单独任务等级。任务是模拟盘或实盘；策略输出的意图可以是通知。

## 3. 顶部导航

操作台使用顶部平铺导航，不使用左侧管理后台导航。

主导航从左到右：

```text
概览 / 研究 / 回测 / 交易
```

右侧工具区从右往左：

```text
退出按钮 / 主题切换按钮 / 中英切换按钮 / 当前操作台账号按钮 / 系统管理菜单
```

系统管理是二级菜单，包含系统辅助能力：

- 通知管理。
- 交易所账号管理。
- 操作台账号管理。
- 运维健康。
- 后续必要辅助能力。

导航原则：

- 顶部直接平铺核心业务。
- 数据同步列表并入研究页，不单独占一个一级导航。
- 系统辅助能力收进系统管理。
- 不把系统做成传统后台。
- 不把通知、账号、运维健康挤到一级导航里抢业务主线权重。

## 4. 前端整体布局

页面整体结构：

```text
顶部主导航
顶部右侧工具区
当前页面主工作区
```

是否需要全局上下文栏暂不固定。页面可以按需要展示当前上下文，例如交易所、账号、交易对、周期、数据状态。

界面气质：

- 像交易研究工作台。
- 不像后台管理系统。
- 整体风格参考 `tictickbot`，再向 `tictick-lite` 的卡片式体验靠拢。
- 主题色参考 `tictick-lite`。
- Logo 使用 `tictick-lite` 的 logo。
- 图表页必须让图表成为主体。
- 列表页必须清爽，操作明确。
- 详情页必须能回到图表观察。
- 像素级抠细节，包括间距、字号、按钮状态、表格密度、卡片边界、暗色主题、中英文长度适配、响应式布局。
- 必须支持浅色 / 深色双色主题。
- 必须支持中文 / 英文切换。
- 能用成熟第三方 UI 库的地方使用第三方 UI 库，不自己手写一套低质量组件。
- 自定义样式只服务产品气质和细节打磨，不能演变成无边界 CSS 堆叠。

### 4.1 UI 技术原则

前端允许并鼓励使用合适的第三方 UI 库。

选择 UI 库时必须满足：

- Vue 3 生态成熟。
- 组件质量稳定。
- 表单、表格、弹窗、菜单、按钮、主题能力完整。
- 支持暗色主题。
- 支持中英文内容。
- 支持主题变量或能被稳定覆盖，确保浅色 / 深色两套主题都能像素级打磨。
- 不强迫页面变成传统后台风格。

确定技术选型：

```text
框架：Vue 3
构建：Vite
语言：TypeScript
UI：Naive UI
状态管理：Pinia
i18n：vue-i18n
图表：TradingView lightweight-charts
请求：原生 fetch + typed API wrapper
包管理：pnpm
构建产物：web/frontend/dist
静态服务：hi api 服务 web/frontend/dist
```

使用 UI 库的边界：

- 表单、表格、菜单、弹窗、按钮、开关、标签、提示等基础组件优先使用 UI 库。
- TradingView 图表独立封装，不强行塞进 UI 库组件体系。
- 页面布局和产品气质可以自定义，但不能重复造基础组件。

不再另选 Element Plus 或其它 UI 库，除非计划文档先更新并说明原因。

### 4.2 主题和国际化

前端必须内置两套主题：

- 浅色主题。
- 深色主题。

主题切换按钮位于顶部右侧工具区。

主题要求：

- 主题色以 `tictick-lite` 为参考来源。
- 两套主题都必须可用，不允许只打磨其中一套。
- 卡片、表格、按钮、菜单、弹窗、图表外壳、状态标签、输入框都要适配。
- 主题色、背景色、边框色、文字层级、hover / active / disabled 状态都要统一。
- TradingView 图表需要跟随主题切换。

前端必须内置中英两套语言：

- 中文。
- 英文。

中英切换按钮位于顶部右侧工具区。

国际化要求：

- 一级导航、系统管理菜单、按钮、表单字段、表格列、状态文案、错误提示、空状态、弹窗确认文案都必须纳入 i18n。
- 不能只翻译导航栏。
- 页面布局要考虑英文更长、中文更短的差异。
- 创建任务、回测详情、交易详情、通知记录等关键页面都必须可切换语言。

## 5. 概览页

概览页用于整体概要展示。

它应该回答：

- 当前有哪些数据同步任务。
- 哪些任务正在实时同步。
- 哪些数据源有缺口或异常。
- 最近回测任务结果概况。
- 当前模拟盘 / 实盘任务运行状态。
- 最近策略意图。
- 最近订单。
- 最近通知。
- 系统健康状态。

概览页不做复杂配置。

概览页不替代数据、研究、回测、交易详情页。

## 6. 研究页

研究页是 K 线图表和数据同步合并后的高权重核心页面。

核心能力：

- 使用 TradingView 开源图表。
- 可选择不同 K 线数据加载展示。
- 支持选择交易所。
- 支持选择交易对。
- 支持选择周期。
- 选择的展示周期可以是原始同步周期，也可以是由系统内部聚合得到的周期。
- 展示数据同步任务列表。
- 支持创建新同步任务。
- 显示数据同步状态。
- 支持从回测详情跳转或嵌入展示买卖点。

研究页的主区域：

- 数据同步列表在上，K 线图表在下。
- K 线图表占主要面积。
- 数据同步列表不会太多，作为研究页中的轻量区域展示。
- K 线图表必须处于固定 viewport 容器内，图表库内部 DOM、canvas 或运行态 inline height 变化不能反向撑高页面。
- 图表工具能力优先于表格堆叠。
- 不做“上面一堆表单，下面一小块图表”。

研究页中的数据同步列表每一项展示：

- 交易所。
- 交易对。
- 周期。
- 同步窗口。
- 补同步来源。
- 最新同步时间。
- 数据健康。
- 缺口摘要。
- 实时状态。
- 同步状态。
- 最近错误。

每一项操作按钮：

- 实时 / 停止实时。
- 同步 / 停止同步。
- 查看图表。
- 查看并修复任务窗口内缺口。
- 删除。

点击“查看图表”时，在当前研究页加载该数据源，不跳转到独立数据页。

当前图表 metadata 发现缺口时，研究页可以修复首个缺口；如果图表来自某个同步任务，修复必须优先通过后端带源任务 ID 的补同步接口创建任务，避免前端手工拼接补同步语义。

研究页上一 / 下一 K 线窗口必须优先使用 `/api/candles` 返回的 opaque cursor；旧的 `from/to` URL 仍可作为兼容入口，但前端不应手工拼接相邻窗口语义。

研究页可逐步增强：

- 指标叠加。
- 成交量。
- 买卖点标记。
- 策略信号点。
- 回测订单点位。
- 数据缺口提示。
- 时间范围切换。

第一版验收：

- 能打开研究页。
- 能选择已有 K 线数据。
- 能加载 K 线图表。
- 能显示当前数据源基础信息。
- 能看到数据同步列表。
- 能创建数据同步任务。
- 能对同步任务执行实时 / 停止实时、同步 / 停止同步、查看图表、删除。
- 图表区域是页面主体。

## 7. 数据同步能力

术语解释：

- “同步”表示补齐或推进历史 / 当前 K 线数据。
- “实时”表示持续监听或持续轮询最新 K 线，让数据保持更新。
- “基础周期”表示直接从交易所同步并持久化的原始 K 线周期，第一版可以先以 `1m` 为基础周期。
- “聚合周期”表示系统内部根据基础周期生成的更高周期，例如 `5m`、`15m`、`1h`、`4h`、`1d`。
- 二者可以在交互上区分，避免用户不知道当前是在补数据还是在保持实时。

基础周期和聚合周期：

- 如果当前只同步 `1m` K 线，`1m` 是系统的 canonical market data。
- 更高周期 K 线必须由系统内部聚合层从 `1m` 生成。
- 图表选择 `5m`、`15m`、`1h` 等周期时，不要求交易所同步任务分别拉取这些周期。
- 回测、模拟盘、实盘策略使用更高周期时，也必须走同一套聚合层。
- 聚合层必须被后端统一封装，不能让前端自行聚合 K 线。
- 聚合层必须能判断聚合 K 线是否闭合。
- 未闭合的聚合 K 线可以用于图表展示，但不能被当成闭合周期信号。
- 聚合边界必须统一使用 UTC 时间，不跟随浏览器时区漂移。

周期数据选择规则：

- 请求某个周期 K 线时，统一通过后端 `CandleProvider` 或等价查询服务获取。
- 如果数据库中存在该周期的健康原始 K 线，可以直接返回该周期数据。
- 如果没有合适的同周期 K 线，系统必须尝试用已同步的更小周期 K 线内部聚合。
- 第一版默认用 `1m` 聚合更高周期。
- 只能从更小周期聚合到更大周期，不能从更大周期反推出更小周期。
- 如果缺少足够的更小周期数据，必须返回数据不足或存在缺口，不能伪造 K 线。
- 返回结果需要标明数据来源，例如 `native` 或 `aggregated`，以及聚合使用的基础周期。
- 图表、回测、模拟盘 / 实盘 runner 都必须走同一套周期数据选择规则。

聚合规则：

- open 取聚合窗口第一根基础 K 线的 open。
- high 取窗口内 high 最大值。
- low 取窗口内 low 最小值。
- close 取窗口内最后一根基础 K 线的 close。
- volume 取窗口内 volume 汇总。
- open_time 为聚合窗口开始时间。
- close_time 为聚合窗口结束时间。
- 只有窗口内基础 K 线完整且最后一根基础 K 线已闭合时，聚合 K 线才算闭合。

聚合数据的存储原则：

- `market_candles` 优先保存交易所同步回来的基础周期原始事实。
- 第一版不要求把聚合 K 线持久化成另一份事实表。
- 如果后续为了性能缓存聚合 K 线，必须明确标记为 derived cache。
- derived cache 可以删除重建，不能成为唯一事实源。
- 聚合缓存失效必须由基础 K 线变化驱动。

数据同步必须支持恢复：

- 系统重启后能根据任务状态继续同步。
- 实时同步任务在系统重启后必须立即恢复。
- `hi sync` 启动后必须扫描所有 `realtime_enabled = true` 且未删除的同步任务。
- 对于实时同步任务，必须根据已持久化的最新 K 线时间继续拉取或订阅。
- 不能因为 daemon 重启就丢失进度。
- 同步进度必须持久化。
- 已恢复的实时同步任务必须更新心跳或运行状态，供研究页显示。

数据同步必须幂等：

- K 线唯一键为 `exchange + symbol + interval + open_time`。
- 写入 K 线必须使用 upsert。
- 同一根 K 线重复拉取、重复写入不能产生重复数据。
- 数据同步任务窗口和补同步窗口的 `start_time` / `end_time` 必须按对应周期的 UTC 边界对齐，不能接受秒级偏移或非周期边界。
- 未闭合 K 线允许更新。
- 已闭合 K 线原则上不可随意改写；如果交易所返回修正数据，必须记录更新时间。
- 不能用前端状态判断同步进度。
- 不能只依赖内存变量保存游标。

恢复规则：

- 同步任务保存 `last_synced_open_time` 或等价游标。
- `hi sync` 启动时先领取任务 lease，再读取游标恢复。
- 实时任务恢复时应从最后已保存 K 线向前重叠一个小窗口重新拉取，靠 upsert 去重。
- 重叠窗口用于修复重启、网络抖动、交易所延迟导致的尾部缺口。
- 每次批量写入成功后再推进游标。
- 拉取成功但写库失败时，不能推进游标。
- 写库成功但心跳失败时，下一轮恢复仍必须安全。
- 停止实时只改变期望状态，不删除历史 K 线。

缺口处理：

- `market_candles` 必须能被检查是否存在时间缺口。
- 缺口扫描和补同步窗口只能使用按周期对齐的 `open_time` 作为连续性边界；错位 `open_time` 属于异常 K 线，必须通过 invalid issue 可观察，不能被当成正常缺口端点。
- 错位 `open_time` 不能用普通补同步伪装成已修复；研究页必须提供明确隔离动作，后端先归档原始 K 线，再从 active `market_candles` 集合移除。
- 研究页需要展示数据健康状态，例如正常、同步中、有缺口、失败、暂停。
- 缺口修复仍走同一套同步任务逻辑，不单独写一套数据修复通道。
- 删除同步任务不等于删除 K 线数据，是否删除数据需要单独确认动作。

第一版验收：

- 能创建数据同步任务。
- 能在研究页看到同步任务列表。
- 能启动 / 停止同步。
- 能启动 / 停止实时。
- 能从列表加载对应图表。
- 能删除同步任务。
- 系统重启后，已开启实时的数据同步任务能自动恢复。
- 恢复后不会重复写乱 K 线。
- 恢复后能从最后同步位置继续。

## 8. 策略沉淀

策略是后端代码资产。

策略代码写在后端目录中。

前端不提供在线写策略代码能力。

前端负责：

- 展示后端已注册策略。
- 展示策略名称。
- 展示策略说明。
- 展示适用市场或周期信息。
- 展示策略参数。
- 在创建回测或交易任务时选择策略并填写参数。

策略参数必须结构化：

- 每个策略声明自己的参数 schema。
- 前端根据 schema 渲染表单。
- 创建任务时保存参数快照。

策略输出是意图：

- 订单意图。
- 通知意图。

策略不能直接下单。

策略不能直接发送通知。

策略不能直接写数据库。

## 9. 回测页

回测页主要是回测任务列表。

回测页必须支持创建新回测任务。

回测列表每一项展示：

- 回测名称。
- 交易所。
- 交易对。
- 数据周期。
- 时间范围。
- 策略。
- 策略参数摘要。
- 回测状态。
- 关键结果摘要。
- 创建时间。

每一项可以进入回测详情页。

回测创建表单包含：

- 交易所。
- 交易对。
- K 线周期。
- 时间范围。
- 策略。
- 策略参数。
- 初始资金。
- 手续费配置。
- 滑点配置。
- 触发方式。

触发方式：

- 闭合周期 K 线触发。
- 更小数据单位模拟盘中触发。

当前没有 tick 级数据，现阶段最小数据单位按分钟级。

回测周期语义：

- 回测选择的 K 线周期可以是 `1m` 基础周期，也可以是内部聚合周期。
- 回测读取 K 线必须通过统一周期数据选择规则。
- 当策略周期大于 `1m` 时，策略闭合周期信号由聚合层产生。
- 更小数据单位模拟盘中触发时，当前使用 `1m` 基础 K 线作为最小推进单位。
- 不能为了某个回测周期临时绕过聚合层直接读取另一套未定义数据源。
- 回测详情图表展示的周期必须和回测任务周期一致，买卖点按对应时间映射到图表。

回测详情页包含：

- K 线图表。
- 买卖点展示。
- 回测概要。
- 收益信息。
- 交易订单。
- 策略意图。
- 参数快照。
- 运行日志或错误。

回测设计目标：

- 清爽。
- 能回到图表上看买卖点。
- 不只是输出一张统计表。
- 避免只有粗糙周期收盘触发。

第一版验收：

- 能创建回测任务。
- 能查看回测列表。
- 能进入回测详情。
- 回测详情有 K 线图表。
- 买卖点能叠加到图表或以明确方式关联图表。
- 能看到订单和关键回测信息。

## 10. 交易页

交易页整体和回测页类似。

交易任务有两个类型：

- 模拟。
- 实盘。

交易页主要是交易任务列表。

交易页必须支持创建新交易任务。

交易列表每一项展示：

- 任务名称。
- 类型：模拟 / 实盘。
- 交易所。
- 账号。
- 交易对。
- K 线周期。
- 策略。
- 策略参数摘要。
- 运行状态。
- 当前持仓。
- 最近订单。
- 最近通知。
- 最近策略意图。
- 创建时间。

每一项可以进入交易详情页。

交易创建表单包含：

- 类型：模拟 / 实盘。
- 交易所。
- 账号。
- 交易对。
- 策略。
- 策略参数。
- 资金配置。
- 风险配置。
- 意图处理配置。

意图处理配置：

- 订单意图自动执行。
- 订单意图只通知。
- 通知意图发送到指定通道。

交易详情页包含：

- K 线图表。
- 策略意图标记。
- 订单标记。
- 当前持仓。
- 订单列表。
- 成交列表。
- 通知列表。
- 运行状态。
- 任务日志。

实盘任务必须比模拟任务更谨慎：

- 创建时必须明确账号。
- 创建时必须明确实盘类型。
- 危险操作需要确认。
- 创建实盘任务时必须展示交易所、账号别名、交易对、策略、资金配置、风险配置和意图处理配置。
- 实盘任务创建确认文案必须明确这是实盘。
- 实盘任务默认不允许静默自动下单，必须在任务配置里明确选择订单意图处理方式。
- live executor 必须和 paper executor 明确分离。
- 实盘下单只能通过 live executor 触发，策略不能直接触达交易所 adapter。
- 实盘订单必须先落库，再提交交易所。
- 交易所返回结果必须回写订单状态和原始响应摘要。
- 订单提交失败必须记录失败原因，不能静默重试成重复下单。
- 同一订单意图必须有幂等键，避免 worker 重启后重复下单。
- 实盘 API key 必须加密保存。
- 前端和日志不能展示完整密钥。
- 实盘账号必须能禁用；禁用后不能创建新的实盘任务，也不能继续自动提交新订单。
- 第一版不做复杂审批流，但必须保留清晰的实盘确认和执行边界。

第一版验收：

- 能创建模拟任务。
- 能创建实盘任务。
- 能查看交易任务列表。
- 能进入交易详情。
- 能看到任务运行状态、意图、订单或通知。

## 11. 策略意图

策略运行结果统一称为意图。

意图类型至少包含：

- 订单意图。
- 通知意图。

订单意图表达：

- 买入 / 卖出。
- 标的。
- 数量。
- 价格条件。
- 下单类型。
- 原因。
- 策略上下文。

通知意图表达：

- 通知标题。
- 通知内容。
- 级别。
- 目标通道。
- 策略上下文。
- 是否需要人工核对。

意图处理原则：

- 策略只产生意图。
- 运行器根据任务配置处理意图。
- 订单意图可以自动执行，也可以只通知。
- 通知意图进入通知路由。
- 所有意图必须被记录。

## 12. 系统管理菜单

系统管理是顶部右侧菜单，不是一级主导航。

第一版菜单项：

- 通知管理。
- 交易所账号管理。
- 操作台账号管理。
- 运维健康。

### 12.1 通知管理

通知管理负责：

- 邮件配置。
- Telegram 配置。
- 飞书配置。
- 通知通道启用 / 停用。
- 通知记录。
- 通知失败原因。

通知通道第一版采用 env-reference 凭据模型：通道 `target` 只能保存非敏感路由配置和环境变量名，真实 token / webhook secret / SMTP password 从 notify worker 环境变量读取。

阶段 5 demo 已启用：

- `local`：只在本地 outbox 中记录投递成功，不访问外部网络。
- `webhook-demo`：webhook-like 演示 provider，只记录目标和模拟投递结果，不访问真实 webhook。
- `webhook`：向 `target` 指定的 HTTP / HTTPS URL 发送 JSON payload。
- `telegram`：`telegram://send?chat_id=<chat-id>&token_env=TELEGRAM_BOT_TOKEN`。
- `feishu`：`feishu://webhook?url_env=FEISHU_WEBHOOK_URL`。
- `email`：`smtp://smtp.example.com:587?from=bot@example.com&to=ops@example.com&username_env=SMTP_USERNAME&password_env=SMTP_PASSWORD`。

真实邮件、Telegram、飞书 provider 已接入基础发送路径；生产级模板、限流、回执、密钥轮换、审计签名和通道更新 / 删除仍未完成。

### 12.2 交易所账号管理

交易所账号管理负责：

- 交易所账号列表。
- 新增账号。
- 编辑账号别名。
- 启用 / 停用账号。
- 查看账号状态。
- 配置交易所账号凭据。
- 作为模拟盘 / 实盘任务选择账号的数据来源。

本地开发版交易所账号密钥使用 `ENCRYPTION_KEY` + AES-GCM 加密保存；生产级 KMS / secret manager、密钥轮换和历史账号迁移策略后续阶段确认。

### 12.3 操作台账号管理

操作台账号管理负责：

- 登录账号列表。
- 新增 / 禁用账号。
- 修改密码。

第一版只要求基础登录能力，不扩展复杂权限系统。

### 12.4 运维健康

运维健康负责：

- 系统运行状态。
- 数据同步 worker 状态。
- 交易任务 runner 状态。
- 数据库连接状态。
- 最近错误。

不做重型监控平台。

## 13. 登录能力

操作台需要基本登录能力。

第一版目标：

- 登录页。
- 退出按钮。
- 当前操作台账号展示。
- 会话保持。
- 保护操作台接口。

不在第一版扩展：

- 多租户。
- 复杂角色权限。
- 审批流。
- MFA。

CSRF 写请求保护和登录失败节流已进入本地 demo 边界；密码策略、MFA、会话审计、完整 session 管理仍后续确认。

## 14. 后端模块边界

后端应围绕产品能力收敛，不做过度平台化。

数据库使用 PostgreSQL。

后端运行形态采用同一 Go 项目的单二进制多子命令模式：

```text
hi api       操作台 API + 前端服务
hi sync      数据同步 worker
hi trading   模拟盘 / 实盘任务 runner
hi backtest  回测 worker
hi notify    通知 outbox worker
hi migrate   数据库迁移
```

这些子命令来自同一个 `hi` 二进制。

这不是微服务拆分。它们共享同一代码仓库、同一核心模块、同一 PostgreSQL。

拆分目标：

- 数据同步长任务不拖垮操作台 API。
- 回测计算不影响实盘 / 模拟盘任务。
- 模拟盘 / 实盘 runner 可以独立重启。
- 每类后台能力可以单独部署、单独观察、单独限流。
- 数据库迁移通过 `hi migrate` 明确执行，不依赖手工 SQL。
- 保持小而美，不引入消息队列和复杂服务治理。

进程协作：

```text
Vue 操作台
  -> hi api
      -> PostgreSQL

hi sync
  -> PostgreSQL
  -> Binance / OKX K线接口

hi trading
  -> PostgreSQL
  -> Binance / OKX 交易接口

hi notify
  -> PostgreSQL
  -> 通知 Provider

hi backtest
  -> PostgreSQL

hi migrate
  -> PostgreSQL migrations
```

协作原则：

- PostgreSQL 是唯一协调中心。
- `hi api` 只创建任务、更新期望状态、读取结果、服务前端静态资源，不运行长任务。
- `hi sync` 领取数据同步任务并更新同步进度。
- `hi trading` 领取模拟盘 / 实盘任务并运行策略。
- `hi backtest` 领取回测任务并写入结果。
- `hi notify` 领取 notification outbox 并执行通知投递。
- `hi migrate` 执行数据库迁移。
- 不使用消息队列。
- 不使用复杂服务注册。
- 每个 worker 进程必须支持优雅停止、任务锁、心跳、超时释放、重启恢复。

### 14.1 统一 worker lease

`hi sync`、`hi backtest`、`hi trading`、`hi notify` 都必须使用统一 worker lease 模型领取任务。

目标：

- 避免同一任务被多个 worker 重复执行。
- 避免 worker 崩溃后任务永久卡住。
- 支持多进程部署。
- 支持超时释放和重启恢复。

核心字段建议：

```text
status
locked_by
locked_until
heartbeat_at
started_at
finished_at
last_error
attempt_count
```

领取任务建议使用 PostgreSQL 行锁：

```text
FOR UPDATE SKIP LOCKED
```

每个 worker 必须：

- 定期 heartbeat。
- 支持 context cancel。
- 支持优雅停止。
- 捕获 panic 并记录错误。
- 超时后释放任务或标记可重试。

状态机必须明确，不允许每个 worker 自己发明状态。

统一状态建议：

```text
pending
running
stopping
paused
succeeded
failed
cancelled
```

状态含义：

- `pending`：等待 worker 领取。
- `running`：worker 已领取并执行中。
- `stopping`：用户或系统要求停止，worker 应尽快优雅退出。
- `paused`：任务已停止执行，但保留进度，可继续。
- `succeeded`：一次性任务完成，例如回测。
- `failed`：任务失败且暂不继续执行。
- `cancelled`：任务被用户取消，不再自动恢复。

不同任务类型的使用边界：

- 数据实时同步任务通常在 `running` 和 `paused` 间切换。
- 历史补齐同步可以从 `pending` 到 `running` 再到 `succeeded` 或 `failed`。
- 回测是一次性任务，结束后进入 `succeeded` 或 `failed`。
- 模拟盘 / 实盘任务通常在 `running`、`stopping`、`paused`、`failed` 间切换。

领取任务必须短事务完成：

```text
BEGIN
  SELECT ... FOR UPDATE SKIP LOCKED
  UPDATE status = running, locked_by = worker_id, locked_until = now + lease_ttl
COMMIT
```

禁止：

- 长事务包住整个同步、回测或交易运行过程。
- worker 持有数据库事务时访问交易所接口。
- worker 只在内存里标记自己拥有任务。
- 没有 `locked_until` 的永久锁。

heartbeat 规则：

- worker 必须周期性刷新 `heartbeat_at` 和 `locked_until`。
- heartbeat 失败达到阈值后，worker 应停止该任务的外部副作用。
- 其他 worker 只能领取 `locked_until < now()` 或明确 `pending` 的任务。
- 任务超时释放必须保留 `last_error`、`attempt_count` 和最近心跳。

停止规则：

- 用户点击停止时，API 只更新期望状态，不直接杀进程。
- worker 观察到 `stopping` 后执行收尾，保存进度，再进入 `paused` 或 `cancelled`。
- 容器收到退出信号时，worker 必须尝试收尾并停止 heartbeat。
- 如果进程直接崩溃，下一次由 `locked_until` 超时释放恢复。

### 14.2 通知 outbox

通知必须采用 outbox 思路。

策略产生 `NotificationIntent` 后：

```text
NotificationIntent
  -> notifications / notification_outbox 落库
  -> 通知投递逻辑发送
  -> 记录成功 / 失败 / 重试
```

要求：

- 通知先落库，再投递。
- 投递失败可重试。
- 通知记录可追溯到 intent 和 task。
- 不允许策略直接调用邮件、Telegram、飞书 provider。

### 14.3 StrategyRunner 统一

回测、模拟盘、实盘必须共享策略运行核心。

建议结构：

```text
StrategyRunner
  -> 读取 market input
  -> 调用 Strategy
  -> 产出 Intent
  -> 交给 Executor
```

不同运行模式只替换 executor：

```text
BacktestExecutor
PaperExecutor
LiveExecutor
```

禁止：

- 回测一套策略运行逻辑。
- 交易一套策略运行逻辑。
- 策略在不同运行模式下绕过 intent 模型。

后端依赖原则：

- 必要第三方库可以使用，例如 PostgreSQL 驱动、HTTP router、密码哈希、JWT / session、配置解析、日志等。
- 不为了炫技引入大型框架。
- 不引入过多间接层。
- 不引入不必要的异步、消息队列、插件系统、规则引擎。
- 安全敏感代码优先使用成熟库，不手写密码学。
- 交易核心逻辑优先用清晰 Go 代码实现，保持可读、可审计。

建议核心模块：

- 数据同步。
- K 线查询。
- 策略注册。
- 回测运行。
- 交易任务运行。
- 意图处理。
- 订单执行。
- 通知投递。
- 账号管理。
- 登录会话。

后端核心原则：

- 策略不直接下单。
- 策略不直接发通知。
- 策略不直接写库。
- 数据同步进度必须持久化。
- 回测详情必须能关联 K 线图表。
- 交易任务必须能关联账号、标的、策略和参数快照。
- 意图、订单、通知必须可追溯到任务。

### 14.4 交易所 adapter

交易所能力必须通过 adapter 边界接入。

当前必须支持：

- Binance。
- OKX。

设计要求：

- 核心交易、回测、策略、意图模型不能依赖具体交易所实现。
- 交易所 adapter 负责交易所 API 协议、认证签名、symbol 映射、K 线拉取、订单提交和订单查询。
- 新增交易所时，应新增 adapter，不应重写核心链路。
- 当前只实现 Binance 和 OKX，不为了未来交易所提前堆复杂能力矩阵。

### 14.5 Docker 运行形态

第一版必须补齐 Docker 运行能力。

Docker 的定位：

- 用于本地开发启动。
- 用于最小生产部署。
- 用于稳定复现 `hi api`、`hi sync`、`hi trading`、`hi backtest` 的运行形态。
- 不把项目拆成多个代码仓库。
- 不引入 Kubernetes、服务网格或复杂发布系统。

镜像原则：

- 构建一个 `hi` 后端镜像。
- 同一镜像内包含同一个 `hi` 二进制。
- 通过不同 command 启动不同子命令。
- 前端构建产物打入镜像，由 `hi api` 服务。
- 镜像内不保存运行状态。
- 运行状态只进入 PostgreSQL 或外部明确配置的持久化位置。

建议服务：

```text
postgres
hi-migrate
hi-api
hi-sync
hi-trading
hi-backtest
```

服务职责：

- `postgres`：唯一数据库和协调中心。
- `hi-migrate`：一次性执行 `hi migrate`。
- `hi-api`：执行 `hi api`，服务操作台 API 和前端静态资源。
- `hi-sync`：执行 `hi sync`，负责数据同步 worker。
- `hi-trading`：执行 `hi trading`，负责模拟盘 / 实盘 runner。
- `hi-backtest`：执行 `hi backtest`，负责回测 worker。

`hi-backtest` 可以在第一版按需启停，但 compose 文件中应该保留服务定义。

Docker Compose 原则：

- `postgres` 必须配置持久化 volume。
- `postgres` 必须配置 healthcheck。
- `hi-migrate` 必须等待 PostgreSQL 可用后再运行。
- `hi-api`、`hi-sync`、`hi-trading`、`hi-backtest` 必须在迁移完成后启动。
- 如果 Compose 版本无法严格表达迁移依赖，应用启动时必须自行等待数据库和 schema 就绪。
- 所有服务通过环境变量读取配置，不把密钥写死进镜像。
- 后端服务必须支持优雅退出，容器停止时能释放或超时释放任务 lease。

配置原则：

```text
DATABASE_URL
APP_SECRET
ENCRYPTION_KEY
HTTP_ADDR
LOG_LEVEL
```

交易所密钥、通知通道密钥等敏感配置不能写入 Dockerfile。

部署边界：

- 第一版用 Docker Compose 已足够。
- 不引入 Redis。
- 不引入 Kafka。
- 不引入独立任务队列。
- 不引入 Kubernetes。
- 不把 Docker 部署做成重运维平台。

最小目录建议：

```text
Dockerfile
docker-compose.yml
docker-compose.override.yml
.dockerignore
```

`docker-compose.yml` 表达稳定运行形态。

`docker-compose.override.yml` 只服务本地开发，例如端口映射、热加载或额外开发配置。

## 15. 前端路由建议

路由建议：

```text
/login
/overview
/research
/backtests
/backtests/new
/backtests/:id
/trading
/trading/new
/trading/:id
/system/notifications
/system/exchange-accounts
/system/operators
/system/health
```

默认进入：

```text
/overview
```

从研究页数据列表“查看图表”切换数据源时，可以更新 URL 查询参数：

```text
/research?exchange=binance&symbol=BTCUSDT&interval=1m
```

从回测详情展示图表时，可以直接在详情页嵌入图表，不必跳转研究页。

## 16. 前端组件建议

核心组件：

- `TopNav`
- `TopActions`
- `SystemMenu`
- `ThemeToggle`
- `LocaleSwitch`
- `AccountButton`
- `AppShell`
- `StatusBadge`
- `ConfirmAction`
- `EmptyState`
- `LoadingState`
- `ErrorState`
- `TradingViewChart`
- `DataSourceSelector`
- `DataSyncTaskTable`
- `BacktestTaskTable`
- `BacktestDetail`
- `TradingTaskTable`
- `TradingTaskDetail`
- `StrategyParamForm`
- `IntentList`
- `OrderList`
- `NotificationList`

组件原则：

- 图表组件要稳定封装，不让 TradingView 接入细节散落各页。
- 策略参数表单由策略 schema 驱动。
- 列表操作按钮保持明确，不隐藏在复杂菜单里。
- 系统管理二级菜单可以折叠，但核心导航不折叠成侧栏。

## 17. 数据模型草案

以下是概念模型，不是最终 SQL。

### 17.1 数据同步

```text
data_sync_tasks
  id
  exchange
  symbol
  interval
  start_time
  end_time
  sync_enabled
  realtime_enabled
  status
  locked_by
  locked_until
  heartbeat_at
  started_at
  finished_at
  last_synced_open_time
  last_error
  attempt_count
  created_at
  updated_at
```

```text
market_candles
  exchange
  symbol
  interval
  open_time
  close_time
  open
  high
  low
  close
  volume
  is_closed
  updated_at
```

`market_candles` 必须有唯一约束：

```text
exchange + symbol + interval + open_time
```

第一版如果只同步 `1m`，`market_candles.interval` 主要保存 `1m` 原始 K 线。

更高周期由聚合查询层生成，不要求进入 `market_candles`。

如果后续新增聚合缓存，必须与 `market_candles` 区分，不能混淆原始事实和派生结果。

### 17.2 策略

```text
strategy_registry
  strategy_id
  name
  description
  param_schema
  supported_intents
```

策略 registry 可以来自后端代码，不一定必须落库。任务需要保存策略 id 和参数快照。

### 17.3 回测

```text
backtest_tasks
  id
  name
  exchange
  symbol
  interval
  time_range
  strategy_id
  strategy_params
  status
  locked_by
  locked_until
  heartbeat_at
  started_at
  finished_at
  last_error
  attempt_count
  result_summary
  created_at
```

```text
backtest_orders
  id
  backtest_id
  intent_id
  side
  price
  quantity
  status
  occurred_at
```

### 17.4 交易任务

```text
trading_tasks
  id
  name
  type
  exchange
  account_id
  symbol
  interval
  strategy_id
  strategy_params
  intent_policy
  status
  locked_by
  locked_until
  heartbeat_at
  started_at
  finished_at
  last_error
  attempt_count
  created_at
```

`type` 为：

- `paper`
- `live`

### 17.5 意图

```text
strategy_intents
  id
  task_id
  task_type
  strategy_id
  intent_type
  idempotency_key
  payload
  policy
  status
  created_at
```

`intent_type` 为：

- `order`
- `notification`

同一任务内的 `idempotency_key` 必须唯一。

### 17.6 订单

```text
orders
  id
  task_id
  task_type
  intent_id
  idempotency_key
  exchange
  account_id
  symbol
  side
  order_type
  price
  quantity
  status
  exchange_order_id
  exchange_response_summary
  last_error
  created_at
  updated_at
```

同一任务内的订单 `idempotency_key` 必须唯一。

实盘订单必须先创建本地订单记录，再提交交易所。

### 17.7 交易所账号

```text
exchange_accounts
  id
  exchange
  alias
  encrypted_api_key
  encrypted_api_secret
  enabled
  created_at
  updated_at
```

要求：

- 密钥只保存密文。
- 列表和详情不返回完整密钥。
- `enabled = false` 时不能用于新的实盘任务和新的实盘订单提交。

### 17.6 通知

```text
notifications
  id
  intent_id
  channel
  title
  body
  status
  error
  created_at
  sent_at
```

## 18. 接口草案

接口命名先保持简单。

数据：

```text
GET    /api/data/tasks
POST   /api/data/tasks
POST   /api/data/tasks/:id/sync/start
POST   /api/data/tasks/:id/sync/stop
POST   /api/data/tasks/:id/realtime/start
POST   /api/data/tasks/:id/realtime/stop
DELETE /api/data/tasks/:id
GET    /api/candles
```

策略：

```text
GET /api/strategies
GET /api/strategies/:id
```

回测：

```text
GET  /api/backtests
POST /api/backtests
GET  /api/backtests/:id
GET  /api/backtests/:id/orders
GET  /api/backtests/:id/intents
```

交易：

```text
GET  /api/trading/tasks
POST /api/trading/tasks
GET  /api/trading/tasks/:id
POST /api/trading/tasks/:id/start
POST /api/trading/tasks/:id/pause
POST /api/trading/tasks/:id/stop
GET  /api/trading/tasks/:id/orders
GET  /api/trading/tasks/:id/intents
GET  /api/trading/tasks/:id/notifications
```

系统管理：

```text
GET  /api/system/notifications
POST /api/system/notifications/:id/retry
GET  /api/system/notifications/channels
POST /api/system/notifications/channels
GET  /api/system/exchange-accounts
POST /api/system/exchange-accounts
GET  /api/system/operators
POST /api/system/operators
GET  /api/system/health
```

登录：

```text
POST /api/auth/login
POST /api/auth/logout
GET  /api/auth/session
```

## 19. 实施阶段

### 阶段 1：前端壳和导航

目标：

- 建立顶部平铺导航。
- 建立右侧工具区。
- 建立系统管理二级菜单。
- 建立页面路由。
- 形成研究工作台气质。

验收：

- 顶部导航包含概览、研究、回测、交易。
- 右侧工具区顺序正确。
- 系统管理菜单包含通知管理、交易所账号管理、操作台账号管理、运维健康。
- 不出现左侧管理后台导航。

### 阶段 2：研究页、图表骨架和数据同步列表

目标：

- 接入 TradingView 开源图表。
- 封装图表组件。
- 支持选择数据源并加载 K 线。
- 在研究页展示轻量数据同步列表。
- 支持创建数据同步任务。

验收：

- 研究页图表占主体。
- 能选择交易所、交易对、周期。
- 能加载 K 线数据。
- 能看到数据同步任务列表。

### 阶段 3：数据同步操作

目标：

- 实现同步 / 停止同步。
- 实现实时 / 停止实时。
- 实现查看图表。
- 实现删除。

验收：

- 研究页数据同步列表操作完整。
- 查看图表能在研究页加载对应数据上下文。
- 同步进度可恢复。

### 阶段 4：策略 registry 和参数表单

目标：

- 后端注册策略。
- 前端展示策略。
- 根据策略参数 schema 渲染表单。

验收：

- 前端能选择策略。
- 能看到策略参数。
- 创建回测和交易任务时能填写参数。

### 阶段 5：回测列表和详情

目标：

- 实现回测创建。
- 实现回测列表。
- 实现回测详情。
- 回测详情包含图表和买卖点。

验收：

- 回测详情能看到 K 线图表。
- 能看到买卖点。
- 能看到订单和回测信息。

### 阶段 6：交易任务

目标：

- 实现交易任务列表。
- 实现模拟 / 实盘任务创建。
- 实现交易详情。
- 实现意图、订单、通知展示。

验收：

- 能创建模拟任务。
- 能创建实盘任务。
- 交易详情能看到策略意图、订单、通知。

### 阶段 7：系统管理和登录

目标：

- 实现基础登录。
- 实现退出。
- 实现当前账号展示。
- 实现通知管理。
- 实现交易所账号管理。
- 实现操作台账号管理。
- 实现运维健康。

验收：

- 未登录不能进入操作台。
- 登录后能看到顶部账号。
- 系统管理菜单各项可进入。

## 20. 明确禁止

禁止：

- 左侧管理后台导航。
- 把通知做成和模拟盘 / 实盘并列的任务类型。
- 把“回测后才能模拟 / 实盘”写成产品限制。
- 把数据同步拆成一个占据大量空间的独立管理后台页。
- 把策略写成前端动态代码。
- 让策略直接访问交易所。
- 让策略直接发通知。
- 让策略直接写数据库。
- 图表页被表单和表格挤到边角。
- 数据页变成复杂数据治理中心。
- 系统管理能力抢占核心一级导航。

## 21. 工程质量约束

本项目必须始终保持最佳工程化实践。

质量目标：

- 代码清晰。
- 边界明确。
- 文件短小。
- 函数短小。
- 命名准确但不过度冗长。
- 依赖克制。
- 测试覆盖关键业务。
- 变更可审查。

### 21.1 总原则

必须遵守：

- 先明确边界，再写实现。
- 先修根因，不做补丁式堆代码。
- 新增代码必须有明确归属模块。
- 不把无关职责塞进同一个文件。
- 不把复杂逻辑塞进 handler、Vue 页面或单个 service 大函数。
- 不为未来可能性提前引入复杂抽象。
- 不复制粘贴大段逻辑。
- 不把测试、mock、demo 数据混进生产路径。
- 不把技术债伪装成“暂时先这样”长期保留。

### 21.2 文件行数限制

默认限制：

```text
Go 生产文件：建议 <= 300 行，硬上限 500 行
Go 测试文件：建议 <= 400 行，硬上限 700 行
Vue 页面文件：建议 <= 300 行，硬上限 450 行
Vue 组件文件：建议 <= 250 行，硬上限 400 行
TypeScript 生产文件：建议 <= 250 行，硬上限 400 行
TypeScript 测试文件：建议 <= 400 行，硬上限 650 行
CSS / SCSS 单文件：建议 <= 300 行，硬上限 500 行
```

超过建议值时必须优先拆分。

超过硬上限时不能继续堆功能，必须先拆分。

允许例外：

- 自动生成文件。
- 明确的协议映射表。
- 少量不可拆的静态 fixture。

例外必须在文档或代码注释中说明原因和退出计划。

### 21.3 函数限制

函数长度：

```text
普通函数：建议 <= 40 行，硬上限 80 行
复杂业务函数：建议 <= 60 行，硬上限 120 行
测试函数：建议 <= 80 行，硬上限 150 行
```

函数参数：

```text
普通函数参数 <= 4 个
超过 4 个参数时优先引入 request / options / config struct
超过 6 个参数默认禁止
```

函数返回值：

```text
Go 函数返回值建议 <= 2 个
超过 2 个返回值必须有明确理由
```

禁止：

- 一个函数同时做校验、查询、业务决策、写库、发通知。
- handler 中直接堆业务流程。
- Vue setup 中塞大量业务逻辑。
- 一个函数靠多层 if / switch 支撑多个模块语义。

### 21.4 命名限制

命名要准确、稳定、不过度缩写。

函数名：

```text
建议 <= 32 字符
硬上限 48 字符
测试函数可放宽到 80 字符
```

文件名：

```text
建议 <= 40 字符
硬上限 64 字符
```

禁止：

- 为了表达所有上下文写超长函数名。
- 使用含糊名字，例如 `Handle`, `Do`, `Process`, `Manager`，除非上下文极清楚。
- 同一个概念出现多套名字，例如 task / job / run 混用但不定义区别。

必须统一的术语：

- `data sync task`：数据同步任务。
- `backtest`：回测。
- `trading task`：模拟 / 实盘交易任务。
- `paper`：模拟盘。
- `live`：实盘。
- `strategy`：后端策略代码。
- `intent`：策略输出意图。
- `order intent`：订单意图。
- `notification intent`：通知意图。

### 21.5 前端质量约束

前端必须：

- 使用成熟 UI 库承担基础组件。
- 使用 Naive UI 作为基础 UI 库。
- 使用 Pinia 管理跨页面状态。
- 使用 vue-i18n 管理中英文文案。
- 使用 lightweight-charts 封装 TradingView K 线图表。
- 使用 pnpm 管理前端依赖。
- 使用原生 fetch + typed API wrapper 调用后端。
- 页面组件只负责布局和页面编排。
- 业务状态放入 composable 或明确的 store。
- API 调用集中封装。
- 表单校验结构化。
- i18n 文案集中管理。
- 主题 token 集中管理。
- 图表组件单独封装。

禁止：

- 在 Vue 页面里直接散落大量 fetch。
- 在模板里写复杂业务表达式。
- 用大量手写 CSS 覆盖 UI 库基础能力。
- 复制粘贴表格列、状态标签、按钮组。
- 中文文案硬编码在多个组件里。
- 只适配浅色主题或只适配中文。

前端交付必须检查：

- 浅色主题。
- 深色主题。
- 中文。
- 英文。
- 桌面宽屏。
- 普通笔记本宽度。
- 窄屏。
- 按钮 hover / active / disabled。
- 空状态。
- 加载状态。
- 错误状态。

前端基础设施骨架必须先建立，再堆页面。

建议目录：

```text
web/frontend/src
  app/
  assets/
  components/
  components/chart/
  components/layout/
  components/tables/
  composables/
  i18n/
  pages/
  router/
  services/api/
  stores/
  styles/
  theme/
  types/
```

目录职责：

- `app/`：应用入口装配，例如 Naive UI provider、Pinia、router、i18n、主题。
- `components/layout/`：顶部导航、系统菜单、账号按钮、主题切换、语言切换。
- `components/chart/`：TradingView lightweight-charts 封装。
- `components/tables/`：可复用业务表格和状态列。
- `composables/`：页面可复用业务状态和交互逻辑。
- `i18n/`：中文、英文文案和类型约束。
- `pages/`：页面级组件，只做布局和编排。
- `router/`：路由定义和登录守卫。
- `services/api/`：typed API wrapper、错误映射、请求拦截。
- `stores/`：登录态、主题、语言、当前账号等跨页面状态。
- `styles/`：基础样式、布局变量、全局修正。
- `theme/`：浅色 / 深色主题 token、Naive UI theme overrides、图表主题映射。
- `types/`：前端共享类型。

必须先完成的基础设施：

- `AppShell`：顶部导航和右侧工具区。
- `router`：核心路由和登录守卫。
- `auth store`：登录态、当前操作台账号、退出。
- `theme store`：浅色 / 深色切换和持久化。
- `locale store`：中文 / 英文切换和持久化。
- `api client`：统一 fetch、错误处理、认证处理、JSON 编解码。
- `TradingViewChart`：统一图表组件，支持主题切换和数据更新。
- `StatusBadge`：统一任务状态展示。
- `ConfirmAction`：危险动作确认。
- `EmptyState` / `LoadingState` / `ErrorState`：统一基础状态。

前端实现顺序：

1. 建立应用壳、主题、语言、路由、API client。
2. 建立图表封装和基础状态组件。
3. 建立研究页骨架。
4. 接入数据同步列表和图表数据。
5. 再实现回测、交易和系统管理页面。

禁止：

- 先写一堆页面，再回头补主题和 i18n。
- 每个页面各自维护一套 loading / error / empty。
- 每个页面各自拼接口地址。
- 每个页面各自写状态颜色。
- 图表初始化逻辑散落在研究页、回测详情、交易详情里。

### 21.6 后端质量约束

后端必须：

- 保持单体清晰模块。
- 核心模型不依赖数据库、HTTP、交易所 adapter。
- handler 只做请求解析、权限校验、响应映射。
- service / runner 承担业务流程。
- store 只承担持久化。
- adapter 只承担交易所协议映射和调用。
- strategy 只产生 intent。

禁止：

- 核心模型 import PostgreSQL 实现。
- 核心模型 import HTTP handler。
- 策略 import 交易所 adapter。
- 策略直接写数据库。
- 运行状态存全局变量。
- 用 `float64` 表示价格、数量、金额等交易事实。
- 静默吞错误。

### 21.7 依赖约束

前端：

- 可以使用成熟 UI 库。
- 可以使用 TradingView 开源图表。
- 可以使用成熟 i18n、状态管理、请求库。
- 不引入多个 UI 库互相覆盖。
- 不引入低质量小众组件堆页面。

后端：

- 可以使用 PostgreSQL 驱动。
- 可以使用成熟 HTTP router。
- 可以使用密码哈希、session/JWT、配置解析、日志等必要库。
- 安全敏感实现必须使用成熟库。
- 不手写密码学。
- 不引入大型业务框架。
- 不引入不必要消息队列、插件系统、规则引擎。

### 21.8 测试约束

必须测试：

- 数据同步续跑和断点恢复。
- K 线查询边界。
- 策略参数校验。
- 策略 intent 输出。
- 回测撮合和订单记录。
- 交易任务状态切换。
- 订单意图和通知意图处理。
- 登录和会话。
- 交易所账号配置校验。
- 通知投递失败记录。

前端必须测试：

- 路由是否可达。
- 顶部导航是否符合设计。
- 主题切换。
- 中英切换。
- 关键表单。
- 关键列表操作。
- 图表容器渲染。

测试原则：

- 小模块用单元测试。
- 跨模块核心链路用集成测试。
- 前端关键交互用组件测试或浏览器检查。
- 不追求虚假的全覆盖率，优先覆盖交易风险和核心体验。

### 21.9 变更约束

每次变更必须：

- 范围清楚。
- 不混入无关重构。
- 不同时改产品、格式化、依赖升级和大重构。
- 不覆盖用户未确认的方向。
- 文档和实现保持一致。

涉及以下内容必须先更新计划文档：

- 一级导航变化。
- 数据 / 研究页面结构变化。
- 任务类型变化。
- 策略意图模型变化。
- 数据库主模型变化。
- 交易所账号管理方式变化。
- 登录和安全边界变化。
- UI 库选择变化。

### 21.10 质量门禁建议

后续实现时应建立轻量门禁：

```text
go test ./...
go vet ./...
前端 typecheck
前端 lint
前端 test
前端 build
文件行数检查
函数长度 / 参数数量检查
命名长度检查
```

门禁要轻量、快速、服务质量。

禁止把门禁做成 `tictick-pro` 那种重证据系统。

## 22. 关键开放问题

需要继续确认：

1. 第一版前端暴露哪些 K 线周期选项。
2. 数据同步实时方式：WebSocket、轮询，还是交易所差异化。
3. TradingView 开源图表具体采用哪个包。
4. 第一版通知通道已接入邮件、Telegram、飞书基础 provider；生产启用、模板、限流和外部回执边界仍需确认。
5. 生产级交易所账号密钥来源、轮换和历史账号迁移策略。
6. 实盘任务创建确认文案、风险默认值和是否需要二次输入确认。
7. 回测第一版是否需要手续费和滑点配置。
8. 是否保留现有 `tictick-hi` Go + Vue 结构，还是进一步收敛目录。

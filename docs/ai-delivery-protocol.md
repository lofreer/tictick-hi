# AI Delivery Protocol

本文档约束所有负责推进 `tictick-hi` 的 AI。

核心原则：不能靠 AI 自觉保证质量，必须靠固定协议、验收标准和质量门禁约束推进。

## 1. 禁止自称完成

AI 不能直接说“已完成”“实现完了”。

每个模块只能处于以下等级之一：

```text
scaffold        骨架存在，但不能作为 demo
demo            能走通演示链路，但不保证真实可用
usable          用户可以围绕该模块进行真实工作
production-safe 具备安全、恢复、审计、稳定运行边界
done            用户明确确认关闭
```

最终回复必须明确写出本轮达到的等级。

如果没有跑完必要检查，最高只能标记为 `scaffold` 或 `demo`，不能标记为 `usable`。

## 2. 开工前固定检查

每轮开工前必须按顺序检查：

1. 阅读 `docs/implementation-plan.md`。
2. 阅读 `docs/quality-audit.md`。
3. 执行 `git status --short`，识别已有用户改动。
4. 检查本轮相关代码。
5. 明确本轮只推进哪个垂直切片。
6. 写出 Definition of Done。

没有 Definition of Done，不准开始写代码。

## 3. 垂直切片推进

禁止横向铺空壳。

正确推进方式：

```text
研究页 + 数据同步 + CandleProvider
  -> 回测
  -> 模拟盘
  -> 实盘
```

每个切片必须从前端交互进入，经过 API，落到 PostgreSQL，再能回到前端观察结果。

## 4. Definition of Done 模板

每个任务必须先写清楚：

```text
目标等级：
范围内：
范围外：
用户可见行为：
后端验收：
前端验收：
数据验收：
安全验收：
测试验收：
质量门禁：
```

示例：`CandleProvider` 的 Definition of Done：

```text
目标等级：usable
范围内：
- native 同周期 K 线查询
- 无同周期时由 1m 聚合更高周期
- 缺口检测
- 返回 native / aggregated 来源
- 图表、回测、交易 runner 统一调用

范围外：
- tick 数据
- 聚合缓存持久化
- 指标系统

质量门禁：
- go test ./...
- CandleProvider 单元测试
- PostgreSQL 集成测试
- scripts/quality-gate.sh
```

## 5. 停机规则

出现以下情况必须停止推进并汇报，不准继续堆代码：

- 计划文档和代码实现冲突。
- 用户语义不清，继续写会产生方向性偏差。
- 安全边界不成立。
- 实盘下单链路无法证明幂等。
- 数据同步无法证明可恢复。
- 测试或质量门禁失败。
- 文件超过硬上限。
- 为了赶进度需要写假实现。
- 需要覆盖用户已有改动。

## 6. 质量门禁

本项目质量门禁分两层。

轻量静态门禁：

```text
scripts/quality-gate.sh
```

完整验证门禁：

```text
go test ./...
go vet ./...
cd web/frontend && pnpm run typecheck
cd web/frontend && pnpm run test
cd web/frontend && pnpm run build
scripts/quality-gate.sh
```

如果某个命令没有执行，最终回复必须写“未执行”。

如果某个命令失败，最终回复必须写“未完成”。

## 7. 最终回复格式

每轮最终回复必须包含：

```text
完成等级：
本轮目标：
改动文件：
验证结果：
未执行检查：
剩余风险：
下一步：
```

禁止只写“已实现”“已完成”“搞定了”。

## 8. 当前项目默认状态

在 `docs/quality-audit.md` 关闭对应问题之前，`tictick-hi` 默认状态为：

```text
整体等级：scaffold
不能称为 demo
不能称为 usable
不能称为 production-safe
```


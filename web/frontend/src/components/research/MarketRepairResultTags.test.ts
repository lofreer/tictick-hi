import { mount } from "@vue/test-utils";
import { NConfigProvider } from "naive-ui";
import { describe, expect, it } from "vitest";
import { defineComponent } from "vue";

import MarketRepairResultTags from "@/components/research/MarketRepairResultTags.vue";
import { i18n } from "@/i18n";
import type { CandleResult, DataSyncGapRepairResult, DataSyncTask } from "@/types/app";

describe("MarketRepairResultTags", () => {
  it("renders repair summary, limited marker, three task windows and hidden count", () => {
    const wrapper = mountResultTags({
      sourceTaskId: "",
      createdTasks: [1, 2, 3, 4].map((index) => repairTask(index)),
      skippedExisting: 2,
      limited: true,
      totalCount: 8,
      repairLimit: 5,
    });

    expect(wrapper.text()).toContain("本次匹配 8 个，已创建 4 个，跳过 2 个，单次上限 5");
    expect(wrapper.text()).toContain("结果受限");
    expect(wrapper.text()).toContain("dst_repair_1");
    expect(wrapper.text()).toContain("等待");
    expect(wrapper.text()).toContain("同步中");
    expect(wrapper.text()).toContain("dst_repair_2");
    expect(wrapper.text()).toContain("dst_repair_3");
    expect(wrapper.text()).not.toContain("dst_repair_4");
    expect(wrapper.text()).toContain("另有 1 个补同步任务");
  });

  it("uses the latest task status from the task list when available", () => {
    const createdTask = repairTask(1);
    const wrapper = mountResultTags({
      sourceTaskId: "",
      createdTasks: [createdTask],
      skippedExisting: 0,
      limited: false,
      totalCount: 1,
      repairLimit: 5,
    }, {
      tasks: [repairTask(1, { dataHealth: "ok", status: "succeeded" })],
    });

    expect(wrapper.text()).toContain("dst_repair_1");
    expect(wrapper.text()).toContain("完成");
    expect(wrapper.text()).toContain("正常");
    expect(wrapper.text()).not.toContain("等待");
  });

  it("summarizes repair settlement as still running", () => {
    const wrapper = mountResultTags(repairResult([repairTask(1), repairTask(2)]), {
      tasks: [
        repairTask(1, { dataHealth: "ok", status: "succeeded" }),
        repairTask(2, { dataHealth: "syncing", status: "running" }),
      ],
    });

    expect(wrapper.text()).toContain("补同步执行中 1 个");
  });

  it("summarizes repair settlement as healthy when every repair task recovered", () => {
    const wrapper = mountResultTags(repairResult([repairTask(1)]), {
      tasks: [repairTask(1, { dataHealth: "ok", status: "succeeded" })],
    });

    expect(wrapper.text()).toContain("补同步窗口已恢复正常");
  });

  it("summarizes repair settlement failures and invalid data", () => {
    const wrapper = mountResultTags(repairResult([repairTask(1)]), {
      tasks: [repairTask(1, { dataHealth: "invalid", status: "succeeded" })],
    });

    expect(wrapper.text()).toContain("补同步已结束，仍有失败或异常");
  });

  it("shows the current chart window health when provided", () => {
    const result = repairResult([repairTask(1)]);

    expect(mountResultTags(result, { candleResult: candleResult({ health: "ok" }) }).text()).toContain("当前图表窗口已正常");
    expect(mountResultTags(result, { candleResult: candleResult({ gaps: [{ from: "2026-06-27T03:00:00Z", to: "2026-06-27T03:02:00Z", missingCandles: 1 }], health: "gap" }) }).text()).toContain("当前图表窗口仍有缺口 1 处");
    expect(mountResultTags(result, { candleResult: candleResult({ health: "invalid", issues: [{ code: "invalid_close_price", message: "invalid close" }] }) }).text()).toContain("当前图表窗口仍有异常 1 处");
  });
});

function mountResultTags(result: DataSyncGapRepairResult, props: { candleResult?: CandleResult; tasks?: DataSyncTask[] } = {}) {
  const wrapper = defineComponent({
    components: { MarketRepairResultTags, NConfigProvider },
    setup: () => ({ candleResult: props.candleResult, result, tasks: props.tasks }),
    template: `
      <NConfigProvider>
        <MarketRepairResultTags :candle-result="candleResult" :result="result" :tasks="tasks" />
      </NConfigProvider>
    `,
  });
  return mount(wrapper, {
    global: {
      plugins: [i18n],
    },
  });
}

function repairTask(index: number, overrides: Partial<DataSyncTask> = {}): DataSyncTask {
  return {
    id: `dst_repair_${index}`,
    exchange: "binance",
    symbol: "BTCUSDT",
    interval: "1m",
    startTime: `2026-06-27T03:0${index}:00Z`,
    endTime: `2026-06-27T03:0${index + 1}:00Z`,
    realtimeEnabled: false,
    syncEnabled: true,
    status: "pending",
    marketStatus: "active",
    dataHealth: "syncing",
    attemptCount: 0,
    createdAt: "2026-06-27T03:00:00Z",
    updatedAt: "2026-06-27T03:00:00Z",
    ...overrides,
  };
}

function repairResult(createdTasks: DataSyncTask[]): DataSyncGapRepairResult {
  return {
    sourceTaskId: "",
    createdTasks,
    skippedExisting: 0,
    limited: false,
    totalCount: createdTasks.length,
    repairLimit: 5,
  };
}

function candleResult(overrides: Partial<CandleResult>): CandleResult {
  return {
    candles: [],
    source: "native",
    requestedInterval: "1m",
    baseInterval: "1m",
    health: "ok",
    gaps: [],
    issues: [],
    coverage: { limitedByBaseWindow: false, requestedLimit: 1000, returnedCandles: 0 },
    window: { count: 0 },
    pagination: { hasNext: false, hasPrevious: false },
    ...overrides,
  };
}

import { mount } from "@vue/test-utils";
import { NConfigProvider } from "naive-ui";
import { describe, expect, it } from "vitest";
import { defineComponent } from "vue";

import MarketRepairResultTags from "@/components/research/MarketRepairResultTags.vue";
import { i18n } from "@/i18n";
import type { DataSyncGapRepairResult, DataSyncTask } from "@/types/app";

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
});

function mountResultTags(result: DataSyncGapRepairResult, props: { tasks?: DataSyncTask[] } = {}) {
  const wrapper = defineComponent({
    components: { MarketRepairResultTags, NConfigProvider },
    setup: () => ({ result, tasks: props.tasks }),
    template: `
      <NConfigProvider>
        <MarketRepairResultTags :result="result" :tasks="tasks" />
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

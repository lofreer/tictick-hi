import { flushPromises, mount } from "@vue/test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import ResearchTaskInvalidIssueModal from "@/components/research/ResearchTaskInvalidIssueModal.vue";
import { i18n } from "@/i18n";
import { dataApi } from "@/services/api/data";
import type { DataSyncTask } from "@/types/app";

const dataApiMocks = vi.hoisted(() => ({
  getTaskInvalidIssues: vi.fn(),
}));

vi.mock("@/services/api/data", () => ({
  dataApi: dataApiMocks,
}));

describe("ResearchTaskInvalidIssueModal", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    dataApiMocks.getTaskInvalidIssues.mockResolvedValue({
      taskId: "dst_1",
      issues: [
        {
          code: "invalid_open_price",
          message: "open price value must be positive",
          openTime: "2026-06-27T07:02:00Z",
        },
      ],
      limited: true,
      totalCount: 2,
      returnedCount: 1,
      issueLimit: 50,
    });
  });

  afterEach(() => {
    document.body.innerHTML = "";
  });

  it("loads and displays invalid candle issues for a task", async () => {
    const wrapper = mount(ResearchTaskInvalidIssueModal, {
      global: {
        plugins: [i18n],
      },
      attachTo: document.body,
    });
    const task = dataSyncTask({ id: "dst_1", exchange: "binance", symbol: "BTCUSDT", interval: "1m" });

    await (wrapper.vm as unknown as { open: (task: DataSyncTask) => Promise<void> }).open(task);
    await flushPromises();

    expect(dataApi.getTaskInvalidIssues).toHaveBeenCalledWith("dst_1");
    expect(document.body.textContent).toContain("binance / BTCUSDT / 1m");
    expect(document.body.textContent).toContain("开盘价必须为正");
    expect(document.body.textContent).toContain("open price value must be positive");
    expect(document.body.textContent).toContain("已显示 1/2 个异常");
  });
});

function dataSyncTask(overrides: Partial<DataSyncTask>): DataSyncTask {
  return {
    id: "dst_1",
    exchange: "binance",
    symbol: "BTCUSDT",
    interval: "1m",
    realtimeEnabled: false,
    syncEnabled: false,
    status: "succeeded",
    marketStatus: "active",
    dataHealth: "invalid",
    attemptCount: 0,
    createdAt: "2026-06-27T00:00:00Z",
    updatedAt: "2026-06-27T00:00:00Z",
    ...overrides,
  };
}

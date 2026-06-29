import { flushPromises, mount } from "@vue/test-utils";
import { NDatePicker, NPagination, NSelect } from "naive-ui";
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
      totalCount: 51,
      returnedCount: 1,
      issueLimit: 50,
      offset: 0,
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

    expect(dataApi.getTaskInvalidIssues).toHaveBeenCalledWith("dst_1", { limit: 50, offset: 0 });
    expect(document.body.textContent).toContain("binance / BTCUSDT / 1m");
    expect(document.body.textContent).toContain("开盘价必须为正");
    expect(document.body.textContent).toContain("open price value must be positive");
    expect(document.body.textContent).toContain("已显示 1/51 个异常");
  });

  it("loads the selected invalid issue page", async () => {
    dataApiMocks.getTaskInvalidIssues
      .mockResolvedValueOnce({
        taskId: "dst_1",
        issues: [
          {
            code: "invalid_open_price",
            message: "open price value must be positive",
            openTime: "2026-06-27T07:02:00Z",
          },
        ],
        limited: true,
        totalCount: 51,
        returnedCount: 50,
        issueLimit: 50,
        offset: 0,
      })
      .mockResolvedValueOnce({
        taskId: "dst_1",
        issues: [
          {
            code: "invalid_close_price",
            message: "close price value must be positive",
            openTime: "2026-06-27T07:52:00Z",
          },
        ],
        limited: false,
        totalCount: 51,
        returnedCount: 1,
        issueLimit: 50,
        offset: 50,
      });
    const wrapper = mount(ResearchTaskInvalidIssueModal, {
      global: {
        plugins: [i18n],
      },
      attachTo: document.body,
    });
    const task = dataSyncTask({ id: "dst_1", exchange: "binance", symbol: "BTCUSDT", interval: "1m" });

    await (wrapper.vm as unknown as { open: (task: DataSyncTask) => Promise<void> }).open(task);
    await flushPromises();
    await wrapper.findComponent(NPagination).vm.$emit("update:page", 2);
    await flushPromises();

    expect(dataApi.getTaskInvalidIssues).toHaveBeenNthCalledWith(1, "dst_1", { limit: 50, offset: 0 });
    expect(dataApi.getTaskInvalidIssues).toHaveBeenNthCalledWith(2, "dst_1", { limit: 50, offset: 50 });
    expect(document.body.textContent).toContain("收盘价必须为正");
    expect(document.body.textContent).toContain("已显示 51/51 个异常");
  });

  it("reloads invalid issues with code and time filters", async () => {
    const wrapper = mount(ResearchTaskInvalidIssueModal, {
      global: {
        plugins: [i18n],
      },
      attachTo: document.body,
    });
    const task = dataSyncTask({ id: "dst_1", exchange: "binance", symbol: "BTCUSDT", interval: "1m" });
    const from = Date.parse("2026-06-27T07:00:00.000Z");
    const to = Date.parse("2026-06-27T08:00:00.000Z");

    await (wrapper.vm as unknown as { open: (task: DataSyncTask) => Promise<void> }).open(task);
    await flushPromises();
    await wrapper.findComponent(NSelect).vm.$emit("update:value", "invalid_close_price");
    await flushPromises();
    await wrapper.findComponent(NDatePicker).vm.$emit("update:value", [from, to]);
    await flushPromises();

    expect(dataApi.getTaskInvalidIssues).toHaveBeenNthCalledWith(2, "dst_1", {
      code: "invalid_close_price",
      limit: 50,
      offset: 0,
    });
    expect(dataApi.getTaskInvalidIssues).toHaveBeenNthCalledWith(3, "dst_1", {
      code: "invalid_close_price",
      from: "2026-06-27T07:00:00.000Z",
      limit: 50,
      offset: 0,
      to: "2026-06-27T08:00:00.000Z",
    });
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

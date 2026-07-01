import { flushPromises, mount } from "@vue/test-utils";
import { NDatePicker, NPagination, NSelect } from "naive-ui";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import ResearchTaskInvalidIssueModal from "@/components/research/ResearchTaskInvalidIssueModal.vue";
import { i18n } from "@/i18n";
import { dataApi } from "@/services/api/data";
import type { DataSyncTask } from "@/types/app";

const dataApiMocks = vi.hoisted(() => ({
  getTaskInvalidIssues: vi.fn(),
  quarantineMarketCandleInvalidIssues: vi.fn(),
  repairTaskInvalidIssues: vi.fn(),
}));

vi.mock("@/services/api/data", () => ({
  dataApi: dataApiMocks,
}));

describe("ResearchTaskInvalidIssueModal", () => {
  beforeEach(() => {
    vi.resetAllMocks();
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
    dataApiMocks.repairTaskInvalidIssues.mockResolvedValue({
      sourceTaskId: "dst_1",
      createdTasks: [{
        id: "dst_repair_1",
        exchange: "binance",
        symbol: "BTCUSDT",
        interval: "1m",
        startTime: "2026-06-27T07:02:00Z",
        endTime: "2026-06-27T07:03:00Z",
        realtimeEnabled: false,
        syncEnabled: true,
        status: "pending",
        marketStatus: "active",
        dataHealth: "syncing",
        attemptCount: 0,
        createdAt: "2026-06-27T07:02:01Z",
        updatedAt: "2026-06-27T07:02:01Z",
      }],
      skippedExisting: 0,
      limited: false,
      totalCount: 1,
      repairLimit: 20,
    });
    dataApiMocks.quarantineMarketCandleInvalidIssues.mockResolvedValue({
      quarantined: [{
        exchange: "binance",
        symbol: "BTCUSDT",
        interval: "1m",
        openTime: "2026-06-27T07:02:30Z",
        closeTime: "2026-06-27T07:03:30Z",
        reason: "invalid_open_time",
        message: "open time is not aligned to interval",
        quarantinedAt: "2026-06-27T07:04:00Z",
      }],
      skippedNonQuarantinable: 0,
      totalCount: 1,
      quarantineLimit: 100,
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

  it("queues repair tasks for the current invalid issue filters", async () => {
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

    clickRepairButton();
    await flushPromises();

    expect(dataApi.repairTaskInvalidIssues).toHaveBeenCalledWith("dst_1", {
      code: "invalid_close_price",
      from: "2026-06-27T07:00:00.000Z",
      to: "2026-06-27T08:00:00.000Z",
    });
    expect(wrapper.emitted("repaired")).toHaveLength(1);
    expect(document.body.textContent).toContain("已排队 1 个异常 K 线补同步任务");
    expect(document.body.textContent).toContain("本次匹配 1 个，已创建 1 个，跳过 0 个，单次上限 20");
    expect(document.body.textContent).toContain("dst_repair_1 /");
  });

  it("refreshes current invalid issue filters after queued repair tasks settle", async () => {
    dataApiMocks.getTaskInvalidIssues
      .mockResolvedValueOnce({
        taskId: "dst_1",
        issues: [{
          code: "invalid_open_price",
          message: "open price value must be positive",
          openTime: "2026-06-27T07:02:00Z",
        }],
        limited: false,
        totalCount: 1,
        returnedCount: 1,
        issueLimit: 50,
        offset: 0,
      })
      .mockResolvedValueOnce({
        taskId: "dst_1",
        issues: [{
          code: "invalid_open_price",
          message: "open price value must be positive",
          openTime: "2026-06-27T07:02:00Z",
        }],
        limited: false,
        totalCount: 1,
        returnedCount: 1,
        issueLimit: 50,
        offset: 0,
      })
      .mockResolvedValueOnce({
        taskId: "dst_1",
        issues: [{
          code: "invalid_open_price",
          message: "open price value must be positive",
          openTime: "2026-06-27T07:02:00Z",
        }],
        limited: false,
        totalCount: 1,
        returnedCount: 1,
        issueLimit: 50,
        offset: 0,
      })
      .mockResolvedValueOnce({
        taskId: "dst_1",
        issues: [],
        limited: false,
        totalCount: 0,
        returnedCount: 0,
        issueLimit: 50,
        offset: 0,
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
    await wrapper.findComponent(NSelect).vm.$emit("update:value", "invalid_open_price");
    await flushPromises();
    clickRepairButton();
    await flushPromises();
    expect(dataApi.getTaskInvalidIssues).toHaveBeenCalledTimes(3);

    await wrapper.setProps({ tasks: [dataSyncTask({ id: "dst_repair_1", status: "running", dataHealth: "syncing" })] });
    await flushPromises();
    expect(dataApi.getTaskInvalidIssues).toHaveBeenCalledTimes(3);

    await wrapper.setProps({ tasks: [dataSyncTask({ id: "dst_repair_1", status: "succeeded", dataHealth: "ok" })] });
    await flushPromises();

    expect(dataApi.getTaskInvalidIssues).toHaveBeenCalledTimes(4);
    expect(dataApi.getTaskInvalidIssues).toHaveBeenLastCalledWith("dst_1", {
      code: "invalid_open_price",
      limit: 50,
      offset: 0,
    });
    expect(document.body.textContent).toContain("暂无异常详情");
    expect(document.body.textContent).toContain("本次匹配 1 个，已创建 1 个，跳过 0 个，单次上限 20");

    await wrapper.setProps({ tasks: [dataSyncTask({ id: "dst_repair_1", status: "succeeded", dataHealth: "ok" })] });
    await flushPromises();
    expect(dataApi.getTaskInvalidIssues).toHaveBeenCalledTimes(4);
  });

  it("quarantines invalid open-time issues without exposing normal repair", async () => {
    dataApiMocks.getTaskInvalidIssues
      .mockResolvedValueOnce({
        taskId: "dst_1",
        issues: [{
          code: "invalid_open_time",
          message: "open time is not aligned to interval",
          openTime: "2026-06-27T07:02:30Z",
        }],
        limited: false,
        totalCount: 1,
        returnedCount: 1,
        issueLimit: 50,
        offset: 0,
      })
      .mockResolvedValueOnce({
        taskId: "dst_1",
        issues: [{
          code: "invalid_open_time",
          message: "open time is not aligned to interval",
          openTime: "2026-06-27T07:02:30Z",
        }],
        limited: false,
        totalCount: 1,
        returnedCount: 1,
        issueLimit: 50,
        offset: 0,
      })
      .mockResolvedValueOnce({
        taskId: "dst_1",
        issues: [],
        limited: false,
        totalCount: 0,
        returnedCount: 0,
        issueLimit: 50,
        offset: 0,
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
    await wrapper.findComponent(NSelect).vm.$emit("update:value", "invalid_open_time");
    await flushPromises();

    expect(document.body.textContent).toContain("K 线开盘时间未对齐周期");
    expect(document.body.textContent).not.toContain("排队修复当前异常");
    expect(document.body.textContent).toContain("隔离错位 K 线");
    clickQuarantineButton();
    await flushPromises();

    expect(dataApi.repairTaskInvalidIssues).not.toHaveBeenCalled();
    expect(dataApi.quarantineMarketCandleInvalidIssues).toHaveBeenCalledWith({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      openTimes: ["2026-06-27T07:02:30Z"],
    });
    expect(wrapper.emitted("quarantined")).toEqual([[task]]);
  });

  it("shows skipped and limited repair metadata", async () => {
    dataApiMocks.repairTaskInvalidIssues.mockResolvedValueOnce({
      sourceTaskId: "dst_1",
      createdTasks: [],
      skippedExisting: 2,
      limited: true,
      totalCount: 25,
      repairLimit: 20,
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
    clickRepairButton();
    await flushPromises();

    expect(document.body.textContent).toContain("异常 K 线补同步任务已存在");
    expect(document.body.textContent).toContain("本次匹配 25 个，已创建 0 个，跳过 2 个，单次上限 20");
    expect(document.body.textContent).toContain("结果受限");
  });

  it("shows no-repair metadata when current filters no longer match invalid candles", async () => {
    dataApiMocks.repairTaskInvalidIssues.mockResolvedValueOnce({
      sourceTaskId: "dst_1",
      createdTasks: [],
      skippedExisting: 0,
      limited: false,
      totalCount: 0,
      repairLimit: 20,
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
    clickRepairButton();
    await flushPromises();

    expect(document.body.textContent).toContain("当前筛选没有可排队的异常 K 线");
    expect(document.body.textContent).toContain("本次匹配 0 个，已创建 0 个，跳过 0 个，单次上限 20");
  });

  it("clears stale repair metadata when filters change", async () => {
    const wrapper = mount(ResearchTaskInvalidIssueModal, {
      global: {
        plugins: [i18n],
      },
      attachTo: document.body,
    });
    const task = dataSyncTask({ id: "dst_1", exchange: "binance", symbol: "BTCUSDT", interval: "1m" });

    await (wrapper.vm as unknown as { open: (task: DataSyncTask) => Promise<void> }).open(task);
    await flushPromises();
    clickRepairButton();
    await flushPromises();
    expect(document.body.textContent).toContain("dst_repair_1");

    await wrapper.findComponent(NSelect).vm.$emit("update:value", "invalid_close_price");
    await flushPromises();

    expect(document.body.textContent).not.toContain("dst_repair_1");
    expect(document.body.textContent).not.toContain("本次匹配 1 个");
  });

  it("shows a sanitized repair failure without leaking upstream URLs", async () => {
    dataApiMocks.repairTaskInvalidIssues.mockRejectedValueOnce(
      new Error('binance klines: Get "https://api.binance.com/api/v3/klines?symbol=BTCUSDT": EOF'),
    );
    const wrapper = mount(ResearchTaskInvalidIssueModal, {
      global: {
        plugins: [i18n],
      },
      attachTo: document.body,
    });
    const task = dataSyncTask({ id: "dst_1", exchange: "binance", symbol: "BTCUSDT", interval: "1m" });

    await (wrapper.vm as unknown as { open: (task: DataSyncTask) => Promise<void> }).open(task);
    await flushPromises();
    clickRepairButton();
    await flushPromises();

    expect(document.body.textContent).toContain("创建异常 K 线补同步任务失败");
    expect(document.body.textContent).not.toContain("api.binance.com");
  });
});

function clickRepairButton() {
  const repairButton = Array.from(document.body.querySelectorAll("button")).find((button) =>
    button.textContent?.includes("排队修复当前异常"),
  );
  if (!repairButton) throw new Error("repair button not found");
  repairButton.click();
}

function clickQuarantineButton() {
  const quarantineButton = Array.from(document.body.querySelectorAll("button")).find((button) =>
    button.textContent?.includes("隔离错位 K 线"),
  );
  if (!quarantineButton) throw new Error("quarantine button not found");
  quarantineButton.click();
}

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

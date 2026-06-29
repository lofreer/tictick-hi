import { mount } from "@vue/test-utils";
import { NDataTable } from "naive-ui";
import { describe, expect, it } from "vitest";

import DataSyncTaskTable from "@/components/tables/DataSyncTaskTable.vue";
import { i18n } from "@/i18n";
import type { DataSyncTask } from "@/types/app";

describe("DataSyncTaskTable", () => {
  it("keeps long errors inside a bounded table viewport", () => {
    const longError =
      'binance klines: Get "https://api.binance.com/api/v3/klines?endTime=1782524388943&interval=1m&limit=500&startTime=1780277926000&symbol=BTCUSDT": EOF';

    const wrapper = mount(DataSyncTaskTable, {
      global: { plugins: [i18n] },
      props: {
        tasks: [
          dataSyncTask({
            id: "sync_1",
            exchange: "binance",
            symbol: "BTCUSDT",
            interval: "1m",
            lastError: longError,
          }),
        ],
      },
    });

    const table = wrapper.findComponent(NDataTable);
    expect(table.props("maxHeight")).toBe(260);
    expect(table.props("scrollX")).toBe(2210);
    expect(wrapper.find(".data-sync-task-table").exists()).toBe(true);
    const columns = table.props("columns") as Array<{ fixed?: string; key?: string; width?: number }>;
    const actionsColumn = columns.find((column) => column.key === "actions");
    expect(actionsColumn?.fixed).toBeUndefined();
    expect(actionsColumn).toMatchObject({ width: 292 });

    const errorText = wrapper.get(".task-error-text");
    expect(errorText.attributes("title")).not.toContain("/api/v3/klines");
    expect(errorText.attributes("title")).not.toContain("symbol=BTCUSDT");
    expect(errorText.attributes("title")).toContain("api.binance.com");
    expect(errorText.text()).toContain("api.binance.com");
    expect(errorText.text()).toContain("EOF");
  });

  it("shows scheduled retry time for temporary sync errors", () => {
    const wrapper = mount(DataSyncTaskTable, {
      global: { plugins: [i18n] },
      props: {
        tasks: [
          dataSyncTask({
            id: "sync_1",
            exchange: "binance",
            symbol: "BTCUSDT",
            interval: "1m",
            status: "running",
            lastError: "temporary EOF",
            nextAttemptAt: "2026-06-28T01:30:00Z",
          }),
        ],
      },
    });

    expect(wrapper.text()).toContain("2026-06-28T01:30:00Z");
  });

  it("shows exchange-level backoff without leaking request URLs", () => {
    const wrapper = mount(DataSyncTaskTable, {
      global: { plugins: [i18n] },
      props: {
        tasks: [
          dataSyncTask({
            id: "sync_1",
            exchange: "binance",
            symbol: "BTCUSDT",
            interval: "1m",
            status: "pending",
            dataHealth: "retrying",
            exchangeBackoffUntil: "2026-06-28T01:45:00Z",
            exchangeBackoffLastError:
              'binance klines temporary unavailable: Get "https://api.binance.com/api/v3/klines?symbol=BTCUSDT": EOF',
          }),
        ],
      },
    });

    expect(wrapper.text()).toContain("交易所退避");
    const backoff = wrapper.get(".task-exchange-backoff");
    expect(backoff.text()).toBe("2026-06-28T01:45:00Z");
    expect(backoff.attributes("title")).toContain("api.binance.com");
    expect(backoff.attributes("title")).not.toContain("/api/v3/klines");
    expect(backoff.attributes("title")).not.toContain("symbol=BTCUSDT");
  });

  it("shows backend-derived data health", () => {
    const wrapper = mount(DataSyncTaskTable, {
      global: { plugins: [i18n] },
      props: {
        tasks: [
          dataSyncTask({
            id: "sync_1",
            exchange: "binance",
            symbol: "BTCUSDT",
            interval: "1m",
            dataHealth: "gap",
          }),
        ],
      },
    });

    expect(wrapper.text()).toContain("数据健康");
    expect(wrapper.text()).toContain("有缺口");
  });

  it("shows inactive market status and disables start commands", () => {
    const wrapper = mount(DataSyncTaskTable, {
      global: { plugins: [i18n] },
      props: {
        tasks: [
          dataSyncTask({
            id: "sync_1",
            marketStatus: "inactive",
            marketStatusDetail: "BREAK",
            status: "paused",
            syncEnabled: false,
            realtimeEnabled: false,
            dataHealth: "paused",
          }),
        ],
      },
    });

    expect(wrapper.text()).toContain("市场状态");
    expect(wrapper.text()).toContain("Inactive · BREAK");
    expect(wrapper.get(".task-market-status").attributes("title")).toBe("Inactive · BREAK");
    expect(wrapper.get('button[title="市场非 active"]').attributes("disabled")).toBeDefined();
    expect(wrapper.emitted("toggle-sync")).toBeUndefined();
    expect(wrapper.emitted("toggle-realtime")).toBeUndefined();
  });

  it("shows sync task window boundaries", () => {
    const wrapper = mount(DataSyncTaskTable, {
      global: { plugins: [i18n] },
      props: {
        tasks: [
          dataSyncTask({
            id: "repair_1",
            startTime: "2026-06-27T03:02:00Z",
            endTime: "2026-06-27T03:03:00Z",
            repairSourceTaskId: "dst_source_1",
          }),
          dataSyncTask({
            id: "tail_1",
            startTime: "2026-06-27T03:02:00Z",
          }),
          dataSyncTask({
            id: "continuous_1",
          }),
        ],
      },
    });

    const windows = wrapper.findAll(".task-sync-window").map((node) => node.text());
    expect(windows).toContain("修复来源 dst_source_1 / 2026-06-27T03:02:00Z 到 2026-06-27T03:03:00Z");
    expect(windows).toContain("从 2026-06-27T03:02:00Z 开始");
    expect(windows).toContain("持续同步");
  });

  it("shows backend-derived gap summary", () => {
    const wrapper = mount(DataSyncTaskTable, {
      global: { plugins: [i18n] },
      props: {
        tasks: [
          dataSyncTask({
            id: "sync_1",
            exchange: "binance",
            symbol: "BTCUSDT",
            interval: "1m",
            dataHealth: "gap",
            gapSummary: {
              count: 2,
              firstGap: {
                from: "2026-06-27T03:02:00Z",
                to: "2026-06-27T03:03:00Z",
                missingCandles: 1,
              },
            },
          }),
        ],
      },
    });

    expect(wrapper.text()).toContain("缺口摘要");
    const summary = wrapper.get(".task-gap-summary");
    expect(summary.text()).toContain("缺口 2 处");
    expect(summary.attributes("title")).toContain("2026-06-27T03:02:00Z");
  });

  it("emits view gaps for tasks with gap summary", async () => {
    const gapTask = dataSyncTask({
      id: "sync_1",
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      dataHealth: "gap",
      gapSummary: {
        count: 1,
        firstGap: {
          from: "2026-06-27T03:02:00Z",
          to: "2026-06-27T03:03:00Z",
          missingCandles: 1,
        },
      },
    });
    const wrapper = mount(DataSyncTaskTable, {
      global: { plugins: [i18n] },
      props: {
        tasks: [gapTask],
      },
    });

    await wrapper.get('button[title="查看缺口"]').trigger("click");

    expect(wrapper.emitted("view-gaps")).toEqual([[gapTask]]);
  });

  it("emits repair for tasks with gap summary", async () => {
    const gapTask = dataSyncTask({
      id: "sync_1",
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      dataHealth: "gap",
      gapSummary: {
        count: 1,
        firstGap: {
          from: "2026-06-27T03:02:00Z",
          to: "2026-06-27T03:03:00Z",
          missingCandles: 1,
        },
      },
    });
    const wrapper = mount(DataSyncTaskTable, {
      global: { plugins: [i18n] },
      props: {
        tasks: [gapTask],
      },
    });

    await wrapper.get('button[title="修复缺口"]').trigger("click");

    expect(wrapper.emitted("repair-gaps")).toEqual([[gapTask]]);
  });

  it("emits retry for failed tasks", async () => {
    const failedTask = dataSyncTask({
      id: "sync_1",
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      syncEnabled: false,
      lastError: "invalid symbol",
    });
    const wrapper = mount(DataSyncTaskTable, {
      global: { plugins: [i18n] },
      props: {
        tasks: [failedTask],
      },
    });

    await wrapper.get('button[title="重试"]').trigger("click");

    expect(wrapper.emitted("retry")).toEqual([[failedTask]]);
    expect(wrapper.find('button[title="同步"]').exists()).toBe(false);
  });

  it("allows restarting succeeded one-shot sync tasks", async () => {
    const succeededTask = dataSyncTask({
      id: "sync_1",
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      status: "succeeded",
      syncEnabled: false,
      dataHealth: "gap",
    });
    const wrapper = mount(DataSyncTaskTable, {
      global: { plugins: [i18n] },
      props: {
        tasks: [succeededTask],
      },
    });

    await wrapper.get('button[title="同步"]').trigger("click");

    expect(wrapper.emitted("toggle-sync")).toEqual([[succeededTask]]);
    expect(wrapper.find('button[title="重试"]').exists()).toBe(false);
  });
});

function dataSyncTask(overrides: Partial<DataSyncTask>): DataSyncTask {
  return {
    id: "sync_1",
    exchange: "binance",
    symbol: "BTCUSDT",
    interval: "1m",
    realtimeEnabled: false,
    syncEnabled: true,
    status: "failed",
    marketStatus: "active",
    dataHealth: "failed",
    attemptCount: 1,
    createdAt: "2026-06-28T00:00:00Z",
    updatedAt: "2026-06-28T00:00:00Z",
    ...overrides,
  };
}

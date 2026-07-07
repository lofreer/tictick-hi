import { mount, flushPromises } from "@vue/test-utils";
import { describe, expect, it, vi, beforeEach } from "vitest";

import { i18n } from "@/i18n";
import { backtestsApi } from "@/services/api/backtests";
import { dataApi } from "@/services/api/data";
import { overviewApi } from "@/services/api/overview";
import { systemApi } from "@/services/api/system";
import { tradingApi } from "@/services/api/trading";
import { useOverviewWorkspace } from "@/composables/useOverviewWorkspace";

const apiMocks = vi.hoisted(() => ({
  listBacktests: vi.fn(),
  listDataTasks: vi.fn(),
  listNotifications: vi.fn(),
  listTradingTasks: vi.fn(),
  overviewRecentFacts: vi.fn(),
  systemHealth: vi.fn(),
}));

vi.mock("@/services/api/backtests", () => ({
  backtestsApi: {
    listBacktests: apiMocks.listBacktests,
  },
}));

vi.mock("@/services/api/data", () => ({
  dataApi: {
    listTasks: apiMocks.listDataTasks,
  },
}));

vi.mock("@/services/api/system", () => ({
  systemApi: {
    health: apiMocks.systemHealth,
    listNotifications: apiMocks.listNotifications,
  },
}));

vi.mock("@/services/api/overview", () => ({
  overviewApi: {
    recentFacts: apiMocks.overviewRecentFacts,
  },
}));

vi.mock("@/services/api/trading", () => ({
  tradingApi: {
    listTasks: apiMocks.listTradingTasks,
  },
}));

describe("useOverviewWorkspace", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    apiMocks.overviewRecentFacts.mockResolvedValue({ orders: [], strategyIntents: [] });
  });

  it("loads real overview sources and derives summary state", async () => {
    apiMocks.systemHealth.mockResolvedValue({
      status: "warning",
      database: "ok",
      checkedAt: isoMinutesAgo(12),
      services: [
        {
          name: "sync",
          status: "warning",
          pendingCount: 2,
          runningCount: 1,
          lockedCount: 1,
          staleLeaseCount: 1,
        },
      ],
    });
    apiMocks.listDataTasks.mockResolvedValue([
      task("sync_1", "running", isoMinutesAgo(11), { dataHealth: "gap", realtimeEnabled: true }),
      task("sync_2", "failed", isoMinutesAgo(10)),
      task("sync_3", "succeeded", isoMinutesAgo(9), { dataHealth: "invalid" }),
    ]);
    apiMocks.listBacktests.mockResolvedValue([
      task("bt_1", "succeeded", isoMinutesAgo(8), { name: "Baseline" }),
      task("bt_2", "failed", isoMinutesAgo(7), { name: "Failed test" }),
    ]);
    apiMocks.listTradingTasks.mockResolvedValue([
      task("tt_1", "running", isoMinutesAgo(6), { name: "Paper", type: "paper" }),
      task("tt_2", "failed", isoMinutesAgo(5), { name: "Live", type: "live" }),
    ]);
    apiMocks.overviewRecentFacts.mockResolvedValue({
      orders: [
        overviewOrder("ord_1", "tt_1", "paper", "Paper", "sell", "66000", "0.2", "filled", isoMinutesAgo(1)),
        overviewOrder("bo_1", "bt_1", "backtest", "Baseline", "buy", "65000", "0.1", "filled", isoMinutesAgo(8)),
      ],
      strategyIntents: [
        overviewIntent("si_bt_1", "bt_1", "backtest", "Baseline", "accepted", "order", "simulate", isoMinutesAgo(2)),
        overviewIntent("si_tt_1", "tt_1", "paper", "Paper", "accepted", "order", "execute", isoMinutesAgo(3)),
      ],
    });
    apiMocks.listNotifications.mockResolvedValue([
      {
        id: "nt_1",
        channel: "ops",
        title: "failed alert",
        status: "failed",
        createdAt: isoMinutesAgo(4),
      },
    ]);

    const beforeMount = Date.now();
    const workspace = mountWorkspace();
    await flushPromises();

    expect(systemApi.health).toHaveBeenCalledTimes(1);
    expect(dataApi.listTasks).toHaveBeenCalledTimes(1);
    expect(backtestsApi.listBacktests).toHaveBeenCalledTimes(1);
    expect(tradingApi.listTasks).toHaveBeenCalledTimes(1);
    expect(overviewApi.recentFacts).toHaveBeenCalledTimes(1);
    expectRecentFactsWindowSince(beforeMount);
    expect(systemApi.listNotifications).toHaveBeenCalledTimes(1);
    expect(workspace.hasLoaded.value).toBe(true);
    expect(workspace.summaryCards.value.find((card) => card.key === "sync")?.value).toBe(3);
    expect(workspace.summaryCards.value.find((card) => card.key === "sync")?.detail).toContain("异常 1");
    expect(workspace.summaryCards.value.find((card) => card.key === "workers")?.detail).toContain("过期锁 1");
    expect(workspace.summaryCards.value.map((card) => ({ key: card.key, to: card.to }))).toEqual([
      { key: "sync", to: { name: "research" } },
      { key: "backtests", to: { name: "backtests" } },
      { key: "trading", to: { name: "trading" } },
      { key: "notifications", to: { name: "system-notifications" } },
      { key: "workers", to: { name: "system-health" } },
    ]);
    expect(workspace.depthMetrics.value.map((metric) => ({ key: metric.key, value: metric.value, statusType: metric.statusType, to: metric.to }))).toEqual([
      { key: "data-quality", value: "0/3", statusType: "error", to: { name: "research" } },
      { key: "automation", value: "1/3", statusType: "error", to: { name: "system-health" } },
      { key: "execution", value: "1/4", statusType: "error", to: { name: "trading" } },
      { key: "delivery", value: "0/1", statusType: "error", to: { name: "system-notifications" } },
    ]);
    expect(workspace.depthMetrics.value.find((metric) => metric.key === "data-quality")?.detail).toContain("缺口 1");
    expect(workspace.depthMetrics.value.find((metric) => metric.key === "automation")?.detail).toContain("过期锁 1");
    expect(workspace.depthMetrics.value.find((metric) => metric.key === "execution")?.detail).toContain("交易失败 1");
    expect(workspace.depthMetrics.value.find((metric) => metric.key === "delivery")?.statusLabel).toBe("风险");
    expect(workspace.alerts.value.map((alert) => alert.key)).toEqual([
      "health",
      "sync-failed",
      "sync-gap",
      "sync-invalid",
      "backtests-failed",
      "trading-failed",
      "notifications-failed",
    ]);
    expect(workspace.recentActivities.value.slice(0, 4).map((activity) => activity.key)).toEqual([
      "order-ord_1",
      "intent-si_bt_1",
      "intent-si_tt_1",
      "notification-nt_1",
    ]);
    expect(workspace.recentActivities.value[0]).toEqual(
      expect.objectContaining({
        title: "订单",
        detail: expect.stringContaining("Paper"),
        status: "filled",
        to: { name: "trading-detail", params: { id: "tt_1" } },
      }),
    );
    expect(workspace.recentActivities.value[1]).toEqual(
      expect.objectContaining({
        title: "策略意图",
        detail: expect.stringContaining("Baseline"),
        status: "accepted",
        to: { name: "backtests-detail", params: { id: "bt_1" } },
      }),
    );
  });

  it("surfaces load failures without marking the overview loaded", async () => {
    apiMocks.systemHealth.mockRejectedValue(new Error("health unavailable"));
    apiMocks.listDataTasks.mockResolvedValue([]);
    apiMocks.listBacktests.mockResolvedValue([]);
    apiMocks.listTradingTasks.mockResolvedValue([]);
    apiMocks.listNotifications.mockResolvedValue([]);

    const workspace = mountWorkspace();
    await flushPromises();

    expect(workspace.hasLoaded.value).toBe(false);
    expect(workspace.error.value).toBe("health unavailable");
  });

  it("keeps the overview loaded when recent facts are degraded", async () => {
    apiMocks.systemHealth.mockResolvedValue({
      status: "ok",
      database: "ok",
      checkedAt: isoMinutesAgo(4),
      services: [],
    });
    apiMocks.listDataTasks.mockResolvedValue([
      task("sync_1", "running", isoMinutesAgo(4)),
    ]);
    apiMocks.listBacktests.mockResolvedValue([
      task("bt_1", "succeeded", isoMinutesAgo(3), { name: "Baseline" }),
    ]);
    apiMocks.listTradingTasks.mockResolvedValue([
      task("tt_1", "running", isoMinutesAgo(2), { name: "Paper", type: "paper" }),
    ]);
    apiMocks.listNotifications.mockResolvedValue([
      {
        id: "nt_1",
        channel: "ops",
        title: "filled alert",
        status: "sent",
        createdAt: isoMinutesAgo(1),
      },
    ]);
    apiMocks.overviewRecentFacts.mockRejectedValue(new Error("recent facts unavailable"));

    const workspace = mountWorkspace();
    await flushPromises();

    expect(overviewApi.recentFacts).toHaveBeenCalledTimes(1);
    expect(workspace.hasLoaded.value).toBe(true);
    expect(workspace.error.value).toBe("");
    expect(workspace.factsError.value).toBe("recent facts unavailable");
    expect(workspace.alerts.value).toEqual([
      expect.objectContaining({
        key: "recent-facts-degraded",
        label: "局部降级",
        detail: "recent facts unavailable",
      }),
    ]);
    expect(workspace.recentActivities.value.map((activity) => activity.key)).toEqual([
      "notification-nt_1",
      "交易任务-tt_1",
      "回测任务-bt_1",
      "数据同步-sync_1",
    ]);
    expect(workspace.recentActivities.value.some((activity) => activity.key.startsWith("intent-"))).toBe(false);
    expect(workspace.recentActivities.value.some((activity) => activity.key.startsWith("order-"))).toBe(false);
  });

  it("switches the recent activity window by reloading only recent facts", async () => {
    apiMocks.systemHealth.mockResolvedValue({ status: "ok", database: "ok", checkedAt: isoMinutesAgo(1), services: [] });
    apiMocks.listDataTasks.mockResolvedValue([task("sync_recent", "running", isoMinutesAgo(10))]);
    apiMocks.listBacktests.mockResolvedValue([]);
    apiMocks.listTradingTasks.mockResolvedValue([]);
    apiMocks.listNotifications.mockResolvedValue([
      { id: "nt_old", channel: "ops", title: "old alert", status: "sent", createdAt: isoDaysAgo(3) },
    ]);
    apiMocks.overviewRecentFacts
      .mockResolvedValueOnce({ orders: [], strategyIntents: [overviewIntent("si_recent", "bt_1", "backtest", "Recent", "accepted", "order", "simulate", isoMinutesAgo(5))] })
      .mockResolvedValueOnce({ orders: [], strategyIntents: [overviewIntent("si_old", "bt_2", "backtest", "Old", "accepted", "order", "simulate", isoDaysAgo(3))] });

    const workspace = mountWorkspace();
    await flushPromises();
    expect(workspace.recentActivityWindow.value).toBe("24h");
    expect(workspace.recentActivities.value.map((activity) => activity.key)).toEqual(["intent-si_recent", "数据同步-sync_recent"]);

    const beforeSwitch = Date.now();
    await workspace.setRecentActivityWindow("7d");
    await flushPromises();

    expect(systemApi.health).toHaveBeenCalledTimes(1);
    expect(dataApi.listTasks).toHaveBeenCalledTimes(1);
    expect(backtestsApi.listBacktests).toHaveBeenCalledTimes(1);
    expect(tradingApi.listTasks).toHaveBeenCalledTimes(1);
    expect(systemApi.listNotifications).toHaveBeenCalledTimes(1);
    expect(overviewApi.recentFacts).toHaveBeenCalledTimes(2);
    expect(workspace.recentActivityWindow.value).toBe("7d");
    expectRecentFactsWindowSince(beforeSwitch, 7 * 24 * 60 * 60 * 1000, 1);
    expect(workspace.recentActivities.value.map((activity) => activity.key)).toEqual(["数据同步-sync_recent", "intent-si_old", "notification-nt_old"]);
  });

  it("recalculates the recent facts window anchor on overview reload", async () => {
    const firstNow = Date.parse("2026-07-07T12:00:00Z");
    const secondNow = firstNow + 60 * 60 * 1000;
    const nowSpy = vi.spyOn(Date, "now").mockReturnValue(firstNow);

    try {
      apiMocks.systemHealth.mockResolvedValue({ status: "ok", database: "ok", checkedAt: "2026-07-07T12:00:00Z", services: [] });
      apiMocks.listDataTasks.mockResolvedValue([]);
      apiMocks.listBacktests.mockResolvedValue([]);
      apiMocks.listTradingTasks.mockResolvedValue([]);
      apiMocks.listNotifications.mockResolvedValue([]);
      apiMocks.overviewRecentFacts.mockResolvedValue({ orders: [], strategyIntents: [] });

      const workspace = mountWorkspace();
      await flushPromises();
      const firstSince = apiMocks.overviewRecentFacts.mock.calls[0]?.[0]?.since;

      nowSpy.mockReturnValue(secondNow);
      await workspace.loadOverview();
      await flushPromises();
      const secondSince = apiMocks.overviewRecentFacts.mock.calls[1]?.[0]?.since;

      expect(Date.parse(firstSince)).toBe(firstNow - 24 * 60 * 60 * 1000);
      expect(Date.parse(secondSince)).toBe(secondNow - 24 * 60 * 60 * 1000);
    } finally {
      nowSpy.mockRestore();
    }
  });
});

function mountWorkspace() {
  const holder: { workspace?: ReturnType<typeof useOverviewWorkspace> } = {};
  mount(
    {
      template: "<div />",
      setup() {
        holder.workspace = useOverviewWorkspace();
        return {};
      },
    },
    {
      global: {
        plugins: [i18n],
      },
    },
  );
  if (!holder.workspace) {
    throw new Error("overview workspace was not mounted");
  }
  return holder.workspace;
}

function expectRecentFactsWindowSince(beforeLoad: number, windowMs = 24 * 60 * 60 * 1000, callIndex = 0) {
  const options = apiMocks.overviewRecentFacts.mock.calls[callIndex]?.[0];
  expect(options).toEqual(expect.objectContaining({ limit: 8, since: expect.any(String) }));
  const since = Date.parse(options.since);
  expect(Number.isNaN(since)).toBe(false);
  expect(since).toBeGreaterThanOrEqual(beforeLoad - windowMs - 1000);
  expect(since).toBeLessThanOrEqual(Date.now() - windowMs + 1000);
}

function task(id: string, status: string, updatedAt: string, overrides: Record<string, unknown> = {}) {
  return {
    id,
    name: id,
    exchange: "binance",
    symbol: "BTCUSDT",
    interval: "1m",
    status,
    dataHealth: "ok",
    updatedAt,
    createdAt: updatedAt,
    realtimeEnabled: false,
    type: "paper",
    ...overrides,
  };
}

function overviewIntent(id: string, taskId: string, taskType: string, taskName: string, status: string, intentType: string, policy: string, createdAt: string) {
  return {
    id,
    taskId,
    taskType,
    taskName,
    exchange: "binance",
    symbol: "BTCUSDT",
    interval: "1m",
    strategyId: "ema-cross",
    intentType,
    policy,
    status,
    createdAt,
  };
}

function overviewOrder(id: string, taskId: string, taskType: string, taskName: string, side: string, price: string, quantity: string, status: string, occurredAt: string) {
  return {
    id,
    taskId,
    taskType,
    taskName,
    exchange: "binance",
    symbol: "BTCUSDT",
    interval: "1m",
    intentId: "si_bt_1",
    side,
    price,
    quantity,
    status,
    occurredAt,
  };
}

function isoMinutesAgo(minutes: number) {
  return new Date(Date.now() - minutes * 60 * 1000).toISOString();
}

function isoDaysAgo(days: number) {
  return new Date(Date.now() - days * 24 * 60 * 60 * 1000).toISOString();
}

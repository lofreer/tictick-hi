import { mount, flushPromises } from "@vue/test-utils";
import { describe, expect, it, vi, beforeEach } from "vitest";

import { i18n } from "@/i18n";
import { backtestsApi } from "@/services/api/backtests";
import { dataApi } from "@/services/api/data";
import { systemApi } from "@/services/api/system";
import { tradingApi } from "@/services/api/trading";
import { useOverviewWorkspace } from "@/composables/useOverviewWorkspace";

const apiMocks = vi.hoisted(() => ({
  listBacktests: vi.fn(),
  listDataTasks: vi.fn(),
  listNotifications: vi.fn(),
  listTradingTasks: vi.fn(),
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

vi.mock("@/services/api/trading", () => ({
  tradingApi: {
    listTasks: apiMocks.listTradingTasks,
  },
}));

describe("useOverviewWorkspace", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("loads real overview sources and derives summary state", async () => {
    apiMocks.systemHealth.mockResolvedValue({
      status: "warning",
      database: "ok",
      checkedAt: "2026-06-28T01:00:00Z",
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
      task("sync_1", "running", "2026-06-28T01:01:00Z", { realtimeEnabled: true }),
      task("sync_2", "failed", "2026-06-28T01:02:00Z"),
    ]);
    apiMocks.listBacktests.mockResolvedValue([
      task("bt_1", "succeeded", "2026-06-28T01:03:00Z", { name: "Baseline" }),
      task("bt_2", "failed", "2026-06-28T01:04:00Z", { name: "Failed test" }),
    ]);
    apiMocks.listTradingTasks.mockResolvedValue([
      task("tt_1", "running", "2026-06-28T01:05:00Z", { name: "Paper", type: "paper" }),
      task("tt_2", "failed", "2026-06-28T01:06:00Z", { name: "Live", type: "live" }),
    ]);
    apiMocks.listNotifications.mockResolvedValue([
      {
        id: "nt_1",
        channel: "ops",
        title: "failed alert",
        status: "failed",
        createdAt: "2026-06-28T01:07:00Z",
      },
    ]);

    const workspace = mountWorkspace();
    await flushPromises();

    expect(systemApi.health).toHaveBeenCalledTimes(1);
    expect(dataApi.listTasks).toHaveBeenCalledTimes(1);
    expect(backtestsApi.listBacktests).toHaveBeenCalledTimes(1);
    expect(tradingApi.listTasks).toHaveBeenCalledTimes(1);
    expect(systemApi.listNotifications).toHaveBeenCalledTimes(1);
    expect(workspace.hasLoaded.value).toBe(true);
    expect(workspace.summaryCards.value.find((card) => card.key === "sync")?.value).toBe(2);
    expect(workspace.summaryCards.value.find((card) => card.key === "workers")?.detail).toContain("过期锁 1");
    expect(workspace.alerts.value.map((alert) => alert.key)).toEqual([
      "health",
      "sync-failed",
      "backtests-failed",
      "trading-failed",
      "notifications-failed",
    ]);
    expect(workspace.recentActivities.value[0]).toEqual(
      expect.objectContaining({
        key: "notification-nt_1",
        status: "failed",
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

function task(id: string, status: string, updatedAt: string, overrides: Record<string, unknown> = {}) {
  return {
    id,
    name: id,
    exchange: "binance",
    symbol: "BTCUSDT",
    interval: "1m",
    status,
    updatedAt,
    createdAt: updatedAt,
    realtimeEnabled: false,
    type: "paper",
    ...overrides,
  };
}

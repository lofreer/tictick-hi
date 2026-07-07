import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useResearchWorkspace } from "@/composables/useResearchWorkspace";
import { i18n } from "@/i18n";
import type { DataSyncTask } from "@/types/app";

const dataApiMocks = vi.hoisted(() => ({
  getCandles: vi.fn(),
  listTasks: vi.fn(),
}));

const routerMocks = vi.hoisted(() => ({
  replace: vi.fn(),
  query: {} as Record<string, string>,
}));

vi.mock("@/services/api/data", () => ({
  dataApi: {
    getCandles: dataApiMocks.getCandles,
    listTasks: dataApiMocks.listTasks,
  },
}));

vi.mock("@/services/api/market", () => ({
  marketApi: {
    listInstrumentSyncStatuses: vi.fn().mockResolvedValue([]),
  },
}));

vi.mock("naive-ui", () => ({
  useDialog: () => ({ warning: vi.fn() }),
  useMessage: () => ({ error: vi.fn(), success: vi.fn() }),
}));

vi.mock("vue-router", () => ({
  useRoute: () => ({ query: routerMocks.query }),
  useRouter: () => ({ replace: routerMocks.replace }),
}));

describe("useResearchWorkspace data health filter", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    routerMocks.query = { dataHealth: "gap", exchange: "binance", interval: "5m", symbol: "BTCUSDT" };
    dataApiMocks.getCandles.mockResolvedValue(candleResult());
    dataApiMocks.listTasks.mockResolvedValue([
      dataSyncTask({ id: "dst_gap", dataHealth: "gap" }),
      dataSyncTask({ id: "dst_invalid", dataHealth: "invalid" }),
    ]);
  });

  it("filters loaded tasks by dataHealth query and preserves the query on context replace", async () => {
    const workspace = mountWorkspace();
    await flushPromises();

    expect(workspace.filteredTasks.value.map((task) => task.id)).toEqual(["dst_gap"]);
    expect(workspace.tasksEmptyTitle.value).toBe("当前筛选下暂无同步任务");

    workspace.interval.value = "15m";
    await flushPromises();

    expect(routerMocks.replace).toHaveBeenCalledWith({
      name: "research",
      query: { dataHealth: "gap", exchange: "binance", interval: "15m", symbol: "BTCUSDT" },
    });
  });
});

function mountWorkspace() {
  const holder: { workspace?: ReturnType<typeof useResearchWorkspace> } = {};
  mount(
    {
      template: "<div />",
      setup() {
        holder.workspace = useResearchWorkspace();
        return {};
      },
    },
    { global: { plugins: [i18n] } },
  );
  if (!holder.workspace) throw new Error("research workspace was not mounted");
  return holder.workspace;
}

function candleResult() {
  return {
    baseInterval: "1m",
    candles: [],
    coverage: { limitedByBaseWindow: false, requestedLimit: 1000, returnedCandles: 0 },
    gaps: [],
    health: "ok",
    pagination: { hasNext: false, hasPrevious: false },
    requestedInterval: "5m",
    source: "aggregated",
    window: { count: 0 },
  };
}

function dataSyncTask(overrides: Partial<DataSyncTask>): DataSyncTask {
  return {
    attemptCount: 0,
    createdAt: "2026-07-07T00:00:00Z",
    dataHealth: "ok",
    exchange: "binance",
    id: "dst_1",
    interval: "1m",
    marketStatus: "active",
    realtimeEnabled: false,
    status: "succeeded",
    symbol: "BTCUSDT",
    syncEnabled: false,
    updatedAt: "2026-07-07T00:00:00Z",
    ...overrides,
  };
}

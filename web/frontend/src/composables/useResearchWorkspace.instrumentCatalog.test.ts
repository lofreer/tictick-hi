import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useResearchWorkspace } from "@/composables/useResearchWorkspace";
import { i18n } from "@/i18n";
import { dataApi } from "@/services/api/data";
import { marketApi } from "@/services/api/market";
import type { DataSyncTask } from "@/types/app";

const dataApiMocks = vi.hoisted(() => ({
  createTask: vi.fn(),
  deleteTask: vi.fn(),
  getCandles: vi.fn(),
  getTaskGaps: vi.fn(),
  listTasks: vi.fn(),
  repairTaskGap: vi.fn(),
  repairTaskGaps: vi.fn(),
  retryTask: vi.fn(),
  setRealtime: vi.fn(),
  setSync: vi.fn(),
}));

const marketApiMocks = vi.hoisted(() => ({
  listInstruments: vi.fn(),
}));

const messageMocks = vi.hoisted(() => ({
  error: vi.fn(),
  success: vi.fn(),
}));

const routerMocks = vi.hoisted(() => ({
  replace: vi.fn(),
  query: {} as Record<string, string>,
}));

vi.mock("@/services/api/data", () => ({
  dataApi: dataApiMocks,
}));

vi.mock("@/services/api/market", () => ({
  marketApi: marketApiMocks,
}));

vi.mock("naive-ui", () => ({
  useDialog: () => ({ warning: vi.fn() }),
  useMessage: () => messageMocks,
}));

vi.mock("vue-router", () => ({
  useRoute: () => ({ query: routerMocks.query }),
  useRouter: () => ({ replace: routerMocks.replace }),
}));

describe("useResearchWorkspace instrument catalog validation", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    routerMocks.query = { exchange: "binance", symbol: "BTCUSDT", interval: "5m" };
    dataApiMocks.listTasks.mockResolvedValue([]);
    dataApiMocks.getCandles.mockResolvedValue({
      candles: [],
      source: "aggregated",
      requestedInterval: "5m",
      baseInterval: "1m",
      health: "ok",
      gaps: [],
      coverage: {
        requestedLimit: 1000,
        returnedCandles: 0,
        limitedByBaseWindow: false,
      },
      window: { count: 0 },
      pagination: {
        hasPrevious: false,
        hasNext: false,
      },
    });
    dataApiMocks.createTask.mockResolvedValue({ id: "dst_created" });
  });

  it("blocks data sync task creation when the exact catalog symbol is inactive", async () => {
    marketApiMocks.listInstruments.mockResolvedValueOnce([
      {
        exchange: "binance",
        symbol: "SOLUSDT",
        baseAsset: "SOL",
        quoteAsset: "USDT",
        instrumentType: "spot",
        status: "inactive",
        searchPriority: 20,
      },
    ]);
    const workspace = mountWorkspace();
    await flushPromises();

    workspace.openCreateTask();
    workspace.createForm.symbol = "SOLUSDT";
    await flushPromises();

    await workspace.createTask();
    await flushPromises();

    expect(marketApi.listInstruments).toHaveBeenCalledWith({
      exchange: "binance",
      limit: 1,
      q: "SOLUSDT",
      status: "all",
    });
    expect(dataApi.createTask).not.toHaveBeenCalled();
    expect(messageMocks.error).toHaveBeenCalledWith("交易对已不在当前交易所 active 目录中，请刷新交易对或选择仍可用的标的。");
  });

  it("blocks starting sync when an existing task market is not active", async () => {
    const workspace = mountWorkspace();
    await flushPromises();

    await workspace.toggleSync(
      dataSyncTask({
        id: "dst_inactive",
        status: "paused",
        syncEnabled: false,
        marketStatus: "inactive",
      }),
    );

    expect(dataApi.setSync).not.toHaveBeenCalled();
    expect(messageMocks.error).toHaveBeenCalledWith("任务交易对不在当前交易所 active 目录中，不能启动同步或实时。");
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
    {
      global: {
        plugins: [i18n],
      },
    },
  );
  if (!holder.workspace) {
    throw new Error("research workspace was not mounted");
  }
  return holder.workspace;
}

function dataSyncTask(overrides: Partial<DataSyncTask>): DataSyncTask {
  return {
    id: "sync_1",
    exchange: "binance",
    symbol: "BTCUSDT",
    interval: "1m",
    realtimeEnabled: false,
    syncEnabled: true,
    status: "succeeded",
    marketStatus: "active",
    dataHealth: "ok",
    attemptCount: 0,
    createdAt: "2026-06-28T00:00:00Z",
    updatedAt: "2026-06-28T00:00:00Z",
    ...overrides,
  };
}

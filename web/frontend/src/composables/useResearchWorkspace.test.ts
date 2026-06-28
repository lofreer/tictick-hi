import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useResearchWorkspace } from "@/composables/useResearchWorkspace";
import { i18n } from "@/i18n";
import { dataApi } from "@/services/api/data";
import type { DataSyncTask } from "@/types/app";

const dataApiMocks = vi.hoisted(() => ({
  createTask: vi.fn(),
  deleteTask: vi.fn(),
  getCandles: vi.fn(),
  getTaskGaps: vi.fn(),
  listTasks: vi.fn(),
  repairTaskGaps: vi.fn(),
  retryTask: vi.fn(),
  setRealtime: vi.fn(),
  setSync: vi.fn(),
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

vi.mock("naive-ui", () => ({
  useDialog: () => ({ warning: vi.fn() }),
  useMessage: () => messageMocks,
}));

vi.mock("vue-router", () => ({
  useRoute: () => ({ query: routerMocks.query }),
  useRouter: () => ({ replace: routerMocks.replace }),
}));

describe("useResearchWorkspace", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    routerMocks.query = { exchange: "binance", symbol: "BTCUSDT", interval: "5m" };
    dataApiMocks.listTasks.mockResolvedValue([]);
    dataApiMocks.getCandles.mockResolvedValue(candleResult({ gaps: [] }));
    dataApiMocks.getTaskGaps.mockResolvedValue({
      taskId: "dst_1",
      gaps: [{ from: "2026-06-27T03:02:00Z", to: "2026-06-27T03:03:00Z", missingCandles: 1 }],
      limited: false,
      totalCount: 1,
      returnedCount: 1,
      repairLimit: 20,
    });
    dataApiMocks.createTask.mockResolvedValue({ id: "dst_repair" });
    dataApiMocks.repairTaskGaps.mockResolvedValue({
      sourceTaskId: "dst_1",
      createdTasks: [{ id: "dst_repair_1" }],
      skippedExisting: 0,
      limited: false,
      totalCount: 1,
      repairLimit: 20,
    });
    dataApiMocks.setSync.mockResolvedValue({ id: "dst_repair" });
  });

  it("queues a sync task for the first chart gap using the base interval", async () => {
    dataApiMocks.getCandles.mockResolvedValue(
      candleResult({
        baseInterval: "1m",
        gaps: [{ from: "2026-06-28T00:01:00Z", to: "2026-06-28T00:03:00Z", missingCandles: 2 }],
      }),
    );

    const workspace = mountWorkspace();
    await flushPromises();

    expect(workspace.canRepairGap.value).toBe(true);

    await workspace.repairFirstGap();
    await flushPromises();

    expect(dataApi.createTask).toHaveBeenCalledWith({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      startTime: "2026-06-28T00:01:00Z",
      endTime: "2026-06-28T00:03:00Z",
    });
    expect(dataApi.setSync).toHaveBeenCalledWith("dst_repair", true);
    expect(dataApi.listTasks).toHaveBeenCalledTimes(2);
    expect(messageMocks.success).toHaveBeenCalledWith("缺口修复任务已排队。");
  });

  it("loads arbitrary valid symbols from the route after normalizing input", async () => {
    routerMocks.query = { exchange: "binance", symbol: "solusdt", interval: "5m" };

    mountWorkspace();
    await flushPromises();

    expect(dataApi.getCandles).toHaveBeenCalledWith({
      exchange: "binance",
      symbol: "SOLUSDT",
      interval: "5m",
    });
  });

  it("loads and navigates adjacent candle windows", async () => {
    routerMocks.query = {
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "5m",
      from: "2026-06-28T00:00:00Z",
      to: "2026-06-28T01:00:00Z",
    };
    dataApiMocks.getCandles.mockResolvedValueOnce(
      candleResult({
        pagination: {
          hasPrevious: true,
          hasNext: true,
          previousFrom: "2026-06-27T23:00:00Z",
          previousTo: "2026-06-27T23:55:00Z",
          nextFrom: "2026-06-28T01:05:00Z",
          nextTo: "2026-06-28T02:00:00Z",
        },
      }),
    );

    const workspace = mountWorkspace();
    await flushPromises();

    expect(dataApi.getCandles).toHaveBeenCalledWith({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "5m",
      from: "2026-06-28T00:00:00Z",
      to: "2026-06-28T01:00:00Z",
    });
    expect(workspace.canLoadNextCandles.value).toBe(true);

    workspace.loadNextCandles();
    await flushPromises();

    expect(routerMocks.replace).toHaveBeenLastCalledWith({
      name: "research",
      query: {
        exchange: "binance",
        symbol: "BTCUSDT",
        interval: "5m",
        from: "2026-06-28T01:05:00Z",
        to: "2026-06-28T02:00:00Z",
      },
    });
    expect(dataApi.getCandles).toHaveBeenLastCalledWith({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "5m",
      from: "2026-06-28T01:05:00Z",
      to: "2026-06-28T02:00:00Z",
    });
  });

  it("does not call the candles API for a symbol that does not match the selected exchange", async () => {
    const workspace = mountWorkspace();
    await flushPromises();
    dataApiMocks.getCandles.mockClear();

    workspace.symbol.value = "BTC-USDT";
    await flushPromises();

    expect(dataApi.getCandles).not.toHaveBeenCalled();
    expect(workspace.candlesError.value).toBe("交易对格式不符合当前交易所。");
  });

  it("blocks data sync task creation when the symbol format does not match the exchange", async () => {
    const workspace = mountWorkspace();
    await flushPromises();
    dataApiMocks.createTask.mockClear();

    workspace.openCreateTask();
    workspace.createForm.exchange = "okx";
    await flushPromises();
    workspace.createForm.symbol = "BTCUSDT";
    await flushPromises();

    expect(workspace.canCreateTask.value).toBe(false);

    await workspace.createTask();

    expect(dataApi.createTask).not.toHaveBeenCalled();
    expect(messageMocks.error).toHaveBeenCalledWith("交易对格式不符合当前交易所。");
  });

  it("creates data sync tasks for arbitrary valid normalized symbols", async () => {
    const workspace = mountWorkspace();
    await flushPromises();
    dataApiMocks.createTask.mockClear();

    workspace.openCreateTask();
    workspace.createForm.symbol = " solusdt ";
    await flushPromises();

    await workspace.createTask();
    await flushPromises();

    expect(dataApi.createTask).toHaveBeenCalledWith({
      exchange: "binance",
      symbol: "SOLUSDT",
      interval: "5m",
      startTime: undefined,
      endTime: undefined,
    });
  });

  it("does not create a repair task when the current chart has no gaps", async () => {
    const workspace = mountWorkspace();
    await flushPromises();

    expect(workspace.canRepairGap.value).toBe(false);

    await workspace.repairFirstGap();

    expect(dataApi.createTask).not.toHaveBeenCalled();
    expect(dataApi.setSync).not.toHaveBeenCalled();
    expect(messageMocks.error).toHaveBeenCalledWith("当前没有可修复缺口。");
  });

  it("queues backend repair tasks for a data sync task gap summary", async () => {
    const workspace = mountWorkspace();
    await flushPromises();

    await workspace.repairTaskGaps(dataSyncTask({ id: "dst_1" }));
    await flushPromises();

    expect(dataApi.repairTaskGaps).toHaveBeenCalledWith("dst_1");
    expect(dataApi.listTasks).toHaveBeenCalledTimes(2);
    expect(messageMocks.success).toHaveBeenCalledWith("已排队 1 个缺口修复任务。");
  });

  it("loads task gap details for the gap modal", async () => {
    const workspace = mountWorkspace();
    await flushPromises();

    await workspace.viewTaskGaps(dataSyncTask({ id: "dst_1" }));
    await flushPromises();

    expect(dataApi.getTaskGaps).toHaveBeenCalledWith("dst_1");
    expect(workspace.gapDetailsModalOpen.value).toBe(true);
    expect(workspace.gapDetails.value).toMatchObject({
      taskId: "dst_1",
      gaps: [{ missingCandles: 1 }],
      limited: false,
    });
    expect(workspace.gapDetailsError.value).toBe("");
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

function candleResult(overrides: Record<string, unknown>) {
  return {
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
    window: {
      count: 0,
    },
    pagination: {
      hasPrevious: false,
      hasNext: false,
    },
    ...overrides,
  };
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
    dataHealth: "gap",
    gapSummary: {
      count: 1,
      firstGap: {
        from: "2026-06-27T03:02:00Z",
        to: "2026-06-27T03:03:00Z",
        missingCandles: 1,
      },
    },
    attemptCount: 0,
    createdAt: "2026-06-28T00:00:00Z",
    updatedAt: "2026-06-28T00:00:00Z",
    ...overrides,
  };
}

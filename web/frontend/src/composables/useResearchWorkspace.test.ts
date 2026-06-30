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
  repairMarketCandleGap: vi.fn(),
  repairTaskGap: vi.fn(),
  repairTaskGaps: vi.fn(),
  retryTask: vi.fn(),
  setRealtime: vi.fn(),
  setSync: vi.fn(),
}));

const messageMocks = vi.hoisted(() => ({
  error: vi.fn(),
  success: vi.fn(),
}));

const marketApiMocks = vi.hoisted(() => ({
  listInstruments: vi.fn(),
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
    dataApiMocks.repairMarketCandleGap.mockResolvedValue({
      sourceTaskId: "",
      createdTasks: [{ id: "dst_market_repair_1" }],
      skippedExisting: 0,
      limited: false,
      totalCount: 1,
      repairLimit: 1,
    });
    dataApiMocks.repairTaskGap.mockResolvedValue({
      sourceTaskId: "dst_source_1",
      createdTasks: [{ id: "dst_repair_1" }],
      skippedExisting: 0,
      limited: false,
      totalCount: 1,
      repairLimit: 1,
    });
    dataApiMocks.repairTaskGaps.mockResolvedValue({
      sourceTaskId: "dst_1",
      createdTasks: [{ id: "dst_repair_1" }],
      skippedExisting: 0,
      limited: false,
      totalCount: 1,
      repairLimit: 20,
    });
    dataApiMocks.setSync.mockResolvedValue({ id: "dst_repair" });
    marketApiMocks.listInstruments.mockResolvedValue([
      {
        exchange: "binance",
        symbol: "BTCUSDT",
        baseAsset: "BTC",
        quoteAsset: "USDT",
        instrumentType: "spot",
        status: "active",
        searchPriority: 1,
      },
    ]);
  });

  it("repairs the first chart gap through validated market gap repair when no source task is selected", async () => {
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

    expect(dataApi.repairMarketCandleGap).toHaveBeenCalledWith({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      from: "2026-06-28T00:01:00Z",
      to: "2026-06-28T00:03:00Z",
    });
    expect(dataApi.createTask).not.toHaveBeenCalled();
    expect(dataApi.setSync).not.toHaveBeenCalled();
    expect(dataApi.listTasks).toHaveBeenCalledTimes(2);
    expect(messageMocks.success).toHaveBeenCalledWith("缺口修复任务已排队。");
  });

  it("marks candle gaps on the first visible candle after the gap", async () => {
    dataApiMocks.getCandles.mockResolvedValue(
      candleResult({
        candles: [
          chartCandle("2026-06-28T00:00:00Z"),
          chartCandle("2026-06-28T00:03:00Z"),
        ],
        gaps: [{ from: "2026-06-28T00:01:00Z", to: "2026-06-28T00:03:00Z", missingCandles: 2 }],
      }),
    );

    const workspace = mountWorkspace();
    await flushPromises();

    expect(workspace.chartMarkers.value).toEqual([
      {
        id: "candle-gap-2026-06-28T00:01:00Z-2026-06-28T00:03:00Z-0",
        time: utcSeconds("2026-06-28T00:03:00Z"),
        position: "aboveBar",
        shape: "square",
        color: "#f7a600",
        text: "缺 2 根",
        size: 1.1,
      },
    ]);
  });

  it("anchors boundary candle gaps to the nearest visible candle", async () => {
    dataApiMocks.getCandles.mockResolvedValue(
      candleResult({
        candles: [
          chartCandle("2026-06-28T00:00:00Z"),
          chartCandle("2026-06-28T00:05:00Z"),
        ],
        gaps: [
          { from: "2026-06-27T23:58:00Z", to: "2026-06-28T00:00:00Z", missingCandles: 2 },
          { from: "2026-06-28T00:06:00Z", to: "2026-06-28T00:08:00Z", missingCandles: 2 },
        ],
      }),
    );

    const workspace = mountWorkspace();
    await flushPromises();

    expect(workspace.chartMarkers.value.map((marker) => marker.time)).toEqual([
      utcSeconds("2026-06-28T00:00:00Z"),
      utcSeconds("2026-06-28T00:05:00Z"),
    ]);
    expect(workspace.chartMarkers.value.map((marker) => marker.text)).toEqual(["缺 2 根", "缺 2 根"]);
  });

  it("repairs the first chart gap through the selected source task", async () => {
    dataApiMocks.getCandles.mockResolvedValue(
      candleResult({
        baseInterval: "1m",
        gaps: [{ from: "2026-06-28T00:01:00Z", to: "2026-06-28T00:03:00Z", missingCandles: 2 }],
      }),
    );

    const workspace = mountWorkspace();
    await flushPromises();

    workspace.selectTask(dataSyncTask({ id: "dst_source_1", interval: "1m" }));
    await flushPromises();

    await workspace.repairFirstGap();
    await flushPromises();

    expect(dataApi.repairTaskGap).toHaveBeenCalledWith("dst_source_1", {
      from: "2026-06-28T00:01:00Z",
      to: "2026-06-28T00:03:00Z",
    });
    expect(dataApi.createTask).not.toHaveBeenCalled();
    expect(dataApi.setSync).not.toHaveBeenCalled();
    expect(dataApi.listTasks).toHaveBeenCalledTimes(2);
    expect(messageMocks.success).toHaveBeenCalledWith("缺口修复任务已排队。");
  });

  it("keeps the selected source task when switching to an aggregated chart interval", async () => {
    dataApiMocks.getCandles.mockResolvedValue(
      candleResult({
        baseInterval: "1m",
        gaps: [{ from: "2026-06-28T00:01:00Z", to: "2026-06-28T00:03:00Z", missingCandles: 2 }],
      }),
    );

    const workspace = mountWorkspace();
    await flushPromises();

    workspace.selectTask(dataSyncTask({ id: "dst_source_1", interval: "1m" }));
    await flushPromises();
    workspace.interval.value = "5m";
    await flushPromises();

    await workspace.repairFirstGap();
    await flushPromises();

    expect(dataApi.repairTaskGap).toHaveBeenCalledWith("dst_source_1", {
      from: "2026-06-28T00:01:00Z",
      to: "2026-06-28T00:03:00Z",
    });
    expect(dataApi.createTask).not.toHaveBeenCalled();
    expect(dataApi.setSync).not.toHaveBeenCalled();
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

  it("prefers opaque candle cursors when navigating adjacent windows", async () => {
    routerMocks.query = {
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "5m",
      cursor: "initial_cursor",
    };
    dataApiMocks.getCandles.mockResolvedValueOnce(
      candleResult({
        pagination: {
          hasPrevious: true,
          hasNext: true,
          previousCursor: "previous_cursor",
          nextCursor: "next_cursor",
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
      cursor: "initial_cursor",
    });

    workspace.loadNextCandles();
    await flushPromises();

    expect(routerMocks.replace).toHaveBeenLastCalledWith({
      name: "research",
      query: {
        exchange: "binance",
        symbol: "BTCUSDT",
        interval: "5m",
        cursor: "next_cursor",
      },
    });
    expect(dataApi.getCandles).toHaveBeenLastCalledWith({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "5m",
      cursor: "next_cursor",
    });
  });

  it("loads a fixed time range through from and to query parameters", async () => {
    const workspace = mountWorkspace();
    await flushPromises();
    dataApiMocks.getCandles.mockClear();

    workspace.applyTimeRange("6h", new Date("2026-06-28T12:00:00.000Z"));
    await flushPromises();

    expect(routerMocks.replace).toHaveBeenLastCalledWith({
      name: "research",
      query: {
        exchange: "binance",
        symbol: "BTCUSDT",
        interval: "5m",
        from: "2026-06-28T06:00:00.000Z",
        to: "2026-06-28T12:00:00.000Z",
      },
    });
    expect(dataApi.getCandles).toHaveBeenLastCalledWith({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "5m",
      from: "2026-06-28T06:00:00.000Z",
      to: "2026-06-28T12:00:00.000Z",
    });
  });

  it("returns to the latest candle window from an opaque cursor", async () => {
    routerMocks.query = {
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "5m",
      cursor: "window_cursor",
    };
    const workspace = mountWorkspace();
    await flushPromises();
    dataApiMocks.getCandles.mockClear();

    workspace.applyTimeRange("latest", new Date("2026-06-28T12:00:00.000Z"));
    await flushPromises();

    expect(routerMocks.replace).toHaveBeenLastCalledWith({
      name: "research",
      query: {
        exchange: "binance",
        symbol: "BTCUSDT",
        interval: "5m",
      },
    });
    expect(dataApi.getCandles).toHaveBeenLastCalledWith({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "5m",
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

  it("blocks data sync task creation when the create window is not forward", async () => {
    const workspace = mountWorkspace();
    await flushPromises();

    workspace.openCreateTask();
    workspace.createForm.startTime = Date.parse("2026-01-01T00:01:00Z");
    workspace.createForm.endTime = Date.parse("2026-01-01T00:00:00Z");
    expect(workspace.canCreateTask.value).toBe(false);

    await workspace.createTask();
    expect(dataApi.createTask).not.toHaveBeenCalled();
    expect(messageMocks.error).toHaveBeenCalledWith("结束时间必须晚于开始时间。");
  });

  it("creates data sync tasks for normalized symbols that exactly match the active catalog", async () => {
    marketApiMocks.listInstruments.mockResolvedValueOnce([
      {
        exchange: "binance",
        symbol: "SOLUSDT",
        baseAsset: "SOL",
        quoteAsset: "USDT",
        instrumentType: "spot",
        status: "active",
        searchPriority: 20,
      },
    ]);
    const workspace = mountWorkspace();
    await flushPromises();
    dataApiMocks.createTask.mockClear();

    workspace.openCreateTask();
    workspace.createForm.symbol = " solusdt ";
    await flushPromises();

    await workspace.createTask();
    await flushPromises();

    expect(marketApi.listInstruments).toHaveBeenCalledWith({ exchange: "binance", limit: 1, q: "SOLUSDT", status: "all" });
    expect(dataApi.createTask).toHaveBeenCalledWith({
      exchange: "binance",
      symbol: "SOLUSDT",
      interval: "5m",
      startTime: undefined,
      endTime: undefined,
    });
  });

  it("blocks data sync task creation when the normalized symbol is missing from the active catalog", async () => {
    marketApiMocks.listInstruments.mockResolvedValueOnce([]);
    const workspace = mountWorkspace();
    await flushPromises();
    dataApiMocks.createTask.mockClear();

    workspace.openCreateTask();
    workspace.createForm.symbol = "SOLUSDT";
    await flushPromises();

    await workspace.createTask();
    await flushPromises();

    expect(marketApi.listInstruments).toHaveBeenCalledWith({ exchange: "binance", limit: 1, q: "SOLUSDT", status: "all" });
    expect(dataApi.createTask).not.toHaveBeenCalled();
    expect(messageMocks.error).toHaveBeenCalledWith("交易对不在当前交易所可用目录中，请先刷新交易对或更换标的。");
  });

  it("blocks data sync task creation when the active catalog cannot be validated", async () => {
    marketApiMocks.listInstruments.mockRejectedValueOnce(new Error("network"));
    const workspace = mountWorkspace();
    await flushPromises();
    dataApiMocks.createTask.mockClear();

    workspace.openCreateTask();
    await flushPromises();

    await workspace.createTask();
    await flushPromises();

    expect(dataApi.createTask).not.toHaveBeenCalled();
    expect(messageMocks.error).toHaveBeenCalledWith("校验交易对目录失败，请稍后重试。");
  });

  it("does not create a repair task when the current chart has no gaps", async () => {
    const workspace = mountWorkspace();
    await flushPromises();

    expect(workspace.canRepairGap.value).toBe(false);

    await workspace.repairFirstGap();

    expect(dataApi.createTask).not.toHaveBeenCalled();
    expect(dataApi.repairTaskGap).not.toHaveBeenCalled();
    expect(dataApi.repairMarketCandleGap).not.toHaveBeenCalled();
    expect(dataApi.setSync).not.toHaveBeenCalled();
    expect(messageMocks.error).toHaveBeenCalledWith("当前没有可修复缺口。");
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

function chartCandle(time: string) {
  return {
    time: utcSeconds(time),
    open: 100,
    high: 110,
    low: 95,
    close: 104,
    volume: 1200,
  };
}

function utcSeconds(value: string) {
  return Math.floor(Date.parse(value) / 1000);
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

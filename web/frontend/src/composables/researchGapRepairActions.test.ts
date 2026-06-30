import { beforeEach, describe, expect, it, vi } from "vitest";

import { repairChartGap } from "@/composables/researchGapRepairActions";
import { dataApi } from "@/services/api/data";
import type { DataSyncGapRepairResult, DataSyncTask } from "@/types/app";

const dataApiMocks = vi.hoisted(() => ({
  repairMarketCandleGap: vi.fn(),
  repairTaskGap: vi.fn(),
}));

vi.mock("@/services/api/data", () => ({
  dataApi: dataApiMocks,
}));

describe("repairChartGap", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("returns market repair metadata and refreshes tasks and candles", async () => {
    const result = repairResult([dataSyncTask({ id: "dst_market_repair_1" })]);
    const loadCandles = vi.fn().mockResolvedValue(undefined);
    const loadTasks = vi.fn().mockResolvedValue(undefined);
    const onSuccess = vi.fn();
    dataApiMocks.repairMarketCandleGap.mockResolvedValue(result);

    await expect(repairChartGap({
      exchange: "binance",
      gap: { from: "2026-06-28T00:01:00Z", to: "2026-06-28T00:03:00Z", missingCandles: 2 },
      loadCandles,
      loadTasks,
      onSuccess,
      repairInterval: "1m",
      sourceTask: null,
      symbol: "BTCUSDT",
    })).resolves.toBe(result);

    expect(dataApi.repairMarketCandleGap).toHaveBeenCalledWith({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      from: "2026-06-28T00:01:00Z",
      to: "2026-06-28T00:03:00Z",
    });
    expect(loadTasks).toHaveBeenCalledTimes(1);
    expect(loadCandles).toHaveBeenCalledTimes(1);
    expect(onSuccess).toHaveBeenCalledWith("research.gapRepairQueued");
  });

  it("returns source task repair metadata without using market repair", async () => {
    const sourceTask = dataSyncTask({ id: "dst_source_1" });
    const result = repairResult([], { skippedExisting: 1 });
    dataApiMocks.repairTaskGap.mockResolvedValue(result);

    await expect(repairChartGap({
      exchange: "binance",
      gap: { from: "2026-06-28T00:01:00Z", to: "2026-06-28T00:03:00Z", missingCandles: 2 },
      loadCandles: vi.fn().mockResolvedValue(undefined),
      loadTasks: vi.fn().mockResolvedValue(undefined),
      onSuccess: vi.fn(),
      repairInterval: "1m",
      sourceTask,
      symbol: "BTCUSDT",
    })).resolves.toBe(result);

    expect(dataApi.repairTaskGap).toHaveBeenCalledWith("dst_source_1", {
      from: "2026-06-28T00:01:00Z",
      to: "2026-06-28T00:03:00Z",
    });
    expect(dataApi.repairMarketCandleGap).not.toHaveBeenCalled();
  });
});

function repairResult(
  createdTasks: DataSyncTask[],
  overrides: Partial<DataSyncGapRepairResult> = {},
): DataSyncGapRepairResult {
  return {
    sourceTaskId: "",
    createdTasks,
    skippedExisting: 0,
    limited: false,
    totalCount: createdTasks.length,
    repairLimit: 1,
    ...overrides,
  };
}

function dataSyncTask(overrides: Partial<DataSyncTask>): DataSyncTask {
  return {
    id: "dst_repair",
    exchange: "binance",
    symbol: "BTCUSDT",
    interval: "1m",
    realtimeEnabled: false,
    syncEnabled: true,
    status: "pending",
    marketStatus: "active",
    dataHealth: "syncing",
    attemptCount: 0,
    createdAt: "2026-06-28T00:00:00Z",
    updatedAt: "2026-06-28T00:00:00Z",
    ...overrides,
  };
}

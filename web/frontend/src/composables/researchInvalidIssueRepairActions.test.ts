import { beforeEach, describe, expect, it, vi } from "vitest";

import { repairChartInvalidIssue } from "@/composables/researchInvalidIssueRepairActions";
import { dataApi } from "@/services/api/data";
import type { DataSyncGapRepairResult, DataSyncTask } from "@/types/app";

const dataApiMocks = vi.hoisted(() => ({
  repairMarketCandleInvalidIssues: vi.fn(),
}));

vi.mock("@/services/api/data", () => ({
  dataApi: dataApiMocks,
}));

describe("repairChartInvalidIssue", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("queues repair for the chart invalid issue open time and refreshes tasks and candles", async () => {
    const result = repairResult([dataSyncTask({ id: "dst_invalid_repair_1" })]);
    const loadCandles = vi.fn().mockResolvedValue(undefined);
    const loadTasks = vi.fn().mockResolvedValue(undefined);
    const onSuccess = vi.fn();
    dataApiMocks.repairMarketCandleInvalidIssues.mockResolvedValue(result);

    await expect(repairChartInvalidIssue({
      exchange: "binance",
      interval: "1m",
      issue: {
        code: "invalid_close_price",
        message: "close price value must be positive",
        openTime: "2026-06-28T00:01:00Z",
      },
      loadCandles,
      loadTasks,
      onSuccess,
      symbol: "BTCUSDT",
    })).resolves.toBe(result);

    expect(dataApi.repairMarketCandleInvalidIssues).toHaveBeenCalledWith({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      openTimes: ["2026-06-28T00:01:00Z"],
    });
    expect(loadTasks).toHaveBeenCalledTimes(1);
    expect(loadCandles).toHaveBeenCalledTimes(1);
    expect(onSuccess).toHaveBeenCalledWith("research.invalidIssueRepairQueued", { count: 1 });
  });

  it("uses already queued feedback when the repair task exists", async () => {
    dataApiMocks.repairMarketCandleInvalidIssues.mockResolvedValue(repairResult([], { skippedExisting: 1 }));
    const onSuccess = vi.fn();

    await repairChartInvalidIssue({
      exchange: "binance",
      interval: "1m",
      issue: { code: "invalid_open_price", message: "", openTime: "2026-06-28T00:01:00Z" },
      loadCandles: vi.fn().mockResolvedValue(undefined),
      loadTasks: vi.fn().mockResolvedValue(undefined),
      onSuccess,
      symbol: "BTCUSDT",
    });

    expect(onSuccess).toHaveBeenCalledWith("research.invalidIssueRepairAlreadyQueued", undefined);
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
    repairLimit: 20,
    ...overrides,
  };
}

function dataSyncTask(overrides: Partial<DataSyncTask>): DataSyncTask {
  return {
    id: "dst_invalid_repair",
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

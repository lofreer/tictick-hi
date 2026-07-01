import { ref } from "vue";
import { describe, expect, it, vi } from "vitest";

import { refreshAfterRepairPolling } from "@/composables/researchRepairPollingRefresh";
import type { DataSyncTask } from "@/types/app";

describe("refreshAfterRepairPolling", () => {
  it("refreshes candles and the open source task gap details", async () => {
    const task = dataSyncTask("dst_1");
    const loadCandles = vi.fn().mockResolvedValue(undefined);
    const viewTaskGaps = vi.fn().mockResolvedValue(undefined);

    await refreshAfterRepairPolling({
      gapDetailsTask: ref(task),
      loadCandles,
      task,
      viewTaskGaps,
    });

    expect(loadCandles).toHaveBeenCalledTimes(1);
    expect(viewTaskGaps).toHaveBeenCalledWith(task, { resetRepairResult: false });
  });

  it("does not overwrite a gap modal that has switched to another task", async () => {
    const loadCandles = vi.fn().mockResolvedValue(undefined);
    const viewTaskGaps = vi.fn().mockResolvedValue(undefined);

    await refreshAfterRepairPolling({
      gapDetailsTask: ref(dataSyncTask("dst_2")),
      loadCandles,
      task: dataSyncTask("dst_1"),
      viewTaskGaps,
    });

    expect(loadCandles).toHaveBeenCalledTimes(1);
    expect(viewTaskGaps).not.toHaveBeenCalled();
  });
});

function dataSyncTask(id: string): DataSyncTask {
  return {
    id,
    exchange: "binance",
    symbol: "BTCUSDT",
    interval: "1m",
    realtimeEnabled: false,
    syncEnabled: true,
    status: "succeeded",
    marketStatus: "active",
    dataHealth: "ok",
    attemptCount: 0,
    createdAt: "2026-06-27T03:00:00Z",
    updatedAt: "2026-06-27T03:00:00Z",
  };
}

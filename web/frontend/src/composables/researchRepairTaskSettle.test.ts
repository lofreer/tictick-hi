import { describe, expect, it } from "vitest";

import { repairTasksSettled, repairTaskSettleKey } from "@/composables/researchRepairTaskSettle";
import type { DataSyncTask } from "@/types/app";

describe("researchRepairTaskSettle", () => {
  it("requires every watched repair task to be present and terminal", () => {
    expect(repairTasksSettled([task("dst_1", "succeeded"), task("dst_2", "failed")], ["dst_1", "dst_2"])).toBe(true);
    expect(repairTasksSettled([task("dst_1", "succeeded"), task("dst_2", "running")], ["dst_1", "dst_2"])).toBe(false);
    expect(repairTasksSettled([task("dst_1", "succeeded")], ["dst_1", "dst_2"])).toBe(false);
    expect(repairTasksSettled(undefined, ["dst_1"])).toBe(false);
    expect(repairTasksSettled([task("dst_1", "succeeded")], [])).toBe(false);
  });

  it("builds a stable refresh key from watched task ids", () => {
    expect(repairTaskSettleKey(["dst_1", "dst_2"])).toBe("dst_1|dst_2");
  });
});

function task(id: string, status: DataSyncTask["status"]): DataSyncTask {
  return {
    id,
    exchange: "binance",
    symbol: "BTCUSDT",
    interval: "1m",
    realtimeEnabled: false,
    syncEnabled: true,
    status,
    marketStatus: "active",
    dataHealth: status === "succeeded" ? "ok" : "syncing",
    attemptCount: 0,
    createdAt: "2026-06-27T03:00:00Z",
    updatedAt: "2026-06-27T03:00:00Z",
  };
}

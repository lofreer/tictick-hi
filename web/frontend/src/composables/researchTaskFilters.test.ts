import { describe, expect, it } from "vitest";

import { dataHealthFilterFromQuery, dataHealthQueryValue, taskMatchesDataHealthFilter } from "@/composables/researchTaskFilters";
import type { DataSyncTask } from "@/types/app";

describe("research task filters", () => {
  it("normalizes data health query values", () => {
    expect(dataHealthFilterFromQuery("gap")).toBe("gap");
    expect(dataHealthFilterFromQuery("invalid")).toBe("invalid");
    expect(dataHealthFilterFromQuery("failed")).toBe("failed");
    expect(dataHealthFilterFromQuery("paused")).toBe("paused");
    expect(dataHealthFilterFromQuery("stale")).toBe("all");
    expect(dataHealthFilterFromQuery(["gap"])).toBe("all");
    expect(dataHealthQueryValue("all")).toBeUndefined();
    expect(dataHealthQueryValue("invalid")).toBe("invalid");
  });

  it("matches exact task data health while keeping all as a pass-through", () => {
    expect(taskMatchesDataHealthFilter(task("gap"), "gap")).toBe(true);
    expect(taskMatchesDataHealthFilter(task("invalid"), "invalid")).toBe(true);
    expect(taskMatchesDataHealthFilter(task("failed"), "gap")).toBe(false);
    expect(taskMatchesDataHealthFilter(task("retrying"), "all")).toBe(true);
  });
});

function task(dataHealth: DataSyncTask["dataHealth"]): DataSyncTask {
  return {
    attemptCount: 0,
    createdAt: "2026-07-07T00:00:00Z",
    dataHealth,
    exchange: "binance",
    id: `dst_${dataHealth}`,
    interval: "1m",
    marketStatus: "active",
    realtimeEnabled: false,
    status: dataHealth === "failed" ? "failed" : "succeeded",
    symbol: "BTCUSDT",
    syncEnabled: false,
    updatedAt: "2026-07-07T00:00:00Z",
  };
}

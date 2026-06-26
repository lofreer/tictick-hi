import { afterEach, describe, expect, it, vi } from "vitest";

import { tradingApi } from "@/services/api/trading";

describe("trading api", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("posts create trading task requests", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify({
          id: "tt_1",
          name: "Paper EMA",
          type: "paper",
          exchange: "binance",
          accountId: "paper",
          symbol: "BTCUSDT",
          interval: "5m",
          strategyId: "ema-cross",
          strategyParams: { fastPeriod: 12 },
          intentPolicy: { orderIntent: "execute" },
          status: "pending",
          attemptCount: 0,
        }),
        { status: 201 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    const task = await tradingApi.createTask({
      name: "Paper EMA",
      type: "paper",
      exchange: "binance",
      accountId: "paper",
      symbol: "BTCUSDT",
      interval: "5m",
      strategyId: "ema-cross",
      strategyParams: { fastPeriod: 12 },
      intentPolicy: { orderIntent: "execute" },
    });

    expect(task.id).toBe("tt_1");
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/trading/tasks",
      expect.objectContaining({
        method: "POST",
        body: expect.stringContaining("\"type\":\"paper\""),
      }),
    );
  });
});

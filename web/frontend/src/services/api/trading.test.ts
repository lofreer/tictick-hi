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

  it("lists trading executions and positions", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(new Response(JSON.stringify([{ id: "exe_1" }]), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify([{ symbol: "BTCUSDT" }]), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await expect(tradingApi.listExecutions("tt_1")).resolves.toEqual([{ id: "exe_1" }]);
    await expect(tradingApi.listPositions("tt_1")).resolves.toEqual([{ symbol: "BTCUSDT" }]);
    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      "/api/trading/tasks/tt_1/executions",
      expect.objectContaining({ method: "GET" }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "/api/trading/tasks/tt_1/positions",
      expect.objectContaining({ method: "GET" }),
    );
  });
});

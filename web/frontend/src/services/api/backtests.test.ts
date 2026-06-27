import { afterEach, describe, expect, it, vi } from "vitest";

import { backtestsApi } from "@/services/api/backtests";

describe("backtests api", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("posts create backtest requests", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify({
          id: "bt_1",
          name: "EMA BTC",
          exchange: "binance",
          symbol: "BTCUSDT",
          interval: "5m",
          strategyId: "ema-cross",
          strategyParams: { fastPeriod: 12 },
          initialBalance: "10000",
          feeBps: "1",
          slippageBps: "0.5",
          triggerMode: "closed_candle",
          status: "pending",
          attemptCount: 0,
          resultSummary: {},
        }),
        { status: 201 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    const created = await backtestsApi.createBacktest({
      name: "EMA BTC",
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "5m",
      startTime: "2026-01-01T00:00:00.000Z",
      endTime: "2026-01-02T00:00:00.000Z",
      strategyId: "ema-cross",
      strategyParams: { fastPeriod: 12 },
      initialBalance: "10000",
      feeBps: "1",
      slippageBps: "0.5",
      triggerMode: "closed_candle",
    });

    expect(created.id).toBe("bt_1");
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/backtests",
      expect.objectContaining({
        method: "POST",
        body: expect.stringContaining("\"strategyId\":\"ema-cross\""),
      }),
    );
  });

  it("lists backtest strategy intents", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify([
          {
            id: "si_1",
            taskId: "bt_1",
            taskType: "backtest",
            strategyId: "ema-cross",
            intentType: "order",
            idempotencyKey: "bt_1:ema-cross_order_1",
            payload: { side: "buy" },
            policy: "simulate",
            status: "accepted",
            createdAt: "2026-01-01T00:00:00Z",
          },
        ]),
        { status: 200 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    const intents = await backtestsApi.listIntents("bt_1");

    expect(intents[0].taskType).toBe("backtest");
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/backtests/bt_1/intents",
      expect.objectContaining({ method: "GET" }),
    );
  });
});

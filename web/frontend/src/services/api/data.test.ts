import { afterEach, describe, expect, it, vi } from "vitest";

import { dataApi } from "@/services/api/data";

describe("data api", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("maps candle decimal strings to chart candles", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        new Response(
          JSON.stringify({
            candles: [
              {
                openTime: "2026-01-01T00:00:00Z",
                open: "100.1",
                high: "101.2",
                low: "99.8",
                close: "100.7",
              },
            ],
            source: "native",
            requestedInterval: "1m",
            baseInterval: "1m",
            health: "ok",
            gaps: [],
          }),
          { status: 200 },
        ),
      ),
    );

    const candles = await dataApi.listCandles({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
    });

    expect(candles).toEqual([
      { time: 1767225600, open: 100.1, high: 101.2, low: 99.8, close: 100.7 },
    ]);
  });

  it("maps candle metadata", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        new Response(
          JSON.stringify({
            candles: [],
            source: "aggregated",
            requestedInterval: "5m",
            baseInterval: "1m",
            health: "gap",
            gaps: [{ from: "2026-01-01T00:01:00Z", to: "2026-01-01T00:03:00Z", missingCandles: 2 }],
            coverage: {
              requestedLimit: 1000,
              returnedCandles: 0,
              requiredBaseCandles: 5000,
              baseLimit: 5000,
              returnedBaseCandles: 2000,
              limitedByBaseWindow: true,
            },
          }),
          { status: 200 },
        ),
      ),
    );

    const result = await dataApi.getCandles({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "5m",
    });

    expect(result).toMatchObject({
      source: "aggregated",
      requestedInterval: "5m",
      baseInterval: "1m",
      health: "gap",
      gaps: [{ missingCandles: 2 }],
      coverage: {
        requestedLimit: 1000,
        returnedCandles: 0,
        requiredBaseCandles: 5000,
        baseLimit: 5000,
        returnedBaseCandles: 2000,
        limitedByBaseWindow: true,
      },
    });
  });

  it("keeps data sync attempt count", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        new Response(
          JSON.stringify([
            {
              id: "dst_1",
              exchange: "binance",
              symbol: "BTCUSDT",
              interval: "1m",
              realtimeEnabled: true,
              syncEnabled: true,
              status: "running",
              dataHealth: "retrying",
              lastError: "temporary EOF",
              attemptCount: 3,
              nextAttemptAt: "2026-06-28T01:30:00Z",
            },
          ]),
          { status: 200 },
        ),
      ),
    );

    await expect(dataApi.listTasks()).resolves.toMatchObject([
      {
        id: "dst_1",
        attemptCount: 3,
        dataHealth: "retrying",
        lastError: "temporary EOF",
        nextAttemptAt: "2026-06-28T01:30:00Z",
      },
    ]);
  });

  it("queues failed data sync task retry", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify({
          id: "dst_1",
          exchange: "binance",
          symbol: "BTCUSDT",
          interval: "1m",
          realtimeEnabled: false,
          syncEnabled: true,
          status: "pending",
          dataHealth: "syncing",
        }),
        { status: 200 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(dataApi.retryTask("dst_1")).resolves.toMatchObject({
      id: "dst_1",
      dataHealth: "syncing",
      syncEnabled: true,
      status: "pending",
    });
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/data/tasks/dst_1/retry",
      expect.objectContaining({ method: "POST" }),
    );
  });
});

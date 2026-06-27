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
    });
  });
});

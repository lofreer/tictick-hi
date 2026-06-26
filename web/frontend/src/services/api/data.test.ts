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
          JSON.stringify([
            {
              openTime: "2026-01-01T00:00:00Z",
              open: "100.1",
              high: "101.2",
              low: "99.8",
              close: "100.7",
            },
          ]),
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
});


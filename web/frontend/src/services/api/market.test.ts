import { afterEach, describe, expect, it, vi } from "vitest";

import { marketApi } from "@/services/api/market";

describe("market api", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("searches market instruments with encoded query params", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify([
          {
            exchange: "binance",
            symbol: "SOLUSDT",
            baseAsset: "SOL",
            quoteAsset: "USDT",
            instrumentType: "spot",
            status: "active",
            searchPriority: 3,
            createdAt: "2026-06-28T00:00:00Z",
            updatedAt: "2026-06-28T00:00:00Z",
          },
        ]),
        { status: 200 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(marketApi.listInstruments({ exchange: "binance", q: "SOL", limit: 20 })).resolves.toMatchObject([
      { exchange: "binance", symbol: "SOLUSDT", baseAsset: "SOL" },
    ]);
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/market/instruments?exchange=binance&q=SOL&limit=20",
      expect.objectContaining({ method: "GET" }),
    );
  });
});

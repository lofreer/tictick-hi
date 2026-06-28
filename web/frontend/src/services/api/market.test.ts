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

    await expect(marketApi.listInstruments({ exchange: "binance", q: "SOL", limit: 20, status: "all" })).resolves.toMatchObject([
      { exchange: "binance", symbol: "SOLUSDT", baseAsset: "SOL" },
    ]);
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/market/instruments?exchange=binance&q=SOL&limit=20&status=all",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("requests a catalog sync for one exchange", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify({
          exchange: "binance",
          activeCount: 100,
          inactiveCount: 2,
          pausedDataSyncTaskCount: 1,
          syncedAt: "2026-06-28T00:00:00Z",
        }),
        { status: 200 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(marketApi.syncInstruments("binance")).resolves.toMatchObject({
      exchange: "binance",
      activeCount: 100,
      pausedDataSyncTaskCount: 1,
    });
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/market/instruments/sync?exchange=binance",
      expect.objectContaining({ method: "POST" }),
    );
  });
});

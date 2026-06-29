import { afterEach, describe, expect, it, vi } from "vitest";

import { dataApi } from "@/services/api/data";

describe("market candle api", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("scans persisted market candle gaps", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify({
          exchange: "binance",
          symbol: "BTCUSDT",
          interval: "1m",
          window: {
            from: "2026-06-27T03:00:00Z",
            to: "2026-06-27T03:06:00Z",
            count: 4,
          },
          gaps: [
            {
              from: "2026-06-27T03:02:00Z",
              to: "2026-06-27T03:03:00Z",
              missingCandles: 1,
            },
          ],
          limited: true,
          totalCount: 2,
          returnedCount: 1,
        }),
        { status: 200 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(dataApi.scanMarketCandleGaps({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      limit: 1,
    })).resolves.toMatchObject({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      totalCount: 2,
      returnedCount: 1,
      limited: true,
      window: { count: 4 },
      gaps: [{ missingCandles: 1 }],
    });
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/market/candle-gaps?exchange=binance&symbol=BTCUSDT&interval=1m&limit=1",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("scans persisted market candle invalid issues", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify({
          exchange: "binance",
          symbol: "BTCUSDT",
          interval: "1m",
          window: {
            from: "2026-06-27T03:00:00Z",
            to: "2026-06-27T03:03:00Z",
            count: 4,
          },
          issues: [
            {
              code: "invalid_open_price",
              message: "open price value must be positive",
              openTime: "2026-06-27T03:01:00Z",
            },
          ],
          limited: true,
          totalCount: 2,
          returnedCount: 1,
        }),
        { status: 200 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(dataApi.scanMarketCandleInvalidIssues({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      limit: 1,
    })).resolves.toMatchObject({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      totalCount: 2,
      returnedCount: 1,
      limited: true,
      window: { count: 4 },
      issues: [{ code: "invalid_open_price" }],
    });
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/market/candle-invalid-issues?exchange=binance&symbol=BTCUSDT&interval=1m&limit=1",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("queues a repair task for a full-history market gap", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify({
          sourceTaskId: "",
          createdTasks: [
            {
              id: "dst_market_repair_1",
              exchange: "binance",
              symbol: "BTCUSDT",
              interval: "1m",
              startTime: "2026-06-27T03:02:00Z",
              endTime: "2026-06-27T03:03:00Z",
              realtimeEnabled: false,
              syncEnabled: true,
              status: "pending",
              dataHealth: "syncing",
            },
          ],
          skippedExisting: 0,
          limited: false,
          totalCount: 1,
          repairLimit: 1,
        }),
        { status: 200 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(dataApi.repairMarketCandleGap({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      from: "2026-06-27T03:02:00Z",
      to: "2026-06-27T03:03:00Z",
    })).resolves.toMatchObject({
      sourceTaskId: "",
      createdTasks: [{ id: "dst_market_repair_1", syncEnabled: true, dataHealth: "syncing" }],
      totalCount: 1,
      repairLimit: 1,
    });
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/market/candle-gaps/repair",
      expect.objectContaining({
        body: JSON.stringify({
          exchange: "binance",
          symbol: "BTCUSDT",
          interval: "1m",
          from: "2026-06-27T03:02:00Z",
          to: "2026-06-27T03:03:00Z",
        }),
        method: "POST",
      }),
    );
  });

  it("queues repair tasks for returned full-history market gaps", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify({
          sourceTaskId: "",
          createdTasks: [
            {
              id: "dst_market_repair_1",
              exchange: "binance",
              symbol: "BTCUSDT",
              interval: "1m",
              startTime: "2026-06-27T03:02:00Z",
              endTime: "2026-06-27T03:03:00Z",
              realtimeEnabled: false,
              syncEnabled: true,
              status: "pending",
              dataHealth: "syncing",
            },
          ],
          skippedExisting: 1,
          limited: false,
          totalCount: 2,
          repairLimit: 100,
        }),
        { status: 200 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(dataApi.repairMarketCandleGaps({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      gaps: [
        { from: "2026-06-27T03:02:00Z", to: "2026-06-27T03:03:00Z" },
        { from: "2026-06-27T03:05:00Z", to: "2026-06-27T03:07:00Z" },
      ],
    })).resolves.toMatchObject({
      sourceTaskId: "",
      createdTasks: [{ id: "dst_market_repair_1", syncEnabled: true, dataHealth: "syncing" }],
      skippedExisting: 1,
      totalCount: 2,
      repairLimit: 100,
    });
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/market/candle-gaps/repair-batch",
      expect.objectContaining({
        body: JSON.stringify({
          exchange: "binance",
          symbol: "BTCUSDT",
          interval: "1m",
          gaps: [
            { from: "2026-06-27T03:02:00Z", to: "2026-06-27T03:03:00Z" },
            { from: "2026-06-27T03:05:00Z", to: "2026-06-27T03:07:00Z" },
          ],
        }),
        method: "POST",
      }),
    );
  });

  it("queues repair tasks for returned full-history invalid market candles", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify({
          sourceTaskId: "",
          createdTasks: [
            {
              id: "dst_market_invalid_repair_1",
              exchange: "binance",
              symbol: "BTCUSDT",
              interval: "1m",
              startTime: "2026-06-27T03:01:00Z",
              endTime: "2026-06-27T03:02:00Z",
              realtimeEnabled: false,
              syncEnabled: true,
              status: "pending",
              dataHealth: "syncing",
            },
          ],
          skippedExisting: 1,
          limited: false,
          totalCount: 2,
          repairLimit: 100,
        }),
        { status: 200 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(dataApi.repairMarketCandleInvalidIssues({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      openTimes: ["2026-06-27T03:01:00Z", "2026-06-27T03:02:00Z"],
    })).resolves.toMatchObject({
      sourceTaskId: "",
      createdTasks: [{ id: "dst_market_invalid_repair_1", syncEnabled: true, dataHealth: "syncing" }],
      skippedExisting: 1,
      totalCount: 2,
      repairLimit: 100,
    });
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/market/candle-invalid-issues/repair",
      expect.objectContaining({
        body: JSON.stringify({
          exchange: "binance",
          symbol: "BTCUSDT",
          interval: "1m",
          openTimes: ["2026-06-27T03:01:00Z", "2026-06-27T03:02:00Z"],
        }),
        method: "POST",
      }),
    );
  });
});

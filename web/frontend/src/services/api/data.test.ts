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
                volume: "1234.56",
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
      { time: 1767225600, open: 100.1, high: 101.2, low: 99.8, close: 100.7, volume: 1234.56 },
    ]);
  });

  it("drops candles with invalid volume", async () => {
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
                volume: "not-a-number",
              },
              {
                openTime: "2026-01-01T00:01:00Z",
                open: "101",
                high: "102",
                low: "100",
                close: "101.5",
                volume: "42",
              },
            ],
            source: "native",
            requestedInterval: "1m",
            health: "ok",
          }),
          { status: 200 },
        ),
      ),
    );

    await expect(
      dataApi.listCandles({
        exchange: "binance",
        symbol: "BTCUSDT",
        interval: "1m",
      }),
    ).resolves.toEqual([{ time: 1767225660, open: 101, high: 102, low: 100, close: 101.5, volume: 42 }]);
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
            window: {
              from: "2026-01-01T00:00:00Z",
              to: "2026-01-01T00:55:00Z",
              count: 12,
            },
            pagination: {
              hasPrevious: true,
              hasNext: true,
              previousCursor: "prev_cursor",
              nextCursor: "next_cursor",
              previousFrom: "2025-12-31T22:00:00Z",
              previousTo: "2025-12-31T23:55:00Z",
              nextFrom: "2026-01-01T00:05:00Z",
              nextTo: "2026-01-01T02:00:00Z",
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
      window: {
        from: "2026-01-01T00:00:00Z",
        to: "2026-01-01T00:55:00Z",
        count: 12,
      },
      pagination: {
        hasPrevious: true,
        hasNext: true,
        previousCursor: "prev_cursor",
        nextCursor: "next_cursor",
        previousFrom: "2025-12-31T22:00:00Z",
        previousTo: "2025-12-31T23:55:00Z",
        nextFrom: "2026-01-01T00:05:00Z",
        nextTo: "2026-01-01T02:00:00Z",
      },
    });
  });

  it("sends candle cursor without explicit window or limit parameters", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify({
          candles: [],
          source: "none",
          requestedInterval: "1m",
          health: "insufficient",
        }),
        { status: 200 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    await dataApi.getCandles({
      exchange: "binance",
      symbol: "BTCUSDT",
      interval: "1m",
      cursor: "opaque_cursor",
      from: "2026-01-01T00:00:00Z",
      to: "2026-01-01T01:00:00Z",
      limit: 500,
    });

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/candles?exchange=binance&symbol=BTCUSDT&interval=1m&cursor=opaque_cursor",
      expect.objectContaining({ method: "GET" }),
    );
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
              repairSourceTaskId: "dst_source_1",
              realtimeEnabled: true,
              syncEnabled: true,
              status: "running",
              dataHealth: "retrying",
              gapSummary: {
                count: 2,
                firstGap: {
                  from: "2026-06-27T03:02:00Z",
                  to: "2026-06-27T03:03:00Z",
                  missingCandles: 1,
                },
              },
              lastError:
                'binance klines: Get "https://api.binance.com/api/v3/klines?endTime=1782524388943&interval=1m&limit=500&startTime=1780277926000&symbol=BTCUSDT": EOF',
              attemptCount: 3,
              nextAttemptAt: "2026-06-28T01:30:00Z",
              exchangeBackoffUntil: "2026-06-28T01:45:00Z",
              exchangeBackoffLastError:
                'binance klines temporary unavailable: Get "https://api.binance.com/api/v3/klines?symbol=BTCUSDT": EOF',
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
        repairSourceTaskId: "dst_source_1",
        gapSummary: {
          count: 2,
          firstGap: {
            from: "2026-06-27T03:02:00Z",
            to: "2026-06-27T03:03:00Z",
            missingCandles: 1,
          },
        },
        lastError: 'binance klines: Get "api.binance.com": EOF',
        nextAttemptAt: "2026-06-28T01:30:00Z",
        exchangeBackoffUntil: "2026-06-28T01:45:00Z",
        exchangeBackoffLastError: 'binance klines temporary unavailable: Get "api.binance.com": EOF',
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

  it("loads data sync task gap details", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify({
          taskId: "dst_1",
          gaps: [
            {
              from: "2026-06-27T03:02:00Z",
              to: "2026-06-27T03:03:00Z",
              missingCandles: 1,
            },
          ],
          limited: false,
          totalCount: 1,
          returnedCount: 1,
          repairLimit: 20,
        }),
        { status: 200 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(dataApi.getTaskGaps("dst_1")).resolves.toEqual({
      taskId: "dst_1",
      gaps: [
        {
          from: "2026-06-27T03:02:00Z",
          to: "2026-06-27T03:03:00Z",
          missingCandles: 1,
        },
      ],
      limited: false,
      totalCount: 1,
      returnedCount: 1,
      repairLimit: 20,
    });
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/data/tasks/dst_1/gaps",
      expect.objectContaining({ method: "GET" }),
    );
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

  it("queues repair tasks for data sync gaps", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify({
          sourceTaskId: "dst_1",
          createdTasks: [
            {
              id: "dst_repair_1",
              exchange: "binance",
              symbol: "BTCUSDT",
              interval: "1m",
              startTime: "2026-06-27T03:02:00Z",
              endTime: "2026-06-27T03:03:00Z",
              repairSourceTaskId: "dst_1",
              realtimeEnabled: false,
              syncEnabled: true,
              status: "pending",
              dataHealth: "syncing",
            },
          ],
          skippedExisting: 1,
          limited: false,
          totalCount: 1,
          repairLimit: 20,
        }),
        { status: 200 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(dataApi.repairTaskGaps("dst_1")).resolves.toMatchObject({
      sourceTaskId: "dst_1",
      createdTasks: [
        {
          id: "dst_repair_1",
          startTime: "2026-06-27T03:02:00Z",
          repairSourceTaskId: "dst_1",
          syncEnabled: true,
          dataHealth: "syncing",
        },
      ],
      skippedExisting: 1,
      limited: false,
      totalCount: 1,
      repairLimit: 20,
    });
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/data/tasks/dst_1/repair-gaps",
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("queues a repair task for a single chart gap", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify({
          sourceTaskId: "dst_1",
          createdTasks: [
            {
              id: "dst_repair_1",
              exchange: "binance",
              symbol: "BTCUSDT",
              interval: "1m",
              startTime: "2026-06-27T03:02:00Z",
              endTime: "2026-06-27T03:03:00Z",
              repairSourceTaskId: "dst_1",
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

    await expect(dataApi.repairTaskGap("dst_1", {
      from: "2026-06-27T03:02:00Z",
      to: "2026-06-27T03:03:00Z",
    })).resolves.toMatchObject({
      sourceTaskId: "dst_1",
      createdTasks: [
        {
          id: "dst_repair_1",
          startTime: "2026-06-27T03:02:00Z",
          repairSourceTaskId: "dst_1",
          syncEnabled: true,
          dataHealth: "syncing",
        },
      ],
      skippedExisting: 0,
      totalCount: 1,
      repairLimit: 1,
    });
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/data/tasks/dst_1/repair-gap",
      expect.objectContaining({
        body: JSON.stringify({
          from: "2026-06-27T03:02:00Z",
          to: "2026-06-27T03:03:00Z",
        }),
        method: "POST",
      }),
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
});

import { afterEach, describe, expect, it, vi } from "vitest";

import { overviewApi } from "@/services/api/overview";

describe("overview api", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("lists recent overview facts", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify({
          strategyIntents: [
            {
              id: "si_1",
              taskId: "bt_1",
              taskType: "backtest",
              taskName: "Baseline",
              exchange: "binance",
              symbol: "BTCUSDT",
              interval: "1m",
              strategyId: "ema-cross",
              intentType: "order",
              policy: "simulate",
              status: "accepted",
              createdAt: "2026-06-28T01:09:00Z",
            },
          ],
          orders: [],
        }),
        { status: 200 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    const facts = await overviewApi.recentFacts({ limit: 8, since: "2026-06-27T01:09:00.000Z" });

    expect(facts.strategyIntents[0].taskName).toBe("Baseline");
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/overview/recent-facts?limit=8&since=2026-06-27T01%3A09%3A00.000Z",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("lists overview trend buckets", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify({
          days: 7,
          from: "2026-07-01T00:00:00Z",
          to: "2026-07-08T00:00:00Z",
          buckets: [
            {
              bucketStart: "2026-07-01T00:00:00Z",
              strategyIntents: 2,
              orders: 1,
              notifications: 1,
              failures: 0,
            },
          ],
        }),
        { status: 200 },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    const trends = await overviewApi.trends({ days: 7 });

    expect(trends.buckets[0].strategyIntents).toBe(2);
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/overview/trends?days=7",
      expect.objectContaining({ method: "GET" }),
    );
  });
});

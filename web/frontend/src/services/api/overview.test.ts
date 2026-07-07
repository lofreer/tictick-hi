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

    const facts = await overviewApi.recentFacts(8);

    expect(facts.strategyIntents[0].taskName).toBe("Baseline");
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/overview/recent-facts?limit=8",
      expect.objectContaining({ method: "GET" }),
    );
  });
});

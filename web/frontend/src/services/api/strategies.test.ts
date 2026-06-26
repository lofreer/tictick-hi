import { afterEach, describe, expect, it, vi } from "vitest";

import { strategiesApi } from "@/services/api/strategies";

describe("strategies api", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("normalizes strategy parameter options", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        new Response(
          JSON.stringify([
            {
              id: "ema-cross",
              name: "EMA Cross",
              version: "v1",
              description: "Cross strategy",
              supportedIntervals: ["1m", "5m"],
              supportedIntents: ["order"],
              params: [
                { key: "fastPeriod", label: "Fast", type: "number", required: true, default: 12 },
                {
                  key: "signalMode",
                  label: "Signal mode",
                  type: "select",
                  required: true,
                  default: "order",
                  options: [{ label: "Order", value: "order" }],
                },
              ],
            },
          ]),
          { status: 200 },
        ),
      ),
    );

    const strategies = await strategiesApi.listStrategies();

    expect(strategies[0].params[0].options).toEqual([]);
    expect(strategies[0].params[1].options).toEqual([{ label: "Order", value: "order" }]);
  });
});

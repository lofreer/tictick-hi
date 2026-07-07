import { describe, expect, it } from "vitest";

import { FIRST_VERSION_MARKET_INTERVALS, marketIntervalOptions } from "@/utils/marketIntervals";

describe("market intervals", () => {
  it("defines the first-version K-line interval surface", () => {
    expect(FIRST_VERSION_MARKET_INTERVALS).toEqual(["1m", "5m", "15m", "1h", "4h", "1d"]);
  });

  it("maps market intervals to select options", () => {
    expect(marketIntervalOptions(["1m", "1h"])).toEqual([
      { label: "1m", value: "1m" },
      { label: "1h", value: "1h" },
    ]);
  });
});

import { describe, expect, it } from "vitest";

import type { CandleResult } from "@/types/app";
import { candleCoverageLabel, candleCoverageTagType, hasBaseCandleCoverage, shouldShowCandleCoverage } from "./researchCandleCoverage";

describe("research candle coverage metadata", () => {
  it("hides coverage for healthy complete candle results", () => {
    expect(shouldShowCandleCoverage(candleResult({ health: "ok", returnedCandles: 1000 }))).toBe(false);
  });

  it("shows coverage for non-ok candle health even when the base window is not limited", () => {
    expect(shouldShowCandleCoverage(candleResult({ health: "gap", returnedCandles: 84, limitedByBaseWindow: false }))).toBe(true);
  });

  it("shows coverage for limited base windows", () => {
    expect(shouldShowCandleCoverage(candleResult({ health: "ok", returnedCandles: 500, limitedByBaseWindow: true }))).toBe(true);
  });

  it("detects base candle coverage only when both base counters are present", () => {
    expect(hasBaseCandleCoverage(candleResult({ returnedBaseCandles: 5099, requiredBaseCandles: 5100 }).coverage)).toBe(true);
    expect(hasBaseCandleCoverage(candleResult({ returnedBaseCandles: 5099 }).coverage)).toBe(false);
  });

  it("formats candle and base coverage for non-ok results", () => {
    const result = candleResult({
      health: "gap",
      requiredBaseCandles: 5100,
      returnedBaseCandles: 5099,
      returnedCandles: 84,
    });

    expect(candleCoverageTagType(result)).toBe("warning");
    expect(candleCoverageLabel(result, translate)).toBe("research.coverageSummary:requested=85,returned=84 / research.baseCoverage:required=5100,returned=5099");
  });
});

function translate(key: string, values: Record<string, number> = {}) {
  return `${key}:${Object.entries(values)
    .map(([name, value]) => `${name}=${value}`)
    .join(",")}`;
}

function candleResult(
  overrides: {
    health?: CandleResult["health"];
    returnedCandles?: number;
    limitedByBaseWindow?: boolean;
    returnedBaseCandles?: number;
    requiredBaseCandles?: number;
  } = {},
): CandleResult {
  return {
    baseInterval: "1m",
    candles: [],
    coverage: {
      requestedLimit: 85,
      returnedCandles: overrides.returnedCandles ?? 85,
      limitedByBaseWindow: overrides.limitedByBaseWindow ?? false,
      ...(overrides.requiredBaseCandles === undefined ? {} : { requiredBaseCandles: overrides.requiredBaseCandles }),
      ...(overrides.returnedBaseCandles === undefined ? {} : { returnedBaseCandles: overrides.returnedBaseCandles }),
    },
    gaps: [],
    health: overrides.health ?? "ok",
    issues: [],
    pagination: {
      hasNext: false,
      hasPrevious: false,
    },
    requestedInterval: "1h",
    source: "aggregated",
    window: {
      count: overrides.returnedCandles ?? 85,
    },
  };
}
